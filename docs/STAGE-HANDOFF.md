# Stage handoff

Everything needed to run the first controlled BSYSTEM acceptance, and an honest
account of what is still missing.

Read [`STAGE-ACCEPTANCE.md`](STAGE-ACCEPTANCE.md) first for the environment
contract. This document is the procedure.

## What is complete and autonomous

These work now, with no owner input, and are exercised by CI on every push:

| Capability | Where |
| --- | --- |
| Environment contract: services, variables, classification, network paths, health URLs | `docs/STAGE-ACCEPTANCE.md` |
| Preflight validation, Linux and Windows | `scripts/stage-preflight.{sh,ps1}` |
| Non-destructive smoke acceptance, Linux and Windows | `scripts/stage-smoke.{sh,ps1}` |
| Machine- and human-readable acceptance report | `artifacts/stage-acceptance.{json,md}` |
| Release manifest across all four repositories | `scripts/release-manifest.sh` |
| Stage Compose overlay | `docker-compose.stage.yml` |
| Adapter contracts and version-sensitive assumptions | `bsystem-integration-core/docs/adapters/*-STAGE.md` |
| Read-only mapping health diagnostic | `bsystem-integration-core/cmd/mapping-audit` |
| Migration audit; migrations proven against an empty database in CI | `bsystem-integration-core/docs/MIGRATION-READINESS.md` |
| authentik configuration checklist | `docs/AUTHENTIK-STAGE.md` |
| Tenant isolation acceptance matrix | `docs/TENANT-ISOLATION-MATRIX.md` |
| Backup and restore runbook | `docs/BACKUP-RESTORE.md` |
| Acceptance examples for every endpoint an acceptance walks | `bsystem-integration-core/docs/openapi.yaml` |

## What the owner must supply

Nothing below can be derived, inferred or guessed from the repositories.

### 1. Secrets — generate, never commit

| Variable | How |
| --- | --- |
| `POSTGRES_PASSWORD` | 32+ random characters |
| `AUTHENTIK_SECRET_KEY` | 50+ random characters |

```bash
openssl rand -base64 48 | tr -d '\n='   # Linux / macOS
```

```powershell
[Convert]::ToBase64String([System.Security.Cryptography.RandomNumberGenerator]::GetBytes(36))
```

### 2. DNS and URLs

`hub.`, `api.` and `id.` for the browser; `crm.`, `pm.` and `wiki.` for egress
to the source systems. All six are placeholders in the repository today.

### 3. authentik

The application, the OAuth2 provider, the generated **client id** (there is no
client secret — the HUB is a public PKCE client), the eight groups, and a
property mapping that actually puts `groups` in the UserInfo response. Follow
`docs/AUTHENTIK-STAGE.md`.

### 4. Source systems

Read-only API credentials for EspoCRM, Redmine and Outline — each scoped to
read the entities BSYSTEM consumes and nothing more. **Enable Redmine's REST
API**; it is off by default and produces a 403 on every call while the instance
looks healthy in a browser.

### 5. Decisions, not values

| Decision | Blocks |
| --- | --- |
| Authoritative customer ownership mapping | every customer ALLOW case in the isolation matrix |
| SLA response and resolution targets per severity | support SLA acceptance |
| Whether any role may use the AI gateway | `ai.query` is granted to no role; only Administrator reaches it |
| npm registry and token for the design system | publishing `@bsystem/*` |

## Execution order

1. Generate secrets and write `.env` — never commit it.
2. Configure authentik; record the client id and issuer.
3. Run **preflight**. Do not continue while it reports a failure.
4. Take a backup and **verify it by restoring** into a scratch database.
5. Start the stack.
6. Run **smoke**. Read the report.
7. Run the **mapping audit**.
8. Walk the **isolation matrix** by hand, one actor at a time.
9. Generate a **release manifest** and store it with the backup.

Steps 3 and 4 come before step 5 deliberately. A stack started against an
unverified backup and a half-filled `.env` produces failures whose cause is two
steps behind where you are looking.

