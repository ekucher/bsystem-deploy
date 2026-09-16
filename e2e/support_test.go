package e2e

import (
	"net/http"
	"testing"
)

type supportRelation struct {
	EntityType string `json:"entity_type"`
	GlobalID   string `json:"global_id"`
}

type supportRecord struct {
	ID             string            `json:"id"`
	Kind           string            `json:"kind"`
	Title          string            `json:"title"`
	Summary        string            `json:"summary"`
	Severity       string            `json:"severity"`
	Status         string            `json:"status"`
	ClientID       string            `json:"client_id"`
	Relations      []supportRelation `json:"relations"`
	ReportedBy     string            `json:"reported_by"`
	AcknowledgedAt string            `json:"acknowledged_at"`
	ResolvedAt     string            `json:"resolved_at"`
	SLA            struct {
		State     string `json:"state"`
		RespondBy string `json:"respond_by"`
		ResolveBy string `json:"resolve_by"`
	} `json:"sla"`
}

type supportPage struct {
	Data       []supportRecord `json:"data"`
	Pagination Pagination      `json:"pagination"`
}

func raiseIncident(t *testing.T, h *Harness, token string, body map[string]any) supportRecord {
	t.Helper()
	response := h.API(t, http.MethodPost, "/api/v1/incidents", token, body)
	if response.Status != http.StatusCreated {
		t.Fatalf("raise incident: status = %d, body = %s", response.Status, truncate(response.Body))
	}
	var record supportRecord
	response.JSON(t, &record)
	return record
}

func patchIncident(t *testing.T, h *Harness, token, id string, body map[string]any) Response {
	t.Helper()
	return h.API(t, http.MethodPatch, "/api/v1/incidents/"+id, token, body)
}

// With no SLA policy configured the platform must make no claim at all.
// "On track" would tell a customer a promise is being kept when the business
// has not made one.
func TestWithNoSLAPolicyThePlatformMakesNoClaim(t *testing.T) {
	h := ready(t)

	record := raiseIncident(t, h, TokenSupport, map[string]any{
		"title": marker("sla"), "severity": "critical",
	})
	if record.SLA.State != "unset" {
		t.Errorf("sla state = %q, want unset with no policy configured", record.SLA.State)
	}
	if record.SLA.RespondBy != "" || record.SLA.ResolveBy != "" {
		t.Errorf("due dates were invented without a policy: %+v", record.SLA)
	}
}

// Severity has no default: answering it on the reporter's behalf would pick
// their response time for them.
func TestSeverityIsRequiredAndValidated(t *testing.T) {
	h := ready(t)

	for name, body := range map[string]map[string]any{
		"no severity":      {"title": "x"},
		"unknown severity": {"title": "x", "severity": "sev1"},
		"no title":         {"severity": "high"},
		"unknown kind":     {"title": "x", "severity": "high", "kind": "ticket"},
	} {
		t.Run(name, func(t *testing.T) {
			response := h.API(t, http.MethodPost, "/api/v1/incidents", TokenSupport, body)
			if response.Status != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body: %s)", response.Status, truncate(response.Body))
			}
			assertNoSecretsOrTopology(t, response.Body)
		})
	}
}

// The lifecycle is a closed graph, and the refusal names both ends.
func TestTheSupportLifecycleIsEnforced(t *testing.T) {
	h := ready(t)

	record := raiseIncident(t, h, TokenSupport, map[string]any{
		"title": marker("lifecycle"), "severity": "high",
	})
	if record.Status != "new" || record.AcknowledgedAt != "" || record.ResolvedAt != "" {
		t.Fatalf("a new record started as %+v", record)
	}

	// Walk it forward, checking the SLA moments are recorded as it goes.
	steps := []struct{ status, wantStatus string }{
		{"acknowledged", "acknowledged"},
		{"in_progress", "in_progress"},
		{"resolved", "resolved"},
		// A fix that did not hold is the same incident.
		{"in_progress", "in_progress"},
		{"resolved", "resolved"},
		{"closed", "closed"},
	}
	var resolvedAt string
	for _, step := range steps {
		response := patchIncident(t, h, TokenSupport, record.ID, map[string]any{"status": step.status})
		if response.Status != http.StatusOK {
			t.Fatalf("-> %s: status = %d, body = %s", step.status, response.Status, truncate(response.Body))
		}
		var updated supportRecord
		response.JSON(t, &updated)
		if updated.Status != step.wantStatus {
			t.Fatalf("status = %q, want %q", updated.Status, step.wantStatus)
		}
		if updated.AcknowledgedAt == "" {
			t.Error("acknowledged_at was not recorded when the record left new")
		}
		if step.wantStatus == "resolved" || step.wantStatus == "closed" {
			if updated.ResolvedAt == "" {
				t.Error("resolved_at was not recorded")
			}
			// Reopening must not erase that the record was once resolved on
			// time, so the first resolution timestamp has to survive.
			if resolvedAt == "" {
				resolvedAt = updated.ResolvedAt
			} else if updated.ResolvedAt != resolvedAt {
				t.Errorf("resolved_at moved from %s to %s; the original resolution was erased", resolvedAt, updated.ResolvedAt)
			}
		}
	}

	// Closed is terminal.
	for _, status := range []string{"new", "acknowledged", "in_progress", "resolved"} {
		response := patchIncident(t, h, TokenSupport, record.ID, map[string]any{"status": status})
		if response.Status != http.StatusConflict {
			t.Errorf("closed -> %s: status = %d, want 409", status, response.Status)
		}
	}
	// And an unknown status is a 400 rather than a 409: the value is wrong,
	// not the move.
	response := patchIncident(t, h, TokenSupport, record.ID, map[string]any{"status": "reticulating"})
	if response.Status != http.StatusBadRequest {
		t.Errorf("unknown status: status = %d, want 400", response.Status)
	}
}

