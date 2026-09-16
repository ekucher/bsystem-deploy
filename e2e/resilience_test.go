package e2e

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// --- Pagination -------------------------------------------------------------

// Walking a collection by cursor must visit every item exactly once and
// terminate, whatever the page size.
func TestCollectionsAreWalkableByCursor(t *testing.T) {
	harness := ready(t)
	tests := []struct {
		name      string
		path      string
		wantTotal int
	}{
		{name: "clients", path: "/api/v1/clients", wantTotal: 3},
		{name: "contacts", path: "/api/v1/contacts", wantTotal: 4},
		{name: "projects", path: "/api/v1/projects", wantTotal: 3},
		{name: "issues", path: "/api/v1/issues", wantTotal: 5},
		{name: "documents", path: "/api/v1/documents", wantTotal: 4},
	}
	for _, test := range tests {
		for _, size := range []int{1, 2, 3, 100} {
			t.Run(fmt.Sprintf("%s/limit=%d", test.name, size), func(t *testing.T) {
				seen := map[string]bool{}
				order := []string{}
				cursor := ""
				for pages := 0; ; pages++ {
					if pages > test.wantTotal+2 {
						t.Fatalf("walking %s did not terminate after %d pages", test.path, pages)
					}
					path := fmt.Sprintf("%s?limit=%d", test.path, size)
					if cursor != "" {
						path += "&cursor=" + cursor
					}
					response := harness.API(t, http.MethodGet, path, TokenAdmin, nil)
					if response.Status != http.StatusOK {
						t.Fatalf("status = %d, want 200 (body: %s)", response.Status, truncate(response.Body))
					}
					var body collectionResponse
					response.JSON(t, &body)

					if body.Pagination.Total != test.wantTotal {
						t.Fatalf("pagination.total = %d, want %d: the total describes the collection, not the page", body.Pagination.Total, test.wantTotal)
					}
					if len(body.Data) > size {
						t.Fatalf("page returned %d items, more than the requested limit of %d", len(body.Data), size)
					}
					if body.Pagination.Limit != len(body.Data) {
						t.Fatalf("pagination.limit = %d, want %d", body.Pagination.Limit, len(body.Data))
					}
					for _, item := range body.Data {
						if seen[item.ID] {
							t.Fatalf("%s appeared on more than one page", item.ID)
						}
						seen[item.ID] = true
						order = append(order, item.ID)
					}
					if body.Pagination.NextCursor == "" {
						break
					}
					cursor = body.Pagination.NextCursor
				}
				if len(order) != test.wantTotal {
					t.Fatalf("walked %d items, want %d", len(order), test.wantTotal)
				}
			})
		}
	}
}

// Page size is clamped rather than rejected, so an over-large request gets the
// maximum instead of an error.
func TestOversizedLimitIsClamped(t *testing.T) {
	harness := ready(t)
	response := harness.API(t, http.MethodGet, "/api/v1/clients?limit=100000", TokenAdmin, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", response.Status, truncate(response.Body))
	}
	var body collectionResponse
	response.JSON(t, &body)
	if len(body.Data) != 3 {
		t.Fatalf("returned %d items, want the whole collection", len(body.Data))
	}
}

// A cursor the platform did not issue cannot be interpreted. Reinterpreting
// one would silently return the wrong window.
func TestForgedCursorsAreRejected(t *testing.T) {
	harness := ready(t)
	tests := []struct {
		name   string
		cursor string
	}{
		{name: "not base64", cursor: "not-a-cursor"},
		{name: "base64 of something else", cursor: "aGVsbG8"},
		{name: "a raw offset", cursor: "100"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := harness.API(t, http.MethodGet, "/api/v1/clients?cursor="+test.cursor, TokenAdmin, nil)
			if response.Status != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body: %s)", response.Status, truncate(response.Body))
			}
			var failure platformError
			response.JSON(t, &failure)
			if failure.Code != "invalid_cursor" {
				t.Fatalf("code = %q, want invalid_cursor", failure.Code)
			}
			assertNoSecretsOrTopology(t, response.Body)
		})
	}
}

// --- Retries ---------------------------------------------------------------

