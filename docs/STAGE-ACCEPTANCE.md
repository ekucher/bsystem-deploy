# Stage environment contract

What a stage deployment of BSYSTEM needs, and what the owner has to supply.

This document is the input list for a controlled acceptance. It contains **no
real value for anything** — every credential, hostname and URL below is a
placeholder. Filling them in is the owner's step, and nothing here should ever
be committed with a real value in it.

## Services

Nine services make up a stage deployment. Six are BSYSTEM's own; three are the
source systems BSYSTEM reads and never writes.

| Service | Owned by | Required | Purpose |
| --- | --- | --- | --- |
| authentik | platform | yes | identity: human sign-in and service identities |
| Integration Core | platform | yes | normalized API, authorization, adapters, events |
| HUB | platform | yes | the browser application |
| PostgreSQL | platform | yes | mappings, RBAC, audit, notifications, support, AI audit |
| NATS | platform | no | event bus; absent means degraded, not down |
| EspoCRM | source | no | authoritative for clients and contacts |
| Redmine | source | no | authoritative for projects and issues |
| Outline | source | no | authoritative for documents |

A source system is optional in the literal sense that the platform starts and
serves without it: the adapter stays disabled when its URL is empty, and the
rest of the platform works. The endpoints that adapter backs answer **`503
upstream_unavailable`**, for collections as well as for detail reads.

That is deliberate, and the tempting alternative would be worse: an empty
collection would tell an operator the platform knows of no clients, when what
is true is that nobody has told it where to look. A missing integration must
not be indistinguishable from missing data.

So a missing Redmine does not stop you accepting the CRM path — but expect
`503` from `/api/v1/projects` and `/api/v1/issues` while it is missing, not an
empty list. The smoke runner accepts that code for exactly this reason.

NATS is the one degraded-but-alive dependency: `/readyz` reports `nats:
degraded` and the platform keeps serving. Events are not queued while it is
absent; they are dropped. That is worth knowing before an acceptance concludes
that eventing works.

## Environment variables

Classification:

- **required** — the platform will not start, or will not work, without it
- **optional** — absence disables a feature rather than breaking one
- **secret** — must never be committed, logged, or pasted into a ticket
- **public config** — safe in a repository, a diagram or a screenshot

### Platform

| Variable | Class | Required | Default | Notes |
| --- | --- | --- | --- | --- |
| `STAGE_PUBLISH_ADDRESS` | public config | optional | `127.0.0.1` | Interface **the stage stack** publishes authentik, the Core and the HUB on. Leave as the loopback default behind a TLS proxy; widening it is a deliberate act, and `scripts/stage-preflight.sh` fails on `0.0.0.0`. |
| `BIND_ADDRESS` | public config | not read in stage | — | Governs the **base** stack only. The stage overlay replaces every port mapping, so setting this here changes nothing; the preflight warns if you have. |
| `POSTGRES_PASSWORD` | **secret** | **required** | — | Also embedded in the Core's `DATABASE_URL` by Compose. |
| `AUTHENTIK_SECRET_KEY` | **secret** | **required** | — | ≥50 random characters. Rotating it invalidates existing sessions. |
| `VITE_OIDC_AUTHORITY` | public config | **required** | — | Browser-visible issuer URL. Baked into the HUB **at build time**, not read at runtime. |
| `VITE_OIDC_CLIENT_ID` | public config | **required** | — | Public OIDC client id. Public by design in a PKCE flow. Also build-time. |

That `VITE_*` values are build-time arguments is the single most common stage
surprise in this list: changing them requires rebuilding the HUB image, not
restarting it.

### Integration Core

