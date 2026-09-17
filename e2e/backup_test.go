package e2e

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// Backup and restore, executed.
//
// docs/BACKUP-RESTORE.md has been a runbook nobody has run. A runbook is a
// description of a procedure; whether the procedure works is a different
// question, and the answer only ever arrives when somebody needs it.
//
// This one takes a real logical backup of a platform that has been used,
// destroys the database, restores it, and asks the platform the same questions
// again through its own API. The identifiers have to be the same identifiers,
// because a Global ID that changed across a restore is a platform that has
// silently renamed every customer's records.
//
// It is opt-in and runs in its own CI job with its own stack. Dropping the
// database underneath the other scenarios would turn one failure here into a
// wall of failures everywhere, and the one that mattered would be lost in it.

// ErrBackupRequired is returned when a run was told the backup scenario is
// mandatory and it was not enabled.
var ErrBackupRequired = errors.New("E2E_BACKUP_REQUIRED is set but E2E_BACKUP is empty")

// checkBackupRequired is the same guard as the other two, for the same reason:
// a job whose whole purpose is this scenario must not pass by skipping it.
func checkBackupRequired(backup, required string) error {
	if strings.TrimSpace(required) == "" {
		return nil
	}
	if strings.TrimSpace(backup) != "" {
		return nil
	}
	return fmt.Errorf("%w: the job exists to run it, and a skipped scenario would report success having restored nothing", ErrBackupRequired)
}

func TestTheBackupRequirementGuard(t *testing.T) {
	if err := checkBackupRequired("", "1"); err == nil {
		t.Error("a required run with the scenario disabled was allowed to skip")
	} else if !errors.Is(err, ErrBackupRequired) {
		t.Errorf("error does not identify itself: %v", err)
	}
	if err := checkBackupRequired("1", "1"); err != nil {
		t.Errorf("a configured required run was refused: %v", err)
	}
	if err := checkBackupRequired("", ""); err != nil {
		t.Errorf("an ordinary run was refused: %v", err)
	}
	if err := checkBackupRequired("  ", "1"); err == nil {
		t.Error("a whitespace E2E_BACKUP satisfied a required run")
	}
}

// platformState is what has to survive a restore, read back through the API
// rather than out of the tables.
//
// Through the API on purpose: a row that survived and that the platform can no
// longer serve is not a successful restore, and comparing tables to tables
// would not notice.
type platformState struct {
	ClientGlobalID string
	ServerGlobalID string
	IncidentID     string
	UserGlobalID   string
	ScopeGrants    string
	AuditEntries   int
	Notifications  int
}

func (h *Harness) readPlatformState(t *testing.T, clientID, serverID, incidentID string) platformState {
	t.Helper()
	state := platformState{ClientGlobalID: clientID, ServerGlobalID: serverID, IncidentID: incidentID}

	mapping := h.API(t, http.MethodGet, "/api/v1/global-ids/"+clientID, TokenAdmin, nil)
	if mapping.Status != http.StatusOK {
		t.Fatalf("read the Global ID mapping for %s: status = %d (body: %s)", clientID, mapping.Status, truncate(mapping.Body))
	}

	incident := h.API(t, http.MethodGet, "/api/v1/incidents/"+incidentID, TokenAdmin, nil)
	if incident.Status != http.StatusOK {
		t.Fatalf("read incident %s: status = %d (body: %s)", incidentID, incident.Status, truncate(incident.Body))
	}

	server := h.API(t, http.MethodGet, "/api/v1/servers/"+serverID, TokenAdmin, nil)
	if server.Status != http.StatusOK {
		t.Fatalf("read server %s: status = %d (body: %s)", serverID, server.Status, truncate(server.Body))
	}

	me := h.API(t, http.MethodGet, "/api/v1/me", TokenAdmin, nil)
	var profile struct {
		ID string `json:"id"`
	}
	me.JSON(t, &profile)
	state.UserGlobalID = profile.ID

	scopes := h.API(t, http.MethodGet, "/api/v1/admin/rbac/scopes?principal_type=user&principal_id="+profile.ID, TokenAdmin, nil)
	if scopes.Status != http.StatusOK {
		t.Fatalf("read scope grants: status = %d (body: %s)", scopes.Status, truncate(scopes.Body))
	}
	state.ScopeGrants = string(scopes.Body)

	audit := h.API(t, http.MethodGet, "/api/v1/audit?limit=500", TokenAdmin, nil)
	if audit.Status != http.StatusOK {
		t.Fatalf("read the audit trail: status = %d (body: %s)", audit.Status, truncate(audit.Body))
	}
	var entries []map[string]any
	audit.JSON(t, &entries)
	state.AuditEntries = len(entries)

	notifications := h.API(t, http.MethodGet, "/api/v1/notifications?limit=100", TokenAdmin, nil)
	if notifications.Status != http.StatusOK {
		t.Fatalf("read notifications: status = %d (body: %s)", notifications.Status, truncate(notifications.Body))
	}
	var page struct {
		Data []map[string]any `json:"data"`
	}
	notifications.JSON(t, &page)
	state.Notifications = len(page.Data)

	return state
}

