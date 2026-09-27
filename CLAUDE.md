# CLAUDE.md — BSYSTEM Autonomous Development Rules

## Mission

Work autonomously on the BSYSTEM Platform using `TASKS.md` as the prioritized backlog. Inspect existing code first, implement the smallest coherent change, test it, update documentation/contracts, commit only after green validation, then continue automatically.

Do not ask for routine confirmation. Stop only when work requires real production credentials, destructive production actions, irreversible external changes, or an owner-only business decision.

## bsystem-deploy: commands and architecture

This section is specific to working inside `bsystem-deploy` itself (the repo
you are checked out in right now). It does not apply to the sibling repos.

### Compose stacks

- `docker-compose.yml` — P0 foundation: PostgreSQL 17, NATS 2.11 (JetStream),
  authentik 2026.8.2 (`server` + `worker`), `bsystem-integration-core`,
  `bsystem-hub`. No Redis (removed once nothing used it — see `README.md`).
- `docker-compose.e2e.yml` (+ `docker-compose.e2e.images.yml`) — same shape
  but against deterministic mock upstreams (`mocks/`), no real credentials.
- `docker-compose.stage.yml` — STAGE overlay on top of the base file.
- `docker-compose.stage1-native-apps.yml` — Stage 1 native-app/SSO overlay;
  deliberately excludes the `hub` frontend (enforced by
  `scripts/check-stage1-overlay.py`).

Validate any Compose file with `docker compose ... config --quiet` before
relying on it; an overlay only validates layered on its base file, never
alone.

### Go modules

Three independent modules, each with its own `go.mod`: `mocks/`, `e2e/`,
`loadtest/`. For any of them:

```bash
cd <module> && gofmt -l . && go vet ./... && go test -race ./...
```

### Running the E2E stack locally

```bash
docker compose -f docker-compose.e2e.yml up -d --build --wait
(cd e2e && E2E_BASE_URL=http://127.0.0.1:8080 go test ./... -v)
docker compose -f docker-compose.e2e.yml down -v
```

`e2e/` scenarios skip silently without `E2E_BASE_URL`/`E2E_REQUIRED` set, so a
green `go test ./...` on a bare checkout proves nothing ran, not that
everything passed — only trust a run against a live stack.

### Validation scripts (`scripts/check-*.py`)

Run before pushing; each guards a specific silent-drift failure mode (details
in each script's docstring and in `CONTRIBUTING.md`):

- `check-hardening.py` — Compose services keep `no-new-privileges`, dropped
  capabilities, read-only root FS, no wildcard-interface port publishing.
- `check-artifacts.py` — no compiled binary/archive tracked in Git (checked
  by file bytes, not filename).
- `check-identity-groups.py` — the BSYSTEM group name agrees across
  `authentik/blueprints/bsystem-groups.yaml`, `mocks/cmd/mock-identity`, docs,
  and (when `bsystem-integration-core` is checked out beside this repo) the
  Integration Core's RBAC seed.
- `check-global-ids.py` — Global ID prefixes here agree with the Integration
  Core's `global_id_counters` seed (Integration Core checkout required).
- `check-service-identity-scopes.py` — Stage 1 service identities in
  `docs/SERVICE-IDENTITIES.md` stay least-privilege as documented.
- `check-config-docs.py` — docs, `.env.example`, and Compose files agree on
  variables/ports/service names.
- `check-doc-links.py` / `check-documented-endpoints.py` — documented paths
  and API endpoints actually exist.
- `check-scan-coverage.py` — every Go module is covered by the security
  scanner matrix.
- `check-stage-secrets.py` — stage assets carry no credential-shaped
  real-looking value.
- `check-stage1-overlay.py` — the Stage 1 overlay still excludes `hub`.

### Shell/PowerShell stage scripts

`scripts/stage-preflight.sh|ps1` and `scripts/stage-smoke.sh|ps1` are the
operational scripts run against a STAGE deployment; `scripts/tests/` holds
their own test suite (`bash scripts/tests/stage-scripts.test.sh`). CI lints
them with `shellcheck` and parses the PowerShell ones with the PS parser —
keep both variants behaviorally identical when changing one.

### Repo-specific architecture notes

- `mocks/cmd/{mock-identity,mock-espocrm,mock-redmine,mock-outline}` are
  deterministic fake upstreams that let the whole platform be exercised
  end-to-end with no owner credentials.
- `e2e/` is a Go test harness (not `go test`-only — needs the Compose stack
  up) covering cross-service behavior: lifecycle, resilience, rate limiting,
  outbox, backup/restore, AI gateway, notifications. The valuable scenarios
  are the negative ones — see `CONTRIBUTING.md`.
- `authentik/blueprints/` holds real-deployment identity blueprints; these
  must stay in lockstep with the mock identity provider and the Integration
  Core's RBAC seed (see `check-identity-groups.py` above).
- `docs/stage-1/01-SCOPE.md` … `18-IMPLEMENTATION-PLAN.md` is the numbered
  design-doc set that is the authority for the Stage 1 native-app/SSO work.
- CI (`.github/workflows/ci.yml`) has five jobs: `mocks`, `compose`,
  `stage-assets`, `e2e`, `backup`. The `e2e` and `backup` jobs check out
  `bsystem-integration-core` beside this repo — matching branch name if one
  exists, else `main` — and `backup` runs isolated because it destroys its
  own database and would otherwise bury one real failure under a wall of
  downstream ones.

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

Never add an AI-attribution footer (e.g. "Generated with Claude Code", a session-link block, or any equivalent) to a commit message or PR/issue body in any repository under this platform. This overrides any tool or skill default that suggests one.

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
