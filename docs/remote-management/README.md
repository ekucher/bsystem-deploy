# BSYSTEM Remote Management

Status: Architecture baseline  
Source context: `REMOTE-MANAGEMENT-CONTEXT.md` supplied 2026-09-23.

## Purpose

BSYSTEM Remote Management is a platform capability for centralized administration and monitoring of managed customer Windows Servers.

Primary transport is **WireGuard + RDP**. Monitoring uses **Zabbix through the management overlay**. **Self-hosted RustDesk** is the independent fallback remote-access provider.

The capability must share BSYSTEM identity, authorization, customer isolation, audit and Global-ID principles. It must not create a separate user database.

## Architecture

```text
Authentik
   |
   v
BSYSTEM-HUB                  Control Plane UX
   |
   v
Integration Core            Security / Integration Boundary
   |
   v
Remote Management API       Desired State / Orchestration
   |
   +-- WireGuard Controller --> WG Hub --> Management Overlay
   +-- Zabbix Adapter -------> Zabbix
   +-- RustDesk Adapter -----> RustDesk
```

HUB must not become a VPN concentrator, packet router, RustDesk relay, Zabbix server or secrets backend.

## Core domain

Use transport-neutral concepts:

- Customer
- Device / Managed Device
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

Provider objects such as WireGuardPeer, ZabbixHost and RustDeskDevice are adapter/infrastructure representations, not domain roots.

## Target scale

Initial target is 50+ managed servers. The architecture must scale to 100, 200 and 500+ devices without a fundamental redesign.

## Security invariants

- no public RDP;
- no public Zabbix agent;
- no required inbound ports on managed customer servers;
- unique WireGuard identity per engineer;
- unique WireGuard identity per direct managed device;
- no shared permanent RustDesk password;
- no customer-to-customer connectivity;
- deny by default;
- server-side authorization;
- customer isolation;
- central audit;
- private device keys generated locally where practical;
- no secrets in Git/API logs/NATS;
- control-plane outage must not terminate established data-plane access.

## Repository responsibilities

### bsystem-hub

Customer/device UX, enrollment UX, status/health, remote-access launchers, monitoring links/views, audit and policy UX.

### bsystem-integration-core

Authorization, normalized identities, RBAC/scopes, adapter contracts, audit/event integration and the normalized Remote Management API boundary.

### bsystem-deploy

WG Hub and Remote Management deployment, Zabbix/RustDesk connectivity, network/firewall policy, secret references, observability, backup and runbooks.

### bsystem-design-system

Reusable device/health/status UI, dangerous-action dialogs and audit timeline components.

## Documents

- [Architecture](REMOTE-MANAGEMENT-ARCHITECTURE.md)
- [Security](REMOTE-MANAGEMENT-SECURITY.md)
- [Enrollment](REMOTE-MANAGEMENT-ENROLLMENT.md)
- [Network](REMOTE-MANAGEMENT-NETWORK.md)
- [Threat Model](REMOTE-MANAGEMENT-THREAT-MODEL.md)
- [Runbook](REMOTE-MANAGEMENT-RUNBOOK.md)

## Implementation order

1. Domain model
2. Threat model and security invariants
3. API contracts
4. Database constraints
5. Enrollment lifecycle
6. WireGuard control plane
7. Network isolation
8. Windows bootstrap
9. Health verification
10. Zabbix adapter
11. RustDesk adapter
12. Integration Core authorization
13. BSYSTEM-HUB UI
14. End-to-end tests
15. Deployment/runbooks

Do not begin implementation from UI and do not make production functionality depend on undocumented manual configuration.
