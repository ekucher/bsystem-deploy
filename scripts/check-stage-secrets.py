#!/usr/bin/env python3
"""Fail when a stage asset carries something that looks like a real value.

gitleaks already scans the whole repository for credential patterns. This is
the narrower, complementary check: the stage files are exactly where someone
pastes a real password "just to test it", and where a placeholder that reads
like a real hostname quietly becomes documentation of production.

It looks for two things:

  * a credential-shaped assignment with a value that is neither empty, a
    ``${VAR}`` reference, nor an obvious placeholder;
  * a hostname that is not a reserved documentation domain.

Exits non-zero on a finding, and never prints the suspicious value itself.
"""

from __future__ import annotations

import pathlib
import re
import sys

STAGE_FILES = [
    "docker-compose.stage.yml",
    "docker-compose.stage1-native-apps.yml",
    ".env.example",
    "docs/STAGE-ACCEPTANCE.md",
    "docs/AUTHENTIK-STAGE.md",
    "docs/AUTHENTIK-OIDC.md",
    "docs/SERVICE-IDENTITIES.md",
    "docs/BACKUP-RESTORE.md",
    "docs/TENANT-ISOLATION-MATRIX.md",
    "docs/STAGE-HANDOFF.md",
    "scripts/stage-preflight.sh",
    "scripts/stage-preflight.ps1",
    "scripts/stage-smoke.sh",
    "scripts/stage-smoke.ps1",
    "scripts/release-manifest.sh",
    # Stage 1 native-apps/SSO blueprints: each builds a redirect URI from an
    # operator-supplied public URL via !Env, and 40- mints an API token for
    # each new service identity from another. Both are exactly the kind of
    # file a real value gets pasted into "to see if it works".
    "authentik/blueprints/custom/30-bsystem-redmine.yaml",
    "authentik/blueprints/custom/31-bsystem-qa.yaml",
    "authentik/blueprints/custom/32-bsystem-outline.yaml",
    "authentik/blueprints/custom/40-bsystem-service-identities.yaml",
]

# Deliberately case-sensitive and anchored to an ALL-CAPS identifier. A
# case-insensitive version matched the English word "token" in the prose
# "no human token: authenticated checks will be skipped", and a checker that
# cries wolf is a checker someone switches off.
SECRET_KEYS = re.compile(
    r"\b([A-Z][A-Z0-9_]*(?:PASSWORD|SECRET|TOKEN|APIKEY|API_KEY|PRIVATE_KEY))\b\s*[:=]\s*(?P<value>[^\s,;)\]}\"']+)"
)

# A value that is plainly a template rather than a setting.
PLACEHOLDER = re.compile(
    r"(?i)^(\$\{.*\}|\$[A-Z_]+|<[^>]*>|''|\"\"|-|none|null|ci-[a-z0-9-]+|"
    r"CHANGE_ME[A-Z_]*|REPLACE_ME[A-Z_]*|TODO|x{3,}|X{3,}|"
    r"your-[a-z0-9-]+|fake-[a-z0-9-]+|test-[a-z0-9-]+|example[a-z0-9-]*|"
    r"\[[^\]]*\]|\*+|secret|password|token)$"
)

# RFC 2606 and RFC 6761 reserve these for documentation.
RESERVED_HOST = re.compile(r"(?i)\.(example|invalid|test|localhost)(\b|$)")
HOST = re.compile(r"(?i)\bhttps?://([a-z0-9][a-z0-9.-]*)")

# Hosts that are neither documentation domains nor secrets: Compose service
# names, loopback, and the normative references documentation legitimately
# links to.
ALLOWED_HOSTS = {
    "localhost", "127.0.0.1", "0.0.0.0", "::1",
    "postgres", "nats", "authentik-server", "integration-core", "hub",
    "mock-redmine", "mock-outline",
    "spec.openapis.org", "www.rfc-editor.org", "tools.ietf.org",
    "github.com", "docs.github.com", "opensource.org",
}


def check(path: pathlib.Path) -> list[str]:
    findings: list[str] = []
    try:
        text = path.read_text(encoding="utf-8")
    except FileNotFoundError:
        # A file listed here but absent is not a secret problem. The doc-link
        # check is what notices missing files.
        return findings

    for number, line in enumerate(text.splitlines(), start=1):
        stripped = line.strip()
        if stripped.startswith("#") or stripped.startswith("//"):
            continue
        for match in SECRET_KEYS.finditer(line):
            key = match.group(1)
            value = match.group("value").strip("`'\",")
            if not value or PLACEHOLDER.match(value):
                continue
            if value.startswith("${") or value.startswith("$"):
                continue
            # ${VAR}, ${VAR:-default} and ${VAR:?message} are environment
            # references, which is exactly what a stage asset should contain.
            # Without this, the required-variable guard in the Compose overlay
            # reads as a hardcoded password.
            if line.rfind("${", 0, match.start(1)) > line.rfind("}", 0, match.start(1)):
                continue
            # Report the key and the length. Never the value: a checker that
            # prints what it found turns every CI log into the disclosure it
            # was meant to prevent.
            findings.append(
                f"{path}:{number}: {key} is assigned a literal value "
                f"({len(value)} characters); stage assets must reference the environment"
            )

        for match in HOST.finditer(line):
            hostname = match.group(1).split(":")[0]
            if hostname in ALLOWED_HOSTS or RESERVED_HOST.search(hostname):
                continue
            findings.append(
                f"{path}:{number}: host {hostname!r} is not a reserved documentation domain"
            )
    return findings


def main() -> int:
    root = pathlib.Path(__file__).resolve().parent.parent
    findings: list[str] = []
    for name in STAGE_FILES:
        findings.extend(check(root / name))

    if findings:
        print("stage assets carry values that should not be committed:\n")
        for finding in findings:
            print(f"  {finding}")
        print(f"\n{len(findings)} finding(s)")
        return 1

    print(f"checked {len(STAGE_FILES)} stage asset(s): no committed secret-like value")
    return 0


if __name__ == "__main__":
    sys.exit(main())
