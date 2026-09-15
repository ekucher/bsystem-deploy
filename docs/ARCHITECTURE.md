# BSYSTEM Deployment Architecture

## Environments

BSYSTEM Platform requires isolated environments:

```text
DEV
STAGE
PROD
```

Credentials, databases and externally reachable endpoints must not be shared between production and non-production environments unless explicitly designed and approved.

## Network model

```text
Users / LAN / Internet
        │
        ▼
Reverse Proxy :443
        │
        ├── BSYSTEM-HUB
        ├── authentik
        ├── CRM
        ├── Projects
        ├── QA
        ├── Wiki
        └── Operations

Internal Docker networks
        │
        ├── PostgreSQL
        ├── Redis
        ├── NATS
        ├── Integration Core
        └── observability backends
```

Databases, Redis and NATS should not be published to untrusted networks.

## Data services

Prefer logical separation of databases per application even when sharing a PostgreSQL server/cluster:

```text
authentik_db
hub_db
integration_db
espocrm_db
redmine_db
outline_db
qa_db
```

Direct cross-database application queries are prohibited.

## Reverse proxy

The reverse proxy is responsible for TLS termination and routing. Product domains should remain stable even when underlying container/service topology changes.

Illustrative naming:

```text
hub.<domain>
auth.<domain>
crm.<domain>
projects.<domain>
qa.<domain>
wiki.<domain>
ops.<domain>
portal.<domain>
```

## Secrets

Secrets are injected at deployment time and never committed to Git.

Potential mechanisms include SOPS-backed deployment secrets, an approved secret manager, or platform-native secret mechanisms.

## Persistence and backup

Persistent components must have documented backup and restore procedures.

Priority data:

- PostgreSQL databases;
- application-uploaded files/object data;
- authentik configuration/data;
- Integration Core mapping data;
- deployment configuration required for recovery.

Search indexes should be reconstructable from authoritative data where practical.

## Observability

Platform deployments should support:

- metrics;
- logs;
- traces;
- health checks;
- alerting.

Target tooling:

```text
Prometheus
Grafana
Loki
OpenTelemetry
```

## Upgrade policy

- pin image/application versions;
- avoid floating `latest` tags in production;
- backup before stateful upgrades;
- test upgrades in STAGE;
- document database migrations and rollback limitations;
- keep third-party customizations outside upstream core where possible.
