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
| `bsystem-hub` | #5 | `e7ed340` | the authorization code stayed in the URL on a failed exchange; the strip moved into a `finally` |
| `bsystem-design-system` | #4 | `03fb5d7` | two dark-theme button labels below WCAG AA; the fill became its own token, and contrast is now measured from `tokens.css` directly |
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

### P18-P27, and then P28-P36

Two further waves followed, both merged to `main` in every repository they
touched, with `CI` and `Security` green at each merge commit. P18-P27 hardened
what existed; P28-P36 is the release-readiness wave, and it is the last
autonomous foundation work — see the freeze at the end of this file.

| Repository | Pull request | Merge commit | What it landed |
| --- | --- | --- | --- |
| `bsystem-deploy` | #28 | `c9eed7e` | P28: the dependency outages, driven from the CI runner, and the fail-fast startup contract they produced |
| `bsystem-integration-core` | #17 | `de883a9` | P29: a transactional outbox for the three events that cannot be re-derived, and the two subject-convention defects the inventory found |
| `bsystem-deploy` | #29 | `9f469cf` | P29: a Global ID minted while the broker is stopped, delivered when it returns |
| `bsystem-integration-core` | #18 | `19125b8` | P30: an authorization change and its audit record in one transaction |
| `bsystem-integration-core` | #19 | `5682c04` | P31, P32: local JWKS validation with no downgrade to UserInfo; a per-principal limiter on the expensive surfaces |
| `bsystem-deploy` | #30 | `5f2fdc7` | P31, P32: the configuration, and one noisy identity that does not throttle another |
| `bsystem-integration-core` | #20 | `c88790c` | P34: the versioning policy, and `apk upgrade` for the CVE the release scan found |
| `bsystem-hub` | #7 | `4ba8ed9` | P34: the consumer gate's own tests, and the same base-image upgrade |
| `bsystem-deploy` | #31 | `7cdf13e` | P33, P34, P35: release evidence bound to the exact image, the compatibility block, and a backup runbook CI runs |
| `bsystem-integration-core` | #21 | `9415138` | P36: the review finding — a discovery document no longer decides where signing keys are fetched from |

Three results from these waves are worth reading as results rather than fixes.

**P29's inventory found two defects before it found a design question.** Three
of the seven event publishers did not use the subject convention, so a consumer
subscribed to `bsystem.events.>` — which the convention, the documentation and
the E2E harness all use — had never received `identity.created`,
`service_identity.created` or `global_id.created`. The metric counted them
published. And the Global ID handler announced *every* allocation call,
including the paths that return an identifier which already existed, so asking
twice for one source record announced two creations of a thing created once.

**P33's first run found a real vulnerability in the product image.** The
Security workflow scanned one mock image, on the reasoning that the four mocks
share a Dockerfile, and nothing had ever scanned the image the platform ships.

**Three tests in P28-P36 failed CI and were themselves what was wrong**, each
recorded where it was fixed: a per-process delivery counter used to describe a
queue two Cores drain; an audit comparison that counted its own observation,
because reading a Global ID is an audited action; and a preflight check that
read an empty `AUTHENTIK_USERINFO_URL` as "cannot authenticate" when the base
Compose file supplies it.

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
- [x] concurrent `EnsureIdentity`
      — one user, and exactly one caller told it was the first. The `created`
      flag matters as much as the Global ID: it is what drives everything that
      happens on first sight, so two callers both told they created the
      identity would each do that work, and the second would look like a
      legitimate first sighting
- [x] concurrent `EnsureServiceIdentity`
      — this one found a real defect. There was no advisory lock:
      `SELECT ... FOR UPDATE` locks a row, and on a first sighting there is no
      row to lock, so every concurrent transaction saw nothing, every one
      bumped the counter, and twelve simultaneous registrations produced three
      distinct Global IDs for one service. The upsert then let the last writer
      win, so the earlier callers walked away holding identifiers the platform
      does not recognise — each of them told `created=true`. `EnsureIdentity`
      had taken `pg_advisory_xact_lock` all along with a comment saying why;
      the machine path beside it did not, and nothing compared the two. For
      service identities this is the normal case: an integration starting
      several replicas registers from all of them at once
- [x] concurrent Global ID allocation and source mapping
- [x] concurrent scope grant/revoke/read
      — identical grants converge on one row, and a grant racing a revoke ends
      in one of the two states somebody asked for rather than a third
- [x] concurrent audit writes
      — all survive. An audit trail that drops an entry under load is worse
      than none, because it is trusted anyway
- [x] adapter registry/readiness concurrency
      — and the first version of this test was worthless. `Registry.Health`
      says it holds the lock only long enough to snapshot, because holding it
      across a network call would block every other reader; I asserted exactly
      that, and the test still passed when the probe was moved inside the lock.
      It is a *read* lock, and readers do not exclude each other. What blocks is
      a writer: `Register` waits for the read lock to clear, and Go's
      `RWMutex` then queues every later reader behind it — so one stalled
      upstream stalls registration and through it `/adapters`, `/readyz` and
      every handler that resolves an adapter. The assertion is on `Register`
      now, and the same mutation fails
- [x] shutdown while requests and DB work are in flight
      — the behaviour lived in the tail of `main` and could not be called at
      all, so it is `serveUntilSignal` now. A request that arrived before
      SIGTERM must finish rather than be dropped: that is what a rolling deploy
      does to every instance several times, and the failure is invisible on a
      healthy platform — users who did nothing but arrive at the wrong moment
      get an error that reads as an intermittent platform fault rather than as
      a deployment, so the deploy is the last place anybody looks

## P20.2 Database correctness

- [x] inspect transaction boundaries and error handling
- [x] verify uniqueness constraints close races rather than application checks
      alone
- [x] test rollback on partial failures
- [x] test concurrent startup/migration behavior
- [x] migration-from-zero plus upgrade from every practical historical schema
      level represented by the repository
      — every deployment that exists is at *some* level, and an upgrade has to
      work from each; the empty database is the one case CI exercises and the
      one case no real deployment is ever in. The levels are derived from the
      migrations directory, so a new file is covered without anybody
      remembering. Verified by mutation: one `CREATE TABLE IF NOT EXISTS`
      turned into a plain `CREATE` fails from that level onwards, which is
      exactly the defect — a non-idempotent migration works on an empty
      database and breaks every upgrade
- [x] inspect indexes against actual lookup/order/filter paths
      — already done by `009_indexes.sql`, which was written by reading every
      query rather than by adding indexes that sound useful, and which records
      the candidates it rejected
- [x] identify N+1 queries and repeated transactions not already covered by P16
      — and my first test for it was worthless. It watched the Global ID
      counter: if the per-record path ran for already-mapped records it would
      allocate. It passed — and kept passing when the batched read was disabled
      entirely, because that path re-reads before allocating, finds the row,
      and never touches the counter. The counter detects re-allocation, not an
      N+1. It counts scans of `global_entities` now, and the same mutation says
      twenty-five scans where one is correct
