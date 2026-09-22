# Remote Management Security

## Authorization

Authentik is the engineer identity provider. Integration Core / the Remote Management backend performs server-side authorization.

Authorization chain:

```text
Engineer identity
 -> role/group
 -> customer/device permission
 -> desired network policy
 -> WG Hub ACL enforcement
```

An IP address is an enforcement representation, not the business authorization rule.

Sensitive operations requiring backend authorization include:

- generate/revoke enrollment;
- grant/revoke engineer access;
- launch RDP/RustDesk;
- change policy;
- rotate/revoke device;
- configuration changes.

## Core invariants

1. Deny by default.
2. No UI-only authorization.
3. Strict customer isolation.
4. No customer-to-customer lateral movement.
5. No public RDP.
6. No public Zabbix agent.
7. No inbound public ports required on managed customer servers.
8. Unique WireGuard identity per engineer/direct device.
9. No shared permanent RustDesk password.
10. No long-lived administrative credential embedded in installer.
11. No secrets in Git, logs, audit events or NATS.
12. Device private WireGuard keys are generated locally where practical.
13. Managed customer endpoints are treated as potentially compromised.
14. Control-plane failure does not automatically kill established data-plane access.

## Enrollment authority

Enrollment tokens are customer-scoped, short-lived, usage-limited, revocable, hashed server-side and audited. They authorize device registration only.

After enrollment, the device receives its own bounded credential for heartbeat/configuration/health. It must not manage other devices.

mTLS/device certificates are a desirable future hardening step.

## RustDesk credentials

Do not use one permanent password across devices. If a permanent credential is necessary, it must be unique per device and stored in a dedicated credential/secrets system, not plaintext in HUB DB.

## RDP credentials

VPN authorization and Windows login authority are separate. Prefer personal Windows administrative accounts. Do not standardize one shared Administrator password across the fleet. PAM/credential brokering is a separate future capability.

## Audit

Audit events include enrollment creation/revocation, provisioning, peer lifecycle, engineer access, provider registration/revocation, policy changes, device revoke and remote-access launch requests.

Audit records contain actor Global ID, customer, device, action, timestamp, result, correlation ID and safe metadata. Never log secrets.

## Supply chain

Installer/agent releases should have code signing, versioning, SHA256, release provenance and a controlled download endpoint. Update mechanisms must verify integrity/signature.
