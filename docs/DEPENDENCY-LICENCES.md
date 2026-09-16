# Dependency licence inventory

What the BSYSTEM platform depends on, and under what terms. Gathered by reading
each dependency's own licence text — not its `package.json` field alone, and not
a summary of it.

Last gathered: 2026-09-16, against each repository's `main`.

## Why it is split by what actually ships

A licence obligation attaches to what is distributed. A test framework that
never leaves CI and a library linked into the server binary are different
questions wearing the same word "dependency", and counting them together is how
a clean inventory hides the one entry that matters.

So each repository is listed twice: what ships, and what only builds or tests.

## bsystem-integration-core

The shipped set is `go list -deps ./cmd/server` — what is linked into the
binary the image runs — rather than `go list -m all`, which also pulls in the
test and tooling dependencies of dependencies.

### Linked into the server binary

| Module | Version | Licence |
| --- | --- | --- |
| `github.com/jackc/pgpassfile` | v1.0.0 | MIT |
| `github.com/jackc/pgservicefile` | v0.0.0-20240606120523 | MIT |
| `github.com/jackc/pgx/v5` | v5.11.0 | MIT |
| `github.com/jackc/puddle/v2` | v2.2.2 | MIT |
| `github.com/klauspost/compress` | v1.18.5 | BSD-3-Clause † |
| `github.com/nats-io/nats.go` | v1.53.1 | Apache-2.0 |
| `github.com/nats-io/nkeys` | v0.4.15 | Apache-2.0 |
| `github.com/nats-io/nuid` | v1.0.1 | Apache-2.0 |
| `golang.org/x/crypto` | v0.55.0 | BSD-3-Clause |
| `golang.org/x/sync` | v0.22.0 | BSD-3-Clause |
| `golang.org/x/sys` | v0.47.0 | BSD-3-Clause |
| `golang.org/x/text` | v0.41.0 | BSD-3-Clause |

† `klauspost/compress` is BSD-3-Clause at the top of its `LICENSE`, which then
carries the full Apache-2.0 text and an MIT block covering vendored portions
(Go's own `compress` tree and Snappy). A first pass over this file that matched
"Apache License" before checking the opening block classified it as Apache-2.0,
which is what the licence *contains* rather than what the module *is*. Written
down because the mistake is easy to repeat and invisible in a summary.

### Test and tooling only

`creack/pty`, `davecgh/go-spew`, `kr/pretty`, `kr/text`, `pmezard/go-difflib`,
`rogpeppe/go-internal`, `stretchr/objx`, `stretchr/testify`, `golang.org/x/mod`,
`golang.org/x/net`, `golang.org/x/term`, `golang.org/x/tools`, `gopkg.in/check.v1`,
`gopkg.in/yaml.v3`. MIT, ISC and BSD throughout; none is distributed.

## bsystem-hub

### Shipped to the browser

Nine packages, production tree, `npm ls --omit=dev --all`.

| Package | Version | Licence |
| --- | --- | --- |
| `cookie` | 1.1.1 | MIT |
| `jwt-decode` | 4.0.0 | MIT |
| `oidc-client-ts` | 3.5.0 | Apache-2.0 |
| `react` | 19.3.0 | MIT |
| `react-dom` | 19.3.0 | MIT |
| `react-router` | 7.18.4 | MIT |
| `react-router-dom` | 7.18.4 | MIT |
| `scheduler` | 0.28.0 | MIT |
| `set-cookie-parser` | 2.7.2 | MIT |

### Build and test only

219 packages in the full tree. Every one declares a licence, and all are
permissive except the MPL-2.0 entries named below.

## bsystem-design-system

Its `package.json` has **no** `dependencies`: `react` and `react-dom` are
`peerDependencies`, supplied by the consumer. The published `files` are `dist`
and two stylesheets, so the package distributes no third-party code at all.

274 packages in the build and test tree, all permissive except the MPL-2.0
entries and `spawndamnit` below.

## bsystem-deploy

Its three Go modules — `mocks`, `e2e`, `loadtest` — have no `go.sum` and no
`require`. There is no third-party code here to account for, and a guard in
`scripts/tests/stage-scripts.test.sh` keeps it that way, because the first
third-party dependency tends to arrive as a convenience in a test.

## Flagged

Nothing is incompatible. Two things are named rather than left in a count:

**MPL-2.0 — `axe-core` 4.11.0, `lightningcss` 1.33.0 and its platform binaries.**
Present in the build and test trees of both frontend repositories, in neither
production tree. MPL-2.0 is file-level weak copyleft: its obligations attach to
modified MPL-licensed files that are then distributed. Neither package is
modified, and neither is distributed — `axe-core` runs accessibility assertions
in the test suite, `lightningcss` is pulled in by Vite. Compatible as used. It
would need revisiting only if either were vendored into a published bundle.

**`spawndamnit` 3.0.1** declares `"SEE LICENSE IN LICENSE"` rather than an SPDX
identifier, which a tool reading only the manifest field would report as
unknown. Its `LICENSE` file is the MIT text verbatim. Dev-only, via
`@changesets/cli`.

## How to regather this

```sh
# Integration Core: what actually ships, then its licences from the module cache
cd bsystem-integration-core
go list -deps ./cmd/server | grep -E '^[a-z0-9.-]+\.[a-z]+/'

# HUB and Design System: production tree, then the full tree
npm ls --omit=dev --all --json
npm ls --all --json
```

Read the licence file in each package rather than trusting the manifest field:
`spawndamnit` above is why, and `klauspost/compress` is why reading only the
first match is not enough either.

A package listed in `npm ls` but absent from `node_modules` is a platform
binary for another operating system or architecture, not a package with a
missing licence. An earlier pass counted 68 of those as unknown; they are
optional dependencies this machine never installed.
