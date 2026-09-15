package e2e

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// collect reads a normalized collection and returns it.
func collect(t *testing.T, harness *Harness, path, token string) []entityView {
	t.Helper()
	response := harness.API(t, http.MethodGet, path, token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET %s: status = %d, want 200 (body: %s)", path, response.Status, truncate(response.Body))
	}
	var body collectionResponse
	response.JSON(t, &body)
	if len(body.Data) == 0 {
		t.Fatalf("GET %s returned nothing to read in detail", path)
	}
	return body.Data
}

// --- Detail endpoints -------------------------------------------------------

// A detail read must return the same entity the collection described, under
// the same Global ID, without re-deriving it.
func TestDetailReadsMatchTheCollection(t *testing.T) {
	harness := ready(t)
	tests := []struct {
		name       string
		collection string
		detail     string
		field      func(entityView) string
	}{
		{name: "clients", collection: "/api/v1/clients", detail: "/api/v1/clients/", field: func(e entityView) string { return e.Name }},
		{name: "contacts", collection: "/api/v1/contacts", detail: "/api/v1/contacts/", field: func(e entityView) string { return e.Name }},
		{name: "projects", collection: "/api/v1/projects", detail: "/api/v1/projects/", field: func(e entityView) string { return e.Name }},
		{name: "issues", collection: "/api/v1/issues", detail: "/api/v1/issues/", field: func(e entityView) string { return e.Subject }},
		{name: "documents", collection: "/api/v1/documents", detail: "/api/v1/documents/", field: func(e entityView) string { return e.Title }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := collect(t, harness, test.collection, TokenAdmin)
			for _, item := range items {
				response := harness.API(t, http.MethodGet, test.detail+item.ID, TokenAdmin, nil)
				if response.Status != http.StatusOK {
					t.Fatalf("GET %s%s: status = %d, want 200 (body: %s)", test.detail, item.ID, response.Status, truncate(response.Body))
				}
				var detail entityView
				response.JSON(t, &detail)
				if detail.ID != item.ID {
					t.Errorf("detail id = %q, want %q", detail.ID, item.ID)
				}
				if detail.Source != item.Source || detail.SourceID != item.SourceID {
					t.Errorf("detail source = %s/%s, want %s/%s", detail.Source, detail.SourceID, item.Source, item.SourceID)
				}
				if test.field(detail) != test.field(item) {
					t.Errorf("detail field = %q, want %q", test.field(detail), test.field(item))
				}
			}
		})
	}
}

// Role permissions must apply to a single record exactly as they apply to the
// collection: an endpoint that is stricter or looser in detail form is a hole.
func TestDetailAuthorizationMatchesTheCollection(t *testing.T) {
	harness := ready(t)
	client := collect(t, harness, "/api/v1/clients", TokenAdmin)[0]
	project := collect(t, harness, "/api/v1/projects", TokenAdmin)[0]
	document := collect(t, harness, "/api/v1/documents", TokenAdmin)[0]

	tests := []struct {
		name       string
		token      string
		path       string
		wantStatus int
	}{
		{name: "support reads a client", token: TokenSupport, path: "/api/v1/clients/" + client.ID, wantStatus: http.StatusOK},
		{name: "developer is denied a client", token: TokenDeveloper, path: "/api/v1/clients/" + client.ID, wantStatus: http.StatusForbidden},
		{name: "qa is denied a client", token: TokenQA, path: "/api/v1/clients/" + client.ID, wantStatus: http.StatusForbidden},
		{name: "developer reads a project", token: TokenDeveloper, path: "/api/v1/projects/" + project.ID, wantStatus: http.StatusOK},
		{name: "devops is denied a project", token: TokenDevOps, path: "/api/v1/projects/" + project.ID, wantStatus: http.StatusForbidden},
		{name: "developer reads a document", token: TokenDeveloper, path: "/api/v1/documents/" + document.ID, wantStatus: http.StatusOK},
		{name: "ungrouped is denied a document", token: TokenNoGroups, path: "/api/v1/documents/" + document.ID, wantStatus: http.StatusForbidden},
		{name: "unauthenticated", path: "/api/v1/clients/" + client.ID, wantStatus: http.StatusUnauthorized},
		{name: "expired token", token: TokenExpired, path: "/api/v1/clients/" + client.ID, wantStatus: http.StatusUnauthorized},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := harness.API(t, http.MethodGet, test.path, test.token, nil)
			if response.Status != test.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", response.Status, test.wantStatus, truncate(response.Body))
			}
			assertNoSecretsOrTopology(t, response.Body)
		})
	}
}

