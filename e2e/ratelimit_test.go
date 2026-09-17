package e2e

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// One noisy identity must not spend another's allowance.
//
// The scenario runs against the spare Integration Core rather than the main
// one, and that is not incidental. The limiter is per process, so spending a
// burst here costs nothing anywhere else — and every other scenario in the
// package talks to the main core. A rate-limit scenario that emptied its
// bucket would otherwise refuse a later scenario's search for a reason that
// has nothing to do with what that scenario is testing, and the failure would
// look like a platform defect.
//
// The spare core's search allowance is turned down in docker-compose.e2e.yml
// so that a burst is reachable from a test at all.
func TestOneNoisyIdentityDoesNotThrottleAnother(t *testing.T) {
	harness := ready(t)

	search := func(token string) Response {
		t.Helper()
		response, err := harness.Do(http.MethodGet, harness.Partial+"/api/v1/search?q=northwind", token, nil)
		if err != nil {
			t.Fatalf("search as %s: %v", token, err)
		}
		return response
	}

	// Make sure the quiet identity works before the noisy one starts, so a
	// later refusal is the limiter's doing rather than a permission.
	if before := search(TokenManager); before.Status != http.StatusOK {
		t.Fatalf("the quiet identity could not search before the test began: status = %d (body: %s)", before.Status, truncate(before.Body))
	}

	refused := 0
	var refusal Response
	for i := 0; i < 40; i++ {
		response := search(TokenAdmin)
		if response.Status == http.StatusTooManyRequests {
			refused++
			refusal = response
		}
	}
	if refused == 0 {
		t.Fatal("the noisy identity was never refused, so the rest of this scenario proves nothing")
	}

	// The whole point: the other identity is untouched.
	for i := 0; i < 5; i++ {
		response := search(TokenManager)
		if response.Status != http.StatusOK {
			t.Fatalf("the quiet identity got %d on request %d while another was being throttled (body: %s)", response.Status, i+1, truncate(response.Body))
		}
	}

	// And the refusal is the platform's normalized shape, with a bounded
	// Retry-After a client can act on.
	var failure struct {
		Error     string `json:"error"`
		Code      string `json:"code"`
		RequestID string `json:"request_id"`
	}
	refusal.JSON(t, &failure)
	if failure.Code != "rate_limited" {
		t.Errorf("code = %q, want rate_limited (body: %s)", failure.Code, truncate(refusal.Body))
	}
	if failure.RequestID == "" {
		t.Error("the refusal carries no request id; a caller reporting it has nothing to quote")
	}
	retryAfter := refusal.Headers.Get("Retry-After")
	seconds, err := strconv.Atoi(retryAfter)
	if err != nil {
		t.Fatalf("Retry-After = %q, which is not a number of seconds", retryAfter)
	}
	if seconds < 1 || seconds > 120 {
		t.Errorf("Retry-After = %ds, want a bounded wait a client can act on", seconds)
	}

	// The metric can say how much of the surface is being refused, and must
	// not say who.
	rejected := harness.metricOrZero(t, harness.Partial, "bsystem_rate_limit_decisions_total",
		map[string]string{"class": "search", "outcome": "rejected"})
	if rejected < float64(refused) {
		t.Errorf("the metric counted %v rejections where %d were observed", rejected, refused)
	}
	body, err := harness.Do(http.MethodGet, harness.Partial+"/metrics", "", nil)
	if err != nil {
		t.Fatalf("read metrics: %v", err)
	}
	for _, line := range strings.Split(string(body.Body), "\n") {
		if !strings.HasPrefix(line, "bsystem_rate_limit_decisions_total") {
			continue
		}
		for _, leak := range []string{"USR-", "SVC-", TokenAdmin, "@"} {
			if strings.Contains(line, leak) {
				t.Errorf("the rate-limit metric identifies who is being limited: %s", line)
			}
		}
	}

	// The main core is a separate process with its own buckets, and every
	// other scenario depends on that being true.
	elsewhere, err := harness.Do(http.MethodGet, harness.Core+"/api/v1/search?q=northwind", TokenAdmin, nil)
	if err != nil {
		t.Fatalf("search on the main core: %v", err)
	}
	if elsewhere.Status != http.StatusOK {
		t.Errorf("the main core answered %d after the spare one's allowance was spent: status = %s", elsewhere.Status, fmt.Sprint(truncate(elsewhere.Body)))
	}
}
