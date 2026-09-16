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

## Delivery status

P1-P17 are delivered and merged to `main` in all four repositories, 2026-09-16.

An item marked `[x]` below is merged, not merely implemented locally. The merge
commit that landed it is listed here, and both `CI` and `Security` are green on
`main` at that commit.

Three `[ ]` items inside P1-P16 are deliberately not done rather than pending;
each carries its reason inline (E2E does not stack the static SPA, no Read Only
role exists, and the integration health summary would require widening the
machine boundary). They are not scheduled work.

P1-P16:

| Repository | Pull request | Merge commit |
| --- | --- | --- |
| `bsystem-integration-core` | #3 | `b23dc4d` |
| `bsystem-deploy` | #2 | `db80ba2` |
| `bsystem-hub` | #3 | `b534182` |
| `bsystem-design-system` | #3 | `7e19afa` |

Merged in that order: `bsystem-integration-core` before `bsystem-deploy`, because
each repository's `Autonomous E2E` job resolves the other's branch by name and
falls back to `main`.

P17, which touched two repositories:

| Repository | Pull request | Merge commit | Method |
| --- | --- | --- | --- |
| `bsystem-integration-core` | #4 | `6e9c66e` | merge |
| `bsystem-deploy` | #3 | `ed0ac39` | **squash** |
| `bsystem-deploy` | #4 | `fe23ed6` | merge |

`bsystem-deploy#3` was squashed deliberately. An intermediate commit on that
branch carried credential-shaped test fixtures — literals that gitleaks and
GitGuardian were right to flag, since a committed random-looking string is
indistinguishable from a real credential. Squashing kept that commit out of
`main` entirely; the merged tree generates those fixtures per run instead.

`bsystem-deploy#4` then removed the `.gitleaksignore` that had been carrying
those findings, because after the squash nothing in the repository reached the
commit they referenced. An allowlist nobody can justify is how a real finding
gets suppressed later.

After P17, work continued on the properties the backlog assumed rather than
proved. A PostgreSQL was installed in the working environment, which changed the
method: these tests were run for real before being pushed, instead of being
pushed for CI to try first.

| Repository | Pull request | Merge commit | What it landed |
| --- | --- | --- | --- |
| `bsystem-deploy` | #5 | `7821ef7` | this delivery record |
| `bsystem-integration-core` | #5 | `e64612a` | the store layer, RBAC and scope grants, Global ID immutability, the audit trail, the mapping audit's own queries, and the tenant isolation matrix, all executed against a real PostgreSQL |
| `bsystem-deploy` | #6 | `366f835` | the correction those tests produced |

`bsystem-deploy#6` is worth reading as a result rather than a fix. Two P17
artefacts of mine — `docs/STAGE-ACCEPTANCE.md` and both smoke runners — described
an unconfigured adapter as answering with an empty collection. It answers `503
upstream_unavailable`. A stage deployment that deliberately left an integration
out, which the same document invites, would have been reported as failing by a
runner watching a platform behave correctly. The handler-level test found it by
failing; nothing in review had.

The same method continued past that point, and kept producing the same kind of
result: a property the codebase asserts somewhere and proves nowhere, a test
written for it, and the test correcting the hypothesis rather than confirming
it.

| Repository | Pull request | Merge commit | What it landed |
| --- | --- | --- | --- |
| `bsystem-integration-core` | #6 | `066613a` | an unconfigured integration answered "does not support this capability"; the status is now checked before the capability |
| `bsystem-hub` | #4 | `420d8a8` | the HUB rendered that same 503 as a bare "Помилка"; three 503s are now told apart, because the reader's next step differs for each |
| `bsystem-hub` | #5 | `e7ed340` | the authorization code stayed in the URL when the exchange failed; the strip moved into a `finally` |
| `bsystem-design-system` | #4 | `03fb5d7` | two dark-theme button labels measured below WCAG AA; the fill became its own token, and contrast is now measured from `tokens.css` directly |
| `bsystem-hub` | #2, #1 | `a5cd6aa`, `0bbc881` | `actions/checkout` and `actions/setup-node` 4 → 7 |
| `bsystem-integration-core` | #7 | `6d37af3` | `bsystem_database_pool_acquires_total` was published as a gauge; the registry gained `CounterFunc`, and `/metrics` now has a contract test |
| `bsystem-deploy` | #8 | `2fea7b8` | how to read an empty Grafana panel, which is the dashboard's most misleading output |
| `bsystem-deploy` | #9 | `3f43534` | a skipped E2E suite reported success; `E2E_REQUIRED=1` in CI now makes a skip a failure |
| `bsystem-deploy` | #10 | `356c392` | Redis removed: nothing ever talked to it, and three documents had come to describe its volume as authentik's cache |
| `bsystem-integration-core` | #8 | `469feed` | the architectural reservation for Redis stands; what changes is that the first feature to need one adds the service with the use |

Two of those are worth reading as results rather than fixes.

`bsystem-deploy#9`: `TestMain` lived in `required.go`, not a `_test.go` file, so
the testing framework never called it. The guard meant to make a skipped suite
fail had itself been skipped since it was written. Running it for real is what
said so; nothing in review had.

`bsystem-deploy#10`: the Redis service was provisioned in P0 and never used. A
running container nothing talks to is worse than no container, and this
deployment demonstrated why — `SECURITY.md`, `docs/DEPLOYMENT.md` and
`docs/BACKUP-RESTORE.md` all came to describe it as authentik's cache and task
broker, which it never was. authentik was configured against PostgreSQL alone,
and the Integration Core never read `REDIS_URL`.

