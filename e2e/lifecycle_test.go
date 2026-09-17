package e2e

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

// --- The guards, which run everywhere ---------------------------------------

// The allowlist and the requirement guard are the two pieces of this file that
// decide whether a lifecycle scenario runs at all and how far it may reach.
// They are plain functions so they can be exercised on any machine, including
// one with no Docker: a guard that is only tested where the stack is up is
// tested exactly where it is least needed.
func TestTheGuardsAroundLifecycleControl(t *testing.T) {
	t.Run("the dependencies a scenario needs are controllable", func(t *testing.T) {
		for _, service := range []string{ServiceNATS, ServicePostgres, ServicePartialCore} {
			if err := checkControllable(service); err != nil {
				t.Errorf("%s is used by a scenario in this file and was refused: %v", service, err)
			}
		}
	})

	// The primary core is the one every other scenario in the package asserts
	// against. Stopping it from here would surface as a failure in an
	// unrelated test, so it is refused by name.
	t.Run("the core the suite depends on is not controllable", func(t *testing.T) {
		for _, service := range []string{"integration-core", "mock-espocrm", "mock-identity", ""} {
			if err := checkControllable(service); err == nil {
				t.Errorf("%q was allowed to be stopped by a scenario", service)
			}
		}
	})

	t.Run("the refusal names what is permitted", func(t *testing.T) {
		err := checkControllable("integration-core")
		if err == nil {
			t.Fatal("no error")
		}
		for _, mention := range []string{ServiceNATS, ServicePostgres} {
			if !strings.Contains(err.Error(), mention) {
				t.Errorf("the refusal does not name %s: %v", mention, err)
			}
		}
	})

	t.Run("a required run that cannot stop a dependency fails rather than skipping", func(t *testing.T) {
		err := checkLifecycleRequired("", "1")
		if err == nil {
			t.Fatal("a required run was allowed to skip every outage scenario")
		}
		if !errors.Is(err, ErrLifecycleRequired) {
			t.Errorf("error does not identify itself: %v", err)
		}
		for _, mention := range []string{"E2E_REQUIRED", "E2E_LIFECYCLE"} {
			if !strings.Contains(err.Error(), mention) {
				t.Errorf("the refusal does not name %s: %v", mention, err)
			}
		}
	})

	t.Run("a required run that can stop a dependency proceeds", func(t *testing.T) {
		if err := checkLifecycleRequired("1", "1"); err != nil {
			t.Errorf("a configured required run was refused: %v", err)
		}
	})

	t.Run("a local run may skip", func(t *testing.T) {
		if err := checkLifecycleRequired("", ""); err != nil {
			t.Errorf("a local run was refused: %v", err)
		}
	})

	// The same whitespace case required_test.go covers, for the same reason: a
	// shell sets a variable to a space more easily than it unsets it.
	t.Run("whitespace is not a value", func(t *testing.T) {
		if err := checkLifecycleRequired("   ", "1"); err == nil {
			t.Error("a whitespace E2E_LIFECYCLE satisfied a required run")
		}
	})
}

// --- The outages ------------------------------------------------------------

// Every scenario below stops a dependency, so none of them may run in
// parallel with anything: the stack is shared and the whole package would see
// the outage. They restore what they stopped through t.Cleanup even when they
// fail, so a failure here stays one failure rather than becoming a wall of
// them in the scenarios that run next.

// docs/STAGE-ACCEPTANCE.md says NATS is the degraded-but-alive dependency:
// /readyz reports it and the platform keeps serving. That was a design
// statement nothing executed. This stops the broker and reads what the
// platform actually does.
func TestTheCoreKeepsServingWithoutTheEventBusAndRecoversWhenItReturns(t *testing.T) {
	harness := ready(t)
	control := LifecycleControl(t)

	before := control.StartedAt(t, "integration-core")

	control.Stop(t, ServiceNATS)

	var degraded readinessReport
	waitFor(t, "the platform to report the event bus degraded", 60*time.Second, func() (bool, string) {
		report, err := harness.readiness(harness.Core)
		if err != nil {
			return false, err.Error()
		}
		degraded = report
		return report.Checks["nats"] == "degraded", report.String()
	})

	// The half that matters: degraded is not down. A platform that stops
	// serving its API because the event bus is gone has turned an optional
	// dependency into a required one, and that is the failure this scenario
	// exists to catch.
	if degraded.code != http.StatusOK || degraded.Status != "ready" {
		t.Errorf("readiness while NATS is gone = %s, want HTTP 200 and status ready: the event bus is documented as degraded, not fatal", degraded)
	}
	if degraded.Checks["database"] != "ok" {
		t.Errorf("the database check reads %q while only NATS is stopped: %s", degraded.Checks["database"], degraded)
	}
	for _, path := range []string{"/api/v1/clients", "/api/v1/projects", "/api/v1/issues"} {
		response := harness.API(t, http.MethodGet, path, TokenAdmin, nil)
		if response.Status != http.StatusOK {
			t.Errorf("GET %s while NATS is gone: status = %d, want 200 (body: %s)", path, response.Status, truncate(response.Body))
		}
	}

	control.Start(t, ServiceNATS)

	waitFor(t, "the platform to reconnect to the event bus", 90*time.Second, func() (bool, string) {
		report, err := harness.readiness(harness.Core)
		if err != nil {
			return false, err.Error()
		}
		return report.Checks["nats"] == "ok", report.String()
	})

	// Reconnection has to be the client's own doing. A core that was restarted
	// by the restart policy also ends up reporting nats: ok, and from the
	// outside the two are the same answer to the same request.
	if after := control.StartedAt(t, "integration-core"); after != before {
		t.Errorf("the Integration Core restarted during the outage (%s -> %s); recovery must not require a restart", before, after)
	}
}

