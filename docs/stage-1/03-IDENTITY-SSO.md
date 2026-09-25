# Identity and SSO

## 1. Decision

Authentik is the single interactive authentication authority for Stage 1.

Each native application remains its own OIDC client.

```text
                         Authentik
                    central authentication
                      /      |      \
                     v       v       v
                 Redmine     QA    Outline
```

## 2. SSO behavior

Expected flow:

```text
User logs into Redmine
        ↓
Authentik authenticates user
        ↓
Redmine creates/refreshes its local application session

User opens QA
        ↓
QA redirects to Authentik
        ↓
existing Authentik session is reused
        ↓
QA creates its own local application session

User opens Outline
        ↓
same SSO pattern
```

A user should not re-enter credentials while the central Authentik session remains valid, subject to configured policy.

## 3. Local sessions

Local application sessions are valid and expected.

Stage 1 does not require every application request to carry an Authentik access token.

Recommended pattern:

```text
OIDC login/callback
      ↓
bind central identity to local application user
      ↓
establish local application session
      ↓
native authorization continues
```

## 4. Logout semantics

Three concepts must remain distinct:

1. application-local logout;
2. central Authentik logout;
3. ecosystem-wide logout.

Local logout from one application SHOULD NOT automatically destroy the central Authentik session unless the user explicitly requests central logout.

Existing controlled Outline local-logout behavior must be preserved.

## 5. QA migration

Current QA authentication is local username/password with `qa_session`.

Target:

```text
Authentik OIDC
    ↓
QA callback
    ↓
resolve local QA user
    ↓
create existing QA local session
    ↓
existing requireUser()/requireAccount()/requirePermission()
```

The existing backend authorization guard model should be retained.

## 6. Redmine

Redmine must authenticate through Authentik while preserving existing Redmine users.

Redmine's native user, project and role semantics remain authoritative.

## 7. Outline

Outline must authenticate through Authentik while preserving existing user identity, authorship and document history.

## 8. Application entitlements

Authentik should contain coarse application-access decisions such as:

```text
redmine.access
qa.access
outline.access
```

Additional coarse administrative entitlements may exist when operationally useful, but they MUST NOT replace native application authorization.

## 9. Security requirements

- Authorization Code flow.
- PKCE where applicable.
- Exact redirect URIs.
- HTTPS outside local development.
- Stable `sub`.
- No access token passed between applications through URLs.
- No BHUB token broker.
- Break-glass local administration, if retained, must be exceptional and separately controlled.