# Deployment

`README.md` gets a stack running. This document is about the decisions a real
deployment has to make, and the ones the repository has already made for it.

## What is deployed

```text
frontend ── hub (nginx, uid 101)
       └── integration-core ── backend ── nats
                           └── data ──── postgres
                                    └─── authentik
```

Three networks, two of them `internal`. `data` and `backend` have no route off
the host, so the database, the event bus and authentik's own store are not
reachable from outside the stack even if a published port were misconfigured.
Only `frontend` carries anything published.

## Published ports bind to loopback

Every published port binds `${BIND_ADDRESS:-127.0.0.1}`.

A `docker compose up` on a host with a public interface therefore exposes
nothing. That default is chosen for the mistake it prevents rather than for the
convenience it offers: the failure mode of the other default is an authentik
admin interface on the open internet, discovered by somebody else.

**Put a TLS-terminating proxy in front and set `BIND_ADDRESS` deliberately.**
The stack serves plain HTTP; it is not the thing that should be facing the
network.

## Container hardening

Every service sets `no-new-privileges` and drops `ALL` capabilities. Where an
image's entrypoint genuinely needs some back to drop its own privileges —
postgres — exactly those are re-added and the reason is written at
the service. The stateless services (Integration Core, HUB, the mock upstreams)
run on a read-only root filesystem.

authentik keeps only `no-new-privileges`. A capability set for it belongs in an
owner-run pass against a real deployment rather than in an unverified guess
committed here.

`scripts/check-hardening.py` runs in CI and fails on a regression in any of
this — including the read-only root filesystems, which it did not check until
the paragraph above was compared against it. Trivy's misconfiguration scanner
has no Docker Compose rules, so that script — not the scanner — is what keeps
the settings from drifting. The E2E stack is additionally started for real on
every push, so a capability set that breaks a container fails the build rather
than production.

See [`../SECURITY.md`](../SECURITY.md) for the current per-service state.

## Configuration and secrets

`.env` is the only place credentials live, it is gitignored, and
`.env.example` carries nothing but `CHANGE_ME` placeholders.

```bash
cp .env.example .env
openssl rand -base64 36   # POSTGRES_PASSWORD
openssl rand -base64 60   # AUTHENTIK_SECRET_KEY
```

The AI gateway's provider credentials (`OPENAI_API_KEY`) are read from the
environment and nowhere else. An unset key means the provider is not
configured, which the platform treats differently from a configured one that
is empty: it falls back to the fake provider with a warning rather than
sending prompts somewhere unintended.

## Order of operations

1. **authentik first.** Groups (`BSYSTEM-Admins` and the rest) must exist
   before anyone signs in, because the platform maps groups to roles and a
   user in no known group resolves to no permissions at all. See
   [AUTHENTIK-OIDC.md](AUTHENTIK-OIDC.md).
2. **Postgres.** The Integration Core runs its own migrations at startup, in
   filename order. They are additive by policy, so the previous binary can run
   against the newer schema — which is what makes a rollback survivable.
3. **Integration Core**, then **HUB**. The HUB is a static bundle and will
   render before the platform is up; it reports the failure rather than
   showing an empty page.

## Upgrading

Roll the Integration Core before the HUB. The API is additive and the HUB
tolerates fields it does not know, so the reverse order is survivable but
pointless — a HUB asking for an endpoint that does not exist yet is a
user-visible error for no reason.

**Migrations are additive and there are no down migrations.** A rollback moves
the binary back, not the schema. That is why nothing may drop a column or a
table autonomously: adding one that nothing reads yet is free, removing one
that something still reads is an outage that a rollback cannot fix.

## Verifying a deployment

```bash
scripts/smoke-p0.sh
```

Then check `/readyz` on the Integration Core, which reports its dependencies
rather than just answering. A healthy `/health` with a failing `/readyz` means
the service is running and cannot do its job — the distinction a load balancer
needs.

`/metrics` exposes the platform's own counters; see
[`../observability/README.md`](../observability/README.md) and the Grafana
dashboard beside it.

## What is not automated

Production deployment, DNS, TLS issuance, credential rotation and authentik
configuration are deliberately outside this repository's reach. They are
recorded as owner actions in `TASKS.md`, and the reason is that each one is
irreversible in a way a compose file is not.