// A relation to something the platform cannot identify is a dangling pointer
// that reads as a fact.
func TestRelationsAreVerifiedBeforeTheyAreStored(t *testing.T) {
	h := ready(t)

	for name, relations := range map[string][]map[string]any{
		"an unknown Global ID": {{"entity_type": "server", "global_id": "SRV-999999"}},
		"the wrong type":       {{"entity_type": "client", "global_id": "SRV-000001"}},
		"a type not allowed":   {{"entity_type": "user", "global_id": "USR-000001"}},
		"not a Global ID":      {{"entity_type": "server", "global_id": "nonsense"}},
	} {
		t.Run(name, func(t *testing.T) {
			response := h.API(t, http.MethodPost, "/api/v1/incidents", TokenSupport, map[string]any{
				"title": marker("relation"), "severity": "low", "relations": relations,
			})
			if response.Status != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body: %s)", response.Status, truncate(response.Body))
			}
		})
	}

	// A relation that does resolve is stored and returned on the detail read.
	bravo := newReporter(t, h)
	host := bravo.register(t, marker("related-host"), "related-host", "prod", nil)
	record := raiseIncident(t, h, TokenSupport, map[string]any{
		"title": marker("related"), "severity": "medium",
		"relations": []map[string]any{{"entity_type": "server", "global_id": host.ID}},
	})
	response := h.API(t, http.MethodGet, "/api/v1/incidents/"+record.ID, TokenSupport, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("read incident: status = %d", response.Status)
	}
	var found supportRecord
	response.JSON(t, &found)
	if len(found.Relations) != 1 || found.Relations[0].GlobalID != host.ID {
		t.Errorf("relations = %+v, want the server", found.Relations)
	}
}

// Reading and writing need their own permissions, and a scope-confined
// customer is refused the collection outright.
func TestSupportAuthorization(t *testing.T) {
	h := ready(t)
	record := raiseIncident(t, h, TokenSupport, map[string]any{
		"title": marker("authz"), "severity": "low",
	})

	t.Run("reading", func(t *testing.T) {
		for name, c := range map[string]struct {
			token string
			want  int
		}{
			"support holds it":             {TokenSupport, http.StatusOK},
			"a manager holds it":           {TokenManager, http.StatusOK},
			"an administrator holds it":    {TokenAdmin, http.StatusOK},
			"a developer does not":         {TokenDeveloper, http.StatusForbidden},
			"a customer is scope-confined": {TokenCustomer, http.StatusForbidden},
			"no groups resolved":           {TokenNoGroups, http.StatusForbidden},
		} {
			t.Run(name, func(t *testing.T) {
				response := h.API(t, http.MethodGet, "/api/v1/incidents", c.token, nil)
				if response.Status != c.want {
					t.Errorf("list: status = %d, want %d", response.Status, c.want)
				}
			})
		}
	})

	t.Run("writing needs support.incident.write", func(t *testing.T) {
		// A manager may read incidents but not raise or change them.
		response := h.API(t, http.MethodPost, "/api/v1/incidents", TokenManager,
			map[string]any{"title": "x", "severity": "low"})
		if response.Status != http.StatusForbidden {
			t.Errorf("manager create: status = %d, want 403", response.Status)
		}
		response = patchIncident(t, h, TokenManager, record.ID, map[string]any{"status": "acknowledged"})
		if response.Status != http.StatusForbidden {
			t.Errorf("manager update: status = %d, want 403", response.Status)
		}
	})

	// A caller who cannot hold the permission at all must get the same answer
	// for a record that exists and one that does not, or the endpoint
	// enumerates incidents.
	t.Run("a refused caller cannot tell an existing record from a missing one", func(t *testing.T) {
		existing := h.API(t, http.MethodGet, "/api/v1/incidents/"+record.ID, TokenDeveloper, nil)
		missing := h.API(t, http.MethodGet, "/api/v1/incidents/INC-999999", TokenDeveloper, nil)
		if existing.Status != missing.Status {
			t.Errorf("existing gave %d and missing gave %d", existing.Status, missing.Status)
		}
		if string(existing.Body) != string(missing.Body) {
			t.Errorf("the two refusals differ: %q vs %q", truncate(existing.Body), truncate(missing.Body))
		}
	})
}

