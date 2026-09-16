package e2e

import (
	"net/http"
	"strings"
	"testing"
)

// Upstream field mapping, end to end.
//
// Each field below travels through four independent declarations before a
// caller sees it: the mock's struct tag, the adapter's struct tag, the
// adapter's mapping into a view, and the view's own tag. The first two are
// separate hand-maintained copies of the same external API — one in
// bsystem-deploy/mocks, one in bsystem-integration-core/internal/adapters —
// and nothing compares them.
//
// Every optional field is `omitempty` at each layer, so a rename or a typo at
// any of the four does not fail: the field simply stops appearing. The rest of
// the suite would not notice. It asserts source_id and Global ID identity
// thoroughly and checks name only for being non-empty, which a broken mapping
// of email, website, phone, description or a relation survives untouched.
//
// The result would be a platform serving clients with no email and contacts
// with no phone number, with green tests and an E2E stack reporting success.
//
// These values are the fixtures in bsystem-deploy/mocks/internal/fixtures.
// They are written out here rather than imported because the e2e module
// deliberately has no dependencies — and because a contract test that derives
// its expectation from the thing under test proves nothing.

// find returns the item a collection carries for one upstream id.
func find(t *testing.T, items []entityView, sourceID string) entityView {
	t.Helper()
	for _, item := range items {
		if item.SourceID == sourceID {
			return item
		}
	}
	t.Fatalf("no item with source_id %q in a collection of %d", sourceID, len(items))
	return entityView{}
}

func equal(t *testing.T, field, got, want string) {
	t.Helper()
	if got == want {
		return
	}
	if got == "" {
		t.Errorf("%s is absent; an omitempty field that stops being mapped disappears rather than failing (want %q)", field, want)
		return
	}
	t.Errorf("%s = %q, want %q", field, got, want)
}

// A client carries everything EspoCRM holds about it, not only its name.
func TestClientFieldsSurviveTheJourney(t *testing.T) {
	harness := ready(t)

	client := find(t, collect(t, harness, "/api/v1/clients", TokenAdmin), "acc-northwind")

	equal(t, "name", client.Name, "Northwind Trading")
	equal(t, "website", client.Website, "https://northwind.example.invalid")
	equal(t, "email", client.Email, "info@northwind.example.invalid")
	equal(t, "phone", client.Phone, "+380000000001")

	// The detail endpoint reads the upstream record again rather than replaying
	// the collection, so it can drift on its own.
	response := harness.API(t, http.MethodGet, "/api/v1/clients/"+client.ID, TokenAdmin, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET client detail: status = %d (body: %s)", response.Status, truncate(response.Body))
	}
	var detail entityView
	response.JSON(t, &detail)
	equal(t, "detail website", detail.Website, client.Website)
	equal(t, "detail email", detail.Email, client.Email)
	equal(t, "detail phone", detail.Phone, client.Phone)
}

// A contact carries its own detail and the Global ID of the client that owns
// it — a mapped identifier, never the upstream account id.
func TestContactFieldsAndOwnershipSurviveTheJourney(t *testing.T) {
	harness := ready(t)
	contacts := collect(t, harness, "/api/v1/contacts", TokenAdmin)

	anna := find(t, contacts, "ct-anna")
	equal(t, "name", anna.Name, "Anna Kovalenko")
	equal(t, "email", anna.Email, "anna@northwind.example.invalid")
	equal(t, "phone", anna.Phone, "+380000000011")

	// The owner is resolved to a Global ID. Leaking the upstream account id
	// here would make an internal identifier of a foreign system addressable,
	// which is the property TestUpstreamSourceIDsAreNotAddressable guards from
	// the other side.
	if !strings.HasPrefix(anna.ClientID, "CL-") {
		t.Errorf("client_id = %q, want a CL- Global ID", anna.ClientID)
	}
	if anna.ClientID == "acc-northwind" {
		t.Error("client_id is the upstream account id; the owner must be mapped, not passed through")
	}

	// And it is the right client: the one whose own source_id is the account
	// the fixture names.
	owner := find(t, collect(t, harness, "/api/v1/clients", TokenAdmin), "acc-northwind")
	if anna.ClientID != owner.ID {
		t.Errorf("client_id = %q, want %q — the contact is owned by a different client than the fixture says", anna.ClientID, owner.ID)
	}

	// A contact with no account must carry no owner rather than an empty
	// string that reads as one, and must not have an identifier minted for a
	// client that does not exist.
	dmytro := find(t, contacts, "ct-dmytro")
	equal(t, "name", dmytro.Name, "Dmytro Shevchenko")
	equal(t, "email", dmytro.Email, "dmytro@example.invalid")
	if dmytro.ClientID != "" {
		t.Errorf("an ownerless contact carries client_id %q; reading a contact must not mint an owner", dmytro.ClientID)
	}
}

// A project carries its identifier and description, neither of which any other
// test reads.
func TestProjectFieldsSurviveTheJourney(t *testing.T) {
	harness := ready(t)

	project := find(t, collect(t, harness, "/api/v1/projects", TokenAdmin), "101")

	equal(t, "name", project.Name, "Northwind Portal")
	equal(t, "identifier", project.Identifier, "northwind-portal")
	equal(t, "description", project.Description, "Customer portal rollout")
}

// An issue carries its status name and the Global ID of its project. Redmine
// nests both inside objects, so the mapping has one more layer to lose.
func TestIssueFieldsAndProjectLinkSurviveTheJourney(t *testing.T) {
	harness := ready(t)

	issue := find(t, collect(t, harness, "/api/v1/issues", TokenAdmin), "5002")

	equal(t, "subject", issue.Subject, "Implement OIDC login")
	// The status is the upstream's name for it, not its numeric id: a number
	// would be meaningless to a reader and would change meaning if Redmine
	// renumbered.
	equal(t, "status", issue.Status, "In Progress")

	if !strings.HasPrefix(issue.ProjectID, "PR-") {
		t.Errorf("project_id = %q, want a PR- Global ID", issue.ProjectID)
	}
	owner := find(t, collect(t, harness, "/api/v1/projects", TokenAdmin), "101")
	if issue.ProjectID != owner.ID {
		t.Errorf("project_id = %q, want %q", issue.ProjectID, owner.ID)
	}
}

// A document carries its URL, collection and timestamp. The URL is what makes
// a search result actionable; without it a reader finds the document and
// cannot open it.
func TestDocumentFieldsSurviveTheJourney(t *testing.T) {
	harness := ready(t)

	document := find(t, collect(t, harness, "/api/v1/documents", TokenAdmin), "doc-northwind-sla")

	equal(t, "title", document.Title, "Northwind SLA")
	equal(t, "url", document.URL, "/doc/northwind-sla")
	equal(t, "collection_id", document.CollectionID, "col-customer")
	equal(t, "updated_at", document.UpdatedAt, "2026-01-07T09:15:00.000Z")
}

// The document body is the one upstream field the platform deliberately does
// not carry. Outline holds the text; BSYSTEM holds the mapping, and a
// normalized collection that returned document contents would be a second copy
// of the wiki governed by nothing.
func TestDocumentTextIsNotCarried(t *testing.T) {
	harness := ready(t)

	response := harness.API(t, http.MethodGet, "/api/v1/documents", TokenAdmin, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET documents: status = %d", response.Status)
	}
	if body := string(response.Body); strings.Contains(body, "Service levels agreed with Northwind Trading") {
		t.Error("a document's text reached the normalized collection; the platform holds mappings, not copies")
	}
}
