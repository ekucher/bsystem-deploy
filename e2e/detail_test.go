package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
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

// A source record that disappears after its Global ID was allocated leaves the
// Global ID behind, and the read that finds nothing says so plainly.
//
// Global IDs are immutable and everything refers to them — audit rows, scope
// grants, support relations. A record deleted upstream must therefore not take
// its Global ID with it: the references held against it have to keep resolving
// to something, or the platform loses the ability to say what an audit entry
// was about.
//
// The other half is the answer. An upstream 404 arriving in the middle of a
// detail read is the platform meeting a record that is gone, not the platform
// breaking, so it must be normalized rather than surfaced as a failure. A 500
// here would send an operator looking for a fault in BSYSTEM over something
// that happened in EspoCRM.
func TestASourceRecordDeletedAfterAllocationKeepsItsGlobalID(t *testing.T) {
	harness := ready(t)

	clients := collect(t, harness, "/api/v1/clients", TokenAdmin)
	if len(clients) == 0 {
		t.Fatal("no clients to work with; the rest of this test would prove nothing")
	}
	client := clients[0]
	if client.ID == "" || client.SourceID == "" {
		t.Fatalf("a client arrived without an identity: %+v", client)
	}

	// This one record is now gone upstream, and only this one. The mock
	// matches a fault path exactly rather than by prefix, so the path has to
	// be the detail path the adapter actually requests — injecting on the
	// collection path would leave the detail read untouched and the test
	// asserting against a platform that met no fault at all.
	//
	// Naming the single record is also what the scenario means. "Everything is
	// down" is a different test, and this one is about a record that was
	// deleted while the rest of the upstream kept working.
	harness.InjectFault(t, harness.EspoCRM, "/api/v1/Account/"+client.SourceID, http.StatusNotFound, 0)

	response := harness.API(t, http.MethodGet, "/api/v1/clients/"+client.ID, TokenAdmin, nil)
	if response.Status == http.StatusInternalServerError {
		t.Fatalf("a record deleted upstream answered 500; that sends an operator looking for a fault in this platform (body: %s)", truncate(response.Body))
	}
	if response.Status != http.StatusNotFound && response.Status != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want a normalized 404 or 503 (body: %s)", response.Status, truncate(response.Body))
	}

	var failure platformError
	response.JSON(t, &failure)
	if failure.Code == "" {
		t.Error("the refusal carries no code; a caller is told to branch on one")
	}
	for _, leak := range []string{"SQLSTATE", "espocrm.invalid", "Bearer", "api_key", "127.0.0.1"} {
		if strings.Contains(string(response.Body), leak) {
			t.Errorf("the refusal carries %q: %s", leak, truncate(response.Body))
		}
	}

	// The Global ID outlives the record. Reading the mapping directly is the
	// question that matters: everything else in the platform refers to this id
	// and has to keep resolving.
	harness.ResetFaults(t, harness.EspoCRM)
	mapping := harness.API(t, http.MethodGet, "/api/v1/global-ids/"+client.ID, TokenAdmin, nil)
	if mapping.Status != http.StatusOK {
		t.Fatalf("the Global ID stopped resolving after its source record was deleted: status = %d (body: %s)", mapping.Status, truncate(mapping.Body))
	}
	var entity struct {
		GlobalID string `json:"global_id"`
		SourceID string `json:"source_id"`
	}
	mapping.JSON(t, &entity)
	if entity.GlobalID != client.ID || entity.SourceID != client.SourceID {
		t.Errorf("the mapping changed: %+v, want %s/%s", entity, client.ID, client.SourceID)
	}
}

// An upstream that answers 200 with a payload the adapter cannot read is a
// different failure from an upstream that is down, and a more dangerous one.
// A refusal is easy to see. A payload that decodes to nothing, or half-decodes,
// is the shape that produces an empty collection nobody questions, or Global
// IDs minted against records that were never really there.
//
// Three things must hold, and the third is the one the P23 item is named for:
//
//   - the read that met the broken payload is refused, with the platform's
//     normalized upstream error rather than the decoder's own words;
//   - the records the broken payload did not describe are untouched — contacts
//     come from the same upstream, over a different path, and must still read;
//   - once the upstream is healthy again, the clients read returns the same
//     Global IDs it returned before. Nothing was dropped, and nothing was
//     minted a second time for a record that already had an ID.
func TestAMalformedUpstreamPayloadCorruptsNothingAroundIt(t *testing.T) {
	harness := ready(t)

	before := map[string]string{}
	for _, item := range collect(t, harness, "/api/v1/clients", TokenAdmin) {
		before[item.SourceID] = item.ID
	}
	contactsBefore := len(collect(t, harness, "/api/v1/contacts", TokenAdmin))

	// A truncated object, not a random string: it starts out looking like the
	// response the adapter expects, so anything that decodes optimistically
	// reads a list and then runs off the end.
	const broken = `{"total": 3, "list": [{"id": "acc-broken", "name": "Tru`
	harness.InjectFaultBody(t, harness.EspoCRM, "/api/v1/Account", http.StatusOK, broken)

	response := harness.API(t, http.MethodGet, "/api/v1/clients", TokenAdmin, nil)
	if response.Status == http.StatusOK {
		t.Fatalf("GET /api/v1/clients answered 200 on an unreadable upstream payload: %s", truncate(response.Body))
	}
	if response.Status != http.StatusBadGateway {
		t.Errorf("GET /api/v1/clients: status = %d, want %d (body: %s)", response.Status, http.StatusBadGateway, truncate(response.Body))
	}
	var failure struct {
		Error  string `json:"error"`
		Code   string `json:"code"`
		Source string `json:"source"`
	}
	response.JSON(t, &failure)
	if failure.Code != "upstream_unavailable" {
		t.Errorf("error code = %q, want %q", failure.Code, "upstream_unavailable")
	}
	if failure.Source != "espocrm" {
		t.Errorf("error source = %q, want %q", failure.Source, "espocrm")
	}
	// The decoder's complaint names the offset, the Go type and often the
	// fragment it choked on. That is upstream payload and internal type
	// topology, and neither belongs in an answer to an API caller.
	body := strings.ToLower(string(response.Body))
	for _, leak := range []string{"unmarshal", "invalid character", "unexpected end", "acc-broken", "json:"} {
		if strings.Contains(body, leak) {
			t.Errorf("error body leaks the decode failure (%q): %s", leak, truncate(response.Body))
		}
	}

	// Contacts are served by the same upstream over a different path. The
	// broken Account payload must not reach them.
	if got := len(collect(t, harness, "/api/v1/contacts", TokenAdmin)); got != contactsBefore {
		t.Errorf("contacts after the malformed Account payload = %d, want %d", got, contactsBefore)
	}

	harness.ResetFaults(t, harness.EspoCRM)

	after := map[string]string{}
	for _, item := range collect(t, harness, "/api/v1/clients", TokenAdmin) {
		after[item.SourceID] = item.ID
	}
	if len(after) != len(before) {
		t.Fatalf("clients after recovery = %d, want %d", len(after), len(before))
	}
	for sourceID, id := range before {
		if after[sourceID] != id {
			t.Errorf("client %s: Global ID = %q after recovery, want %q", sourceID, after[sourceID], id)
		}
	}
}