// What the platform says while its database is unreachable is two claims: that
// it becomes unready, and that it does not describe its own database while
// doing so. The second is the one worth a test — an unready platform is
// obvious, and a readiness probe that answers "dial tcp 172.18.0.2:5432:
// connect: connection refused" has published its topology to anyone who can
// reach an unauthenticated endpoint.
func TestTheCoreBecomesUnreadyWithoutItsDatabaseAndDescribesNothing(t *testing.T) {
	harness := ready(t)
	control := LifecycleControl(t)

	before := control.StartedAt(t, "integration-core")

	control.Stop(t, ServicePostgres)

	var unready readinessReport
	waitFor(t, "the platform to report itself unready", 60*time.Second, func() (bool, string) {
		report, err := harness.readiness(harness.Core)
		if err != nil {
			return false, err.Error()
		}
		unready = report
		return report.code == http.StatusServiceUnavailable, report.String()
	})

	if unready.Status != "not_ready" {
		t.Errorf("readiness status = %q, want not_ready: %s", unready.Status, unready)
	}
	if unready.Checks["database"] != "error" {
		t.Errorf("the database check reads %q, want error: %s", unready.Checks["database"], unready)
	}

	// The readiness body is unauthenticated, and so is /health. An API error
	// is reached with a token but by anybody who has one. All three are
	// checked against the same list because a DSN leaking from any of them is
	// the same disclosure.
	probes := map[string]string{"/readyz": unready.raw}
	if health, err := harness.Do(http.MethodGet, harness.Core+"/health", "", nil); err == nil {
		probes["/health"] = string(health.Body)
	}
	if clients, err := harness.Do(http.MethodGet, harness.Core+"/api/v1/clients", TokenAdmin, nil); err == nil {
		probes["/api/v1/clients"] = string(clients.Body)
		if clients.Status < 400 {
			t.Errorf("GET /api/v1/clients answered %d while the database is gone; an authorization decision cannot be made without it", clients.Status)
		}
	}
	// Host, port, user, database name, password, driver and the shape of a Go
	// dial error. The port is matched with its colon: a bare 5432 also occurs
	// inside a hex request id, and a check that flakes is a check that gets
	// removed. Each one of these has appeared in somebody's readiness
	// endpoint by way of an err.Error() passed straight through.
	forbidden := []string{
		"postgres://", "sslmode", "bsystem_integration", ":5432", PostgresTestPassword,
		"pgx", "dial tcp", "connection refused", "@postgres", "password",
	}
	for where, body := range probes {
		lowered := strings.ToLower(body)
		for _, secret := range forbidden {
			if strings.Contains(lowered, strings.ToLower(secret)) {
				t.Errorf("%s described the platform's database while it was unreachable: %q appears in %s", where, secret, truncate([]byte(body)))
			}
		}
	}

	control.Start(t, ServicePostgres)

	waitFor(t, "readiness to recover", 120*time.Second, func() (bool, string) {
		report, err := harness.readiness(harness.Core)
		if err != nil {
			return false, err.Error()
		}
		return report.code == http.StatusOK && report.Checks["database"] == "ok", report.String()
	})

	if after := control.StartedAt(t, "integration-core"); after != before {
		t.Errorf("the Integration Core restarted during the database outage (%s -> %s); the pool is expected to reconnect without one", before, after)
	}
}

// A running platform losing its database and a platform starting without one
// are different contracts, and only the first had ever been considered. The
// Core calls log.Fatalf when it cannot open the database at startup: it fails
// fast and leaves recovery to the restart policy, rather than serving while
// unable to answer anything.
//
// That is a defensible contract and it was undocumented. This scenario pins it
// from both ends: while the database is absent the core never serves, and once
// the database returns the core comes up on its own with no intervention.
//
// The spare core is used rather than the primary one, so the rest of the suite
// keeps its stack.
func TestACoreStartingWithoutADatabaseNeverServesAndComesUpWhenTheDatabaseReturns(t *testing.T) {
	harness := ready(t)
	control := LifecycleControl(t)

	control.Stop(t, ServicePostgres)
	control.Restart(t, ServicePartialCore)

	// Fail-fast means there is no window in which it answers. Polling for one
	// is the assertion: a single check could land between two crashes and
	// prove nothing either way.
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		response, err := harness.Do(http.MethodGet, harness.Partial+"/readyz", "", nil)
		if err == nil && response.Status == http.StatusOK {
			t.Fatalf("a core started with no database answered /readyz with 200: %s", truncate(response.Body))
		}
		time.Sleep(500 * time.Millisecond)
	}

	control.Start(t, ServicePostgres)

	// The recovery is the restart policy's, not a human's. The window is
	// generous because Docker backs off between restarts and the database has
	// its own start-up to do first.
	waitFor(t, "the restarted core to come up once the database returned", 180*time.Second, func() (bool, string) {
		report, err := harness.readiness(harness.Partial)
		if err != nil {
			return false, err.Error()
		}
		return report.code == http.StatusOK && report.Checks["database"] == "ok", report.String()
	})
}
