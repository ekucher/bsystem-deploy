#!/usr/bin/env python3
"""Fail when a Go module in this repository is outside the security scanners.

The Go vulnerability job takes its modules from a matrix written by hand. It
listed `mocks` and `e2e`; `loadtest` was added later, compiles in CI, ships in
this repository, and was scanned by nothing. Nobody removed it — it simply was
never added, and there was no way to notice: the job stayed green, because a
module that is not in the matrix produces no failure and no output.

That is the shape worth guarding. A module outside the scanners and a module
with no findings look identical from the outside, so the check compares the
modules that exist against the modules each matrix names, rather than trusting
that somebody remembered.
"""

from __future__ import annotations

import sys
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parent.parent
WORKFLOW = ROOT / ".github" / "workflows" / "security.yml"

# Each job that must name every Go module, and what it would miss.
SCANNED_BY = {
    "vulnerabilities": "govulncheck",
    "staticcheck": "staticcheck",
}


def go_modules() -> set[str]:
    """Every Go module, named the way the matrix names it: by directory."""
    modules = set()
    for gomod in ROOT.rglob("go.mod"):
        if ".git" in gomod.parts:
            continue
        modules.add(str(gomod.parent.relative_to(ROOT)))
    return modules


def matrix_modules(jobs: dict, job: str) -> set[str] | None:
    """The modules a job's matrix names, or None when there is no such matrix."""
    definition = jobs.get(job)
    if not isinstance(definition, dict):
        return None
    matrix = (definition.get("strategy") or {}).get("matrix") or {}
    listed = matrix.get("module")
    if not isinstance(listed, list):
        return None
    return {str(entry) for entry in listed}


def main(arguments: list[str]) -> int:
    """With no argument, check this repository's own security workflow.

    A path may be passed instead, so that a test can check a copy rather than
    editing the real workflow in place: a test that mutates a tracked file and
    restores it afterwards leaves the working tree wrong if it is interrupted.
    """
    workflow = Path(arguments[0]) if arguments else WORKFLOW
    modules = go_modules()

    # A repository with one module, or none, would let every comparison below
    # succeed without comparing anything.
    if len(modules) < 2:
        print(
            f"found {len(modules)} Go module(s); the coverage check proves nothing",
            file=sys.stderr,
        )
        return 1

    with workflow.open(encoding="utf-8") as handle:
        jobs = (yaml.safe_load(handle) or {}).get("jobs") or {}

    problems: list[str] = []
    for job, scanner in sorted(SCANNED_BY.items()):
        listed = matrix_modules(jobs, job)
        if listed is None:
            problems.append(
                f"security.yml has no job {job!r} with a module matrix, so nothing "
                f"says which modules {scanner} covers"
            )
            continue
        for missing in sorted(modules - listed):
            problems.append(
                f"{missing}/go.mod exists but {job!r} does not list it, so "
                f"{scanner} never sees it and the job stays green"
            )
        for stale in sorted(listed - modules):
            problems.append(
                f"{job!r} lists {stale!r}, which has no go.mod; a matrix entry "
                f"naming nothing scans nothing"
            )

    if problems:
        print("Go modules are not covered by the security scanners:\n")
        for problem in problems:
            print(f"  {problem}")
        return 1

    print(
        "checked %d Go module(s) against %d scanner matrix(es): every module is covered"
        % (len(modules), len(SCANNED_BY))
    )
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
