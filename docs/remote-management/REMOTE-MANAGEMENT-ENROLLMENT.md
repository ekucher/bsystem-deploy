# Remote Management Enrollment

## Goal

A new managed Windows Server must be enrolled without manually editing WireGuard, Zabbix or RustDesk configuration.

## Enrollment token

Token properties:

- customer-scoped;
- short-lived;
- usage-limited;
- revocable;
- stored as a server-side hash;
- audited.

The bootstrap token grants only registration authority and expires/loses authority after successful enrollment.

## Flow

```text
Administrator
 -> BSYSTEM-HUB
 -> Generate enrollment
 -> short-lived token
 -> Windows Server bootstrap
 -> Provisioning API
 -> persistent Device identity
 -> local WG key generation
 -> desired configuration
 -> component provisioning
 -> completion report
 -> server-side validation
 -> MANAGED
```

## Bootstrap responsibilities

- preflight;
- enrollment/device registration;
- local WireGuard key generation;
- management address/configuration;
- WireGuard installation;
- RDP/NLA/firewall configuration;
- Zabbix Agent 2 installation/configuration;
- RustDesk installation/configuration;
- health checks;
- completion callback.

## Preflight

Check administrator rights, supported Windows version, architecture, PowerShell, Internet/DNS/time, disk space, endpoint reachability and existing WireGuard/RustDesk/Zabbix/RDP/firewall state.

## Idempotency

Repeating bootstrap/repair must not create:

- duplicate Device records;
- another management IP;
- duplicate WG peers;
- duplicate firewall rules;
- duplicate Zabbix hosts;
- duplicate RustDesk mappings.

Database uniqueness/transactions must enforce concurrency correctness for Global Device ID, management IP, WG public key, active external mappings and token hash.

Provisioning operations should support idempotency keys.

## Partial failures

Do not blindly roll back successful external steps.

Example:

```text
WG       OK
Zabbix   OK
RustDesk FAILED
```

The device remains PROVISIONING/DEGRADED with a precise reason. Retry resumes/reconciles the missing step.

## Repair and test

Provide bounded workflows equivalent to:

- `Repair-BSYSTEMRemote`: reuse existing Device identity, reconcile and repair;
- `Test-BSYSTEMRemote`: verify WG service/address/handshake, RDP, Zabbix, RustDesk, API and configuration revision.

## Revocation

Revocation is privileged and historical records are retained.

Expected effects include disabling WG peer/ACL, disabling RustDesk access/mapping, adjusting Zabbix monitoring where appropriate, disabling DNS mapping, setting REVOKED and writing audit evidence.

Local uninstall, device revoke and decommission are distinct operations.
