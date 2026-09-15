# TASKS.md — BSYSTEM Autonomous Backlog

## Execution rules

Execute this backlog autonomously in priority order.

After each coherent green task:
1. update this file;
2. mark completed items;
3. commit the repository change;
4. continue automatically.

Legend:

```text
[ ] TODO
[-] IN PROGRESS
[x] DONE
[!] BLOCKED
```

# P1 — Autonomous E2E test environment

Priority: CRITICAL

## P1.1 Mock identity service

- [x] create minimal fake OIDC/UserInfo service
- [x] valid admin/developer/support/customer/service identities
- [x] configurable groups
- [x] invalid/expired token cases
- [x] test-only secrets only
- [x] documentation

## P1.2 Mock EspoCRM

- [x] Accounts
- [x] Contacts
- [x] pagination
- [x] 401/403/404/429/500
- [x] timeout case

## P1.3 Mock Redmine

- [x] Projects
- [x] Issues
- [x] pagination
- [x] 401/404/429/500
- [x] timeout case

## P1.4 Mock Outline

- [x] documents.list
- [x] documents.info
- [x] search when adapter supports it
- [x] 401/429/500
- [x] timeout case

## P1.5 E2E Compose

- [x] PostgreSQL
- [x] NATS
- [x] fake identity
- [x] mock EspoCRM
- [x] mock Redmine
- [x] mock Outline
- [x] Integration Core
- [ ] HUB when practical — deferred: the scenarios assert the normalized API
      contract the HUB consumes, which is what the HUB builds against. Adding the
      static SPA to the stack would not exercise anything the contract does not.
- [x] isolated network
- [x] deterministic healthchecks

## P1.6 Automated E2E scenarios

- [x] stable `USR-*`
- [x] stable `SVC-*`
- [x] `/api/v1/me`
- [x] module filtering
- [x] client/project/issue/document normalization
- [x] Global ID stability
- [x] audit creation
- [x] event publishing
- [x] customer denial for unscoped documents
- [x] upstream error normalization
- [x] request ID propagation

# P2 — API contract completion

Priority: HIGH

## P2.1 OpenAPI

- [x] inventory all active endpoints
- [x] sync `docs/openapi.yaml`
- [x] common Error schema
- [x] Me/Module/Client/Contact/Project/Issue/Document schemas
- [x] pagination schema — the bounded `limit` parameter is documented as the
      current contract; a cursor/offset envelope is P3.4
- [x] service identity schemas
- [x] RBAC admin schemas
- [x] auth requirements
- [x] examples

## P2.2 OpenAPI CI

- [x] add Spectral or Redocly
- [x] fail CI on invalid contract

## P2.3 Detail endpoints

- [x] `GET /api/v1/clients/{id}`
- [x] `GET /api/v1/contacts/{id}`
- [x] `GET /api/v1/projects/{id}`
- [x] `GET /api/v1/issues/{id}`
- [x] `GET /api/v1/documents/{id}`
- [x] Global ID resolution
- [x] permission + scope enforcement
- [x] normalized 404
- [x] tests
- [x] OpenAPI update

# P3 — Adapter hardening

Priority: HIGH

## P3.1 Shared HTTP behavior

- [x] bounded timeouts
- [x] context cancellation
- [x] timeout/auth/rate-limit/upstream normalized errors
- [x] response body size limits
- [x] no credential logging

## P3.2 Retry policy

- [x] safe/idempotent requests only
- [x] exponential backoff
- [x] jitter
- [x] bounded attempts
- [x] Retry-After support
- [x] tests

## P3.3 Circuit breaker

- [x] closed/open/half-open states
- [x] configurable thresholds
- [x] metrics
- [x] health integration
- [x] deterministic tests

## P3.4 Pagination

- [x] EspoCRM Accounts/Contacts
- [x] Redmine Projects/Issues
- [x] Outline Documents
- [x] normalized `limit`
- [x] cursor/offset abstraction — opaque cursors, offset-encoded today; only
      the codec changes when an upstream gains real cursors
