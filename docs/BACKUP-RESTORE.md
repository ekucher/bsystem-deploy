# Backup and restore runbook (stage)

What to back up, how to verify it, and how to restore — for a stage
deployment. No external backup target is configured here; where the backup
goes is an owner decision, and this runbook deliberately stops at "write it
somewhere that is not this host".

## What must be backed up

| Data | Where | Why |
| --- | --- | --- |
| `bsystem_integration` database | `postgres_data` volume | Global IDs, mappings, RBAC, scope grants, audit, notifications, support records, AI audit |
| authentik's database | same PostgreSQL instance | users, groups, applications, providers, flows |
| authentik media | authentik volume | uploaded branding and certificates |
| `.env` | wherever you keep it | **not in the repository**; it holds secrets |

Global IDs are the reason the first row matters more than it looks. They are
immutable and everything else refers to them. Losing `global_entities` does not
lose rows; it breaks every reference held against those ids — in audit, in
scope grants, in support relations — and there is no way to recompute them,
because a Global ID is deliberately not derivable from any upstream value.

## What does not need backing up

| Not backed up | Why |
| --- | --- |
| Redis (`redis_data`) | authentik cache and task broker; rebuilt on start |
| NATS | no durable state in this deployment; events in flight are lost, which is already true whenever NATS is down |
| Container images | rebuilt from the repositories; the release manifest records which commits |
| EspoCRM, Redmine, Outline data | **source systems are authoritative and are backed up by whoever runs them.** BSYSTEM holds mappings, not copies |
| Search index | rebuilt by re-indexing; the in-memory provider has nothing to lose |

That last row is the architectural point: BSYSTEM is not a second CRM. Its
backup protects the mapping and governance layer, not the business data, and a
restore does not — and must not — try to put a source system back.

## Taking a backup

Stop nothing. `pg_dump` is online and consistent.

```bash
# Linux / macOS
docker compose exec -T postgres \
  pg_dump -U bsystem -d bsystem_integration --format=custom \
  > "backups/bsystem_integration-$(date -u +%Y%m%dT%H%M%SZ).dump"

docker compose exec -T postgres \
  pg_dump -U bsystem -d authentik --format=custom \
  > "backups/authentik-$(date -u +%Y%m%dT%H%M%SZ).dump"
```

```powershell
# Windows PowerShell
$stamp = (Get-Date).ToUniversalTime().ToString('yyyyMMddTHHmmssZ')
docker compose exec -T postgres pg_dump -U bsystem -d bsystem_integration --format=custom |
  Set-Content -Path "backups/bsystem_integration-$stamp.dump" -AsByteStream
docker compose exec -T postgres pg_dump -U bsystem -d authentik --format=custom |
  Set-Content -Path "backups/authentik-$stamp.dump" -AsByteStream
```

Then record what the environment was running, so a restore can be paired with
the code that produced it:

```bash
./scripts/release-manifest.sh > "backups/release-manifest-$(date -u +%Y%m%dT%H%M%SZ).json"
```

**Move the files off this host.** A backup on the same volume as the data
protects against exactly one failure mode — someone dropping a table — and none
of the others.

## Verifying a backup

An unverified backup is a belief. Verification means restoring it, not
inspecting it.

```bash
# Restore into a scratch database on the same instance.
docker compose exec -T postgres createdb -U bsystem restore_check
docker compose exec -T postgres pg_restore -U bsystem -d restore_check --no-owner < backups/<file>.dump

# The row counts that matter. Zero in any of these is a failed verification.
docker compose exec -T postgres psql -U bsystem -d restore_check -c \
  "SELECT 'global_entities' t, count(*) FROM global_entities
   UNION ALL SELECT 'identities', count(*) FROM identities
   UNION ALL SELECT 'roles', count(*) FROM roles
   UNION ALL SELECT 'principal_scopes', count(*) FROM principal_scopes
   UNION ALL SELECT 'schema_migrations', count(*) FROM schema_migrations;"

docker compose exec -T postgres dropdb -U bsystem restore_check
```

Acceptance for a verified backup:

- [ ] `pg_restore` completes with no error
- [ ] `global_entities` is non-empty, and its count matches the source
- [ ] `roles` is non-empty — a database with no roles denies everyone
- [ ] `schema_migrations` reports the expected level
- [ ] the scratch database is dropped afterwards

## Restoring

Ordering matters here more than the commands do.

1. **Stop the Integration Core first.**
   ```bash
   docker compose stop integration-core hub
   ```
   This is not optional. The Core applies migrations on every startup, so a Core
   left running against a restored database will migrate it straight back up —
   and if the reason for the restore was a bad migration, the restore silently
   undoes itself.

2. **Stop authentik** if its database is part of the restore.
   ```bash
   docker compose stop authentik-server authentik-worker
   ```

3. **Restore**, into a freshly created database rather than over a live one.
   ```bash
   docker compose exec -T postgres dropdb -U bsystem --if-exists bsystem_integration
   docker compose exec -T postgres createdb -U bsystem bsystem_integration
   docker compose exec -T postgres pg_restore -U bsystem -d bsystem_integration --no-owner < backups/<file>.dump
   ```

4. **Start in dependency order** — PostgreSQL and Redis, then authentik, then
   the Core, then the HUB.
   ```bash
   docker compose up -d postgres redis nats
   docker compose up -d authentik-server authentik-worker
   docker compose up -d integration-core
   docker compose up -d hub
   ```

5. **Verify** before telling anyone it worked:
   ```bash
   ./scripts/stage-smoke.sh
   DATABASE_URL=... go run ./cmd/mapping-audit   # in bsystem-integration-core
   ```

## Acceptance criteria after a restore

- [ ] `/readyz` reports `database: ok`
- [ ] `bsystem_schema_migrations_applied` in `/metrics` shows the expected level
- [ ] a user signs in through authentik and `GET /api/v1/me` returns the
      expected Global ID — **the same `USR-*` as before the restore**
- [ ] `scripts/stage-smoke.sh` reports no failure
- [ ] `cmd/mapping-audit` reports no error-severity finding
- [ ] an existing Global ID still resolves to the same upstream record
- [ ] scope grants still exist: a scoped user sees what they saw before

The third and last bullets are the ones that catch a restore that "worked". A
Global ID that changed, or a scope grant that vanished, is a successful restore
of a broken state.

## What this runbook does not do

- It configures no external backup target, no schedule and no retention
  policy. Where backups go, how long they are kept and who can read them are
  owner decisions with security consequences.
- It does not encrypt the dumps. A BSYSTEM dump contains identities, audit and
  RBAC — treat it as CONFIDENTIAL and encrypt it at rest.
- It does not cover restoring a source system. EspoCRM, Redmine and Outline are
  authoritative and are backed up by whoever operates them.
- It has not been executed against a real stage deployment. The commands are
  standard PostgreSQL operations and the ordering is derived from how this
  platform starts, but the first real run is part of the acceptance, not a
  formality after it.
