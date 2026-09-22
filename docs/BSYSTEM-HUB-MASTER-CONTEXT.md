# BSYSTEM-HUB — Master Project Context

**Context date:** 2026-09-23  
**Status:** canonical project context  
**Purpose:** architecture, implementation planning, audits, ChatGPT/Claude Code and autonomous development.

> This document is the repository-level canonical summary of the owner-supplied BSYSTEM-HUB Master Project Context. Statements about CURRENT implementation, runtime, CI or deployment must still be verified against GitHub/runtime before use.

## 1. Platform definition

BSYSTEM is a corporate platform that unifies independent business systems and infrastructure services into one workspace.

BSYSTEM must **not** become a monolith that reimplements CRM, wiki, file storage, project management, monitoring or remote support.

```text
                    AUTHENTIK
                        |
                        v
                   BSYSTEM-HUB
                        |
                        v
              BSYSTEM Integration Core
                        |
               +--------+---------+
               |        |         |
              APIs     NATS    Adapters
               |        |         |
               v        v         v
             Domain / external systems
```

- HUB = corporate portal / control plane / unified UX.
- Integration Core = security + integration boundary.
- External systems remain authoritative for their domains.
- Authentik = sole interactive identity authority.

## 2. Canonical repositories

```text
ekucher/bsystem-hub
ekucher/bsystem-integration-core
ekucher/bsystem-design-system
ekucher/bsystem-deploy
```

Repositories are independently versioned. Do not introduce relative filesystem imports, accidental cross-repo coupling or unversioned shared source trees.

`bsystem-deploy` coordinates canonical `CLAUDE.md`, `TASKS.md`, deployment documentation and cross-repository work.

## 3. Repository responsibilities

### bsystem-hub

Frontend, portal, navigation, dashboards, identity/RBAC-aware UX, customer/device UX, Remote Management UX, monitoring and audit presentation.

It must not contain infrastructure implementation.

### bsystem-integration-core

Normalized APIs, server-side authorization, RBAC/scopes, Global IDs, module registry, adapter contracts, audit, integration logic, NATS events and service boundaries.

Integration Core is the security boundary.

### bsystem-design-system

Reusable visual primitives: forms, dialogs, tables, navigation, status/health indicators, device cards, dangerous-action dialogs and audit timelines.

### bsystem-deploy

Docker/Compose, Authentik blueprints, reverse proxy, networking, environment configuration, secret references, observability, backup, runbooks and cross-repository coordination.

## 4. Source-of-truth workflow

Before repository changes verify actual GitHub state: branch, HEAD, canonical remote branch, working tree and ahead/behind state.

Never:

- force-push without explicit necessity;
- commit secrets;
- rewrite history casually;
- claim CI is green without checking actual workflows;
- weaken tests/security to obtain green CI.

Tests must be non-vacuous.

For autonomous work, follow `TASKS.md` priority order. Record blockers with evidence and continue independent work. Owner input is required for production credentials, destructive/irreversible actions, explicit business/product decisions or explicit approval gates.

## 5. Identity and SSO

Authentik is the sole interactive IdP/IAM.

```text
Browser -> Authentik -> OIDC Authorization Code + PKCE -> Application
```

HUB is **not** an IdP, OAuth proxy or token broker.

Each application is its own OIDC client:

```text
Authentik
 +-- BSYSTEM-HUB
 +-- Redmine
 +-- Outline
 +-- Nextcloud
 +-- Remote Management components
 +-- future modules
```

Direct canonical module access is first-class. HUB is launcher/navigation/control UX, not a mandatory reverse proxy.

Launch URL and adapter/API URL are separate concepts. Never place HUB tokens in module browser URLs.

## 6. Authoritative systems

- EspoCRM: CRM/customer/contact authority.
- Redmine: project/task authority.
- Outline: knowledge/wiki authority.
- Nextcloud: file/content authority.
- Vaultwarden: independent secrets/security boundary.
- BSYSTEM-owned domains: normalized platform metadata and explicitly defined native capabilities.

No cross-database SQL. Integrate through application API -> adapter -> normalized contract.

## 7. Authorization

Authentik owns identity/groups. BSYSTEM authorization is a separate server-side layer.

```text
Authentik Group -> BSYSTEM Role -> Permission -> Scope
```

Permission pattern:

```text
module.resource.action
```

Examples:

```text
crm.client.read
projects.task.read
support.incident.read
remote.device.view
remote.rdp.connect
remote.rustdesk.connect
remote.monitoring.view
```