// Filters must be validated rather than passed through: an unknown value
// would match nothing, and "no results" reads as "none exist".
func TestSupportFiltersAreValidatedAndApplied(t *testing.T) {
	h := ready(t)

	title := marker("filtered")
	record := raiseIncident(t, h, TokenSupport, map[string]any{
		"title": title, "severity": "critical", "kind": "request",
	})

	for _, query := range []string{"?status=reticulating", "?severity=sev1", "?kind=ticket"} {
		response := h.API(t, http.MethodGet, "/api/v1/incidents"+query, TokenSupport, nil)
		if response.Status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", query, response.Status)
		}
	}

	page := supportPage{}
	response := h.API(t, http.MethodGet, "/api/v1/incidents?kind=request&severity=critical&open=true&limit=100", TokenSupport, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("filtered list: status = %d", response.Status)
	}
	response.JSON(t, &page)
	found := false
	for _, item := range page.Data {
		if item.Kind != "request" || item.Severity != "critical" {
			t.Errorf("the filter returned a %s/%s", item.Kind, item.Severity)
		}
		if item.Status == "resolved" || item.Status == "closed" {
			t.Errorf("open=true returned a %s record", item.Status)
		}
		if item.ID == record.ID {
			found = true
		}
	}
	if !found {
		t.Error("the record the filter describes was not returned")
	}

	// Once closed it must leave the open listing.
	if response := patchIncident(t, h, TokenSupport, record.ID, map[string]any{"status": "closed"}); response.Status != http.StatusOK {
		t.Fatalf("close: status = %d", response.Status)
	}
	response = h.API(t, http.MethodGet, "/api/v1/incidents?open=true&limit=100", TokenSupport, nil)
	response.JSON(t, &page)
	for _, item := range page.Data {
		if item.ID == record.ID {
			t.Error("a closed record is still in the open listing")
		}
	}
}

// Raising an incident interrupts the people who handle them; resolving it
// does not.
func TestIncidentCreationNotifiesAndResolutionDoesNot(t *testing.T) {
	h := ready(t)

	title := marker("notify-incident")
	record := raiseIncident(t, h, TokenSupport, map[string]any{"title": title, "severity": "high"})
	if _, found := findByBody(t, h, TokenSupport, title); !found {
		t.Error("raising an incident did not notify the people who handle incidents")
	}

	if response := patchIncident(t, h, TokenSupport, record.ID, map[string]any{"status": "resolved"}); response.Status != http.StatusOK {
		t.Fatalf("resolve: status = %d", response.Status)
	}
	// The resolution event carries the same title, so a second notification
	// would show up as a duplicate of the first.
	page := listNotifications(t, h, TokenSupport, "?limit=100")
	seen := 0
	for _, item := range page.Data {
		if item.Body == title {
			seen++
		}
	}
	if seen != 1 {
		t.Errorf("%d notifications carry this incident's title, want exactly the one raised at creation", seen)
	}
}

// The summary must not travel in a published event: it is where a reporter
// pastes logs and occasionally a credential, and an event goes further than
// the record it came from.
func TestTheIncidentSummaryDoesNotTravelInEvents(t *testing.T) {
	h := ready(t)

	title := marker("title-only")
	secret := "summary-" + marker("secret")
	raiseIncident(t, h, TokenSupport, map[string]any{
		"title": title, "summary": secret, "severity": "critical",
	})

	notification, found := findByBody(t, h, TokenSupport, title)
	if !found {
		t.Fatal("the incident raised no notification")
	}
	if notification.Body == secret {
		t.Error("the summary was carried into the notification body")
	}
	page := listNotifications(t, h, TokenSupport, "?limit=100")
	for _, item := range page.Data {
		if item.Body == secret {
			t.Errorf("notification %d carries the incident summary", item.ID)
		}
	}
}