- [x] bounded max page size

# P4 — Tenant isolation and authorization

Priority: CRITICAL

## P4.1 Scope evaluator

- [x] reusable evaluator
- [x] global/tenant/client/project/resource scopes
- [x] deny-by-default
- [x] remove duplicated route auth logic

## P4.2 Authorization matrix tests

Roles:

- [x] Administrator
- [x] Manager
- [x] Developer
- [x] QA
- [x] Support
- [x] DevOps
- [x] Customer
- [ ] Read Only — no such role exists. An unmapped principal resolves to no
      roles, permissions or modules and is denied everywhere; that case is
      covered instead
- [x] Service Core

Resources:

- [x] clients
- [x] contacts
- [x] projects
- [x] issues
- [x] documents
- [x] RBAC admin
- [x] service API

## P4.3 IDOR tests

- [x] Client Global ID manipulation
- [x] Project Global ID manipulation
- [x] Document Global ID manipulation
- [x] source ID manipulation if exposed
- [x] cross-tenant denial
- [x] missing mapping cannot broaden access

## P4.4 Customer API safety

- [x] no unscoped internal data for Customer
- [x] customer-safe API only if mapping exists
- [x] otherwise preserve block and document owner decision —
      `bsystem-integration-core/docs/adr/ADR-005-customer-isolation-boundary.md`

# P5 — HUB application foundation

Priority: HIGH

## P5.1 Routing

- [x] React Router
- [x] `/`
- [x] `/profile`
- [x] `/clients`
- [x] `/clients/:id`
- [x] `/projects`
- [x] `/projects/:id`
- [x] `/issues`
- [x] `/documents`
- [x] `/403`
- [x] `/404`

## P5.2 API client

- [x] centralized API client
- [x] Bearer token handling
- [x] request ID support
- [x] normalized error parsing
- [x] 401/403 handling
- [x] cancellation

## P5.3 Dashboard and pages

- [x] current user summary
- [x] module cards
- [ ] integration health summary — adapter health is on the machine API
      (`/api/service/v1/adapters/health`), which a human token cannot reach.
      Surfacing it needs a human-API endpoint first; raised as a gap, not done
      by widening the service boundary
- [x] Clients list/detail
- [x] Projects list/detail
- [x] Issues list
- [x] Documents list
- [x] loading/empty/error states

## P5.4 Accessibility

- [x] keyboard navigation
- [x] visible focus
- [x] semantic headings
- [x] labels
- [x] automated accessibility checks — axe over every page; verified to fail
      on an injected violation. Contrast is not checkable in jsdom

# P6 — Design System

Priority: MEDIUM-HIGH

## P6.1 Components

- [x] Input
- [x] Textarea
- [x] Select
- [x] Table
- [x] Dialog
- [x] Alert
- [x] Spinner
- [x] Skeleton
- [x] Tabs
- [x] Breadcrumbs
- [x] Dropdown
- [x] Pagination
- [x] StatusBadge

## P6.2 Theme/accessibility

- [x] light semantic tokens
- [x] dark semantic tokens
- [x] keyboard support
- [x] focus states
- [x] ARIA/tests

## P6.3 Versioned distribution prep

- [x] changelog strategy
- [x] Changesets or equivalent
- [x] GitHub Packages/private npm documentation
- [x] package ready for versioned publishing

- [!] publishing itself

```text
BLOCKED:
Task: Publish @bsystem/design-system to a registry
Repository: ekucher/bsystem-design-system
Reason: Three owner decisions, none of which this repository should make on
  the owner's behalf: whether the package is published at all (it is still
  `private: true`, which must be lifted deliberately rather than as a side
  effect of a tooling change); whether `@bsystem` goes to GitHub Packages or
  to a private registry; and who may publish, which is what a
  `write:packages` token grants.
What is required from owner: a decision on the registry, a publishing token
  stored as a repository secret, and approval to lift `private: true`.
Safe work already completed: Changesets configured with a bump policy,
  CHANGELOG.md, package metadata and `publishConfig` for GitHub Packages,
  test sources excluded from the build output, consumer `.npmrc` and
  dependency instructions, and `docs/RELEASING.md` recording exactly what
  remains. Publishing is one configuration step, not a project.
Related commit/PR: ekucher/bsystem-design-system#3
```

