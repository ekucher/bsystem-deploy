#!/usr/bin/env python3
"""Check that the Global ID prefixes agree between the rules and the platform.

Global IDs are immutable and everything refers to them: audit rows, scope
grants, support relations. Their prefixes are written twice, in two
repositories that never meet:

  1. CLAUDE.md                                        the rule
  2. the Integration Core's global_id_counters seed   what actually allocates

A prefix documented but never seeded allocates nothing: the allocation fails
with "unsupported entity type" for a type the rules say exists. A prefix seeded
but not documented is worse in the other direction — it hands out identifiers
under a letter combination nobody has agreed to, and Global IDs are immutable,
so a prefix that ships is a prefix forever.

Neither is caught by anything else. The E2E stack exercises the types it
happens to use, and a type nothing exercises has no test that would notice.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
RULES = ROOT / "CLAUDE.md"
# Matches the layout documented in CLAUDE.md: the repositories sit side by side.
MIGRATIONS = ROOT.parent / "bsystem-integration-core" / "internal" / "platformdb" / "migrations"

# `CL-*   Client` in the rules' Global IDs block.
DOCUMENTED = re.compile(r"^([A-Z]{2,4})-\*\s+\S", re.MULTILINE)
# `('client', 'CL', 1)` in a counters seed, in any migration that writes one.
SEEDED = re.compile(r"\(\s*'([a-z_]+)'\s*,\s*'([A-Z]{2,4})'\s*,\s*\d+\s*\)")


def seeded_prefixes() -> dict[str, str] | None:
    """Every prefix any migration seeds, or None when Core is not checked out.

    Read across all migrations rather than from the initial one: `SVC` is
    seeded by 002, and a check that read only 001 would report the service
    prefix as undocumented-in-reverse — which is how a correct pair gets
    "fixed" into a wrong one.
    """
    if not MIGRATIONS.is_dir():
        return None
    found: dict[str, str] = {}
    for migration in sorted(MIGRATIONS.glob("*.sql")):
        text = migration.read_text(encoding="utf-8")
        if "global_id_counters" not in text:
            continue
        for entity, prefix in SEEDED.findall(text):
            found[prefix] = entity
    return found or None


def main() -> int:
    documented = set(DOCUMENTED.findall(RULES.read_text(encoding="utf-8")))
    if len(documented) < 2:
        print(
            f"found {len(documented)} documented prefix(es) in CLAUDE.md; "
            f"the check proves nothing",
            file=sys.stderr,
        )
        return 1

    seeded = seeded_prefixes()
    if seeded is None:
        print(
            "the Integration Core is not checked out beside this repository, so "
            "the documented prefixes were not compared against the platform's seed"
        )
        return 0

    problems: list[str] = []
    for prefix in sorted(documented - set(seeded)):
        problems.append(
            f"CLAUDE.md documents the {prefix}- prefix, but no migration seeds it; "
            f"allocating that type fails with unsupported_entity_type"
        )
    for prefix in sorted(set(seeded) - documented):
        problems.append(
            f"a migration seeds the {prefix}- prefix for {seeded[prefix]!r}, but "
            f"CLAUDE.md does not document it; a Global ID that ships is permanent"
        )

    if problems:
        print("the Global ID prefixes disagree:\n")
        for problem in problems:
            print(f"  {problem}")
        return 1

    print(f"checked {len(documented)} Global ID prefix(es): the rules and the seed agree")
    return 0


if __name__ == "__main__":
    sys.exit(main())
