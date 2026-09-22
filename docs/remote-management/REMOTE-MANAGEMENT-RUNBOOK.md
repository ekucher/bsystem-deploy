# Remote Management Runbook Baseline

This is an architectural runbook baseline. Production commands, DNS names, secrets and concrete firewall implementation must be added only after deployment decisions are approved.

## New device

1. Confirm customer and authorization.
2. Generate a short-lived customer-scoped enrollment token.
3. Run the signed bootstrap as Administrator on the target Windows Server.
4. Verify enrollment created one Device identity.
5. Verify local WG private key did not leave the device.
6. Verify management address allocation.
7. Verify WG peer/handshake.
8. Verify RDP + NLA and firewall restriction.
9. Verify Zabbix reporting.
10. Verify RustDesk fallback registration.
11. Verify desired/applied revision.
12. Confirm server-side health and MANAGED state.
13. Review audit events.

## Degraded device

Determine the failed channel independently:

- Windows/OS;
- WireGuard;
- RDP;
- customer Internet;
- Zabbix;
- RustDesk;
- Provisioning/Control Plane.

Do not classify a device offline solely because one provider is unavailable.

If WG/RDP fails while RustDesk is available, use RustDesk as fallback to repair primary management. If RustDesk fails while WG/RDP remains healthy, primary administration continues.

## Configuration drift

Compare desired and actual state plus `desiredRevision` / `appliedRevision`. Reconcile only resources proven to be BSYSTEM-managed. Never blindly delete provider objects.

## Revoke device

Privileged workflow:

1. deny/revoke network access;
2. disable WG peer;
3. disable RustDesk access/mapping;
4. adjust Zabbix monitoring if appropriate;
5. disable management DNS mapping if present;
6. mark REVOKED;
7. preserve historical record;
8. verify audit evidence.

## Revoke engineer

1. revoke Authentik/platform access;
2. deny authorization;
3. revoke engineer WG peer/network policy;
4. verify no customer-server reconfiguration is required.

## Control-plane outage

Existing data plane should continue using last applied configuration:

- WireGuard;
- RDP;
- Zabbix;
- RustDesk.

New enrollment, desired-state changes and some management actions may be unavailable. Do not attempt destructive fleet-wide changes solely because the control plane is down.

## Backup requirements

Back up:

- Remote Management PostgreSQL;
- WG Hub server keys/configuration;
- RustDesk critical server keys/configuration;
- deployment configuration.

Device private WG keys do not need centralized backup when the lifecycle supports re-enrollment/key rotation.

## Observability

Monitor structured logs, metrics, health/readiness, correlation IDs and audit integration. Alert especially on enrollment failures, reconciliation failures, IP-pool exhaustion, Zabbix/RustDesk API failures and ACL reconciliation failures.
