// Command mock-identity serves a deterministic stand-in for the authentik
// OIDC userinfo endpoint so that BSYSTEM can be validated end to end without
// a real identity provider.
//
// Every token it accepts is a documented test-only placeholder. The mock must
// never be deployed alongside production data.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/ekucher/bsystem-deploy/mocks/internal/mockhttp"
)

type api struct {
	directory *Directory
	issuer    string
}

func (a *api) userinfo(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		unauthorized(w, "missing bearer token")
		return
	}
	principal, ok := a.directory.Lookup(strings.TrimPrefix(header, "Bearer "))
	if !ok {
		unauthorized(w, "token is not recognized")
		return
	}
	if principal.Expired {
		unauthorized(w, "token expired")
		return
	}
	groups := principal.Groups
	if groups == nil {
		groups = []string{}
	}
	mockhttp.WriteJSON(w, http.StatusOK, map[string]any{
		"sub":                principal.Sub,
		"email":              principal.Email,
		"name":               principal.Name,
		"preferred_username": principal.PreferredUsername,
		"groups":             groups,
	})
}

func unauthorized(w http.ResponseWriter, description string) {
	w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
	mockhttp.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_token", "error_description": description})
}

func (a *api) discovery(w http.ResponseWriter, _ *http.Request) {
	mockhttp.WriteJSON(w, http.StatusOK, map[string]any{
		"issuer":                                a.issuer,
		"userinfo_endpoint":                     a.issuer + "userinfo/",
		"authorization_endpoint":                a.issuer + "authorize/",
		"token_endpoint":                        a.issuer + "token/",
		"jwks_uri":                              a.issuer + "jwks/",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"scopes_supported":                      []string{"openid", "profile", "email", "groups"},
		"token_endpoint_auth_methods_supported": []string{"none"},
	})
}

// listPrincipals exposes the configured identities without their tokens so
// that test harnesses can introspect group wiring without reading credentials
// out of the mock.
func (a *api) listPrincipals(w http.ResponseWriter, _ *http.Request) {
	mockhttp.WriteJSON(w, http.StatusOK, a.directory.Public())
}

// upsertPrincipal lets a test reconfigure groups at runtime, which keeps the
// authorization matrix scenarios from needing a mock restart per role.
func (a *api) upsertPrincipal(w http.ResponseWriter, r *http.Request) {
	var principal Principal
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&principal); err != nil {
		mockhttp.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := a.directory.Put(principal); err != nil {
		mockhttp.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	mockhttp.WriteJSON(w, http.StatusAccepted, PublicPrincipal{Sub: principal.Sub, Name: principal.Name, PreferredUsername: principal.PreferredUsername, Groups: principal.Groups, Expired: principal.Expired})
}

func newAPI(directory *Directory, issuer string) *api {
	if !strings.HasSuffix(issuer, "/") {
		issuer += "/"
	}
	return &api{directory: directory, issuer: issuer}
}

func (a *api) register(server *mockhttp.Server) {
	server.Handle("GET /application/o/userinfo/", a.userinfo)
	server.Handle("GET /application/o/bsystem-hub/.well-known/openid-configuration", a.discovery)
	server.Handle("GET /__mock/principals", a.listPrincipals)
	server.Handle("POST /__mock/principals", a.upsertPrincipal)
}

func main() {
	mockhttp.MaybeHealthCheck(":9000")
	principals, err := LoadPrincipals(os.Getenv("IDENTITY_FIXTURES"))
	if err != nil {
		log.Fatalf("load identity fixtures: %v", err)
	}
	directory, err := NewDirectory(principals)
	if err != nil {
		log.Fatalf("build identity directory: %v", err)
	}
	issuer := mockhttp.Secret("IDENTITY_ISSUER", "http://mock-identity:9000/application/o/bsystem-hub/")
	server := mockhttp.NewServer("mock-identity")
	newAPI(directory, issuer).register(server)
	log.Fatal(server.ListenAndServe("HTTP_ADDR", ":9000"))
}
