#!/usr/bin/env python3
"""Fail when the documentation and the Compose files disagree about a fact.

Three facts live in both places, and each drifts silently:

  1. **Variables.** `.env.example` is what an operator copies. A variable the
     stacks read and that file never mentions is one nobody knows exists —
     which matters most for the ones that decide exposure.

  2. **Ports.** A document telling somebody to open `localhost:8081` when
     nothing publishes 8081 costs them twenty minutes and some of their trust
     in everything else the document says.

  3. **Service names.** A documented `docker compose ... <service>` command
     that names a service the stacks do not have fails in front of whoever is
     following it, usually under time pressure.

The Compose files are parsed as YAML rather than rendered with Docker, so this
runs on a machine that has no Docker — the same reason check-doc-links.py
exists. It therefore reads the literal files rather than a merged stack, which
is enough for names and ports and is why the variable scan is textual.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

import yaml

class ComposeLoader(yaml.SafeLoader):
    """A loader that tolerates Compose's own YAML tags.

    docker-compose.stage.yml writes `ports: !override`, which SafeLoader
    refuses outright. Rendering with Docker would resolve it, but this check
    deliberately does not need Docker, so the tag is dropped and the value
    underneath is kept — which is the value the check is reading anyway.
    """


ComposeLoader.add_multi_constructor(
    "!", lambda loader, suffix, node: loader.construct_object(node.__class__(
        tag=loader.resolve(node.__class__, node.value, (True, False)),
        value=node.value,
        start_mark=node.start_mark,
        end_mark=node.end_mark,
    ), deep=True)
)

ROOT = Path(__file__).resolve().parent.parent
STACKS = ["docker-compose.yml", "docker-compose.e2e.yml", "docker-compose.stage.yml"]
ENV_EXAMPLE = ROOT / ".env.example"

VARIABLE = re.compile(r"\$\{([A-Z_][A-Z0-9_]*)")
DECLARATION = re.compile(r"^\s*#?\s*([A-Z_][A-Z0-9_]*)\s*=", re.MULTILINE)
DOC_PORT = re.compile(r"(?:127\.0\.0\.1|localhost):(\d{2,5})")
# A command is matched inside its backtick span, not across the line. The first
# version ran past the closing backtick: `docker compose up -d --build` followed
# by the prose word "must" was read as a service named "must". A pattern that
# reaches into the sentence around the command will keep finding services in
# English.
BACKTICK = re.compile(r"`([^`\n]+)`")
COMPOSE_SERVICE = re.compile(
    r"docker\s+compose\b.*?\b(?:up|run|exec|logs|restart|stop|start|build|pull)\b"
    r"((?:\s+-[^\s]+)*)\s+([a-z][a-z0-9-]*)"
)

SKIP_DIRS = {".git", "node_modules", "artifacts", "backups"}

# Variables the stacks read that .env.example deliberately does not carry, each
# with its reason. A variable here is a decision; one merely absent is a gap.
UNDOCUMENTED_VARIABLES: dict[str, str] = {
    "INTEGRATION_CORE_CONTEXT": (
        "a build-context path the E2E stack reads in CI, where the sibling "
        "repository is checked out beside this one; not an operator setting"
    ),
}

# Ports named in documentation that no stack publishes, each with its reason —
# an upstream a reader runs themselves, for instance.
EXTERNAL_PORTS: dict[str, str] = {
    "5432": "PostgreSQL's own port, named when describing the container's internal address",
    "4222": "NATS' own port, named when describing the container's internal address",
}

# Words that match the service-name pattern but are Compose's own vocabulary
# rather than a service.
NOT_A_SERVICE = {"up", "down", "run", "exec", "logs", "config", "build", "ps", "pull"}


def documents() -> list[Path]:
    return sorted(
        path
        for path in ROOT.rglob("*.md")
        if not any(part in SKIP_DIRS for part in path.parts)
    )


def stack_text() -> str:
    return "\n".join((ROOT / name).read_text(encoding="utf-8") for name in STACKS)


def stack_services() -> set[str]:
    names: set[str] = set()
    for name in STACKS:
        loaded = yaml.load((ROOT / name).read_text(encoding="utf-8"), ComposeLoader) or {}
        names |= set((loaded.get("services") or {}).keys())
    return names


def published_ports() -> set[str]:
    """Host ports the stacks publish, read from the literal `ports:` entries."""
    ports: set[str] = set()
    for name in STACKS:
        loaded = yaml.load((ROOT / name).read_text(encoding="utf-8"), ComposeLoader) or {}
        for service in (loaded.get("services") or {}).values():
            for entry in (service or {}).get("ports") or []:
                if isinstance(entry, dict):
                    published = str(entry.get("published", ""))
                else:
                    # "127.0.0.1:8080:8080" or "8080:8080"
                    parts = str(entry).split(":")
                    published = parts[-2] if len(parts) >= 2 else ""
                # A published port may itself be a variable in the overlay.
                for token in VARIABLE.findall(published):
                    del token
                if published.isdigit():
                    ports.add(published)
    return ports


def main() -> int:
    problems: list[str] = []

    # 1. Variables.
    # A commented-out assignment counts as declared. An optional setting is
    # presented exactly that way — `#LOG_LEVEL=info` shows the name and the
    # default without overriding it — and what an operator needs from this file
    # is to learn the variable exists, not to have it set.
    declared = set(DECLARATION.findall(ENV_EXAMPLE.read_text(encoding="utf-8")))
    used = set(VARIABLE.findall(stack_text()))
    if not used:
        print("no variable was found in any stack; the check proves nothing", file=sys.stderr)
        return 1
    for missing in sorted(used - declared - set(UNDOCUMENTED_VARIABLES)):
        problems.append(
            f".env.example does not mention {missing}, which the Compose files read; "
            f"an operator copying that file cannot know it exists"
        )

    # 2. Ports.
    published = published_ports()
    if not published:
        print("no published port was found; the port check proves nothing", file=sys.stderr)
        return 1
    for document in documents():
        if document.name == "TASKS.md":
            continue
        for number, line in enumerate(document.read_text(encoding="utf-8").splitlines(), 1):
            for port in DOC_PORT.findall(line):
                if port in published or port in EXTERNAL_PORTS:
                    continue
                problems.append(
                    f"{document.relative_to(ROOT)}:{number}: names port {port}, "
                    f"which no stack publishes"
                )

    # 3. Service names.
    services = stack_services()
    if not services:
        print("no service was found in any stack; the check proves nothing", file=sys.stderr)
        return 1
    for document in documents():
        if document.name == "TASKS.md":
            continue
        for number, line in enumerate(document.read_text(encoding="utf-8").splitlines(), 1):
            for span in BACKTICK.findall(line):
                found = COMPOSE_SERVICE.search(span)
                if not found:
                    continue
                name = found.group(2)
                if name in services or name in NOT_A_SERVICE:
                    continue
                problems.append(
                    f"{document.relative_to(ROOT)}:{number}: names Compose service "
                    f"{name!r}, which no stack defines"
                )

    if problems:
        print("documentation and the Compose files disagree:\n")
        for problem in problems:
            print(f"  {problem}")
        print(f"\n{len(problems)} finding(s)")
        return 1

    print(
        "checked %d variable(s), %d published port(s) and %d service(s) against the documentation"
        % (len(used), len(published), len(services))
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
