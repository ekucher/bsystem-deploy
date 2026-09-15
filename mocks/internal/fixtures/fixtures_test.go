package fixtures

import (
	"strconv"
	"testing"
)

// The Integration Core allocates Global IDs keyed by (source, source_id), so
// duplicate fixture identifiers would silently collapse two entities into one.
func TestFixtureIdentifiersAreUnique(t *testing.T) {
	t.Run("accounts", func(t *testing.T) { assertUnique(t, ids(Accounts(), func(a Account) string { return a.ID })) })
	t.Run("contacts", func(t *testing.T) { assertUnique(t, ids(Contacts(), func(c Contact) string { return c.ID })) })
	t.Run("projects", func(t *testing.T) {
		assertUnique(t, ids(Projects(), func(p Project) string { return strconv.Itoa(p.ID) }))
		assertUnique(t, ids(Projects(), func(p Project) string { return p.Identifier }))
	})
	t.Run("issues", func(t *testing.T) { assertUnique(t, ids(Issues(), func(i Issue) string { return strconv.Itoa(i.ID) })) })
	t.Run("documents", func(t *testing.T) { assertUnique(t, ids(Documents(), func(d Document) string { return d.ID })) })
}

// Every cross-entity reference must resolve, otherwise a normalization test
// would assert against a relationship the upstream data cannot express.
func TestFixtureReferencesResolve(t *testing.T) {
	accounts := map[string]bool{}
	for _, account := range Accounts() {
		accounts[account.ID] = true
	}
	for _, contact := range Contacts() {
		if contact.AccountID != "" && !accounts[contact.AccountID] {
			t.Errorf("contact %s references unknown account %s", contact.ID, contact.AccountID)
		}
	}

	projects := map[int]string{}
	for _, project := range Projects() {
		projects[project.ID] = project.Name
	}
	for _, issue := range Issues() {
		name, ok := projects[issue.Project.ID]
		if !ok {
			t.Errorf("issue %d references unknown project %d", issue.ID, issue.Project.ID)
			continue
		}
		if name != issue.Project.Name {
			t.Errorf("issue %d project name = %q, want %q", issue.ID, issue.Project.Name, name)
		}
	}
}

// At least one contact must be unowned so that the missing-owner path in the
// normalized API stays exercised.
func TestAtLeastOneContactHasNoAccount(t *testing.T) {
	for _, contact := range Contacts() {
		if contact.AccountID == "" {
			return
		}
	}
	t.Fatal("no contact fixture without an account")
}

// Fixtures are returned fresh so that a mock handler slicing a page cannot
// mutate the shared dataset for later requests.
func TestFixturesAreNotShared(t *testing.T) {
	first := Accounts()
	first[0].Name = "mutated"
	if Accounts()[0].Name == "mutated" {
		t.Fatal("Accounts() returned a shared backing array")
	}
}

func ids[T any](items []T, key func(T) string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, key(item))
	}
	return result
}

func assertUnique(t *testing.T, values []string) {
	t.Helper()
	seen := map[string]bool{}
	for _, value := range values {
		if value == "" {
			t.Errorf("empty identifier in fixture set")
		}
		if seen[value] {
			t.Errorf("duplicate identifier %q", value)
		}
		seen[value] = true
	}
}
