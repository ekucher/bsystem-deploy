#!/usr/bin/env python3
"""Refuse a compiled or archived artefact tracked in Git, and the paths that let one in.

bsystem-integration-core carried a 13 MB executable at its root because
`go build ./cmd/x` with no -o writes ./x, and .gitignore covered only the
directories a build can be *told* to use. This repository has the same shape
and the same scar: a `loadtest/loadtest` line added after it happened here once.

Adding one more path each time somebody notices is not a guard. This checks two
properties instead:

  1. nothing tracked in Git is a compiled executable or an archive, decided by
     leading bytes rather than by filename — a stray binary does not arrive
     with a helpful suffix, which is how the other one passed;

  2. every default build output this repository's Go modules can produce is
     ignored, so committing one is never a `git add -A` away.
"""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

# Leading bytes, and what they mean to a reader of the failure.
MAGIC = [
    (b"\x7fELF", "an ELF executable"),
    (b"MZ", "a Windows executable"),
    (b"\xfe\xed\xfa\xce", "a Mach-O executable"),
    (b"\xfe\xed\xfa\xcf", "a Mach-O executable"),
    (b"\xcf\xfa\xed\xfe", "a Mach-O executable"),
    (b"\xca\xfe\xba\xbe", "a Mach-O universal binary"),
    (b"PK\x03\x04", "a zip or jar archive"),
    (b"\x1f\x8b", "a gzip archive"),
    (b"\xfd7zXZ", "an xz archive"),
    (b"\x28\xb5\x2f\xfd", "a zstd archive"),
]

# Binary files that are legitimately tracked, each with its reason. Empty: this
# repository tracks none, and an addition should argue for itself in a diff
# somebody reads, which is the one thing a binary blob otherwise escapes.
ALLOWED: dict[str, str] = {}


def git(*args: str) -> str:
    return subprocess.run(
        ["git", "-C", str(ROOT), *args], check=True, capture_output=True, text=True
    ).stdout


def tracked() -> list[str]:
    return [name for name in git("ls-files", "-z").split("\0") if name]


def is_ignored(path: str) -> bool:
    """Ask git, rather than reimplementing .gitignore precedence."""
    return (
        subprocess.run(
            ["git", "-C", str(ROOT), "check-ignore", "-q", path], capture_output=True
        ).returncode
        == 0
    )


def has_main_package(directory: Path) -> bool:
    for source in directory.glob("*.go"):
        try:
            for line in source.read_text(encoding="utf-8", errors="replace").splitlines():
                if line.strip() == "package main":
                    return True
        except OSError:
            continue
    return False


def default_build_outputs() -> list[str]:
    """Where `go build` writes when nobody passes -o.

    A main package at a module's root builds to a file named after the
    directory; one under cmd/ builds to a file named after the command, in the
    directory the build was invoked from.
    """
    outputs: list[str] = []
    for gomod in sorted(ROOT.rglob("go.mod")):
        if ".git" in gomod.parts:
            continue
        module = gomod.parent
        relative = module.relative_to(ROOT)
        if has_main_package(module):
            outputs.append(str(relative / module.name))
        commands = module / "cmd"
        if commands.is_dir():
            for entry in sorted(commands.iterdir()):
                if entry.is_dir() and has_main_package(entry):
                    outputs.append(str(relative / entry.name))
    return outputs


def main() -> int:
    problems: list[str] = []

    names = tracked()
    if len(names) < 2:
        print(f"git reported {len(names)} tracked file(s); the check would pass vacuously", file=sys.stderr)
        return 1

    examined = 0
    for name in names:
        if name in ALLOWED:
            continue
        path = ROOT / name
        if not path.is_file() or path.is_symlink():
            continue
        try:
            head = path.open("rb").read(8)
        except OSError as err:
            problems.append(f"cannot read tracked file {name}: {err}")
            continue
        examined += 1
        for prefix, kind in MAGIC:
            if head.startswith(prefix):
                problems.append(
                    f"{name} is tracked in Git and is {kind}; build output belongs "
                    f"in .gitignore, and a binary that must be tracked belongs in "
                    f"ALLOWED with its reason"
                )
                break

    if examined == 0:
        print("no tracked file was examined; the check proves nothing", file=sys.stderr)
        return 1

    outputs = default_build_outputs()
    if not outputs:
        print("no Go command found; the build-output check would pass vacuously", file=sys.stderr)
        return 1
    for output in outputs:
        if not is_ignored(output):
            problems.append(
                f"`go build` writes {output}, which git does not ignore; "
                f"committing it is then one `git add -A` away"
            )

    if problems:
        print("repository artefact check failed:", file=sys.stderr)
        for problem in problems:
            print(f"  {problem}", file=sys.stderr)
        return 1

    print(f"checked {examined} tracked file(s) and {len(outputs)} default build output(s): clean")
    return 0


if __name__ == "__main__":
    sys.exit(main())