// Allocation on first sighting is by nature something several requests do at
// once: a record nothing has seen before becomes visible to everything at the
// same moment, and anything that fans out asks for it in parallel.
//
// Every one of those callers must come away with the same Global ID. Two IDs
// for one source record is the failure the whole identity scheme is built to
// prevent — audit entries, scope grants and support relations would then be
// held against two different names for the same thing, with nothing in the
// data saying they are the same. A 500 to the losers is the milder failure and
// still wrong: the caller asked whether this record has an ID, the answer
// exists, and losing a race is not the caller's problem.
//
// The source record is named after the test run so that it is genuinely new
// on every run. A source_id some earlier scenario already mapped would send
// every request down the read path and the race would never happen.
func TestConcurrentAllocationsOfOneNewRecordAgreeOnOneGlobalID(t *testing.T) {
	harness := ready(t)

	sourceID := fmt.Sprintf("acc-concurrent-%d", time.Now().UnixNano())
	const callers = 8

	type outcome struct {
		status   int
		globalID string
		body     []byte
		err      error
	}
	results := make([]outcome, callers)
	start := make(chan struct{})
	var group sync.WaitGroup
	for index := range callers {
		group.Add(1)
		go func() {
			defer group.Done()
			// Released together, so the requests overlap inside the platform
			// rather than queueing behind each other's setup.
			<-start
			response, err := harness.Do(http.MethodPost, harness.Core+"/api/v1/global-ids", TokenAdmin, map[string]any{
				"entity_type": "client",
				"source":      "espocrm",
				"source_id":   sourceID,
			})
			if err != nil {
				results[index] = outcome{err: err}
				return
			}
			var created struct {
				GlobalID string `json:"global_id"`
			}
			_ = json.Unmarshal(response.Body, &created)
			results[index] = outcome{status: response.Status, globalID: created.GlobalID, body: response.Body}
		}()
	}
	close(start)
	group.Wait()

	allocated := map[string]int{}
	for index, result := range results {
		if result.err != nil {
			t.Fatalf("caller %d: %v", index, result.err)
		}
		if result.status != http.StatusCreated {
			t.Fatalf("caller %d: status = %d, want %d (body: %s)", index, result.status, http.StatusCreated, truncate(result.body))
		}
		if result.globalID == "" {
			t.Fatalf("caller %d: answered %d without a global_id (body: %s)", index, result.status, truncate(result.body))
		}
		allocated[result.globalID]++
	}
	if len(allocated) != 1 {
		t.Fatalf("%d callers allocated %d distinct Global IDs for one source record: %v", callers, len(allocated), allocated)
	}

	var globalID string
	for id := range allocated {
		globalID = id
	}
	if !strings.HasPrefix(globalID, "CL-") {
		t.Errorf("global_id = %q, want a client ID carrying the CL- prefix", globalID)
	}

	// The ID the concurrent callers agreed on is the one the platform will
	// answer with from now on. An ID that resolves to a different source
	// record, or to nothing, would mean the agreement was on a value that was
	// never stored.
	response := harness.API(t, http.MethodGet, "/api/v1/global-ids/"+globalID, TokenAdmin, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET /api/v1/global-ids/%s: status = %d, want 200 (body: %s)", globalID, response.Status, truncate(response.Body))
	}
	var mapping struct {
		GlobalID string `json:"global_id"`
		Source   string `json:"source"`
		SourceID string `json:"source_id"`
	}
	response.JSON(t, &mapping)
	if mapping.GlobalID != globalID || mapping.SourceID != sourceID || mapping.Source != "espocrm" {
		t.Errorf("mapping = %s -> %s/%s, want %s -> espocrm/%s", mapping.GlobalID, mapping.Source, mapping.SourceID, globalID, sourceID)
	}
}
