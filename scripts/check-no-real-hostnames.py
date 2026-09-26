#!/usr/bin/env python3
"""Fail when any tracked file references a hostname that is neither a
reserved documentation domain (RFC 2606/6761) nor an explicitly allowed
real service.

`check-stage-secrets.py` already does this for the specific STAGE_FILES most
likely to accumulate a real value pasted in "to see if it works" -- this is
the broader, repo-wide sweep: nothing here is scoped to a curated file list,
because a leaked hostname can land in a test file or a code comment just as
easily as a stage asset (that is exactly how it happened once, in a sibling
repository's test file, during a Stage 1 SSO round -- this script exists so
the same class of leak is caught here too, in any file, not just the ones
someone remembered to list).

Exits non-zero on a finding. Prints the hostname (not a secret, so this is
safe, unlike check-stage-secrets.py's credential findings).
"""

from __future__ import annotations

import pathlib
import re
import subprocess
import sys

# RFC 2606 and RFC 6761 reserve these for documentation and examples.
RESERVED_HOST = re.compile(r"(?i)(?:^|\.)(?:example|invalid|test|localhost)(?:\.[a-z]{2,})?$")

# Real services this platform's docs/code legitimately reference (specs,
# standards bodies, actual third-party APIs) -- not a deployment hostname.
ALLOWED_HOSTS = {
    "127.0.0.1", "0.0.0.0", "::1",
    "github.com", "docs.github.com", "opensource.org",
    "goauthentik.io", "apps.nextcloud.com",
    "spec.openapis.org", "www.rfc-editor.org", "tools.ietf.org", "www.w3.org",
}

HOST_RE = re.compile(r"\bhttps?://([a-zA-Z0-9][a-zA-Z0-9.-]*)")

# Generated/vendored/binary content that would only produce noise no one can
# act on -- not a real exemption from the check, just not source a human
# wrote by hand.
EXCLUDE_PATH = re.compile(
    r"(^|/)(node_modules|\.git)/"
    r"|(^|/)(go\.sum|package-lock\.json)$"
    r"|\.(png|jpg|jpeg|gif|svg|ico|pdf|zip|gz)$"
)


def tracked_files(root: pathlib.Path) -> list[str]:
    out = subprocess.run(
        ["git", "ls-files"], cwd=root, capture_output=True, text=True, check=True
    ).stdout
    return [f for f in out.splitlines() if f and not EXCLUDE_PATH.search(f)]


def check(path: pathlib.Path, rel: str) -> list[str]:
    findings: list[str] = []
    try:
        raw = path.read_bytes()
    except FileNotFoundError:
        return findings
    if b"\x00" in raw[:8000]:
        return findings  # binary
    try:
        text = raw.decode("utf-8")
    except UnicodeDecodeError:
        return findings

    for number, line in enumerate(text.splitlines(), start=1):
        for match in HOST_RE.finditer(line):
            hostname = match.group(1).split(":")[0].lower()
            # A bare single label (no dot at all) is never a real,
            # publicly-resolvable domain -- a Compose service name
            # (`redmine`, `authentik-server`) or an obvious placeholder
            # (`IP`, `x`), neither of which this check is meant to catch.
            if "." not in hostname:
                continue
            if hostname in ALLOWED_HOSTS or RESERVED_HOST.search(hostname):
                continue
            findings.append(f"{rel}:{number}: host {hostname!r} is not a reserved documentation domain or an allowed real service")
    return findings


def main() -> int:
    root = pathlib.Path(__file__).resolve().parent.parent
    files = tracked_files(root)
    findings: list[str] = []
    for rel in files:
        findings.extend(check(root / rel, rel))

    if findings:
        print("Found real-looking hostnames committed to the repo:\n")
        for finding in findings:
            print(f"  {finding}")
        print(
            f"\n{len(findings)} finding(s). Use a reserved documentation domain "
            "instead (e.g. qa.example, *.example.invalid), or add the host to "
            "ALLOWED_HOSTS in scripts/check-no-real-hostnames.py if it's a real "
            "third-party service this platform legitimately references."
        )
        return 1

    print(f"checked {len(files)} tracked file(s): no committed real-looking hostname")
    return 0


if __name__ == "__main__":
    sys.exit(main())
