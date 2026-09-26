# BSYSTEM Product & Architecture Specification v1.0

Status: Draft for implementation  
Owner decision: approved to formalize, 2026-09-22  
Canonical repository: `ekucher/bsystem-deploy`

## 1. Purpose

BSYSTEM is a unified corporate workspace over specialized systems. Its purpose is to remove repeated manual entry, provide one identity and administration model, connect records across systems, and present users with one consistent interface.

BSYSTEM is **one product experience, not one database and not one monolithic application**.

## 2. Product principles

1. One sign-on and one administrative control plane.
2. One BSYSTEM HUB user experience and corporate design system.
3. Source systems remain authoritative for the domains they own.
4. Integration Core is the integration, authorization and normalization boundary.
5. Cross-system relationships use immutable BSYSTEM Global IDs, never mutable names, emails or hostnames.
6. Backend authorization is mandatory and deny-by-default.
7. Customer/tenant isolation is strict.
8. Data is not copied merely to make the UI convenient; caches and projections require an explicit purpose and lifecycle.
9. Specialized engines may remain independently deployable even when their native UI is not the primary user entry point.
10. AI is subject to the same authorization, scope and classification rules as ordinary APIs.

## 3. Scope

The unified workspace covers:

- CRM clients and contacts;
- projects and tasks;
- QA test cases and bugs;
- knowledge base;
- files and collaboration;
- products and installations;
- operational health relevant to supported products;
- backups, maintenance, updates and operational events;
- incidents and support;
- search and notifications;
- administration, RBAC, scopes and audit;
- AI-assisted workflows.

## 4. Non-scope

BSYSTEM is not:

- a replacement for all customer infrastructure monitoring;
- a CMDB for routers, switches, printers or unrelated customer systems;
- a second independent copy of CRM, Redmine, Outline or Nextcloud data;
- a direct cross-database integration layer;
- an authorization layer implemented only in the browser.

Operations observe only the server/environment required to operate a BSYSTEM-supported product.

## 5. Target architecture

```text
                         authentik
                    Identity / SSO / MFA
                              |
                              v
                       BSYSTEM HUB
                 Unified corporate workspace
                              |
                              v
                     Integration Core
          AuthZ / Global IDs / Relations / Audit
             Normalized APIs / Events / Search
                              |
       +-----------+----------+----------+-----------+
       |           |          |          |           |
       v           v          v          v           v
    EspoCRM     Redmine    Outline   Nextcloud   BSYSTEM native
      CRM      Projects     Wiki       Files     QA / Support /
                Tasks                           Operations / AI
```

Direct module access is first-class. HUB is the primary corporate workspace and launcher, but it is not a mandatory reverse proxy or authentication gateway. Each application remains an independent OIDC client of authentik and may be opened at its canonical URL. Where a normalized BSYSTEM workflow exists, HUB provides the unified experience without removing direct native access.

## 6. Identity and administration

authentik is the sole human identity provider. HUB uses OIDC Authorization Code + PKCE. Integration Core validates identity and performs authorization.

BSYSTEM administration provides a unified control plane for:

- users;
- roles;
- permissions;
- scopes;
- tenant/client access;
- service identities;
- integration status/configuration metadata;
- audit.

Existing logical roles remain Administrator, Manager, Developer, QA, Support, DevOps, Customer and Service Core. Authorization remains deny-by-default.

## 7. Unified corporate style

All BSYSTEM-owned screens use `@bsystem/design-system`.

The design system owns tokens and reusable UI primitives; business logic remains in HUB. Upstream systems are not required to be visually forked merely to match HUB. Where a workflow is important enough to be part of the unified experience, HUB implements the normalized BSYSTEM view/form against Integration Core.

This prevents permanent forks of CRM/Redmine/Outline/Nextcloud solely for appearance.

## 8. Domain model

Existing Global IDs remain authoritative. This specification introduces:

```text
INST-*  Product Installation
RUN-*   Operational Execution
```

Conceptual graph:

```text
Client CL-*
|
+-- Contact CT-*
|
+-- Project PR-*
|   +-- Task TSK-*
|   +-- Bug BUG-*
|   +-- Test Case TST-*
|   +-- Release REL-*
|   +-- Document DOC-*
|
+-- Installation INST-*
|   +-- Product APP-*
|   +-- Server SRV-*
|   +-- Backup RUN-*
|   +-- Maintenance RUN-*
|   +-- Update RUN-*
|   +-- SelfTest RUN-*
|   +-- Health
|   +-- Events
|
+-- Incident INC-*
```

`Installation` is the primary business-facing operational entity. `Server` remains a technical entity associated with an installation.

A single server may host more than one supported component; the model must not assume a permanent 1:1 relationship.

## 9. Cross-system relationships

Relationships are first-class BSYSTEM metadata.

Examples:

- Redmine task `TSK-1842` relates to QA bug `BUG-391`;
- incident `INC-81` relates to installation `INST-87`;
- installation `INST-87` belongs to client `CL-42`;
- document `DOC-71` relates to project `PR-27`.

The relationship stores Global IDs and relation semantics, not duplicated source records.

## 10. Product installations and Operations

BSYSTEM manages supported product installations rather than arbitrary customer infrastructure.

An installation may expose:

- product/version;
- environment;
- host/server facts relevant to product operation;
- product services;
- CPU/RAM/disk indicators where required for product health;
- application health;
- backup state/history;
- maintenance state/history;
- update state/history;
- self-test results;
- operational events;
- incidents and related tasks.

