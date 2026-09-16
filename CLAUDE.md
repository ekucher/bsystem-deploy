# CLAUDE.md — BSYSTEM Autonomous Development Rules

## Mission

Work autonomously on the BSYSTEM Platform using `TASKS.md` as the prioritized backlog. Inspect existing code first, implement the smallest coherent change, test it, update documentation/contracts, commit only after green validation, then continue automatically.

Do not ask for routine confirmation. Stop only when work requires real production credentials, destructive production actions, irreversible external changes, or an owner-only business decision.

## Multi-repository workspace

Expected sibling repositories:

```text
BSYSTEM/
├── CLAUDE.md                 # local copy may live one level above repos
├── TASKS.md                  # local copy may live one level above repos
├── bsystem-hub/
├── bsystem-integration-core/
├── bsystem-design-system/
└── bsystem-deploy/
```

Canonical copies of this file and `TASKS.md` are versioned in `bsystem-deploy`.

Repositories:

- `ekucher/bsystem-hub`
- `ekucher/bsystem-integration-core`
- `ekucher/bsystem-design-system`
- `ekucher/bsystem-deploy`

Treat every repository as independently versioned. Do not create runtime coupling through relative filesystem imports between repositories.

## Architecture

```text
                    authentik
                       │
          ┌────────────┴────────────┐
          │                         │
      Human users              Service identities
         USR-*                       SVC-*
          │                           │
          └────────────┬──────────────┘
                       ▼
                 BSYSTEM-HUB
                       │
                       ▼
               Integration Core
                       │
       ┌───────────────┼────────────────┐
       ▼               ▼                ▼
    Adapters          NATS           PostgreSQL
       │                                │
 ┌─────┼──────┐                         ├── Global IDs
 ▼     ▼      ▼                         ├── RBAC
CRM  Redmine Outline                    ├── Scopes
                                        └── Audit
```

Future layers: QA, Development, Operations, Support, Customer Portal, Search, Notifications, AI Gateway.

## Architectural invariants

1. Source systems remain authoritative.
2. No direct cross-system database joins.
3. Global IDs are immutable.
4. authentik handles identity; BSYSTEM handles roles, permissions and scopes.
5. Backend authorization is mandatory; UI hiding is not security.
6. Deny by default.
7. Customer isolation is strict.
8. Adapters isolate upstream APIs from HUB.
9. Upstream errors must not leak secrets/internal topology.
10. AI must never bypass authorization.

### Source ownership

```text
Client      → EspoCRM
Contact     → EspoCRM
Project     → Redmine
Task/Issue  → Redmine
Document    → Outline
User        → authentik
```

Integration Core may store mappings, integration metadata, audit, RBAC metadata, scopes, event state and explicitly designed caches. It must not become a second CRM, Redmine or Wiki.

## Global IDs

```text
USR-*  User
SVC-*  Service identity
CL-*   Client
CT-*   Contact
PR-*   Project
TSK-*  Task
SRV-*  Server
INC-*  Incident
TST-*  Test Case
BUG-*  Bug
DOC-*  Document
REL-*  Release
REP-*  Repository
APP-*  Application
```

Never derive stable identity from names, hostnames, emails or mutable labels.

## Security classification

```text
PUBLIC
INTERNAL
CONFIDENTIAL
SECRET
CREDENTIAL
```

Rules:

- `CREDENTIAL` must never be sent to an LLM.
- Never commit secrets.
- Never log API keys or Authorization headers.
- Do not copy sensitive production payloads into fixtures.
- Keep `.env` ignored; use placeholders in `.env.example`.

## Technology direction

HUB: React, TypeScript, Vite, OIDC Authorization Code + PKCE.

Integration Core: Go, PostgreSQL, NATS, HTTP adapters.

Identity: authentik, OIDC/OAuth2, service identities.

Deployment: Docker and Docker Compose.

Observability: Prometheus-compatible metrics, structured logs, `X-Request-ID`, OpenTelemetry-ready architecture.

Do not replace a major technology without an ADR and a compelling reason.

## Repository responsibilities

### bsystem-hub

Owns unified UI, browser OIDC flow, navigation, dashboard/profile, normalized API consumption, loading/error/empty states.

Must not contain upstream-specific API handling or rely on frontend-only authorization.

### bsystem-integration-core

Owns normalized API, adapters, Global IDs, RBAC/scopes, service identities, events, audit, readiness/health/metrics and orchestration.

### bsystem-design-system

Owns reusable visual tokens and UI primitives. No business logic.

### bsystem-deploy

Owns Compose, environment templates, local/dev runtime, healthchecks, networks, volumes, smoke tests and cross-repo operational guidance.

## Development workflow

For every task:

