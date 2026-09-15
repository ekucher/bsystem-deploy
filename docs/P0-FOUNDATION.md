# BSYSTEM Deploy P0 — Foundation

## Goal

Provide a reproducible Docker Compose environment for the first BSYSTEM platform milestone.

## P0 services

Required:

- reverse proxy
- authentik
- PostgreSQL
- Redis
- `bsystem-hub`
- `bsystem-integration-core`

Optional until required by features:

- NATS
- OpenSearch
- Prometheus
- Grafana
- Loki

## Network model

Use separate Docker networks:

```text
edge        reverse proxy -> published web services
platform    HUB <-> Integration Core <-> internal platform services
data        application services -> PostgreSQL/Redis
```

PostgreSQL and Redis must not be published to the public interface.

## Recommended DNS names

Development examples:

- `hub.dev.bsystem.local`
- `auth.dev.bsystem.local`

Future modules:

- `crm.dev.bsystem.local`
- `projects.dev.bsystem.local`
- `qa.dev.bsystem.local`
- `wiki.dev.bsystem.local`
- `ops.dev.bsystem.local`

## Environment files

Repositories may contain `.env.example`, but never `.env` with real secrets.

Expected classes of configuration:

```text
POSTGRES_*
REDIS_*
AUTHENTIK_*
OIDC_*
HUB_*
INTEGRATION_CORE_*
```

## Persistence

Durable Docker volumes are required for:

- PostgreSQL
- authentik media/data where applicable
- future search index if reconstruction is not desirable

Redis should be treated as reconstructable unless a feature explicitly requires persistence.

## Backup baseline

P0 must document and test restoration of:

- PostgreSQL databases
- authentik configuration/state
- deployment configuration
- external secrets source references

## Health checks

Compose healthchecks should be defined for all services that expose health endpoints. Startup ordering must rely on health/readiness where required, not arbitrary sleep timers.

## Startup target

A fresh DEV host should be able to reach a functional login flow using only documented prerequisites, repository checkout, environment configuration and Docker Compose commands.

## Definition of Done

- `docker compose config` succeeds
- all required P0 services start
- authentik login page is reachable through reverse proxy
- HUB is reachable through HTTPS/DEV TLS policy
- HUB can complete OIDC login
- HUB can reach Integration Core internally
- PostgreSQL/Redis are not exposed publicly
- containers have restart policies
- no real credentials exist in Git
- backup/restore procedure is documented
