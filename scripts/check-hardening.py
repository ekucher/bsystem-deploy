#!/usr/bin/env python3
"""Assert the container hardening in the Compose files has not regressed.

Trivy's misconfiguration scanner understands Dockerfiles but not Compose, so
nothing else in CI would notice a service that quietly loses
no-new-privileges, keeps every Linux capability, or publishes a port on every
interface. This reads the rendered configuration (so anchors, overrides and
variable defaults are already resolved) and fails on any of those.

    python3 scripts/check-hardening.py docker-compose.yml docker-compose.e2e.yml

Rendering needs Docker. `--rendered FILE` checks an already-rendered stack
instead, which is how this script's own tests exercise it on a machine with no
Docker installed.
"""

import subprocess
import sys

import yaml

# authentik's entrypoint manages its own privileges and writes media and
# certificate material. A capability set for it belongs in an owner-run
# hardening pass against a real deployment rather than in an unverified guess,
# so it is exempt from cap_drop and named here instead of failing silently.
CAPABILITY_EXEMPT = {"authentik-server", "authentik-worker"}

# Services that run on a read-only root filesystem today. This is a pin rather
# than a policy: PostgreSQL, NATS and authentik all write to theirs and are
# deliberately absent. What it stops is a stateless service quietly losing the
# setting — nothing else would notice, because losing it breaks nothing. The
# container keeps running, and a writable root filesystem is only visible at
# the moment somebody is using it.
#
# A service named here is checked in every stack that contains it, and ignored
# in a stack that does not. docs/DEPLOYMENT.md states this property as one this
# script enforces.
READ_ONLY = {
    "integration-core",
    "integration-core-no-outline",
    "hub",
    "mock-identity",
    "mock-espocrm",
    "mock-redmine",
    "mock-outline",
}


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


# Host paths that are legitimately mounted writable, each with its reason.
# Empty: every bind mount in this repository's stacks is configuration or seed
# data the container reads and must not alter. A writable bind is a container
# given a handle on the host filesystem, so an addition should argue for itself
# in a diff somebody reads.
WRITABLE_BINDS: dict[str, str] = {}

# User-administration credentials are deliberately server-side. Pin their
# placement so a future Compose edit cannot hand either value to the HUB/browser
# container or to an unrelated service.
ADMIN_SECRET_ENV = {
    "BSYSTEM_AUTHENTIK_ADMIN_TOKEN": {"authentik-worker"},
    "AUTHENTIK_ADMIN_TOKEN": {"integration-core", "integration-core-no-outline"},
}


def inspect(label, rendered, failures, seen=None):
    """Check one rendered stack. Returns the number of services it checked."""
    services = (rendered or {}).get("services") or {}

    # A stack that renders no service passes every check below without
    # executing one of them. That is the failure mode this script is least able
    # to notice from its own output, and the most plausible: a renamed file, an
    # overlay checked on its own, a variable that expands to nothing. An
    # unhardened stack and a stack nobody looked at both print OK.
    if not services:
        failures.append("%s renders no service; the checks would pass vacuously" % label)
        return 0

    for name, service in sorted(services.items()):
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

        environment = service.get("environment") or {}
        if isinstance(environment, list):
            environment = {
                str(item).split("=", 1)[0]: item
                for item in environment
            }
        for variable, allowed_services in ADMIN_SECRET_ENV.items():
            if variable in environment and name not in allowed_services:
                failures.append(
                    "%s receives %s, which is restricted to %s"
                    % (where, variable, ", ".join(sorted(allowed_services)))
                )

        if name in READ_ONLY and not service.get("read_only"):
            failures.append(
                "%s does not run on a read-only root filesystem; either restore "
                "read_only or remove it from READ_ONLY with the reason" % where
            )

        for volume in service.get("volumes") or []:
            source = volume.get("source") if isinstance(volume, dict) else volume
            if source and "docker.sock" in str(source):
                failures.append("%s mounts the Docker socket" % where)

            # A bind mount is a handle on the host filesystem. Every one in
            # these stacks carries configuration or seed data inward — an
            # authentik blueprint, the PostgreSQL init scripts — and a
            # container that can rewrite those can change what the next start
            # believes. They are all :ro today, and were before this check;
            # what was missing is anything that keeps them that way, since
            # dropping :ro is a two-character edit that renders and runs.
            if not isinstance(volume, dict) or volume.get("type") != "bind":
                continue
            target = volume.get("target")
            if seen is not None:
                seen["binds"] = seen.get("binds", 0) + 1
            if str(source) in WRITABLE_BINDS:
                continue
            if not volume.get("read_only"):
                failures.append(
                    "%s mounts %s at %s writable; add :ro, or list the source "
                    "in WRITABLE_BINDS with the reason" % (where, source, target)
                )

        for port in service.get("ports") or []:
            published = port.get("published") if isinstance(port, dict) else None
            host_ip = port.get("host_ip") if isinstance(port, dict) else None
            if published is None:
                continue
            if host_ip in (None, "", "0.0.0.0", "::"):
                failures.append(
                    "%s publishes port %s on every interface" % (where, published)
                )

    return len(services)


def main(arguments):
    """Each argument is one stack; use '+' to layer a base file with overlays."""
    failures = []
    checked = []
    seen = {}

    if arguments and arguments[0] == "--rendered":
        if len(arguments) != 2:
            print("usage: check-hardening.py --rendered FILE", file=sys.stderr)
            return 2
        path = arguments[1]
        with open(path, encoding="utf-8") as handle:
            checked.append(inspect(path, yaml.safe_load(handle), failures, seen))
    else:
        for stack in [argument.split("+") for argument in arguments]:
            checked.append(inspect(" + ".join(stack), render(stack), failures, seen))

    # Findings first. A vacuity guard that returns before them answers a
    # different question than the caller asked: a stack with real hardening
    # failures and no bind mount would print only "nothing was examined", and
    # the failures it did find would never be seen.
    for failure in failures:
        print("FAIL: %s" % failure)
    if failures:
        return 1

    # As with the empty stack above: if no stack rendered a bind mount, the
    # read-only check ran over nothing and its silence means nothing. Every
    # stack this repository ships carries at least one.
    if not seen.get("binds"):
        print(
            "no bind mount was examined; the read-only bind check proves nothing",
            file=sys.stderr,
        )
        return 1
    # The counts are printed because they are what a reader can sanity-check:
    # a stack that silently shrank still says OK.
    print(
        "OK: %d stack(s), %d service(s), %d bind mount(s), pass the hardening checks"
        % (len(checked), sum(checked), seen.get("binds", 0))
    )
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:] or ["docker-compose.yml", "docker-compose.e2e.yml"]))
