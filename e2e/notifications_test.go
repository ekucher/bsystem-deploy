package e2e

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

type notification struct {
	ID                 int64  `json:"id"`
	Event              string `json:"event"`
	Source             string `json:"source"`
	Severity           string `json:"severity"`
	Title              string `json:"title"`
	Body               string `json:"body"`
	DeepLink           string `json:"deep_link"`
	EntityID           string `json:"entity_id"`
	RecipientID        string `json:"recipient_id"`
	AudiencePermission string `json:"audience_permission"`
	CorrelationID      string `json:"correlation_id"`
	OccurredAt         string `json:"occurred_at"`
	Read               bool   `json:"read"`
}

type notificationPage struct {
	Data        []notification `json:"data"`
	Pagination  Pagination     `json:"pagination"`
	UnreadCount int            `json:"unread_count"`
}

// marker returns a value unique to this run. The stack's database outlives a
// single test, so scenarios identify their own notifications rather than
// asserting on absolute counts that earlier runs have already moved.
func marker(name string) string {
	return fmt.Sprintf("e2e-%s-%d", name, time.Now().UnixNano())
}

// publishEvent puts an event on the platform as the service identity.
func publishEvent(t *testing.T, h *Harness, envelope map[string]any) {
	t.Helper()
	response := h.Request(t, http.MethodPost, h.Core+"/api/service/v1/events", TokenService, envelope, nil)
	if response.Status != http.StatusAccepted {
		t.Fatalf("publish %v: status = %d, body = %s", envelope["event"], response.Status, truncate(response.Body))
	}
}

// listNotifications reads a caller's notifications.
func listNotifications(t *testing.T, h *Harness, token, query string) notificationPage {
	t.Helper()
	response := h.API(t, http.MethodGet, "/api/v1/notifications"+query, token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("list notifications: status = %d, body = %s", response.Status, truncate(response.Body))
	}
	var page notificationPage
	response.JSON(t, &page)
	return page
}

// findByBody returns the notification carrying a marker, walking pages until
// it is found or the collection is exhausted.
func findByBody(t *testing.T, h *Harness, token, body string) (notification, bool) {
	t.Helper()
	query := "?limit=100"
	for page := 0; page < 20; page++ {
		result := listNotifications(t, h, token, query)
		for _, item := range result.Data {
			if item.Body == body {
				return item, true
			}
		}
		if result.Pagination.NextCursor == "" {
			return notification{}, false
		}
		query = "?limit=100&cursor=" + result.Pagination.NextCursor
	}
	t.Fatal("notification search did not terminate")
	return notification{}, false
}

// A mapped event must reach the people whose permission it names, and nobody
// else. This is the whole point of addressing a notification to a permission
// rather than to a person.
func TestMappedEventReachesItsAudienceAndNoOneElse(t *testing.T) {
	h := ready(t)

	body := marker("backup")
	publishEvent(t, h, map[string]any{
		"event": "backup.failed", "source": "e2e-operations",
		"entity_id": "SRV-000004", "severity": "warning",
		"data": map[string]any{"message": body, "api_key": "must-not-be-carried"},
	})

	// The audience is operations.server.read: DevOps holds it, QA does not.
	// An administrator reads every audience.
	for _, reader := range []struct {
		name  string
		token string
		want  bool
	}{
		{"DevOps holds the audience permission", TokenDevOps, true},
		{"an administrator reads every audience", TokenAdmin, true},
		{"QA does not hold it", TokenQA, false},
		// A developer does hold operations.server.read in the RBAC model, and
		// so is in this audience. Stated as an expectation rather than
		// removed, because it is the line that says who a backup failure
		// reaches.
		{"a developer holds it too", TokenDeveloper, true},
		// A scope-confined principal reads only what names it, whatever its
		// role permissions say. Its permissions mean "inside my own scope",
		// and reading them as an audience would cross the tenant boundary.
		{"a customer is scope-confined", TokenCustomer, false},
		{"a principal with no groups resolves to nothing", TokenNoGroups, false},
	} {
		t.Run(reader.name, func(t *testing.T) {
			found, ok := findByBody(t, h, reader.token, body)
			if ok != reader.want {
				t.Fatalf("visible = %v, want %v", ok, reader.want)
			}
			if !ok {
				return
			}
			if found.AudiencePermission != "operations.server.read" {
				t.Errorf("audience = %q, want operations.server.read", found.AudiencePermission)
			}
			if found.RecipientID != "" {
				t.Errorf("a mapped event named a recipient %q; the platform does not know one", found.RecipientID)
			}
			// The platform classifies backup.failed as critical and the
			// publisher reported warning. The floor wins.
			if found.Severity != "critical" {
				t.Errorf("severity = %q, want critical: a publisher must not be able to downgrade it", found.Severity)
			}
			// No HUB page exists for a server yet, so no link is offered.
			if found.DeepLink != "" {
				t.Errorf("deep_link = %q, want none for SRV-*", found.DeepLink)
			}
			assertNoSecretsOrTopology(t, []byte(found.Body))
			if found.Body != body {
				t.Errorf("body = %q, want only the event message", found.Body)
			}
		})
	}
}

