// Command mock-redmine serves a deterministic stand-in for the Redmine REST
// API surface consumed by the BSYSTEM project adapter.
//
// The API key it accepts is a documented test-only placeholder.
package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/ekucher/bsystem-deploy/mocks/internal/fixtures"
	"github.com/ekucher/bsystem-deploy/mocks/internal/mockhttp"
)

const (
	defaultLimit = 25
	maxLimit     = 100
)

type api struct{ apiKey string }

func (a *api) authorized(w http.ResponseWriter, r *http.Request) bool {
	return mockhttp.RequireHeader(w, r, "X-Redmine-API-Key", a.apiKey)
}

// currentUser backs the adapter health probe.
func (a *api) currentUser(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(w, r) {
		return
	}
	mockhttp.WriteJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{"id": 1, "login": "mock-api-user", "firstname": "Mock", "lastname": "Api"},
	})
}

func (a *api) projects(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(w, r) {
		return
	}
	all := fixtures.Projects()
	offset := mockhttp.QueryInt(r, "offset", 0)
	start, end := mockhttp.Page(len(all), offset, mockhttp.QueryInt(r, "limit", defaultLimit), defaultLimit, maxLimit)
	page := all[start:end]
	mockhttp.WriteJSON(w, http.StatusOK, map[string]any{
		"projects":    page,
		"total_count": len(all),
		"offset":      start,
		"limit":       len(page),
	})
}

// matchesProject accepts either a numeric project id or a project identifier,
// mirroring Redmine's own project_id filter.
func matchesProject(issue fixtures.Issue, filter string) bool {
	if filter == "" {
		return true
	}
	if strconv.Itoa(issue.Project.ID) == filter {
		return true
	}
	for _, project := range fixtures.Projects() {
		if project.ID == issue.Project.ID && project.Identifier == filter {
			return true
		}
	}
	return false
}

// matchesStatus implements the subset of Redmine's status_id filter the
// adapter uses: "*" for every status, "open"/"closed" buckets, or an exact id.
func matchesStatus(issue fixtures.Issue, filter string) bool {
	switch filter {
	case "", "*":
		return true
	case "open":
		return issue.Status.Name != "Closed" && issue.Status.Name != "Rejected"
	case "closed":
		return issue.Status.Name == "Closed" || issue.Status.Name == "Rejected"
	default:
		return strconv.Itoa(issue.Status.ID) == filter
	}
}

func (a *api) issues(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(w, r) {
		return
	}
	project := strings.TrimSpace(r.URL.Query().Get("project_id"))
	status := strings.TrimSpace(r.URL.Query().Get("status_id"))
	filtered := []fixtures.Issue{}
	for _, issue := range fixtures.Issues() {
		if matchesProject(issue, project) && matchesStatus(issue, status) {
			filtered = append(filtered, issue)
		}
	}
	start, end := mockhttp.Page(len(filtered), mockhttp.QueryInt(r, "offset", 0), mockhttp.QueryInt(r, "limit", defaultLimit), defaultLimit, maxLimit)
	page := filtered[start:end]
	mockhttp.WriteJSON(w, http.StatusOK, map[string]any{
		"issues":      page,
		"total_count": len(filtered),
		"offset":      start,
		"limit":       len(page),
	})
}

func (a *api) issueByID(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(w, r) {
		return
	}
	id := strings.TrimSuffix(r.PathValue("id"), ".json")
	for _, issue := range fixtures.Issues() {
		if strconv.Itoa(issue.ID) == id {
			mockhttp.WriteJSON(w, http.StatusOK, map[string]any{"issue": issue})
			return
		}
	}
	mockhttp.WriteError(w, http.StatusNotFound, "issue not found")
}

func (a *api) projectByID(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(w, r) {
		return
	}
	id := strings.TrimSuffix(r.PathValue("id"), ".json")
	for _, project := range fixtures.Projects() {
		if strconv.Itoa(project.ID) == id || project.Identifier == id {
			mockhttp.WriteJSON(w, http.StatusOK, map[string]any{"project": project})
			return
		}
	}
	mockhttp.WriteError(w, http.StatusNotFound, "project not found")
}

func (a *api) register(server *mockhttp.Server) {
	server.Handle("GET /users/current.json", a.currentUser)
	server.Handle("GET /projects.json", a.projects)
	server.Handle("GET /projects/{id}", a.projectByID)
	server.Handle("GET /issues.json", a.issues)
	server.Handle("GET /issues/{id}", a.issueByID)
}

func main() {
	mockhttp.MaybeHealthCheck(":8091")
	server := mockhttp.NewServer("mock-redmine")
	(&api{apiKey: mockhttp.Secret("REDMINE_API_KEY", "test-redmine-api-key")}).register(server)
	log.Fatal(server.ListenAndServe("HTTP_ADDR", ":8091"))
}
