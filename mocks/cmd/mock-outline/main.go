// Command mock-outline serves a deterministic stand-in for the Outline RPC
// API surface consumed by the BSYSTEM document adapter.
//
// The API key it accepts is a documented test-only placeholder.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/ekucher/bsystem-deploy/mocks/internal/fixtures"
	"github.com/ekucher/bsystem-deploy/mocks/internal/mockhttp"
)

const (
	defaultLimit = 25
	maxLimit     = 100
)

type api struct{ apiKey string }

// rpcRequest is the union of the request fields the adapter sends. Outline
// takes every parameter in the POST body rather than the query string.
type rpcRequest struct {
	ID     string `json:"id"`
	Query  string `json:"query"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

func decode(w http.ResponseWriter, r *http.Request) (rpcRequest, bool) {
	var request rpcRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16))
	if err := decoder.Decode(&request); err != nil {
		mockhttp.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return rpcRequest{}, false
	}
	return request, true
}

func (a *api) authorized(w http.ResponseWriter, r *http.Request) bool {
	return mockhttp.RequireBearer(w, r, a.apiKey)
}

func (a *api) documentsList(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(w, r) {
		return
	}
	request, ok := decode(w, r)
	if !ok {
		return
	}
	all := fixtures.Documents()
	start, end := mockhttp.Page(len(all), request.Offset, request.Limit, defaultLimit, maxLimit)
	page := all[start:end]
	mockhttp.WriteJSON(w, http.StatusOK, map[string]any{
		"data":       page,
		"pagination": map[string]any{"offset": start, "limit": len(page), "total": len(all)},
	})
}

func (a *api) documentsInfo(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(w, r) {
		return
	}
	request, ok := decode(w, r)
	if !ok {
		return
	}
	id := strings.TrimSpace(request.ID)
	if id == "" {
		mockhttp.WriteError(w, http.StatusBadRequest, "id is required")
		return
	}
	for _, document := range fixtures.Documents() {
		if document.ID == id {
			mockhttp.WriteJSON(w, http.StatusOK, map[string]any{"data": document})
			return
		}
	}
	mockhttp.WriteError(w, http.StatusNotFound, "document not found")
}

// documentsSearch returns Outline's search envelope, where each hit wraps the
// document alongside a context snippet.
func (a *api) documentsSearch(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(w, r) {
		return
	}
	request, ok := decode(w, r)
	if !ok {
		return
	}
	query := strings.ToLower(strings.TrimSpace(request.Query))
	if query == "" {
		mockhttp.WriteError(w, http.StatusBadRequest, "query is required")
		return
	}
	hits := []map[string]any{}
	for _, document := range fixtures.Documents() {
		if strings.Contains(strings.ToLower(document.Title), query) || strings.Contains(strings.ToLower(document.Text), query) {
			hits = append(hits, map[string]any{"context": document.Text, "ranking": 1.0, "document": document})
		}
	}
	start, end := mockhttp.Page(len(hits), request.Offset, request.Limit, defaultLimit, maxLimit)
	page := hits[start:end]
	mockhttp.WriteJSON(w, http.StatusOK, map[string]any{
		"data":       page,
		"pagination": map[string]any{"offset": start, "limit": len(page), "total": len(hits)},
	})
}

func (a *api) register(server *mockhttp.Server) {
	server.Handle("POST /api/documents.list", a.documentsList)
	server.Handle("POST /api/documents.info", a.documentsInfo)
	server.Handle("POST /api/documents.search", a.documentsSearch)
}

func main() {
	mockhttp.MaybeHealthCheck(":8092")
	server := mockhttp.NewServer("mock-outline")
	(&api{apiKey: mockhttp.Secret("OUTLINE_API_KEY", "test-outline-api-key")}).register(server)
	log.Fatal(server.ListenAndServe("HTTP_ADDR", ":8092"))
}