Permissions require scopes in multi-customer contexts. Authorization is deny-by-default.

A hidden frontend button is not authorization. Backend validates identity, role, permission, scope, resource and action.

## 8. Global identity

Cross-system normalized entities use persistent immutable BSYSTEM Global IDs. External/provider numeric IDs, hostnames, emails and display names are not canonical identity.

The master context uses examples such as:

```text
CUS-000042
DEV-000123
USR-000012
PRJ-000055
```

Existing product documentation also contains established prefixes such as `CL-*`, `PR-*`, `INST-*`, `TSK-*`, etc. These namespaces must **not be silently renamed**. Prefix convergence is a schema/migration decision and requires explicit implementation design. Until then, semantics and immutable mappings take precedence over cosmetic prefix consistency.

## 9. Integration Core

Prefer a modular monolith until a concrete runtime/security/scaling reason justifies service extraction.

Logical modules may include identity, authorization, module registry, customers, projects, audit, adapters, Remote Management, AI tool gateway and events.

NATS carries versioned normalized domain/platform events and never secrets.

## 10. Module Registry

Normalized module representation may include:

```text
id
name
enabled
status
launchUrl
adapter
capabilities
requiredPermissions
health
```

Module Registry does not turn HUB into a reverse proxy.

Prefer canonical FQDNs such as `hub.example`, `auth.example`, `redmine.example`, `wiki.example`, `cloud.example`, `remote.example`.

## 11. Deployment/security baseline

Current deployment baseline is Proxmox VM -> Docker / Docker Compose.

Internal-only services include PostgreSQL, Redis, NATS and internal Core endpoints. Public edge uses reverse proxy + HTTPS + canonical FQDNs.

Containers should be non-root/minimal privilege where practical, have healthchecks/resource limits and pinned versions/digests where appropriate. Avoid privileged mode, host network/PID and Docker socket without a documented requirement.

Never commit passwords, API tokens, WG private keys, OIDC secrets, DB credentials, RustDesk admin tokens or device credentials.

## 12. AI

AI is a platform capability behind ordinary authorization:

```text
User -> HUB -> Integration Core -> Authorized AI tools -> Domain adapters
```

AI inherits user authorization and cannot become an authorization bypass or unrestricted administrative shell.

## 13. Remote Management

Remote Management provides centralized secure administration, monitoring and fallback support for 50+ customer Windows Servers.

Accepted model:

```text
PRIMARY     WireGuard + RDP
MONITORING  Zabbix over WireGuard
FALLBACK    RustDesk self-hosted
```

Control plane:

```text
Authentik -> HUB -> Integration Core -> Remote Management API
```

Data plane:

```text
WireGuard / firewall / RDP / RustDesk / Zabbix / DNS / Agent
```

Control-plane outage must not automatically stop a working data plane.

Detailed canonical Remote Management documentation is under `docs/remote-management/`.

## 14. Remote Management invariants

- hub-and-spoke management topology;
- proposed overlay `10.253.0.0/16` requires production conflict audit;
- no public RDP;
- no public Zabbix Agent;
- no required inbound WAN ports on managed customer servers;
- customer server connects outbound to BSYSTEM WG Hub;
- unique WG identity per engineer;
- unique WG identity per direct-managed device;
- device WG private key generated locally and never leaves device;
- WireGuard `AllowedIPs` is not RBAC;
- hub firewall/ACL is deny-by-default;
- Customer A -> Customer B is denied;
- Windows login is a separate security layer after VPN/RBAC;
- no shared permanent RustDesk password.

## 15. Remote Management domain

Use transport/provider-neutral concepts:

- Customer
- Device
- Engineer
- Enrollment
- Provisioning
- Management Transport
- Remote Access Capability
- Monitoring Capability
- Policy
- Desired State / Actual State
- Health
- Configuration Revision
- Audit

A Device has immutable BSYSTEM identity. Hostname and provider IDs are mutable mappings.

Lifecycle:

```text
NEW -> ENROLLING -> PROVISIONING -> MANAGED
                         |             |
                         v             v
                       ERROR     DEGRADED/OFFLINE
                                       |
                                       v
                                    REVOKED
```

A device becomes MANAGED only after server-side validation, not merely installer exit code 0.

## 16. Remote provisioning

Manual configuration is not the normal workflow for 50+ servers.

Initial bootstrap may use `Install-BSYSTEMRemote.ps1`; long-term target may be a signed agent.

Enrollment tokens are short-lived, customer-scoped, usage-limited, revocable, hashed server-side and audited.