| Variable | Class | Required | Default | Notes |
| --- | --- | --- | --- | --- |
| `HTTP_ADDR` | public config | optional | `:8080` | Listen address. |
| `DATABASE_URL` | **secret** | **required** | — | Carries the password. Never log it; the mapping audit redacts it from errors. |
| `NATS_URL` | public config | optional | — | Empty disables eventing. |
| `AUTHENTIK_USERINFO_URL` | public config | **required** | — | Server-to-server UserInfo endpoint. Must be reachable **from the Core container**, which is not the same as from your laptop. |
| `LOG_LEVEL` | public config | optional | `info` | |
| `DATABASE_MAX_CONNS` | public config | optional | driver default | Bounded at 500; an out-of-range value falls back rather than failing. |
| `DATABASE_MIN_CONNS` | public config | optional | driver default | Same bounding. |
| `HTTP_READ_HEADER_TIMEOUT` | public config | optional | `5s` | |
| `HTTP_READ_TIMEOUT` | public config | optional | `15s` | |
| `HTTP_WRITE_TIMEOUT` | public config | optional | `60s` | Raised automatically if `AI_TIMEOUT` would exceed it. |
| `HTTP_IDLE_TIMEOUT` | public config | optional | `60s` | |
| `HTTP_SHUTDOWN_GRACE` | public config | optional | `10s` | |
| `ADAPTER_TIMEOUT` | public config | optional | `10s` | Per upstream attempt. |
| `ADAPTER_CIRCUIT_OPEN_FOR` | public config | optional | `30s` | |

### Source systems

| Variable | Class | Required | Notes |
| --- | --- | --- | --- |
| `ESPOCRM_URL` | public config | optional | Empty disables the adapter. |
| `ESPOCRM_API_KEY` | **secret** | optional | `X-Api-Key`. API user with read-only access to Accounts and Contacts. |
| `REDMINE_URL` | public config | optional | Empty disables the adapter. |
| `REDMINE_API_KEY` | **secret** | optional | `X-Redmine-API-Key`. **The REST API must be enabled in Redmine itself.** |
| `OUTLINE_URL` | public config | optional | Empty disables the adapter. |
| `OUTLINE_API_KEY` | **secret** | **required when `OUTLINE_URL` is set** | Bearer. Outline has no anonymous read surface, so the adapter refuses to construct without it. |

### Optional subsystems

| Variable | Class | Notes |
| --- | --- | --- |
| `OPENSEARCH_URL`, `OPENSEARCH_INDEX` | public config | Empty keeps the in-memory search provider. |
| `AI_PROVIDER` | public config | `fake` (default), `ollama` or `openai`. |
| `AI_TIMEOUT` | public config | Bounds one provider call. |
| `OLLAMA_URL`, `OLLAMA_MODEL` | public config | |
| `OPENAI_URL`, `OPENAI_MODEL` | public config | |
| `OPENAI_API_KEY` | **secret** | Only when `AI_PROVIDER=openai`. |

Per the platform's classification, a `CREDENTIAL` is never sent to an LLM. If
an AI provider is enabled for the acceptance, that boundary is part of what is
being accepted — see `bsystem-integration-core/docs/AI-GATEWAY-CONTRACT.md`.

## No real secret is committed

`.env` is ignored; `.env.example` holds `CHANGE_ME_*` placeholders. CI runs
gitleaks, GitGuardian and Trivy on every push, and `scripts/stage-preflight.sh`
fails when a required variable still holds a placeholder — so a half-filled
`.env` is caught before it reaches a service rather than after.

## Network paths

```text
browser ──TLS──▶ reverse proxy ──▶ HUB            (static assets)
browser ──TLS──▶ reverse proxy ──▶ authentik      (OIDC redirect, token)
browser ──TLS──▶ reverse proxy ──▶ Integration Core  /api/v1/*

Integration Core ──▶ authentik     UserInfo (server-to-server)
Integration Core ──▶ PostgreSQL    internal network only
Integration Core ──▶ NATS          internal network only
Integration Core ──▶ EspoCRM / Redmine / Outline   egress
authentik        ──▶ PostgreSQL                    internal network only
```

Two properties hold in the Compose topology and should hold in stage:

