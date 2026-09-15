package mockhttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	server := NewServer("test-mock")
	server.Handle("GET /api/items", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "normal"})
	})
	server.Handle("GET /api/other", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "other"})
	})
	return server
}

func do(t *testing.T, handler http.Handler, method, target string, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader = strings.NewReader(body)
	request := httptest.NewRequest(method, target, reader)
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestHealthIsAlwaysServed(t *testing.T) {
	server := newTestServer(t)
	if err := server.Faults.Add(Fault{Path: FaultMatchAll, Status: http.StatusInternalServerError, Remaining: 10}); err != nil {
		t.Fatalf("add fault: %v", err)
	}
	response := do(t, server.Handler(), http.MethodGet, "/health", "", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("health must bypass injected faults, got %d", response.Code)
	}
}

func TestFaultInjection(t *testing.T) {
	tests := []struct {
		name       string
		fault      Fault
		target     string
		wantStatus int
		wantHeader string
	}{
		{name: "unauthorized", fault: Fault{Path: "/api/items", Status: http.StatusUnauthorized}, target: "/api/items", wantStatus: http.StatusUnauthorized},
		{name: "forbidden", fault: Fault{Path: "/api/items", Status: http.StatusForbidden}, target: "/api/items", wantStatus: http.StatusForbidden},
		{name: "not found", fault: Fault{Path: "/api/items", Status: http.StatusNotFound}, target: "/api/items", wantStatus: http.StatusNotFound},
		{name: "rate limited sets retry-after", fault: Fault{Path: "/api/items", Status: http.StatusTooManyRequests}, target: "/api/items", wantStatus: http.StatusTooManyRequests, wantHeader: "1"},
		{name: "server error", fault: Fault{Path: "/api/items", Status: http.StatusInternalServerError}, target: "/api/items", wantStatus: http.StatusInternalServerError},
		{name: "catch-all applies to any path", fault: Fault{Path: FaultMatchAll, Status: http.StatusBadGateway}, target: "/api/other", wantStatus: http.StatusBadGateway},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newTestServer(t)
			if err := server.Faults.Add(test.fault); err != nil {
				t.Fatalf("add fault: %v", err)
			}
			response := do(t, server.Handler(), http.MethodGet, test.target, "", nil)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if test.wantHeader != "" && response.Header().Get("Retry-After") != test.wantHeader {
				t.Fatalf("Retry-After = %q, want %q", response.Header().Get("Retry-After"), test.wantHeader)
			}
			// The fault is consumed, so the next request is served normally.
			again := do(t, server.Handler(), http.MethodGet, test.target, "", nil)
			if again.Code != http.StatusOK {
				t.Fatalf("fault was not consumed: status = %d", again.Code)
			}
		})
	}
}

func TestFaultRemainingIsHonoured(t *testing.T) {
	server := newTestServer(t)
	if err := server.Faults.Add(Fault{Path: "/api/items", Status: http.StatusInternalServerError, Remaining: 2}); err != nil {
		t.Fatalf("add fault: %v", err)
	}
	for attempt := 1; attempt <= 2; attempt++ {
		if response := do(t, server.Handler(), http.MethodGet, "/api/items", "", nil); response.Code != http.StatusInternalServerError {
			t.Fatalf("attempt %d status = %d, want 500", attempt, response.Code)
		}
	}
	if response := do(t, server.Handler(), http.MethodGet, "/api/items", "", nil); response.Code != http.StatusOK {
		t.Fatalf("third attempt status = %d, want 200", response.Code)
	}
}

func TestFaultDelayIsApplied(t *testing.T) {
	server := newTestServer(t)
	if err := server.Faults.Add(Fault{Path: "/api/items", DelayMS: 60}); err != nil {
		t.Fatalf("add fault: %v", err)
	}
	start := time.Now()
	response := do(t, server.Handler(), http.MethodGet, "/api/items", "", nil)
	elapsed := time.Since(start)
	// A delay-only fault still yields the real response, which is what lets a
	// timeout scenario exercise the adapter rather than an error body.
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if elapsed < 50*time.Millisecond {
		t.Fatalf("delay was not applied: elapsed = %s", elapsed)
	}
}

func TestFaultValidation(t *testing.T) {
	tests := []struct {
		name    string
		fault   Fault
		wantErr bool
	}{
		{name: "status only", fault: Fault{Path: "/a", Status: 500}},
		{name: "delay only", fault: Fault{Path: "/a", DelayMS: 10}},
		{name: "empty path defaults to catch-all", fault: Fault{Status: 500}},
		{name: "neither status nor delay", fault: Fault{Path: "/a"}, wantErr: true},
		{name: "impossible status", fault: Fault{Path: "/a", Status: 99}, wantErr: true},
		{name: "negative delay", fault: Fault{Path: "/a", DelayMS: -1}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := NewFaults().Add(test.fault)
			if test.wantErr != (err != nil) {
				t.Fatalf("Add() error = %v, wantErr = %v", err, test.wantErr)
			}
		})
	}
}

