// Package e2e drives the BSYSTEM autonomous E2E stack over HTTP.
//
// The scenarios run against the stack defined in docker-compose.e2e.yml. They
// are skipped unless E2E_BASE_URL points at a running Integration Core, so
// `go test ./...` stays safe on a machine with no stack up.
//
// The package deliberately has no third-party dependencies: it talks plain
// HTTP to the platform and speaks the NATS wire protocol directly.
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// Test-only credentials served by the mock upstreams. They match the defaults
// documented in mocks/README.md and cannot reach any real system.
const (
	TokenAdmin            = "test-token-admin"
	TokenManager          = "test-token-manager"
	TokenDeveloper        = "test-token-developer"
	TokenQA               = "test-token-qa"
	TokenSupport          = "test-token-support"
	TokenDevOps           = "test-token-devops"
	TokenCustomer         = "test-token-customer"
	TokenService          = "test-token-service"
	TokenServiceUngrouped = "test-token-service-ungrouped"
	TokenNoGroups         = "test-token-no-groups"
	TokenExpired          = "test-token-expired"
	TokenUnknown          = "test-token-not-issued"
	EspoCRMTestAPIKey     = "test-espocrm-api-key"
	RedmineTestAPIKey     = "test-redmine-api-key"
	OutlineTestAPIKey     = "test-outline-api-key"
	PostgresTestPassword  = "e2e-test-postgres-password"
)

// Harness holds the endpoints of a running E2E stack.
type Harness struct {
	Core     string
	Identity string
	EspoCRM  string
	Redmine  string
	Outline  string
	NATSAddr string

	client *http.Client
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return strings.TrimRight(value, "/")
	}
	return fallback
}

// New returns a harness, skipping the test when no stack is configured.
func New(t *testing.T) *Harness {
	t.Helper()
	core := env("E2E_BASE_URL", "")
	if core == "" {
		t.Skip("E2E_BASE_URL is not set; start docker-compose.e2e.yml to run the E2E scenarios")
	}
	return &Harness{
		Core:     core,
		Identity: env("E2E_IDENTITY_URL", "http://127.0.0.1:9000"),
		EspoCRM:  env("E2E_ESPOCRM_URL", "http://127.0.0.1:8090"),
		Redmine:  env("E2E_REDMINE_URL", "http://127.0.0.1:8091"),
		Outline:  env("E2E_OUTLINE_URL", "http://127.0.0.1:8092"),
		NATSAddr: env("E2E_NATS_ADDR", "127.0.0.1:4222"),
		// Longer than the adapters' own 10s upstream timeout, so a timeout
		// scenario is observed as a normalized platform error rather than as a
		// client-side cancellation.
		client: &http.Client{Timeout: 45 * time.Second},
	}
}

// Pagination is the pagination block every normalized collection returns.
type Pagination struct {
	Total      int    `json:"total"`
	Limit      int    `json:"limit"`
	NextCursor string `json:"next_cursor"`
}

// Response is a decoded platform response.
type Response struct {
	Status  int
	Headers http.Header
	Body    []byte
}

// JSON decodes the response body into target.
func (r Response) JSON(t *testing.T, target any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, target); err != nil {
		t.Fatalf("decode response body: %v (body: %s)", err, truncate(r.Body))
	}
}

func truncate(body []byte) string {
	const limit = 512
	if len(body) <= limit {
		return string(body)
	}
	return string(body[:limit]) + "..."
}

// Request performs an HTTP call against the stack.
func (h *Harness) Request(t *testing.T, method, url, token string, body any, headers map[string]string) Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode request body: %v", err)
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := h.client.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return Response{Status: response.StatusCode, Headers: response.Header, Body: payload}
}

// API calls the human API of the Integration Core.
func (h *Harness) API(t *testing.T, method, path, token string, body any) Response {
	t.Helper()
	return h.Request(t, method, h.Core+path, token, body, nil)
}

// InjectFault queues a fault on a mock upstream.
func (h *Harness) InjectFault(t *testing.T, mockURL, path string, status, delayMS int) {
	t.Helper()
	fault := map[string]any{"path": path}
	if status != 0 {
		fault["status"] = status
	}
	if delayMS != 0 {
		fault["delay_ms"] = delayMS
	}
	response := h.Request(t, http.MethodPost, mockURL+"/__mock/faults", "", fault, nil)
	if response.Status != http.StatusAccepted {
		t.Fatalf("inject fault on %s: status = %d, body = %s", mockURL, response.Status, truncate(response.Body))
	}
	t.Cleanup(func() { h.ResetFaults(t, mockURL) })
}

// InjectFaultTimes queues a fault that applies to the next count requests.
func (h *Harness) InjectFaultTimes(t *testing.T, mockURL, path string, status, delayMS, count int) {
	t.Helper()
	fault := map[string]any{"path": path, "remaining": count}
	if status != 0 {
		fault["status"] = status
	}
	if delayMS != 0 {
		fault["delay_ms"] = delayMS
	}
	response := h.Request(t, http.MethodPost, mockURL+"/__mock/faults", "", fault, nil)
	if response.Status != http.StatusAccepted {
		t.Fatalf("inject fault on %s: status = %d, body = %s", mockURL, response.Status, truncate(response.Body))
	}
	t.Cleanup(func() { h.ResetFaults(t, mockURL) })
}

// ResetFaults clears every queued fault on a mock upstream.
func (h *Harness) ResetFaults(t *testing.T, mockURL string) {
	t.Helper()
	response := h.Request(t, http.MethodDelete, mockURL+"/__mock/faults", "", nil, nil)
	if response.Status != http.StatusNoContent {
		t.Fatalf("reset faults on %s: status = %d", mockURL, response.Status)
	}
}

// WaitReady blocks until the Integration Core reports readiness, so the
// scenarios never race the stack's own startup and migrations.
func (h *Harness) WaitReady(t *testing.T, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		response, err := h.client.Get(h.Core + "/readyz")
		if err == nil {
			body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<16))
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
			lastErr = fmt.Errorf("readyz returned HTTP %d: %s", response.StatusCode, truncate(body))
		} else {
			lastErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("Integration Core did not become ready within %s: %v", timeout, lastErr)
}
