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

- [x] timestamp/level/message
- [x] request_id
- [x] route/method/status/duration
- [x] actor Global ID when safe
- [x] credential-safe redaction

## P7.2 Metrics

- [x] HTTP request count
- [x] latency
- [x] status class
- [x] bounded route labels
- [x] adapter requests/failures/latency/rate-limit/circuit
- [x] DB pool metrics
- [x] NATS status/publish metrics

## P7.3 Grafana

- [x] dashboard JSON
- [x] requests
- [x] errors
- [x] latency
- [x] DB
- [x] NATS
- [x] adapters
- [x] datasource assumptions documented

# P8 — Security automation

Priority: HIGH

- [x] govulncheck
- [x] go test -race
- [x] staticcheck
- [x] npm audit policy
- [!] ESLint where missing — blocked upstream for the HUB: `typescript-eslint`
      peers on `typescript >=4.8.4 <6.1.0` and the HUB is on TypeScript 7.
      Adding it means forcing an unsupported resolution or downgrading the
      compiler. `tsc --strict`, the tests and the axe checks run instead
- [x] Gitleaks — full history in all four repositories
- [x] Trivy filesystem scan
- [x] Docker image scan where applicable — the mock upstream image; the HUB
      and Integration Core images are covered by their repositories' Dockerfile
      misconfiguration scans
- [x] SBOM generation — CycloneDX, published as a build artifact
- [x] dependency review where available — Dependabot for npm and Actions
- [x] CodeQL where useful — Go in the Integration Core, TypeScript in the HUB
      and design system

Findings the scans produced and how they were resolved:

- `golang.org/x/text` v0.35.0 GO-2026-5970, reachable through `pgxpool.New`,
  upgraded to v0.39.0 and later v0.41.0
- `golang.org/x/crypto` v0.49.0, ten HIGH ssh advisories, upgraded to v0.55.0
- `react-router-dom` 7.9.1, two HIGH advisories on the HUB's redirect-based
  authentication path, upgraded to 7.18.4
- `vitest` 3.2.4 critical, upgraded to 5.0.1 in both Node repositories
- the mock image linked a Go 1.24 standard library carrying twelve advisories;
  both deploy modules now require go 1.26.6
- the HUB image ran nginx as root; it is now nginx-unprivileged as uid 101

Docker hardening review:

- [x] non-root — every service; the HUB moved to nginx-unprivileged and the
      mocks are scratch images running as 65532
- [x] no-new-privileges — every service in both Compose files
- [x] read-only filesystem where practical — Integration Core, HUB, mocks
- [x] tmpfs where practical — the HUB's `/tmp`
- [x] explicit networks — internal `data`/`backend` and `e2e-data` networks
- [x] bounded exposed ports — published ports bind `${BIND_ADDRESS:-127.0.0.1}`
- [x] no secrets in layers — build arguments carry only public OIDC client
      configuration; credentials arrive as runtime environment

`scripts/check-hardening.py` runs in CI and fails on a regression in any of
the above; `docker-compose.e2e.yml` is additionally started for real by the
E2E job, so a capability set that breaks a container fails the build.

Capabilities are left to the image's own entrypoint for authentik, and
postgres and redis keep the five and four capabilities their entrypoints need
to drop their own privileges. Those are documented at the services.

# P9 — Notifications

Priority: MEDIUM

- [x] persistence model — `notifications` and `notification_reads`, migration 004
- [x] recipient/severity/source/title/body/deep-link/read-state
- [x] `GET /api/v1/notifications`
- [x] mark-read endpoint — `POST /api/v1/notifications/{id}/read`
- [x] pagination — keyset on the notification id, not an offset: the store is
      append-heavy at the head, so an offset cursor would shift every later
      page on each arrival
- [x] authorization — addressed to a Global user ID or to a permission,
      resolved from RBAC at read time; a scope-confined principal reads only
      what names it
- [x] event mappings for backup/test/build/incident failures — six events;
      everything else raises nothing
- [x] HUB notification center — `/notifications`
- [x] unread badge — count from the collection envelope, polled
- [x] deep links — only for entity types the HUB has a page for

Do not invent production recipients.

None were invented. A notification is addressed either to a Global user ID the
publisher named and the platform verified, or to a permission — and who
satisfies a permission is resolved from RBAC when the notification is read, so
the platform never stores a guess about who a failed backup concerns.

The E2E suite found `test.failed` addressed to `qa.report.read`, which only the
Manager role holds: the QA team would never have seen it and nothing would have
failed. It is now addressed to `qa.testcase.read`. No RBAC grant was widened —
see `bsystem-integration-core/docs/NOTIFICATIONS.md`.

# P10 — Search

Priority: MEDIUM

- [x] normalized searchable entity — carries no upstream payload; title and
      summary are the only text
- [x] type/title/summary/source/tenant/permissions/timestamp — plus the scope
      a grant would be written against, which is what makes a confined
      principal decidable
- [x] provider abstraction — `Name`/`Index`/`Delete`/`Search`, upsert and
      idempotent delete
- [x] in-memory test provider — the default, so search is always answerable
- [x] OpenSearch adapter skeleton — shapes, bulk NDJSON framing, error
      normalization and resilience, contract-tested against a fake cluster.
      Index mappings and lifecycle are absent: they depend on cluster
      decisions nobody has made, and nothing has run against a real cluster
- [x] `GET /api/v1/search`
- [x] query/type filter/pagination — offset cursors, because relevance has no
      stable key to resume from
- [x] authorization filtering — every candidate evaluated individually against
      the document's own scope; the provider's narrowing is an optimisation,
      not the decision
- [x] index/update/delete event contracts — `POST`/`DELETE
      /api/service/v1/search/documents`, with the event-to-operation mapping
      documented for an indexer