- [x] document query-plan evidence when an index is added or rejected
      — `009_indexes.sql` held the reasoning, which is not evidence. The
      evidence is a test that seeds enough rows for the planner to have a
      choice and asserts the index is chosen. Seeding is the point: on an empty
      table PostgreSQL prefers a sequential scan whatever indexes exist, so a
      plan taken against an empty database proves nothing at all

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
- [x] NATS unavailable and recovery
      — closed by P28. The limit recorded here was the harness's, not the
      platform's, and it was the wrong limit: the test binary runs on the CI
      runner next to the Docker daemon, so it could always have driven
      `docker compose` itself. See
      `TestTheCoreKeepsServingWithoutTheEventBusAndRecoversWhenItReturns`
- [x] PostgreSQL unavailable during readiness/startup where practical
      — closed by P28, both halves of it. See
      `TestTheCoreBecomesUnreadyWithoutItsDatabaseAndDescribesNothing` and
      `TestACoreStartingWithoutADatabaseNeverServesAndComesUpWhenTheDatabaseReturns`
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

- [x] documented env vars vs Compose/runtime env vars
      — `scripts/check-config-docs.py`, and it found seven variables the stacks
      read that `.env.example` never mentioned. One of them is
      `STAGE_PUBLISH_ADDRESS`, which is what decides whether stage is exposed:
      the file an operator copies documented `BIND_ADDRESS`, which does nothing
      in stage, and was silent about the variable that does. That is the same
      defect `deploy#12` fixed in the preflight, one place further along — the
      question was worth asking again. All seven are now documented, the
      optional ones commented out with their defaults, and a commented
      assignment counts as declared because that is how an optional setting is
      presented
- [x] documented published ports vs Compose rendered ports
      — same script. A document telling somebody to open a port nothing
      publishes costs them twenty minutes and some of their trust in the rest
      of the document. No drift today; the guard is what keeps it so
- [x] documented service names vs Compose service names
      — same script. The first version of the pattern read past the closing
      backtick and reported the prose word "must" as a service, from
      `docker compose up -d --build` followed by "must be smoke-tested". A
      pattern that reaches into the sentence around a command will keep finding
      services in English, so it is now anchored inside the backtick span
- [x] authentik blueprint groups vs mock identity groups vs RBAC seed mappings
      — already closed by `scripts/check-identity-groups.py`, which compares
      the blueprint, the identity mock and the platform's RBAC seed
- [x] Global ID prefixes/types vs implementation
      — `scripts/check-global-ids.py`, comparing the rules in CLAUDE.md against
      the counters seed in Integration Core, in both directions. A prefix
      documented but never seeded allocates nothing, and a prefix seeded but
      never documented is worse, because a Global ID that ships is permanent.
      It reads every migration, not the first: `SVC` is seeded by 002, and a
      version limited to 001 reports the service prefix as unseeded — verified
      by running it that way. Wired into the E2E job rather than Stage assets,
      because only that job checks out Core; anywhere else it would print a
      note and pass, which is the vacuous green this wave keeps finding
- [x] metric names/types documented vs emitted
      — already closed by the metrics contract test in Integration Core
- [x] adapter capability names vs registry/implementation/docs
      — `core#15`. The capability list is written twice, once in the adapter and
      once beside `disabledAdapter`, and nothing made them agree. A capability
      added to one and not the other makes the platform answer a different
      question depending on whether the integration happens to be configured:
      a caller reading `/adapters` on a deployment without Outline would be
      told the product cannot search documents, which is a statement about that
      deployment dressed as a statement about the product. The E2E stack cannot
      catch it — it configures every adapter, so it only ever sees the real
      lists — and the partial-deployment scenario reads the placeholder but
      asserts its status, not its capabilities
- [x] route names/endpoints in docs vs OpenAPI
      — `scripts/check-documented-endpoints.py`. An endpoint named in a runbook
      is followed by somebody at a keyboard, and a 404 from a documented path
      reads as a broken deployment rather than a stale document. Two shapes are
      deliberately not endpoints and would otherwise dominate the findings: a
      version prefix written as a rule (`/api/v1/*`), which is policy rather
      than a path, and an upstream's own path (`/api/v1/Account`), which
      belongs to EspoCRM. Only the second needs a list, because only it is
      indistinguishable by shape. Integration Core's own documentation is
      checked by Integration Core's CI: this repository is not the place to
      make another repository's docs fail
- [x] documented file/script paths exist
      — already closed by `scripts/check-doc-links.py`
- [ ] stale references to removed services/dependencies fail CI
      — partly: a removed service is caught the moment a document names it in a
      Compose command, and a removed file the moment a document links to it.
      A dependency dropped from a manifest but still described in prose is not

Definition of Done:
- at least the high-risk duplicated facts are machine-checked;
- a one-character drift in identity group/capability/metric names fails CI;
- docs are updated only where implementation is authoritative.

# P26 — Cross-repository compatibility gate

Priority: HIGH
Depends on: P22, P23, P25

Goal: a green repository must not silently depend on an incompatible sibling
`main`.

- [x] define the provider/consumer compatibility matrix for Integration Core,
      Deploy, HUB and Design System
      — `docs/COMPATIBILITY.md`, which names each pairing and what holds it,
      including the one held by nothing
- [x] run cross-repo checks against sibling `main` for pull requests where
      practical
      — the E2E job here and the contract job in HUB both resolve a matching
      sibling branch where one exists and fall back to `main` where none does
- [x] validate Integration Core + Deploy E2E together
      — already covered by the `Autonomous E2E` job, which checks out Core,
      builds the stack from that source and runs the scenario suite against it.
      Recorded rather than rebuilt: this is the pairing with the most at stake
      and it is validated in the real runtime, not against a description of it
- [x] validate HUB against the normalized API/OpenAPI contract
      — `hub#6`. The HUB's tests answer every request from a fake, which is the
      right tool for testing the HUB and the wrong one for deciding whether the
      platform serves a path: a fake answers a request for an endpoint that has
      never existed. A path renamed in the platform left that repository green
      and the failure waited for a browser
- [!] validate Design System package consumer build without consuming an
      unversioned `main`
      — nothing consumes the Design System. The HUB declares no dependency on
      it and imports neither its tokens nor its components, so the rule against
      consuming an unversioned `main` is satisfied in the only way it currently
      can be, and there is no consumer build to validate. Inventing a consumer
      to produce a green check would report a compatibility that nothing
      depends on. This needs an owner decision — whether the HUB should adopt
      the Design System — rather than more automation
- [x] make fallback-to-main behavior explicit and fail loudly when a requested
      sibling ref is missing in CI
      — the fallback was already logged, but a line in a long log is not a
      record. The run summary now names the Core ref and why it was chosen. As
      for failing loudly: failing whenever no paired branch exists would break
      every ordinary pull request, so it has to mean a ref asked for by hand.
      `workflow_dispatch` takes a `core_ref` input, and a value that cannot be
      resolved fails rather than validating against `main` and reporting it as
      the requested pair. HUB's gate draws the line once more: with Core absent
      entirely it fails, because a gate that passes when its provider is
      missing disappears exactly when CI is misconfigured
