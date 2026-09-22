# Remote Management Architecture

## Decision

Remote Management is a separate BSYSTEM platform capability, not "WireGuard inside HUB".

```text
CONTROL PLANE
Authentik -> HUB -> Integration Core -> Remote Management API
                                      -> Desired State / Policy / Audit

DATA / EXECUTION PLANE
WireGuard Hub / Firewall / RDP / Zabbix / RustDesk / DNS / Agent
```

A control-plane outage must not automatically break existing WireGuard tunnels, RDP access, Zabbix monitoring or RustDesk connectivity.

## Transport model

```text
PRIMARY    WireGuard + RDP
MONITORING Zabbix over WireGuard overlay
FALLBACK   RustDesk self-hosted
```

The domain model is transport-neutral. Supported management transports include `wireguard-direct`, `wireguard-gateway`, `openvpn` and `none`. Remote access providers include `rdp`, `rustdesk` and future providers.

## Device identity

A managed endpoint has one immutable BSYSTEM Device identity. Hostname and provider IDs are mutable mappings.

Example:

```text
Device ID:       DEV-000123
Customer:        00702972
Hostname:        WIN2016-LIMS
Display ID:      BS-00702972-WIN2016-LIMS
Management IP:   10.253.10.17
Zabbix mapping:  BS-00702972-WIN2016-LIMS
RustDesk mapping:BS-00702972-WIN2016-LIMS
```

The Global Device ID must not depend on hostname.

## Desired-state model

PostgreSQL is the canonical desired-state store. The following are runtime/external representations, not sources of truth:

- WireGuard configuration;
- firewall ACLs;
- Zabbix objects/database;
- RustDesk objects/database;
- DNS records;
- agent-applied configuration.

Each device has `desiredRevision` and `appliedRevision`. A mismatch is configuration drift. A controller reconciles desired and actual state or marks a clear ERROR/DRIFT condition.

External resources require ownership markers such as `managed-by=bsystem` and BSYSTEM Global IDs before destructive reconciliation.

## Device lifecycle

```text
NEW
 -> ENROLLING
 -> PROVISIONING
 -> MANAGED
 -> DEGRADED / OFFLINE / ERROR
 -> REVOKED
```

MANAGED requires server-side verification; installer exit code 0 alone is insufficient.

Verification may include WG peer/handshake, management reachability, RDP, Zabbix reporting and RustDesk registration.

## Health

Health is multi-channel, not a single online/offline bit.

Examples:

```text
WG OK + Zabbix OK + RustDesk OK       = HEALTHY
WG OK + Zabbix FAILED + RustDesk OK   = MONITORING_DEGRADED
WG FAILED + Zabbix FAILED + RustDesk OK = VPN_FAILURE / DEGRADED
all failed                            = DEVICE_OR_NETWORK_OFFLINE
```

## Agent

Initial bootstrap may be `Install-BSYSTEMRemote.ps1`; a long-term agent may fetch desired state, compare revisions, apply bounded declarative changes and report health/inventory.

The agent must not become a general arbitrary remote-shell mechanism.

## Events

Versioned normalized events may include:

- `remote.device.enrolled`
- `remote.device.managed`
- `remote.device.degraded`
- `remote.device.offline`
- `remote.device.revoked`
- `remote.access.granted`
- `remote.access.revoked`
- `remote.configuration.changed`

Events must never carry private keys, passwords, deployment tokens or device credentials.