The search envelope carries no total: the count of matches before
authorization would tell a caller how many records exist that they may not
read. See `bsystem-integration-core/docs/SEARCH.md`.

# P11 — Operations foundation

Priority: MEDIUM

- [x] Server model — no hostname or address: the platform never connects to a
      server, and an inventory of reachable addresses is internal topology
- [x] HealthEvent — `server.ok/warning/error/offline`
- [x] BackupEvent — `backup.succeeded/failed`, plus `selftest.*`
- [x] MaintenanceEvent — `maintenance.started/completed`
- [x] `SRV-*` — allocated by the platform, keyed by the reporter's identifier,
      so re-registering a host returns the same Global ID
- [x] relation to `CL-*` and optional `PR-*` — verified to resolve before
      anything is stored; ambiguous ownership is refused
- [x] `/api/v1/servers`
- [x] `/api/v1/servers/{id}` — authorized against the owning client when there
      is one, and refused before the lookup so it cannot enumerate hosts
- [x] `/api/v1/operations/events`
- [x] mock provider — the E2E `reporter`, which drives the contract from a
      reporter's side and depends on nothing BRAVO-specific
- [x] BRAVO event contracts — the platform's half: what BRAVO must call, with
      what vocabulary, as which identity. See the blocked item below for the
      other direction.

A backup or self-test outcome deliberately does not move the server's status:
it reports on a service the server runs, not on whether the server is up.
Recording it as an error would put a healthy machine on a dashboard as broken.
See `bsystem-integration-core/docs/OPERATIONS-MODULE.md`.

- [!] BRAVO inbound adapter — the platform reading an inventory or a history
      *out of* BRAVO

```text
BLOCKED:
Task: BRAVO inbound adapter (platform pulls inventory and history from BRAVO)
Repository: ekucher/bsystem-integration-core
Reason: BRAVO's own API is not documented in any of the four repositories. An
  adapter written against a guessed API would be a fiction that compiles: it
  would pass its own contract tests, because the tests would be written
  against the same guess.
What is required from owner: BRAVO's API documentation — base URL shape,
  authentication scheme, the inventory and event-history endpoints, and their
  payloads. A sample response for each is enough to start.
Safe work already completed: the push direction is done and green. BRAVO can
  integrate today by registering hosts and reporting events against the
  platform's machine API, which needs nothing from BRAVO's own API. That is
  also the direction the architecture prefers — a platform that polled
  infrastructure would be a second monitoring system disagreeing with the one
  on call.
Related commit/PR: ekucher/bsystem-integration-core#3, ekucher/bsystem-deploy#2
```

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

- [x] Incident
- [x] Request — one table with the incident: they share a lifecycle, a
      severity and an audience
- [x] SLA state — the mechanism, not the targets; see the blocked item below
- [x] Severity — `low`/`medium`/`high`/`critical`, required and never
      defaulted, and deliberately not the event severity scale
- [x] Status — a closed graph; `closed` is terminal, `resolved` may reopen
- [x] relations to CL/SRV/PR/TSK/BUG/DOC — verified to resolve before storage
- [x] additive migration — `007_support.sql`
- [x] repository layer
- [x] audit — creation and update
- [x] list/detail/create/update API — the detail endpoints refuse before they
      read, so a refused caller cannot tell an existing record from a missing
      one
- [x] tests — unit, contract and end to end
- [x] incident.created/updated/resolved events — only creation notifies; the
      title travels in the event, the summary never does

See `bsystem-integration-core/docs/SUPPORT.md`.

- [!] SLA targets — the actual response and resolution times per severity

```text
BLOCKED:
Task: Fill support_sla_policies with the business's response and resolution
  targets for each severity
Repository: ekucher/bsystem-integration-core
Reason: What the business promises a customer, and what it owes when it
  misses, is a commercial commitment. CLAUDE.md lists final SLA policy among
  the stop conditions. Plausible-looking defaults committed here would appear
  in front of customers as a promise nobody made, and would be believed
  precisely because they would sit in the same field a real one does.
What is required from owner: response and resolution minutes for low, medium,
  high and critical. Any subset works — a severity with no row simply reports
  its SLA state as unset.
Safe work already completed: the whole mechanism is built and tested. Due
  dates, the at-risk threshold, breach detection, the rule that a resolved
  record's state stops moving, and the rule that the response target stops
  binding once met are all covered by unit tests. The policy table is seeded
  empty on purpose, and with no policy the platform reports "unset" rather
  than "on track" — a distinction that exists so the platform never claims a
  promise is being kept when none was made.
Related commit/PR: ekucher/bsystem-integration-core#3, ekucher/bsystem-deploy#2
```

# P13 — AI Gateway skeleton

Priority: MEDIUM

No real LLM credentials required.

## P13.1 Providers

- [-] generic provider interface
- [-] fake provider
- [-] Ollama client
- [-] OpenAI client with env-only config
- [-] no committed secrets

## P13.2 Authorization-aware context

- [-] actor identity
- [-] Integration Core authorization
- [-] explicit sources
- [-] explicit Global IDs
- [-] no unrestricted SQL

## P13.3 Classification and redaction

- [-] PUBLIC/INTERNAL/CONFIDENTIAL/SECRET/CREDENTIAL policy
- [-] block `CREDENTIAL`
- [-] Authorization/token/password/API-key redaction
- [-] tests proving credentials never reach provider payload

## P13.4 AI audit/API

- [-] actor
- [-] provider/model
- [-] requested sources
- [-] entities
- [-] classification summary
- [-] request ID/result
- [-] no full sensitive prompts by default
- [-] internal AI endpoint
- [-] fake-provider E2E
- [-] timeout/cancellation
- [-] request size limits

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