// --- IDOR ------------------------------------------------------------------

// A Global ID of the wrong entity type must not be readable through another
// entity's endpoint, however well-formed it looks. Nothing may be derivable
// from the shape of an identifier.
func TestGlobalIDsAreNotInterchangeableBetweenEndpoints(t *testing.T) {
	harness := ready(t)
	client := collect(t, harness, "/api/v1/clients", TokenAdmin)[0]
	contact := collect(t, harness, "/api/v1/contacts", TokenAdmin)[0]
	project := collect(t, harness, "/api/v1/projects", TokenAdmin)[0]
	issue := collect(t, harness, "/api/v1/issues", TokenAdmin)[0]
	document := collect(t, harness, "/api/v1/documents", TokenAdmin)[0]

	endpoints := map[string]string{
		"clients":   client.ID,
		"contacts":  contact.ID,
		"projects":  project.ID,
		"issues":    issue.ID,
		"documents": document.ID,
	}
	for endpoint := range endpoints {
		for owner, id := range endpoints {
			if owner == endpoint {
				continue
			}
			t.Run(fmt.Sprintf("%s via %s", id, endpoint), func(t *testing.T) {
				response := harness.API(t, http.MethodGet, "/api/v1/"+endpoint+"/"+id, TokenAdmin, nil)
				if response.Status != http.StatusNotFound {
					t.Fatalf("status = %d, want 404 (body: %s)", response.Status, truncate(response.Body))
				}
				// The refusal must not confirm what the identifier really is.
				if strings.Contains(strings.ToLower(string(response.Body)), strings.ToLower(owner[:len(owner)-1])) {
					t.Fatalf("the refusal disclosed the entity's real type: %s", truncate(response.Body))
				}
			})
		}
	}
}

// A manipulated or invented Global ID must be refused identically to one that
// exists but is out of reach, so the endpoint cannot be used as an oracle.
func TestManipulatedGlobalIDsAreRefusedIdentically(t *testing.T) {
	harness := ready(t)
	clients := collect(t, harness, "/api/v1/clients", TokenAdmin)
	client := clients[0]

	tests := []struct {
		name string
		id   string
	}{
		{name: "an id that was never allocated", id: "CL-999999"},
		{name: "the id just past the allocated range", id: beyondAllocated(t, clients)},
		{name: "a lower-cased prefix", id: strings.ToLower(client.ID)},
		{name: "the upstream source id", id: client.SourceID},
		{name: "a malformed id", id: "CL-"},
		{name: "a bare number", id: "1"},
		{name: "another entity's prefix", id: strings.Replace(client.ID, "CL-", "PR-", 1)},
	}

	var bodies []string
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := harness.API(t, http.MethodGet, "/api/v1/clients/"+test.id, TokenAdmin, nil)
			if response.Status != http.StatusNotFound {
				t.Fatalf("status = %d, want 404 (body: %s)", response.Status, truncate(response.Body))
			}
			assertNoSecretsOrTopology(t, response.Body)
			bodies = append(bodies, string(response.Body))
		})
	}
	for i, body := range bodies {
		if body != bodies[0] {
			t.Fatalf("refusals differ between cases %q and %q, which makes the endpoint an oracle:\n  %s\n  %s",
				tests[0].name, tests[i].name, bodies[0], body)
		}
	}
}

