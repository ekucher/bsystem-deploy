package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ekucher/bsystem-deploy/mocks/internal/fixtures"
	"github.com/ekucher/bsystem-deploy/mocks/internal/mockhttp"
)

const testKey = "test-redmine-api-key"

func newServer(t *testing.T) *mockhttp.Server {
	t.Helper()
	server := mockhttp.NewServer("mock-redmine")
	(&api{apiKey: testKey}).register(server)
	return server
}

func get(t *testing.T, handler http.Handler, target, key string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	if key != "" {
		request.Header.Set("X-Redmine-API-Key", key)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

type issuesBody struct {
	Issues []fixtures.Issue `json:"issues"`
	Total  int              `json:"total_count"`
	Offset int              `json:"offset"`
	Limit  int              `json:"limit"`
}

func TestProjectsCollection(t *testing.T) {
	response := get(t, newServer(t).Handler(), "/projects.json?limit=50", testKey)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	var body struct {
		Projects []fixtures.Project `json:"projects"`
		Total    int                `json:"total_count"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Total != len(fixtures.Projects()) || len(body.Projects) != len(fixtures.Projects()) {
		t.Fatalf("total = %d, projects = %d, want %d", body.Total, len(body.Projects), len(fixtures.Projects()))
	}
	if body.Projects[0].Identifier == "" {
		t.Fatal("projects must carry an identifier, the core stores it as Global ID metadata")
	}
}

func TestProjectPagination(t *testing.T) {
	handler := newServer(t).Handler()
	tests := []struct {
		name     string
		target   string
		wantIDs  []int
		wantOffs int
	}{
		{name: "first page", target: "/projects.json?limit=2&offset=0", wantIDs: []int{101, 102}, wantOffs: 0},
		{name: "second page", target: "/projects.json?limit=2&offset=2", wantIDs: []int{103}, wantOffs: 2},
		{name: "past end", target: "/projects.json?limit=2&offset=50", wantIDs: []int{}, wantOffs: 3},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := get(t, handler, test.target, testKey)
			var body struct {
				Projects []fixtures.Project `json:"projects"`
				Total    int                `json:"total_count"`
				Offset   int                `json:"offset"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Total != len(fixtures.Projects()) {
				t.Fatalf("total_count = %d, want %d", body.Total, len(fixtures.Projects()))
			}
			if body.Offset != test.wantOffs {
				t.Fatalf("offset = %d, want %d", body.Offset, test.wantOffs)
			}
			if len(body.Projects) != len(test.wantIDs) {
				t.Fatalf("page size = %d, want %d", len(body.Projects), len(test.wantIDs))
			}
			for i, want := range test.wantIDs {
				if body.Projects[i].ID != want {
					t.Fatalf("page[%d] = %d, want %d", i, body.Projects[i].ID, want)
				}
			}
		})
	}
}

// The adapter always sends status_id=* and optionally project_id, so both
// filters have to behave exactly as Redmine does.
func TestIssueFilters(t *testing.T) {
	handler := newServer(t).Handler()
	tests := []struct {
		name    string
		target  string
		wantIDs []int
	}{
		{name: "all statuses", target: "/issues.json?limit=100&status_id=*", wantIDs: []int{5001, 5002, 5003, 5004, 5005}},
		{name: "by numeric project", target: "/issues.json?limit=100&status_id=*&project_id=101", wantIDs: []int{5001, 5002}},
		{name: "by project identifier", target: "/issues.json?limit=100&status_id=*&project_id=globex-scada", wantIDs: []int{5003, 5005}},
		{name: "open only", target: "/issues.json?limit=100&status_id=open", wantIDs: []int{5001, 5002, 5004, 5005}},
		{name: "closed only", target: "/issues.json?limit=100&status_id=closed", wantIDs: []int{5003}},
		{name: "exact status id", target: "/issues.json?limit=100&status_id=1", wantIDs: []int{5001, 5004}},
		{name: "unknown project", target: "/issues.json?limit=100&status_id=*&project_id=nope", wantIDs: []int{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := get(t, handler, test.target, testKey)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.Code)
			}
			var body issuesBody
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Total != len(test.wantIDs) {
				t.Fatalf("total_count = %d, want %d", body.Total, len(test.wantIDs))
			}
			if len(body.Issues) != len(test.wantIDs) {
				t.Fatalf("issues = %d, want %d", len(body.Issues), len(test.wantIDs))
			}
			for i, want := range test.wantIDs {
				if body.Issues[i].ID != want {
					t.Fatalf("issues[%d] = %d, want %d", i, body.Issues[i].ID, want)
				}
			}
		})
	}
}

// Filtering has to be applied before paging, otherwise total_count would
// describe the wrong collection.
func TestIssuePaginationAppliesAfterFiltering(t *testing.T) {
	response := get(t, newServer(t).Handler(), "/issues.json?status_id=*&project_id=101&limit=1&offset=1", testKey)
	var body issuesBody
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Total != 2 {
		t.Fatalf("total_count = %d, want 2", body.Total)
	}
	if len(body.Issues) != 1 || body.Issues[0].ID != 5002 {
		t.Fatalf("unexpected page: %+v", body.Issues)
	}
}

func TestAuthorizationIsRequired(t *testing.T) {
	handler := newServer(t).Handler()
	for _, target := range []string{"/users/current.json", "/projects.json", "/issues.json", "/issues/5001.json"} {
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
	if response := get(t, newServer(t).Handler(), "/users/current.json", testKey); response.Code != http.StatusOK {
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
		{name: "known issue", target: "/issues/5001.json", wantStatus: http.StatusOK},
		{name: "unknown issue", target: "/issues/9999.json", wantStatus: http.StatusNotFound},
		{name: "project by id", target: "/projects/101.json", wantStatus: http.StatusOK},
		{name: "project by identifier", target: "/projects/globex-scada.json", wantStatus: http.StatusOK},
		{name: "unknown project", target: "/projects/nope.json", wantStatus: http.StatusNotFound},
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
	for _, status := range []int{http.StatusUnauthorized, http.StatusNotFound, http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := newServer(t)
			if err := server.Faults.Add(mockhttp.Fault{Path: "/projects.json", Status: status}); err != nil {
				t.Fatalf("add fault: %v", err)
			}
			if response := get(t, server.Handler(), "/projects.json", testKey); response.Code != status {
				t.Fatalf("status = %d, want %d", response.Code, status)
			}
		})
	}
}