- [x] produce a compact compatibility manifest/report as a CI artifact
      — `scripts/compatibility-manifest.py`, published as `compatibility.json`.
      A green E2E run is a statement about a pair of commits and did not say
      which pair; a stale provider and the intended one looked identical from
      outside. The Core commit is read from the checkout the job is about to
      use, not from an environment variable, because the variable records what
      was asked for rather than what was resolved

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

- [x] re-run an independent review of `docker-compose.stage.yml`, preflight,
      smoke runner, release manifest and acceptance report generation
      — reviewed clause by clause. The stage overlay renders clean through the
      hardening checker with every bind read-only; the preflight refuses an
      exposure mistake and reads the variable that actually controls the stage
      mapping; the release manifest generates in CI; the acceptance report is
      produced by the smoke runner
- [x] ensure every remaining real-environment prerequisite is represented as a
      clear BLOCKED item rather than a guessed value
      — fifteen `[!]` items, all present and none carrying an invented value.
      `docs/HANDOFF.md` lists them in execution order
- [x] verify stage preflight cannot print secrets in success or failure paths
      — the bearer token travels in a `--header` argument, the response body
      goes to `/dev/null`, and no failure path echoes either. The way that
      would break is shell tracing, which echoes the whole invocation including
      the Authorization header; a test now refuses `set -x` in the preflight and
      both smoke runners
- [x] verify smoke checks are read-only and cannot mutate upstream systems
      — true by construction today: every call is a default GET with no body.
      Nothing held it, and adding `-X POST` to a check is a small edit that
      would look like more thorough smoke testing. A test now refuses a
      mutating method, a request body or an upload on any line that invokes
      curl. The first version of that pattern roamed the whole file and matched
      `[ -d "$dir/.git" ]` — a shell directory test — as if it were curl's
      `--data`; a pattern about a command has to be anchored to that command
- [x] verify rollback and backup/restore instructions match the actual current
      stack after P18-P26
      — checked against what this wave changed. The base stack is untouched:
      the only new service, `integration-core-no-outline`, exists solely in the
      E2E stack, and `docs/E2E-ENVIRONMENT.md` documents it. No rollback or
      restore instruction enumerates a service that no longer exists
- [x] verify all example hosts/IPs/credentials are reserved placeholders
      — they are: `.example` under RFC 2606, `.local`, and only loopback or
      unspecified addresses. Nothing held that either, so the check now refuses
      a hostname outside the reserved spaces and an address that is neither
      loopback, private, nor RFC 5737 documentation. A runbook is copied, and a
      real hostname in one sends somebody's traffic to a stranger
- [x] produce a final autonomous handoff report with exact owner actions in
      execution order
      — `docs/HANDOFF.md`. Ordered by what each item unblocks rather than by
      where it appeared in this file, because an ordered list without that is a
      queue rather than a plan. It also records the two things this wave could
      not demonstrate, so they are not mistaken for oversights

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

# P28 — Failure-injection E2E orchestration

Priority: CRITICAL
Depends on: P23

Goal: close the two remaining E2E resilience gaps by controlling dependency
lifecycle from CI rather than only observing a fully running stack.

- [x] add a CI/harness control surface that can stop/start only designated E2E
      dependency containers without giving the application containers Docker
      socket access
      — `e2e/lifecycle.go`. The surface is outside the stack: the test binary
      runs on the CI runner, next to the Docker daemon, and drives
      `docker compose` itself. No container is given the socket and the compose
      files are unchanged. P23 had recorded this as impossible from the
      harness; that was a wrong reading of where the harness runs, not a
      missing capability
- [x] stop NATS while Integration Core remains running
- [x] verify `/readyz` reports NATS degraded exactly as designed and the Core
      remains ready if that is the documented policy
      — it does. `TestTheCoreKeepsServingWithoutTheEventBusAndRecoversWhenItReturns`
      also reads three collections through the outage, because "degraded, not
      down" is a claim about the API rather than about the probe
- [x] restore NATS and verify automatic recovery without Core restart
      — it reconnects on its own. The scenario compares the core container's
      `StartedAt` across the outage, because a core the restart policy brought
      back also ends up reporting `nats: ok` and from the outside the two are
      the same HTTP 200
- [x] stop PostgreSQL while Integration Core remains running
- [x] verify `/readyz` becomes non-ready and does not leak DSN, host, user or
      driver details
      — it does not, and the check covers `/health` and an API error with the
      same list, since a DSN leaking from any of them is the same disclosure.
      The port is matched as `:5432`: a bare `5432` also occurs inside a hex
      request id, and a check that flakes is a check that gets removed
- [x] restore PostgreSQL and verify readiness recovers without Core restart
      — the pool reconnects; `StartedAt` is compared for the same reason
- [x] exercise startup with PostgreSQL initially unavailable, where practical,
      and document whether retry/restart policy or fail-fast startup is the
      intended contract
      — **fail-fast**, now written down rather than inferred. `main` calls
      `log.Fatalf` when it cannot open the database, so the process exits
      instead of serving while unable to answer anything, and recovery belongs
      to the restart policy. The scenario pins both ends: no window in which
      such a core answers `/readyz` with 200, and no intervention needed once
      PostgreSQL returns. The consequence for a deployment is now in
      `docs/STAGE-ACCEPTANCE.md` — a Core deployed *without* a restart policy
      leaves a stopped container behind after a database blip rather than a
      core that waits
- [x] make lifecycle tests deterministic, bounded and impossible to silently
      skip in `E2E_REQUIRED=1` CI
      — every wait is bounded and reports its last observation, so a timeout
      names what it saw instead of becoming a re-run. `E2E_LIFECYCLE` is opt-in
      locally and mandatory in CI: with `E2E_REQUIRED` set and `E2E_LIFECYCLE`
      empty, `TestMain` refuses to start. That is the same silent skip
      `required_test.go` exists for, one level down — without it these
      scenarios would skip on a runner with no Docker and the job would exit
      zero having stopped nothing

Findings:

- what may be stopped is an allowlist, not an argument. The primary
  Integration Core is deliberately absent from it: every other scenario in the
  package asserts against that core, and a restart reaching it from a lifecycle
  test would surface as an unrelated failure somewhere else in the suite. The
  allowlist and the skip guard are plain functions, so they are exercised by
  `go test ./...` on a machine with no Docker at all — a guard tested only
  where the stack is up is tested exactly where it is least needed
- the guards were verified by mutation: allowing every service, and allowing a
  required run with no lifecycle control, each fail the tests that cover them

Definition of Done:
- P23's NATS and PostgreSQL outage boxes can be closed with executable evidence;
- no production credentials or privileged application container is introduced;
- expected degraded/unready semantics are pinned by tests;
- Autonomous E2E and Security are green.

