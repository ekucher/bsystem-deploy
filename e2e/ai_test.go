package e2e

import (
	"net/http"
	"strings"
	"testing"
)

type aiAnswer struct {
	Answer         string   `json:"answer"`
	Provider       string   `json:"provider"`
	Model          string   `json:"model"`
	Classification string   `json:"classification"`
	UsedSources    []string `json:"used_sources"`
	RequestID      string   `json:"request_id"`
}

type aiAuditRecord struct {
	ID               int64          `json:"id"`
	ActorID          string         `json:"actor_id"`
	Provider         string         `json:"provider"`
	Model            string         `json:"model"`
	RequestedSources []string       `json:"requested_sources"`
	EntityIDs        []string       `json:"entity_ids"`
	Classification   map[string]any `json:"classification"`
	RequestID        string         `json:"request_id"`
	Result           string         `json:"result"`
	PromptBytes      int            `json:"prompt_bytes"`
	AnswerBytes      int            `json:"answer_bytes"`
}

func ask(t *testing.T, h *Harness, token string, body map[string]any) Response {
	t.Helper()
	return h.API(t, http.MethodPost, "/api/v1/ai/ask", token, body)
}

func aiAudit(t *testing.T, h *Harness) []aiAuditRecord {
	t.Helper()
	response := h.API(t, http.MethodGet, "/api/v1/ai/audit?limit=200", TokenAdmin, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("ai audit: status = %d, body = %s", response.Status, truncate(response.Body))
	}
	var records []aiAuditRecord
	response.JSON(t, &records)
	return records
}

// The gateway must be usable with no model configured, so that its
// authorization, classification and audit behaviour is exercised in every
// deployment rather than only where somebody has paid for one.
func TestTheGatewayAnswersWithTheFakeProvider(t *testing.T) {
	h := ready(t)

	response := ask(t, h, TokenAdmin, map[string]any{"question": "What is BSYSTEM?"})
	if response.Status != http.StatusOK {
		t.Fatalf("ask: status = %d, body = %s", response.Status, truncate(response.Body))
	}
	var answer aiAnswer
	response.JSON(t, &answer)
	if answer.Provider != "fake" || answer.Answer == "" {
		t.Errorf("answer = %+v, want the fake provider to answer", answer)
	}
	// A question with no platform context discloses nothing of the
	// platform's, so it is PUBLIC.
	if answer.Classification != "PUBLIC" {
		t.Errorf("classification = %q, want PUBLIC for a question with no context", answer.Classification)
	}
}

// ai.query is granted to no role, so only an administrator reaches the
// gateway until an owner grants it.
func TestOnlyTheWildcardReachesTheGatewayByDefault(t *testing.T) {
	h := ready(t)

	for name, c := range map[string]struct {
		token string
		want  int
	}{
		"an administrator": {TokenAdmin, http.StatusOK},
		"a manager":        {TokenManager, http.StatusForbidden},
		"a developer":      {TokenDeveloper, http.StatusForbidden},
		"support":          {TokenSupport, http.StatusForbidden},
		"a customer":       {TokenCustomer, http.StatusForbidden},
		"no groups":        {TokenNoGroups, http.StatusForbidden},
	} {
		t.Run(name, func(t *testing.T) {
			response := ask(t, h, c.token, map[string]any{"question": "hello"})
			if response.Status != c.want {
				t.Errorf("status = %d, want %d (body: %s)", response.Status, c.want, truncate(response.Body))
			}
		})
	}
	if response := ask(t, h, "", map[string]any{"question": "hello"}); response.Status != http.StatusUnauthorized {
		t.Errorf("no token: status = %d, want 401", response.Status)
	}
	// The audit endpoint is administrator-only too.
	if response := h.API(t, http.MethodGet, "/api/v1/ai/audit", TokenSupport, nil); response.Status != http.StatusForbidden {
		t.Errorf("audit as support: status = %d, want 403", response.Status)
	}
}