- **PostgreSQL and NATS are on internal networks** and are not published.
  Nothing outside the stack reaches them.
- **The browser talks to authentik and the Core directly**, not through the HUB.
  The HUB is static files; it holds no token and proxies no API call in stage.

The direction that catches people is `Integration Core → authentik`: it needs
a URL that resolves **inside the container network**, which is usually not the
public issuer URL the browser uses.

## DNS names (placeholders)

| Purpose | Placeholder | Notes |
| --- | --- | --- |
| HUB | `hub.stage.example` | browser-facing |
| Integration Core | `api.stage.example` | browser-facing |
| authentik | `id.stage.example` | browser-facing, must match the issuer exactly |
| EspoCRM | `crm.stage.example` | egress only |
| Redmine | `pm.stage.example` | egress only |
| Outline | `wiki.stage.example` | egress only |

`.example` is reserved for documentation (RFC 2606). Replace all six; do not
commit the replacements.

The issuer URL must match **exactly**, including the trailing slash. An issuer
mismatch produces a token the platform rejects with no useful message, and it
is the most common OIDC misconfiguration there is.

## Health and readiness

| Check | URL | Auth | Meaning |
| --- | --- | --- | --- |
| Core liveness | `/health` | none | process is up; reports database and NATS as `ok` or `degraded` |
| Core readiness | `/readyz` | none | `503` when the database is unreachable; `200` with `nats: degraded` or `adapter:*: circuit_open` otherwise |
| Core metrics | `/metrics` | none | Prometheus text, including `bsystem_build_info` and `bsystem_schema_migrations_applied` |
| HUB | `/healthz` | none | nginx is serving |
| authentik | `/application/o/<app-slug>/.well-known/openid-configuration` | none | discovery document |
| Adapter health | `/api/service/v1/adapters/health` | **service token** | per-adapter reachability and credential |

Adapter health is on the **machine** API and a human token cannot reach it.
That is deliberate and is not to be relaxed for the convenience of an
acceptance: the smoke runner marks it skipped when no service token is
available rather than asking for the boundary to be moved.

## Rollback assumptions

- Every migration is additive; no down-migration exists. Rolling back **code**
  means deploying the previous image — the extra tables are inert to it.
- Rolling back a bad **migration** means restoring the database. That is why a
  verified backup is a prerequisite for the acceptance, not a follow-up.
- The Core re-applies migrations on every startup, so it must be **stopped**
  before a restore, or it will migrate the restored database straight back up.
- Rotating `AUTHENTIK_SECRET_KEY` invalidates sessions; it is not a rollback
  step and should not be done during one.
- Global IDs are immutable. A restore that loses `global_entities` does not
  merely lose rows: it breaks every reference held against those ids elsewhere.

## Backup prerequisites

Before first startup, and before any acceptance that writes:

1. A PostgreSQL backup that has been **restored at least once**, into a scratch
   database. An unverified backup is a belief, not a backup.
2. A copy of authentik's persistent media and certificate volumes.
3. A record of which image tags and commits the environment is running —
   `scripts/release-manifest.sh` generates it.
4. Confirmation that the backup does **not** live on the same volume, or the
   same host, as the data it protects.

The full procedure is in [`BACKUP-RESTORE.md`](BACKUP-RESTORE.md).

## What is still blocked

These cannot be resolved from the repository and are the owner's inputs:

- real stage URLs and DNS for all six names
- `POSTGRES_PASSWORD`, `AUTHENTIK_SECRET_KEY`
- the authentik application, provider, client id and group mapping
- source system URLs and read-only API credentials
- the authoritative customer ownership mapping (which client each account is)
- SLA response and resolution targets per severity

Until the ownership mapping exists, every customer-scoped case in
[`TENANT-ISOLATION-MATRIX.md`](TENANT-ISOLATION-MATRIX.md) is marked BLOCKED
rather than guessed. Unknown ownership denies; it does not broaden.
