# Stage 1 Scope

## 1. Goal

Stage 1 must make Redmine, QA and Outline behave as one coherent ecosystem while preserving native application ownership and avoiding a premature BHUB frontend.

The user experience is:

```text
Authenticate once through Authentik
        ↓
Open Redmine / QA / Outline directly
        ↓
Move between related objects by native deep links
        ↓
Continue working in the target native application
```

## 2. In scope

### Identity and SSO

- Authentik as central IdP.
- Redmine as an Authentik-backed application.
- QA as an Authentik-backed application.
- Outline as an Authentik-backed application.
- SSO between these applications.
- Preservation and binding of existing local users.
- Application-level access entitlements.
- Central MFA and credential policy.
- Local application sessions remain allowed.

### Native authorization

- Redmine native projects, roles and memberships.
- QA native roles and project memberships.
- Outline native groups, collections and document permissions.
- No centralized replacement for application-level fine-grained RBAC.

### Cross-system relationships

At minimum:

- Redmine Issue
- QA Requirement
- QA Test Case
- QA Bug
- QA Task
- Outline Document

Capabilities:

- create relationship;
- remove relationship;
- list related objects;
- canonical deep links;
- semantic relation types;
- audit of relationship operations;
- migration of existing QA↔Redmine links.

### Native integration UI

Native application pages should expose a compact "Related Objects" surface.

No separate BHUB page is required.

## 3. Out of scope

The following are explicitly outside Stage 1:

- BHUB dashboard;
- BHUB launcher;
- BHUB global navigation;
- BHUB user administration;
- BHUB module administration;
- centralized replacement for Redmine/QA/Outline administration;
- centralized detailed RBAC;
- full data synchronization;
- shadow copies of application content;
- unified workflow engine;
- replacing Redmine issues with BHUB entities;
- replacing QA entities with BHUB entities;
- replacing Outline documents with BHUB entities;
- large-scale Redmine major-version upgrade unless independently approved;
- broad federated search that cannot preserve target-application authorization.

## 4. Stage 1 success definition

Stage 1 is successful when:

1. one Authentik identity can authenticate into Redmine, QA and Outline;
2. existing local user history remains intact;
3. native application permissions still determine what a user may do;
4. a user can navigate between related Redmine, QA and Outline objects;
5. Integration Core is authoritative for cross-system relationships;
6. applications remain independently usable through their canonical URLs;
7. BHUB UI is not a runtime dependency.