BRAVO Toolkit is an operations/telemetry producer for BRAVO installations, not a top-level business domain.

Toolkit results should be ingested as structured events/executions. HTML/PDF/log output may be retained as diagnostic artifacts but must not be the primary data model.

## 11. Backup and maintenance UX

Backup and maintenance belong to an installation.

The installation detail view should provide tabs such as:

```text
Overview | Health | Backups | Maintenance | Updates | Events | Incidents
```

Fleet-level dashboards aggregate these states across installations. A dedicated top-level Backup module is not required unless a future operational role needs independent management of repositories, retention, restore jobs and schedules at scale.

## 12. Information architecture

Target HUB navigation:

```text
Home
Clients
Projects
Tasks
QA
Knowledge Base
Files
Products
Installations
Support
  Incidents
  Requests
Reports
AI
Administration
```

Backup, maintenance and updates are contextual installation capabilities and dashboard filters rather than mandatory top-level modules.

## 13. Core user journeys

### Unified work item journey

```text
Client -> Project -> Task -> Bug -> Test -> Release
```

A user can move between related records without manually copying identifiers or searching in separate systems.

### Operations journey

```text
Client -> Installation -> Health/Backup/Maintenance
                         -> Incident -> Task
```

A failed backup can create or link an incident with client, installation, product and execution context already known.

### Knowledge journey

```text
Client/Project/Product -> related Documents -> Files
```

### Customer portal journey

Customers see only explicitly authorized client-scoped data and actions.

## 14. Data ownership

The detailed ownership matrix is defined in `DATA-OWNERSHIP.md`. Core rule: exactly one authoritative owner exists for each source-domain field.

Integration Core may own integration metadata, immutable mappings, relationships, RBAC/scopes, audit, notifications, operations/support state, and explicitly designed projections.

## 15. API and event model

Public normalized APIs remain versioned under `/api/v1/*`; machine APIs remain separate under `/api/service/v1/*`.

New product work must be contract-first in OpenAPI and preserve normalized error envelopes.

Operational events use `entity.action` naming. Expected examples include:

```text
backup.succeeded
backup.failed
maintenance.started
maintenance.completed
selftest.succeeded
selftest.failed
installation.warning
installation.error
incident.created
```

Machine producers receive the minimum capability/scope required for their installation.

## 16. Security boundaries

- authentik authenticates humans.
- Integration Core authorizes every request.
- HUB visibility is not authorization.
- Service identities are separate from human identities.
- Customer access is deny-by-default and scoped.
- Secrets are never source data for search/AI.
- CREDENTIAL-class data must never be sent to an LLM.
- Operational ingestion requires authentication, validation, bounded payloads and replay/idempotency protection where applicable.

## 17. AI

AI is a consumer of authorized BSYSTEM capabilities, not a bypass around them.

```text
User -> authentik -> HUB -> Core authorization -> AI Gateway -> permitted sources
```

AI write operations require explicit permission/scope, schema validation, audit and additional confirmation/policy for high-impact actions.

## 18. Delivery strategy

Do not rewrite EspoCRM, Redmine, Outline or Nextcloud merely to achieve visual consistency.

Deliver product capabilities as vertical slices:

1. formalize domain model and ownership;
2. add Installation/Relationship contracts;
3. expose normalized APIs;
4. build HUB list/detail/form flows with Design System;
5. prove authorization and cross-system relationships;
6. integrate operational producers such as BRAVO Toolkit;
7. expand customer portal and AI only over proven normalized capabilities.

## 19. Acceptance principles

A feature is not complete merely because a screen exists.

For applicable features, acceptance requires:

- authoritative source is defined;
- Global ID semantics are defined;
- authorization and tenant scope are tested;
- API/OpenAPI agree;
- UI handles loading/empty/error/forbidden states;
- audit semantics are defined for mutations;
- cross-system relation does not depend on mutable labels;
- tests are non-vacuous;
- CI/Security remain green.

## 20. Remote Management

Remote Management is a platform capability with a separate control plane and data plane. The accepted baseline is WireGuard + RDP for primary administration, Zabbix over the management overlay for monitoring, and self-hosted RustDesk as fallback.

HUB provides Remote Management UX; Integration Core enforces authorization and normalized contracts; deployment/runtime components such as WG Hub, ACL/firewall, Zabbix and RustDesk remain outside HUB.

Managed customer servers require no inbound public WAN ports for normal Remote Management operation. Access is deny-by-default, customer-isolated and based on unique engineer/device identities.

Detailed documentation is under `../remote-management/`.

## 21. Current / Target / Gap

This specification defines TARGET architecture. It must not be cited as proof that a capability is currently implemented.

For implementation/audit work always distinguish:

- CURRENT — verified in GitHub/runtime/CI/deployment;
- TARGET — this approved architectural direction;
- GAP — required implementation work.

## 22. Related specifications

- `DOMAIN-MODEL.md`
- `DATA-OWNERSHIP.md`
- `INFORMATION-ARCHITECTURE.md`
- `UX-FLOWS.md`
- `MVP-ROADMAP.md`
- `../adr/ADR-001-unified-workspace-specialized-engines.md`
- `../BSYSTEM-HUB-MASTER-CONTEXT.md`
- `../remote-management/README.md`
