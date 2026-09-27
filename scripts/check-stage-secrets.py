#!/usr/bin/env python3
"""Fail when a tracked file carries something that looks like a real value.

gitleaks already scans the whole repository for credential patterns tuned to
API-key/token shapes. This is the narrower, complementary check for two
things gitleaks' default rules do not cover well:

  * a credential-shaped assignment (KEY=value / KEY: value) with a value
    that is neither empty, a ``${VAR}`` reference, nor an obvious
    placeholder -- including inside comments, where a real value pasted
    "just to see if it works" is just as much a leak as one in live config;
  * a hostname that is not a reserved documentation domain -- with or
    without a URL scheme, since a bare hostname mentioned in prose is
    exactly how a real stage domain has slipped into this repo before.

Scans every git-tracked file (not a fixed list): a new config/doc file is
covered from the moment it's added, not from the moment someone remembers to
list it here. Exits non-zero on a finding, and never prints the suspicious
value itself.
"""

from __future__ import annotations

import pathlib
import subprocess
import sys

import re

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
    r"CHANGE_ME[A-Z_]*|REPLACE_ME[A-Z_]*|TODO|x{3,}|X{3,}|(.)\2{4,}|"
    r"your-[a-z0-9-]+|fake-[a-z0-9-]+|test-[a-z0-9-]+|e2e-[a-z0-9-]+|"
    r"a-[a-z0-9-]+-somebody-forgot|example[a-z0-9-]*|placeholder|"
    r"\[[^\]]*\]|\*+|secret|password|token)$"
)

# RFC 2606 and RFC 6761 reserve these for documentation and local-only use --
# either as their own suffix (foo.example, foo.local) or, per RFC 2606's own
# example.com/.org/.net, as a subdomain of a real TLD.
RESERVED_HOST = re.compile(
    r"(?i)(?:^|\.)(?:example|invalid|test|localhost|local)(?:\.(?:com|org|net))?$"
)
SCHEMED_HOST = re.compile(r"(?i)\bhttps?://([a-z0-9][a-z0-9.-]*)")

# A bare hostname with no URL scheme is only checked against a curated list
# of real public TLDs -- NOT "any dotted alphabetic label", which matches
# overwhelmingly more dotted code identifiers (`json.load`, `bsystem.events`,
# `hub.access`) and Go/JS method chains than real domains. Deliberately
# excludes ccTLDs that collide with common file extensions (.sh, .io is kept
# since it rarely collides and is heavily used for real services). Extend
# this list if a real domain with a different suffix needs catching, rather
# than widening the shape of the match.
REAL_TLDS = {
    "com", "org", "net", "io", "dev", "app", "co", "ai", "biz",
    "me", "uk", "de", "pl", "ua", "fr", "es", "it", "nl", "ru", "cn", "jp",
    "in", "us", "ca", "au", "br", "xyz", "cloud", "tech", "site", "online",
    "store", "pro", "tv", "cc", "gg",
}
BARE_HOST = re.compile(
    r"(?i)\b((?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.){1,}[a-z]{2,24})\b"
)

# Hosts that are neither documentation domains nor secrets: Compose service
# names, loopback, and the normative references documentation legitimately
# links to.
ALLOWED_HOSTS = {
    "localhost", "127.0.0.1", "0.0.0.0", "::1",
    "postgres", "nats", "authentik-server", "integration-core", "hub",
    "mock-redmine", "mock-outline", "mock-identity", "mock-espocrm",
    "redmine", "nextcloud",
    "spec.openapis.org", "www.rfc-editor.org", "tools.ietf.org",
    "github.com", "docs.github.com", "opensource.org", "goauthentik.io",
    "golang.org", "gopkg.in", "ghcr.io", "honnef.co", "apps.nextcloud.com",
    "www.w3.org",
}

# Binary/generated/vendor content this check cannot meaningfully evaluate.
EXCLUDE_PATH = re.compile(
    r"(?:^|/)(node_modules|\.git|dist|build|vendor|coverage)/|"
    r"package-lock\.json$|\.(?:png|jpg|jpeg|gif|ico|svg|woff2?|ttf|eot|pdf|zip|gz|tar)$"
)


def tracked_files(root: pathlib.Path) -> list[pathlib.Path]:
    out = subprocess.run(
        ["git", "ls-files"], cwd=root, capture_output=True, text=True, check=True
    ).stdout
    return [
        root / name
        for name in out.splitlines()
        if name and not EXCLUDE_PATH.search(name)
    ]


def check(path: pathlib.Path) -> list[str]:
    findings: list[str] = []
    try:
        text = path.read_text(encoding="utf-8")
    except (FileNotFoundError, UnicodeDecodeError):
        return findings

    for number, line in enumerate(text.splitlines(), start=1):
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
                f"({len(value)} characters); commit only a reference to the environment"
            )

        hosts_on_line: set[str] = set()
        for match in SCHEMED_HOST.finditer(line):
            hosts_on_line.add(match.group(1).split(":")[0])
        for match in BARE_HOST.finditer(line):
            hostname = match.group(1)
            if hostname.rsplit(".", 1)[-1].lower() not in REAL_TLDS:
                continue
            hosts_on_line.add(hostname)

        for hostname in hosts_on_line:
            if hostname.lower() in ALLOWED_HOSTS or RESERVED_HOST.search(hostname):
                continue
            findings.append(
                f"{path}:{number}: host {hostname!r} is not a reserved documentation domain"
            )
    return findings


def main() -> int:
    root = pathlib.Path(__file__).resolve().parent.parent
    findings: list[str] = []
    files = tracked_files(root)
    for path in files:
        findings.extend(check(path))

    if findings:
        print("tracked files carry values that should not be committed:\n")
        for finding in findings:
            print(f"  {finding}")
        print(f"\n{len(findings)} finding(s)")
        return 1

    print(f"checked {len(files)} tracked file(s): no committed secret-like value or real hostname")
    return 0


if __name__ == "__main__":
    sys.exit(main())
