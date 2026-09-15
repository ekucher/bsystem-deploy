package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ekucher/bsystem-deploy/mocks/internal/mockhttp"
)

func newHandler(t *testing.T) http.Handler {
	t.Helper()
	directory, err := NewDirectory(DefaultPrincipals())
	if err != nil {
		t.Fatalf("build directory: %v", err)
	}
	server := mockhttp.NewServer("mock-identity")
	newAPI(directory, "http://mock-identity:9000/application/o/bsystem-hub/").register(server)
	return server.Handler()
}

func get(t *testing.T, handler http.Handler, target, token string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestUserinfoResolvesEveryFixtureGroup(t *testing.T) {
	handler := newHandler(t)
	tests := []struct {
		token     string
		wantSub   string
		wantGroup string
	}{
		{token: "test-token-admin", wantSub: "mock-admin", wantGroup: "BSYSTEM-Admins"},
		{token: "test-token-manager", wantSub: "mock-manager", wantGroup: "BSYSTEM-Managers"},
		{token: "test-token-developer", wantSub: "mock-developer", wantGroup: "BSYSTEM-Developers"},
		{token: "test-token-qa", wantSub: "mock-qa", wantGroup: "BSYSTEM-QA"},
		{token: "test-token-support", wantSub: "mock-support", wantGroup: "BSYSTEM-Support"},
		{token: "test-token-devops", wantSub: "mock-devops", wantGroup: "BSYSTEM-DevOps"},
		{token: "test-token-customer", wantSub: "mock-customer", wantGroup: "BSYSTEM-Customers"},
		{token: "test-token-service", wantSub: "mock-service-core", wantGroup: "BSYSTEM-Services"},
	}
	for _, test := range tests {
		t.Run(test.token, func(t *testing.T) {
			response := get(t, handler, "/application/o/userinfo/", test.token)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.Code)
			}
			var info struct {
				Sub               string   `json:"sub"`
				PreferredUsername string   `json:"preferred_username"`
				Groups            []string `json:"groups"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &info); err != nil {
				t.Fatalf("decode userinfo: %v", err)
			}
			if info.Sub != test.wantSub {
				t.Fatalf("sub = %q, want %q", info.Sub, test.wantSub)
			}
			if info.PreferredUsername == "" {
				t.Fatal("preferred_username must be set so the core can derive a username")
			}
			if len(info.Groups) != 1 || info.Groups[0] != test.wantGroup {
				t.Fatalf("groups = %v, want [%s]", info.Groups, test.wantGroup)
			}
		})
	}
}

func TestUserinfoRejections(t *testing.T) {
	handler := newHandler(t)
	tests := []struct {
		name  string
		token string
	}{
		{name: "no token"},
		{name: "unknown token", token: "test-token-does-not-exist"},
		{name: "expired token", token: "test-token-expired"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := get(t, handler, "/application/o/userinfo/", test.token)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", response.Code)
			}
			if !strings.Contains(response.Header().Get("WWW-Authenticate"), "invalid_token") {
				t.Fatalf("WWW-Authenticate = %q", response.Header().Get("WWW-Authenticate"))
			}
		})
	}
}

// A principal with no groups must still authenticate: the core, not the
// identity provider, is what denies access when no role maps.
func TestUserinfoAllowsPrincipalWithoutGroups(t *testing.T) {
	response := get(t, newHandler(t), "/application/o/userinfo/", "test-token-no-groups")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"groups":[]`) {
		t.Fatalf("groups must serialize as an empty array, got %s", response.Body.String())
	}
}

func TestDiscoveryDocument(t *testing.T) {
	response := get(t, newHandler(t), "/application/o/bsystem-hub/.well-known/openid-configuration", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	var document struct {
		Issuer                       string   `json:"issuer"`
		UserinfoEndpoint             string   `json:"userinfo_endpoint"`
		CodeChallengeMethodsSupport  []string `json:"code_challenge_methods_supported"`
		TokenEndpointAuthMethodsSupp []string `json:"token_endpoint_auth_methods_supported"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode discovery document: %v", err)
	}
	if document.Issuer == "" || !strings.HasSuffix(document.UserinfoEndpoint, "/userinfo/") {
		t.Fatalf("unexpected discovery document: %+v", document)
	}
	// The HUB uses Authorization Code + PKCE against a public client.
	if len(document.CodeChallengeMethodsSupport) == 0 || document.CodeChallengeMethodsSupport[0] != "S256" {
		t.Fatalf("PKCE S256 must be advertised, got %v", document.CodeChallengeMethodsSupport)
	}
}

func TestPrincipalIntrospectionNeverExposesTokens(t *testing.T) {
	response := get(t, newHandler(t), "/__mock/principals", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, principal := range DefaultPrincipals() {
		if strings.Contains(body, principal.Token) {
			t.Fatalf("introspection leaked the token for %s", principal.Sub)
		}
	}
}

func TestUpsertPrincipalReconfiguresGroups(t *testing.T) {
	handler := newHandler(t)
	body := `{"token":"test-token-temp","sub":"mock-temp","preferred_username":"mock-temp","groups":["BSYSTEM-QA","BSYSTEM-Support"]}`
	request := httptest.NewRequest(http.MethodPost, "/__mock/principals", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("upsert status = %d, want 202", recorder.Code)
	}

	response := get(t, handler, "/application/o/userinfo/", "test-token-temp")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if !strings.Contains(response.Body.String(), "BSYSTEM-Support") {
		t.Fatalf("reconfigured groups missing: %s", response.Body.String())
	}
}

func TestUpsertPrincipalValidation(t *testing.T) {
	handler := newHandler(t)
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed json", body: "{"},
		{name: "missing token", body: `{"sub":"mock-x"}`},
		{name: "missing sub", body: `{"token":"test-token-x"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/__mock/principals", strings.NewReader(test.body))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", recorder.Code)
			}
		})
	}
}