func TestFaultControlEndpoints(t *testing.T) {
	server := newTestServer(t)
	handler := server.Handler()

	if response := do(t, handler, http.MethodPost, "/__mock/faults", `{"path":"/api/items","status":503}`, nil); response.Code != http.StatusAccepted {
		t.Fatalf("add status = %d, want 202", response.Code)
	}
	if response := do(t, handler, http.MethodPost, "/__mock/faults", `not json`, nil); response.Code != http.StatusBadRequest {
		t.Fatalf("malformed body status = %d, want 400", response.Code)
	}

	listed := do(t, handler, http.MethodGet, "/__mock/faults", "", nil)
	var pending []Fault
	if err := json.Unmarshal(listed.Body.Bytes(), &pending); err != nil {
		t.Fatalf("decode pending faults: %v", err)
	}
	if len(pending) != 1 || pending[0].Status != http.StatusServiceUnavailable {
		t.Fatalf("unexpected pending faults: %+v", pending)
	}

	if response := do(t, handler, http.MethodDelete, "/__mock/faults", "", nil); response.Code != http.StatusNoContent {
		t.Fatalf("reset status = %d, want 204", response.Code)
	}
	if response := do(t, handler, http.MethodGet, "/api/items", "", nil); response.Code != http.StatusOK {
		t.Fatalf("after reset status = %d, want 200", response.Code)
	}
}

func TestRequireHeaderAndBearer(t *testing.T) {
	const secret = "test-secret-value"
	tests := []struct {
		name       string
		headers    map[string]string
		bearer     bool
		wantStatus int
	}{
		{name: "matching api key", headers: map[string]string{"X-Api-Key": secret}, wantStatus: http.StatusOK},
		{name: "wrong api key", headers: map[string]string{"X-Api-Key": "wrong"}, wantStatus: http.StatusUnauthorized},
		{name: "missing api key", wantStatus: http.StatusUnauthorized},
		{name: "matching bearer", headers: map[string]string{"Authorization": "Bearer " + secret}, bearer: true, wantStatus: http.StatusOK},
		{name: "wrong bearer", headers: map[string]string{"Authorization": "Bearer wrong"}, bearer: true, wantStatus: http.StatusUnauthorized},
		{name: "non-bearer scheme", headers: map[string]string{"Authorization": secret}, bearer: true, wantStatus: http.StatusUnauthorized},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ok := false
				if test.bearer {
					ok = RequireBearer(w, r, secret)
				} else {
					ok = RequireHeader(w, r, "X-Api-Key", secret)
				}
				if ok {
					WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
				}
			})
			response := do(t, handler, http.MethodGet, "/api/items", "", test.headers)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			// A rejection must never echo the expected or supplied credential.
			if strings.Contains(response.Body.String(), secret) {
				t.Fatal("response body leaked a credential")
			}
		})
	}
}

func TestPage(t *testing.T) {
	tests := []struct {
		name                   string
		total, offset, limit   int
		wantStart, wantEnd     int
		defaultLimit, maxLimit int
	}{
		{name: "full page", total: 5, offset: 0, limit: 10, defaultLimit: 25, maxLimit: 100, wantStart: 0, wantEnd: 5},
		{name: "first page", total: 5, offset: 0, limit: 2, defaultLimit: 25, maxLimit: 100, wantStart: 0, wantEnd: 2},
		{name: "middle page", total: 5, offset: 2, limit: 2, defaultLimit: 25, maxLimit: 100, wantStart: 2, wantEnd: 4},
		{name: "last partial page", total: 5, offset: 4, limit: 2, defaultLimit: 25, maxLimit: 100, wantStart: 4, wantEnd: 5},
		{name: "offset past end", total: 5, offset: 9, limit: 2, defaultLimit: 25, maxLimit: 100, wantStart: 5, wantEnd: 5},
		{name: "zero limit falls back", total: 5, offset: 0, limit: 0, defaultLimit: 3, maxLimit: 100, wantStart: 0, wantEnd: 3},
		{name: "limit is clamped to max", total: 5, offset: 0, limit: 5000, defaultLimit: 25, maxLimit: 2, wantStart: 0, wantEnd: 2},
		{name: "negative offset is clamped", total: 5, offset: -3, limit: 2, defaultLimit: 25, maxLimit: 100, wantStart: 0, wantEnd: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start, end := Page(test.total, test.offset, test.limit, test.defaultLimit, test.maxLimit)
			if start != test.wantStart || end != test.wantEnd {
				t.Fatalf("Page() = (%d,%d), want (%d,%d)", start, end, test.wantStart, test.wantEnd)
			}
		})
	}
}

func TestQueryInt(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   int
	}{
		{name: "missing", target: "/api/items", want: 7},
		{name: "valid", target: "/api/items?limit=3", want: 3},
		{name: "zero", target: "/api/items?limit=0", want: 0},
		{name: "malformed", target: "/api/items?limit=abc", want: 7},
		{name: "negative", target: "/api/items?limit=-2", want: 7},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.target, nil)
			if got := QueryInt(request, "limit", 7); got != test.want {
				t.Fatalf("QueryInt() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestSecretFallback(t *testing.T) {
	if got := Secret("BSYSTEM_MOCKHTTP_UNSET_VALUE", "fallback"); got != "fallback" {
		t.Fatalf("Secret() = %q, want fallback", got)
	}
	t.Setenv("BSYSTEM_MOCKHTTP_TEST_VALUE", "configured")
	if got := Secret("BSYSTEM_MOCKHTTP_TEST_VALUE", "fallback"); got != "configured" {
		t.Fatalf("Secret() = %q, want configured", got)
	}
}
