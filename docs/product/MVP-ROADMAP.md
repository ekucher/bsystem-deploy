# BSYSTEM Product Implementation Roadmap

Status: Proposed owner-requested product wave after the P36 foundation freeze.

This roadmap does not reopen foundation work. New tasks must implement the approved product specification or fix defects/acceptance findings.

## P37 — Product specification and domain contract

- [ ] approve Product Specification v1.0
- [ ] approve Domain Model and Data Ownership Matrix
- [ ] add ADR for unified workspace + specialized engines
- [ ] define `INST-*` and `RUN-*` lifecycle/invariants in Core
- [ ] define relationship types and authorization semantics
- [ ] update OpenAPI only after contracts are agreed
- [ ] add non-vacuous tests for Global ID/relation invariants

## P38 — Installation model vertical slice

- [ ] persistence/migrations for installations
- [ ] normalized list/detail APIs
- [ ] client/product/server relationships
- [ ] tenant/scope authorization matrix
- [ ] audit
- [ ] HUB Installations list/detail
- [ ] Design System-only UI
- [ ] E2E

## P39 — Cross-system relations

- [ ] first-class relation persistence/API
- [ ] Task <-> Bug relation
- [ ] Incident <-> Installation relation
- [ ] Document <-> Project relation
- [ ] relation-aware UI
- [ ] authorization/IDOR tests
- [ ] search must not leak inaccessible related entities

## P40 — Product catalog and fleet view

- [ ] APP-* product catalog semantics
- [ ] Products list/detail
- [ ] installations by product
- [ ] version/health aggregation with provenance
- [ ] no arbitrary customer-infrastructure inventory

## P41 — Operational execution model

- [ ] `RUN-*` persistence/API
- [ ] backup execution
- [ ] maintenance execution
- [ ] self-test execution
- [ ] update execution
- [ ] structured result/check model
- [ ] artifact references
- [ ] append/audit semantics

## P42 — BRAVO Toolkit ingestion

- [ ] authoritative Toolkit event/transport contract
- [ ] least-privilege service identity scope per installation
- [ ] authentication
- [ ] validation
- [ ] idempotency/replay protection
- [ ] event-to-RUN mapping
- [ ] credential-safe payload policy
- [ ] mock producer and E2E before live rollout

## P43 — Operations dashboard

- [ ] all-installations overview
- [ ] backup status aggregation
- [ ] maintenance status aggregation
- [ ] update/self-test attention
- [ ] active incident aggregation
- [ ] drill-down through filters rather than duplicate modules

## P44 — Unified task + QA workflow

- [ ] Task detail relations
- [ ] QA Bug detail
- [ ] link/unlink task and bug
- [ ] permissions/audit
- [ ] project context
- [ ] E2E from Project -> Task -> Bug

## P45 — Knowledge Base unified UX

- [ ] normalized document browsing
- [ ] relation-aware document context
- [ ] consistent BSYSTEM styling
- [ ] preserve Outline as document authority
- [ ] define edit/create capability before implementing HUB forms

## P46 — Customer portal slice

BLOCKED until authoritative customer ownership/tenant mapping is approved.

After mapping exists:

- [ ] customer product/installations view
- [ ] support requests
- [ ] permitted incident visibility
- [ ] knowledge/files scope
- [ ] strict tenant negative E2E

## P47 — Product-wave acceptance

- [ ] main-to-main compatibility
- [ ] security/invariant review
- [ ] OpenAPI/implementation agreement
- [ ] accessibility
- [ ] non-vacuous E2E
- [ ] release evidence
- [ ] stage handoff for real integrations

## Owner/runtime blockers

Do not invent:

- customer ownership mapping;
- production tenant assignment;
- BRAVO live inbound contract if not authoritative;
- production credentials;
- SLA targets;
- production DNS/TLS/registry/deployment decisions.