The 14 `[!]` items are unaffected and remain the only work left in this backlog.

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
- [x] sync `bsystem-integration-core/docs/openapi.yaml`
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
  dependency instructions, and `bsystem-design-system/docs/RELEASING.md` recording exactly what
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
postgres keeps the five capabilities its entrypoint needs to drop its own
privileges. Those are documented at the service. Redis kept four for the same
reason until the service was removed for want of anything using it.

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

- [x] generic provider interface — `Name`/`Model`/`Complete`
- [x] fake provider — the default, so the gateway is exercised everywhere
- [x] Ollama client — local; the prompt does not leave the deployment
- [x] OpenAI client with env-only config — endpoint, key and model all
      required, no defaults; a misconfigured provider falls back to the fake
      rather than failing open
- [x] no committed secrets — the redactor's own test fixtures are assembled at
      run time, after the platform's secret scanner correctly flagged them

## P13.2 Authorization-aware context

- [x] actor identity
- [x] Integration Core authorization — each source uses exactly the permission
      and scope its own read endpoint uses, asserted against the route
      inventory
- [x] explicit sources — a source the caller may not use is refused, not
      dropped
- [x] explicit Global IDs — no search, no inference, no related records
- [x] no unrestricted SQL

## P13.3 Classification and redaction

- [x] PUBLIC/INTERNAL/CONFIDENTIAL/SECRET/CREDENTIAL policy
- [x] block `CREDENTIAL` — refused outright rather than redacted
- [x] Authorization/token/password/API-key redaction — keyed values, pasted
      secrets recognised by shape, and PEM blocks removed whole
- [x] tests proving credentials never reach provider payload — asserted
      against the payload the provider actually received, not the redactor in
      isolation

## P13.4 AI audit/API

- [x] actor
- [x] provider/model
- [x] requested sources
- [x] entities
- [x] classification summary
- [x] request ID/result
- [x] no full sensitive prompts by default — no prompt is stored at all
- [x] internal AI endpoint — `POST /api/v1/ai/ask`
- [x] fake-provider E2E
- [x] timeout/cancellation — bounded independently of the caller's deadline
- [x] request size limits

`ai.query` is granted to no role. Who may spend money on a model, and whose
data may be put in front of one, is an owner decision; administrators reach it
through the wildcard. See `bsystem-integration-core/docs/AI-GATEWAY.md` and
`adr/ADR-010`.

# P14 — Documentation and ADRs

Priority: CONTINUOUS

- [x] API — `bsystem-integration-core/docs/API.md` and `openapi.yaml`
- [x] RBAC — `AUTHORIZATION.md`, with the group-to-role-to-permission table
- [x] SCOPES — `AUTHORIZATION.md` and `adr/ADR-005`
- [x] GLOBAL-IDS — `GLOBAL-IDS.md`
- [x] EVENTS — `EVENTS.md`, verified against the mapping table in code
- [x] ADAPTERS — `ADAPTERS.md`
- [x] OBSERVABILITY — `OBSERVABILITY.md` and `bsystem-deploy/observability/`
- [x] SECURITY — `bsystem-deploy/SECURITY.md`
- [x] DEPLOYMENT — `bsystem-deploy/docs/DEPLOYMENT.md`
- [x] DEVELOPMENT — `bsystem-integration-core/docs/DEVELOPMENT.md`
- [x] AI — `AI-GATEWAY.md`, with the principle in `AI-GATEWAY-CONTRACT.md`

Every repository now has a `docs/README.md` index, and each marks its P0
documents as historical with the rule that current documents win.

Three documents in the HUB described the platform rather than the HUB, having
been written before the Integration Core existed. One had drifted into
claiming the HUB decides what a user may do, which is backwards. They are now
pointers that say so explicitly: a platform concept is documented where it is
enforced, because a second copy drifts and a reader cannot tell which is
current.

ADRs:

- [x] retry/circuit policy — `adr/ADR-006`
- [x] pagination convention — `adr/ADR-007`
- [x] normalized errors — `adr/ADR-008`
- [x] customer isolation boundary — `adr/ADR-005`
- [x] Design System distribution — `bsystem-design-system/docs/adr/ADR-011`
- [x] search provider architecture — `adr/ADR-009`
- [x] AI routing policy — `adr/ADR-010`

# P15 — Repository hygiene

Priority: MEDIUM

- [x] `.editorconfig` — all four repositories
- [x] `.gitattributes` — LF normalization, and lockfiles marked generated
- [x] `.gitignore` — audited and deduplicated
- [x] CONTRIBUTING.md — per repository, since the toolchains differ
- [x] PR template — asks how a change was verified, not whether tests pass
- [x] issue templates — blank issues disabled, security findings routed to the
      private process
- [x] conventional commit guidance
- [x] release policy — stated per repository rather than invented
- [x] changelog policy — only the design system has one, because only it
      publishes a package
- [x] developer setup

Do not add CODEOWNERS unless ownership is known.

CODEOWNERS was not added. Ownership is not known, and a file claiming
otherwise would route reviews to people who never agreed to them.

# P16 — Performance and resilience

Priority: MEDIUM

- [x] benchmark Global ID/mapping paths — plus redaction, prompt assembly,
      search and cursors; measured, with the machine recorded
- [x] inspect DB indexes — four added in migration `009`, and the rejected
      candidates recorded with them, including one that would have duplicated
      a primary key
- [x] inspect N+1 adapter behavior — every listing endpoint was N+1, two
      transactions per row for contacts and issues; now batched
- [x] bounded concurrency — adapter health probes ran in sequence holding the
      registry lock, so a readiness check cost the sum of every adapter's
      timeout
