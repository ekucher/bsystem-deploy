// Package fixtures holds the deterministic dataset shared by every BSYSTEM
// mock upstream.
//
// The data is invented for testing. It must never be replaced with a copy of a
// production payload: fixtures are committed to a public repository and are
// classified PUBLIC.
//
// Identifiers are stable across restarts so that Global ID allocation in the
// Integration Core is reproducible between E2E runs.
package fixtures

// Account mirrors the EspoCRM Account fields the CRM adapter consumes.
type Account struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Website string `json:"website,omitempty"`
	Email   string `json:"emailAddress,omitempty"`
	Phone   string `json:"phoneNumber,omitempty"`
}

// Contact mirrors the EspoCRM Contact fields the CRM adapter consumes.
type Contact struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AccountID string `json:"accountId,omitempty"`
	Email     string `json:"emailAddress,omitempty"`
	Phone     string `json:"phoneNumber,omitempty"`
}

// Project mirrors the Redmine project fields the Redmine adapter consumes.
type Project struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Identifier  string `json:"identifier"`
	Description string `json:"description,omitempty"`
	Status      int    `json:"status"`
}

// IssueRef is the nested project/status shape Redmine returns on an issue.
type IssueRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Issue mirrors the Redmine issue fields the Redmine adapter consumes.
type Issue struct {
	ID      int      `json:"id"`
	Subject string   `json:"subject"`
	Project IssueRef `json:"project"`
	Status  IssueRef `json:"status"`
}

// Document mirrors the Outline document fields the Outline adapter consumes.
type Document struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Text         string `json:"text,omitempty"`
	URL          string `json:"url,omitempty"`
	CollectionID string `json:"collectionId,omitempty"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
}

// Accounts is the ordered EspoCRM account fixture set.
func Accounts() []Account {
	return []Account{
		{ID: "acc-globex", Name: "Globex Industrial", Website: "https://globex.example.invalid", Email: "ops@globex.example.invalid", Phone: "+380000000002"},
		{ID: "acc-initech", Name: "Initech Software", Website: "https://initech.example.invalid", Email: "hello@initech.example.invalid", Phone: "+380000000003"},
		{ID: "acc-northwind", Name: "Northwind Trading", Website: "https://northwind.example.invalid", Email: "info@northwind.example.invalid", Phone: "+380000000001"},
	}
}

// Contacts is the ordered EspoCRM contact fixture set. The final entry has no
// account so that unmapped-owner handling stays covered.
func Contacts() []Contact {
	return []Contact{
		{ID: "ct-anna", Name: "Anna Kovalenko", AccountID: "acc-northwind", Email: "anna@northwind.example.invalid", Phone: "+380000000011"},
		{ID: "ct-boris", Name: "Boris Melnyk", AccountID: "acc-globex", Email: "boris@globex.example.invalid", Phone: "+380000000012"},
		{ID: "ct-clara", Name: "Clara Doyle", AccountID: "acc-initech", Email: "clara@initech.example.invalid", Phone: "+380000000013"},
		{ID: "ct-dmytro", Name: "Dmytro Shevchenko", Email: "dmytro@example.invalid", Phone: "+380000000014"},
	}
}

// Projects is the ordered Redmine project fixture set.
func Projects() []Project {
	return []Project{
		{ID: 101, Name: "Northwind Portal", Identifier: "northwind-portal", Description: "Customer portal rollout", Status: 1},
		{ID: 102, Name: "Globex SCADA", Identifier: "globex-scada", Description: "Industrial telemetry integration", Status: 1},
		{ID: 103, Name: "Initech Billing", Identifier: "initech-billing", Description: "Billing modernization", Status: 1},
	}
}

// Issues is the ordered Redmine issue fixture set, spanning several projects
// and statuses.
func Issues() []Issue {
	return []Issue{
		{ID: 5001, Subject: "Design portal navigation", Project: IssueRef{ID: 101, Name: "Northwind Portal"}, Status: IssueRef{ID: 1, Name: "New"}},
		{ID: 5002, Subject: "Implement OIDC login", Project: IssueRef{ID: 101, Name: "Northwind Portal"}, Status: IssueRef{ID: 2, Name: "In Progress"}},
		{ID: 5003, Subject: "Collect telemetry baseline", Project: IssueRef{ID: 102, Name: "Globex SCADA"}, Status: IssueRef{ID: 5, Name: "Closed"}},
		{ID: 5004, Subject: "Model invoice lifecycle", Project: IssueRef{ID: 103, Name: "Initech Billing"}, Status: IssueRef{ID: 1, Name: "New"}},
		{ID: 5005, Subject: "Harden SCADA gateway", Project: IssueRef{ID: 102, Name: "Globex SCADA"}, Status: IssueRef{ID: 3, Name: "Resolved"}},
	}
}

// Documents is the ordered Outline document fixture set.
func Documents() []Document {
	return []Document{
		{ID: "doc-onboarding", Title: "BSYSTEM Onboarding", Text: "How to get started with the platform.", URL: "/doc/bsystem-onboarding", CollectionID: "col-internal", UpdatedAt: "2026-01-05T10:00:00.000Z"},
		{ID: "doc-runbook", Title: "Integration Core Runbook", Text: "Operational runbook for the Integration Core.", URL: "/doc/integration-core-runbook", CollectionID: "col-internal", UpdatedAt: "2026-01-06T11:30:00.000Z"},
		{ID: "doc-northwind-sla", Title: "Northwind SLA", Text: "Service levels agreed with Northwind Trading.", URL: "/doc/northwind-sla", CollectionID: "col-customer", UpdatedAt: "2026-01-07T09:15:00.000Z"},
		{ID: "doc-globex-notes", Title: "Globex Migration Notes", Text: "Migration notes for the Globex SCADA rollout.", URL: "/doc/globex-migration-notes", CollectionID: "col-customer", UpdatedAt: "2026-01-08T16:45:00.000Z"},
	}
}