1. Read relevant docs and inspect current code.
2. Reuse existing patterns.
3. Implement the smallest coherent change.
4. Add/update tests.
5. Update docs/OpenAPI when public behavior changes.
6. Run repository validation.
7. Fix regressions introduced by the change.
8. Commit only after green checks.
9. Use a conventional commit.
10. Update `TASKS.md` and continue.

## Commit policy

Prefer one coherent task per commit.

Examples:

```text
feat: add normalized client detail endpoint
test: add tenant isolation matrix
fix: normalize upstream timeout errors
docs: document adapter retry policy
ci: add OpenAPI validation
refactor: extract authorization scope evaluator
```

Never force-push or rewrite published history autonomously.

## Required validation

### Go

```bash
gofmt -w .
go vet ./...
go test ./...
```

When practical:

```bash
go test -race ./...
staticcheck ./...
govulncheck ./...
```

### React / TypeScript

```bash
npm ci
npm run build
```

When scripts exist:

```bash
npm run typecheck
npm run lint
npm test
```

### Deploy

```bash
docker compose config --quiet
```

Build affected services when Docker is available.

### Design System

```bash
npm ci
npm run check
npm run build
```

## Testing policy

Prefer table-driven Go tests, `httptest`, deterministic fake upstreams, negative authorization tests, contract tests and ephemeral PostgreSQL/NATS integration tests. Tests must not depend on production systems.

## Autonomous mock environment

Build mocks where useful:

```text
mock-authentik
mock-espocrm
mock-redmine
mock-outline
PostgreSQL
NATS
Integration Core
HUB
```

Goal: validate end-to-end behavior without owner credentials.

## Error model

Prefer stable normalized errors, e.g.:

```json
{
  "error": "upstream service unavailable",
  "code": "upstream_unavailable",
  "source": "espocrm",
  "request_id": "..."
}
```

Never expose raw SQL errors, internal hostnames, credentials, stack traces or confidential upstream payloads.

## Request correlation

Preserve or generate `X-Request-ID` across:

```text
HUB → Integration Core → Adapter → Audit/logs/events
```

## Events

Use `entity.action` naming, e.g.:

```text
client.updated
project.created
task.completed
test.failed
bug.created
build.failed
release.created
server.offline
backup.failed
incident.created
document.updated
```

NATS subject convention:

```text
bsystem.events.<event>
```

Event envelopes should support event, version, source, timestamp, entity_id, tenant_id, correlation_id, causation_id, severity and data.

## Adapter policy

Every adapter should expose consistent concepts:

```text
Info()
Health()
Capabilities()
```

HTTP adapters must have context cancellation, bounded timeouts, normalized errors, credential-safe logging and contract tests. Retries must be bounded and used only for safe/idempotent operations unless idempotency is explicitly guaranteed.

## API policy

Version APIs:

```text
/api/v1/*
/api/service/v1/*
```

Keep human and machine APIs separate. Keep `bsystem-integration-core/docs/openapi.yaml` aligned with implementation.

## Database policy

All schema changes require migrations. Prefer additive evolution. Do not autonomously drop production columns/tables or perform destructive rewrites.

## Design System policy

Do not consume an unversioned `main` branch as a production dependency. Use versioned package distribution when configured. Components must be accessible, token-driven and free of business logic.

## Documentation policy

Update docs when changing APIs, authorization semantics, Global IDs, events, adapters, deployment requirements, environment variables or security assumptions. Prefer Mermaid for architecture diagrams and ADRs for architectural decisions.

## Autonomous permissions

You may autonomously implement/refactor code, add tests/mocks/docs/CI checks, add non-destructive migrations/internal APIs, improve security/observability, build frontend pages, fix CI failures caused by your changes, and commit/push when access permits.

## Stop conditions

Stop and report instead of guessing when work requires:

- real secrets/API keys/passwords;
- production deploy/restart;
- live DNS/TLS changes;
- production authentik/EspoCRM/Redmine/Outline mutations;
- credential rotation;
- destructive DB operations;
- authoritative customer ownership mapping that is not already defined;
- final legal/SLA/billing policy decisions.

When blocked, record:

```text
BLOCKED:
Task:
Reason:
What is needed from owner:
Safe work already completed:
```

Then continue with the next independent task.

## Definition of Done

A task is done only when applicable items are satisfied:

```text
[ ] implementation complete
[ ] tests added/updated
[ ] negative/security cases covered
[ ] docs updated
[ ] OpenAPI updated if API changed
[ ] migrations added if DB changed
[ ] no secrets committed
[ ] local checks green
[ ] CI green where accessible
[ ] commit created
```

## Autonomous execution rule

Process `TASKS.md` top to bottom:

```text
READ → IMPLEMENT → TEST → FIX → DOCUMENT → VALIDATE → COMMIT → CONTINUE
```

Continue until all autonomous tasks are complete or only blocked tasks remain.
