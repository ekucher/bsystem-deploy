# Stage 1 Migration Plan

## Production targets

Canonical PROD native application URLs:

```text
QA      https://qa.bravosoft.org
Redmine https://redmine.bravosoft.org
Wiki    https://kb.bsystem.com.ua
```

These endpoints are migration targets. Stage 1 changes must not assume different
production hostnames unless this document is updated first.

## Phase 0 — Freeze and inventory

- record exact repository SHAs;
- ensure clean working trees;
- create archive/safety refs where appropriate;
- inventory existing users in Authentik, Redmine, QA and Outline;
- inventory existing QA↔Redmine links;
- inventory current OIDC settings;
- inventory application URLs and callback URLs;
- verify that PROD application URLs match the canonical targets above;
- verify backups.

## Phase 1 — Documentation and contracts

Finalize:

- identity contract;
- binding rules;
- relation vocabulary;
- canonical object identities;
- Integration Core API contract;
- security model;
- acceptance test matrix.

No production cutover in this phase.

## Phase 2 — Authentik application configuration

Create/manage independent Authentik applications/providers for:

```text
Redmine
QA
Outline
```

Configure:

- exact redirect URIs;
- scopes;
- MFA policy;
- access entitlements;
- logout behavior.

Production redirect/origin configuration must be derived from the canonical
production URLs, not from development hostnames.

## Phase 3 — Identity binding inventory

Build a controlled mapping:

```text
Authentik identity
<-> Redmine user
<-> QA user
<-> Outline user
```

Resolve ambiguity manually.

Do not enable auto-provisioning for ambiguous existing identities.

## Phase 4 — QA SSO

Implement:

- OIDC login;
- callback;
- `authentik_subject`;
- existing-user binding;
- local QA session creation;
- logout semantics;
- migration tests.

Preserve native QA authorization.

## Phase 5 — Redmine SSO validation

Validate:

- existing user reuse;
- project memberships unchanged;
- issue/journal authorship unchanged;
- direct canonical login path;
- SSO reuse.

## Phase 6 — Outline SSO validation

Validate:

- existing user reuse;
- authorship unchanged;
- document access unchanged;
- local logout behavior.

## Phase 7 — Integration Core relationship model

Implement/extend:

- object registry;
- relation store;
- relation vocabulary;
- audit;
- native-app service authentication;
- actor propagation;
- canonical deep links.

Reuse existing Global-ID primitives where possible.

## Phase 8 — Migrate QA↔Redmine links

- export legacy QA links;
- import into Integration Core;
- verify row counts;
- verify object identity mapping;
- compare representative samples;
- switch QA reads;
- stop legacy writes;
- retain rollback path.

## Phase 9 — QA Related Objects UI

Generalize:

```text
RedmineLinks
-> RelatedObjects
```

Add Outline targets.

## Phase 10 — Redmine native plugin

Implement `redmine_bsystem_integration`:

- related object panel;
- add/remove relation;
- Integration Core client;
- graceful degradation;
- deep links;
- authorization-safe behavior.

## Phase 11 — Outline related-object surface

Implement the smallest maintainable Outline integration.

## Phase 12 — Acceptance and cutover

Run the full test matrix against the production hostnames and the exact release
candidate configuration before production activation.

Do not declare Stage 1 complete until exact deployed SHAs and test evidence are recorded.

## Rollback requirements

Each cutover must have a documented rollback.

Examples:

### QA auth rollback

- retain migration mapping;
- preserve local users;
- avoid destructive password-column removal in the first release;
- allow controlled rollback to previous auth until acceptance closes.

### Link migration rollback

- keep legacy QA relation tables unchanged/read-only;
- snapshot Integration Core relationship DB;
- allow read-path rollback until migration verification is complete.
