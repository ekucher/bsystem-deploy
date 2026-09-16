# BSYSTEM mock upstreams

Deterministic stand-ins for the systems BSYSTEM integrates with, so that the
platform can be validated end to end without owner credentials or production
access.

| Mock | Stands in for | Default port | Consumed by |
| --- | --- | --- | --- |
| `mock-identity` | authentik OIDC userinfo | `9000` | Integration Core `AUTHENTIK_USERINFO_URL` |
| `mock-espocrm` | EspoCRM REST API | `8090` | `internal/adapters/espocrm` |
| `mock-redmine` | Redmine REST API | `8091` | `internal/adapters/redmine` |
| `mock-outline` | Outline RPC API | `8092` | `internal/adapters/outline` |

## Security classification

Everything in this module is `PUBLIC`.

- Every credential accepted by a mock is a **test-only placeholder** with a
  `test-` prefix. None of them can reach a real system.
- Fixtures are invented. Production payloads must never be copied here.
- Request logging records method, path and status only. Headers and query
  strings are never logged, because they carry the test credentials.

## Test credentials

| Variable | Default | Used by |
| --- | --- | --- |
| `ESPOCRM_API_KEY` | `test-espocrm-api-key` | `X-Api-Key` header |
| `REDMINE_API_KEY` | `test-redmine-api-key` | `X-Redmine-API-Key` header |
| `OUTLINE_API_KEY` | `test-outline-api-key` | `Authorization: Bearer` |

## Identity fixtures

`mock-identity` serves `GET /application/o/userinfo/`. One principal exists per
BSYSTEM group, plus the negative cases the authorization tests need.

| Token | Subject | Groups |
| --- | --- | --- |
| `test-token-admin` | `mock-admin` | `BSYSTEM-Admins` |
| `test-token-manager` | `mock-manager` | `BSYSTEM-Managers` |
| `test-token-developer` | `mock-developer` | `BSYSTEM-Developers` |
| `test-token-qa` | `mock-qa` | `BSYSTEM-QA` |
| `test-token-support` | `mock-support` | `BSYSTEM-Support` |
| `test-token-devops` | `mock-devops` | `BSYSTEM-DevOps` |
| `test-token-customer` | `mock-customer` | `BSYSTEM-Customers` |
| `test-token-service` | `mock-service-core` | `BSYSTEM-Services` |
| `test-token-service-ungrouped` | `mock-service-ungrouped` | none |
| `test-token-no-groups` | `mock-no-groups` | none |
| `test-token-expired` | `mock-expired` | `BSYSTEM-Developers` (always rejected) |

Any unknown token is rejected with `401`.

Set `IDENTITY_FIXTURES` to a JSON file to replace the whole set, or `POST` a
principal to `/__mock/principals` to reconfigure groups without a restart.
`GET /__mock/principals` lists the configured identities without their tokens.

## Fault injection

Each mock exposes a test-only control plane so that upstream failures are
reproducible without editing fixtures.

```bash
# Fail the next EspoCRM account listing with 429
curl -X POST http://localhost:8090/__mock/faults \
  -d '{"path":"/api/v1/Account","status":429}'

# Make the next Redmine project listing hang past the adapter timeout
curl -X POST http://localhost:8091/__mock/faults \
  -d '{"path":"/projects.json","delay_ms":12000}'

# Inspect and clear
curl http://localhost:8090/__mock/faults
curl -X DELETE http://localhost:8090/__mock/faults
```

| Field | Meaning |
| --- | --- |
| `path` | Request path the fault applies to; `*` or omitted matches every path |
| `status` | HTTP status returned instead of the normal response |
| `body` | Optional raw response body |
| `delay_ms` | Delay before responding; a delay without a status still returns the real response, which is what exercises adapter timeouts |
| `remaining` | How many requests the fault applies to; defaults to `1` |

`/health` and `/__mock/*` are never faulted, so the control plane stays
reachable while a fault is active.

## Running

```bash
go test ./...
go run ./cmd/mock-espocrm

docker build --build-arg MOCK=mock-espocrm -t bsystem/mock-espocrm .
```

The E2E stack in `docker-compose.e2e.yml` builds all four.
