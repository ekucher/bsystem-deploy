# BSYSTEM Stage 1 — Documentation Baseline

**Status:** Implementation-ready architecture baseline  
**Stage:** Stage 1  
**UX level:** Level B — Native Applications  
**Date:** 2026-09-25

## 1. Purpose

Stage 1 creates a unified BSYSTEM application ecosystem without introducing a user-facing BHUB interface.

Users continue to work directly in the native applications:

- Redmine
- QA
- Outline

Stage 1 adds two common platform capabilities:

1. **Central authentication and SSO through Authentik.**
2. **Cross-system relationships between application objects through BSYSTEM Integration Core.**

## 2. Stage 1 principle

```text
No BHUB user interface in Stage 1.

Authentik provides identity and authentication.
Each native application remains authoritative for its own users, roles,
permissions, content and administration.
Integration Core becomes authoritative for cross-system relationships.
```

## 3. Target topology

```text
                           AUTHENTIK
                  identity / authentication / MFA
                           /   |   \
                         OIDC OIDC OIDC
                         /     |     \
                        v      v      v

                    REDMINE   QA   OUTLINE
                    native   native  native
                      UI       UI      UI
                      |        |       |
                      +--------+-------+
                               |
                               v
                    BSYSTEM Integration Core
                               |
                  object registry / relationships
                    audit / integration metadata
```

## Native application endpoints

Stage 1 uses deployment variables for browser-visible native application origins:

```text
QA_PUBLIC_URL
REDMINE_PUBLIC_URL
OUTLINE_PUBLIC_URL
```

Safe documentation/test values:

```dotenv
QA_PUBLIC_URL=https://qa.example
REDMINE_PUBLIC_URL=https://redmine.example
OUTLINE_PUBLIC_URL=https://kb.example
```

Real PROD hostnames are stored only in the untracked deployment environment.
Stage 1 does not place a BHUB UI in front of these native applications.

## 4. Source-of-truth model

| Domain | Source of Truth |
|---|---|
| Central identity | Authentik |
| Passwords / MFA / central SSO session | Authentik |
| Application access entitlement | Authentik |
| Redmine users / roles / projects / issues | Redmine |
| QA users / roles / project membership / QA entities | QA |
| Outline users / groups / collections / documents | Outline |
| Cross-system relationships | Integration Core |
| Integration audit | Integration Core |

## 5. Canonical Stage 1 documents

1. [01-SCOPE.md](01-SCOPE.md)
2. [02-ARCHITECTURE.md](02-ARCHITECTURE.md)
3. [03-IDENTITY-SSO.md](03-IDENTITY-SSO.md)
4. [04-IDENTITY-BINDING.md](04-IDENTITY-BINDING.md)
5. [05-NATIVE-AUTHORIZATION.md](05-NATIVE-AUTHORIZATION.md)
6. [06-CROSS-SYSTEM-RELATIONSHIPS.md](06-CROSS-SYSTEM-RELATIONSHIPS.md)
7. [07-INTEGRATION-CORE.md](07-INTEGRATION-CORE.md)
8. [08-REDMINE-INTEGRATION.md](08-REDMINE-INTEGRATION.md)
9. [09-QA-INTEGRATION.md](09-QA-INTEGRATION.md)
10. [10-OUTLINE-INTEGRATION.md](10-OUTLINE-INTEGRATION.md)
11. [11-MIGRATION-PLAN.md](11-MIGRATION-PLAN.md)
12. [12-REPOSITORY-DISPOSITION.md](12-REPOSITORY-DISPOSITION.md)
13. [13-SECURITY.md](13-SECURITY.md)
14. [14-ACCEPTANCE-CRITERIA.md](14-ACCEPTANCE-CRITERIA.md)
15. [15-TEST-MATRIX.md](15-TEST-MATRIX.md)
16. [16-OPEN-DECISIONS.md](16-OPEN-DECISIONS.md)
17. [17-INTEGRATION-CORE-AUDIT.md](17-INTEGRATION-CORE-AUDIT.md)
18. [18-IMPLEMENTATION-PLAN.md](18-IMPLEMENTATION-PLAN.md)

## 6. Stage 1 invariants

### S1-INV-01
Authentik MUST be the central interactive authentication authority for Redmine, QA and Outline.

### S1-INV-02
Stage 1 MUST NOT require a BHUB user interface.

### S1-INV-03
Each application MUST retain its native administration and fine-grained authorization model.

### S1-INV-04
Existing application-local user identities MUST be preserved.

### S1-INV-05
SSO migration MUST NOT create duplicate local accounts for already existing users when a controlled binding exists.

### S1-INV-06
Historical authorship, ownership, assignments and audit history MUST remain attached to the original application-local user.

### S1-INV-07
Integration Core MUST be the source of truth for cross-system relationships.

### S1-INV-08
Redmine, QA and Outline MUST remain sources of truth for their own domain objects.

### S1-INV-09
A cross-system relationship MUST NOT imply authorization to the target object.

### S1-INV-10
Cross-system discovery MUST NOT disclose protected target metadata to a user who cannot access that object in the authoritative application.

### S1-INV-11
Cross-system links MUST use stable object identities and canonical deep links.

### S1-INV-12
Native applications SHOULD NOT build direct N×N relationship storage between each other.

## 7. Current audited baselines

### QA

Canonical repository:

```text
bsystem-code/qa
```

Audited branch:

```text
main
```

Audited SHA:

```text
803a49f11c14c1ce181b1be41f8c626c21cc1407
```

Important existing Stage 1 assets:

- local application user model and native RBAC;
- project membership isolation;
- native QA entities and history;
- Redmine search/link/unlink UI;
- Redmine API integration;
- local Redmine relation tables;
- production updater and deployment hardening.

### Integration Core

Canonical repository:

```text
ekucher/bsystem-integration-core
```

Existing reusable assets include:

- OIDC token verification;
- Authentik integration;
- service identities;
- Global IDs;
- Redmine adapter;
- Outline adapter;
- search;
- audit;
- PostgreSQL migrations;
- adapter health / retry / circuit breaker.

### Redmine

Deployment integration already exists under:

```text
ekucher/bsystem-deploy/redmine
```

Native Stage 1 integration plugin repository:

```text
ekucher/redmine_bsystem_integration
```

The plugin repository is the canonical location for new Redmine-native relationship UI and Integration Core client logic.

### Outline

Canonical controlled build:

```text
ekucher/bsystem-outline
```

Existing controlled OIDC logout behavior MUST be preserved.

## 8. Document authority

Where an older BHUB specification, prototype or repository document conflicts with these Stage 1 documents, the Stage 1 documentation takes precedence for Stage 1 implementation.


### Integration Core live audit

Canonical repository:

```text
ekucher/bsystem-integration-core
```

Audited branch:

```text
main
```

Audited SHA:

```text
dd2c7f90c887183db9392fdf6db6b980c2adccbf
```

Exact-head GitHub Actions evidence on this SHA:

```text
CI       success
Security success
```

The audit established that Stage 1 MUST extend the existing Global ID, audit,
search, adapter and service-identity primitives rather than create parallel
infrastructure.

See:

- [17-INTEGRATION-CORE-AUDIT.md](17-INTEGRATION-CORE-AUDIT.md)
- [18-IMPLEMENTATION-PLAN.md](18-IMPLEMENTATION-PLAN.md)