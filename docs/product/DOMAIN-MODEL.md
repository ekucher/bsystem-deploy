# BSYSTEM Domain Model v1.0

## Purpose

This document defines the canonical business entities and their relationships. It does not redefine source-system ownership.

## Identity rules

Every cross-system entity uses an immutable BSYSTEM Global ID. Mutable labels such as name, email, hostname or upstream display title are never authoritative cross-system keys.

The canonical master context also uses example prefixes `CUS-*`, `DEV-*`, `USR-*` and `PRJ-*`. Existing product contracts use `CL-*`, `PR-*`, `INST-*` and other prefixes below. This documentation does **not** silently rename persisted identifiers. Prefix convergence requires an explicit schema/API/migration decision. Semantics and immutable mappings are authoritative until that decision is implemented.

## Entity catalog

| Prefix | Entity | Authority |
| --- | --- | --- |
| USR-* | User | authentik identity + BSYSTEM mapping |
| SVC-* | Service identity | Integration Core |
| CL-* | Client | EspoCRM |
| CT-* | Contact | EspoCRM |
| PR-* | Project | Redmine |
| TSK-* | Task/Issue | Redmine |
| TST-* | Test Case | BSYSTEM QA |
| BUG-* | Bug | BSYSTEM QA |
| DOC-* | Document | Outline |
| REL-* | Release | BSYSTEM domain |
| REP-* | Repository | BSYSTEM domain |
| APP-* | Product/Application | BSYSTEM catalog |
| SRV-* | Server/host relevant to supported product | BSYSTEM Operations |
| INST-* | Product Installation | BSYSTEM Operations |
| RUN-* | Operational Execution | BSYSTEM Operations |
| INC-* | Incident | BSYSTEM Support |
| DEV-* | Remote Managed Device | BSYSTEM Remote Management |

`INST-*` and `RUN-*` are new identifiers introduced by Product Specification v1.0 and require implementation/migration design before production use. `DEV-*` is the immutable identity of a Remote Management endpoint and is not synonymous with `SRV-*` or `INST-*`.

## Core relationships

```text
CL-* Client
 +-- CT-* Contact
 +-- PR-* Project
 |    +-- TSK-* Task
 |    +-- BUG-* Bug
 |    +-- TST-* Test Case
 |    +-- DOC-* Document
 +-- INST-* Installation
 |    +-- APP-* Product
 |    +-- SRV-* Server
 |    +-- RUN-* Backup/Maintenance/Update/SelfTest
 |    +-- INC-* Incident
 +-- INC-* Incident
```

## Installation

Installation represents a deployed supported product for a client/environment.

Minimum conceptual attributes:

- Global ID;
- client Global ID;
- product Global ID;
- environment;
- lifecycle/status;
- related server Global ID(s);
- source/integration metadata;
- created/updated timestamps.

Product version and host facts may originate from telemetry/discovery and must record provenance when authoritative behavior depends on them.

## Operational execution

`RUN-*` represents one bounded execution such as backup, maintenance, update or self-test.

Conceptual fields:

- Global ID;
- installation Global ID;
- type;
- state/result;
- started/finished timestamps;
- producer/service identity;
- correlation/request ID;
- structured checks/summary;
- artifact references;
- failure code/message where applicable.

Raw logs are artifacts, not the execution identity.

## Relationship model

Cross-domain links are first-class records conceptually containing:

- source Global ID;
- target Global ID;
- relation type;
- provenance;
- actor/service identity;
- timestamps.

Examples:

```text
TSK-1842 --implements/fixes--> BUG-391
INC-81   --affects----------> INST-87
INST-87  --belongs_to-------> CL-42
DOC-71   --documents--------> PR-27
```

Authorization must be evaluated before exposing either endpoint of a relationship.

## Remote Managed Device

`DEV-*` identifies an endpoint enrolled into BSYSTEM Remote Management. Its hostname, management IP, WireGuard peer, Zabbix host and RustDesk identifier are mutable/provider mappings.

A Remote Device may be related to a supported-product Installation, but the relationship is not assumed to be 1:1. Remote Management must remain transport/provider-neutral.

## Invariants

1. Global IDs are immutable.
2. Source IDs may be mapped but never exposed as the only cross-system identity.
3. Deleting or renaming a source record must not cause a different entity to inherit its Global ID.
4. A relation cannot grant access to an otherwise unauthorized entity.
5. Installation is not synonymous with Server.
6. Operational executions are append-oriented historical facts; corrections must preserve auditability.
7. Remote Device is not synonymous with Installation or Server.
8. Provider identifiers and management IPs are not canonical Device identity.
