# Contributing to bsystem-deploy

This repository owns the runtime: Compose files, the mock upstreams, the E2E
suite, and the platform's canonical `CLAUDE.md` and `TASKS.md`.

## Setup

Docker with Compose v2, and Go 1.26 for the mocks and the E2E harness.

```bash
docker compose -f docker-compose.e2e.yml up -d --build --wait
E2E_BASE_URL=http://127.0.0.1:8080 go test -C e2e ./... -v
docker compose -f docker-compose.e2e.yml down -v
```

The stack needs no credentials. Every secret in `docker-compose.e2e.yml` is a
test-only placeholder for a throwaway container.

## Before you push

```bash
docker compose config --quiet
docker compose -f docker-compose.e2e.yml config --quiet
python3 scripts/check-hardening.py
python3 scripts/check-identity-groups.py
(cd mocks && gofmt -l . && go vet ./... && go test -race ./...)
(cd e2e && gofmt -l . && go vet ./...)
```

`check-hardening.py` is the one that will surprise you. Trivy's
misconfiguration scanner has no Docker Compose rules, so that script is what
stops a service quietly losing `no-new-privileges`, regaining capabilities,
losing its read-only root filesystem, or publishing a port on every interface.
None of those break anything at runtime, which is the whole problem: the
container starts and serves, and the setting is missed only by whoever is
exploiting it.

It renders the stacks with Docker. If you have none, `--rendered FILE` checks
an already-rendered stack, which is how `scripts/tests/stage-scripts.test.sh`
exercises it — so the script's own logic is covered on every push whether or
not the job that renders can run.

`check-artifacts.py` refuses a compiled executable or an archive tracked in
Git, decided by leading bytes rather than by filename, and checks that every
path `go build` writes to by default is ignored. `go build ./cmd/x` with no
`-o` writes `./x`, and `bsystem-integration-core` committed a 13 MB binary
straight through that gap. When this check was first run here it found four
uncovered mock binaries. Add a Go command and it tells you to add its ignore
line.

`check-identity-groups.py` covers the one mistake the E2E stack cannot catch,
because the stack replaces authentik with a mock. A BSYSTEM group name is
written three times — in `authentik/blueprints/bsystem-groups.yaml`, in the
identity mock, and in the Integration Core's RBAC seed — and the platform is
deny-by-default, so a name that disagrees in one of them grants nothing without
raising anything. On a real deployment that reads as a platform refusing
everyone, with green tests and the cause one character deep in a YAML file. Run
it with the Integration Core checked out beside this repository and it compares
all three lists; on its own it compares the two this repository owns and says
so.

## Writing an E2E scenario

The scenarios are the platform's only test of behaviour that spans services,
and the ones that earn their keep are the **negative** ones: what a caller may
not see, what a refused caller cannot distinguish, what does not travel in an
event.

Two conventions:

- **Identify your own records with a per-run marker.** The stack's database
  outlives a single test, so asserting on absolute counts is a test that
  passes the first time and fails afterwards.
- **Assert byte-identical refusals, not just matching status codes.** A
  different message distinguishes "exists but not yours" from "does not
  exist", which is an enumeration oracle wearing a 404.

## Canonical files

`CLAUDE.md` and `TASKS.md` here are canonical for the whole platform. Update
the task state in the same push as the work, so a reader of either can trust
the other.

## Commits

Conventional Commit style, one coherent change per commit:

```text
feat: add normalized client detail endpoint
fix: normalize upstream timeout errors
test: add tenant isolation matrix
docs: document adapter retry policy
ci: add OpenAPI validation
refactor: extract authorization scope evaluator
security: fix a reachable vulnerability
chore: bump the Go toolchain
```

Not `misc changes`, `update files`, `fix stuff`, `wip`.

**The message body is where the reasoning goes.** A diff shows what changed; it
cannot show what else was considered, or what the change is protecting
against. If a commit's body seems long, read a few in the history and then try
reconstructing the same decision from the diff alone.

Never force-push a shared branch. Never rewrite published history.

## Releases and the changelog

This repository publishes no package. The deployable artifact is a container
image built from a commit, so **the commit is the version** and the image is
tagged with its SHA.

There is therefore no `CHANGELOG.md`, and adding one would create a second
history that drifts from the first. The commit history is the changelog, which
is a large part of why commit messages here carry the reasoning rather than a
restatement of the diff. `TASKS.md` records what is done, what is in progress
and what is blocked.

Whether the platform should also publish tagged releases is an owner decision
that has not been made. Nothing depends on it today: every deployment is built
from a known commit.

## The rule that matters most

**Never weaken a check to get a green build.** Not a disabled test, not a
skipped lint rule, not a broadened allow-list, not a lowered severity
threshold.

A check exists because something went wrong once. Turning it off does not
remove the problem; it removes the only thing that would have told you about
the next one. If a check is wrong, fix the check and say why in the commit —
that is a change a reviewer can evaluate, which a silent exemption is not.

A finding that is genuinely a false positive gets the narrowest possible
remedy, scoped so it cannot mask anything else, with the reasoning written
down. There is a worked example in `bsystem-integration-core/.gitleaksignore`.

## When to stop and ask

Some work cannot be finished without a decision only the owner can make:

- real credentials, API keys or passwords
- production deployment, restart, DNS or TLS
- credential rotation
- destructive database operations
- a commercial commitment, such as an SLA target
- customer ownership that nobody has defined yet

For these, record a blocked entry in `TASKS.md` with what is needed, and move
to the next independent task. **A plausible default for one of these is worse
than a blocked task**, because a blocked task is visible and a guess is not:
an invented SLA target appears in front of a customer as a promise, and reads
exactly like a real one.
