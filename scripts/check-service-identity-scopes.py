#!/usr/bin/env python3
"""Assert the Stage 1 service identities stay declared least-privilege.

docs/SERVICE-IDENTITIES.md carries a table of exactly which Integration Core
permissions each Stage 1 native-application service identity is supposed to
need, and says outright that `svc-outline` is read-only: it only ever lists
relationships, so it "deliberately does not receive `relationships.write`".
That table is the one place this deployment's intended grant is written down
— today Integration Core cannot yet enforce it per-service (every
`BSYSTEM-Services` member resolves to one coarse role, which the same
document also says plainly) — so nothing else would notice the table quietly
drifting to claim `svc-outline` needs write access, or losing the identity
altogether.

This also checks that every identity documented in the table is actually
provisioned in the blueprint, and vice versa: a permission promised for an
identity nobody creates is not a grant, and an identity with no documented
permissions is unreviewable.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
DOC = ROOT / "docs" / "SERVICE-IDENTITIES.md"
BLUEPRINT = ROOT / "authentik" / "blueprints" / "custom" / "40-bsystem-service-identities.yaml"

# Identities this check requires to be read-only per the platform's own
# documented intent: it may only ever list relationships, so a write-shaped
# permission on this row is the table quietly granting more than the caller
# it describes needs.
READ_ONLY_IDENTITIES = {"svc-outline"}

TABLE_ROW = re.compile(
    r"^\|\s*`(svc-[a-z0-9-]+)`\s*\|(?P<caller>[^|]*)\|(?P<permissions>[^|]*)\|\s*$",
    re.MULTILINE,
)
USERNAME = re.compile(r"^\s*username:\s*(svc-[a-z0-9-]+)\s*$", re.MULTILINE)


def documented_permissions() -> dict[str, str]:
    text = DOC.read_text(encoding="utf-8")
    rows = {}
    for match in TABLE_ROW.finditer(text):
        identity = match.group(1)
        permissions = match.group("permissions").strip()
        # Table rows in this file also include a header separator
        # (`|---|---|---|`) and rows for other tables; only a `svc-*`
        # identifier in the first column is one this check cares about.
        rows[identity] = permissions
    return rows


def blueprint_identities() -> set[str]:
    text = BLUEPRINT.read_text(encoding="utf-8")
    return set(USERNAME.findall(text))


def main() -> int:
    if not DOC.is_file():
        print(f"cannot find {DOC}", file=sys.stderr)
        return 1
    if not BLUEPRINT.is_file():
        print(f"cannot find {BLUEPRINT}", file=sys.stderr)
        return 1

    documented = documented_permissions()
    provisioned = blueprint_identities()

    if not documented:
        print(
            f"no `svc-*` permission row was found in {DOC}; this check would "
            f"pass vacuously",
            file=sys.stderr,
        )
        return 1
    if not provisioned:
        print(
            f"no `svc-*` username was found in {BLUEPRINT}; this check would "
            f"pass vacuously",
            file=sys.stderr,
        )
        return 1

    problems: list[str] = []

    for identity in sorted(documented.keys() - provisioned):
        problems.append(
            f"{identity} has a documented permission row in {DOC.relative_to(ROOT)} "
            f"but {BLUEPRINT.relative_to(ROOT)} does not provision that identity"
        )
    for identity in sorted(provisioned - documented.keys()):
        problems.append(
            f"{identity} is provisioned in {BLUEPRINT.relative_to(ROOT)} but has no "
            f"documented permission row in {DOC.relative_to(ROOT)}; its grant is "
            f"unreviewable"
        )

    for identity in READ_ONLY_IDENTITIES & documented.keys():
        permissions = documented[identity]
        if "relationships.write" in permissions:
            problems.append(
                f"{identity} is documented with permissions {permissions!r}, which "
                f"includes write access; {DOC.relative_to(ROOT)} states this identity "
                f"must be read-only"
            )
        if "relationships.read" not in permissions:
            problems.append(
                f"{identity} is documented with permissions {permissions!r}, which "
                f"does not even grant the read access its caller needs"
            )

    if problems:
        print("service identity permissions are not least-privilege:\n")
        for problem in problems:
            print(f"  {problem}")
        print(f"\n{len(problems)} finding(s)")
        return 1

    print(
        f"checked {len(documented)} service identity permission row(s): "
        f"{', '.join(sorted(READ_ONLY_IDENTITIES & documented.keys()))} "
        f"stay documented read-only, and every identity is provisioned in "
        f"exactly the identities documented"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
