# Security Policy

`bsystem-deploy` controls platform exposure and therefore is security-sensitive.

## Mandatory rules

- Never commit production secrets, tokens, passwords, private keys, certificates with private material, or database credentials.
- Use `.env.example` only with non-sensitive placeholders.
- Production services must use HTTPS.
- Databases, Redis, NATS and other internal backends must not be exposed to untrusted networks.
- Restrict management interfaces to approved networks/VPN where applicable.
- Pin production image versions; avoid `latest`.
- Apply least privilege to containers and service accounts.
- Back up stateful services before upgrades.
- Test restore procedures periodically.
- Keep DEV/STAGE/PROD credentials isolated.
- Record security-relevant deployment changes through version control and review.

## Container hardening targets

Where supported:

- run as non-root;
- use read-only filesystem for stateless services;
- drop unnecessary Linux capabilities;
- define CPU/memory limits;
- use health checks;
- avoid mounting Docker socket into application containers;
- minimize published ports.

## AI deployment

Local/cloud model credentials are secrets. AI-facing services must be isolated from raw secret stores and must receive only authorized, sanitized context.

## Reporting

Do not publish exploitable security findings in public issues. Use the repository owner's private security process when configured.