# P29 — Event delivery reliability contract

Priority: CRITICAL
Depends on: P28

Goal: make event delivery guarantees explicit and executable rather than
implicit in a successful NATS publish call.

## P29.1 Semantics decision from existing architecture

- [x] inventory every event publisher and consumer contract
      — seven publishers, and the inventory found two defects rather than a
      clean picture. **Three of the seven did not use the subject
      convention**: `identity.created`, `service_identity.created` and
      `global_id.created` went to bare subjects instead of `bsystem.events.*`,
      so a consumer subscribed to `bsystem.events.>` — which the convention,
      the documentation and the E2E harness all use — had never received one
      of them. The metric counted them published; nothing conforming could
      see them. **The Global ID handler announced every allocation call**,
      including the two paths that return an identifier which already existed,
      so asking twice for one source record announced two creations of a thing
      created once
- [x] document current delivery semantics: loss window, duplication window,
      ordering scope and behavior while NATS is unavailable
      — a table of both, in `bsystem-integration-core/docs/EVENTS.md`
- [x] decide, based on current product invariants, which events may be
      best-effort and which require durable delivery; do not invent a business
      promise
      — decided by one question, and no new promise: **can a consumer that
      missed this event recover what it says by reading the platform
      afterwards?** For the support and operations events, yes — the record
      stays readable, so a missed event costs a notification and not a fact.
      For the three allocation events, no: each announces something that
      happens exactly once in the lifetime of a subject, no later event
      restates it, and a consumer cannot tell "I missed it" from "it never
      happened"
- [x] give every durable event an immutable event ID and stable occurred-at
      time
      — `event_id` on every envelope, best-effort ones included, minted by the
      platform and never by the publisher: a caller-supplied id would let one
      service claim another's and make a consumer discard a real event as a
      duplicate. `occurred_at` is when the thing happened, so an event
      delivered after a two-hour outage does not sort as though it happened
      after it
- [x] document idempotency expectations for consumers
      — one sentence of contract: a consumer that sees the same `event_id`
      twice has seen the same event twice. The platform does not promise
      exactly-once delivery; it promises an identifier that makes exactly-once
      *processing* possible

## P29.2 Durable path where required

- [x] if any existing event is required for correctness/audit/notification
      state, implement a transactional outbox or equivalent DB-backed durable
      queue
      — migration `010_event_outbox.sql` and `internal/platformdb/outbox.go`.
      The row is written **inside the transaction that caused it**, which is
      the property worth having: it exists if and only if the allocation
      committed. An event cannot be published for a transaction that rolled
      back, and an allocation cannot commit while its announcement is lost to
      a broker that happened to be down. Nothing is published from the request
      path
- [x] publish outbox rows to JetStream with bounded retry/backoff
      — exponential and capped, `MaxOutboxAttempts` in total. A row that
      exhausts its budget is marked failed and **kept**: the point of a
      durable event is that somebody can still find out it was never delivered
- [x] mark delivery only after broker acknowledgement
      — which is why it is JetStream and not core NATS. `nc.Publish` returns
      when the bytes reach a socket buffer; treating that as delivery would
      put a durable table in front of a silent loss
- [x] tolerate duplicate delivery by immutable event ID
      — the event id is also the `Nats-Msg-Id`, so the broker collapses the
      one case retrying cannot fix: an acknowledgement lost on the way back,
      where the event is stored and the outbox believes it is not. Outside the
      broker's duplicate window, `event_id` is the consumer's own defence
- [x] recover undelivered rows after process restart
      — by construction rather than by bookkeeping. Claiming schedules the
      next attempt **before** handing the row out, so a publisher that dies
      mid-delivery leaves the row due again after its backoff. There is no
      crash detection to get wrong
- [x] metrics for queued, delivered, retrying and permanently failed events
      — `bsystem_event_outbox_events{state}` and
      `bsystem_event_outbox_attempts_total{subject,outcome}`, both pinned by
      the metrics contract test. The outcome worth an alert is
      `ack_not_recorded`: the broker has the event and the platform could not
      write that down
- [x] deterministic DB/NATS failure tests
      — a broker that refuses on demand, which a real one will not do
      reliably; plus the E2E scenario that stops the real broker, mints a
      Global ID, and watches the queue drain when NATS returns

Findings:

- the outbox needed one behaviour change outside itself. NATS connected
  without `RetryOnFailedConnect`, so a Core that booted a second before the
  broker held a nil connection for the rest of its life: every event counted
  "unavailable" and dropped, `/readyz` reporting `nats: degraded` forever, and
  only a restart fixing it. The outbox would have queued behind a broker this
  process had decided did not exist
- verified by mutation, each confirmed to land: announcing on the
  already-existed path fails the duplicate test; an enqueue on a separate
  connection survives a rollback and fails the transaction test; marking
  delivered without an acknowledgement fails the failure test; dropping the
  immediate follow-up pass leaves a backlog undrained
- the first CI run failed and the test was the thing that was wrong. The
  delivery counter is per process and the stack runs two Cores against one
  database, so a row claimed by the second Core is counted there and nowhere
  else. A scenario watching only the core it made its request to misses every
  delivery the other one did. Taking disjoint rows is the property that makes
  two publishers drain faster instead of delivering twice; the counter is per
  process and the queue is not. The scenarios sum across both
- the E2E scenarios assert through `/metrics` rather than by subscribing to
  the bus. A core NATS subscriber only receives what is published while it is
  subscribed, so a test that stops the broker, produces an event and
  subscribes again is racing the publisher's next pass — and a race that
  usually wins is a test that occasionally fails for a reason nobody can
  reproduce

Definition of Done:
- every event category has an explicit delivery guarantee;
- durable events survive broker outage and process restart without silent loss;
- duplicates are safe by contract;
- no event payload gains credential/secret data;
- docs and metrics describe the implemented guarantee, not an aspiration.

# P30 — Audit durability policy

Priority: CRITICAL
Depends on: P29

Goal: define which successful operations are allowed to exist without a durable
audit record, then enforce that policy consistently.

- [x] inventory all audit-producing read and mutation paths
      — eight: two authorization mutations, three business mutations, two
      sensitive reads and one informational. Listed in
      `bsystem-integration-core/docs/AUDIT.md`
- [x] classify audit events into security-critical mutation, business
      mutation, sensitive read and informational/operational categories
- [x] define fail-closed vs fail-open behavior for each category from existing
      security invariants; owner/business decisions remain BLOCKED rather than
      guessed
      — decided by one question: **if this record is missing, can anyone
      afterwards establish that the action happened, and who asked for it?**
      For a notification marked read, an incident raised or a Global ID
      allocated the answer is yes — the record is still there and the audit
      row is a convenience for reading a trail. For a scope grant it is no:
      `principal_scopes` holds the grant and not its provenance, so a grant
      that took effect without a record is indistinguishable from one nobody
      made. Nothing here is an owner decision waiting to be taken; each row
      follows from an invariant the platform already has. Retention *is* a
      business decision and is `[!]`
