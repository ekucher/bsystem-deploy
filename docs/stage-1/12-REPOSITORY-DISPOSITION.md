# Repository Disposition — Stage 1

## 1. `bsystem-code/qa`

**Status:** ACTIVE — Stage 1 canonical QA application

Baseline audited:

```text
main
803a49f11c14c1ce181b1be41f8c626c21cc1407
```

### Keep

- domain model;
- native roles;
- project isolation;
- project memberships;
- application history;
- production updater;
- deployment hardening;
- Redmine integration UI concepts.

### Refactor

- local auth -> Authentik SSO;
- Redmine-specific links -> generic related objects;
- direct Redmine integration -> Integration Core relationship boundary.

### Migrate

- existing local QA↔Redmine relationship rows.

### Deprecate

- per-user Redmine settings/API key;
- ordinary local password authentication after successful SSO cutover.

## 2. `ekucher/bsystem-integration-core`

**Status:** ACTIVE — Stage 1 platform integration backbone

Audited canonical baseline:

```text
main
dd2c7f90c887183db9392fdf6db6b980c2adccbf
```

Exact-head GitHub Actions:

```text
CI       success
Security success
```

### Reuse

- OIDC verification;
- service identities;
- Global IDs;
- Redmine adapter;
- Outline adapter;
- normalized API patterns;
- search infrastructure;
- audit;
- DB migrations;
- health/retry/circuit breaker.

### Stage 1 extension

- generic relationship store;
- relation API;
- object registry extensions for QA;
- relation audit;
- actor propagation;
- authorization-safe metadata resolution.

### Not Stage 1 authority

Existing central platform RBAC/module administration is not the authoritative permission system for Redmine/QA/Outline native resources.

## 3. `ekucher/bsystem-deploy`

**Status:** ACTIVE — canonical Stage 1 deployment/orchestration repository

### Reuse

- Authentik deployment;
- PostgreSQL;
- Redmine runtime;
- Integration Core runtime;
- existing OIDC work;
- operational hardening.

### Change

Introduce/define a Stage 1 deployment profile that does not require the BHUB frontend runtime.

## 4. `ekucher/redmine_bsystem_integration`

**Status:** ACTIVE — canonical Redmine Stage 1 native plugin repository

Repository is intentionally suitable for the new plugin.

Purpose:

- native Related Objects UI;
- Integration Core client;
- Redmine issue context;
- add/remove relation;
- graceful degradation.

## 5. `ekucher/redmine`

**Status:** SUPPORTING / UPSTREAM FORK / EXPERIMENTAL

Do not make it the canonical Stage 1 integration repository.

Existing theme work should be extracted cleanly where useful.

Avoid merging unrelated upstream changes merely to obtain the theme.

## 6. `ekucher/bsystem-redmine`

**Status:** UNASSIGNED / CANDIDATE FOR ARCHIVE OR PACKAGING ROLE

Do not duplicate `redmine_bsystem_integration`.

A separate role must be explicitly defined before adding Stage 1 code.

## 7. `ekucher/bsystem-outline`

**Status:** ACTIVE — controlled Outline build

Reuse the existing OIDC logout patch.

Add only minimal, well-isolated integration patches if needed.

## 8. `ekucher/bsystem-hub`

**Status:** FROZEN FOR STAGE 1 / FUTURE STAGE

The existing BHUB frontend and administration work is not discarded.

It becomes future-stage material.

Stage 1 MUST NOT depend on it at runtime.

Open BHUB-specific feature work should be marked/deferred as post-Stage-1 rather than mixed into the Stage 1 baseline.

## 9. `ekucher/bsystem-design-system`

**Status:** SUPPORTING / FUTURE

May provide:

- visual tokens;
- terminology;
- native integration styling guidance.

No BHUB frontend dependency is introduced.

## 10. Suggested branch naming

### QA

```text
stage1/qa-authentik-sso
stage1/qa-cross-system-links
```

### Integration Core

```text
stage1/relationships
```

or a more specific feature branch once the existing relation primitives are audited.

### Redmine plugin

```text
stage1/native-related-objects
```

### Deploy

```text
stage1/native-apps-sso
```

Branch names should be created from freshly fetched canonical main branches only after precheck.