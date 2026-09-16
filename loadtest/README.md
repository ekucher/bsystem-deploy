# Load test harness

```bash
docker compose -f ../docker-compose.e2e.yml up -d --build --wait
go run . -concurrency 16 -requests 2000
docker compose -f ../docker-compose.e2e.yml down -v
```

## What this is, and what it is not

**It is a regression harness.** Run it before a change and after one; a
difference between the two runs means something.

**It is not a capacity test, and its absolute numbers are not a capacity
plan.** It drives four mock upstreams that answer from memory, on whatever
machine is to hand. The numbers describe the mocks and the machine. A real
figure needs production-class hardware, realistic data volumes and real
upstreams, and BSYSTEM has none of the three — see
`bsystem-integration-core/docs/PERFORMANCE.md`, which says the same thing at
more length and refuses to quote a number it cannot stand behind.

## What it exercises, and why those

| Target | Why it is in the list |
| --- | --- |
| `health`, `readyz` | the platform with no identity in the path: the floor |
| `me` | every authenticated request pays for identity resolution |
| `clients`, `contacts`, `issues` | the listings that were N+1; a regression shows here first |
| `notifications`, `servers` | platform-owned reads with no upstream at all |
| `search` | the per-document authorization filter |

`contacts` and `issues` are there specifically because they map two entities
per row. If batching ever regresses to per-item mapping, those two move before
anything else does.

## Errors fail the run

A run with errors exits non-zero, and says so. Percentiles measured over
"whatever happened to succeed" are worse than no percentiles: they look like a
healthy fast platform, because the requests that took longest are the ones
that failed and got excluded.