- [x] graceful shutdown — SIGTERM was killing requests in flight, which during
      a rolling deploy looks like an intermittent platform fault
- [x] server read/write/idle timeouts — the write timeout was shorter than the
      AI provider bound, so an AI request could not complete
- [x] DB pool config — explicit bounds, overridable, with the configured size
      bounded after CodeQL found the int32 conversion wrapping
- [x] fake-upstream load-test harness — `bsystem-deploy/loadtest/`
- [x] measured baseline documentation —
      `bsystem-integration-core/docs/PERFORMANCE.md`

Do not invent performance numbers.

None were invented. Every figure in `PERFORMANCE.md` was measured, the machine
is recorded with them, and the document states plainly that no end-to-end
throughput figure exists — producing an honest one needs production-class
hardware, realistic data volumes and real upstreams, and the platform has none
of the three. The load harness is a regression tool whose absolute numbers
describe four mock upstreams.

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

# P17 — Stage readiness and acceptance preparation

Priority: HIGH

Goal: when real stage credentials and URLs arrive, the owner runs a controlled
acceptance with minimal manual work. Nothing here deploys, requests production
credentials, or makes a destructive change.

## P17.1 Stage environment contract

- [x] `docs/STAGE-ACCEPTANCE.md`
- [x] required services and which are optional
- [x] every environment variable classified: required/optional, secret/public
- [x] no real secret value committed
- [x] network paths, including the Core→authentik direction that trips people
- [x] DNS placeholders on reserved documentation domains
- [x] health and readiness URLs, including which need a service token
- [x] rollback assumptions
- [x] backup prerequisites

## P17.2 Environment validation

- [x] `scripts/stage-preflight.sh`
- [x] `scripts/stage-preflight.ps1`
- [x] required variables present; placeholders rejected
- [x] URL format, DNS resolution, bounded TCP connect
- [x] required local files, Docker and Compose availability, stage render
- [x] secrets reported by length, never by value
- [x] non-zero exit on a blocking failure
- [x] fixture tests, including one that fails if a secret is ever printed

## P17.3 Stage smoke runner

- [x] `scripts/stage-smoke.sh`
- [x] `scripts/stage-smoke.ps1`
- [x] `/health`, `/readyz`, `/metrics`, HUB `/healthz`, OIDC discovery
- [x] human API, machine API, adapter health
- [x] PostgreSQL and NATS through readiness rather than probed directly
- [x] anonymous callers asserted to be rejected
- [x] every check a GET; nothing is created, updated or deleted
- [x] missing token SKIPs its section with a reason instead of failing the run
- [x] no credential in output or report

## P17.4 Acceptance report

- [x] `artifacts/stage-acceptance.json`
- [x] `artifacts/stage-acceptance.md`
- [x] timestamps, per-repository commit, service, check, PASS/FAIL/SKIP/BLOCKED
- [x] duration and request id for correlation with the audit trail
- [x] safe diagnostics only

## P17.5 Real-adapter contract readiness

- [x] `bsystem-integration-core/docs/adapters/ESPOCRM-STAGE.md`
- [x] `bsystem-integration-core/docs/adapters/REDMINE-STAGE.md`
- [x] `bsystem-integration-core/docs/adapters/OUTLINE-STAGE.md`
- [x] base URL, auth header, endpoints, pagination, envelope, fields consumed
- [x] timeout, retry and circuit behaviour
- [x] behaviour when an optional field is absent
- [x] version-sensitive assumptions listed per adapter
- [x] no claim of live compatibility anywhere

## P17.6 authentik stage checklist

- [x] `docs/AUTHENTIK-STAGE.md`
- [x] application, provider, public PKCE client, exact issuer
- [x] redirect and logout URIs, with why wildcards are refused
- [x] scopes, groups, claims, service identity expectations
- [x] MFA, customer and service identity test checklists
- [x] no client secret anywhere

## P17.7 Tenant-isolation acceptance matrix

- [x] `docs/TENANT-ISOLATION-MATRIX.md`
- [x] nine actors against ten resources, human and machine surfaces
- [x] ALLOW / DENY / SCOPED / N/A derived from seeded grants and route declarations
- [x] customer ALLOW cases marked BLOCKED rather than invented
- [x] `404` not `403` for a resource the caller may not learn exists

## P17.8 Data mapping audit tool

- [x] `bsystem-integration-core/cmd/mapping-audit`
- [x] invalid prefixes, unregistered types, one record mapped twice
- [x] references to Global IDs that do not resolve
- [x] records with no owning client, reported as warnings
- [x] read-only: every statement a SELECT, no source system contacted
- [x] identifiers and fixed details only; tests fail if that changes

## P17.9 Migration readiness

- [x] `bsystem-integration-core/docs/MIGRATION-READINESS.md`
- [x] order, tables and indexes added, backward compatibility
- [x] duration classified rather than invented
- [x] rollback strategy, and where locks would matter
- [x] confirmed no migration drops, rewrites or deletes
- [x] CI applies every migration to an empty database
- [x] CI starts the application against the migrated schema

## P17.10 Backup and restore runbook

- [x] `docs/BACKUP-RESTORE.md`
- [x] what to back up and what not to
- [x] verification by restoring, not by inspecting
- [x] stop/start ordering, with why the Core must stop first
- [x] acceptance criteria after a restore
- [x] no external backup target configured

## P17.11 Release manifest

- [x] `scripts/release-manifest.sh`
- [x] repository, branch, commit, dirty state per repository
- [x] schema level, OpenAPI version and hash, design system version
- [x] `bsystem_build_info` and `bsystem_schema_migrations_applied` on `/metrics`

