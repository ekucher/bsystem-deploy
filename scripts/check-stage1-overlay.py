#!/usr/bin/env python3
"""Assert what the Stage 1 native-apps/SSO overlay claims about itself.

docker-compose.stage1-native-apps.yml documents, in its own header comment,
that it exists to validate authentik/Integration Core/native-application SSO
"without needing the BHUB frontend running at all" — the whole point of the
overlay is that `hub` is not part of it. Nothing else in this repository's
checks would notice that claim silently becoming false: check-hardening.py
inspects whichever services a rendered stack contains, and a `hub` that crept
back in would simply be one more service it hardens, not a failure.

This renders the overlay layered on the base stack (needs Docker, exactly like
check-hardening.py) and fails when:

  1. the layered stack does not render at all (structural validity — the same
     property `docker compose config --quiet` checks, kept here so a broken
     overlay is reported with which service or key is missing rather than a
     bare `--quiet` exit code);
  2. `hub` appears in the resolved service list;
  3. `hub` is not present, but the mechanism that excludes it (the compose
     profile the overlay comment attributes it to) has been removed — a
     `hub` that vanished because someone renamed it, rather than because the
     overlay excluded it, would otherwise look identical to a clean pass.

    python3 scripts/check-stage1-overlay.py

`--rendered FILE` checks an already-rendered stack instead, so this script's
own tests can run without Docker — the same escape hatch check-hardening.py
uses.
"""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parent.parent
BASE = "docker-compose.yml"
OVERLAY = "docker-compose.stage1-native-apps.yml"
EXCLUDED_SERVICE = "hub"
EXCLUSION_PROFILE = "hub"

# Services the overlay's own header comment promises are brought up in place
# of a locally-unavailable Redmine/Outline. Their absence means the promise in
# the comment is not what the file actually does.
EXPECTED_MOCKS = {"mock-redmine", "mock-outline"}


def render(paths: list[str]) -> dict:
    argv = ["docker", "compose"]
    for path in paths:
        argv += ["-f", path]
    argv.append("config")
    result = subprocess.run(argv, cwd=ROOT, capture_output=True, text=True)
    if result.returncode != 0:
        print("the layered stack did not render:\n", file=sys.stderr)
        print(result.stderr, file=sys.stderr)
        raise SystemExit(1)
    return yaml.safe_load(result.stdout) or {}


def overlay_declares_hub_profile() -> bool:
    """Whether the overlay still routes `hub` through a Compose profile.

    Read from the literal overlay file, not the rendered one: `docker compose
    config` resolves profile membership away entirely when the profile is not
    active, so the only place this is still visible is the source file.
    """
    text = (ROOT / OVERLAY).read_text(encoding="utf-8")
    loaded = yaml.safe_load(text) or {}
    hub = (loaded.get("services") or {}).get(EXCLUDED_SERVICE) or {}
    return EXCLUSION_PROFILE in (hub.get("profiles") or [])


def main(arguments: list[str]) -> int:
    if arguments and arguments[0] == "--rendered":
        if len(arguments) != 2:
            print("usage: check-stage1-overlay.py --rendered FILE", file=sys.stderr)
            return 2
        with open(arguments[1], encoding="utf-8") as handle:
            rendered = yaml.safe_load(handle) or {}
    else:
        rendered = render([BASE, OVERLAY])

    services = set((rendered.get("services") or {}).keys())
    if not services:
        print(
            "the layered stack rendered no service; this check would pass vacuously",
            file=sys.stderr,
        )
        return 1

    problems: list[str] = []

    if EXCLUDED_SERVICE in services:
        problems.append(
            f"{EXCLUDED_SERVICE!r} is present in the resolved Stage 1 overlay "
            f"configuration; the overlay's own header says it starts the "
            f"platform without the BHUB frontend"
        )
    elif not overlay_declares_hub_profile():
        problems.append(
            f"{OVERLAY} no longer gates {EXCLUDED_SERVICE!r} behind the "
            f"{EXCLUSION_PROFILE!r} Compose profile; hub happens to be absent "
            f"from this render, but nothing would stop it reappearing"
        )

    missing_mocks = EXPECTED_MOCKS - services
    if missing_mocks:
        problems.append(
            "the overlay's header comment promises mock upstreams that are not "
            "in the resolved service list: " + ", ".join(sorted(missing_mocks))
        )

    if problems:
        print("the Stage 1 native-apps overlay does not match what it claims:\n")
        for problem in problems:
            print(f"  {problem}")
        print(f"\n{len(problems)} finding(s)")
        return 1

    print(
        f"checked the Stage 1 overlay: {len(services)} service(s) resolved, "
        f"{EXCLUDED_SERVICE!r} absent and gated by the {EXCLUSION_PROFILE!r} "
        f"profile, mock upstreams present"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
