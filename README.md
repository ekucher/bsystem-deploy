# BSYSTEM Deploy

Deployment and environment repository for BSYSTEM Platform.

## Purpose

`bsystem-deploy` defines reproducible DEV, STAGE and PROD deployments for BSYSTEM services and integrated third-party platforms.

## Current P0 stack

The repository now contains a runnable Docker Compose foundation with:

- PostgreSQL 17
- NATS 2.11 with JetStream
- authentik 2026.8.2 (`server` + `worker`)
- `bsystem-integration-core`
- `bsystem-hub`

The stack has no Redis. It carried one until nothing had used it for the
length of the project: authentik here is configured against PostgreSQL alone,
and the Integration Core reads no `REDIS_URL` and references Redis nowhere.
`bsystem-integration-core/docs/ARCHITECTURE.md` reserves Redis for caching,
distributed locks, short-lived state and rate limiting; the first feature that
needs one adds the service back, with a use, rather than finding it already
running and assuming it is in use.

## Quick start

Clone these repositories side-by-side:

```text
workspace/
├── bsystem-hub/
├── bsystem-integration-core/
└── bsystem-deploy/
```

Then:

```bash
cd bsystem-deploy
cp .env.example .env
# Replace all CHANGE_ME values.
docker compose up -d --build
```

Verify:

```bash
docker compose ps
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/modules
curl http://localhost:8081/healthz
```

Local endpoints:

- BSYSTEM HUB: `http://localhost:8081`
- Integration Core: `http://localhost:8080`
- authentik: `http://localhost:9000`

Detailed runbook: [docs/RUN-P0.md](docs/RUN-P0.md).

## Autonomous E2E stack

`docker-compose.e2e.yml` brings up the platform against deterministic mock
upstreams, so BSYSTEM can be validated end to end with no owner credentials and
no production access:

```bash
docker compose -f docker-compose.e2e.yml up -d --build --wait
(cd e2e && E2E_BASE_URL=http://127.0.0.1:8080 go test ./...)
docker compose -f docker-compose.e2e.yml down -v
```

Every credential in that stack is a documented test-only placeholder. See
[docs/E2E-ENVIRONMENT.md](docs/E2E-ENVIRONMENT.md) and
[mocks/README.md](mocks/README.md).

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

Recommended split as the platform grows:

```text
01-core
  authentik
  PostgreSQL
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
- every owned service must expose `/health` (or `/healthz` for static edge containers);
- upgrades must be version-pinned and documented.

## Related repositories

- `ekucher/bsystem-hub`
- `ekucher/bsystem-integration-core`
- `ekucher/bsystem-design-system`

See [Architecture](docs/ARCHITECTURE.md), [P0 Foundation](docs/P0-FOUNDATION.md), [Run P0](docs/RUN-P0.md), [E2E Environment](docs/E2E-ENVIRONMENT.md), and [Roadmap](docs/ROADMAP.md).
