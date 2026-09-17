# Human user administration

BSYSTEM human accounts are managed in authentik. Platform authorization is
derived from one BSYSTEM human group.

Operational CLI:

```bash
./scripts/bsystem-userctl
```

The normal administration surface is the **Користувачі** page in BSYSTEM-HUB.
The CLI remains the local operator / break-glass path.

The HUB never receives an authentik API credential. Browser requests go to
Integration Core, which enforces BSYSTEM RBAC and uses a dedicated
least-privilege authentik service account server-side.

## HUB permissions

- `identity.user.read` — read the human account directory. Granted to
  Administrator, Manager and Support.
- `identity.user.manage` — create users, change non-administrator human roles,
  enable/disable accounts and reset local passwords. Granted to Administrator
  and Manager.
- `identity.user.admin` — administrator-class targets and assignments. It is
  not granted to Manager; Administrator receives it through the platform `*`
  wildcard.

`BSYSTEM-Services` identities are excluded from this surface even if they are
misconfigured with a human group.

## Human roles

| CLI role | authentik group |
| --- | --- |
| `admin` | `BSYSTEM-Admins` |
| `manager` | `BSYSTEM-Managers` |
| `developer` | `BSYSTEM-Developers` |
| `qa` | `BSYSTEM-QA` |
| `support` | `BSYSTEM-Support` |
| `devops` | `BSYSTEM-DevOps` |
| `customer` | `BSYSTEM-Customers` |

`BSYSTEM-Services` is not a human role. Machine identities are documented in
[SERVICE-IDENTITIES.md](SERVICE-IDENTITIES.md).

## Commands

```bash
./scripts/bsystem-userctl list
./scripts/bsystem-userctl show USERNAME
./scripts/bsystem-userctl create
./scripts/bsystem-userctl set-role USERNAME support
./scripts/bsystem-userctl passwd USERNAME
./scripts/bsystem-userctl disable USERNAME
./scripts/bsystem-userctl enable USERNAME
```

There is intentionally no `delete` command yet. User deletion requires a
separate lifecycle decision for the persistent Integration Core identity,
Global User ID, and audit/history references.

The HUB follows the same rule: there is no destructive delete action. A user
can be disabled while the persistent `USR-*` identity and audit/history remain
intact.

## Enabling HUB mutations

Human-account mutation is disabled when `BSYSTEM_AUTHENTIK_ADMIN_TOKEN` is
empty. The directory remains readable from Integration Core's persistent
identity store.

Generate the token on the deployment host and keep it only in the local
`.env`:

```bash
openssl rand -hex 32
```

Store that output as `BSYSTEM_AUTHENTIK_ADMIN_TOKEN`. Do not commit it.

The authentik blueprint provisions a dedicated service account and RBAC role
with only:

- `authentik_core.view_user`
- `authentik_core.add_user`
- `authentik_core.change_user`
- `authentik_core.reset_user_password`
- `authentik_core.view_group`

It is not an authentik superuser and does not use the bootstrap/akadmin token.

## Password handling

Passwords are entered through a hidden interactive prompt. They are never
accepted as command-line arguments and are passed to authentik over stdin.

The CLI rejects empty passwords and verifies confirmation before mutation. The
live integration test also verifies that password values do not appear in its
output.

## Role behavior

`set-role` replaces the previous BSYSTEM human role instead of accumulating
multiple human roles.

A human role cannot be assigned to an identity in `BSYSTEM-Services`.

Role changes are immediate in authentik. Integration Core observes the updated
group mapping on the next authenticated request.

## Administrator safety

The CLI prevents removal of the final active BSYSTEM administrator.

If only one active member of `BSYSTEM-Admins` remains, both disabling that user
and moving that user to another role are refused.

The guard runs transactionally. `set-role` and `disable` acquire locks in the
same order: the `BSYSTEM-Admins` group first, then user rows. This avoids a
lock-order inversion between concurrent administrative operations.

The live test found and corrected a PostgreSQL-specific defect in the original
guard path: `SELECT ... FOR UPDATE` cannot be combined with the `DISTINCT`
query produced by the original many-to-many lookup. The final implementation
locks the administrator group first and queries users through that exact group
relation without `DISTINCT`.

## Global User IDs

Creating a user in authentik does not allocate a Global User ID.

Integration Core allocates the Global User ID when that identity first
successfully authenticates through the platform.

Global IDs are persistent. Gaps are normal and IDs must never be manually
decremented or reused.

## Live integration test

Run against the local BSYSTEM stack:

```bash
./scripts/tests/bsystem-userctl.live.test.sh
```

The test creates a uniquely named temporary user, exercises the supported
mutations, and removes it through an exit trap.

The owner-run local test completed with:

```text
RESULT: passed=17 failed=0 skipped=0
Remaining userctl-test users: 0
```

It covers creation, duplicate rejection, role replacement, disable/enable,
password reset, direct authentik password verification, invalid-role rejection,
password secrecy, and both last-administrator protection paths.

This is a live integration test, not a credential-free CI fixture: it requires
the real local authentik and PostgreSQL containers.