- [x] make security-critical mutations and their audit record atomic where
      technically possible
      — `AddScopeGrant` and `DeleteScopeGrant` write the change and its record
      in one transaction. Either both rows exist or neither does. It is
      possible only because both live in the same database, and pretending
      otherwise elsewhere would mean a distributed transaction across systems
      that do not have one
- [x] where atomicity is impossible, make audit failure visible through a
      durable/observable failure state rather than log-only behavior
      — the AI audit was the last path whose failure was log-only. An AI
      request that touched authorized content and left no record is the audit
      hole nobody can close afterwards, and it looked exactly like a quiet day
      on every dashboard
- [x] add metrics and alerts for audit persistence failures
      — `bsystem_audit_writes_total{action,outcome}`, and two outcomes that
      need different people: `failed` is an action that happened and is not
      attributable, `refused` is an action that did not happen. The alert
      expression is in `bsystem-integration-core/docs/AUDIT.md`
- [x] prove a mutation cannot return success when its policy requires a
      durable audit and the audit write fails
      — against an audit table made to refuse writes by a trigger, which is
      deterministic and confined to the test's own throwaway database. The
      assertion is not the 503: it is that the grant **does not appear in the
      principal's scopes**. A 503 that left the scope granted would be worse
      than a 201, because nobody would go looking for it
- [x] prove low-risk paths do not become globally unavailable from a
      noncritical audit sink failure when policy says fail-open
      — the same broken table, and an audited read plus an ordinary request
      both still answer 200 with the failure counted. Fail-closed applied
      everywhere would turn a bookkeeping failure into an outage, which is its
      own kind of failure

Findings:

- verified by mutation, each confirmed to land: writing the audit record
  outside the grant's transaction makes the fail-closed test report a 201 and
  a granted scope; swallowing the insert error makes the fail-open test find
  no counter
- the refusal carries no database detail. This endpoint decides who may see
  what, and a raw error here names tables and constraints
- [!] audit **retention** — how long these records are kept, and who may read
      them — is a business decision and is not made here

Definition of Done:
- audit durability behavior is explicit per operation class;
- tests fail when a required audit record is dropped;
- no raw DB/audit error leaks to clients;
- documentation, metrics and behavior agree.

# P31 — Local JWT/JWKS authentication hardening

Priority: HIGH
Depends on: P30

Goal: remove per-request network dependence on authentik UserInfo for token
validity while preserving authentik as the issuer and source of identity.

- [x] inspect the exact token/claims contract currently expected from authentik
      — `sub`, `email`, `name`, `preferred_username`, `groups`, read from the
      UserInfo response. The same five are what the token must carry, so the
      authorization boundary is unchanged by where they come from
- [x] implement OIDC discovery and JWKS retrieval with bounded timeout
      — `internal/oidc`. The discovery document must name the issuer it was
      fetched from: one that names somebody else is a misconfiguration or a
      redirect somebody arranged, and following it means taking keys from
      whoever answered
- [x] validate signature locally
- [x] validate issuer, audience/client, expiry, not-before and algorithm policy
- [x] support JWKS cache with bounded TTL and refresh on unknown `kid`
- [x] handle signing-key rotation without requiring Core restart
      — and the old key keeps working while the provider still publishes it,
      which is what makes a rotation a rotation rather than a cutover
- [x] define safe clock-skew tolerance
      — 60s by default, and it is a tolerance rather than an extension: a
      token ten seconds past expiry is accepted, one five minutes past is not
- [x] reject `alg=none`, unexpected algorithms and malformed claims
      — including the one worth naming: a token signed HS256 with the
      provider's own **published public key**, which verifies against a
      verifier careless enough to treat the key as a shared secret
- [x] decide whether groups/claims come from token, UserInfo enrichment or
      both; preserve the existing authorization boundary
      — the token, with UserInfo as enrichment only when the token carries no
      `groups`. Some authentik configurations serve groups from UserInfo only,
      and that deployment still works
- [x] if UserInfo remains for enrichment, make its outage unable to turn an
      otherwise invalid token valid or broaden permissions
      — two rules. A failure is not an error: the request proceeds with no
      groups, which is no role, which is deny-by-default. An unreachable
      UserInfo can cost a caller their access and can never give them somebody
      else's. And an answer about a different subject is discarded
- [x] fake-OIDC tests for valid token, expired token, wrong issuer, wrong
      audience, wrong algorithm, unknown kid, rotated key and discovery/JWKS
      outage
      — thirteen refusals in one table, plus the outage from both sides: a
      provider that goes away cannot revoke a valid token while the keys are
      held, and cannot make an expired one valid
- [x] update authentik stage documentation with exact claims required
      — `bsystem-integration-core/docs/AUTHENTICATION.md`, and the variables
      in `docs/STAGE-ACCEPTANCE.md` and `.env.example`

Findings:

- **there must be no downgrade between the two modes.** A platform configured
  to validate locally refuses a token it cannot verify rather than asking
  UserInfo. The fallback would restore the network dependency the
  configuration exists to remove, reachable by anybody who sends something
  that is not a JWT
- writing the rotation test found the tension the refresh sits in. Refreshing
  on an unknown `kid` is what makes rotation work; doing it on every unknown
  `kid` turns a stream of invented ids into a load generator pointed at the
  platform's own identity provider. The first version of the rate limit also
  blocked the rotation, because a cache refreshed a moment earlier for
  ordinary reasons counted against it. A rotation refresh and a cache fetch
  are different events and are tracked separately now
- 401 and 503 are different answers. A provider outage answered 401 sends
  every signed-in person to the login page during an incident that has nothing
  to do with their session, where they will fail to sign in as well
- local validation is **not the default**: it requires authentik to issue JWT
  access tokens. A deployment whose provider issues opaque ones is not broken
  by this being available, it simply does not set `OIDC_ISSUER_URL`

Definition of Done:
- request authentication does not require a successful UserInfo network call
      when the token itself carries all required validated claims;
- key rotation works in tests;
- authorization behavior is unchanged or more restrictive;
- no token or signing material is logged.

# P32 — Rate limiting and abuse controls

Priority: HIGH
Depends on: P31

Goal: bound expensive or abuse-prone request classes without introducing a
shared cache until a distributed requirement is demonstrated.

- [x] inventory public/human, machine/service, AI, search and expensive adapter
      routes
      — taken from the route table, which is already the authoritative
      inventory, rather than listed a second time beside it. A route added
      there falls into a class from its authentication kind, so a new endpoint
      is limited by default rather than unlimited until somebody remembers it;
      `TestEveryAuthenticatedRouteFallsIntoAClass` enforces that
- [x] implement a small per-instance limiter abstraction with separate policy
      buckets for human API, service API, AI and search/expensive routes
      — `internal/ratelimit`, a token bucket with a bounded key table. The AI
      class is the tight one: one request there costs a model call and a
      fan-out of authorized reads rather than a query
