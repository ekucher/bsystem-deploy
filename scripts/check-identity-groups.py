#!/usr/bin/env python3
"""Check that the BSYSTEM group names agree wherever they are written down.

A group name decides access, and it is written four times in four places
that never meet:

  1. authentik/blueprints/bsystem-groups.yaml   what a real deployment creates
  2. mocks/cmd/mock-identity/principals.go      what the E2E stack exercises
  3. docs/AUTHENTIK-OIDC.md                     what the operator verifies
  4. the Integration Core's RBAC seed migration what actually grants a role

The platform maps a group name to a role and is deny-by-default, so a name
that appears in one list and not another does not raise an error anywhere. It
resolves to no roles, and the person in that group can do nothing.

The E2E stack cannot catch this, by construction: it replaces authentik with
the mock, so it compares (2) against (4) and never reads (1) at all. A typo in
the blueprint therefore passes every test and shows up on a stage acceptance as
a platform that refuses everyone — with the cause one character deep in a YAML
file nobody suspects, because the tests are green.

This script compares the three lists this repository owns, and the fourth as
well when a sibling checkout of bsystem-integration-core is present.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
BLUEPRINT = ROOT / "authentik" / "blueprints" / "bsystem-groups.yaml"
PRINCIPALS = ROOT / "mocks" / "cmd" / "mock-identity" / "principals.go"
SETUP_DOC = ROOT / "docs" / "AUTHENTIK-OIDC.md"
# Matches the layout documented in CLAUDE.md: the repositories sit side by side.
MIGRATIONS = ROOT.parent / "bsystem-integration-core" / "internal" / "platformdb" / "migrations"

GROUP = r"BSYSTEM-[A-Za-z0-9-]+"


def blueprint_groups(text: str) -> set[str]:
    """Group names the blueprint creates in authentik.

    Only `name:` under an identifiers block names a group; the blueprint's own
    metadata name is not one, which is why the pattern is anchored to the
    prefix rather than to the key.
    """
    return {m.group(1) for m in re.finditer(rf"name:\s*({GROUP})\b", text)}


def principal_groups(text: str) -> set[str]:
    """Group names the identity mock hands out.

    Read from the Groups literals rather than by running the mock, so the check
    costs nothing and works on a machine with no Go toolchain.
    """
    found: set[str] = set()
    for literal in re.finditer(r"Groups:\s*\[\]string\{([^}]*)\}", text):
        found.update(re.findall(rf'"({GROUP})"', literal.group(1)))
    return found


def documented_groups(text: str) -> set[str]:
    """Group names the setup document tells an operator to verify.

    The operator reads this list against authentik's Directory page, so a group
    the blueprint creates and the document omits is one nobody checks, and a
    group the document names and the blueprint does not is a hunt for something
    that was never going to be there. Only the bullet list is read: the prose
    and the role table below it mention the same names for other reasons.
    """
    return {m.group(1) for m in re.finditer(rf"^-\s+`({GROUP})`", text, re.M)}


def seeded_groups(text: str) -> set[str]:
    """Group names the platform maps to a role."""
    block = re.search(
        r"INSERT\s+INTO\s+group_role_mappings\s*\([^)]*\)\s*VALUES(.*?);",
        text,
        re.S | re.I,
    )
    if not block:
        return set()
    return set(re.findall(rf"'({GROUP})'", block.group(1)))


def read(path: Path) -> str | None:
    try:
        return path.read_text(encoding="utf-8")
    except OSError:
        return None


def compare(problems: list[str], left_name: str, left: set[str], right_name: str, right: set[str]) -> None:
    for name in sorted(left - right):
        problems.append(f"{name} is in {left_name} but not in {right_name}")
    for name in sorted(right - left):
        problems.append(f"{name} is in {right_name} but not in {left_name}")


def main() -> int:
    problems: list[str] = []
    notes: list[str] = []

    blueprint_text = read(BLUEPRINT)
    principals_text = read(PRINCIPALS)
    if blueprint_text is None:
        print(f"cannot read {BLUEPRINT}", file=sys.stderr)
        return 1
    if principals_text is None:
        print(f"cannot read {PRINCIPALS}", file=sys.stderr)
        return 1

    blueprint = blueprint_groups(blueprint_text)
    principals = principal_groups(principals_text)
    if not blueprint:
        print(f"no group names found in {BLUEPRINT}; the check would pass vacuously", file=sys.stderr)
        return 1
    if not principals:
        print(f"no group names found in {PRINCIPALS}; the check would pass vacuously", file=sys.stderr)
        return 1

    compare(problems, "the authentik blueprint", blueprint, "the identity mock", principals)

    # The operator verifies the groups by reading this list, so it is part of
    # the deployment rather than a description of it.
    setup_text = read(SETUP_DOC)
    if setup_text is None:
        print(f"cannot read {SETUP_DOC}", file=sys.stderr)
        return 1
    documented = documented_groups(setup_text)
    if not documented:
        print(f"no group names found in {SETUP_DOC}; the check would pass vacuously", file=sys.stderr)
        return 1
    compare(problems, "the authentik blueprint", blueprint, "the setup document", documented)

    # The authoritative mapping lives in the Integration Core. Checking it here
    # is best-effort: this repository is cloned on its own often enough that a
    # hard requirement would fail for the wrong reason.
    seed_text = None
    if MIGRATIONS.is_dir():
        for migration in sorted(MIGRATIONS.glob("*.sql")):
            text = read(migration)
            if text and seeded_groups(text):
                seed_text = text
                break
    if seed_text:
        seeded = seeded_groups(seed_text)
        compare(problems, "the authentik blueprint", blueprint, "the platform's RBAC seed", seeded)
    else:
        notes.append(
            "the Integration Core is not checked out beside this repository, "
            "so the platform's own RBAC seed was not compared"
        )

    if problems:
        print("group names disagree:", file=sys.stderr)
        for problem in problems:
            print(f"  {problem}", file=sys.stderr)
        print(
            "\nA group the platform does not map grants nothing, silently: the platform "
            "is deny-by-default, so the person in it can do nothing and no error is raised. "
            "The E2E stack replaces authentik with the mock and cannot see this.",
            file=sys.stderr,
        )
        return 1

    print(f"checked {len(blueprint)} group name(s): every list agrees")
    for note in notes:
        print(f"  note: {note}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