// An event the platform does not map must interrupt nobody, including an
// administrator who can read every audience.
func TestUnmappedEventsRaiseNoNotification(t *testing.T) {
	h := ready(t)

	body := marker("unmapped")
	publishEvent(t, h, map[string]any{
		"event": "client.updated", "source": "e2e-crm",
		"entity_id": "CL-000001", "data": map[string]any{"message": body},
	})

	if _, found := findByBody(t, h, TokenAdmin, body); found {
		t.Fatal("client.updated raised a notification; only mapped events should")
	}
}

// Read state is per user: one reader marking an audience notification read
// must not hide it from everyone else who shares that audience.
func TestReadStateIsPerUser(t *testing.T) {
	h := ready(t)

	body := marker("readstate")
	publishEvent(t, h, map[string]any{
		"event": "test.failed", "source": "e2e-qa",
		"entity_id": "TST-000001", "data": map[string]any{"message": body},
	})

	// QA and an administrator both see it; the administrator reads it.
	forQA, ok := findByBody(t, h, TokenQA, body)
	if !ok {
		t.Fatal("QA should see a qa.report.read notification")
	}
	if forQA.Read {
		t.Fatal("a new notification must start unread")
	}
	forAdmin, ok := findByBody(t, h, TokenAdmin, body)
	if !ok {
		t.Fatal("an administrator should see every audience")
	}

	before := listNotifications(t, h, TokenAdmin, "?limit=1").UnreadCount
	response := h.API(t, http.MethodPost, fmt.Sprintf("/api/v1/notifications/%d/read", forAdmin.ID), TokenAdmin, nil)
	if response.Status != http.StatusNoContent {
		t.Fatalf("mark read: status = %d, body = %s", response.Status, truncate(response.Body))
	}
	after := listNotifications(t, h, TokenAdmin, "?limit=1").UnreadCount
	if after != before-1 {
		t.Errorf("administrator unread count went %d -> %d, want a decrease of one", before, after)
	}

	// Marking again is idempotent rather than an error.
	if response := h.API(t, http.MethodPost, fmt.Sprintf("/api/v1/notifications/%d/read", forAdmin.ID), TokenAdmin, nil); response.Status != http.StatusNoContent {
		t.Errorf("marking an already-read notification: status = %d, want 204", response.Status)
	}

	again, ok := findByBody(t, h, TokenQA, body)
	if !ok {
		t.Fatal("the notification disappeared for QA after another user read it")
	}
	if again.Read {
		t.Error("QA's copy is marked read because a different user read it")
	}
}

