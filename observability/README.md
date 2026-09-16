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

The full metric reference is in
`bsystem-integration-core/docs/OBSERVABILITY.md`.
