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

On the stage stack the variable is `STAGE_PUBLISH_ADDRESS`: the overlay
replaces every port mapping, so `BIND_ADDRESS` is read by nothing there. The
default is the same loopback either way, which is what makes the difference
easy to miss — it shows up only when somebody tries to widen the exposure
deliberately and sets the variable that does not govern.
`scripts/stage-preflight.sh` fails on a stage stack published on every
interface and warns when `BIND_ADDRESS` is set for one.

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
   user in no known group resolves to no permissions at all. All eight are
   created by the blueprint mounted at `authentik/blueprints/bsystem-groups.yaml`,
   so this is a thing to verify rather than to do by hand. See
   [AUTHENTIK-OIDC.md](AUTHENTIK-OIDC.md).
2. **Postgres.** The Integration Core runs its own migrations at startup, in
   filename order. They are additive by policy, so the previous binary can run
   against the newer schema — which is what makes a rollback survivable.
3. **Integration Core**, then **HUB**. The HUB is a static bundle and will
   render before the platform is up; it reports the failure rather than
   showing an empty page.

## A host with no browser

Every published port binds loopback, so on a VM reached only over SSH there is
nothing to open. Do not widen `BIND_ADDRESS` to fix that: `/if/flow/initial-setup/`
is an unauthenticated page that creates the first administrator, and publishing
it on an interface somebody else can reach is the exact failure the loopback
default exists to prevent.

Forward the ports instead. From the machine with the browser:

```bash
ssh -L 9000:127.0.0.1:9000 -L 8080:127.0.0.1:8080 -L 8081:127.0.0.1:8081 user@host
```

The tunnel lasts as long as the session. `http://127.0.0.1:9000` in your own
browser is then authentik on the host, `8080` the Integration Core and `8081`
the HUB. Nothing in the stack changes, and the ports stay unreachable from
anywhere else.

**"Request has been denied" on the setup page means the administrator already
exists.** The flow is available only until the first user is created; after
that it refuses, which reads like a permissions problem and is not one. Sign in
at `http://127.0.0.1:9000/` instead.

### Bootstrapping without the setup page

`AUTHENTIK_BOOTSTRAP_EMAIL`, `AUTHENTIK_BOOTSTRAP_PASSWORD` and
`AUTHENTIK_BOOTSTRAP_TOKEN` create `akadmin` on the first start, so the setup

### BSYSTEM-HUB user administration credential

`BSYSTEM_AUTHENTIK_ADMIN_TOKEN` is separate from the bootstrap token. Set it
only when the HUB must create or modify human accounts. Generate at least
32 random bytes locally (for example `openssl rand -hex 32`) and keep it in
the host's untracked `.env`.

The authentik worker reconciles this value into a dedicated least-privilege
service-account token; Integration Core receives the same value for server-side
API calls. The authentik server container, HUB container and browser bundle do
not receive it. Leaving the value empty keeps account mutations disabled.

page is never served at all and the token is an API token to configure the rest
with. All three are empty by default and change nothing when unset.

They are worth it where no tunnel is possible, or where the deployment is
scripted. The cost is real and easy to miss: **authentik reads them on every
start, not only the first.** A password left in `.env` afterwards is a static
administrator credential sitting in a running container's environment, where
nothing else would mention it. Clear all three once the account exists —
`scripts/stage-preflight.sh` warns while they are set, and names the variable
rather than printing its value.

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