# P7 — Observability

Priority: HIGH

## P7.1 Structured JSON logs

- [ ] timestamp/level/message
- [ ] request_id
- [ ] route/method/status/duration
- [ ] actor Global ID when safe
- [ ] credential-safe redaction

## P7.2 Metrics

- [ ] HTTP request count
- [ ] latency
- [ ] status class
- [ ] bounded route labels
- [ ] adapter requests/failures/latency/rate-limit/circuit
- [ ] DB pool metrics
- [ ] NATS status/publish metrics

## P7.3 Grafana

- [ ] dashboard JSON
- [ ] requests
- [ ] errors
- [ ] latency
- [ ] DB
- [ ] NATS
- [ ] adapters
- [ ] datasource assumptions documented

# P8 — Security automation

Priority: HIGH

- [ ] govulncheck
- [ ] go test -race
- [ ] staticcheck
- [ ] npm audit policy
- [!] ESLint where missing — blocked upstream for the HUB: `typescript-eslint`
      peers on `typescript >=4.8.4 <6.1.0` and the HUB is on TypeScript 7.
      Adding it means forcing an unsupported resolution or downgrading the
      compiler. `tsc --strict`, the tests and the axe checks run instead
- [ ] Gitleaks
- [ ] Trivy filesystem scan
- [ ] Docker image scan where applicable
- [ ] SBOM generation
- [ ] dependency review where available
- [ ] CodeQL where useful

Docker hardening review:

- [ ] non-root
- [ ] no-new-privileges
- [ ] read-only filesystem where practical
- [ ] tmpfs where practical
- [ ] explicit networks
- [ ] bounded exposed ports
- [ ] no secrets in layers

# P9 — Notifications

Priority: MEDIUM

- [ ] persistence model
- [ ] recipient/severity/source/title/body/deep-link/read-state
- [ ] `GET /api/v1/notifications`
- [ ] mark-read endpoint
- [ ] pagination
- [ ] authorization
- [ ] event mappings for backup/test/build/incident failures
- [ ] HUB notification center
- [ ] unread badge
- [ ] deep links

Do not invent production recipients.

# P10 — Search

Priority: MEDIUM

- [ ] normalized searchable entity
- [ ] type/title/summary/source/tenant/permissions/timestamp
- [ ] provider abstraction
- [ ] in-memory test provider
- [ ] OpenSearch adapter skeleton
- [ ] `GET /api/v1/search`
- [ ] query/type filter/pagination
- [ ] authorization filtering
- [ ] index/update/delete event contracts

# P11 — Operations foundation

Priority: MEDIUM

- [ ] Server model
- [ ] HealthEvent
- [ ] BackupEvent
- [ ] MaintenanceEvent
- [ ] `SRV-*`
- [ ] relation to `CL-*` and optional `PR-*`
- [ ] `/api/v1/servers`
- [ ] `/api/v1/servers/{id}`
- [ ] `/api/v1/operations/events`
- [ ] mock provider
- [ ] BRAVO event contracts

Events:

```text
backup.succeeded
backup.failed
selftest.succeeded
selftest.failed
maintenance.started
maintenance.completed
server.warning
server.error
```

# P12 — Support foundation

Priority: MEDIUM

- [ ] Incident
- [ ] Request
- [ ] SLA state
- [ ] Severity
- [ ] Status
- [ ] relations to CL/SRV/PR/TSK/BUG/DOC
- [ ] additive migration
- [ ] repository layer
- [ ] audit
- [ ] list/detail/create/update API
- [ ] tests
- [ ] incident.created/updated/resolved events

