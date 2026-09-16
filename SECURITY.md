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
`govulncheck` and `staticcheck` over every Go module, Trivy against the
Dockerfiles and working tree, and a Trivy image scan of a built mock upstream
with a published CycloneDX SBOM.

"Every Go module" is checked rather than asserted. The matrix that decides
what those two scanners see is written by hand, and a module absent from it
produces no failure and no output — an unscanned module and a clean one print
the same nothing. `loadtest` sat in exactly that state: compiled by CI, shipped
by this repository, scanned by neither. `scripts/check-scan-coverage.py`
compares the modules that exist against the modules each matrix names, in both
directions, so a new module cannot be quietly uncovered and a matrix entry
cannot go on naming a module that is gone.

## Supply chain

### Action pinning

Every workflow across the four repositories pins its actions to a **major tag**
— `actions/checkout@v7`, `aquasecurity/trivy-action@v0.36.0`,
`gitleaks/gitleaks-action@v2` — not to an immutable commit SHA.

A major tag is mutable. Whoever controls the action's repository can move it,
so pinning this way is a decision to trust each action's publisher on every run
rather than to pin the code that was reviewed. That is the weaker of the two
positions, and it is stated here so it is a choice rather than an oversight:
the actions in use are GitHub's own, Aqua's and Gitleaks', the workflows hold
no deployment credential, and `contents: read` is the default permission. SHA
pinning without automated bumping goes stale, and a stale security scanner is
its own risk.

Revisit this before any workflow is given a token that can write to a package
registry, a deployment target or another repository. At that point the trade
changes and the pins should become SHAs with an updater.

The tags have also drifted apart, which is worth fixing on its own terms: a
reader cannot tell a deliberate difference from an unnoticed one.

| Action | bsystem-deploy | bsystem-integration-core | bsystem-hub | bsystem-design-system |
| --- | --- | --- | --- | --- |
| `actions/checkout` | v7 | v4 | v7 | v4 |
| `actions/setup-node` | — | v4 | v7 | v4 |
| `actions/setup-go` | v5 | v5 | — | — |
| `aquasecurity/trivy-action` | v0.36.0 | v0.36.0 | v0.36.0 | v0.36.0 |
| `gitleaks/gitleaks-action` | v2 | v2 | v2 | v2 |
| `github/codeql-action` | — | v3 | v3 | v3 |

### Base images

Base images are pinned to a version tag and not to a digest: `golang:1.26-alpine`,
`alpine:3.22`, `node:22-alpine`, `nginxinc/nginx-unprivileged:1.30-alpine`, and
the service images in Compose. The same reasoning applies — a tag that keeps
receiving patch updates is what makes a rebuild pick up a fixed CVE, and Trivy
scans what was actually built. A digest pin without an updater freezes the
vulnerabilities along with the version.

### What the SBOM covers

The SBOM published by this repository is generated from an image built inside
the `Security` job, not from the image the E2E job ran. The four mock upstreams
share one Dockerfile and one module, so their dependency sets are identical and
one SBOM describes them all — but it is a rebuild of the same source, not the
tested artefact. Treat it as an answer to "were we shipping this dependency at
this commit", which is the question an advisory raises, and not as a bill of
materials for a specific running container.

### CodeQL

CodeQL runs in Integration Core, HUB and the Design System, and deliberately
not here. This repository's Go is mock upstreams, the E2E harness and the load
harness — none of it serves a request from a real caller or handles a real
credential. It gets `govulncheck` and `staticcheck`; the product surfaces get
CodeQL as well. Recorded because an absent job otherwise reads as an omission.

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
up on a host with a public interface does not expose it by default. The stage
overlay replaces every port mapping and reads `${STAGE_PUBLISH_ADDRESS:-127.0.0.1}`
instead — same default, different variable, and `BIND_ADDRESS` does nothing
there.

## AI deployment

Local/cloud model credentials are secrets. AI-facing services must be isolated from raw secret stores and must receive only authorized, sanitized context.

## Reporting

Do not publish exploitable security findings in public issues. Use the repository owner's private security process when configured.
