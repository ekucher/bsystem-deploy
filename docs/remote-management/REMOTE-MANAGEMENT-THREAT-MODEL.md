# Remote Management Threat Model

## Trust assumptions

A managed customer server is a potentially compromised endpoint. Enrollment metadata/fingerprints assist inventory and duplicate detection but do not replace authentication.

## Minimum threats

- stolen engineer laptop;
- stolen WireGuard configuration;
- compromised customer server;
- compromised enrollment token;
- malicious customer server;
- compromised RustDesk credential;
- Provisioning API compromise;
- WireGuard Hub compromise;
- customer-to-customer lateral movement;
- privilege escalation;
- token replay;
- device impersonation;
- installer/update supply-chain compromise.

## Required mitigations

### Engineer compromise

Use unique engineer identity/peer. Revoking an engineer in Authentik/platform authorization must lead to revocation of that engineer's remote-management network access without changing every customer server.

### Customer endpoint compromise

Hub firewall remains deny-by-default. A customer device cannot reach another customer, engineer network, BSYSTEM internal LAN, databases or administrative endpoints unless a narrowly defined flow explicitly requires it.

### Enrollment token theft/replay

Tokens are short-lived, customer-scoped, usage-limited, revocable, hashed at rest and audited. A consumed one-time token cannot enroll another device.

### Device impersonation

Persistent Device identity is separate from hostname. Direct devices use unique WG public keys; duplicate MachineId/WG key/external mappings must be detected and constrained.

### Secret leakage

Do not log private keys, passwords, full enrollment tokens, access tokens or device credentials. Do not publish them to NATS or commit them to Git.

### Supply chain

Sign/version installer and agent releases, publish SHA256/provenance and verify update integrity/signatures.

## Acceptance scenarios

Security tests must demonstrate:

- unauthorized engineer cannot access device;
- Customer A cannot access Customer B;
- revoked engineer loses network access;
- revoked device cannot reconnect;
- expired token cannot enroll;
- consumed one-time token cannot enroll another device;
- duplicate enrollment does not allocate another IP;
- private key never appears in API logs;
- unauthorized API call is denied server-side;
- UI hiding is not relied upon;
- hub firewall remains deny-by-default.
