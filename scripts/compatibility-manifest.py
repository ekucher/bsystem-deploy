#!/usr/bin/env python3
"""Write the compatibility manifest: which pair of commits was validated.

A green Autonomous E2E run is a statement about a *pair* — this repository's
stack and a particular Integration Core commit — and until this existed it did
not say which pair. The resolution was logged, but a line in a long log is not
a record: a run validated against a stale provider and one validated against
the intended commit looked identical from the outside.

Written as an artifact rather than only a summary line so that the answer to
"what was this actually green against" survives after the logs scroll by, and
so a stage acceptance can point at a file rather than at a memory.

The Integration Core commit is read from the checkout the job is about to use,
which is the only honest source: an environment variable would record what was
asked for rather than what was resolved.
"""

from __future__ import annotations

import json
import os
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
CORE = ROOT / "bsystem-integration-core"


def head(directory: Path) -> str | None:
    try:
        return subprocess.run(
            ["git", "-C", str(directory), "rev-parse", "HEAD"],
            check=True, capture_output=True, text=True,
        ).stdout.strip()
    except (subprocess.CalledProcessError, FileNotFoundError):
        return None


def main() -> int:
    core_sha = head(CORE)
    if core_sha is None:
        # The manifest exists to say what was validated. A manifest that cannot
        # name the provider is worse than none, because it looks like a record.
        print(
            "the Integration Core checkout is missing, so there is no pair to record",
            file=sys.stderr,
        )
        return 1

    deploy_sha = os.environ.get("DEPLOY_SHA") or head(ROOT) or "unknown"
    reason = os.environ.get("CORE_REASON", "unknown")
    manifest = {
        "deploy": {
            "repository": "ekucher/bsystem-deploy",
            "commit": deploy_sha,
        },
        "integration_core": {
            "repository": "ekucher/bsystem-integration-core",
            "ref": os.environ.get("CORE_REF", "unknown"),
            "commit": core_sha,
            # Whether the pair was chosen or merely defaulted to. A run that
            # fell back to main has validated something real, but not the
            # pairing a reader might assume from a matching branch name.
            "resolution": reason,
        },
        "validated_by": "Autonomous E2E",
    }
    json.dump(manifest, sys.stdout, indent=2, sort_keys=True)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
