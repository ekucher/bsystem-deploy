# ADR-001: Unified workspace over specialized engines

Status: Proposed  
Date: 2026-09-22

## Context

BSYSTEM needs one corporate workspace across CRM, projects/tasks, QA, knowledge, files, support and operations. Users should not repeatedly enter the same client/project/product context in multiple systems. The platform also requires unified SSO, administration, RBAC, audit and visual language.

The alternative is either to keep independent products with weak links, or merge/rewrite all domains into one monolithic application/database.

## Decision

Adopt a **unified BSYSTEM workspace over independently deployable specialized engines**.

- HUB is the primary user workspace.
- authentik is the identity provider.
- Integration Core is the authorization/integration/normalization boundary.
- EspoCRM, Redmine, Outline and Nextcloud remain authoritative engines for their domains.
- BSYSTEM-native domains such as QA, Support and Operations can live behind the same Core boundary.
- Cross-system relationships use immutable Global IDs.
- Shared corporate appearance is implemented in HUB through the BSYSTEM Design System rather than maintained forks of every upstream UI.
- Native upstream UIs may remain available for administration or capabilities not yet normalized.

## Why not one monolith

A monolith could simplify some transactions and UI work, but would require BSYSTEM to own replacements for mature CRM, project, wiki and file capabilities, migrations, security updates and domain behavior. It would also increase release coupling and make independent upgrades harder.

## Why not independent applications only

Independent applications preserve upgrade autonomy but leave users with repeated navigation, inconsistent identity/permissions/style, duplicated context and fragile manual links.

## Consequences

Positive:

- one user experience;
- one SSO/admin model;
- source ownership remains clear;
- engines can evolve independently;
- cross-domain links become explicit;
- BSYSTEM can replace an engine behind an adapter without rewriting every consumer.

Costs:

- Integration Core contracts require discipline;
- distributed failure modes remain;
- normalized UI does not automatically expose every native feature;
- cross-system writes require careful transactional/idempotency design;
- Design System consistency applies directly to BSYSTEM UI, not automatically to every native upstream admin screen.

## Guardrails

No direct cross-system DB joins. No local shadow write when an authoritative upstream write fails. No frontend-only authorization. No relation may grant access to an unauthorized entity. No upstream fork solely for visual consistency without a separate ADR and maintenance justification.
