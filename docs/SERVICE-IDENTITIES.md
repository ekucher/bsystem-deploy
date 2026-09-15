# BSYSTEM Service Identities

## Purpose

Machine-to-machine callers use separate identities from human users.

Examples:

```text
svc-hub
svc-integration
svc-redmine
svc-qa
svc-operations
```

All service identities must be members of:

```text
BSYSTEM-Services
```

## Security boundary

Human API:

```text
/api/v1/*
```

Service API:

```text
/api/service/v1/*
```

Service credentials must never be reused for interactive login. Human tokens must not be used for scheduled workers or adapters.

## authentik

Create a non-interactive service identity and issue an OAuth2/OIDC token using the deployment's approved machine-to-machine flow. The resulting identity must include `BSYSTEM-Services` in its group claims.

Do not store client secrets in Git. Use Docker secrets, an external secret manager, or protected environment variables.

## First request

On first accepted service request Integration Core creates a stable BSYSTEM identity:

```text
authentik subject -> SVC-000001
```

The token itself is never stored.

## Verification

Without a token:

```bash
curl -i http://localhost:8080/api/service/v1/whoami
```

Expected:

```text
HTTP/1.1 401 Unauthorized
```

With a valid service token:

```bash
curl \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8080/api/service/v1/whoami
```

Expected response includes a stable `SVC-*` ID.

Adapter discovery:

```bash
curl \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8080/api/service/v1/adapters
```

P0.3 initially exposes planned adapters as disabled until their credentials/configuration are supplied.
