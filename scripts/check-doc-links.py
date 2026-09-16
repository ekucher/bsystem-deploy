#!/usr/bin/env python3
"""Fail when a document points at a path that does not exist.

Acceptance documentation is followed by someone under time pressure, often on
a different machine, frequently at night. A link to a file that was renamed or
a command naming a script that moved costs that person twenty minutes and some
of their confidence in everything else the document says.

Checks two things across the repository's Markdown:

  * relative Markdown links resolve to a file that exists;
  * paths mentioned in backticks that look like repository paths exist, either
    here or in a sibling repository.
"""

from __future__ import annotations

import pathlib
import re
import sys

MARKDOWN_LINK = re.compile(r"\[[^\]]*\]\(([^)\s]+)(?:\s+\"[^\"]*\")?\)")
BACKTICK_PATH = re.compile(r"`([^`\n]+)`")

# A backtick span is a path worth checking only if it looks like one: it has a
# directory separator or a known extension, and no spaces or shell syntax.
PATH_LIKE = re.compile(r"^[\w./-]+$")
CHECKED_SUFFIXES = (".md", ".yml", ".yaml", ".sh", ".ps1", ".py", ".go", ".sql", ".json")

SIBLINGS = ("bsystem-integration-core", "bsystem-hub", "bsystem-design-system")

# Directories this repository owns. A backticked path under one of these is
# unambiguously ours, so an absent file is a real broken reference.
OWNED_ROOTS = (
    "docs/", "scripts/", ".github/", "mocks/", "e2e/", "loadtest/",
    "postgres/", "observability/", "authentik/",
)

SKIP_DIRS = {".git", "node_modules", "artifacts", "backups"}


def repository_files(root: pathlib.Path) -> set[str]:
    files: set[str] = set()
    for path in root.rglob("*"):
        if any(part in SKIP_DIRS for part in path.parts):
            continue
        if path.is_file():
            files.add(str(path.relative_to(root)))
    return files


def sibling_exists(reference: str, root: pathlib.Path) -> bool:
    """Resolve a path that names another repository in the workspace."""
    for sibling in SIBLINGS:
        if reference.startswith(sibling + "/"):
            candidate = root.parent / reference
            # A sibling that is not checked out is not a broken link. Only an
            # absent file inside a checkout that does exist is.
            if not (root.parent / sibling).is_dir():
                return True
            return candidate.exists()
    return False


def check(path: pathlib.Path, root: pathlib.Path, known: set[str]) -> list[str]:
    findings: list[str] = []
    text = path.read_text(encoding="utf-8")
    relative = path.relative_to(root)

    for number, line in enumerate(text.splitlines(), start=1):
        for target in MARKDOWN_LINK.findall(line):
            if target.startswith(("http://", "https://", "mailto:", "#")):
                continue
            target = target.split("#")[0]
            if not target:
                continue
            resolved = (path.parent / target).resolve()
            if resolved.exists():
                continue
            if sibling_exists(target, root):
                continue
            findings.append(f"{relative}:{number}: link target {target!r} does not exist")

        for span in BACKTICK_PATH.findall(line):
            if not PATH_LIKE.match(span) or not span.endswith(CHECKED_SUFFIXES):
                continue
            if span.startswith(("http", "$", "~")):
                continue
            # removeprefix, not lstrip: lstrip takes a character set, so
            # ".github/workflows/ci.yml".lstrip("./") returns
            # "github/workflows/ci.yml" and every dotfile path looks missing.
            candidate = span[2:] if span.startswith("./") else span
            if candidate in known:
                continue
            if sibling_exists(candidate, root):
                continue
            # Only paths rooted in a directory this repository owns are
            # checked. A bare filename like `openapi.yaml`, or a path relative
            # to another repository's root, is ambiguous across the workspace:
            # flagging those produced twenty findings, every one of which named
            # a file that exists in a sibling repository.
            if not candidate.startswith(OWNED_ROOTS):
                continue
            findings.append(f"{relative}:{number}: {span!r} is referenced but no such file exists")
    return findings


def main() -> int:
    root = pathlib.Path(__file__).resolve().parent.parent
    known = repository_files(root)

    documents = sorted(
        path for path in root.rglob("*.md")
        if not any(part in SKIP_DIRS for part in path.parts)
    )

    findings: list[str] = []
    for document in documents:
        findings.extend(check(document, root, known))

    if findings:
        print("documentation references paths that do not exist:\n")
        for finding in findings:
            print(f"  {finding}")
        print(f"\n{len(findings)} finding(s)")
        return 1

    print(f"checked {len(documents)} document(s): every referenced path exists")
    return 0


if __name__ == "__main__":
    sys.exit(main())