func TestABackupOfAUsedPlatformRestoresIntoAFreshDatabase(t *testing.T) {
	if strings.TrimSpace(os.Getenv("E2E_BACKUP")) == "" {
		t.Skip("E2E_BACKUP is not set; this scenario destroys the stack's database and runs in its own CI job")
	}
	harness := ready(t)
	control := LifecycleControl(t)
	stamp := time.Now().UnixNano()

	// --- Give the platform something to lose -------------------------------
	//
	// Written through the API, so what is backed up is what the platform
	// actually produces rather than rows a test invented. Every category the
	// restore has to preserve is represented: an identity, a service
	// identity, a Global ID mapping, an RBAC grant, an audit trail, a
	// notification, an operations record and a support record.

	created := harness.API(t, http.MethodPost, "/api/v1/global-ids", TokenAdmin,
		map[string]any{"entity_type": "client", "source": "e2e-backup", "source_id": fmt.Sprintf("acc-%d", stamp)})
	if created.Status != http.StatusCreated {
		t.Fatalf("allocate a Global ID: status = %d (body: %s)", created.Status, truncate(created.Body))
	}
	var entity struct {
		GlobalID string `json:"global_id"`
	}
	created.JSON(t, &entity)

	incident := raiseIncident(t, harness, TokenSupport, map[string]any{
		"kind": "incident", "title": fmt.Sprintf("backup fixture %d", stamp),
		"severity": "high", "client_id": entity.GlobalID,
	})

	reports := &reporter{h: harness, source: "e2e-backup"}
	registered := reports.register(t, fmt.Sprintf("srv-%d", stamp), "backup fixture", "stage", nil)
	reports.report(t, registered.ID, "backup.failed", "a fixture event, so an operations record exists to lose")

	// A scope grant is the fail-closed audit path, so this also puts a record
	// in the audit trail that cannot have been written without the grant.
	profile := harness.API(t, http.MethodGet, "/api/v1/me", TokenAdmin, nil)
	var me struct {
		ID string `json:"id"`
	}
	profile.JSON(t, &me)
	grant := map[string]any{
		"principal_type": "user", "principal_id": me.ID,
		"scope_type": "client", "scope_id": entity.GlobalID,
		"permission_id": "crm.client.read",
	}
	if granted := harness.API(t, http.MethodPost, "/api/v1/admin/rbac/scopes", TokenAdmin, grant); granted.Status != http.StatusCreated {
		t.Fatalf("grant a scope: status = %d (body: %s)", granted.Status, truncate(granted.Body))
	}
	t.Cleanup(func() {
		harness.API(t, http.MethodDelete, "/api/v1/admin/rbac/scopes", TokenAdmin, grant)
	})

	before := harness.readPlatformState(t, entity.GlobalID, registered.ID, incident.ID)
	if before.AuditEntries == 0 || before.Notifications == 0 {
		t.Fatalf("the fixture produced no audit entries or no notifications (%+v); the restore would prove nothing about either", before)
	}

	// --- Take the backup ---------------------------------------------------
	//
	// pg_dump inside the database container, with the test-only credentials
	// the stack already carries. Nothing is published outside the job: the
	// dump stays on the runner and is not uploaded as an artifact, because a
	// database dump is the one artefact nobody should be able to download
	// from a CI run.
	dump := control.Exec(t, ServicePostgres, "pg_dump", "-U", "bsystem", "--clean", "--if-exists", "bsystem_integration")
	if len(dump) < 2000 {
		t.Fatalf("the backup is %d bytes; that is not a database", len(dump))
	}
	for _, expected := range []string{"CREATE TABLE", "schema_migrations", "global_entities", "audit_events"} {
		if !strings.Contains(dump, expected) {
			t.Fatalf("the backup does not contain %q; it is not a backup of this platform", expected)
		}
	}

	// --- What a backup must not carry --------------------------------------
	//
	// The platform holds upstream credentials in its environment and must
	// never write one into a row. A dump is copied to laptops, attached to
	// tickets and kept for years, so it is the worst possible place for one
	// to appear — and this is the only check that would notice if a future
	// change started storing them.
	for _, secret := range []string{EspoCRMTestAPIKey, RedmineTestAPIKey, OutlineTestAPIKey, PostgresTestPassword, "Bearer "} {
		if strings.Contains(dump, secret) {
			t.Errorf("the backup contains %q: the application layer has written a credential into a row", secret)
		}
	}

	// --- Destroy and restore ------------------------------------------------
	control.Exec(t, ServicePostgres, "psql", "-U", "bsystem", "-d", "postgres", "-v", "ON_ERROR_STOP=1",
		"-c", "DROP DATABASE bsystem_integration WITH (FORCE)")
	control.Exec(t, ServicePostgres, "psql", "-U", "bsystem", "-d", "postgres", "-v", "ON_ERROR_STOP=1",
		"-c", "CREATE DATABASE bsystem_integration")

	// The platform must be unable to serve while its database is gone. If it
	// answered here, the restore below would be proving nothing: the data
	// would still be in a cache.
	gone := harness.API(t, http.MethodGet, "/api/v1/global-ids/"+entity.GlobalID, TokenAdmin, nil)
	if gone.Status == http.StatusOK {
		t.Fatalf("the platform served a record from a database that had just been dropped and recreated empty")
	}

	restored := control.ExecInput(t, ServicePostgres, dump,
		"psql", "-U", "bsystem", "-d", "bsystem_integration", "-v", "ON_ERROR_STOP=1")
	_ = restored

	// --- The platform comes back on its own ---------------------------------
	//
	// No restart. The pool reconnects, and the schema is the restored one:
	// migrations run at startup, so a restore that needed the process
	// restarted to be usable would be a different, worse contract.
	waitFor(t, "the platform to serve from the restored database", 90*time.Second, func() (bool, string) {
		report, err := harness.readiness(harness.Core)
		if err != nil {
			return false, err.Error()
		}
		return report.code == http.StatusOK && report.Checks["database"] == "ok", report.String()
	})

	after := harness.readPlatformState(t, entity.GlobalID, registered.ID, incident.ID)

	// --- The identifiers are the identifiers --------------------------------
	//
	// This is the assertion the whole scenario is for. A Global ID that
	// changed across a restore is a platform that has silently renamed every
	// customer's records, and every mapping held by every other system is now
	// wrong.
	if after.UserGlobalID != before.UserGlobalID {
		t.Errorf("the administrator's Global ID changed across the restore: %s -> %s", before.UserGlobalID, after.UserGlobalID)
	}
	if after.ScopeGrants != before.ScopeGrants {
		t.Errorf("the scope grants differ across the restore:\n  before: %s\n  after:  %s", before.ScopeGrants, after.ScopeGrants)
	}
	if after.AuditEntries != before.AuditEntries {
		t.Errorf("the audit trail holds %d entries after the restore, %d before", after.AuditEntries, before.AuditEntries)
	}
	if after.Notifications != before.Notifications {
		t.Errorf("notifications: %d after the restore, %d before", after.Notifications, before.Notifications)
	}

	// A newly allocated Global ID must continue the sequence rather than
	// restart it. A counter restored to zero would mint an identifier that
	// already belongs to something else, which is worse than losing the row.
	next := harness.API(t, http.MethodPost, "/api/v1/global-ids", TokenAdmin,
		map[string]any{"entity_type": "client", "source": "e2e-backup", "source_id": fmt.Sprintf("acc-after-%d", stamp)})
	if next.Status != http.StatusCreated {
		t.Fatalf("allocate after the restore: status = %d (body: %s)", next.Status, truncate(next.Body))
	}
	var allocated struct {
		GlobalID string `json:"global_id"`
	}
	next.JSON(t, &allocated)
	if allocated.GlobalID == entity.GlobalID {
		t.Fatalf("the Global ID counter restarted: %s was minted twice", allocated.GlobalID)
	}

	// --- The schema is intact and knows it ----------------------------------
	//
	// A restore that lost the migration bookkeeping looks healthy until the
	// next deployment reapplies a migration onto a schema that already has
	// it. Drift is the other half: a restored database that reports the same
	// level while holding a different schema is invisible to every other
	// series.
	applied := harness.metricOrZero(t, harness.Core, "bsystem_schema_migrations_applied", nil)
	if applied == 0 {
		t.Error("the restored database reports no applied migrations; the migration history did not survive")
	}
	if drifted := harness.metricOrZero(t, harness.Core, "bsystem_schema_migrations_drifted", nil); drifted != 0 {
		t.Errorf("the restored database reports %v drifted migration(s): its schema does not match the one the code would produce", drifted)
	}
}
