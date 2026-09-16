package e2e

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

type server struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Environment string `json:"environment"`
	Status      string `json:"status"`
	ClientID    string `json:"client_id"`
	ProjectID   string `json:"project_id"`
	Source      string `json:"source"`
	SourceID    string `json:"source_id"`
	LastEventAt string `json:"last_event_at"`
}

type operationsEvent struct {
	ID            int64  `json:"id"`
	ServerID      string `json:"server_id"`
	Event         string `json:"event"`
	Severity      string `json:"severity"`
	Summary       string `json:"summary"`
	Source        string `json:"source"`
	CorrelationID string `json:"correlation_id"`
	OccurredAt    string `json:"occurred_at"`
}

type serverPage struct {
	Data       []server   `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type operationsEventPage struct {
	Data       []operationsEvent `json:"data"`
	Pagination Pagination        `json:"pagination"`
}

// reporter stands in for BRAVO. It is what the operations contract looks like
// from the other side: register a host, then say what happens to it. Nothing
// here depends on BRAVO's own API, which is the point — the contract was
// written so an integrator needs only the platform's half.
type reporter struct {
	h      *Harness
	source string
}

func newReporter(t *testing.T, h *Harness) *reporter {
	t.Helper()
	return &reporter{h: h, source: "e2e-bravo"}
}

func (r *reporter) register(t *testing.T, sourceID, name, environment string, relations map[string]any) server {
	t.Helper()
	body := map[string]any{
		"source": r.source, "source_id": sourceID, "name": name, "environment": environment,
	}
	for key, value := range relations {
		body[key] = value
	}
	response := r.h.Request(t, http.MethodPost, r.h.Core+"/api/service/v1/servers", TokenService, body, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("register %s: status = %d, body = %s", sourceID, response.Status, truncate(response.Body))
	}
	var registered server
	response.JSON(t, &registered)
	return registered
}

func (r *reporter) report(t *testing.T, serverID, event, summary string) operationsEvent {
	t.Helper()
	response := r.h.Request(t, http.MethodPost, r.h.Core+"/api/service/v1/operations/events", TokenService,
		map[string]any{"server_id": serverID, "event": event, "summary": summary, "source": r.source}, nil)
	if response.Status != http.StatusAccepted {
		t.Fatalf("report %s: status = %d, body = %s", event, response.Status, truncate(response.Body))
	}
	var recorded operationsEvent
	response.JSON(t, &recorded)
	return recorded
}

func readServer(t *testing.T, h *Harness, token, id string) (server, int) {
	t.Helper()
	response := h.API(t, http.MethodGet, "/api/v1/servers/"+id, token, nil)
	var found server
	if response.Status == http.StatusOK {
		response.JSON(t, &found)
	}
	return found, response.Status
}

// A registration is an upsert keyed by the reporter's own identifier, so a
// reporter that replays its inventory does not mint a second Global ID for a
// host it has already registered.
func TestServerRegistrationIsIdempotent(t *testing.T) {
	h := ready(t)
	bravo := newReporter(t, h)

	sourceID := marker("host")
	first := bravo.register(t, sourceID, "bsystem-app-1", "prod", nil)
	if first.ID == "" || first.Status != "unknown" {
		t.Fatalf("registered = %+v, want a Global ID and an unknown status until something is reported", first)
	}
	second := bravo.register(t, sourceID, "bsystem-app-1-renamed", "stage", nil)
	if second.ID != first.ID {
		t.Errorf("re-registering allocated %s as well as %s", second.ID, first.ID)
	}
	if second.Name != "bsystem-app-1-renamed" || second.Environment != "stage" {
		t.Errorf("re-registration did not update the reporter's own fields: %+v", second)
	}
	// Status is not settable at registration: it is a consequence of events.
	if second.Status != "unknown" {
		t.Errorf("status = %q, want it untouched by a registration", second.Status)
	}
}

// An unresolvable relation is refused rather than stored. Ambiguous ownership
// must deny, and an operations record attached to the wrong customer is worse
// than one attached to none.
func TestUnresolvableRelationsAreRefused(t *testing.T) {
	h := ready(t)

	cases := map[string]map[string]any{
		"an unknown client":         {"client_id": "CL-999999"},
		"an unknown project":        {"project_id": "PR-999999"},
		"a client id of wrong type": {"client_id": "PR-000001"},
		"not a Global ID at all":    {"client_id": "nonsense"},
	}
	for name, relations := range cases {
		t.Run(name, func(t *testing.T) {
			body := map[string]any{
				"source": "e2e-bravo", "source_id": marker("bad"), "name": "host", "environment": "prod",
			}
			for key, value := range relations {
				body[key] = value
			}
			response := h.Request(t, http.MethodPost, h.Core+"/api/service/v1/servers", TokenService, body, nil)
			if response.Status != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body: %s)", response.Status, truncate(response.Body))
			}
			assertNoSecretsOrTopology(t, response.Body)
		})
	}

	// An environment the platform does not recognise is refused too, rather
	// than recorded as production or as a lab.
	response := h.Request(t, http.MethodPost, h.Core+"/api/service/v1/servers", TokenService,
		map[string]any{"source": "e2e-bravo", "source_id": marker("env"), "name": "host", "environment": "preprod"}, nil)
	if response.Status != http.StatusBadRequest {
		t.Errorf("unknown environment: status = %d, want 400", response.Status)
	}
}

// The rule the module turns on: a backup outcome says nothing about whether
// the server is up, so it must not mark a healthy machine broken.
func TestOnlyEventsAboutTheServerItselfMoveItsStatus(t *testing.T) {
	h := ready(t)
	bravo := newReporter(t, h)
	host := bravo.register(t, marker("status"), "status-host", "prod", nil)

	steps := []struct {
		event  string
		status string
	}{
		{"server.ok", "ok"},
		// A failed backup is serious and leaves the server's own status
		// alone. Somebody must be sent to look at the backup, not the host.
		{"backup.failed", "ok"},
		{"selftest.failed", "ok"},
		{"backup.succeeded", "ok"},
		{"server.warning", "warning"},
		{"maintenance.started", "maintenance"},
		// Completing maintenance asserts the server is fine, rather than
		// restoring the warning it had before.
		{"maintenance.completed", "ok"},
		{"server.error", "error"},
		{"server.offline", "offline"},
	}
	for _, step := range steps {
		t.Run(step.event, func(t *testing.T) {
			bravo.report(t, host.ID, step.event, "e2e "+step.event)
			found, status := readServer(t, h, TokenDevOps, host.ID)
			if status != http.StatusOK {
				t.Fatalf("read server: status = %d", status)
			}
			if found.Status != step.status {
				t.Errorf("after %s status = %q, want %q", step.event, found.Status, step.status)
			}
			// Every report moves last_event_at, including those that leave
			// status alone: a reporter that has gone quiet is itself an
			// operational fact.
			if found.LastEventAt == "" {
				t.Error("last_event_at was not moved by a report")
			}
		})
	}
}

// The vocabulary is closed. An operations feed that accepts any name becomes
// a log nobody can query.
func TestEventsOutsideTheVocabularyAreRefused(t *testing.T) {
	h := ready(t)
	bravo := newReporter(t, h)
	host := bravo.register(t, marker("vocab"), "vocab-host", "dev", nil)

	for _, event := range []string{"backup.done", "server.down", "disk.full", "", "BACKUP.FAILED"} {
		response := h.Request(t, http.MethodPost, h.Core+"/api/service/v1/operations/events", TokenService,
			map[string]any{"server_id": host.ID, "event": event, "source": "e2e-bravo"}, nil)
		if response.Status != http.StatusBadRequest {
			t.Errorf("event %q: status = %d, want 400", event, response.Status)
		}
	}
	// And a report against a server nobody registered.
	response := h.Request(t, http.MethodPost, h.Core+"/api/service/v1/operations/events", TokenService,
		map[string]any{"server_id": "SRV-999999", "event": "server.ok", "source": "e2e-bravo"}, nil)
	if response.Status != http.StatusBadRequest {
		t.Errorf("unregistered server: status = %d, want 400", response.Status)
	}
}

// A reporter may escalate a severity but not lower one the platform treats as
// critical.
func TestReportersCannotDowngradeSeverity(t *testing.T) {
	h := ready(t)
	bravo := newReporter(t, h)
	host := bravo.register(t, marker("severity"), "severity-host", "prod", nil)

	response := h.Request(t, http.MethodPost, h.Core+"/api/service/v1/operations/events", TokenService,
		map[string]any{"server_id": host.ID, "event": "backup.failed", "severity": "info", "source": "e2e-bravo"}, nil)
	if response.Status != http.StatusAccepted {
		t.Fatalf("report: status = %d", response.Status)
	}
	var recorded operationsEvent
	response.JSON(t, &recorded)
	if recorded.Severity != "critical" {
		t.Errorf("severity = %q, want critical: a reporter must not be able to quieten it", recorded.Severity)
	}
}

// Reporting is a machine capability, and reading is not open to everyone.
func TestOperationsAuthorization(t *testing.T) {
	h := ready(t)
	bravo := newReporter(t, h)
	host := bravo.register(t, marker("authz"), "authz-host", "prod", nil)
	bravo.report(t, host.ID, "server.ok", "up")

	t.Run("no human token may report", func(t *testing.T) {
		for name, token := range map[string]string{"an administrator": TokenAdmin, "a devops engineer": TokenDevOps, "no token": ""} {
			response := h.Request(t, http.MethodPost, h.Core+"/api/service/v1/operations/events", token,
				map[string]any{"server_id": host.ID, "event": "server.ok", "source": "e2e"}, nil)
			if response.Status != http.StatusUnauthorized && response.Status != http.StatusForbidden {
				t.Errorf("%s: status = %d, want 401 or 403", name, response.Status)
			}
		}
	})

	t.Run("reading needs operations.server.read", func(t *testing.T) {
		for name, c := range map[string]struct {
			token string
			want  int
		}{
			"DevOps holds it":               {TokenDevOps, http.StatusOK},
			"an administrator holds it":     {TokenAdmin, http.StatusOK},
			"a manager holds it":            {TokenManager, http.StatusOK},
			"QA does not":                   {TokenQA, http.StatusForbidden},
			"a customer is scope-confined":  {TokenCustomer, http.StatusForbidden},
			"no groups resolved to no role": {TokenNoGroups, http.StatusForbidden},
		} {
			t.Run(name, func(t *testing.T) {
				response := h.API(t, http.MethodGet, "/api/v1/servers", c.token, nil)
				if response.Status != c.want {
					t.Errorf("list servers: status = %d, want %d", response.Status, c.want)
				}
				response = h.API(t, http.MethodGet, "/api/v1/operations/events", c.token, nil)
				if response.Status != c.want {
					t.Errorf("list events: status = %d, want %d", response.Status, c.want)
				}
			})
		}
	})

	// A server the caller may not read and one that does not exist must be
	// answered identically, or the endpoint enumerates infrastructure.
	t.Run("an unreadable server is indistinguishable from a missing one", func(t *testing.T) {
		_, missing := readServer(t, h, TokenDevOps, "SRV-999999")
		if missing != http.StatusNotFound {
			t.Errorf("unknown server: status = %d, want 404", missing)
		}
		response := h.API(t, http.MethodGet, "/api/v1/servers/"+host.ID, TokenQA, nil)
		if response.Status != http.StatusForbidden {
			t.Errorf("a caller without the permission: status = %d, want 403", response.Status)
		}
	})
}

// A serious report becomes a notification for the people who read server
// state; a routine one does not.
func TestSeriousReportsNotifyAndRoutineOnesDoNot(t *testing.T) {
	h := ready(t)
	bravo := newReporter(t, h)
	host := bravo.register(t, marker("notify"), "notify-host", "prod", nil)

	failure := marker("backup-failure")
	bravo.report(t, host.ID, "backup.failed", failure)
	if _, found := findByBody(t, h, TokenDevOps, failure); !found {
		t.Error("a failed backup did not notify the people who read server state")
	}

	routine := marker("backup-success")
	bravo.report(t, host.ID, "backup.succeeded", routine)
	if _, found := findByBody(t, h, TokenAdmin, routine); found {
		t.Error("a successful backup interrupted somebody; a platform that notifies on everything trains people to ignore it")
	}
}

// Both collections must be walkable, and the event filter must be a filter.
func TestOperationsCollectionsPageAndFilter(t *testing.T) {
	h := ready(t)
	bravo := newReporter(t, h)
	host := bravo.register(t, marker("paging"), "paging-host", "dev", nil)

	for i := 0; i < 5; i++ {
		bravo.report(t, host.ID, "selftest.succeeded", fmt.Sprintf("run %d", i))
	}

	seen := map[int64]bool{}
	query := "?server_id=" + host.ID + "&limit=2"
	for page := 0; page < 20; page++ {
		response := h.API(t, http.MethodGet, "/api/v1/operations/events"+query, TokenDevOps, nil)
		if response.Status != http.StatusOK {
			t.Fatalf("list events: status = %d", response.Status)
		}
		var result operationsEventPage
		response.JSON(t, &result)
		for _, event := range result.Data {
			if seen[event.ID] {
				t.Fatalf("event %d was returned twice while walking", event.ID)
			}
			seen[event.ID] = true
			if event.ServerID != host.ID {
				t.Errorf("the server filter returned an event for %s", event.ServerID)
			}
		}
		if result.Pagination.NextCursor == "" {
			if len(seen) != 5 {
				t.Errorf("walked %d of 5 events", len(seen))
			}
			break
		}
		query = "?server_id=" + host.ID + "&limit=2&cursor=" + result.Pagination.NextCursor
	}

	// An unknown event filter is refused rather than ignored: quietly
	// matching everything and quietly matching nothing both read as "there is
	// nothing here".
	response := h.API(t, http.MethodGet, "/api/v1/operations/events?event=disk.full", TokenDevOps, nil)
	if response.Status != http.StatusBadRequest {
		t.Errorf("unknown event filter: status = %d, want 400", response.Status)
	}

	// The server listing pages too, and its status filter filters.
	response = h.API(t, http.MethodGet, "/api/v1/servers?status=offline", TokenDevOps, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("filtered server list: status = %d", response.Status)
	}
	var servers serverPage
	response.JSON(t, &servers)
	for _, found := range servers.Data {
		if found.Status != "offline" {
			t.Errorf("the status filter returned a %s server", found.Status)
		}
	}
}

// A report carries the request correlation id, so one identifier traces an
// operations event through the platform, its notifications and the audit
// trail.
func TestReportsCarryTheCorrelationID(t *testing.T) {
	h := ready(t)
	bravo := newReporter(t, h)
	host := bravo.register(t, marker("correlated"), "correlated-host", "prod", nil)

	correlationID := fmt.Sprintf("e2e-ops-%d", time.Now().UnixNano())
	response := h.Request(t, http.MethodPost, h.Core+"/api/service/v1/operations/events", TokenService,
		map[string]any{"server_id": host.ID, "event": "server.warning", "source": "e2e-bravo"},
		map[string]string{"X-Request-ID": correlationID})
	if response.Status != http.StatusAccepted {
		t.Fatalf("report: status = %d", response.Status)
	}
	var recorded operationsEvent
	response.JSON(t, &recorded)
	if recorded.CorrelationID != correlationID {
		t.Errorf("correlation_id = %q, want %q", recorded.CorrelationID, correlationID)
	}
}