// Addressing a resource by its upstream identifier rather than its Global ID
// must not work. The source id is not an access token, and the platform
// boundary is the only way in.
func TestUpstreamSourceIDsAreNotAddressable(t *testing.T) {
	harness := ready(t)
	tests := []struct {
		name     string
		endpoint string
		sourceID string
	}{
		{name: "crm account", endpoint: "clients", sourceID: "acc-northwind"},
		{name: "crm contact", endpoint: "contacts", sourceID: "ct-anna"},
		{name: "redmine project", endpoint: "projects", sourceID: "101"},
		{name: "redmine issue", endpoint: "issues", sourceID: "5001"},
		{name: "outline document", endpoint: "documents", sourceID: "doc-runbook"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := harness.API(t, http.MethodGet, "/api/v1/"+test.endpoint+"/"+test.sourceID, TokenAdmin, nil)
			if response.Status != http.StatusNotFound {
				t.Fatalf("status = %d, want 404 (body: %s)", response.Status, truncate(response.Body))
			}
		})
	}
}

// A customer is scope-confined: with no grant it must not reach any record,
// and it must not be able to tell an existing record from an invented one.
func TestCustomerCannotProbeForResources(t *testing.T) {
	harness := ready(t)
	document := collect(t, harness, "/api/v1/documents", TokenAdmin)[0]
	client := collect(t, harness, "/api/v1/clients", TokenAdmin)[0]

	real := harness.API(t, http.MethodGet, "/api/v1/documents/"+document.ID, TokenCustomer, nil)
	invented := harness.API(t, http.MethodGet, "/api/v1/documents/DOC-999999", TokenCustomer, nil)

	if real.Status != http.StatusNotFound || invented.Status != http.StatusNotFound {
		t.Fatalf("a confined caller must not learn a resource exists: real = %d, invented = %d", real.Status, invented.Status)
	}
	if string(real.Body) != string(invented.Body) {
		t.Fatalf("the refusals differ, which reveals which resource exists:\n  %s\n  %s", truncate(real.Body), truncate(invented.Body))
	}
	assertNoSecretsOrTopology(t, real.Body)

	// A permission the Customer role does not carry at all is refused the same
	// way: nothing about the client is disclosed either.
	denied := harness.API(t, http.MethodGet, "/api/v1/clients/"+client.ID, TokenCustomer, nil)
	if denied.Status != http.StatusNotFound && denied.Status != http.StatusForbidden {
		t.Fatalf("status = %d, want a refusal", denied.Status)
	}
	if strings.Contains(string(denied.Body), client.Name) {
		t.Fatalf("the refusal disclosed the client: %s", truncate(denied.Body))
	}
}

// A missing mapping must never widen access: reading a contact whose upstream
// account is unmapped must not invent a client link, and must not allocate one
// as a side effect.
func TestUnmappedReferencesStayUnmapped(t *testing.T) {
	harness := ready(t)
	contacts := collect(t, harness, "/api/v1/contacts", TokenAdmin)

	var unowned entityView
	for _, contact := range contacts {
		if contact.SourceID == "ct-dmytro" {
			unowned = contact
		}
	}
	if unowned.ID == "" {
		t.Fatal("the unowned contact fixture is missing from the normalized response")
	}

	response := harness.API(t, http.MethodGet, "/api/v1/contacts/"+unowned.ID, TokenAdmin, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", response.Status, truncate(response.Body))
	}
	var detail entityView
	response.JSON(t, &detail)
	if detail.ClientID != "" {
		t.Fatalf("a contact with no upstream account got client_id = %q", detail.ClientID)
	}
}

// beyondAllocated returns the Global ID one past the highest one allocated in
// items. Walking to the next identifier is the most obvious thing an attacker
// tries, and it must be refused like any other unknown id.
func beyondAllocated(t *testing.T, items []entityView) string {
	t.Helper()
	prefix, width, highest := "", 0, 0
	for _, item := range items {
		separator := strings.LastIndex(item.ID, "-")
		if separator < 0 {
			t.Fatalf("Global ID %q has no prefix separator", item.ID)
		}
		var value int
		if _, err := fmt.Sscanf(item.ID[separator+1:], "%d", &value); err != nil {
			t.Fatalf("Global ID %q has no numeric suffix", item.ID)
		}
		if value >= highest {
			prefix, width, highest = item.ID[:separator], len(item.ID)-separator-1, value
		}
	}
	return fmt.Sprintf("%s-%0*d", prefix, width, highest+1)
}
