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

- [ ] `GET /api/v1/clients/{id}`
- [ ] `GET /api/v1/contacts/{id}`
- [ ] `GET /api/v1/projects/{id}`
- [ ] `GET /api/v1/issues/{id}`
- [ ] `GET /api/v1/documents/{id}`
- [ ] Global ID resolution
- [ ] permission + scope enforcement
- [ ] normalized 404
- [ ] tests
- [ ] OpenAPI update

# P3 — Adapter hardening

Priority: HIGH

## P3.1 Shared HTTP behavior

- [ ] bounded timeouts
- [ ] context cancellation
- [ ] timeout/auth/rate-limit/upstream normalized errors
- [ ] response body size limits
- [ ] no credential logging

## P3.2 Retry policy

- [ ] safe/idempotent requests only
- [ ] exponential backoff
- [ ] jitter
- [ ] bounded attempts
- [ ] Retry-After support
- [ ] tests

## P3.3 Circuit breaker

- [ ] closed/open/half-open states
- [ ] configurable thresholds
- [ ] metrics
- [ ] health integration
- [ ] deterministic tests

## P3.4 Pagination

- [ ] EspoCRM Accounts/Contacts
- [ ] Redmine Projects/Issues
- [ ] Outline Documents
- [ ] normalized `limit`
- [ ] cursor/offset abstraction
- [ ] bounded max page size

# P4 — Tenant isolation and authorization

Priority: CRITICAL

## P4.1 Scope evaluator

- [ ] reusable evaluator
- [ ] global/tenant/client/project/resource scopes
- [ ] deny-by-default
- [ ] remove duplicated route auth logic

## P4.2 Authorization matrix tests

Roles:

- [ ] Administrator
- [ ] Manager
- [ ] Developer
- [ ] QA
- [ ] Support
- [ ] DevOps
- [ ] Customer
- [ ] Read Only
- [ ] Service Core

Resources:

- [ ] clients
- [ ] contacts
- [ ] projects
- [ ] issues
- [ ] documents
- [ ] RBAC admin
- [ ] service API

## P4.3 IDOR tests

- [ ] Client Global ID manipulation
- [ ] Project Global ID manipulation
- [ ] Document Global ID manipulation
- [ ] source ID manipulation if exposed
- [ ] cross-tenant denial
- [ ] missing mapping cannot broaden access

## P4.4 Customer API safety

- [ ] no unscoped internal data for Customer
- [ ] customer-safe API only if mapping exists
- [ ] otherwise preserve block and document owner decision

# P5 — HUB application foundation

Priority: HIGH

## P5.1 Routing

- [ ] React Router
- [ ] `/`
- [ ] `/profile`
- [ ] `/clients`
- [ ] `/clients/:id`
- [ ] `/projects`
- [ ] `/projects/:id`
- [ ] `/issues`
- [ ] `/documents`
- [ ] `/403`
- [ ] `/404`

## P5.2 API client

- [ ] centralized API client
- [ ] Bearer token handling
- [ ] request ID support
- [ ] normalized error parsing
- [ ] 401/403 handling
- [ ] cancellation

## P5.3 Dashboard and pages

- [ ] current user summary
- [ ] module cards
- [ ] integration health summary
- [ ] Clients list/detail
- [ ] Projects list/detail
- [ ] Issues list
- [ ] Documents list
- [ ] loading/empty/error states

## P5.4 Accessibility

- [ ] keyboard navigation
- [ ] visible focus
- [ ] semantic headings
- [ ] labels
- [ ] automated accessibility checks

# P6 — Design System

Priority: MEDIUM-HIGH

## P6.1 Components

- [ ] Input
- [ ] Textarea
- [ ] Select
- [ ] Table
- [ ] Dialog
- [ ] Alert
- [ ] Spinner
- [ ] Skeleton
- [ ] Tabs
- [ ] Breadcrumbs
- [ ] Dropdown
- [ ] Pagination
- [ ] StatusBadge

## P6.2 Theme/accessibility

- [ ] light semantic tokens
- [ ] dark semantic tokens
- [ ] keyboard support
- [ ] focus states
- [ ] ARIA/tests

## P6.3 Versioned distribution prep

- [ ] changelog strategy
- [ ] Changesets or equivalent
- [ ] GitHub Packages/private npm documentation
- [ ] package ready for versioned publishing

Publishing itself may be `[!] BLOCKED` if credentials/config are required.

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
- [ ] ESLint where missing
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
