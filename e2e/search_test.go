package e2e

import (
	"fmt"
	"net/http"
	"testing"
)

type searchHit struct {
	ID                 string   `json:"id"`
	Type               string   `json:"type"`
	Title              string   `json:"title"`
	Summary            string   `json:"summary"`
	Source             string   `json:"source"`
	TenantID           string   `json:"tenant_id"`
	Permissions        []string `json:"permissions"`
	ScopeType          string   `json:"scope_type"`
	ScopeID            string   `json:"scope_id"`
	Score              float64  `json:"score"`
	AudiencePermission string   `json:"-"`
}

type searchPage struct {
	Data       []searchHit `json:"data"`
	Pagination struct {
		Limit      int    `json:"limit"`
		NextCursor string `json:"next_cursor"`
	} `json:"pagination"`
}

// indexDocuments puts documents into the platform's index as the service
// identity, and removes them when the test finishes so scenarios do not leak
// into one another.
func indexDocuments(t *testing.T, h *Harness, documents ...map[string]any) {
	t.Helper()
	response := h.Request(t, http.MethodPost, h.Core+"/api/service/v1/search/documents", TokenService,
		map[string]any{"documents": documents}, nil)
	if response.Status != http.StatusAccepted {
		t.Fatalf("index documents: status = %d, body = %s", response.Status, truncate(response.Body))
	}
	t.Cleanup(func() {
		for _, document := range documents {
			id, _ := document["id"].(string)
			h.Request(t, http.MethodDelete, h.Core+"/api/service/v1/search/documents/"+id, TokenService, nil, nil)
		}
	})
}

func searchAs(t *testing.T, h *Harness, token, query string) searchPage {
	t.Helper()
	response := h.API(t, http.MethodGet, "/api/v1/search"+query, token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("search%s: status = %d, body = %s", query, response.Status, truncate(response.Body))
	}
	var page searchPage
	response.JSON(t, &page)
	return page
}

func hitIDs(page searchPage) []string {
	ids := make([]string, 0, len(page.Data))
	for _, hit := range page.Data {
		ids = append(ids, hit.ID)
	}
	return ids
}

// The boundary the whole design exists for: a search returns records the
// caller never named, so every result must be authorized individually.
func TestSearchReturnsOnlyWhatTheCallerMayRead(t *testing.T) {
	h := ready(t)

	term := marker("searchable")
	indexDocuments(t, h,
		map[string]any{
			"id": "CL-000001", "type": "client", "title": "Northwind " + term,
			"source": "espocrm", "permissions": []string{"crm.client.read"},
			"scope_type": "client", "scope_id": "CL-000001",
		},
		map[string]any{
			"id": "PR-000001", "type": "project", "title": "Migration " + term,
			"source": "redmine", "permissions": []string{"projects.task.read"},
			"scope_type": "project", "scope_id": "PR-000001",
		},
		map[string]any{
			"id": "DOC-000001", "type": "document", "title": "Runbook " + term,
			"source": "outline", "permissions": []string{"wiki.document.read"},
			"scope_type": "resource", "scope_id": "DOC-000001",
		},
	)

	cases := []struct {
		name    string
		token   string
		want    []string
		absent  []string
		anyHits bool
	}{
		{
			name: "an administrator sees every document", token: TokenAdmin,
			want: []string{"CL-000001", "PR-000001", "DOC-000001"}, anyHits: true,
		},
		{
			// QA holds projects.task.read and wiki.document.read but not
			// crm.client.read.
			name: "QA sees only what its role covers", token: TokenQA,
			want: []string{"PR-000001", "DOC-000001"}, absent: []string{"CL-000001"}, anyHits: true,
		},
		{
			name: "DevOps holds none of these permissions", token: TokenDevOps,
			absent: []string{"CL-000001", "PR-000001"},
		},
		{
			// The case that matters. A customer's crm.client.read means "my
			// own record", not "clients". Without per-document evaluation a
			// search would hand them every client in the platform.
			name: "a scope-confined customer sees nothing it was not granted", token: TokenCustomer,
			absent: []string{"CL-000001", "PR-000001", "DOC-000001"},
		},
		{
			name: "a principal whose groups mapped to nothing sees nothing", token: TokenNoGroups,
			absent: []string{"CL-000001", "PR-000001", "DOC-000001"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			page := searchAs(t, h, c.token, "?q="+term+"&limit=50")
			ids := hitIDs(page)
			for _, wanted := range c.want {
				if !contains(ids, wanted) {
					t.Errorf("%s is missing from %v", wanted, ids)
				}
			}
			for _, forbidden := range c.absent {
				if contains(ids, forbidden) {
					t.Errorf("%s was disclosed to a caller who may not read it: %v", forbidden, ids)
				}
			}
			if c.anyHits && len(ids) == 0 {
				t.Error("no results at all; the scenario is not exercising anything")
			}
		})
	}
}