- [x] key human limits by validated principal rather than untrusted headers
- [x] key machine limits by service identity
      — both by Global ID, and the middleware runs **after** authentication so
      the principal exists, **before** authorization because refusing early is
      the cheaper half of the point
- [x] return normalized `429` with bounded `Retry-After`
      — at least one second, because a caller told to wait for nothing retries
      immediately and is refused again; never longer than a full refill,
      because a bound longer than that is a caller giving up rather than
      backing off
- [x] never include tokens, usernames, email or raw client IP in metric labels
- [x] metrics for allowed/rejected requests with bounded route/class labels
      — class and outcome, and deliberately nothing else. A principal is one
      time series per person or machine identity, and a username or an address
      would put *who is being throttled* into a store read far more widely
      than the audit trail. Asserted in both the unit and the E2E scenario
- [x] tests for burst, refill, cancellation and independent principals
      — the refill tests drive the clock rather than sleeping: a test that
      waits is measuring the runner as well as the rate. "Cancellation" is
      covered as the property that matters here — the limiter never blocks. A
      limiter that waits for a token has turned a rejection into latency,
      which is the same exhaustion the limit exists to prevent, held open one
      goroutine at a time
- [x] E2E proving one noisy identity does not throttle another
      — against the **spare** core, with its search allowance turned down in
      the E2E Compose file. Spending a burst on the core every other scenario
      talks to would refuse a later scenario's search for a reason that has
      nothing to do with what it is testing, and the failure would look like a
      platform defect
- [x] document the limitation of per-instance enforcement
      — `bsystem-integration-core/docs/RATE-LIMITS.md`
- [x] add Redis/distributed coordination only if an explicit multi-replica
      requirement cannot be met safely without it; do not reintroduce unused
      infrastructure speculatively
      — **not added, deliberately.** A shared counter makes the limit exact
      and adds a dependency whose outage has to be answered: fail open and the
      limit is gone at the moment it is most likely to be needed, fail closed
      and the platform is gone. Nothing here has demonstrated the need for an
      exact limit, and an approximate one enforced n times for n replicas
      still bounds abuse by a factor of n. Recorded as an explicit future
      decision rather than left implicit

Findings:

- the bucket table has to be bounded. A platform whose principals are machine
  identities minted per job would otherwise grow a bucket per job and release
  none. The cost of the bound is that a very busy platform may forget a bucket
  early and that principal starts full — the limit is approximate by design,
  and this is one of the ways
- verified by mutation, each confirmed to land: keying every request the same
  makes the noisy-neighbour test refuse the quiet principal on its first
  request; putting the operational endpoints in a class fails the
  classification test on `/health` and `/readyz`

Definition of Done:
- expensive surfaces have bounded abuse behavior;
- 429 behavior is in OpenAPI and HUB error handling where applicable;
- no high-cardinality/sensitive labels are added;
- distributed rate limiting remains an explicit future decision unless proven
      necessary.

# P33 — Immutable release artifact pipeline

Priority: CRITICAL
Depends on: P28-P32

Goal: ensure the artifact scanned, described and promoted is the exact artifact
that CI tested, not a later rebuild from the same source.

- [x] define releasable artifacts for Integration Core and HUB; keep mocks/test
      images separate from product artifacts
      — the two product images are release artifacts; the mocks and the two Go
      harnesses are not, and the E2E override deliberately leaves the mocks
      building from source. A manifest that described a test fixture would be
      describing something no deployment runs
- [x] build each release image once per commit
- [x] tag by immutable commit SHA and record image digest
      — a commit SHA is the only name that cannot later be reused for
      something else. A local image has no registry digest until it is pushed,
      so the identity recorded is the **image ID**: the sha256 of its
      configuration, which covers its layers, is immutable for a given build,
      and changes if anything about the image changes
- [x] use the exact built image for runtime/E2E validation where practical
      — `docker-compose.e2e.images.yml`, and the absence of `--build` is the
      whole mechanism: with an `image:` set and no `--build`, Compose uses what
      is there rather than making another one
- [x] scan that exact image with Trivy rather than a rebuild
- [x] generate SBOM from that exact image and bind it to its digest
- [x] generate provenance/attestation with repository, commit, workflow run and
      digest; do not add signing credentials unless owner-configured
      — unsigned, and `provenance.json` says so in the document itself rather
      than only in the documentation: it records origin, not authenticity. A
      signature needs a key the owner has not configured
- [x] publish release manifest as CI artifact containing image digests, schema
      level, OpenAPI hash, HUB/Core commits and compatibility manifest
      — `release-manifest.sh` gained an `images` block, read from the file
      written at build time rather than inspected when the manifest is
      generated. Those are the same thing only if nothing rebuilt in between,
      and "only if" is what the pipeline exists to remove
- [x] verify a digest mismatch between tested/scanned/reported artifacts fails
      CI
      — two checks, not one. The identities are re-read at the end and
      compared with what was built, and the generated manifest is compared
      with the same file, so a manifest that described a different image would
      fail even if nothing had been rebuilt
- [x] document promotion flow without performing a production deployment
      — `docs/RELEASE.md`. Steps 4 to 6 need a registry credential and a
      deployment target and are `[!]` below

Findings:

- a check that only ever sees matching identities has never been shown to
  detect a mismatch, so `scripts/image-digests.py` is exercised from both
  directions with the readings supplied from files — reproducing the failure
  with real images would mean corrupting one. An unchanged pair verifies, a
  rebuilt image is caught **by name**, a vanished one is caught, and a
  manifest recording no image at all is refused rather than trivially verified
- **the first run of the release pipeline found a real vulnerability**, which
  is the argument for the pipeline existing. The Security workflow scans one
  mock image, on the reasoning that the four mocks share a Dockerfile — and
  nothing had ever scanned the image the platform actually ships. The first
  scan of it found CVE-2026-14456 in `libssl3` and `libcrypto3`, fixed
  upstream and still waiting for the `alpine:3.22` tag to move. Both product
  Dockerfiles now `apk upgrade` rather than only `apk add`: the packages that
  carry vulnerabilities in an image like this are the base image's own, and
  `apk add` does not touch them
- **the HUB image this pipeline builds is not deployable.** The HUB bakes its
  OIDC issuer and client id in at build time, so a release image is specific
  to the authentik it was built for. The pipeline proves the image builds,
  scans clean and is described; a deployable one is built by the owner with
  their own issuer, and its identity will differ from the manifest's. That is
  recorded rather than papered over with a placeholder nobody reads
- [!] pushing the images to a registry needs a registry credential, and
      deploying them needs a target. Both owner-only; see `docs/RELEASE.md`
      and `docs/HANDOFF.md`

Definition of Done:
- one immutable digest identifies what was tested, scanned and described;
- SBOM/provenance refer to that digest;
- a rebuild cannot silently substitute for the tested image;
- no registry write credential is required unless publishing is explicitly
      owner-enabled.

# P34 — Versioning and compatibility policy