## Commands — Linux shell

```bash
cd bsystem-deploy

# 3. preflight
./scripts/stage-preflight.sh

# 4. backup, then verify it (see docs/BACKUP-RESTORE.md for the full procedure)
mkdir -p backups
docker compose exec -T postgres pg_dump -U bsystem -d bsystem_integration --format=custom \
  > "backups/bsystem_integration-$(date -u +%Y%m%dT%H%M%SZ).dump"

# 5. start
docker compose -f docker-compose.yml -f docker-compose.stage.yml up -d
docker compose -f docker-compose.yml -f docker-compose.stage.yml ps

# 6. smoke
CORE_URL=https://api.stage.example \
HUB_URL=https://hub.stage.example \
AUTHENTIK_URL=https://id.stage.example/application/o/bsystem-hub \
./scripts/stage-smoke.sh

# with tokens, once you have them; they are read from the environment and
# never printed, logged or written to the report
BSYSTEM_HUMAN_TOKEN='...' BSYSTEM_SERVICE_TOKEN='...' \
CORE_URL=https://api.stage.example HUB_URL=https://hub.stage.example \
./scripts/stage-smoke.sh

# 7. mapping audit
cd ../bsystem-integration-core
DATABASE_URL='postgres://bsystem:...@127.0.0.1:5432/bsystem_integration?sslmode=disable' \
  go run ./cmd/mapping-audit -format=json

# 9. release manifest
cd ../bsystem-deploy
./scripts/release-manifest.sh > "backups/release-manifest-$(date -u +%Y%m%dT%H%M%SZ).json"
```

## Commands — Windows PowerShell

```powershell
Set-Location bsystem-deploy

# 3. preflight
.\scripts\stage-preflight.ps1

# 4. backup
New-Item -ItemType Directory -Force -Path backups | Out-Null
$stamp = (Get-Date).ToUniversalTime().ToString('yyyyMMddTHHmmssZ')
docker compose exec -T postgres pg_dump -U bsystem -d bsystem_integration --format=custom |
  Set-Content -Path "backups/bsystem_integration-$stamp.dump" -AsByteStream

# 5. start
docker compose -f docker-compose.yml -f docker-compose.stage.yml up -d
docker compose -f docker-compose.yml -f docker-compose.stage.yml ps

# 6. smoke
$env:CORE_URL = 'https://api.stage.example'
$env:HUB_URL = 'https://hub.stage.example'
$env:AUTHENTIK_URL = 'https://id.stage.example/application/o/bsystem-hub'
.\scripts\stage-smoke.ps1

# with tokens
$env:BSYSTEM_HUMAN_TOKEN = '...'
$env:BSYSTEM_SERVICE_TOKEN = '...'
.\scripts\stage-smoke.ps1

# 7. mapping audit
Set-Location ..\bsystem-integration-core
$env:DATABASE_URL = 'postgres://bsystem:...@127.0.0.1:5432/bsystem_integration?sslmode=disable'
go run ./cmd/mapping-audit -format=json

# 9. release manifest
Set-Location ..\bsystem-deploy
bash ./scripts/release-manifest.sh > "backups/release-manifest-$stamp.json"
```

`release-manifest.sh` has no PowerShell twin. It is a git-and-hash reader with
no Windows-specific behaviour, and Git for Windows ships the bash that runs it;
a second implementation would be a second thing to keep correct.

## Expected outcomes

| Step | Success looks like |
| --- | --- |
| Preflight | `preflight passed`, exit 0 |
| Start | every service `healthy`; `integration-core` may take up to 60s while it migrates |
| Smoke, no tokens | operational and authorization-boundary checks PASS; authenticated checks SKIP; exit 0 |
| Smoke, with tokens | human and machine API checks PASS; exit 0 |
| Mapping audit | `0 error(s)`; warnings for records with no owner are expected before ownership is mapped |
| Isolation matrix | every DENY cell refuses; every detail read a caller may not see returns `404`, not `403` |