## P17.12 Stage Compose profile

- [x] `docker-compose.stage.yml` as an overlay, leaving dev and E2E untouched
- [x] no hardcoded secret; required variables fail at render
- [x] no public bind unless `STAGE_PUBLISH_ADDRESS` says so
- [x] healthchecks, restart policies, internal networks, volumes
- [x] resource limits only where justified

## P17.13 CI validation for stage assets

- [x] stage Compose renders, and is hardening-checked as part of its stack
- [x] shell scripts parse and pass shellcheck
- [x] PowerShell scripts parse
- [x] `scripts/check-stage-secrets.py` — no committed secret-like value
- [x] `scripts/check-doc-links.py` — every referenced path exists
- [x] migration-from-zero test runs
- [x] fixture tests for preflight and smoke

## P17.14 OpenAPI acceptance examples

- [x] 2xx examples for every endpoint an acceptance walks
- [x] 401 documented everywhere; 403 where a permission or scope can reject;
      404 where a single resource is addressed
- [x] a contract test fails on a documented rejection the platform cannot return
- [x] example hosts restricted to reserved documentation domains

## P17.15 Stage handoff

- [x] `docs/STAGE-HANDOFF.md`
- [x] what is autonomous, what the owner must supply
- [x] execution order, with PowerShell and shell command sequences
- [x] expected outcomes, rollback, acceptance checklist
- [x] known limitations and the 14 remaining blocked items

# P18 — Repository artifact cleanup

Priority: HIGH

Goal: remove generated or accidental repository artefacts and make their return
fail CI rather than rely on review.

- [x] inspect the committed `bsystem-integration-core/mapping-audit` root file
- [x] if it is a compiled/generated executable, remove it from Git while keeping
      `cmd/mapping-audit` source intact
- [x] add precise ignore rule(s) for local build output without hiding source
- [x] scan all four repositories for committed binaries, archives, coverage
      outputs, temporary files and generated build directories
- [x] add a CI guard that rejects known generated executable/build artefacts
- [x] document any intentionally committed generated file and why it belongs

It was a 13 MB ELF executable, and it held no credential: the strings that look
like one are Go stdlib and pgx symbols. It did embed the build machine's module
paths, which is a consequence of committing a binary rather than of anything in
it.

The cause is the part worth keeping. `go build ./cmd/x` with no `-o` writes
`./x` in the repository root, and both Go repositories ignored only the
directories a build can be *told* to use. This repository already carried the
same scar — a `loadtest/loadtest` line added the last time it happened — and
the new check found four more uncovered mock binaries on its first run.

So both guards check the property rather than a list of paths: nothing tracked
may be a compiled executable or an archive, decided by leading bytes because a
stray binary does not arrive with a helpful suffix; and every default build
output must be ignored, asked of `git check-ignore` rather than by
reimplementing precedence. `bsystem-hub` and `bsystem-design-system` were
scanned and are clean; their build output is directory-shaped (`dist/`,
`node_modules/`) and already ignored, so the root-file gap that produced this
one does not exist there.

History is not rewritten. The blob stays reachable in the commit that added it,
because rewriting published history is not an autonomous act. What changed is
that it is no longer in the tree and cannot return unnoticed.

Delivered in `bsystem-integration-core#10` and `bsystem-deploy#13`.

One thing was found on the way and is fixed in the same change: the commit that
added this P18-P27 backlog turned `main` red. `check-doc-links.py` flagged the
OpenAPI document referenced in P22 by a bare `docs/`-rooted path: that file
lives in the Integration Core, and the reference resolved against this
repository, where there is none. P22 now names the repository that owns it.

Quoting the broken path in this note reintroduced the finding, because the
checker reads backticked paths wherever they appear — including in prose about
the fix. It is right to: a path in a document is a claim that the path exists,
and a document explaining a correction is no exception.

Definition of Done:
- no accidental build artefact remains tracked;
- the source needed to reproduce tools remains tracked;
- a regression test/CI check fails if the same class of artefact returns;
- relevant CI and Security workflows are green.

# P19 — Independent invariant audit

Priority: CRITICAL
Depends on: P18

Goal: do not treat this backlog, documentation or existing tests as proof that
all important properties are covered. Find claims the platform makes that no
test actually proves.

- [x] inventory security, correctness and operational invariants across all four
      repositories
- [x] identify every invariant described in docs/comments/config but lacking a
      direct test or executable check
- [-] prioritize authorization, tenant isolation, audit integrity, Global ID
      immutability, migration safety, event delivery and secret handling
- [x] add non-vacuous tests: prove each new test fails when its invariant is
      deliberately broken
- [x] fix defects discovered by those tests rather than changing expectations
      to match incorrect behavior
- [x] record findings and rationale in the repository that owns the invariant

Event delivery and secret handling in adapter errors remain, and are scheduled
as P21 rather than duplicated here.

The audit found ten defects so far, and the pattern in them is worth keeping:
in almost every case the property was *stated* somewhere — a comment, a
document, a variable name — and the statement was what made it look covered.