// A notification a caller may not see must be answered exactly as one that
// does not exist, or notification ids become enumerable.
func TestUnreadableNotificationsAreIndistinguishableFromMissingOnes(t *testing.T) {
	h := ready(t)

	body := marker("idor")
	publishEvent(t, h, map[string]any{
		"event": "build.failed", "source": "e2e-ci",
		"entity_id": "REL-000001", "data": map[string]any{"message": body},
	})
	// development.repo.read: a developer holds it, QA does not.
	target, ok := findByBody(t, h, TokenDeveloper, body)
	if !ok {
		t.Fatal("a developer should see a development.repo.read notification")
	}

	refusals := map[string]Response{}
	for name, path := range map[string]string{
		"a notification addressed to another audience": fmt.Sprintf("/api/v1/notifications/%d/read", target.ID),
		"an id far beyond anything allocated":          fmt.Sprintf("/api/v1/notifications/%d/read", target.ID+1_000_000),
		"a negative id":                                "/api/v1/notifications/-1/read",
		"a non-numeric id":                             "/api/v1/notifications/abc/read",
	} {
		response := h.API(t, http.MethodPost, path, TokenQA, nil)
		if response.Status != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404 (body: %s)", name, response.Status, truncate(response.Body))
		}
		refusals[name] = response
	}

	// Byte-identical, not merely the same status: a different message would
	// distinguish "exists but not yours" from "does not exist".
	var reference string
	for name, response := range refusals {
		if reference == "" {
			reference = string(response.Body)
			continue
		}
		if string(response.Body) != reference {
			t.Errorf("%s answered differently from the other refusals: %q vs %q", name, truncate(response.Body), reference)
		}
	}

	// And the developer who may see it can still mark it read.
	if response := h.API(t, http.MethodPost, fmt.Sprintf("/api/v1/notifications/%d/read", target.ID), TokenDeveloper, nil); response.Status != http.StatusNoContent {
		t.Errorf("the entitled reader was refused: status = %d", response.Status)
	}
}

// A publisher that knows its recipient may name one, and only that person
// sees the result.
func TestDirectlyAddressedNotificationsReachOnlyTheirRecipient(t *testing.T) {
	h := ready(t)

	// Resolve a real Global user ID rather than inventing one.
	response := h.API(t, http.MethodGet, "/api/v1/me", TokenSupport, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("resolve the support identity: status = %d", response.Status)
	}
	var me struct {
		ID string `json:"id"`
	}
	response.JSON(t, &me)
	if me.ID == "" {
		t.Fatal("the support identity has no Global user ID")
	}

	body := marker("direct")
	created := h.Request(t, http.MethodPost, h.Core+"/api/service/v1/notifications", TokenService, map[string]any{
		"recipient_id": me.ID, "severity": "warning", "title": "Directly addressed",
		"body": body, "entity_id": "PR-000002", "source": "e2e-direct",
	}, nil)
	if created.Status != http.StatusCreated {
		t.Fatalf("raise a direct notification: status = %d, body = %s", created.Status, truncate(created.Body))
	}
	var raised notification
	created.JSON(t, &raised)
	if raised.DeepLink != "/projects/PR-000002" {
		t.Errorf("deep_link = %q, want the HUB project page", raised.DeepLink)
	}

	if found, ok := findByBody(t, h, TokenSupport, body); !ok {
		t.Error("the named recipient cannot see their own notification")
	} else if found.RecipientID != me.ID {
		t.Errorf("recipient_id = %q, want %q", found.RecipientID, me.ID)
	}
	// Not even an administrator sees a notification addressed to one person:
	// reading every audience is not the same as reading everyone's post.
	if _, ok := findByBody(t, h, TokenAdmin, body); ok {
		t.Error("an administrator can read a notification addressed to another person by name")
	}
}

// An address the platform cannot resolve is refused rather than stored:
// reporting success for a notification nobody can read would be a lie.
func TestUnresolvableAddressesAreRefused(t *testing.T) {
	h := ready(t)

	cases := map[string]map[string]any{
		"an unknown Global user ID": {"recipient_id": "USR-999999", "title": "x", "source": "e2e"},
		"an undefined permission":   {"audience_permission": "not.a.permission", "title": "x", "source": "e2e"},
		"no address at all":         {"title": "x", "source": "e2e"},
		"both addresses":            {"recipient_id": "USR-000001", "audience_permission": "portal.read", "title": "x", "source": "e2e"},
		"no title":                  {"audience_permission": "portal.read", "source": "e2e"},
	}
	for name, request := range cases {
		t.Run(name, func(t *testing.T) {
			response := h.Request(t, http.MethodPost, h.Core+"/api/service/v1/notifications", TokenService, request, nil)
			if response.Status != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body: %s)", response.Status, truncate(response.Body))
			}
			assertNoSecretsOrTopology(t, response.Body)
		})
	}
}

