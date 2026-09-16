package main

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"sync"
)

// Principal is a fake identity served by the mock userinfo endpoint.
//
// Token values are TEST-ONLY. They are deliberately obvious placeholders so
// that they can never be confused with a real authentik credential.
type Principal struct {
	Token             string   `json:"token"`
	Sub               string   `json:"sub"`
	Email             string   `json:"email"`
	Name              string   `json:"name"`
	PreferredUsername string   `json:"preferred_username"`
	Groups            []string `json:"groups"`
	// Expired marks a token that authenticates as a known fixture but must be
	// rejected, so that expiry handling stays covered end to end.
	Expired bool `json:"expired,omitempty"`
}

// PublicPrincipal is the credential-free projection exposed for introspection.
type PublicPrincipal struct {
	Sub               string   `json:"sub"`
	Name              string   `json:"name"`
	PreferredUsername string   `json:"preferred_username"`
	Groups            []string `json:"groups"`
	Expired           bool     `json:"expired,omitempty"`
}

// Directory holds the configured principals keyed by token.
type Directory struct {
	mu      sync.RWMutex
	byToken map[string]Principal
}

// NewDirectory returns a directory seeded with principals.
func NewDirectory(principals []Principal) (*Directory, error) {
	d := &Directory{byToken: map[string]Principal{}}
	for _, principal := range principals {
		if err := d.Put(principal); err != nil {
			return nil, err
		}
	}
	return d, nil
}

// Put inserts or replaces a principal.
func (d *Directory) Put(principal Principal) error {
	principal.Token = strings.TrimSpace(principal.Token)
	principal.Sub = strings.TrimSpace(principal.Sub)
	if principal.Token == "" {
		return errors.New("principal token is required")
	}
	if principal.Sub == "" {
		return errors.New("principal sub is required")
	}
	if principal.Groups == nil {
		principal.Groups = []string{}
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.byToken[principal.Token] = principal
	return nil
}

// Lookup resolves a bearer token to a principal.
func (d *Directory) Lookup(token string) (Principal, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	principal, ok := d.byToken[strings.TrimSpace(token)]
	return principal, ok
}

// Public returns every principal without its token, sorted by subject.
func (d *Directory) Public() []PublicPrincipal {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make([]PublicPrincipal, 0, len(d.byToken))
	for _, principal := range d.byToken {
		result = append(result, PublicPrincipal{Sub: principal.Sub, Name: principal.Name, PreferredUsername: principal.PreferredUsername, Groups: principal.Groups, Expired: principal.Expired})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Sub < result[j].Sub })
	return result
}

// DefaultPrincipals is the built-in identity fixture set. It covers one
// principal per BSYSTEM group, a service identity, a service identity that is
// missing the service group, a principal with no groups at all, and an expired
// token.
func DefaultPrincipals() []Principal {
	return []Principal{
		{Token: "test-token-admin", Sub: "mock-admin", Email: "admin@bsystem.example.invalid", Name: "Mock Administrator", PreferredUsername: "mock-admin", Groups: []string{"BSYSTEM-Admins"}},
		{Token: "test-token-manager", Sub: "mock-manager", Email: "manager@bsystem.example.invalid", Name: "Mock Manager", PreferredUsername: "mock-manager", Groups: []string{"BSYSTEM-Managers"}},
		{Token: "test-token-developer", Sub: "mock-developer", Email: "developer@bsystem.example.invalid", Name: "Mock Developer", PreferredUsername: "mock-developer", Groups: []string{"BSYSTEM-Developers"}},
		{Token: "test-token-qa", Sub: "mock-qa", Email: "qa@bsystem.example.invalid", Name: "Mock QA", PreferredUsername: "mock-qa", Groups: []string{"BSYSTEM-QA"}},
		{Token: "test-token-support", Sub: "mock-support", Email: "support@bsystem.example.invalid", Name: "Mock Support", PreferredUsername: "mock-support", Groups: []string{"BSYSTEM-Support"}},
		{Token: "test-token-devops", Sub: "mock-devops", Email: "devops@bsystem.example.invalid", Name: "Mock DevOps", PreferredUsername: "mock-devops", Groups: []string{"BSYSTEM-DevOps"}},
		{Token: "test-token-customer", Sub: "mock-customer", Email: "customer@bsystem.example.invalid", Name: "Mock Customer", PreferredUsername: "mock-customer", Groups: []string{"BSYSTEM-Customers"}},
		{Token: "test-token-service", Sub: "mock-service-core", Email: "", Name: "Mock Service Core", PreferredUsername: "svc-mock-core", Groups: []string{"BSYSTEM-Services"}},
		{Token: "test-token-service-ungrouped", Sub: "mock-service-ungrouped", Email: "", Name: "Mock Service Without Group", PreferredUsername: "svc-mock-ungrouped", Groups: []string{}},
		{Token: "test-token-no-groups", Sub: "mock-no-groups", Email: "nogroups@bsystem.example.invalid", Name: "Mock Without Groups", PreferredUsername: "mock-no-groups", Groups: []string{}},
		{Token: "test-token-expired", Sub: "mock-expired", Email: "expired@bsystem.example.invalid", Name: "Mock Expired", PreferredUsername: "mock-expired", Groups: []string{"BSYSTEM-Developers"}, Expired: true},
	}
}

// LoadPrincipals reads a JSON principal array from path. An empty path selects
// the built-in fixtures.
func LoadPrincipals(path string) ([]Principal, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return DefaultPrincipals(), nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var principals []Principal
	if err := json.Unmarshal(body, &principals); err != nil {
		return nil, err
	}
	if len(principals) == 0 {
		return nil, errors.New("principal fixture file is empty")
	}
	return principals, nil
}