| Finding | Where it was claimed | Landed |
| --- | --- | --- |
| an unconfigured integration answered "does not support this capability" | the registry knew it was unconfigured and `/health` said so | `core#6` |
| the HUB rendered that 503 as a bare error | `DataState`'s own contract distinguishes these states | `hub#4` |
| the authorization code stayed in the URL on a failed exchange | — | `hub#5` |
| two dark-theme button labels below WCAG AA | `docs/ARCHITECTURE.md` requires sufficient contrast | `ds#4` |
| a `_total` series published as a gauge | the name itself | `core#7` |
| a skipped E2E suite reported success | the guard existed, in a file the test runner never reads | `deploy#9` |
| stage preflight checked a variable the stage stack ignores | three documents named it as the one that governs | `deploy#12` |
| the hardening checker never looked at `read_only` | `docs/DEPLOYMENT.md` said it failed on a regression in "any of this" | `deploy#12` |
| an edited migration erased its own evidence | the comment beside the checksum said it made one visible | `core#9` |
| a failed audit write was only logged | the same platform counts event and notification outcomes, for this exact reason | `core#11` |
| concurrent allocation answered some callers with raw SQL | the error model forbids exposing SQL errors | `core#12` |

Two are worth separating from the rest. The migration checksum and the audit
write were both *mechanisms built for this purpose* that did the opposite: one
overwrote the evidence it existed to preserve, the other recorded nothing a
dashboard could see. And the concurrent-allocation case was not unnoticed at
all — a test tolerated it deliberately, reasoning that a refused duplicate
beats a silent second identity. That reasoning was sound and the options were
three, not two.

Definition of Done:
- each newly claimed invariant is backed by an executable check;
- mutation/restoration evidence exists for high-risk checks;
- defects found by the audit are fixed or explicitly BLOCKED;
- all affected repositories are green.

# P20 — Concurrency and database correctness

Priority: CRITICAL
Depends on: P19

## P20.1 Concurrency

- [x] run and expand `go test -race ./...`
- [ ] concurrent `EnsureIdentity`
- [ ] concurrent `EnsureServiceIdentity`
- [x] concurrent Global ID allocation and source mapping
- [ ] concurrent scope grant/revoke/read
- [ ] concurrent audit writes
- [ ] adapter registry/readiness concurrency
- [ ] shutdown while requests and DB work are in flight

## P20.2 Database correctness

- [x] inspect transaction boundaries and error handling
- [x] verify uniqueness constraints close races rather than application checks
      alone
- [x] test rollback on partial failures
- [x] test concurrent startup/migration behavior
- [ ] migration-from-zero plus upgrade from every practical historical schema
      level represented by the repository
- [ ] inspect indexes against actual lookup/order/filter paths
- [ ] identify N+1 queries and repeated transactions not already covered by P16
- [ ] document query-plan evidence when an index is added or rejected

`core#12` landed the first of these. The constraint did close the race — no
duplicate Global ID was ever minted — but the application path did not handle
losing it: the read-then-insert took its snapshot before the winner committed,
so the loser's INSERT hit the constraint and the driver error travelled all the
way out as HTTP 400 with the table, column tuple and constraint name in the
body. Eight racers, measured: six answered, two refused.

The remaining P20.1 items (identity and service-identity races, scope grant and
revoke, audit writes, adapter registry, shutdown in flight) are not yet done.

P20.2 found the worst of this wave. **Several Core instances could not start at
once.** Four released together against a fresh database: three of the four did
not boot, on every one of three runs, with

```text
create schema history: ERROR: duplicate key value violates unique
constraint "pg_type_typname_nsp_index" (SQLSTATE 23505)
```

Migrations are written to be idempotent, which makes them safe to re-run and
says nothing about running them simultaneously. `CREATE TABLE IF NOT EXISTS` is
not atomic against another session creating the same table: the existence check
and the creation do not share a lock, so both pass the check and one loses on an
internal unique index. `Migrate` runs from `Open`, so it is a failure to start.

That is the ordinary shape of a deployment — several replicas, a rolling
restart, or everything returning at once after an outage — and a restart policy
turns it into flapping containers rather than a report. On a stage acceptance it
would have read as services that come up on the second or third try, with
nobody able to say why. Fixed with a session-scoped advisory lock in
`bsystem-integration-core#13`.

Rollback on partial failure was covered by half: the existing case has the first
write failing, where there is nothing to undo. The new one stores an event and
then fails the status update. Worth recording what that test proves and what it
cannot: it proves the two writes share a transaction, and it cannot prove the
error handling around them, because PostgreSQL aborts a transaction at the first
failed statement and discards the event either way. Dropping the error check
leaves it green; splitting the writes across transactions is what fails it.

Still open in P20.2: migration from every historical schema level, index
evidence against real lookup paths, and the N+1 review.

Definition of Done:
- race detector green;
- targeted concurrent tests exist for identity and Global ID paths;
- schema constraints protect uniqueness under concurrency;
- migration and transaction failure cases are tested;
- no invented performance claims.

# P21 — Adapter chaos and failure semantics

Priority: HIGH
Depends on: P19

For EspoCRM, Redmine and Outline, exercise:

- [x] slow response / context deadline
- [x] malformed JSON
- [x] truncated response body
- [x] oversized response body
- [x] connection reset / transport error
- [x] 401 / 403 / 404
- [x] 408 / 429
- [x] 500 / 502 / 503 / 504
- [x] valid and invalid `Retry-After`
- [x] cancellation while sleeping between retries
- [x] circuit breaker closed/open/half-open transitions under concurrent calls
- [x] recovery after a transient upstream outage
- [x] verify credentials never appear in returned errors, logs or metrics

`bsystem-integration-core#13` closed the one item that was genuinely open. The
rest of this list was already covered by `internal/adapters/httpx`, and is
marked on that basis rather than on new work — checking each clause honestly is
the task, not adding tests to things that have them.

The credential clause holds three ways: the error-message test refuses the key,
the internal host, the database user, the request path and the word `Bearer`;
the adapter packages log nothing at all, so there is no adapter log to leak
through; and the metric labels are drawn from closed vocabularies and pinned by
the metrics contract test.