// The envelope must not carry a total. The number of matches before
// authorization counts records the caller may not be allowed to know exist.
func TestSearchNeverDisclosesTheUnfilteredMatchCount(t *testing.T) {
	h := ready(t)

	term := marker("counted")
	indexDocuments(t, h,
		map[string]any{
			"id": "CL-000001", "type": "client", "title": "Northwind " + term,
			"source": "espocrm", "permissions": []string{"crm.client.read"},
			"scope_type": "client", "scope_id": "CL-000001",
		},
		map[string]any{
			"id": "CL-000002", "type": "client", "title": "Globex " + term,
			"source": "espocrm", "permissions": []string{"crm.client.read"},
			"scope_type": "client", "scope_id": "CL-000002",
		},
	)

	response := h.API(t, http.MethodGet, "/api/v1/search?q="+term, TokenCustomer, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("search: status = %d", response.Status)
	}
	var raw map[string]any
	response.JSON(t, &raw)
	pagination, _ := raw["pagination"].(map[string]any)
	if _, present := pagination["total"]; present {
		t.Errorf("the search envelope carries a total: %s", truncate(response.Body))
	}
	// And the customer must not have seen the clients themselves either.
	var page searchPage
	response.JSON(t, &page)
	if len(page.Data) != 0 {
		t.Errorf("a scope-confined caller received %v", hitIDs(page))
	}
}

func TestSearchFiltersByType(t *testing.T) {
	h := ready(t)

	term := marker("typed")
	indexDocuments(t, h,
		map[string]any{
			"id": "PR-000001", "type": "project", "title": "Alpha " + term,
			"source": "redmine", "permissions": []string{"projects.task.read"},
			"scope_type": "project", "scope_id": "PR-000001",
		},
		map[string]any{
			"id": "DOC-000001", "type": "document", "title": "Alpha " + term,
			"source": "outline", "permissions": []string{"wiki.document.read"},
			"scope_type": "resource", "scope_id": "DOC-000001",
		},
	)

	page := searchAs(t, h, TokenAdmin, "?q="+term+"&type=document")
	ids := hitIDs(page)
	if !contains(ids, "DOC-000001") || contains(ids, "PR-000001") {
		t.Errorf("type filter gave %v, want only the document", ids)
	}

	// An unknown type is refused rather than ignored: dropping it would
	// answer a broader question than the caller asked, and they would read
	// the result as though it had been filtered.
	response := h.API(t, http.MethodGet, "/api/v1/search?q="+term+"&type=server", TokenAdmin, nil)
	if response.Status != http.StatusBadRequest {
		t.Errorf("unknown type: status = %d, want 400", response.Status)
	}
	assertNoSecretsOrTopology(t, response.Body)
}

// A one-character query would match most of the index and read as a listing
// rather than a search.
func TestShortQueriesAndForgedCursorsAreRefused(t *testing.T) {
	h := ready(t)

	for _, query := range []string{"?q=", "?q=a", "?q=%20%20"} {
		response := h.API(t, http.MethodGet, "/api/v1/search"+query, TokenAdmin, nil)
		if response.Status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", query, response.Status)
		}
	}
	for _, cursor := range []string{"not-base64!", "bjoxMA", "0", "abc"} {
		response := h.API(t, http.MethodGet, "/api/v1/search?q=northwind&cursor="+cursor, TokenAdmin, nil)
		if response.Status != http.StatusBadRequest {
			t.Errorf("cursor %q: status = %d, want 400", cursor, response.Status)
		}
	}
}

// Indexing is a machine capability. No human token may reach it, whatever its
// role: a person searching does not index.
func TestOnlyServiceIdentitiesMayIndex(t *testing.T) {
	h := ready(t)

	document := map[string]any{
		"id": "CL-000001", "type": "client", "title": "Northwind",
		"source": "espocrm", "permissions": []string{"crm.client.read"},
		"scope_type": "client", "scope_id": "CL-000001",
	}
	for name, token := range map[string]string{"an administrator": TokenAdmin, "no token": ""} {
		t.Run(name, func(t *testing.T) {
			response := h.Request(t, http.MethodPost, h.Core+"/api/service/v1/search/documents", token,
				map[string]any{"documents": []any{document}}, nil)
			if response.Status != http.StatusUnauthorized && response.Status != http.StatusForbidden {
				t.Errorf("index: status = %d, want 401 or 403", response.Status)
			}
			response = h.Request(t, http.MethodDelete, h.Core+"/api/service/v1/search/documents/CL-000001", token, nil, nil)
			if response.Status != http.StatusUnauthorized && response.Status != http.StatusForbidden {
				t.Errorf("delete: status = %d, want 401 or 403", response.Status)
			}
		})
	}
}