Priority: HIGH
Depends on: P33

Goal: turn existing cross-repo compatibility checks into an explicit release
contract.

- [x] define versioning policy for Integration Core HTTP API
- [x] define what constitutes additive vs breaking API change
      — both in `bsystem-integration-core/docs/VERSIONING.md`, with the part
      that usually goes unsaid said: the `error` text is prose and may change
      freely, the `code` is what clients switch on and may not. And
      **tightening authorization is not a breaking change** — a caller who
      could reach something they should not, and now cannot, is a defect being
      fixed
- [x] define OpenAPI version/source-of-truth policy
      — the document is the source of truth and the implementation is tested
      against it, rather than the document being generated from the code. The
      direction is the point: a document generated from the code cannot
      disagree with the code, which makes it useless as a check
- [x] define schema compatibility level and minimum/maximum supported
      migration direction for a release
      — a release supports the level it embeds and every earlier one it can
      migrate forward from, which is executed by
      `TestUpgradeWorksFromEveryHistoricalSchemaLevel` rather than asserted.
      There is no downward migration, and a newer database under an older Core
      is **unsupported** — it usually works, because the schema is additive,
      and "usually" is not a support statement
- [x] define HUB ↔ Core compatibility declaration without inventing a consumer
      version matrix that is not tested
      — a **validated pair**, not a range. "These two commits were verified
      together" is true and checkable; "this HUB works with Core 1.2 through
      1.7" is a matrix nothing has tried
- [x] expose build/release version metadata consistently through metrics
      and/or a safe version endpoint
      — already exposed, and left as it is deliberately. `bsystem_build_info`
      carries version, commit, build time and embedded schema level;
      `bsystem_schema_migrations_applied` carries the level the database is
      actually at, and the two disagree exactly when a database is behind its
      code. No endpoint was added to report the OpenAPI hash: the commit label
      names the exact document, and embedding a second copy in the binary
      creates two places that can disagree
- [x] make release manifest record compatibility requirements
      — a `compatibility` block naming both API versions, the migration
      direction, the unsupported combination and where the policy lives
- [x] add CI fixtures proving an additive API change passes and an
      intentionally breaking provider change fails the consumer gate
      — `bsystem-hub/scripts/tests/contract-gate.test.sh`, run in the HUB's
      contract job. The gate had only ever been run against a specification
      and a source tree that agree, so it had never been shown to catch a
      change that breaks them
- [x] document emergency rollback constraints when a release includes an
      irreversible schema change; do not add an irreversible migration here
      — a release with only additive migrations rolls back by redeploying the
      previous image. One with a destructive migration does not: restoring the
      older code against a database that has had a column dropped means
      restoring the database too, from a backup taken before the migration,
      which loses everything written since. So a destructive change is a
      two-release change, never one. **No irreversible migration exists today
      and none was added to write that section**

Findings:

- the policy says, for each rule, whether a machine checks it. A compatibility
  policy nothing enforces is a description of intentions
- and one gap is recorded rather than implied by the absence of a check:
  **field-level additive-versus-breaking is not checked.** Nothing compares
  this release's schemas against the previous release's, so removing a
  response field passes CI. The path gate covers paths, not fields and not
  error codes
- the consumer gate's own test includes the two vacuity cases, because a gate
  that passes when it compared nothing is the failure it exists to prevent: a
  consumer requesting no path, and a specification with one path in it, are
  both refused

Definition of Done:
- a release states which Core/HUB/schema/OpenAPI combination it represents;
- breaking-change rules are machine-tested where practical;
- compatibility language matches what CI actually verifies.

# P35 — Automated backup/restore verification

Priority: CRITICAL
Depends on: P33, P34

Goal: prove restoreability in an ephemeral environment rather than only maintain
a runbook.

- [x] create deterministic fixture data covering identities, service
      identities, Global IDs/mappings, RBAC/scopes, audit, notifications,
      operations and support records
      — written **through the platform's own API**, not as SQL. What is backed
      up is then what the platform actually produces rather than rows a test
      invented, and a restore that loses a shape the platform writes but a
      fixture did not would still be caught
- [x] take a PostgreSQL logical backup in CI/E2E using test-only credentials
      — `pg_dump` inside the database container, which is on an internal
      network and deliberately not published: a scenario that needed the port
      opened would have weakened the isolation it is testing against
- [x] destroy/recreate the ephemeral database
      — and the scenario checks the platform **cannot** serve the record at
      that moment. Without that, the restore below would prove nothing: the
      data could be in a cache
- [x] restore the backup
- [x] start the application against the restored database
      — it was never stopped. The pool reconnects on its own, which is the
      better contract: a restore that needed the process restarted to be
      usable is a different and worse one
- [x] verify schema migration history/checksums remain valid
      — applied is non-zero and **drifted is zero**. A restore that lost the
      bookkeeping looks healthy until the next deployment reapplies a
      migration onto a schema that already has it, and drift is the only
      series that can see a restored database reporting the right level while
      holding a different schema
- [x] verify representative Global IDs are unchanged
      — the assertion the scenario exists for. A Global ID that changed across
      a restore is a platform that has silently renamed every customer's
      records. The **counter** is checked too: one restored to zero would mint
      an identifier that already belongs to something else, which is worse
      than losing the row
- [x] verify RBAC/scopes, notifications, support and audit data survive
      — read back through the API rather than counted in the tables. A row
      that survived and that the platform can no longer serve is not a
      successful restore, and comparing tables to tables would not notice
- [x] verify secrets are not written into backup artifacts by the application
      layer; do not publish DB backup artifacts outside the CI job
      — the dump is searched for every upstream API key and the database
      password. A dump is copied to laptops, attached to tickets and kept for
      years, so it is the worst possible place for a credential to appear, and
      this is the only check that would notice if a future change started
      storing them. The dump is never uploaded as an artefact and the stack is
      destroyed with its volumes
- [x] exercise restore followed by upgrade to current schema when practical
      — partially, and the limit is recorded rather than glossed. The restored
      database is at the current level, so what this executes is that a Core
      accepts it and reports no drift. A restore from an **older** level
      followed by an upgrade is covered one layer down, by Core's
      `TestUpgradeWorksFromEveryHistoricalSchemaLevel`
- [x] document what this proves and what still requires real infrastructure
      backup acceptance
      — `docs/BACKUP-RESTORE.md` gained both halves. What it proves: the
      procedure is sound. What it does not: a backup taken from a real
      deployment at real size, the backup **target** and whether anything can
      be read back from it, authentik's database (which the E2E stack replaces
      with a mock), and the restore **window** — how long a real restore takes,
      which is the number an incident actually needs

Findings:

- the scenario runs in its own CI job with its own stack. Dropping the database
  underneath the other scenarios would turn one failure here into a wall of
  failures everywhere, and the one that mattered would be lost in it
- it carries the same no-silent-skip guard as the other two: the job exists to
  run one scenario, so a run that skipped it and exited zero would be the worst
  possible outcome — a green check that restored nothing