The finding was the circuit breaker under concurrency, and it was the worst kind
— the breaker doing the opposite of its job at the one moment it exists for. A
call still in flight when the circuit opened reported its outcome into whatever
state the breaker had reached by the time it returned. A success arriving during
half-open closed the circuit on evidence gathered before the upstream was even
suspected, and, having reset the failure count on the way, left the real probe's
failure one short of reopening it. The upstream was down, the probe had just
proved it, and full traffic was flowing.

It survived a file with fourteen cases, one of them named for concurrency,
because every case admits a call and reports that same call's result
immediately. That test exercised the mutex; the race detector was and is clean.
The defect was in whose result the breaker listens to, not in how it reads it.

Definition of Done:
- failure behavior is deterministic and normalized;
- retry occurs only where idempotent and safe;
- cancellation immediately stops unnecessary retries;
- circuit behavior is tested without wall-clock-flaky sleeps;
- all adapter contract tests and CI are green.

# P22 — OpenAPI ↔ implementation drift detection

Priority: HIGH
Depends on: P19

Goal: Spectral validates the document; this task validates that the document and
running HTTP surface describe the same contract.

- [x] build an inventory of registered human and service routes
- [x] compare implemented method/path pairs with `bsystem-integration-core/docs/openapi.yaml`
- [x] fail CI on undocumented implemented public API routes
- [x] fail CI on documented routes with no implementation
- [x] verify important success and rejection status codes against handlers
- [x] verify documented query/path parameters exist in implementation
- [x] ensure normalized DTO/error envelope contract tests use OpenAPI examples or
      a generated schema validator where practical
- [x] keep health/metrics/internal exceptions explicit rather than silently
      ignored

Seven of these eight were already enforced by
`bsystem-integration-core/cmd/server/openapi_test.go`, which compares the route
inventory in both directions, the authentication boundary, the failures each
route documents, every emitted error code, success and rejection examples, and
operational routes as explicit exceptions. It refused a change earlier in this
wave until two new error codes were documented.

The open one was parameters, added in `bsystem-integration-core#13`. Both
directions held already, so it is a guard rather than a fix. Parameters fail
more quietly than routes: a route that disappears gives a caller a 404, while a
parameter that stops being read gives them a 200 and the wrong rows —
`?severity=critical` returning every incident, with nothing anywhere saying the
filter was ignored.

Resolving `$ref` was not optional. Three parameters are declared only under
`components`, `limit` and `cursor` among them, and `limit` appears on twelve
routes. Reading the inline form alone would have made them invisible and the
check would have passed because it never looked at them — the exact failure
this wave keeps finding, and a poor thing to ship in the check written to stop
it.

Definition of Done:
- adding/removing a public handler without updating OpenAPI fails CI;
- adding a nonexistent OpenAPI endpoint fails CI;
- auth/error status drift is caught by tests;
- Spectral and implementation-drift checks are both green.

# P23 — Negative and resilience E2E expansion

Priority: HIGH
Depends on: P20, P21, P22

Add deterministic E2E scenarios for:

- [x] multi-page pagination across all supported adapters
      — `TestCollectionsAreWalkableByCursor`, plus the notification and search
      pagination scenarios, which also prove a walk neither repeats nor loses
- [x] upstream 429 followed by recovery
      — `TestTransientUpstreamFailuresAreRetried`, one failure then success,
      within the retry budget, so the caller never sees it
- [x] upstream 5xx followed by circuit-open/recovery behavior
      — `TestRetriesAreBounded`, `TestDeterministicFailuresAreNotRetried`,
      `TestCircuitOpensUnderSustainedFailure`
- [x] malformed upstream entity without cross-record corruption
      — `TestAMalformedUpstreamPayloadCorruptsNothingAroundIt`. An upstream
      answering 200 with an unreadable payload is worse than one that is down,
      because the refusal is invisible. The scenario pins the normalized 502,
      that the decoder's own words do not travel with it, that contacts (same
      upstream, different path) still read, and that after recovery every
      client carries the Global ID it carried before
- [x] duplicate source identity/mapping conflict
      — `TestNormalizedEntitiesAndGlobalIDStability`, `TestUserGlobalIDIsStable`,
      `TestServiceGlobalIDIsStable`, `TestServerRegistrationIsIdempotent`
- [x] source record deleted after a Global ID was allocated
      — `TestASourceRecordDeletedAfterAllocationKeepsItsGlobalID`
- [x] expired/invalid human identity
      — `TestAuthenticationRejections`, and `TestRejectionsNeverLeakSecretsOrTopology`
      for what the refusal is allowed to say
- [x] adapter disabled/unconfigured
      — `TestACoreWithAnUnconfiguredAdapterStartsAndSaysSo`. This needed a
      stack change, not just a scenario: the E2E stack now runs a second
      Integration Core from the same image against the same database with
      `OUTLINE_URL` deliberately absent. Core has integration tests for the
      refusal, but nothing had ever run the deployment — "the platform starts
      without Outline" was an assumption about a configuration no stack had
      ever brought up, and it is the configuration a stage acceptance is most
      likely to meet
- [ ] NATS unavailable and recovery
      — not covered, and not coverable from the harness as it stands. The
      scenarios talk HTTP and the NATS wire protocol to a running stack; they
      cannot stop and start a container. Proving *recovery* in particular needs
      container lifecycle control in the E2E job rather than a new assertion.
      Readiness already distinguishes it (`nats: degraded` without making the
      platform unready), so what is missing is the demonstration, not the
      behavior
