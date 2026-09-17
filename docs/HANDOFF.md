# Autonomous handoff

Everything that could be built, proved and merged without owner credentials is
done. What remains needs a person with access to a real environment or the
authority to make a business decision.

This lists those items in the order an owner would execute them. Each says what
it unblocks, because an ordered list without that is a queue rather than a plan.

## What "done" means here

Every autonomous task in this wave was merged to `main` with CI and Security
green, and with its protected property verified by mutation rather than by
reading — the check was broken on purpose and confirmed to fail. Where a
mutation silently did nothing, that was found and the mutation redone; three of
those happened, and they are recorded where they were fixed.

Nothing below is blocked on work in this repository.

## Order of execution

### 1. A Docker host

**Blocks:** everything after it.

Run `docker compose up -d --build` on the intended host, then
`scripts/stage-preflight.sh` and `scripts/stage-smoke.sh`. The preflight refuses
to continue on an exposure mistake and never prints a secret; the smoke runner
only reads, and a test holds both properties.

Until this is done, the whole stage package is validated in CI and has never
run on the machine it is meant for.

### 2. Secrets for that host

**Blocks:** authentik, the database, every adapter.

`.env.example` lists every variable the stacks read, including the ones with
defaults. Two decide exposure and are easy to confuse: `BIND_ADDRESS` governs
the base stack and `STAGE_PUBLISH_ADDRESS` governs stage — setting the first
and expecting it to hold for stage is the mistake the preflight now catches.

Generate `POSTGRES_PASSWORD` and `AUTHENTIK_SECRET_KEY` freshly. Nothing in
this repository has ever held a real one, and `scripts/check-stage-secrets.py`
fails CI if a value that looks like a secret is committed.

### 3. authentik: client, provider, MFA

**Blocks:** every human login, and therefore every acceptance step below.

`docs/AUTHENTIK-STAGE.md` has the exact values. The group names it creates must
match the platform's RBAC seed; `scripts/check-identity-groups.py` compares the
blueprint, the identity mock and the seed, so a typo fails CI rather than
producing a platform that refuses everyone.

### 4. DNS, TLS and the reverse proxy

**Blocks:** browser access and the OIDC redirect URIs registered in step 3.

The stack binds to loopback by default and expects a TLS-terminating proxy in
front. Decide the `X-Forwarded-For` trust rules before the platform is reachable
from the internet: the Integration Core reads that header and it is only
trustworthy when the proxy sets it.

### 5. Upstream credentials, one integration at a time

**Blocks:** the normalized read paths for each.

- EspoCRM — URL, API key, and a check that the account and contact fields match
  what the adapter expects.
- Redmine — URL, API key, and the custom-field validation.
- Outline — URL and a **scoped** key, plus its permissions.

Take them one at a time. A deployment with an integration left out is a
supported configuration and the E2E stack now runs one every build, so a
missing upstream degrades cleanly rather than breaking the platform.

### 6. Customer ownership mapping

**Blocks:** tenant isolation in production, and nothing else.

This is a business fact, not a technical one: which client records belong to
which tenant. The platform is deny-by-default and an unmapped record resolves to
no access, so a wrong guess here is not a security hole — but it is invisible
wrongness, which is worse to discover later. It has to come from the owner.

### 7. Production tenant assignment

**Blocks:** go-live.

Follows from step 6 and cannot be derived from it automatically.

### 8. Backup and restore, accepted on real data

**Blocks:** any claim that the platform is recoverable.

`docs/BACKUP-RESTORE.md` has the procedure. Accepting it means performing a
restore, not reading about one.

### 9. A container registry, and the HUB's own build

**Blocks:** deploying the images CI has tested, scanned and described.

The release pipeline builds each product image once, validates the exact image
it built, scans it, and publishes a manifest and SBOMs bound to its identity —
and pushes nothing. Pushing needs a registry credential.

The HUB needs one more thing: it bakes its OIDC issuer and client id in at
build time, so the image CI builds carries reserved placeholders and is not
deployable. A deployable HUB image is built with the deployment's own issuer,
and its identity will differ from the manifest's. Record the new one.

`docs/RELEASE.md` has the promotion steps and the rule that matters: never
retag an existing tag onto a new image, because the tag is how the manifest is
joined to the thing running.

## Decisions, not credentials

These need an owner's judgement rather than access.

### SLA targets

The support module enforces whatever policy it is given and currently claims
none. Response and resolution times per severity are a commitment to customers,
and inventing plausible ones would put a number in front of somebody who would
reasonably treat it as agreed.

### Whether the HUB adopts the Design System

Nothing consumes the Design System today: the HUB declares no dependency on it
and imports neither its tokens nor its components. The package is ready for
versioned publishing. Until something consumes it, there is no consumer build
to validate, and a check invented for one would report a compatibility nothing
depends on.

### Publishing the Design System package

Ready, not done. Needs a registry and the decision to publish.

### The BRAVO inbound adapter

The platform writing to BRAVO is specified; reading an inventory or a history
*out of* it is not, and the shape of that data is an owner question.

### ESLint for the HUB

Blocked upstream rather than by choice: `typescript-eslint` supports
`typescript >=4.8.4 <6.1.0` and the HUB is on TypeScript 7. Adding it means
forcing an unsupported resolution or downgrading the compiler. The HUB is
typechecked and tested; it is linting specifically that waits on upstream.

## Two things this wave could not demonstrate

Recorded so they are not mistaken for oversights.

**NATS unavailable, and PostgreSQL unavailable during readiness.** Readiness
already distinguishes both — `nats: degraded` without making the platform
unready, and 503 with `database: error` when the pool cannot ping. What is
missing is the demonstration: the scenarios talk HTTP and the NATS wire
protocol to a *running* stack and cannot stop a container mid-suite. Proving
recovery in particular needs container lifecycle control in the E2E job.

**A dependency dropped from a manifest but still described in prose.** A removed
service is caught the moment a document names it in a Compose command, and a
removed file the moment a document links to it. A dependency described only in
prose is not.
