# BSYSTEM Deploy Roadmap

## P0 — Local/DEV platform

- base Docker Compose layout
- `.env.example`
- isolated Docker networks
- PostgreSQL
- Redis
- authentik
- BSYSTEM-HUB
- Integration Core
- reverse proxy
- health checks
- persistent volumes
- baseline backup notes

## P1 — Business modules

- EspoCRM
- Redmine
- BSYSTEM QA
- initial database separation
- OIDC wiring where supported

## P2 — Knowledge and operations

- Outline
- BSYSTEM Operations
- observability stack
- centralized logs
- platform health dashboard

## P3 — Event and AI services

- NATS
- search backend
- BSYSTEM AI Gateway
- local model provider integration
- optional cloud model provider integration

## P4 — STAGE/PROD hardening

- environment separation
- secret management
- production TLS
- backup automation
- restore tests
- monitoring and alerting
- resource limits
- image/version pinning
- upgrade runbooks

## P5 — Scale/HA

Evaluate Kubernetes/k3s only when operational requirements justify it:

- horizontal scaling;
- high availability;
- rolling deployment requirements;
- multi-node scheduling;
- stronger orchestration needs.

Docker Compose remains the preferred starting point until these requirements become real.
