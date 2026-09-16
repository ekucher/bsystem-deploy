# Deployment documentation

| Document | What it answers |
| --- | --- |
| [DEPLOYMENT.md](DEPLOYMENT.md) | the decisions a real deployment makes |
| [E2E-ENVIRONMENT.md](E2E-ENVIRONMENT.md) | the credential-free stack CI runs |
| [AUTHENTIK-OIDC.md](AUTHENTIK-OIDC.md) | identity provider setup |
| [SERVICE-IDENTITIES.md](SERVICE-IDENTITIES.md) | machine callers |
| [ARCHITECTURE.md](ARCHITECTURE.md) | how the pieces fit |
| [../SECURITY.md](../SECURITY.md) | what is enforced, and the current hardening state |

## Canonical files

`CLAUDE.md` and `TASKS.md` in this repository are the canonical copies for the
whole platform. A copy anywhere else is a copy.

## Historical

`P0-FOUNDATION.md` and `RUN-P0.md` record the foundation work as it was done.
Where they disagree with [DEPLOYMENT.md](DEPLOYMENT.md), that document is
right.

## Stage readiness (P17)

- [`STAGE-ACCEPTANCE.md`](STAGE-ACCEPTANCE.md) — the environment contract:
  services, every variable classified, network paths, health URLs, rollback and
  backup prerequisites.
- [`STAGE-HANDOFF.md`](STAGE-HANDOFF.md) — the procedure: what is autonomous,
  what the owner supplies, the exact command sequence for shell and PowerShell,
  and the remaining blocked items.
- [`AUTHENTIK-STAGE.md`](AUTHENTIK-STAGE.md) — the exact identity configuration,
  with MFA, customer and service identity checklists.
- [`TENANT-ISOLATION-MATRIX.md`](TENANT-ISOLATION-MATRIX.md) — nine actors
  against ten resources, derived from the seeded grants rather than assumed.
- [`BACKUP-RESTORE.md`](BACKUP-RESTORE.md) — what to back up, how to verify it
  by restoring, and the stop/start ordering a restore depends on.

Scripts:

```bash
./scripts/stage-preflight.sh     # before starting anything
./scripts/stage-smoke.sh         # non-destructive acceptance pass
./scripts/release-manifest.sh    # what this checkout would deploy
```

`.ps1` equivalents of the first two exist for Windows. None of them prints a
secret: values are reported by length, and `scripts/tests/stage-scripts.test.sh`
fails if that ever stops being true.
