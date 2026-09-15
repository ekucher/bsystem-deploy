# Run P0

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

## Start

```bash
docker compose up -d --build
```

## Verify

```bash
docker compose ps
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/modules
curl http://localhost:8081/healthz
```

Expected local endpoints:

- BSYSTEM HUB: `http://localhost:8081`
- Integration Core: `http://localhost:8080`
- authentik: `http://localhost:9000`

## Logs

```bash
docker compose logs -f --tail=200
```

For one service:

```bash
docker compose logs -f integration-core
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

## Current P0 limitations

- authentik is deployed but OIDC Application/Provider configuration is not yet automated.
- HUB renders the platform shell; real authenticated user/session state is the next increment.
- Integration Core currently exposes only health and module registry endpoints.
- PostgreSQL, Redis and NATS are infrastructure-ready but business persistence/events are not wired yet.
