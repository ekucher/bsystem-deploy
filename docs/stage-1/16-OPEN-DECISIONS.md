# Stage 1 Open Decisions

These items require explicit resolution before final implementation.

## OD-01 — Integration Core relationship reuse — RESOLVED

Live audit of `main @ dd2c7f90c887183db9392fdf6db6b980c2adccbf` confirms:

- existing Global ID/object mapping primitives are reused;
- no generic cross-system relationship table currently exists;
- Stage 1 adds relationship persistence/API on top of `global_entities`;
- no parallel ID subsystem is created.

## OD-02 — Authorization-safe cross-system search — BASELINE DECISION

For the first Stage 1 secure MVP:

```text
exact object ID / canonical URL
```

is sufficient.

Broad federated search MUST NOT be enabled for a source until native-resource
authorization can be preserved for every returned result.

The current Core search is permission-aware, but its platform RBAC is not a
substitute for Redmine/QA/Outline native authorization.

## OD-03 — Redmine existing-user binding mechanism

**Question:** What exact Redmine OAuth/OIDC plugin field or mapping will carry stable Authentik identity?

Must be verified against the deployed plugin/version.

## OD-04 — Outline existing-user binding behavior

**Question:** Does current Outline OIDC behavior map existing users by stable identity/email in a way that preserves the existing local user record?

Must be verified against the pinned controlled build.

## OD-05 — Outline Related Objects UI extension point

**Question:** Is there a stable extension/hook, or is a controlled patch required?

Prefer minimal patching.

## OD-06 — Authentik entitlement naming

Minimum conceptual entitlements:

```text
redmine.access
qa.access
outline.access
```

Final naming and group-binding conventions must be fixed in deployment documentation.

## OD-07 — Identity binding administration

**Question:** Where is the one-time cross-application identity-binding inventory maintained during migration?

Possible options:

- controlled migration file outside Git secrets;
- deployment migration tool;
- per-application admin workflow;
- dedicated audited migration command.

## OD-08 — Legacy QA password rollback window

**Question:** How long are local password hashes retained after SSO cutover before destructive cleanup?

Recommendation: do not remove them in the first cutover release.

## OD-09 — QA user provisioning

Choose explicitly:

- pre-provisioning only;
- controlled first-login provisioning;
- hybrid.

For enterprise Stage 1, pre-provisioning or hybrid with least-privilege defaults is preferred.

## OD-10 — Relationship type governance

Define which roles/users may create which semantic relationship types and whether arbitrary custom types are allowed.

Recommendation: fixed Stage 1 vocabulary, extensible only through controlled configuration.