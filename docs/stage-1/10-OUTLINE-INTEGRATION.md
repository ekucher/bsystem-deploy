# Outline Stage 1 Integration

## 1. Canonical controlled build

Repository:

```text
ekucher/bsystem-outline
```

The existing controlled build already contains a BSYSTEM patch for local OIDC logout behavior.

## 2. Critical existing behavior

Local Outline logout must not automatically destroy the central Authentik browser session.

This behavior is Stage 1-compatible and must be preserved.

## 3. Existing users and articles

Outline/Wiki already contains:

- local users;
- documents/articles;
- document authorship;
- revisions/history;
- collections/groups;
- sharing/access rules.

Stage 1 must preserve these local users and their historical ownership.

## 4. Identity binding

Authentik identity must be bound to the existing Outline local user where possible.

A successful SSO migration must not produce:

```text
old user -> owns existing documents
new OIDC duplicate user -> same person, empty history
```

## 5. Native authorization

Outline remains authoritative for:

- document access;
- editing;
- collection membership;
- groups;
- sharing;
- workspace administration.

## 6. Existing Integration Core adapter

The current Integration Core already provides Outline document read/search adapter functionality.

This should be reused for Stage 1 object resolution.

## 7. Native relationship surface

Stage 1 should expose a native or controlled extension surface for related objects where technically maintainable.

Example:

```text
Related Work

Redmine
  #5026

QA
  TC-128
  BUG-55
```

## 8. Implementation policy

Prefer the smallest maintainable mechanism:

1. official extension/hook mechanism if available and sufficient;
2. controlled BSYSTEM patch if necessary;
3. avoid broad upstream forking.

Each additional patch should be separate and independently testable.

Example:

```text
0001-bsystem-local-logout.patch
0002-bsystem-related-objects.patch
```

## 9. Open implementation decision

The exact current Outline extension point for injecting the Related Objects UI must be verified against the pinned Outline version before implementation.