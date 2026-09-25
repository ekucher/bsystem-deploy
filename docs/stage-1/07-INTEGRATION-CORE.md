# Integration Core — Stage 1 Role

> Live implementation audit: see [17-INTEGRATION-CORE-AUDIT.md](17-INTEGRATION-CORE-AUDIT.md).

## 1. Existing reusable foundation

The current Integration Core already provides useful Stage 1 primitives:

- Authentik-backed authentication support;
- local JWT/JWKS validation capability;
- service identities;
- Global IDs / source mappings;
- Redmine adapter;
- Outline adapter;
- normalized search;
- audit;
- PostgreSQL persistence;
- adapter health;
- retries;
- timeouts;
- circuit breaker behavior;
- OpenAPI contract.

Stage 1 should reuse these primitives rather than create a new integration service.

## 2. Stage 1 responsibility reduction

Integration Core currently contains broader platform concerns such as:

- persistent platform RBAC;
- scopes;
- module registry;
- module administration;
- Authentik human-user administration.

Those capabilities are not necessarily deleted, but they are not Stage 1 application-authorization authority.

## 3. Required Stage 1 capabilities

### Object registry

Must resolve stable object references across:

```text
redmine/issue
qa/requirement
qa/test_case
qa/bug
qa/task
outline/document
```

### Relationship store

The live audit confirms this generic store does **not** currently exist and must be added on top of the existing `global_entities` mapping layer.

Must provide canonical cross-system relationship persistence.

### Relationship API

Minimum conceptual surface:

```text
GET    /integration/v1/objects/{system}/{type}/{id}/relationships
POST   /integration/v1/relationships
DELETE /integration/v1/relationships/{id}
```

Optional:

```text
GET /integration/v1/objects/search
GET /integration/v1/objects/{system}/{type}/{id}
```

Search must be authorization-safe.

### Audit

Create/remove relationship operations must record:

- actor;
- source object;
- target object;
- relation type;
- timestamp;
- request/correlation context where available.

## 4. Authentication of native applications

Native applications may use service identities for server-to-server Integration Core calls.

Examples:

```text
SVC-redmine
SVC-qa
SVC-outline
```

However, a machine identity alone is not sufficient for end-user authorization decisions.

Where an operation is user-initiated, the Core should preserve actor context for audit and authorization policy.

## 5. Metadata cache

Integration Core may cache:

- source identity;
- display identifier;
- title where authorization rules allow;
- status where authorization rules allow;
- canonical URL;
- last-seen timestamp.

It must not replicate full issue/test/document content.

## 6. Adapter policy

Adapters remain isolated from domain consumers.

Redmine/Outline/QA-specific upstream field names should not leak into generic relationship storage.

## 7. Live-audit decision

The audit is complete for baseline `dd2c7f90c887183db9392fdf6db6b980c2adccbf`.

Decision:

```text
Global ID/object identity primitives -> REUSE
Generic relationship persistence     -> ADD
Audit subsystem                       -> REUSE/EXTEND
Search                                -> REUSE SELECTIVELY
Service identities                    -> REUSE
```

No parallel object-identity subsystem should be created.