- the first CI run failed on the audit comparison, and the test was what was
  wrong. Reading a Global ID is itself an audited action, so the read that
  takes the "before" snapshot writes a row and the read that takes the "after"
  one writes another: a count comparison was measuring its own observation and
  was off by exactly one, every time. It compares the entries by id, action,
  resource and request id now, and asserts that every entry which existed
  before the backup is still there. New entries after a restore are expected —
  the platform is still being used

Definition of Done:
- CI proves a real database backup can be restored into a fresh PostgreSQL and
      accepted by the current Core;
- identity/authorization identifiers remain stable;
- test backup is ephemeral and not retained as a downloadable artifact;
- production backup target/retention remains owner-controlled.

# P36 — Release candidate freeze and stage handoff

Priority: CRITICAL
Depends on: P28-P35

Goal: stop expanding the platform foundation and produce one reproducible
release-candidate set for real stage acceptance.

- [x] perform a final independent invariant/security review of changes from
      P28-P35
      — against the ten architectural invariants, and it found one. The OIDC
      verifier refused a discovery document that named a *different* issuer,
      and then followed whatever `jwks_uri` that document contained, wherever
      it pointed. A misconfigured provider — or an answer from something else
      on the network — could send the Core to any address it can reach and, in
      the worst case, have it load signing keys from there, after which a
      token signed with those keys verifies. The key set must now be on the
      issuer's own origin. Every OIDC provider publishes it there, so a
      correct deployment pays nothing
- [x] ensure all non-owner-dependent P28-P35 work is DONE or explicitly
      BLOCKED for a real reason
- [x] ensure Integration Core, HUB, Design System and Deploy `main`
      CI/Security are green
- [x] run main-to-main compatibility and Autonomous E2E against the exact RC
      commits
      — the `Release artifacts` run on `main` resolves both product
      repositories to `main`, builds the images once and runs the E2E suite
      against the image it built, so the main-to-main validation and the
      artifact validation are the same run rather than two that might disagree
- [x] generate final compatibility manifest
- [x] generate final immutable artifact/release manifest with image digests,
      commits, schema level and OpenAPI hash
      — both are CI artifacts of that run rather than files in the
      repository, which is the point: a manifest committed to git describes
      what somebody wrote down, and one produced by the run that built the
      images describes what was built
- [x] update `docs/HANDOFF.md` with exact RC commits/digests and owner actions
      — with the commits as a signpost and the manifest named as the record.
      A table in a document drifts from the artefact the moment either moves
- [x] verify stage preflight/smoke scripts against the current RC
      configuration
      — and the preflight gained a check for the one new configuration
      mistake that looks like a working deployment: an issuer set with no
      audience accepts a token minted for any client of that issuer. Writing
      it also produced a **wrong** check — "neither authentication mode is
      configured" — which failed two existing tests because the base Compose
      file supplies `AUTHENTIK_USERINFO_URL` itself. The check was wrong, not
      the tests; the correct behaviour is now pinned so it is not written
      again
- [x] explicitly freeze further autonomous foundation feature work after P36;
      new foundation changes require a defect, failed acceptance check or
      owner decision
      — stated in `docs/HANDOFF.md`. "It would be better if" is not on that
      list: the platform is at the point where the next thing it needs is
      contact with reality, and more autonomous building delays that rather
      than helping it
- [!] real authentik/EspoCRM/Redmine/Outline credentials, customer mapping,
      SLA policy, DNS/TLS, a container registry and production deployment
      remain owner-only. `docs/HANDOFF.md` lists them in execution order

Findings:

- the wave's first Release artifacts run **found a real vulnerability in the
  product image**, which is the argument for the pipeline existing at all. The
  Security workflow scanned one mock image, on the reasoning that the four
  mocks share a Dockerfile, and nothing had ever scanned the image the
  platform ships. Both product Dockerfiles now `apk upgrade` rather than only
  `apk add`
- three tests in this wave failed CI and were themselves the thing that was
  wrong: the outbox delivery counter is per process and the stack runs two
  Cores; the audit comparison counted its own observation, because reading a
  Global ID is an audited action; and the preflight check above. Each is
  recorded where it was fixed rather than quietly corrected

Definition of Done:
- one exact RC set is reproducible from GitHub and CI artifacts;
- all autonomous checks are green for that set;
- owner receives an ordered real-stage acceptance handoff;
- no production/stage deployment is performed autonomously.

## P28-P36 execution rule

This is the final autonomous foundation/release-readiness wave before real stage
acceptance. Do not add unrelated business modules or speculative infrastructure.

A task may be marked `[x]` only when:

```text
1. implementation and tests are committed/merged in GitHub;
2. relevant CI, Security and cross-repository checks are green;
3. the protected property is demonstrated non-vacuously where practical;
4. OpenAPI/docs/migrations/metrics are synchronized when affected;
5. TASKS.md records findings, limitations and new owner-only BLOCKED items.
```

If a task requires production/stage credentials, a commercial/SLA decision,
customer ownership mapping, registry credentials, DNS/TLS changes or another
owner-only decision, mark that part `[!]` and continue with every independent
safe item.

After P36, stop autonomous foundation expansion. The next phase is real stage
acceptance against authentik, EspoCRM, Redmine and Outline.

**Frozen.** P28-P36 are complete and merged. Further foundation work requires a
defect, a failed acceptance check or an owner decision; defect fixes, security
patches and answers to failed acceptance checks continue as normal. See
`docs/HANDOFF.md`.

## Owner-requested operational work

### Human account administration

- [x] add normal BSYSTEM-HUB administration for human authentik accounts
      — owner-requested after the P36 foundation freeze, so this is an explicit
      owner decision rather than autonomous foundation expansion
      — Integration Core PR #24 adds the server-side authentik boundary,
      account list/create/update/password endpoints, persistent Global User ID
      joining, `identity.user.manage`, an Administrator-only target boundary,
      service-identity exclusion and the final-active-administrator guard
      — HUB PR #11 adds create, role, enable/disable and password-reset controls
      while keeping Global User IDs read-only and hiding mutation actions from
      callers without `identity.user.manage`
      — Deploy PR #41 provisions a dedicated non-superuser authentik service
      account/RBAC role and consumes `BSYSTEM_AUTHENTIK_ADMIN_TOKEN` only in
      the authentik worker and Integration Core; an empty token leaves the
      directory read-only
      — destructive delete remains intentionally absent: the persistent
      Integration Core identity, immutable `USR-*` identifier and audit/history
      require an explicit lifecycle policy before deletion is safe
      — the final-active-administrator lock serializes mutations initiated
      through Integration Core replicas; direct authentik UI/API changes do not
      participate in that PostgreSQL advisory lock and can bypass the BSYSTEM
      guard
      — PR checks and the post-merge main checks are green for Integration
      Core, HUB and Deploy, including Security, Autonomous E2E,
      backup/restore and immutable Release artifacts