// A document nobody may read would be indexed and never returned. The
// publisher should learn now rather than when somebody reports missing results.
func TestDocumentsWithoutAnAudienceAreRefused(t *testing.T) {
	h := ready(t)

	cases := map[string]map[string]any{
		"no permissions": {"id": "CL-000001", "type": "client", "title": "x", "source": "s", "scope_type": "client", "scope_id": "CL-000001"},
		"no scope":       {"id": "CL-000001", "type": "client", "title": "x", "source": "s", "permissions": []string{"crm.client.read"}},
		"unknown type":   {"id": "SRV-000001", "type": "server", "title": "x", "source": "s", "permissions": []string{"operations.server.read"}, "scope_type": "resource", "scope_id": "SRV-000001"},
		"no title":       {"id": "CL-000001", "type": "client", "source": "s", "permissions": []string{"crm.client.read"}, "scope_type": "client", "scope_id": "CL-000001"},
	}
	for name, document := range cases {
		t.Run(name, func(t *testing.T) {
			response := h.Request(t, http.MethodPost, h.Core+"/api/service/v1/search/documents", TokenService,
				map[string]any{"documents": []any{document}}, nil)
			if response.Status != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body: %s)", response.Status, truncate(response.Body))
			}
			assertNoSecretsOrTopology(t, response.Body)
		})
	}
}

// Indexing is an upsert and deleting is idempotent, so an indexer that
// replays its work converges instead of duplicating or failing.
func TestIndexingConvergesOnReplay(t *testing.T) {
	h := ready(t)

	term := marker("replayed")
	document := map[string]any{
		"id": "CL-000003", "type": "client", "title": "Original " + term,
		"source": "espocrm", "permissions": []string{"crm.client.read"},
		"scope_type": "client", "scope_id": "CL-000003",
	}
	indexDocuments(t, h, document)

	updated := map[string]any{}
	for key, value := range document {
		updated[key] = value
	}
	updated["title"] = "Updated " + term
	indexDocuments(t, h, updated)

	page := searchAs(t, h, TokenAdmin, "?q="+term+"&limit=50")
	if len(page.Data) != 1 {
		t.Fatalf("hits = %v, want the document replaced rather than duplicated", hitIDs(page))
	}
	if page.Data[0].Title != "Updated "+term {
		t.Errorf("title = %q, want the re-indexed value", page.Data[0].Title)
	}

	// Deleting twice, and deleting something never indexed, both succeed: a
	// delete that failed once the record was gone would turn a retry into an
	// error.
	for _, id := range []string{"CL-000003", "CL-000003", "CL-999999"} {
		response := h.Request(t, http.MethodDelete, h.Core+"/api/service/v1/search/documents/"+id, TokenService, nil, nil)
		if response.Status != http.StatusNoContent {
			t.Errorf("delete %s: status = %d, want 204", id, response.Status)
		}
	}
	if page := searchAs(t, h, TokenAdmin, "?q="+term+"&limit=50"); len(page.Data) != 0 {
		t.Errorf("the deleted document is still searchable: %v", hitIDs(page))
	}
}

// Following the cursor must reach every result the caller may see, exactly
// once, and terminate.
func TestSearchPaginationWalksWithoutRepeatingOrLosing(t *testing.T) {
	h := ready(t)

	term := marker("paged")
	documents := make([]map[string]any, 0, 5)
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("PR-00001%d", i)
		documents = append(documents, map[string]any{
			"id": id, "type": "project", "title": fmt.Sprintf("Release %d %s", i, term),
			"source": "redmine", "permissions": []string{"projects.task.read"},
			"scope_type": "project", "scope_id": id,
		})
	}
	indexDocuments(t, h, documents...)

	seen := map[string]bool{}
	query := "?q=" + term + "&limit=2"
	for page := 0; page < 20; page++ {
		result := searchAs(t, h, TokenAdmin, query)
		if result.Pagination.Limit > 2 {
			t.Fatalf("limit=2 returned %d results", result.Pagination.Limit)
		}
		for _, hit := range result.Data {
			if seen[hit.ID] {
				t.Fatalf("%s was returned twice while walking", hit.ID)
			}
			seen[hit.ID] = true
		}
		if result.Pagination.NextCursor == "" {
			if len(seen) != len(documents) {
				t.Errorf("walked %d of %d documents", len(seen), len(documents))
			}
			return
		}
		query = "?q=" + term + "&limit=2&cursor=" + result.Pagination.NextCursor
	}
	t.Fatal("the pagination walk did not terminate")
}