# P13 — AI Gateway skeleton

Priority: MEDIUM

No real LLM credentials required.

## P13.1 Providers

- [ ] generic provider interface
- [ ] fake provider
- [ ] Ollama client
- [ ] OpenAI client with env-only config
- [ ] no committed secrets

## P13.2 Authorization-aware context

- [ ] actor identity
- [ ] Integration Core authorization
- [ ] explicit sources
- [ ] explicit Global IDs
- [ ] no unrestricted SQL

## P13.3 Classification and redaction

- [ ] PUBLIC/INTERNAL/CONFIDENTIAL/SECRET/CREDENTIAL policy
- [ ] block `CREDENTIAL`
- [ ] Authorization/token/password/API-key redaction
- [ ] tests proving credentials never reach provider payload

## P13.4 AI audit/API

- [ ] actor
- [ ] provider/model
- [ ] requested sources
- [ ] entities
- [ ] classification summary
- [ ] request ID/result
- [ ] no full sensitive prompts by default
- [ ] internal AI endpoint
- [ ] fake-provider E2E
- [ ] timeout/cancellation
- [ ] request size limits

# P14 — Documentation and ADRs

Priority: CONTINUOUS

- [ ] API
- [ ] RBAC
- [ ] SCOPES
- [ ] GLOBAL-IDS
- [ ] EVENTS
- [ ] ADAPTERS
- [ ] OBSERVABILITY
- [ ] SECURITY
- [ ] DEPLOYMENT
- [ ] DEVELOPMENT
- [ ] AI

ADRs:

- [ ] retry/circuit policy
- [ ] pagination convention
- [ ] normalized errors
- [ ] customer isolation boundary
- [ ] Design System distribution
- [ ] search provider architecture
- [ ] AI routing policy

# P15 — Repository hygiene

Priority: MEDIUM

- [ ] `.editorconfig`
- [ ] `.gitattributes`
- [ ] `.gitignore`
- [ ] CONTRIBUTING.md
- [ ] PR template
- [ ] issue templates
- [ ] conventional commit guidance
- [ ] release policy
- [ ] changelog policy
- [ ] developer setup

Do not add CODEOWNERS unless ownership is known.

# P16 — Performance and resilience

Priority: MEDIUM

- [ ] benchmark Global ID/mapping paths
- [ ] inspect DB indexes
- [ ] inspect N+1 adapter behavior
- [ ] bounded concurrency
- [ ] graceful shutdown
- [ ] server read/write/idle timeouts
- [ ] DB pool config
- [ ] fake-upstream load-test harness
- [ ] measured baseline documentation

Do not invent performance numbers.

# BLOCKED — real environment acceptance

These must remain blocked until owner runtime/credentials are available.

- [!] real `docker compose up -d --build`
- [!] live Docker host acceptance
- [!] real authentik client/provider/MFA
- [!] real EspoCRM URL/API credential/schema validation
- [!] real Redmine URL/API credential/custom-field validation
- [!] real Outline URL/scoped key/permissions validation
- [!] authoritative customer ownership mapping
- [!] production tenant assignment
- [!] production DNS/TLS/secrets/reverse proxy
- [!] production backup/restore acceptance

Continue all non-blocked tasks even when these remain blocked.

# Final autonomous completion criteria

```text
✓ E2E mocks exist
✓ CI green
✓ normalized APIs contract-tested
✓ authorization matrix tested
✓ tenant isolation negative tests exist
✓ adapters resilient
✓ HUB has normalized business pages
✓ Design System reusable and version-ready
✓ observability implemented
✓ security automation enabled
✓ Search/Notifications foundations exist
✓ Operations/Support skeletons exist
✓ AI Gateway fake-provider path tested
✓ documentation matches implementation
```

Final report must contain:

```text
Completed
Remaining blocked
CI status
Security findings
Architecture changes
Migration changes
Required owner actions
Recommended production acceptance sequence
```