// A transient upstream failure must be absorbed by the adapter rather than
// surfacing to the caller. This is the behaviour that makes a dropped packet
// invisible instead of a failed page.
func TestTransientUpstreamFailuresAreRetried(t *testing.T) {
	harness := ready(t)
	tests := []struct {
		name     string
		mockURL  string
		mockPath string
		status   int
		apiPath  string
	}{
		{name: "espocrm rate limit", mockURL: harness.EspoCRM, mockPath: "/api/v1/Account", status: http.StatusTooManyRequests, apiPath: "/api/v1/clients"},
		{name: "redmine server error", mockURL: harness.Redmine, mockPath: "/projects.json", status: http.StatusInternalServerError, apiPath: "/api/v1/projects"},
		{name: "outline gateway error", mockURL: harness.Outline, mockPath: "/api/documents.list", status: http.StatusBadGateway, apiPath: "/api/v1/documents"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// One failure, then success: within the retry budget, so the
			// caller should never see it.
			harness.InjectFault(t, test.mockURL, test.mockPath, test.status, 0)
			response := harness.API(t, http.MethodGet, test.apiPath, TokenAdmin, nil)
			if response.Status != http.StatusOK {
				t.Fatalf("status = %d, want 200: a single transient failure must be retried (body: %s)", response.Status, truncate(response.Body))
			}
		})
	}
}

// Retries are bounded. Once the budget is spent the caller gets a normalized
// error rather than the platform trying indefinitely.
func TestRetriesAreBounded(t *testing.T) {
	harness := ready(t)
	// More failures than the retry budget allows.
	harness.InjectFaultTimes(t, harness.EspoCRM, "/api/v1/Account", http.StatusServiceUnavailable, 0, 10)

	start := time.Now()
	response := harness.API(t, http.MethodGet, "/api/v1/clients", TokenAdmin, nil)
	elapsed := time.Since(start)

	assertNormalizedUpstreamFailure(t, response, "espocrm")
	// Bounded attempts with a capped backoff cannot take anywhere near the
	// adapter timeout, let alone the client's.
	if elapsed > 20*time.Second {
		t.Fatalf("a bounded retry budget took %s", elapsed)
	}
}

// An authorization failure is deterministic: retrying it only wastes the
// upstream's capacity and delays the caller's error.
func TestDeterministicFailuresAreNotRetried(t *testing.T) {
	harness := ready(t)
	// Exactly one injected rejection. If the adapter retried, the second
	// attempt would succeed and the caller would see 200.
	harness.InjectFault(t, harness.EspoCRM, "/api/v1/Account", http.StatusUnauthorized, 0)
	response := harness.API(t, http.MethodGet, "/api/v1/clients", TokenAdmin, nil)
	assertNormalizedUpstreamFailure(t, response, "espocrm")
}

// --- Circuit breaker -------------------------------------------------------

