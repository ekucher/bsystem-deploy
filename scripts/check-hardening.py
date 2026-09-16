#!/usr/bin/env python3
"""Assert the container hardening in the Compose files has not regressed.

Trivy's misconfiguration scanner understands Dockerfiles but not Compose, so
nothing else in CI would notice a service that quietly loses
no-new-privileges, keeps every Linux capability, or publishes a port on every
interface. This reads the rendered configuration (so anchors, overrides and
variable defaults are already resolved) and fails on any of those.

    python3 scripts/check-hardening.py docker-compose.yml docker-compose.e2e.yml
"""

import subprocess
import sys

import yaml

# authentik's entrypoint manages its own privileges and writes media and
# certificate material. A capability set for it belongs in an owner-run
# hardening pass against a real deployment rather than in an unverified guess,
# so it is exempt from cap_drop and named here instead of failing silently.
CAPABILITY_EXEMPT = {"authentik-server", "authentik-worker"}


def render(paths):
    """Render one stack. Several paths are layered, as `docker compose -f a -f b`.

    An overlay is not a complete stack on its own: docker-compose.stage.yml
    renders to nothing useful alone, and checking it alone would either error
    or pass vacuously. What has to be hardened is the composition that actually
    runs.
    """
    argv = ["docker", "compose"]
    for path in paths:
        argv += ["-f", path]
    argv.append("config")
    out = subprocess.run(argv, check=True, capture_output=True, text=True)
    return yaml.safe_load(out.stdout)


def check(paths, failures):
    label = " + ".join(paths)
    for name, service in sorted(render(paths).get("services", {}).items()):
        where = "%s: %s" % (label, name)

        if "no-new-privileges:true" not in (service.get("security_opt") or []):
            failures.append("%s does not set no-new-privileges" % where)

        dropped = [c.upper() for c in (service.get("cap_drop") or [])]
        if name in CAPABILITY_EXEMPT:
            if dropped:
                failures.append(
                    "%s drops capabilities but is listed as exempt; remove it "
                    "from CAPABILITY_EXEMPT" % where
                )
        elif "ALL" not in dropped:
            failures.append("%s does not drop ALL capabilities" % where)

        if service.get("privileged"):
            failures.append("%s runs privileged" % where)

        for volume in service.get("volumes") or []:
            source = volume.get("source") if isinstance(volume, dict) else volume
            if source and "docker.sock" in str(source):
                failures.append("%s mounts the Docker socket" % where)

        for port in service.get("ports") or []:
            published = port.get("published") if isinstance(port, dict) else None
            host_ip = port.get("host_ip") if isinstance(port, dict) else None
            if published is None:
                continue
            if host_ip in (None, "", "0.0.0.0", "::"):
                failures.append(
                    "%s publishes port %s on every interface" % (where, published)
                )


def main(arguments):
    """Each argument is one stack; use '+' to layer a base file with overlays."""
    stacks = [argument.split("+") for argument in arguments]
    failures = []
    for stack in stacks:
        check(stack, failures)
    for failure in failures:
        print("FAIL: %s" % failure)
    if failures:
        return 1
    print("OK: %d stack(s) pass the hardening checks" % len(stacks))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:] or ["docker-compose.yml", "docker-compose.e2e.yml"]))
