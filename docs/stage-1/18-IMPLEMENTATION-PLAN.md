# Stage 1 Implementation Plan

This plan translates the Stage 1 documentation and the live Integration Core
audit into implementation workstreams.

## 1. Baselines

### QA

```text
repo: bsystem-code/qa
branch: main
sha: 803a49f11c14c1ce181b1be41f8c626c21cc1407
```

### Integration Core

```text
repo: ekucher/bsystem-integration-core
branch: main
sha: dd2c7f90c887183db9392fdf6db6b980c2adccbf
CI: success
Security: success
```

Before implementing, re-fetch both repos and re-freeze exact working SHAs.

## 2. Workstream A — Integration Core relationships

Suggested branch:

```text
stage1/relationships
```

### A1. Add Requirement Global ID

Append-only migration:

```text
requirement -> REQ
```

Tests:

- first allocation;
- repeat allocation idempotence;
- concurrency;
- source separation.

### A2. Add relationship persistence

Create append-only migration and store layer.

Required behaviors:

- one canonical stored edge;
- no self-edge;
- controlled relation type;
- idempotent duplicate handling;
- list by either endpoint;
- inverse relation derived;
- no source-content duplication.

### A3. Add relationship audit

Audit:

```text
relationship.created
relationship.deleted
```

Persist mutation + mandatory provenance atomically where possible.

### A4. Add Stage 1 service permissions

Suggested:

```text
relationships.read
relationships.write
objects.register
objects.read
```

Do not grant wildcard access to application service identities.

### A5. Add machine API

Conceptual endpoints:

```text
GET    /api/service/v1/relationships
POST   /api/service/v1/relationships
DELETE /api/service/v1/relationships/{id}
```

Optional object endpoint:

```text
POST /api/service/v1/objects/register
```

### A6. Actor context

Implement a server-only actor assertion contract.

Minimum recorded context:

- authenticated service `SVC-*`;
- asserted end-user Authentik subject;
- resolved `USR-*` if available;
- source application's local user ID;
- request ID.

Browser-controlled actor headers must be rejected/stripped.

### A7. OpenAPI/tests

Update:

- `docs/openapi.yaml`;
- route inventory;
- contract tests;
- DB integration tests;
- negative authorization tests;
- audit failure tests;
- concurrency/idempotency tests.

## 3. Workstream B — QA Authentik SSO

Suggested branch:

```text
stage1/qa-authentik-sso
```

### B1. Add identity binding

Add:

```text
users.authentik_subject
```

without changing existing `users.id`.

### B2. OIDC flow

Implement:

```text
/auth/login -> Authentik
/auth/callback -> bind user -> create qa_session
```

Retain:

```text
requireUser()
requireAccount()
requirePermission()
```

### B3. Existing-user migration

Build controlled mapping and dry-run report before write.

Do not silently bind ambiguous users.

### B4. Local password deprecation

First cutover release:

- stop ordinary password login after acceptance switch;
- hide password-management UI;
- keep hashes temporarily for rollback;
- do not destructively drop column yet.

## 4. Workstream C — QA cross-system links

Suggested branch:

```text
stage1/qa-cross-system-links
```

### C1. Keep current UX

Reuse existing:

- search/link modal concepts;
- link chips;
- deep links;
- link/unlink flow.

### C2. Generalize component

```text
RedmineLinks
-> RelatedObjects
```

### C3. Replace local relationship authority

New reads/writes go through Integration Core.

### C4. Migrate existing Redmine links

Inventory:

```text
bug_redmine_issues
test_case_redmine_issues
task_redmine_issues
```

Migration:

1. count;
2. export;
3. resolve/create Global IDs;
4. import relationships;
5. verify;
6. switch reads;
7. disable legacy writes;
8. retain read-only rollback state.

### C5. Deprecate per-user Redmine credential UI

`RedmineSettings` and `user_redmine` become legacy after new integration is
accepted.

## 5. Workstream D — Redmine native plugin

Repository:

```text
ekucher/redmine_bsystem_integration
```

Suggested branch:

```text
stage1/native-related-objects
```

Implement:

- issue-page Related Objects panel;
- Integration Core machine client;
- actor assertion after Redmine native authorization;
- add/remove relation;
- deep links;
- degraded state when Core is unavailable.

Do not make plugin availability a requirement for normal Redmine issue work.

## 6. Workstream E — Outline

Repository:

```text
ekucher/bsystem-outline
```

Tasks:

1. verify current OIDC existing-user binding;
2. preserve local logout patch;
3. identify minimal UI extension point;
4. implement Related Objects surface only if maintainable;
5. otherwise use a small isolated controlled patch.

## 7. Workstream F — Deploy/Authenik

Repository:

```text
ekucher/bsystem-deploy
```

Configure separate OIDC applications/providers:

```text
Redmine
QA
Outline
```

Define service identities for native backends.

Stage 1 deployment profile MUST NOT require the BHUB frontend.

## 8. Dependency order

```text
A1 Global ID extension
   ↓
A2 relationship store
   ↓
A3/A4 audit + permissions
   ↓
A5/A6 service API + actor context
   ↓
C QA relationship client/migration
   ├── D Redmine plugin
   └── E Outline integration

B QA SSO can progress largely in parallel.

F deployment work spans all streams.
```

## 9. Merge policy

Each workstream should have independent PRs with exact-head CI evidence.

Do not combine:

- QA SSO migration;
- relationship DB migration;
- Redmine major upgrade;
- Outline upstream upgrade;
- BHUB UI work.

## 10. Definition of Implementation Ready

The architecture is considered implementation-ready when:

- this documentation is canonical;
- baselines are re-fetched before coding;
- OD-02 actor/delegated-search boundaries are accepted;
- identity-binding inventories are prepared;
- relationship vocabulary is frozen for Stage 1;
- rollback plans are attached to migration PRs.

## 11. First recommended implementation PR

Start with Integration Core only:

```text
feat(stage1): add canonical relationship persistence
```

Scope:

- requirement Global ID;
- relationship table/store;
- relation vocabulary;
- DB tests;
- no external API yet.

This isolates the most foundational schema decision before network/auth concerns.