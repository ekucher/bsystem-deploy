package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ekucher/bsystem-deploy/mocks/internal/fixtures"
	"github.com/ekucher/bsystem-deploy/mocks/internal/mockhttp"
)

const testKey = "test-espocrm-api-key"

func newServer(t *testing.T) *mockhttp.Server {
	t.Helper()
	server := mockhttp.NewServer("mock-espocrm")
	(&api{apiKey: testKey}).register(server)
	return server
}

func get(t *testing.T, handler http.Handler, target, key string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	if key != "" {
		request.Header.Set("X-Api-Key", key)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestCollectionsMatchFixtures(t *testing.T) {
	handler := newServer(t).Handler()
	tests := []struct {
		name      string
		target    string
		wantTotal int
		wantFirst string
	}{
		{name: "accounts", target: "/api/v1/Account?maxSize=50&orderBy=name&order=asc", wantTotal: len(fixtures.Accounts()), wantFirst: fixtures.Accounts()[0].ID},
		{name: "contacts", target: "/api/v1/Contact?maxSize=50&orderBy=name&order=asc", wantTotal: len(fixtures.Contacts()), wantFirst: fixtures.Contacts()[0].ID},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := get(t, handler, test.target, testKey)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.Code)
			}
			var body struct {
				Total int `json:"total"`
				List  []struct {
					ID string `json:"id"`
				} `json:"list"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Total != test.wantTotal || len(body.List) != test.wantTotal {
				t.Fatalf("total = %d, list = %d, want %d", body.Total, len(body.List), test.wantTotal)
			}
			if body.List[0].ID != test.wantFirst {
				t.Fatalf("first id = %q, want %q", body.List[0].ID, test.wantFirst)
			}
		})
	}
}

// The total must stay the collection size while the list shrinks to the page,
// because that is the contract the adapter's pagination will rely on.
func TestPaginationKeepsTotalStable(t *testing.T) {
	handler := newServer(t).Handler()
	tests := []struct {
		name     string
		target   string
		wantIDs  []string
		wantSize int
	}{
		{name: "first page", target: "/api/v1/Account?maxSize=2&offset=0", wantIDs: []string{"acc-globex", "acc-initech"}},
		{name: "second page", target: "/api/v1/Account?maxSize=2&offset=2", wantIDs: []string{"acc-northwind"}},
		{name: "offset past end", target: "/api/v1/Account?maxSize=2&offset=99", wantIDs: []string{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := get(t, handler, test.target, testKey)
			var body struct {
				Total int                `json:"total"`
				List  []fixtures.Account `json:"list"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Total != len(fixtures.Accounts()) {
				t.Fatalf("total = %d, want %d", body.Total, len(fixtures.Accounts()))
			}
			if len(body.List) != len(test.wantIDs) {
				t.Fatalf("page size = %d, want %d", len(body.List), len(test.wantIDs))
			}
			for i, want := range test.wantIDs {
				if body.List[i].ID != want {
					t.Fatalf("page[%d] = %q, want %q", i, body.List[i].ID, want)
				}
			}
		})
	}
}

func TestAuthorizationIsRequired(t *testing.T) {
	handler := newServer(t).Handler()
	for _, target := range []string{"/api/v1/App/user", "/api/v1/Account", "/api/v1/Contact", "/api/v1/Account/acc-globex"} {
		t.Run(target, func(t *testing.T) {
			if response := get(t, handler, target, ""); response.Code != http.StatusUnauthorized {
				t.Fatalf("missing key status = %d, want 401", response.Code)
			}
			if response := get(t, handler, target, "wrong-key"); response.Code != http.StatusUnauthorized {
				t.Fatalf("wrong key status = %d, want 401", response.Code)
			}
		})
	}
}

func TestHealthProbeEndpoint(t *testing.T) {
	if response := get(t, newServer(t).Handler(), "/api/v1/App/user", testKey); response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
}

func TestDetailLookups(t *testing.T) {
	handler := newServer(t).Handler()
	tests := []struct {
		name       string
		target     string
		wantStatus int
	}{
		{name: "known account", target: "/api/v1/Account/acc-globex", wantStatus: http.StatusOK},
		{name: "unknown account", target: "/api/v1/Account/acc-missing", wantStatus: http.StatusNotFound},
		{name: "known contact", target: "/api/v1/Contact/ct-anna", wantStatus: http.StatusOK},
		{name: "unknown contact", target: "/api/v1/Contact/ct-missing", wantStatus: http.StatusNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if response := get(t, handler, test.target, testKey); response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}

func TestUpstreamFaultScenarios(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := newServer(t)
			if err := server.Faults.Add(mockhttp.Fault{Path: "/api/v1/Account", Status: status}); err != nil {
				t.Fatalf("add fault: %v", err)
			}
			if response := get(t, server.Handler(), "/api/v1/Account", testKey); response.Code != status {
				t.Fatalf("status = %d, want %d", response.Code, status)
			}
		})
	}
}
