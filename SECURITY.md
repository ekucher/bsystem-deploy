# Security Policy

`bsystem-deploy` controls platform exposure and therefore is security-sensitive.

## Mandatory rules

- Never commit production secrets, tokens, passwords, private keys, certificates with private material, or database credentials.
- Use `.env.example` only with non-sensitive placeholders.
- Production services must use HTTPS.
- Databases, NATS and other internal backends must not be exposed to untrusted networks.
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

## What is enforced automatically

`scripts/check-hardening.py` renders both Compose files with `docker compose
config` and fails CI when a service loses `no-new-privileges`, stops dropping
`ALL` capabilities, runs privileged, mounts the Docker socket, or publishes a
port on every interface. Trivy's misconfiguration scanner has no Docker
Compose rules, so this script — not the scanner — is what keeps the settings
below from regressing.

The `Security` workflow additionally runs Gitleaks over the full history,
`govulncheck` over the `mocks` and `e2e` modules, Trivy against the
Dockerfiles and working tree, and a Trivy image scan of a built mock upstream
with a published CycloneDX SBOM.

### Current state

| Service | no-new-privileges | capabilities | root filesystem |
| --- | --- | --- | --- |
| postgres | yes | `ALL` dropped, five re-added for the entrypoint's privilege drop | writable (PGDATA) |
| nats | yes | `ALL` dropped | writable (JetStream store) |
| authentik-server, authentik-worker | yes | left to the image | writable |
| integration-core | yes | `ALL` dropped | read-only |
| hub (nginx-unprivileged) | yes | `ALL` dropped | read-only, tmpfs for `/tmp` |
| mock upstreams | yes | `ALL` dropped | read-only (scratch image) |

The settings in `docker-compose.e2e.yml` are exercised on every push: the E2E
job starts that stack and runs the scenario suite against it, so a capability
set that breaks a container fails CI. The equivalent settings in
`docker-compose.yml` are only validated as configuration — its services are
not run in CI, and authentik in particular is left to the image's entrypoint
rather than given an unverified capability set.

Published ports bind to `${BIND_ADDRESS:-127.0.0.1}`, so bringing the stack
up on a host with a public interface does not expose it by default.

## AI deployment

Local/cloud model credentials are secrets. AI-facing services must be isolated from raw secret stores and must receive only authorized, sanitized context.

## Reporting

Do not publish exploitable security findings in public issues. Use the repository owner's private security process when configured.