// Raising a notification is a machine capability. No human token may reach it,
// whatever its role.
func TestOnlyServiceIdentitiesMayRaiseNotifications(t *testing.T) {
	h := ready(t)

	request := map[string]any{"audience_permission": "portal.read", "title": "x", "source": "e2e"}
	for name, token := range map[string]string{
		"an administrator":  TokenAdmin,
		"a devops engineer": TokenDevOps,
		"no token":          "",
	} {
		t.Run(name, func(t *testing.T) {
			response := h.Request(t, http.MethodPost, h.Core+"/api/service/v1/notifications", token, request, nil)
			if response.Status != http.StatusUnauthorized && response.Status != http.StatusForbidden {
				t.Fatalf("status = %d, want 401 or 403 (body: %s)", response.Status, truncate(response.Body))
			}
		})
	}
}

// The collection must be walkable: following next_cursor until it is absent
// reaches every notification exactly once, and the walk terminates.
func TestNotificationPaginationWalksWithoutRepeatingOrLosing(t *testing.T) {
	h := ready(t)

	const raised = 5
	bodies := map[string]bool{}
	for i := 0; i < raised; i++ {
		body := marker(fmt.Sprintf("page-%d", i))
		bodies[body] = false
		publishEvent(t, h, map[string]any{
			"event": "server.offline", "source": "e2e-operations",
			"entity_id": "SRV-000004", "data": map[string]any{"message": body},
		})
	}

	seen := map[int64]bool{}
	query := "?limit=2"
	for page := 0; page < 200; page++ {
		result := listNotifications(t, h, TokenDevOps, query)
		if result.Pagination.Limit > 2 {
			t.Fatalf("limit=2 returned %d items", result.Pagination.Limit)
		}
		for _, item := range result.Data {
			if seen[item.ID] {
				t.Fatalf("notification %d was returned twice while walking", item.ID)
			}
			seen[item.ID] = true
			if _, ours := bodies[item.Body]; ours {
				bodies[item.Body] = true
			}
		}
		if result.Pagination.NextCursor == "" {
			for body, found := range bodies {
				if !found {
					t.Errorf("%s was never returned by the walk", body)
				}
			}
			return
		}
		query = "?limit=2&cursor=" + result.Pagination.NextCursor
	}
	t.Fatal("the pagination walk did not terminate")
}

// A cursor the platform did not issue must be refused rather than
// reinterpreted: silently starting somewhere else returns the wrong window and
// the caller never learns it happened.
func TestForgedNotificationCursorsAreRefused(t *testing.T) {
	h := ready(t)

	for _, cursor := range []string{"not-base64!", "bzoxMA", "0", "-1", "abc"} {
		response := h.API(t, http.MethodGet, "/api/v1/notifications?cursor="+cursor, TokenAdmin, nil)
		if response.Status != http.StatusBadRequest {
			t.Errorf("cursor %q: status = %d, want 400", cursor, response.Status)
		}
	}
}

// The unread filter must actually filter, and the counts must describe the
// whole visible collection rather than the page a caller happens to ask for.
func TestUnreadFilterAndCounts(t *testing.T) {
	h := ready(t)

	// Two, so the assertions below hold whatever earlier scenarios left in
	// the store — including none at all.
	for i := 0; i < 2; i++ {
		publishEvent(t, h, map[string]any{
			"event": "incident.created", "source": "e2e-support",
			"entity_id": "INC-000001", "data": map[string]any{"message": marker("unread")},
		})
	}

	page := listNotifications(t, h, TokenSupport, "?limit=1&unread=true")
	if len(page.Data) != 1 {
		t.Fatalf("unread page returned %d items, want 1", len(page.Data))
	}
	if page.Data[0].Read {
		t.Error("the unread filter returned a notification marked read")
	}
	if page.Pagination.Total < page.UnreadCount {
		t.Errorf("total %d is below unread_count %d; both must describe the whole visible collection",
			page.Pagination.Total, page.UnreadCount)
	}
	if page.Pagination.Total <= 1 || page.UnreadCount < 1 {
		t.Errorf("counts describe the page rather than the collection: total=%d unread=%d limit=%d",
			page.Pagination.Total, page.UnreadCount, page.Pagination.Limit)
	}
}
