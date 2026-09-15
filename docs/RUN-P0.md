# Run P0.2

## Repository layout

Clone the repositories side-by-side:

```text
workspace/
├── bsystem-hub/
├── bsystem-integration-core/
└── bsystem-deploy/
```

## Prepare configuration

```bash
cd bsystem-deploy
cp .env.example .env
```

Replace every `CHANGE_ME` value in `.env`.

Recommended secret generation examples:

```bash
openssl rand -base64 36
openssl rand -base64 60
```

Do not commit `.env`.

For the first startup, the authentik Client ID does not exist yet. Start infrastructure and authentik first:

```bash
docker compose up -d postgres redis nats authentik-server authentik-worker integration-core
```

Complete authentik initial setup, create the `BSYSTEM-HUB` OAuth2/OIDC Public provider, then put its Client ID into `.env`. See [AUTHENTIK-OIDC.md](AUTHENTIK-OIDC.md).

Build and start HUB:

```bash
docker compose up -d --build hub
```

Or rebuild the entire stack:

```bash
docker compose up -d --build
```

## Verify containers

```bash
docker compose ps
curl http://localhost:8080/health
curl -i http://localhost:8080/api/v1/me
curl http://localhost:8081/healthz
```

Expected behavior:

- `/health` returns HTTP 200 when PostgreSQL is available;
- the health response reports `database` and `nats` dependency state;
- `/healthz` returns HTTP 200;
- `/api/v1/me` without `Authorization: Bearer ...` returns HTTP 401;
- after login, HUB shows only modules permitted by BSYSTEM RBAC;
- the first successful authenticated request allocates a stable `USR-*` Global ID.

Expected local endpoints:

- BSYSTEM HUB: `http://localhost:8081`
- Integration Core: `http://localhost:8080`
- authentik: `http://localhost:9000`

## Automated smoke check

```bash
chmod +x scripts/smoke-p0.sh
./scripts/smoke-p0.sh
```

The smoke test checks Integration Core, HUB, authentik liveness and verifies that unauthenticated `/api/v1/me` is protected with HTTP 401.

## OIDC validation

After completing [AUTHENTIK-OIDC.md](AUTHENTIK-OIDC.md), open:

```text
http://localhost:8081
```

After login, HUB calls:

```text
GET /api/v1/me
GET /api/v1/modules
```

A new user receives a persistent identifier such as:

```text
USR-000001
```

Changing the email, username or group membership must not change this ID.

## P0.2 persistence API

Administrator-only endpoints:

```text
POST /api/v1/global-ids
GET  /api/v1/global-ids/{id}
GET  /api/v1/audit
```

Example Global ID request:

```json
{
  "entity_type": "client",
  "source": "espocrm",
  "source_id": "example-source-id"
}
```

Expected ID format:

```text
CL-000001
```

Repeated requests for the same `(source, entity_type, source_id)` must return the same Global ID.

## PostgreSQL persistence

Integration Core automatically applies its embedded P0.2 schema to `bsystem_integration` on startup.

Platform-owned tables:

```text
identities
global_id_counters
global_entities
modules
audit_events
```

Do not add CRM, Redmine, QA or Wiki business tables to this database.

## Logs

```bash
docker compose logs -f --tail=200
```

Identity/persistence troubleshooting:

```bash
docker compose logs -f postgres nats authentik-server authentik-worker integration-core hub
```

## Stop

```bash
docker compose down
```

To remove local P0 data as well:

```bash
docker compose down -v
```

Use `-v` only when destroying the local databases is intentional.

## Implemented through P0.2

- authentik central Identity Provider;
- OIDC Authorization Code + PKCE;
- backend-enforced RBAC;
- persistent authentik identity mapping;
- stable `USR-*` identifiers;
- persistent module registry;
- transactional Global ID allocation;
- audit storage;
- NATS event publishing;
- dependency-aware `/health`;
- PostgreSQL integration tests in CI;
- Docker smoke test script.

## Remaining P0.2 validation

- CI must be green on the final Integration Core commit;
- `docker compose up -d --build` must be smoke-tested on the target Docker host;
- production reverse-proxy trust rules for `X-Forwarded-For` must be defined before Internet exposure.

## Next increment

P0.3: service identities, versioned migrations, normalized event envelopes, adapter SDK/contracts and the first real business integration (recommended: EspoCRM).
