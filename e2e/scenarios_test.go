package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

type meResponse struct {
	ID          string   `json:"id"`
	Subject     string   `json:"subject"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	Username    string   `json:"username"`
	Groups      []string `json:"groups"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	Modules     []string `json:"modules"`
}

type entityView struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	SourceID string `json:"source_id"`
	// Only one of these is populated per entity type.
	Name         string `json:"name"`
	Subject      string `json:"subject"`
	Title        string `json:"title"`
	ClientID     string `json:"client_id"`
	ProjectID    string `json:"project_id"`
	Identifier   string `json:"identifier"`
	Status       string `json:"status"`
	CollectionID string `json:"collection_id"`
	UpdatedAt    string `json:"updated_at"`
	// Optional upstream detail. Every one of these is `omitempty` all the way
	// down, so a broken field mapping removes it from the response rather than
	// failing — see upstream_fields_test.go.
	Website     string `json:"website"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// collectionResponse is the envelope every normalized collection returns.
type collectionResponse struct {
	Data       []entityView `json:"data"`
	Pagination Pagination   `json:"pagination"`
}

type platformError struct {
	Error     string `json:"error"`
	Code      string `json:"code"`
	Source    string `json:"source"`
	RequestID string `json:"request_id"`
}

// sustainedFailures is comfortably more than the adapter retry budget, so an
// injected fault represents an outage rather than a blip the adapter absorbs.
const sustainedFailures = 8

func ready(t *testing.T) *Harness {
	t.Helper()
	harness := New(t)
	harness.WaitReady(t, 90*time.Second)
	return harness
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

// --- Platform surface -------------------------------------------------------

func TestPlatformEndpointsAreServed(t *testing.T) {
	harness := ready(t)
	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{name: "health", path: "/health", wantStatus: http.StatusOK},
		{name: "readiness", path: "/readyz", wantStatus: http.StatusOK},
		{name: "metrics", path: "/metrics", wantStatus: http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := harness.API(t, http.MethodGet, test.path, "", nil)
			if response.Status != test.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", response.Status, test.wantStatus, truncate(response.Body))
			}
		})
	}
}

// Every adapter must report ready, which proves the Integration Core reaches
// all three mock upstreams with the configured test credentials.
func TestAdaptersReachTheMockUpstreams(t *testing.T) {
	harness := ready(t)
	response := harness.API(t, http.MethodGet, "/api/service/v1/adapters/health", TokenService, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", response.Status, truncate(response.Body))
	}
	var health map[string]struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	response.JSON(t, &health)
	for _, adapter := range []string{"espocrm", "redmine", "outline"} {
		state, ok := health[adapter]
		if !ok {
			t.Fatalf("adapter %s is missing from the health report: %+v", adapter, health)
		}
		if state.Status != "ready" {
			t.Errorf("adapter %s status = %q (%s), want ready", adapter, state.Status, state.Message)
		}
	}
}

// --- Identity and authorization --------------------------------------------

func TestUserGlobalIDIsStable(t *testing.T) {
	harness := ready(t)
	first := harness.API(t, http.MethodGet, "/api/v1/me", TokenAdmin, nil)
	if first.Status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", first.Status, truncate(first.Body))
	}
	var me meResponse
	first.JSON(t, &me)
	if !strings.HasPrefix(me.ID, "USR-") {
		t.Fatalf("id = %q, want a USR-* Global ID", me.ID)
	}
	if me.Subject != "mock-admin" {
		t.Fatalf("subject = %q, want mock-admin", me.Subject)
	}

	second := harness.API(t, http.MethodGet, "/api/v1/me", TokenAdmin, nil)
	var again meResponse
	second.JSON(t, &again)
	if again.ID != me.ID {
		t.Fatalf("Global ID is not stable across requests: %q then %q", me.ID, again.ID)
	}
}

func TestServiceGlobalIDIsStable(t *testing.T) {
	harness := ready(t)
	var first, second struct {
		ID          string   `json:"id"`
		Subject     string   `json:"subject"`
		Roles       []string `json:"roles"`
		Permissions []string `json:"permissions"`
	}
	harness.API(t, http.MethodGet, "/api/service/v1/whoami", TokenService, nil).JSON(t, &first)
	if !strings.HasPrefix(first.ID, "SVC-") {
		t.Fatalf("id = %q, want an SVC-* Global ID", first.ID)
	}
	if !contains(first.Permissions, "events.publish") || !contains(first.Permissions, "adapters.read") {
		t.Fatalf("service permissions = %v, want events.publish and adapters.read", first.Permissions)
	}
	harness.API(t, http.MethodGet, "/api/service/v1/whoami", TokenService, nil).JSON(t, &second)
	if second.ID != first.ID {
		t.Fatalf("service Global ID is not stable: %q then %q", first.ID, second.ID)
	}
}

func TestRolesResolveFromGroups(t *testing.T) {
	harness := ready(t)
	tests := []struct {
		name           string
		token          string
		wantRole       string
		wantPermission string
	}{
		{name: "administrator", token: TokenAdmin, wantRole: "Administrator", wantPermission: "*"},
		{name: "manager", token: TokenManager, wantRole: "Manager", wantPermission: "crm.client.read"},
		{name: "developer", token: TokenDeveloper, wantRole: "Developer", wantPermission: "projects.task.read"},
		{name: "qa", token: TokenQA, wantRole: "QA", wantPermission: "qa.testcase.execute"},
		{name: "support", token: TokenSupport, wantRole: "Support", wantPermission: "support.incident.write"},
		{name: "devops", token: TokenDevOps, wantRole: "DevOps", wantPermission: "operations.server.manage"},
		{name: "customer", token: TokenCustomer, wantRole: "Customer", wantPermission: "portal.read"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := harness.API(t, http.MethodGet, "/api/v1/me", test.token, nil)
			if response.Status != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body: %s)", response.Status, truncate(response.Body))
			}
			var me meResponse
			response.JSON(t, &me)
			if !contains(me.Roles, test.wantRole) {
				t.Fatalf("roles = %v, want %s", me.Roles, test.wantRole)
			}
			if !contains(me.Permissions, test.wantPermission) {
				t.Fatalf("permissions = %v, want %s", me.Permissions, test.wantPermission)
			}
		})
	}
}

// A principal with no group must authenticate but resolve to no role at all:
// deny-by-default lives in the platform, not in the identity provider.
func TestPrincipalWithoutGroupsGetsNoAccess(t *testing.T) {
	harness := ready(t)
	response := harness.API(t, http.MethodGet, "/api/v1/me", TokenNoGroups, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", response.Status, truncate(response.Body))
	}
	var me meResponse
	response.JSON(t, &me)
	if len(me.Roles) != 0 || len(me.Permissions) != 0 || len(me.Modules) != 0 {
		t.Fatalf("ungrouped principal resolved access: roles=%v permissions=%v modules=%v", me.Roles, me.Permissions, me.Modules)
	}

	for _, path := range []string{"/api/v1/clients", "/api/v1/contacts", "/api/v1/projects", "/api/v1/issues", "/api/v1/documents"} {
		t.Run(path, func(t *testing.T) {
			denied := harness.API(t, http.MethodGet, path, TokenNoGroups, nil)
			if denied.Status != http.StatusForbidden {
				t.Fatalf("status = %d, want 403 (body: %s)", denied.Status, truncate(denied.Body))
			}
		})
	}
}

func TestAuthenticationRejections(t *testing.T) {
	harness := ready(t)
	tests := []struct {
		name  string
		token string
	}{
		{name: "no token"},
		{name: "unknown token", token: TokenUnknown},
		{name: "expired token", token: TokenExpired},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := harness.API(t, http.MethodGet, "/api/v1/me", test.token, nil)
			if response.Status != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401 (body: %s)", response.Status, truncate(response.Body))
			}
		})
	}
}

// The service API is a separate boundary: a valid human token must not reach
// it, and a service token missing the service group must be refused.
func TestServiceAPIBoundary(t *testing.T) {
	harness := ready(t)
	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "service identity", token: TokenService, wantStatus: http.StatusOK},
		{name: "service without the services group", token: TokenServiceUngrouped, wantStatus: http.StatusForbidden},
		{name: "human token", token: TokenDeveloper, wantStatus: http.StatusForbidden},
		{name: "expired token", token: TokenExpired, wantStatus: http.StatusUnauthorized},
		{name: "no token", wantStatus: http.StatusUnauthorized},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := harness.API(t, http.MethodGet, "/api/service/v1/whoami", test.token, nil)
			if response.Status != test.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", response.Status, test.wantStatus, truncate(response.Body))
			}
		})
	}
}

func TestModuleFiltering(t *testing.T) {
	harness := ready(t)
	modulesFor := func(t *testing.T, token string) map[string]bool {
		t.Helper()
		response := harness.API(t, http.MethodGet, "/api/v1/modules", token, nil)
		if response.Status != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body: %s)", response.Status, truncate(response.Body))
		}
		var modules []struct {
			ID string `json:"id"`
		}
		response.JSON(t, &modules)
		result := map[string]bool{}
		for _, module := range modules {
			result[module.ID] = true
		}
		return result
	}

	admin := modulesFor(t, TokenAdmin)
	customer := modulesFor(t, TokenCustomer)
	ungrouped := modulesFor(t, TokenNoGroups)

	if len(admin) == 0 {
		t.Fatal("the administrator must see at least one module")
	}
	if len(ungrouped) != 0 {
		t.Fatalf("an ungrouped principal must see no modules, got %v", ungrouped)
	}
	if len(customer) >= len(admin) {
		t.Fatalf("the customer module set (%d) must be narrower than the administrator's (%d)", len(customer), len(admin))
	}
	for id := range customer {
		if !admin[id] {
			t.Fatalf("customer sees module %q that the administrator does not", id)
		}
	}
	if customer["crm"] || customer["development"] {
		t.Fatalf("customer must not see internal modules, got %v", customer)
	}
}

// --- Normalization and Global IDs ------------------------------------------

func TestNormalizedEntitiesAndGlobalIDStability(t *testing.T) {
	harness := ready(t)
	tests := []struct {
		name       string
		path       string
		token      string
		wantPrefix string
		wantSource string
		wantCount  int
		check      func(t *testing.T, item entityView)
	}{
		{
			name: "clients", path: "/api/v1/clients", token: TokenAdmin, wantPrefix: "CL-", wantSource: "espocrm", wantCount: 3,
			check: func(t *testing.T, item entityView) {
				if item.Name == "" {
					t.Error("client name must be normalized")
				}
			},
		},
		{
			name: "contacts", path: "/api/v1/contacts", token: TokenAdmin, wantPrefix: "CT-", wantSource: "espocrm", wantCount: 4,
			check: func(t *testing.T, item entityView) {
				if item.Name == "" {
					t.Error("contact name must be normalized")
				}
			},
		},
		{
			name: "projects", path: "/api/v1/projects", token: TokenAdmin, wantPrefix: "PR-", wantSource: "redmine", wantCount: 3,
			check: func(t *testing.T, item entityView) {
				if item.Identifier == "" {
					t.Error("project identifier must be normalized")
				}
			},
		},
		{
			name: "issues", path: "/api/v1/issues", token: TokenAdmin, wantPrefix: "TSK-", wantSource: "redmine", wantCount: 5,
			check: func(t *testing.T, item entityView) {
				if item.Subject == "" {
					t.Error("issue subject must be normalized")
				}
				if !strings.HasPrefix(item.ProjectID, "PR-") {
					t.Errorf("issue project_id = %q, want a PR-* Global ID", item.ProjectID)
				}
			},
		},
		{
			name: "documents", path: "/api/v1/documents", token: TokenAdmin, wantPrefix: "DOC-", wantSource: "outline", wantCount: 4,
			check: func(t *testing.T, item entityView) {
				if item.Title == "" {
					t.Error("document title must be normalized")
				}
				if item.CollectionID == "" {
					t.Error("document collection_id must be normalized")
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := harness.API(t, http.MethodGet, test.path, test.token, nil)
			if response.Status != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body: %s)", response.Status, truncate(response.Body))
			}
			var body collectionResponse
			response.JSON(t, &body)
			items := body.Data
			if body.Pagination.Total != test.wantCount {
				t.Fatalf("pagination.total = %d, want %d", body.Pagination.Total, test.wantCount)
			}
			if len(items) != test.wantCount {
				t.Fatalf("returned %d items, want %d", len(items), test.wantCount)
			}

			byGlobalID := map[string]string{}
			for _, item := range items {
				if !strings.HasPrefix(item.ID, test.wantPrefix) {
					t.Errorf("id = %q, want prefix %s", item.ID, test.wantPrefix)
				}
				if item.Source != test.wantSource {
					t.Errorf("source = %q, want %s", item.Source, test.wantSource)
				}
				if item.SourceID == "" {
					t.Error("source_id must be preserved so the source system stays authoritative")
				}
				if previous, seen := byGlobalID[item.ID]; seen {
					t.Errorf("Global ID %q was allocated twice (%s and %s)", item.ID, previous, item.SourceID)
				}
				byGlobalID[item.ID] = item.SourceID
				test.check(t, item)
			}

			// Global IDs are immutable: a second read of the same upstream
			// records must return exactly the same mapping.
			repeat := harness.API(t, http.MethodGet, test.path, test.token, nil)
			var repeated collectionResponse
			repeat.JSON(t, &repeated)
			for _, item := range repeated.Data {
				sourceID, ok := byGlobalID[item.ID]
				if !ok {
					t.Errorf("Global ID %q appeared only on the second read", item.ID)
					continue
				}
				if sourceID != item.SourceID {
					t.Errorf("Global ID %q remapped from source %s to %s", item.ID, sourceID, item.SourceID)
				}
			}
		})
	}
}

// A contact without an upstream account must not invent a client link. An
// absent mapping has to narrow what is returned, never broaden it.
func TestUnmappedContactHasNoClientLink(t *testing.T) {
	harness := ready(t)
	var body collectionResponse
	harness.API(t, http.MethodGet, "/api/v1/contacts", TokenAdmin, nil).JSON(t, &body)
	found := false
	for _, contact := range body.Data {
		if contact.SourceID != "ct-dmytro" {
			continue
		}
		found = true
		if contact.ClientID != "" {
			t.Fatalf("contact without an upstream account got client_id = %q", contact.ClientID)
		}
	}
	if !found {
		t.Fatal("the unowned contact fixture is missing from the normalized response")
	}
}

func TestCustomerIsDeniedUnscopedDocuments(t *testing.T) {
	harness := ready(t)
	response := harness.API(t, http.MethodGet, "/api/v1/documents", TokenCustomer, nil)
	if response.Status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body: %s)", response.Status, truncate(response.Body))
	}
	var failure platformError
	response.JSON(t, &failure)
	if failure.Code != "scope_required" {
		t.Fatalf("code = %q, want scope_required", failure.Code)
	}
	// The denial must not disclose any document the customer cannot see.
	for _, title := range []string{"Northwind SLA", "Integration Core Runbook", "doc-onboarding"} {
		if strings.Contains(string(response.Body), title) {
			t.Fatalf("denial leaked document data: %s", truncate(response.Body))
		}
	}
}

func TestBusinessAuthorizationMatrix(t *testing.T) {
	harness := ready(t)
	tests := []struct {
		name       string
		token      string
		path       string
		wantStatus int
	}{
		{name: "developer reads projects", token: TokenDeveloper, path: "/api/v1/projects", wantStatus: http.StatusOK},
		{name: "developer reads issues", token: TokenDeveloper, path: "/api/v1/issues", wantStatus: http.StatusOK},
		{name: "developer is denied clients", token: TokenDeveloper, path: "/api/v1/clients", wantStatus: http.StatusForbidden},
		{name: "support reads clients", token: TokenSupport, path: "/api/v1/clients", wantStatus: http.StatusOK},
		{name: "qa is denied clients", token: TokenQA, path: "/api/v1/clients", wantStatus: http.StatusForbidden},
		{name: "devops is denied projects", token: TokenDevOps, path: "/api/v1/projects", wantStatus: http.StatusForbidden},
		{name: "customer is denied clients", token: TokenCustomer, path: "/api/v1/clients", wantStatus: http.StatusForbidden},
		{name: "customer is denied projects", token: TokenCustomer, path: "/api/v1/projects", wantStatus: http.StatusForbidden},
		{name: "administrator reads documents", token: TokenAdmin, path: "/api/v1/documents", wantStatus: http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := harness.API(t, http.MethodGet, test.path, test.token, nil)
			if response.Status != test.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", response.Status, test.wantStatus, truncate(response.Body))
			}
		})
	}
}

// Global ID administration is administrator-only. The administrator is let
// through authorization and then meets a normalized 404; nobody else gets far
// enough to learn whether the Global ID exists.
func TestGlobalIDAdministrationIsRestricted(t *testing.T) {
	harness := ready(t)
	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "administrator", token: TokenAdmin, wantStatus: http.StatusNotFound},
		{name: "manager", token: TokenManager, wantStatus: http.StatusForbidden},
		{name: "developer", token: TokenDeveloper, wantStatus: http.StatusForbidden},
		{name: "support", token: TokenSupport, wantStatus: http.StatusForbidden},
		{name: "customer", token: TokenCustomer, wantStatus: http.StatusForbidden},
		{name: "ungrouped", token: TokenNoGroups, wantStatus: http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := harness.API(t, http.MethodGet, "/api/v1/global-ids/CL-999999", test.token, nil)
			if response.Status != test.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", response.Status, test.wantStatus, truncate(response.Body))
			}
		})
	}
}

// Creating a Global ID mapping is administrator-only too, otherwise any role
// could mint platform identifiers for upstream records.
func TestGlobalIDCreationIsAdministratorOnly(t *testing.T) {
	harness := ready(t)
	body := map[string]any{"entity_type": "client", "source": "e2e", "source_id": "e2e-denied"}
	for _, token := range []string{TokenManager, TokenDeveloper, TokenSupport, TokenCustomer, TokenNoGroups} {
		t.Run(token, func(t *testing.T) {
			response := harness.API(t, http.MethodPost, "/api/v1/global-ids", token, body)
			if response.Status != http.StatusForbidden {
				t.Fatalf("status = %d, want 403 (body: %s)", response.Status, truncate(response.Body))
			}
		})
	}
}

// --- Audit and request correlation -----------------------------------------

func TestAuditAndRequestIDPropagation(t *testing.T) {
	harness := ready(t)
	requestID := fmt.Sprintf("e2e-%d", time.Now().UnixNano())
	sourceID := fmt.Sprintf("e2e-audit-%d", time.Now().UnixNano())

	created := harness.Request(t, http.MethodPost, harness.Core+"/api/v1/global-ids", TokenAdmin,
		map[string]any{"entity_type": "client", "source": "e2e", "source_id": sourceID},
		map[string]string{"X-Request-ID": requestID})
	if created.Status != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", created.Status, truncate(created.Body))
	}
	if echoed := created.Headers.Get("X-Request-ID"); echoed != requestID {
		t.Fatalf("X-Request-ID = %q, want %q", echoed, requestID)
	}
	var entity struct {
		GlobalID string `json:"global_id"`
		SourceID string `json:"source_id"`
	}
	created.JSON(t, &entity)
	if !strings.HasPrefix(entity.GlobalID, "CL-") {
		t.Fatalf("global_id = %q, want a CL-* Global ID", entity.GlobalID)
	}

	// A Global ID is immutable: recreating the same mapping must return the
	// identifier already allocated.
	repeat := harness.API(t, http.MethodPost, "/api/v1/global-ids", TokenAdmin,
		map[string]any{"entity_type": "client", "source": "e2e", "source_id": sourceID})
	var again struct {
		GlobalID string `json:"global_id"`
	}
	repeat.JSON(t, &again)
	if again.GlobalID != entity.GlobalID {
		t.Fatalf("Global ID changed on re-mapping: %q then %q", entity.GlobalID, again.GlobalID)
	}

	audit := harness.API(t, http.MethodGet, "/api/v1/audit?limit=100", TokenAdmin, nil)
	if audit.Status != http.StatusOK {
		t.Fatalf("audit status = %d, want 200 (body: %s)", audit.Status, truncate(audit.Body))
	}
	var events []struct {
		Action       string `json:"action"`
		ResourceID   string `json:"resource_id"`
		RequestID    string `json:"request_id"`
		GlobalUserID string `json:"global_user_id"`
		Subject      string `json:"subject"`
	}
	audit.JSON(t, &events)
	for _, event := range events {
		if event.RequestID != requestID {
			continue
		}
		if event.Action != "global_id.created" {
			t.Fatalf("audit action = %q, want global_id.created", event.Action)
		}
		if event.ResourceID != entity.GlobalID {
			t.Fatalf("audit resource_id = %q, want %q", event.ResourceID, entity.GlobalID)
		}
		if !strings.HasPrefix(event.GlobalUserID, "USR-") {
			t.Fatalf("audit global_user_id = %q, want a USR-* Global ID", event.GlobalUserID)
		}
		return
	}
	t.Fatalf("no audit event carries request ID %q; correlation is broken between HUB, core and audit", requestID)
}

// The audit trail is administrator-only: it records who read what.
func TestAuditIsAdministratorOnly(t *testing.T) {
	harness := ready(t)
	for _, token := range []string{TokenDeveloper, TokenSupport, TokenCustomer, TokenNoGroups} {
		t.Run(token, func(t *testing.T) {
			response := harness.API(t, http.MethodGet, "/api/v1/audit", token, nil)
			if response.Status != http.StatusForbidden {
				t.Fatalf("status = %d, want 403 (body: %s)", response.Status, truncate(response.Body))
			}
		})
	}
}

// A generated request ID must still be returned, so that every platform
// response is traceable even when the caller supplies nothing.
func TestRequestIDIsGeneratedWhenAbsent(t *testing.T) {
	harness := ready(t)
	response := harness.API(t, http.MethodGet, "/health", "", nil)
	if response.Headers.Get("X-Request-ID") == "" {
		t.Fatal("X-Request-ID must be generated when the caller does not supply one")
	}
}

// --- Upstream failure normalization ----------------------------------------

func TestUpstreamErrorsAreNormalized(t *testing.T) {
	harness := ready(t)
	tests := []struct {
		name       string
		mockURL    string
		mockPath   string
		injectCode int
		apiPath    string
		wantSource string
	}{
		{name: "espocrm unauthorized", mockURL: harness.EspoCRM, mockPath: "/api/v1/Account", injectCode: http.StatusUnauthorized, apiPath: "/api/v1/clients", wantSource: "espocrm"},
		{name: "espocrm rate limited", mockURL: harness.EspoCRM, mockPath: "/api/v1/Contact", injectCode: http.StatusTooManyRequests, apiPath: "/api/v1/contacts", wantSource: "espocrm"},
		{name: "espocrm server error", mockURL: harness.EspoCRM, mockPath: "/api/v1/Account", injectCode: http.StatusInternalServerError, apiPath: "/api/v1/clients", wantSource: "espocrm"},
		{name: "redmine not found", mockURL: harness.Redmine, mockPath: "/projects.json", injectCode: http.StatusNotFound, apiPath: "/api/v1/projects", wantSource: "redmine"},
		{name: "redmine server error", mockURL: harness.Redmine, mockPath: "/issues.json", injectCode: http.StatusInternalServerError, apiPath: "/api/v1/issues", wantSource: "redmine"},
		{name: "outline rate limited", mockURL: harness.Outline, mockPath: "/api/documents.list", injectCode: http.StatusTooManyRequests, apiPath: "/api/v1/documents", wantSource: "outline"},
		{name: "outline server error", mockURL: harness.Outline, mockPath: "/api/documents.list", injectCode: http.StatusInternalServerError, apiPath: "/api/v1/documents", wantSource: "outline"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// The failure has to outlast the retry budget. A single injected
			// fault would be retried away, which is the adapter behaving
			// correctly but says nothing about how a real outage normalizes.
			harness.InjectFaultTimes(t, test.mockURL, test.mockPath, test.injectCode, 0, sustainedFailures)
			response := harness.API(t, http.MethodGet, test.apiPath, TokenAdmin, nil)
			assertNormalizedUpstreamFailure(t, response, test.wantSource)

			// The platform must recover once the upstream does: a normalized
			// error is not a latched failure.
			harness.ResetFaults(t, test.mockURL)
			recovered := harness.API(t, http.MethodGet, test.apiPath, TokenAdmin, nil)
			if recovered.Status != http.StatusOK {
				t.Fatalf("after the upstream recovered, status = %d, want 200 (body: %s)", recovered.Status, truncate(recovered.Body))
			}
		})
	}
}

// A hanging upstream must surface as the same normalized error as an explicit
// failure, rather than as a stalled request.
func TestUpstreamTimeoutIsNormalized(t *testing.T) {
	harness := ready(t)
	// Every attempt must hang, or the retry would find a healthy upstream and
	// the timeout would never reach the caller. The delay comfortably exceeds
	// the stack's per-attempt bound.
	harness.InjectFaultTimes(t, harness.Redmine, "/projects.json", 0, 8000, sustainedFailures)
	start := time.Now()
	response := harness.API(t, http.MethodGet, "/api/v1/projects", TokenAdmin, nil)
	elapsed := time.Since(start)
	assertNormalizedUpstreamFailure(t, response, "redmine")
	// Bounded attempts against a hanging upstream must still return well
	// before the injected delay would have.
	if elapsed > 30*time.Second {
		t.Fatalf("the adapter did not bound the upstream call: it took %s", elapsed)
	}
}

func assertNormalizedUpstreamFailure(t *testing.T, response Response, wantSource string) {
	t.Helper()
	if response.Status != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502 (body: %s)", response.Status, truncate(response.Body))
	}
	var failure platformError
	response.JSON(t, &failure)
	if failure.Code != "upstream_unavailable" {
		t.Fatalf("code = %q, want upstream_unavailable", failure.Code)
	}
	if failure.Source != wantSource {
		t.Fatalf("source = %q, want %q", failure.Source, wantSource)
	}
	assertNoSecretsOrTopology(t, response.Body)
}

// A normalized error must never disclose credentials or internal topology.
func assertNoSecretsOrTopology(t *testing.T, body []byte) {
	t.Helper()
	payload := strings.ToLower(string(body))
	forbidden := []string{
		EspoCRMTestAPIKey, RedmineTestAPIKey, OutlineTestAPIKey, PostgresTestPassword,
		"mock-espocrm:8090", "mock-redmine:8091", "mock-outline:8092", "mock-identity:9000",
		"postgres://", "x-api-key", "x-redmine-api-key", "authorization",
		"goroutine", "panic:", "pq:", "pgx",
	}
	for _, needle := range forbidden {
		if strings.Contains(payload, strings.ToLower(needle)) {
			t.Fatalf("normalized error leaked %q: %s", needle, truncate(body))
		}
	}
}

// Every rejection path, not just upstream failures, has to stay free of
// credentials and internal topology.
func TestRejectionsNeverLeakSecretsOrTopology(t *testing.T) {
	harness := ready(t)
	tests := []struct {
		name  string
		token string
		path  string
	}{
		{name: "unauthenticated", path: "/api/v1/me"},
		{name: "expired", token: TokenExpired, path: "/api/v1/me"},
		{name: "forbidden business read", token: TokenCustomer, path: "/api/v1/clients"},
		{name: "forbidden audit", token: TokenDeveloper, path: "/api/v1/audit"},
		{name: "unknown global id", token: TokenAdmin, path: "/api/v1/global-ids/CL-999999"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := harness.API(t, http.MethodGet, test.path, test.token, nil)
			if response.Status < 400 {
				t.Fatalf("status = %d, want a rejection", response.Status)
			}
			assertNoSecretsOrTopology(t, response.Body)
		})
	}
}

// --- Events -----------------------------------------------------------------

func TestServiceEventIsPublishedToNATS(t *testing.T) {
	harness := ready(t)
	subscriber, err := SubscribeNATS(harness.NATSAddr, "bsystem.events.>")
	if err != nil {
		t.Fatalf("subscribe to NATS: %v", err)
	}
	defer subscriber.Close()

	requestID := fmt.Sprintf("e2e-event-%d", time.Now().UnixNano())
	entityID := fmt.Sprintf("TSK-%d", time.Now().Unix()%1000000)
	published := harness.Request(t, http.MethodPost, harness.Core+"/api/service/v1/events", TokenService,
		map[string]any{
			"event":     "test.failed",
			"source":    "bsystem-e2e",
			"entity_id": entityID,
			"severity":  "error",
			"data":      map[string]any{"suite": "e2e"},
		},
		map[string]string{"X-Request-ID": requestID})
	if published.Status != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 (body: %s)", published.Status, truncate(published.Body))
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		message, err := subscriber.Next(time.Until(deadline))
		if err != nil {
			t.Fatalf("await published event: %v", err)
		}
		var envelope struct {
			Event      string         `json:"event"`
			Source     string         `json:"source"`
			ActorID    string         `json:"actor_id"`
			EntityID   string         `json:"entity_id"`
			Severity   string         `json:"severity"`
			RequestID  string         `json:"request_id"`
			OccurredAt time.Time      `json:"occurred_at"`
			Data       map[string]any `json:"data"`
		}
		if err := json.Unmarshal(message.Payload, &envelope); err != nil {
			continue
		}
		if envelope.RequestID != requestID {
			// Another scenario's event; keep waiting for ours.
			continue
		}
		if message.Subject != "bsystem.events.test.failed" {
			t.Fatalf("subject = %q, want bsystem.events.test.failed", message.Subject)
		}
		if envelope.Event != "test.failed" || envelope.Source != "bsystem-e2e" {
			t.Fatalf("unexpected envelope: %+v", envelope)
		}
		if !strings.HasPrefix(envelope.ActorID, "SVC-") {
			t.Fatalf("actor_id = %q, want the publishing SVC-* identity", envelope.ActorID)
		}
		if envelope.EntityID != entityID || envelope.Severity != "error" {
			t.Fatalf("envelope lost fields: %+v", envelope)
		}
		if envelope.OccurredAt.IsZero() {
			t.Fatal("occurred_at must be populated")
		}
		return
	}
	t.Fatalf("no event carrying request ID %q reached NATS", requestID)
}

func TestEventPublishingIsRestrictedAndValidated(t *testing.T) {
	harness := ready(t)
	tests := []struct {
		name       string
		token      string
		body       map[string]any
		wantStatus int
	}{
		{name: "human token is refused", token: TokenAdmin, body: map[string]any{"event": "test.failed", "source": "e2e"}, wantStatus: http.StatusForbidden},
		{name: "ungrouped service is refused", token: TokenServiceUngrouped, body: map[string]any{"event": "test.failed", "source": "e2e"}, wantStatus: http.StatusForbidden},
		{name: "missing source is rejected", token: TokenService, body: map[string]any{"event": "test.failed"}, wantStatus: http.StatusBadRequest},
		{name: "event without an action is rejected", token: TokenService, body: map[string]any{"event": "test", "source": "e2e"}, wantStatus: http.StatusBadRequest},
		{name: "unsupported severity is rejected", token: TokenService, body: map[string]any{"event": "test.failed", "source": "e2e", "severity": "catastrophic"}, wantStatus: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := harness.API(t, http.MethodPost, "/api/service/v1/events", test.token, test.body)
			if response.Status != test.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", response.Status, test.wantStatus, truncate(response.Body))
			}
		})
	}
}