Device provisioning includes local WG key generation, RDP/NLA/firewall configuration, Zabbix, RustDesk and health reporting. Backend handles identity, IP allocation, peer/ACL desired state, provider mappings and actual-state verification.

IP allocation must be transactional/concurrency-safe. Do not derive management IP from customer code, hostname or hash.

Desired state lives in PostgreSQL. Provider configs/databases are actual/external state. Reconciliation compares desired and actual state and exposes revision drift.

Agent behavior is declarative; do not turn it into a generic cloud PowerShell shell.

## 17. Remote health

Health is multi-channel:

```text
WireGuard
RDP
Zabbix
RustDesk
Agent
Configuration revision
```

Do not infer overall OFFLINE from one failed provider.

Central checks may include WG handshake, management reachability, TCP/3389, Zabbix freshness and RustDesk state.

Legacy Windows Server 2012 R2 may use `wireguard-gateway`, `openvpn` or another supported transport rather than forcing current direct WireGuard client assumptions.

## 18. Remote access authorization

Model:

```text
WHO   Engineer identity
WHAT  Permission
WHERE Customer / Device scope
HOW   Network/provider enforcement
```

Example capabilities include view, RDP, RustDesk, monitoring, enroll, manage, revoke, access-manage and audit-view.

Future Just-In-Time access may grant scoped access for a duration/reason and revoke automatically.

## 19. Audit and observability

Privileged operations record actor, action, resource, customer, timestamp, result, correlation ID and safe metadata.

Never log secrets.

Platform services expose structured logs, health, readiness, metrics and correlation IDs.

Remote Management additionally observes enrollment failures, IP pool exhaustion, WG/ACL reconciliation errors and Zabbix/RustDesk adapter failures.

## 20. OpenAPI, testing and failure handling

Implementation and OpenAPI must not drift.

Negative/resilience tests include invalid/expired tokens, insufficient scope, customer isolation, duplicate enrollment, timeouts, provider failure, NATS failure, DB races and partial provisioning.

Concurrency correctness relies on database constraints, transactions, locking and idempotency keys.

Partial provisioning is not blindly rolled back. Example: WG OK + Zabbix OK + RustDesk FAILED remains PROVISIONING/DEGRADED and retry continues the missing step.

## 21. CURRENT / TARGET / GAP rule

Every audit and implementation plan must distinguish:

- **CURRENT** — verified in GitHub/runtime/CI/deployment;
- **TARGET** — canonical architectural direction;
- **GAP** — work required to move from CURRENT to TARGET.

Never describe target architecture as already implemented.

## 22. Before each implementation wave

1. Fetch/check current repositories.
2. Read canonical `CLAUDE.md`.
3. Read `TASKS.md`.
4. Check relevant open PRs/issues.
5. Verify branch/HEAD/working state.
6. Audit current implementation.
7. Separate CURRENT / GAP / TARGET.
8. Implement the smallest coherent change.
9. Run real validation.
10. Verify GitHub CI.
11. Update documentation/task state.

## 23. Definition of Done

A task is not DONE because code exists.

Applicable completion evidence includes implementation, meaningful tests, lint/typecheck/build, actually verified CI, updated docs, preserved security invariants, no secrets, no unexplained drift and checked cross-repository impact.

## 24. Core invariants

Never violate:

```text
Authentik is the sole interactive IdP
HUB is not IdP/token broker
Integration Core enforces authorization
UI hiding is not security
deny by default
source systems remain authoritative
no cross-database SQL
canonical immutable Global IDs
module launch URL != adapter API URL
no HUB tokens in module browser URLs
no secrets in Git/events/logs
tests are meaningful
CI status is actually verified
no force-push
no customer-to-customer Remote Management traffic
no public RDP
no public Zabbix agent
managed Remote devices require no WAN inbound ports
per-engineer WG identity
per-device WG identity
control-plane failure does not destroy working data-plane access
```

## 25. Final definition

BSYSTEM-HUB is not a set of iframes and not a reverse proxy to external systems.

It is:

```text
Corporate Workspace
Identity-aware Portal
Control Plane
Normalized Integration UX
Customer / Project / Device workspace
Secure launcher
Remote Management UI
Monitoring entry point
Audit and operational dashboard
AI-assisted workspace
```

The architectural decomposition remains:

```text
Authentik         = Identity
Integration Core  = Security + Integration
External modules  = Authoritative Domain Systems
BSYSTEM-HUB       = Unified User Experience / Control Plane
```