A SKIP is not a pass. A run that is entirely SKIPs and PASSes with no token
configured has verified that the platform is up and that it rejects anonymous
callers — nothing about authorization.

## Rollback

```bash
# code only: the previous image, with the schema left alone
docker compose -f docker-compose.yml -f docker-compose.stage.yml stop integration-core hub
# check out the previous commit, then
docker compose -f docker-compose.yml -f docker-compose.stage.yml up -d --build integration-core hub
```

Every migration is additive, so the extra tables are inert to older code and no
down-migration is needed.

For a bad migration, restore the database — and **stop the Core first**. It
applies migrations on every startup, so a Core left running will migrate the
restored database straight back up and silently undo the restore. Full
procedure in [`BACKUP-RESTORE.md`](BACKUP-RESTORE.md).

## Acceptance checklist

- [ ] preflight passes with no failure
- [ ] a backup exists and has been restored into a scratch database
- [ ] every service reports healthy
- [ ] `/readyz` reports `database: ok`
- [ ] `bsystem_schema_migrations_applied` reports the expected level
- [ ] smoke reports no FAIL
- [ ] anonymous callers are rejected on both `/api/v1/*` and `/api/service/v1/*`
- [ ] a human token is rejected on the machine API
- [ ] a service token is rejected on the human API
- [ ] a user in no BSYSTEM group signs in and sees nothing, without an error
- [ ] each role reaches exactly the resources in the isolation matrix
- [ ] a detail read the caller may not see returns `404`, not `403`
- [ ] the mapping audit reports no error-severity finding
- [ ] adapter health is green for every configured source system
- [ ] MFA cannot be bypassed
- [ ] the release manifest is stored with the backup

## Known limitations

1. **No adapter has been tested against a live upstream.** Every contract is
   read off the code. The version-sensitive assumptions in each adapter
   document are the specific things to check first.
2. **Migration durations are classified, not measured.** No timing in this
   repository comes from a real database.
3. **`CREATE INDEX` is not concurrent.** Irrelevant on an empty database; on a
   populated `audit_events` it would block writes. Recorded rather than
   quietly changed.
4. **NATS absence is partly silent.** `/readyz` reports `nats: degraded` and
   the platform serves. The three identity and Global ID allocation events are
   queued in the platform's own database and delivered when the broker
   returns; every other event produced during the outage is dropped. See
   `bsystem-integration-core/docs/EVENTS.md` for which is which and why.
5. **The E2E stack does not include the HUB.** It asserts the normalized API
   contract the HUB consumes.
6. **No ESLint in the HUB.** `typescript-eslint` does not support TypeScript 7.
7. **The backup runbook has never been executed against a real deployment.**
   Its first real run is part of the acceptance, not a formality after it.

## Remaining BLOCKED items

| # | Item | What is required |
| --- | --- | --- |
| 1 | Customer ownership mapping | which client each customer account owns |
| 2 | SLA targets | respond/resolve minutes per severity |
| 3 | Design system publishing | registry choice and a `write:packages` token |
| 4 | BRAVO inbound adapter | BRAVO's API documentation |
| 5 | ESLint for the HUB | blocked upstream on TypeScript 7 support |
| 6 | Live Docker host acceptance | a real host |
| 7 | Real authentik acceptance | the configured instance |
| 8 | Real EspoCRM acceptance | URL and credential |
| 9 | Real Redmine acceptance | URL, credential, REST enabled |
| 10 | Real Outline acceptance | URL and scoped key |
| 11 | Production tenant assignment | authoritative ownership |
| 12 | Production DNS/TLS/secrets | the environment |
| 13 | Backup/restore acceptance | a real host and a real backup target |
| 14 | Production deployment | owner action; never autonomous |

Items 1 and 2 are decisions rather than access, and they are the two that most
constrain what the acceptance can conclude. Until ownership is authoritative,
customer isolation can be accepted for *deny* and not for *allow*: it can be
shown that a customer sees nothing they should not, but not yet that they see
everything they should.