// Named context must be authorized as though the caller had read it directly.
func TestNamedContextIsAuthorizedLikeADirectRead(t *testing.T) {
	h := ready(t)
	bravo := newReporter(t, h)
	host := bravo.register(t, marker("ai-host"), "ai-host", "prod", nil)

	response := ask(t, h, TokenAdmin, map[string]any{
		"question": "What is the state of this server?",
		"sources":  []map[string]any{{"entity_type": "server", "global_id": host.ID}},
	})
	if response.Status != http.StatusOK {
		t.Fatalf("ask with context: status = %d, body = %s", response.Status, truncate(response.Body))
	}
	var answer aiAnswer
	response.JSON(t, &answer)
	if len(answer.UsedSources) != 1 || answer.UsedSources[0] != host.ID {
		t.Errorf("used_sources = %v, want the named server", answer.UsedSources)
	}
	if answer.Classification != "INTERNAL" {
		t.Errorf("classification = %q, want INTERNAL for a server", answer.Classification)
	}
}

// A source the caller cannot resolve and one they may not use are the same
// refusal, so the gateway cannot be used to probe for identifiers that an
// endpoint would not disclose.
func TestUnusableSourcesAreRefusedIdentically(t *testing.T) {
	h := ready(t)

	unknown := ask(t, h, TokenAdmin, map[string]any{
		"question": "tell me",
		"sources":  []map[string]any{{"entity_type": "server", "global_id": "SRV-999999"}},
	})
	if unknown.Status != http.StatusForbidden {
		t.Errorf("an unknown source: status = %d, want 403", unknown.Status)
	}
	wrongType := ask(t, h, TokenAdmin, map[string]any{
		"question": "tell me",
		"sources":  []map[string]any{{"entity_type": "server", "global_id": "CL-000001"}},
	})
	if wrongType.Status != http.StatusForbidden {
		t.Errorf("a source of the wrong type: status = %d, want 403", wrongType.Status)
	}

	// An entity type the gateway does not accept is a different answer: it is
	// the request that is malformed, not the caller's access.
	badType := ask(t, h, TokenAdmin, map[string]any{
		"question": "tell me",
		"sources":  []map[string]any{{"entity_type": "user", "global_id": "USR-000001"}},
	})
	if badType.Status != http.StatusBadRequest {
		t.Errorf("an unaccepted entity type: status = %d, want 400", badType.Status)
	}
	assertNoSecretsOrTopology(t, badType.Body)
}

// The bounds protect the platform from cost, latency and a prompt large
// enough to be an exfiltration channel.
func TestTheGatewayIsBounded(t *testing.T) {
	h := ready(t)

	oversized := ask(t, h, TokenAdmin, map[string]any{"question": strings.Repeat("a", 4001)})
	if oversized.Status != http.StatusBadRequest {
		t.Errorf("an oversized question: status = %d, want 400", oversized.Status)
	}

	sources := make([]map[string]any, 0, 21)
	for i := 0; i < 21; i++ {
		sources = append(sources, map[string]any{"entity_type": "server", "global_id": "SRV-000001"})
	}
	tooMany := ask(t, h, TokenAdmin, map[string]any{"question": "tell me", "sources": sources})
	if tooMany.Status != http.StatusBadRequest {
		t.Errorf("too many sources: status = %d, want 400", tooMany.Status)
	}

	empty := ask(t, h, TokenAdmin, map[string]any{"question": "   "})
	if empty.Status != http.StatusBadRequest {
		t.Errorf("an empty question: status = %d, want 400", empty.Status)
	}
}

