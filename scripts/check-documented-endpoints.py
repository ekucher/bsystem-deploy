#!/usr/bin/env python3
"""Check that every platform endpoint this repository documents exists.

An endpoint named in a runbook is followed by somebody at a keyboard. If the
platform does not serve it they get a 404, and a 404 from a documented path
reads as a broken deployment rather than a stale document — so the next twenty
minutes go into the deployment.

The authority is the Integration Core's OpenAPI specification, which its own CI
keeps in step with the routing table. This compares the paths named in this
repository's Markdown against that specification. Integration Core's own
documentation is checked by Integration Core's CI; this repository is not the
place to make that repository's docs fail.

Two shapes are deliberately not endpoints and would otherwise dominate the
findings:

  * a version prefix written as a rule, `/api/v1/*`, which is policy rather
    than a path;
  * an upstream's own path, `/api/v1/Account`, which belongs to EspoCRM and is
    named when describing what an adapter requests or what a mock serves.

Only the second needs a list, because only it is indistinguishable from a
platform path by shape.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parent.parent
SPEC = ROOT.parent / "bsystem-integration-core" / "docs" / "openapi.yaml"

ENDPOINT = re.compile(r"/api/(?:service/)?v1/[A-Za-z0-9{}/_-]*\*?")
SKIP_DIRS = {".git", "node_modules", "artifacts", "backups"}

# Paths that look like the platform's and are not, each with its owner. An
# upstream path is named when describing what an adapter requests or what a
# mock serves, and no specification of ours describes it.
UPSTREAM_PATHS = {
    "/api/v1/Account": "EspoCRM's own accounts endpoint, served by mock-espocrm",
    "/api/v1/Contact": "EspoCRM's own contacts endpoint, served by mock-espocrm",
    "/api/v1/App/user": "EspoCRM's own session endpoint, served by mock-espocrm",
}


def documented_paths() -> set[str]:
    """Paths the specification serves, with placeholders normalised."""
    spec = yaml.safe_load(SPEC.read_text(encoding="utf-8")) or {}
    return {re.sub(r"\{[^}]+\}", "{}", path) for path in (spec.get("paths") or {})}


def main() -> int:
    if not SPEC.is_file():
        print(
            "the Integration Core is not checked out beside this repository, so "
            "the documented endpoints were not compared against its OpenAPI spec"
        )
        return 0

    served = documented_paths()
    if len(served) < 2:
        print(
            f"the specification lists {len(served)} path(s); the check proves nothing",
            file=sys.stderr,
        )
        return 1

    problems: list[str] = []
    checked = 0
    for document in sorted(ROOT.rglob("*.md")):
        if any(part in SKIP_DIRS for part in document.parts):
            continue
        if document.name == "TASKS.md":
            continue
        for number, line in enumerate(document.read_text(encoding="utf-8").splitlines(), 1):
            for raw in ENDPOINT.findall(line):
                endpoint = raw.rstrip(".,;:)`\"'")
                # A trailing star is the versioning rule, not a path.
                if endpoint.endswith("*"):
                    continue
                endpoint = endpoint.rstrip("/")
                if endpoint in UPSTREAM_PATHS:
                    continue
                checked += 1
                # A concrete identifier in a runbook stands for a placeholder.
                candidates = {
                    re.sub(r"\{[^}]+\}", "{}", endpoint),
                    re.sub(r"/[^/]+$", "/{}", endpoint),
                }
                if candidates & served:
                    continue
                problems.append(
                    f"{document.relative_to(ROOT)}:{number}: documents {endpoint}, "
                    f"which the Integration Core's OpenAPI specification does not serve"
                )

    if checked == 0:
        print("no endpoint was named in any document; the check proves nothing", file=sys.stderr)
        return 1

    if problems:
        print("documented endpoints the platform does not serve:\n")
        for problem in problems:
            print(f"  {problem}")
        return 1

    print(f"checked {checked} documented endpoint reference(s) against {len(served)} specified path(s)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