// A sustained outage must stop being called, and the platform must say so
// without disclosing anything about the upstream.
func TestCircuitOpensUnderSustainedFailure(t *testing.T) {
	harness := ready(t)
	// Far more failures than the breaker threshold, across enough requests
	// that the circuit certainly opens.
	harness.InjectFaultTimes(t, harness.Redmine, "/projects.json", http.StatusServiceUnavailable, 0, 100)

	for attempt := 0; attempt < 6; attempt++ {
		response := harness.API(t, http.MethodGet, "/api/v1/projects", TokenAdmin, nil)
		assertNormalizedUpstreamFailure(t, response, "redmine")
	}

	// Once open, the circuit is visible in readiness without making the
	// platform unready: the rest of the API keeps working.
	readiness := harness.API(t, http.MethodGet, "/readyz", "", nil)
	if readiness.Status != http.StatusOK {
		t.Fatalf("readiness = %d, want 200: one shed upstream must not make the platform unready", readiness.Status)
	}
	var state struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	readiness.JSON(t, &state)
	if got := state.Checks["adapter:redmine"]; !strings.HasPrefix(got, "circuit_") {
		t.Fatalf("readiness checks = %v, want the redmine circuit reported", state.Checks)
	}

	// An unaffected upstream keeps serving while one is shed.
	clients := harness.API(t, http.MethodGet, "/api/v1/clients", TokenAdmin, nil)
	if clients.Status != http.StatusOK {
		t.Fatalf("an unrelated adapter returned %d; one failing upstream must not take out another", clients.Status)
	}

	// The circuit state is exported for alerting.
	metrics := harness.API(t, http.MethodGet, "/metrics", "", nil)
	if metrics.Status != http.StatusOK {
		t.Fatalf("metrics status = %d, want 200", metrics.Status)
	}
	// The assertion is that the circuit is not closed, rather than that it is
	// exactly open: with a short window the breaker may already have moved to
	// half_open by the time metrics are scraped, and both states mean load is
	// being shed.
	if !strings.Contains(string(metrics.Body), `bsystem_adapter_circuit_state{adapter="redmine",state="closed"} 0`) {
		t.Fatalf("a shed upstream must be exported as a non-closed circuit:\n%s", truncate(metrics.Body))
	}
	assertNoSecretsOrTopology(t, metrics.Body)

	// Shedding load must be temporary. Once the upstream recovers, the
	// circuit has to close again on its own — a breaker that never resets is
	// an outage of its own making.
	harness.ResetFaults(t, harness.Redmine)
	recovered := false
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if response := harness.API(t, http.MethodGet, "/api/v1/projects", TokenAdmin, nil); response.Status == http.StatusOK {
			recovered = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !recovered {
		t.Fatal("the circuit did not close after the upstream recovered")
	}
}

// A deployment that does not use one of the integrations is a supported
// configuration, not a fault, and it is the one a stage acceptance is most
// likely to meet — a customer who runs no wiki, or an environment brought up
// before the Outline credentials exist.
//
// Nothing in this stack proved the platform survives it. The single core has
// every adapter configured, so "starts without Outline" was an assumption
// about a deployment nobody had ever run. The stack now runs one: a second
// Integration Core, same image, same database, with OUTLINE_URL absent.
//
// What must hold is that the missing integration is contained. The process
// starts, readiness passes, the other integrations serve normally, and the one
// that is absent says so with a stable code — not a 500, and not the "does not
// support this capability" answer, which reads as a permanent limit of the
// product rather than an environment variable nobody set.
func TestACoreWithAnUnconfiguredAdapterStartsAndSaysSo(t *testing.T) {
	harness := ready(t)

	// The partial core is a separate process with its own startup. Waiting on
	// its readiness is not incidental to the scenario — starting at all is
	// half of what is being proved.
	deadline := time.Now().Add(60 * time.Second)
	var readiness Response
	for {
		response, err := harness.Do(http.MethodGet, harness.Partial+"/readyz", "", nil)
		if err == nil && response.Status == http.StatusOK {
			readiness = response
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the core with no Outline configuration never became ready: %v (last status: %d)", err, response.Status)
		}
		time.Sleep(500 * time.Millisecond)
	}
	var state struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	readiness.JSON(t, &state)
	if state.Status != "ready" {
		t.Errorf("readiness status = %q, want ready (checks: %v)", state.Status, state.Checks)
	}

	// The absent integration.
	documents, err := harness.Do(http.MethodGet, harness.Partial+"/api/v1/documents", TokenAdmin, nil)
	if err != nil {
		t.Fatalf("GET documents from the partial core: %v", err)
	}
	if documents.Status != http.StatusServiceUnavailable {
		t.Fatalf("GET /api/v1/documents: status = %d, want %d (body: %s)", documents.Status, http.StatusServiceUnavailable, truncate(documents.Body))
	}
	var failure struct {
		Error  string `json:"error"`
		Code   string `json:"code"`
		Source string `json:"source"`
	}
	documents.JSON(t, &failure)
	if failure.Code != "adapter_not_configured" {
		t.Errorf("error code = %q, want %q (body: %s)", failure.Code, "adapter_not_configured", truncate(documents.Body))
	}
	if failure.Source != "outline" {
		t.Errorf("error source = %q, want %q", failure.Source, "outline")
	}

	// The integrations that are configured must be unaffected. A platform that
	// refuses everything because one integration is missing is not a partial
	// deployment, it is a broken one.
	for _, path := range []string{"/api/v1/clients", "/api/v1/contacts", "/api/v1/projects", "/api/v1/issues"} {
		response, err := harness.Do(http.MethodGet, harness.Partial+path, TokenAdmin, nil)
		if err != nil {
			t.Fatalf("GET %s from the partial core: %v", path, err)
		}
		if response.Status != http.StatusOK {
			t.Errorf("GET %s: status = %d, want 200 (body: %s)", path, response.Status, truncate(response.Body))
		}
	}

	// The registry describes the whole intended surface rather than omitting
	// what is not configured, so an operator can tell "disabled" from "this
	// build has no such integration".
	health, err := harness.Do(http.MethodGet, harness.Partial+"/api/service/v1/adapters/health", TokenService, nil)
	if err != nil {
		t.Fatalf("GET adapter health from the partial core: %v", err)
	}
	if health.Status != http.StatusOK {
		t.Fatalf("GET /api/service/v1/adapters/health: status = %d, want 200 (body: %s)", health.Status, truncate(health.Body))
	}
	var report map[string]struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	health.JSON(t, &report)
	outline, present := report["outline"]
	if !present {
		t.Fatalf("outline is missing from the partial core's health report rather than being reported disabled: %+v", report)
	}
	if outline.Status != "disabled" {
		t.Errorf("outline status = %q (%s), want disabled", outline.Status, outline.Message)
	}
	for _, adapter := range []string{"espocrm", "redmine"} {
		if state := report[adapter].Status; state != "ready" {
			t.Errorf("adapter %s status = %q on the partial core, want ready", adapter, state)
		}
	}

	// The fully configured core is a separate process and must be untouched by
	// any of this.
	if response := harness.API(t, http.MethodGet, "/api/v1/documents", TokenAdmin, nil); response.Status != http.StatusOK {
		t.Errorf("the configured core's documents read = %d, want 200 (body: %s)", response.Status, truncate(response.Body))
	}
}