// Every call is audited, answered or refused, and the audit carries no
// prompt: an audit trail is read by more people than the request was.
func TestEveryGatewayCallIsAuditedWithoutItsPrompt(t *testing.T) {
	h := ready(t)
	bravo := newReporter(t, h)
	host := bravo.register(t, marker("ai-audited"), "ai-audited-host", "prod", nil)

	question := "Tell me about " + marker("audited")
	correlationID := marker("ai-correlation")
	response := h.Request(t, http.MethodPost, h.Core+"/api/v1/ai/ask", TokenAdmin, map[string]any{
		"question": question,
		"sources":  []map[string]any{{"entity_type": "server", "global_id": host.ID}},
	}, map[string]string{"X-Request-ID": correlationID})
	if response.Status != http.StatusOK {
		t.Fatalf("ask: status = %d, body = %s", response.Status, truncate(response.Body))
	}

	records := aiAudit(t, h)
	var found *aiAuditRecord
	for i := range records {
		if records[i].RequestID == correlationID {
			found = &records[i]
			break
		}
	}
	if found == nil {
		t.Fatal("the call was not audited")
	}
	if found.Result != "answered" || found.Provider != "fake" {
		t.Errorf("audit record = %+v", found)
	}
	if found.ActorID == "" || !strings.HasPrefix(found.ActorID, "USR-") {
		t.Errorf("actor_id = %q, want the caller's Global user ID", found.ActorID)
	}
	if len(found.EntityIDs) != 1 || found.EntityIDs[0] != host.ID {
		t.Errorf("entity_ids = %v, want the server that entered the prompt", found.EntityIDs)
	}
	if found.PromptBytes <= 0 || found.AnswerBytes <= 0 {
		t.Errorf("sizes = %d/%d, want both recorded", found.PromptBytes, found.AnswerBytes)
	}
	if found.Classification["overall"] != "INTERNAL" {
		t.Errorf("classification = %v, want INTERNAL overall", found.Classification)
	}

	// The audit must not carry the question, the answer or the context. Every
	// field is checked rather than a named one, so a field added later cannot
	// quietly start carrying content.
	body := string(response.Body)
	auditResponse := h.API(t, http.MethodGet, "/api/v1/ai/audit?limit=200", TokenAdmin, nil)
	auditBody := string(auditResponse.Body)
	for _, content := range []string{question, "ai-audited-host"} {
		if strings.Contains(auditBody, content) {
			t.Errorf("the audit trail carries %q", content)
		}
	}
	if !strings.Contains(body, "fake answer") {
		t.Errorf("the answer did not come from the fake provider: %s", truncate(response.Body))
	}
}

// A refused call must be audited too, or the audit shows only the requests
// that succeeded and a pattern of refusals is invisible.
func TestRefusedCallsAreAudited(t *testing.T) {
	h := ready(t)

	correlationID := marker("ai-refused")
	response := h.Request(t, http.MethodPost, h.Core+"/api/v1/ai/ask", TokenAdmin, map[string]any{
		"question": "tell me",
		"sources":  []map[string]any{{"entity_type": "server", "global_id": "SRV-999999"}},
	}, map[string]string{"X-Request-ID": correlationID})
	if response.Status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Status)
	}

	for _, record := range aiAudit(t, h) {
		if record.RequestID == correlationID {
			if record.Result != "refused_unauthorized" {
				t.Errorf("result = %q, want refused_unauthorized", record.Result)
			}
			if record.PromptBytes != 0 {
				t.Errorf("prompt_bytes = %d on a refused call, want 0: nothing was assembled", record.PromptBytes)
			}
			return
		}
	}
	t.Error("a refused call was not audited")
}

// A caller pasting a credential into a question is the likeliest way one
// reaches a model. The gateway is the last place to catch it, and the fake
// provider's answer reports how many fragments it saw rather than echoing
// them — so this asserts the request completes without the secret reaching
// anything that stores it.
func TestACredentialInAQuestionIsNotStored(t *testing.T) {
	h := ready(t)

	secret := "sk-live-" + strings.ReplaceAll(marker("secret"), "-", "")
	correlationID := marker("ai-secret")
	response := h.Request(t, http.MethodPost, h.Core+"/api/v1/ai/ask", TokenAdmin, map[string]any{
		"question": "Why does my key " + secret + " fail?",
	}, map[string]string{"X-Request-ID": correlationID})
	if response.Status != http.StatusOK {
		t.Fatalf("ask: status = %d, body = %s", response.Status, truncate(response.Body))
	}
	if strings.Contains(string(response.Body), secret) {
		t.Errorf("the response echoed the credential: %s", truncate(response.Body))
	}

	auditResponse := h.API(t, http.MethodGet, "/api/v1/ai/audit?limit=200", TokenAdmin, nil)
	if strings.Contains(string(auditResponse.Body), secret) {
		t.Error("the audit trail carries a credential from a question")
	}
}
