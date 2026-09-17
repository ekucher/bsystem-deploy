# authentik OIDC setup for BSYSTEM-HUB

This document completes P0.1 Identity & RBAC for the local Docker deployment.

## 1. Start the platform

```bash
cp .env.example .env
# set POSTGRES_PASSWORD and AUTHENTIK_SECRET_KEY
docker compose up -d --build
```

Open authentik at `http://localhost:9000`.

For a fresh installation, if the initial setup flow is not shown automatically, open:

```text
http://localhost:9000/if/flow/initial-setup/
```

Create the `akadmin` password.

## 2. Verify BSYSTEM groups

The mounted blueprint `authentik/blueprints/bsystem-groups.yaml` declares these groups:

- `BSYSTEM-Admins`
- `BSYSTEM-Managers`
- `BSYSTEM-Developers`
- `BSYSTEM-QA`
- `BSYSTEM-Support`
- `BSYSTEM-DevOps`
- `BSYSTEM-Customers`
- `BSYSTEM-Services` — service identities (`SVC-*`), not people

In authentik Admin, verify them under **Directory → Groups**. If the blueprint has not yet reconciled, check worker logs and the Blueprints page.

## 3. Create the BSYSTEM-HUB application and provider

In authentik Admin:

1. Go to **Applications → Applications**.
2. Create a new application.
3. Name: `BSYSTEM-HUB`.
4. Slug: `bsystem-hub`.
5. Create an **OAuth2/OIDC** provider for this application.
6. Client type: **Public**.
7. Enable **Authorization Code** flow and PKCE support.
8. Redirect URI for local DEV:

```text
http://localhost:8081/auth/callback
```

9. Post logout/launch URL can point to:

```text
http://localhost:8081/
```

10. Configure the provider with the default `openid`, `profile`, and `email` scope mappings. The authentik `profile` scope includes username, name, and group membership.

## 4. Copy the Client ID

Copy the generated OAuth2 Client ID from authentik and set it in `.env`:

```dotenv
VITE_OIDC_AUTHORITY=http://localhost:9000/application/o/bsystem-hub/
VITE_OIDC_CLIENT_ID=<client-id-from-authentik>
```

The authority uses authentik's default per-provider issuer mode.

Rebuild HUB because Vite injects these values during the frontend build:

```bash
docker compose up -d --build hub
```

## 5. Create a test user

Create a normal authentik user and add it to one or more BSYSTEM groups, for example:

```text
BSYSTEM-Developers
BSYSTEM-QA
```

Do not give ordinary BSYSTEM users authentik superuser status.

## 6. Test login

Open:

```text
http://localhost:8081
```

Click **Увійти через BSYSTEM Identity**.

Expected flow:

```text
BSYSTEM-HUB
  → authentik
  → Authorization Code + PKCE
  → /auth/callback
  → access token
  → /api/v1/me
  → Integration Core
  → authentik UserInfo
  → BSYSTEM RBAC
  → allowed modules
```

## 7. Backend authorization behavior

Integration Core does not trust the frontend's module list. For every protected API request it receives a bearer token and validates it by calling authentik's UserInfo endpoint over the internal Docker network.

Current mapping:

| authentik group | BSYSTEM role |
|---|---|
| BSYSTEM-Admins | Administrator |
| BSYSTEM-Managers | Manager |
| BSYSTEM-Developers | Developer |
| BSYSTEM-QA | QA |
| BSYSTEM-Support | Support |
| BSYSTEM-DevOps | DevOps |
| BSYSTEM-Customers | Customer |

A user can belong to multiple groups and receives the union of roles, permissions, and module access.

## 8. Useful checks

```bash
# public health endpoints
curl http://localhost:8080/health
curl http://localhost:8081/healthz

# protected API without a token must return 401
curl -i http://localhost:8080/api/v1/me

# logs
docker compose logs -f authentik-server authentik-worker integration-core hub
```

## Security notes

- Use HTTPS for STAGE and PROD.
- Never use the local `http://localhost` authority outside DEV.
- Do not store a client secret in the SPA; BSYSTEM-HUB uses a Public OAuth2/OIDC client with PKCE.
- Backend authorization is mandatory even when the frontend hides unavailable modules.
- Service-to-service authentication will use separate confidential clients/service accounts in a later phase.
