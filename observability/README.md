# Observability

## What is here

```text
observability/
└── grafana/
    └── bsystem-platform.json   the platform dashboard
```

## Datasource assumptions

The dashboard uses a **datasource variable**, not a hard-coded datasource id.
A dashboard with an embedded uid only works in the Grafana instance it was
exported from, and fails with an empty panel rather than an error anywhere
else.

On import, pick the Prometheus datasource that scrapes the Integration Core.

The queries assume:

| Assumption | Why it matters |
| --- | --- |
| A Prometheus-compatible datasource | The queries use `rate`, `histogram_quantile` and `clamp_min` |
| The Integration Core's `/metrics` is scraped | Every series comes from there |
| A scrape interval of 30s or finer | The panels use `[5m]` windows, which need several samples to be meaningful |
| Metric names are unprefixed | Series are used as exposed: `bsystem_*`. A scrape config that adds a prefix needs the queries adjusted |

`/metrics` is unauthenticated by design and carries no business data, so it
can be scraped without a credential. It should not be exposed publicly — not
because it leaks data, but because it describes the platform's internals to
anyone who asks.

## Scrape configuration

```yaml
scrape_configs:
  - job_name: bsystem-integration-core
    scrape_interval: 15s
    static_configs:
      - targets: ["integration-core:8080"]
```

## Importing

Grafana → Dashboards → New → Import → Upload JSON, then choose the Prometheus
datasource when prompted.

The dashboard is versioned here rather than edited in place in Grafana, so a
change is reviewable and survives the instance being rebuilt. Editing in
Grafana is fine for exploring; export the result back here to keep it.

## What the panels are for

The panels are annotated with what they *mean*, not only what they plot,
because the person reading them at three in the morning is not the person who
built them.

Three are worth knowing about before an incident:

- **Retry share** — the proportion of upstream attempts that were retries. A
  rise means the platform is working harder for the same traffic, and it moves
  before anything user-visible does.
- **Pool acquisitions that waited** — connections acquired against an empty
  pool. This is what explains a slow platform when every individual query is
  fast, and it rises before latency does.
- **Events published by outcome** — "nothing happened" and "everything failed
  to publish" look identical on a chart that counts only successes.

## Reading an empty panel

An empty panel is the platform's most misleading output, because at three in
the morning it reads as "no traffic" when it usually means something else.
Before treating one as evidence, rule these out in order — they are listed
cheapest first.

**The dashboard was imported against the wrong datasource.** Covered above:
every panel is empty, not one.

**The platform has not done that thing yet.** A Prometheus counter publishes no
series at all until something increments it, so a freshly started or freshly
restarted platform shows "No data" on several panels until the first request,
the first upstream call and the first published event. This is the platform
being new, not the query being wrong. It resolves itself; nothing needs doing.

**No adapter has a circuit breaker.** `bsystem_adapter_circuit_state` publishes
nothing when no adapter has one, and the disabled placeholder standing in for
an unconfigured integration does not. On a deployment with no integrations
configured, the Circuit state panel is empty by design.

**The integration was deliberately left out.** An unconfigured integration
answers `503 adapter_not_configured` rather than serving requests, so its
upstream panels stay empty because nothing is calling it. `docs/STAGE-ACCEPTANCE.md`
treats that as a supported configuration; the dashboard has no way to
distinguish it from an integration that has simply gone quiet, so check the
deployment's environment before concluding anything.

**The query is wrong.** Least likely, and now the easiest to rule out: the
Integration Core pins its exposition in `cmd/server/metrics_contract_test.go`,
which renders the real registry and checks every series, its type and its
labels against the names this dashboard uses. A rename that would have emptied
a panel fails that test first.

The full metric reference is in
`bsystem-integration-core/docs/OBSERVABILITY.md`.
