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

## Stage 1 native application service identities

`authentik/blueprints/custom/40-bsystem-service-identities.yaml` provisions
three concrete service identities for the Stage 1 native applications, each a
non-interactive `service_account` in `BSYSTEM-Services`, with a static API
token minted from `SVC_REDMINE_TOKEN` / `SVC_QA_TOKEN` / `SVC_OUTLINE_TOKEN`
in `.env` (see `.env.example`). Each token is used only by that one
application's own backend/plugin as the bearer credential it presents to
Integration Core's `/api/service/v1/*` surface — never by a browser, never
by another application.

| authentik username | caller | Integration Core permissions actually required |
|---|---|---|
| `svc-redmine` | `redmine_bsystem_integration` plugin (`core_client.rb`): lists, creates and deletes relationship edges for the issue-page Related Objects panel | `relationships.read`, `relationships.write` |
| `svc-qa` | QA's cross-system links UI (docs/stage-1/09-QA-INTEGRATION.md workstream C: reuses the existing link/unlink flow against Integration Core instead of QA's own local tables) | `relationships.read`, `relationships.write` |
| `svc-outline` | Outline's Related Objects surface (docs/stage-1/10-OUTLINE-INTEGRATION.md section 7: a read-only "Related Work" panel; no add/remove relation is specified for Outline, unlike Redmine's D3 and QA's link/unlink flow) | `relationships.read` only |

None of the three receives the `*` wildcard service permission, and
`svc-outline` deliberately does not receive `relationships.write`: it only
ever lists relationships, so it should not be able to create or delete one.

**Current limitation, recorded rather than hidden:** Integration Core today
maps every member of `BSYSTEM-Services` to the single `service-core` role
(`internal/platformdb/migrations/003_rbac_scopes.sql`,
`020_relationship_permissions.sql` in `bsystem-integration-core`), which
already carries both `relationships.read` and `relationships.write`. Until
Integration Core adds per-service authentik groups (for example
`BSYSTEM-Service-Redmine`, `BSYSTEM-Service-QA`, `BSYSTEM-Service-Outline`)
mapped to per-service roles — an additive `group_role_mappings` /
`role_permissions` migration owned by that repository, not this one —
`svc-outline` is provisioned into the same coarse group as the other two and
is in practice as privileged as they are. The table above is this
deployment's declared target grant for that follow-up, not a claim that it is
already enforced.
