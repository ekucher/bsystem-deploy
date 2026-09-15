// Command mock-espocrm serves a deterministic stand-in for the EspoCRM REST
// API surface consumed by the BSYSTEM CRM adapter.
//
// The API key it accepts is a documented test-only placeholder.
package main

import (
	"log"
	"net/http"

	"github.com/ekucher/bsystem-deploy/mocks/internal/fixtures"
	"github.com/ekucher/bsystem-deploy/mocks/internal/mockhttp"
)

const (
	defaultMaxSize = 50
	maxMaxSize     = 200
)

type api struct{ apiKey string }

// listResponse matches the EspoCRM collection envelope: a total across the
// whole collection plus the requested page.
type listResponse[T any] struct {
	Total int `json:"total"`
	List  []T `json:"list"`
}

func (a *api) authorized(w http.ResponseWriter, r *http.Request) bool {
	return mockhttp.RequireHeader(w, r, "X-Api-Key", a.apiKey)
}

// appUser backs the adapter health probe.
func (a *api) appUser(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(w, r) {
		return
	}
	mockhttp.WriteJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{"id": "mock-api-user", "userName": "mock-api-user", "type": "api"},
	})
}

func (a *api) accounts(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(w, r) {
		return
	}
	all := fixtures.Accounts()
	start, end := mockhttp.Page(len(all), mockhttp.QueryInt(r, "offset", 0), mockhttp.QueryInt(r, "maxSize", defaultMaxSize), defaultMaxSize, maxMaxSize)
	mockhttp.WriteJSON(w, http.StatusOK, listResponse[fixtures.Account]{Total: len(all), List: all[start:end]})
}

func (a *api) contacts(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(w, r) {
		return
	}
	all := fixtures.Contacts()
	start, end := mockhttp.Page(len(all), mockhttp.QueryInt(r, "offset", 0), mockhttp.QueryInt(r, "maxSize", defaultMaxSize), defaultMaxSize, maxMaxSize)
	mockhttp.WriteJSON(w, http.StatusOK, listResponse[fixtures.Contact]{Total: len(all), List: all[start:end]})
}

// entityByID answers detail reads for the entities BSYSTEM maps to Global IDs.
func (a *api) accountByID(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(w, r) {
		return
	}
	id := r.PathValue("id")
	for _, account := range fixtures.Accounts() {
		if account.ID == id {
			mockhttp.WriteJSON(w, http.StatusOK, account)
			return
		}
	}
	mockhttp.WriteError(w, http.StatusNotFound, "Account not found")
}

func (a *api) contactByID(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(w, r) {
		return
	}
	id := r.PathValue("id")
	for _, contact := range fixtures.Contacts() {
		if contact.ID == id {
			mockhttp.WriteJSON(w, http.StatusOK, contact)
			return
		}
	}
	mockhttp.WriteError(w, http.StatusNotFound, "Contact not found")
}

func (a *api) register(server *mockhttp.Server) {
	server.Handle("GET /api/v1/App/user", a.appUser)
	server.Handle("GET /api/v1/Account", a.accounts)
	server.Handle("GET /api/v1/Account/{id}", a.accountByID)
	server.Handle("GET /api/v1/Contact", a.contacts)
	server.Handle("GET /api/v1/Contact/{id}", a.contactByID)
}

func main() {
	mockhttp.MaybeHealthCheck(":8090")
	server := mockhttp.NewServer("mock-espocrm")
	(&api{apiKey: mockhttp.Secret("ESPOCRM_API_KEY", "test-espocrm-api-key")}).register(server)
	log.Fatal(server.ListenAndServe("HTTP_ADDR", ":8090"))
}
