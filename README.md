# BSYSTEM Deploy

Deployment and environment repository for BSYSTEM Platform.

## Purpose

`bsystem-deploy` defines reproducible DEV, STAGE and PROD deployments for BSYSTEM services and integrated third-party platforms.

## Deployment strategy

Initial platform target:

```text
Proxmox VM
  ↓
Ubuntu / Debian
  ↓
Docker Engine
  ↓
Docker Compose
  ↓
BSYSTEM Platform
```

Future migration to Kubernetes/k3s must be possible without changing application domain contracts.

## Initial stacks

Recommended split:

```text
01-core
  authentik
  PostgreSQL
  Redis
  NATS

02-platform
  bsystem-hub
  bsystem-integration-core

03-business
  EspoCRM
  Redmine
  Outline
  BSYSTEM QA

04-operations
  BSYSTEM Operations

05-observability
  Prometheus
  Grafana
  Loki
  OpenTelemetry components
```

## Rules

- no production secrets in Git;
- separate DEV/STAGE/PROD configuration;
- internal databases and brokers are not exposed publicly;
- HTTPS terminates at the approved reverse proxy;
- persistent data uses named volumes or explicitly managed storage;
- backup/restore is part of deployment design;
- every owned service must expose `/health`;
- upgrades must be version-pinned and documented.

## Related repositories

- `ekucher/bsystem-hub`
- `ekucher/bsystem-integration-core`
- `ekucher/bsystem-design-system`

See [Architecture](docs/ARCHITECTURE.md) and [Roadmap](docs/ROADMAP.md).
