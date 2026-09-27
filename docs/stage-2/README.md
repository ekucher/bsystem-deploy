# BSYSTEM Stage 2 — Documentation Baseline

**Status:** Draft — first document
**Stage:** Stage 2

## 1. Relationship to Stage 1

`docs/stage-1/` establishes `S1-INV-02`: *"Stage 1 MUST NOT require a BHUB user interface."* Stage 1 native applications (Redmine, QA, Outline) work standalone behind Authentik SSO, with no dependency on `bsystem-hub`.

Stage 2 documents build **on top of** that Stage 1 baseline and may introduce a dependency on the HUB UI where doing so gives real operational value (e.g. a single admin screen instead of four). A Stage 2 document extending or depending on HUB does not amend or violate any Stage 1 invariant — Stage 1 native applications remain fully usable without HUB; HUB is additive tooling for platform administrators, not a required part of the native end-user experience.

## 2. Canonical Stage 2 documents

1. [01-CENTRALIZED-USER-PROVISIONING.md](01-CENTRALIZED-USER-PROVISIONING.md) — single-point user creation with eager/lazy account provisioning across Redmine, QA, Nextcloud, Outline.
