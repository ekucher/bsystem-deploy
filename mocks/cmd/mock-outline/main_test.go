package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ekucher/bsystem-deploy/mocks/internal/fixtures"
	"github.com/ekucher/bsystem-deploy/mocks/internal/mockhttp"
)

const testKey = "test-outline-api-key"

func newServer(t *testing.T) *mockhttp.Server {
	t.Helper()
	server := mockhttp.NewServer("mock-outline")
	(&api{apiKey: testKey}).register(server)
	return server
}

func rpc(t *testing.T, handler http.Handler, method, body, key string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/"+method, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if key != "" {
		request.Header.Set("Authorization", "Bearer "+key)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestDocumentsList(t *testing.T) {
	response := rpc(t, newServer(t).Handler(), "documents.list", `{"limit":50}`, testKey)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	var body struct {
		Data       []fixtures.Document `json:"data"`
		Pagination struct {
			Total int `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data) != len(fixtures.Documents()) || body.Pagination.Total != len(fixtures.Documents()) {
		t.Fatalf("documents = %d, total = %d, want %d", len(body.Data), body.Pagination.Total, len(fixtures.Documents()))
	}
	if body.Data[0].CollectionID == "" || body.Data[0].UpdatedAt == "" {
		t.Fatalf("documents must carry collectionId and updatedAt: %+v", body.Data[0])
	}
}

func TestDocumentsListPagination(t *testing.T) {
	handler := newServer(t).Handler()
	tests := []struct {
		name    string
		body    string
		wantIDs []string
	}{
		{name: "first page", body: `{"limit":2,"offset":0}`, wantIDs: []string{"doc-onboarding", "doc-runbook"}},
		{name: "second page", body: `{"limit":2,"offset":2}`, wantIDs: []string{"doc-northwind-sla", "doc-globex-notes"}},
		{name: "past end", body: `{"limit":2,"offset":99}`, wantIDs: []string{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := rpc(t, handler, "documents.list", test.body, testKey)
			var body struct {
				Data       []fixtures.Document `json:"data"`
				Pagination struct {
					Total int `json:"total"`
				} `json:"pagination"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Pagination.Total != len(fixtures.Documents()) {
				t.Fatalf("total = %d, want %d", body.Pagination.Total, len(fixtures.Documents()))
			}
			if len(body.Data) != len(test.wantIDs) {
				t.Fatalf("page size = %d, want %d", len(body.Data), len(test.wantIDs))
			}
			for i, want := range test.wantIDs {
				if body.Data[i].ID != want {
					t.Fatalf("page[%d] = %q, want %q", i, body.Data[i].ID, want)
				}
			}
		})
	}
}

func TestDocumentsInfo(t *testing.T) {
	handler := newServer(t).Handler()
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "known document", body: `{"id":"doc-runbook"}`, wantStatus: http.StatusOK},
		{name: "unknown document", body: `{"id":"doc-missing"}`, wantStatus: http.StatusNotFound},
		{name: "missing id", body: `{}`, wantStatus: http.StatusBadRequest},
		{name: "malformed body", body: `{`, wantStatus: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := rpc(t, handler, "documents.info", test.body, testKey)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}

func TestDocumentsSearch(t *testing.T) {
	handler := newServer(t).Handler()
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantHits   int
	}{
		{name: "title match", body: `{"query":"runbook"}`, wantStatus: http.StatusOK, wantHits: 1},
		{name: "case insensitive", body: `{"query":"NORTHWIND"}`, wantStatus: http.StatusOK, wantHits: 1},
		{name: "body match", body: `{"query":"migration notes"}`, wantStatus: http.StatusOK, wantHits: 1},
		{name: "no match", body: `{"query":"zzzz"}`, wantStatus: http.StatusOK, wantHits: 0},
		{name: "missing query", body: `{}`, wantStatus: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := rpc(t, handler, "documents.search", test.body, testKey)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if test.wantStatus != http.StatusOK {
				return
			}
			var body struct {
				Data []struct {
					Document fixtures.Document `json:"document"`
					Context  string            `json:"context"`
				} `json:"data"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(body.Data) != test.wantHits {
				t.Fatalf("hits = %d, want %d", len(body.Data), test.wantHits)
			}
		})
	}
}

func TestAuthorizationIsRequired(t *testing.T) {
	handler := newServer(t).Handler()
	for _, method := range []string{"documents.list", "documents.info", "documents.search"} {
		t.Run(method, func(t *testing.T) {
			if response := rpc(t, handler, method, `{}`, ""); response.Code != http.StatusUnauthorized {
				t.Fatalf("missing key status = %d, want 401", response.Code)
			}
			if response := rpc(t, handler, method, `{}`, "wrong-key"); response.Code != http.StatusUnauthorized {
				t.Fatalf("wrong key status = %d, want 401", response.Code)
			}
		})
	}
}

func TestUpstreamFaultScenarios(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := newServer(t)
			if err := server.Faults.Add(mockhttp.Fault{Path: "/api/documents.list", Status: status}); err != nil {
				t.Fatalf("add fault: %v", err)
			}
			if response := rpc(t, server.Handler(), "documents.list", `{"limit":1}`, testKey); response.Code != status {
				t.Fatalf("status = %d, want %d", response.Code, status)
			}
		})
	}
}