func TestLoadPrincipals(t *testing.T) {
	if principals, err := LoadPrincipals(""); err != nil || len(principals) != len(DefaultPrincipals()) {
		t.Fatalf("empty path must select the built-in fixtures: %v %d", err, len(principals))
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "principals.json")
	if err := os.WriteFile(path, []byte(`[{"token":"test-token-file","sub":"mock-file","groups":["BSYSTEM-Admins"]}]`), 0o600); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
	principals, err := LoadPrincipals(path)
	if err != nil {
		t.Fatalf("load from file: %v", err)
	}
	if len(principals) != 1 || principals[0].Sub != "mock-file" {
		t.Fatalf("unexpected principals: %+v", principals)
	}

	empty := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(empty, []byte(`[]`), 0o600); err != nil {
		t.Fatalf("write empty fixture file: %v", err)
	}
	if _, err := LoadPrincipals(empty); err == nil {
		t.Fatal("an empty fixture file must be rejected")
	}
	if _, err := LoadPrincipals(filepath.Join(dir, "missing.json")); err == nil {
		t.Fatal("a missing fixture file must be rejected")
	}
}

// Fault injection is shared scaffolding, but the identity mock is where an
// unavailable IdP has to be reproducible, so assert it end to end here too.
func TestIdentityFaultInjection(t *testing.T) {
	directory, err := NewDirectory(DefaultPrincipals())
	if err != nil {
		t.Fatalf("build directory: %v", err)
	}
	server := mockhttp.NewServer("mock-identity")
	newAPI(directory, "http://mock-identity:9000/application/o/bsystem-hub/").register(server)
	if err := server.Faults.Add(mockhttp.Fault{Path: "/application/o/userinfo/", Status: http.StatusServiceUnavailable}); err != nil {
		t.Fatalf("add fault: %v", err)
	}
	if response := get(t, server.Handler(), "/application/o/userinfo/", "test-token-admin"); response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", response.Code)
	}
}
