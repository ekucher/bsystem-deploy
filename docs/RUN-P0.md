# Run P0.1

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

## Verify

```bash
docker compose ps
curl http://localhost:8080/health
curl -i http://localhost:8080/api/v1/me
curl http://localhost:8081/healthz
```

Expected behavior:

- `/health` returns HTTP 200;
- `/healthz` returns HTTP 200;
- `/api/v1/me` without `Authorization: Bearer ...` returns HTTP 401;
- opening HUB redirects to authentik only after the user clicks the login button;
- after login, HUB shows only modules permitted by BSYSTEM RBAC.

Expected local endpoints:

- BSYSTEM HUB: `http://localhost:8081`
- Integration Core: `http://localhost:8080`
- authentik: `http://localhost:9000`

## Logs

```bash
docker compose logs -f --tail=200
```

Identity/RBAC troubleshooting:

```bash
docker compose logs -f authentik-server authentik-worker integration-core hub
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

## Implemented in P0.1

- authentik deployed as central Identity Provider;
- BSYSTEM groups managed through a mounted authentik blueprint;
- BSYSTEM-HUB uses OIDC Authorization Code + PKCE;
- bearer token is forwarded only to the same-origin `/api` path;
- Integration Core validates the access token through authentik UserInfo;
- `/api/v1/me` resolves groups to BSYSTEM roles and permissions;
- `/api/v1/modules` returns only authorized modules;
- nginx proxies HUB `/api/*` to Integration Core over Docker networking;
- backend authorization remains authoritative.

## Next increment

P0.2 should add persistent user/global-ID mapping, audit events, service registry persistence, tests, and CI gates.