- [ ] PostgreSQL unavailable during readiness/startup where practical
      — same limit, and "where practical" is doing real work in that line.
      Readiness answers 503 with `database: error` when the pool cannot ping,
      but showing it means taking PostgreSQL away from a running stack
- [x] concurrent reads of the same newly discovered source records
      — `TestConcurrentAllocationsOfOneNewRecordAgreeOnOneGlobalID`. Eight
      callers released together against a source record named after the run.
      Non-vacuous against the code as it stood before P20.2: without the
      unique-violation fallback in `CreateGlobalEntity` the losers answer 500
- [x] authorization scope changed between requests
      — `TestARevokedScopeStopsWorkingOnTheNextRequest`. True by construction
      today (the evaluator holds no cache), which is exactly why it is pinned:
      the obvious way to make the platform faster is a cache, and one added
      without this test keeps a revoked scope working with nothing failing and
      nothing logged
- [x] prove an E2E-required suite cannot pass by skipping all scenarios
      — `TestTheStackRequirementGuard`

Two items are left open above rather than quietly dropped. Both need the E2E
job to stop and start containers mid-suite, which is a change to how the
scenarios are run, not another assertion. Recorded here so the gap is visible
instead of being inferred from an unchecked box.

Definition of Done:
- scenarios are deterministic and do not require production credentials;
- expected degraded behavior is distinguished from platform failure;
- no test succeeds vacuously;
- Autonomous E2E remains green on `main`.

# P24 — Docker and supply-chain hardening follow-up

Priority: HIGH
Depends on: P18

## P24.1 Runtime/container review

- [x] re-evaluate user/root, `read_only`, `cap_drop`, `no-new-privileges`, tmpfs,
      healthchecks, restart policy, internal networks and published ports for
      every rendered base/E2E/stage service
      — gone through clause by clause against the three rendered stacks. What
      the checker already enforces on every service: `no-new-privileges`,
      `cap_drop: ALL` (with a named exempt set), no `privileged`, `read_only`
      for a pinned set of services, no Docker socket, and no port published on
      every interface. What was inspected rather than enforced, and found
      sound: every image sets a non-root `USER` (mocks `65532`, Integration
      Core `app`, HUB `101`), so no Compose `user:` override is needed and the
      comment claiming it is now verified rather than assumed; the read-only
      services need no tmpfs beyond HUB's `/tmp`, which the E2E stack
      demonstrates every run; healthchecks are present and gate `depends_on`;
      the restart policy is `unless-stopped`; data networks are `internal`.
      One real gap came out of it and is fixed below
- [x] ensure the hardening checker fails on an empty or unexpectedly incomplete
      rendered stack
      — closed by `deploy#12`, and extended here: the new bind-mount check has
      its own vacuity guard, because a stack rendering no bind mount would let
      that check pass in silence exactly as an empty stack once did
- [x] ensure stage exposure checks use the variable that actually controls the
      stage mapping
      — closed by `deploy#12`
- [x] check volume ownership/permissions and writable paths
      — the gap the review found. Every bind mount in every stack is `:ro`
      today and was before this work, but nothing kept it that way, and
      dropping `:ro` is a two-character edit that renders and runs. A bind
      mount is a handle on the host filesystem, and these carry configuration
      and seed data inward — an authentik blueprint, the PostgreSQL init
      scripts — so a container that can rewrite them can change what the next
      start believes. The checker now refuses a writable bind, with an empty
      `WRITABLE_BINDS` allowlist for an exception that argues for itself.
      Verified by mutation: removing `:ro` from `postgres/init` fails the
      check, and a stack with no bind mount fails the vacuity guard

Found while doing the above, and fixed with it: `.gitignore` covered no Python
bytecode, although `scripts/` is full of Python that CI and contributors run —
importing any of it as a module leaves an untracked `.pyc`. `check-artifacts.py`
could not have caught a committed one either: its magic table knew ELF, MZ,
Mach-O, zip, gzip, xz and zstd, but not bytecode. Both halves are closed.
Bytecode has no fixed prefix to match — the magic number is a version counter
that changes with every CPython release — so the check matches its shape
instead, and is verified by compiling a real `.pyc` and confirming both that
the checker rejects it and that a CRLF text file is not mistaken for one.

## P24.2 Supply chain

- [x] review GitHub Actions pinning policy and document whether major tags or
      immutable SHAs are required
      — every workflow in all four repositories pins to a mutable major tag,
      not an immutable SHA. SECURITY.md now states that as a decision with its
      reasoning and the condition that should reverse it (the first workflow
      given a token that can write to a registry, a deployment target or
      another repository), rather than leaving it to look like nobody checked.
      The tags have also drifted apart — `actions/checkout` is v7 here and in
      HUB, v4 in Core and the Design System; `actions/setup-node` is v7 in HUB,
      v4 in the other two — and the table records it, because a reader cannot
      tell a deliberate difference from an unnoticed one
- [x] verify govulncheck/npm audit/Trivy/Gitleaks/CodeQL still cover every repo
      and relevant image
      — verified per repository, and it found a gap: `govulncheck` here took
      its modules from a hand-written matrix listing `mocks` and `e2e`.
      `loadtest` is compiled by CI and shipped by this repository and was
      scanned by nothing. It was never removed — it was never added, and there
      was no way to notice, because a module absent from the matrix produces
      no failure and no output. Fixed, and now machine-checked by
      `scripts/check-scan-coverage.py` in both directions. The same review
      found this repository's Go had `govulncheck` and no static analysis at
      all while Integration Core has run `staticcheck` since it had CI; that
      asymmetry is closed too. An earlier note in this session said deploy has
      "no application code" — that was wrong, it has three Go modules
- [x] verify SBOM artefacts are generated from the exact build being tested
      — checked, and the answer is no. The SBOM is generated from an image
      built inside the Security job, not from the image the E2E job ran. The
      four mocks share one Dockerfile and one module so one SBOM does describe
      them all, but it is a rebuild of the same source rather than the tested
      artefact. Recorded in SECURITY.md as what the SBOM does and does not
      answer, instead of being left to imply more than it delivers
- [x] review Docker base image pinning/update policy
      — pinned to version tags, not digests, and documented with the same
      reasoning as the actions: a tag that keeps receiving patch updates is
      what makes a rebuild pick up a fixed CVE, and a digest pin without an
      updater freezes the vulnerabilities along with the version
- [x] generate a dependency/license inventory and flag incompatible licenses if
      any
      — `docs/DEPENDENCY-LICENCES.md`, gathered by reading each dependency's
      own licence text rather than its manifest field. Split by what actually
      ships, because an obligation attaches to what is distributed and counting
      test tooling alongside linked libraries is how a clean inventory hides
      the one entry that matters. Integration Core links twelve third-party
      modules (MIT, Apache-2.0, BSD-3-Clause); HUB ships nine packages (eight
      MIT, `oidc-client-ts` Apache-2.0); the Design System ships none at all —
      React is a peer dependency; this repository has no third-party Go.
      Nothing is incompatible, and two things are named rather than left in a
      count: MPL-2.0 (`axe-core`, `lightningcss`) is in both frontend build
      trees and neither production tree, and `spawndamnit` declares
      `SEE LICENSE IN LICENSE` where the file is MIT verbatim.
      Two of my own errors are recorded there because both are easy to repeat:
      `klauspost/compress` is BSD-3-Clause whose licence file *contains* the
      Apache-2.0 text for vendored portions, and a first pass matching "Apache
      License" called it Apache-2.0; and 68 packages first counted as having no
      licence are platform binaries for other architectures that this machine
      never installed

Definition of Done:
- rendered stacks are security-checked rather than source YAML only;
- no silent public exposure regression is possible through the documented
      variables;
- security workflows stay green without broad allowlists.

# P25 — Documentation/configuration consistency automation

Priority: MEDIUM-HIGH
Depends on: P19

Automate checks for facts that currently exist in more than one place:

- [ ] documented env vars vs Compose/runtime env vars
- [ ] documented published ports vs Compose rendered ports
- [ ] documented service names vs Compose service names
- [ ] authentik blueprint groups vs mock identity groups vs RBAC seed mappings
- [ ] Global ID prefixes/types vs implementation
- [ ] adapter capability names vs registry/implementation/docs
- [ ] metric names/types documented vs emitted
- [ ] route names/endpoints in docs vs OpenAPI
- [ ] documented file/script paths exist
- [ ] stale references to removed services/dependencies fail CI

Definition of Done:
- at least the high-risk duplicated facts are machine-checked;
- a one-character drift in identity group/capability/metric names fails CI;
- docs are updated only where implementation is authoritative.

# P26 — Cross-repository compatibility gate

Priority: HIGH
Depends on: P22, P23, P25

Goal: a green repository must not silently depend on an incompatible sibling
`main`.

- [ ] define the provider/consumer compatibility matrix for Integration Core,
      Deploy, HUB and Design System
- [ ] run cross-repo checks against sibling `main` for pull requests where
      practical
- [ ] validate Integration Core + Deploy E2E together
- [ ] validate HUB against the normalized API/OpenAPI contract
- [ ] validate Design System package consumer build without consuming an
      unversioned `main`
- [ ] make fallback-to-main behavior explicit and fail loudly when a requested
      sibling ref is missing in CI
- [ ] produce a compact compatibility manifest/report as a CI artifact

Definition of Done:
- a breaking provider change is detected before a consumer merge where the
      repository permissions/workflow allow it;
- cross-repo jobs cannot pass because the intended sibling branch/ref was
      silently skipped;
- main-to-main compatibility is green.

# P27 — Stage readiness package final audit

Priority: HIGH
Depends on: P18-P26

This remains autonomous preparation only. Do not deploy to a real environment
and do not request production secrets merely to complete it.

- [ ] re-run an independent review of `docker-compose.stage.yml`, preflight,
      smoke runner, release manifest and acceptance report generation
- [ ] ensure every remaining real-environment prerequisite is represented as a
      clear BLOCKED item rather than a guessed value
- [ ] verify stage preflight cannot print secrets in success or failure paths
- [ ] verify smoke checks are read-only and cannot mutate upstream systems
- [ ] verify rollback and backup/restore instructions match the actual current
      stack after P18-P26
- [ ] verify all example hosts/IPs/credentials are reserved placeholders
- [ ] produce a final autonomous handoff report with exact owner actions in
      execution order

Definition of Done:
- all non-owner-dependent stage preparation is green and reproducible;
- no real credential, tenant mapping or SLA value is invented;
- only true runtime/owner decisions remain BLOCKED;
- CI/Security/Autonomous E2E are green for every repository touched.

## P18-P27 execution rule

For this hardening wave, do not add unrelated business features. Prefer finding
and proving hidden defects over increasing feature count.

A task may be marked `[x]` only when:

```text
1. the change is committed/merged in GitHub according to the repository workflow;
2. the relevant CI and Security workflows are green;
3. tests are non-vacuous and fail when the protected property is deliberately broken;
4. API/docs/migrations are synchronized where applicable;
5. TASKS.md records the resulting state and any new BLOCKED owner action.
```

If one task is blocked by production access, credentials or an owner-only
business decision, mark only that task `[!]` and continue with the next
independent task.
