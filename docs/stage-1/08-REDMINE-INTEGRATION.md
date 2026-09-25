# Redmine Stage 1 Integration

## 1. Existing assets

Existing deployment work already includes:

- managed Redmine runtime;
- PostgreSQL;
- Authentik OIDC integration;
- dedicated Integration Core read-only Redmine API identity;
- Redmine adapter support in Integration Core.

Canonical native integration repository:

```text
ekucher/redmine_bsystem_integration
```

This repository should contain the Stage 1 Redmine plugin.

## 2. Native user preservation

Redmine user IDs must remain stable.

Existing Redmine users may own:

- issues;
- journals;
- comments;
- time entries;
- assignments;
- project memberships;
- preferences/history.

SSO migration must bind Authentik identities to existing Redmine users wherever possible.

## 3. Native authorization

Redmine remains authoritative for:

- project membership;
- issue visibility;
- roles;
- trackers;
- workflow;
- project settings;
- administration.

## 4. Native relationship panel

The Stage 1 plugin should add a native Related Objects surface to issue pages.

Example:

```text
Related Objects

QA
  TC-128  Перевірка створення РЖ
  BUG-55  Некоректний статус

Documentation
  Functional Requirements / Вид РЖ

[+ Add relation]
```

## 5. Integration Core interaction

The plugin should communicate with Integration Core, not directly with QA and Outline for relationship storage.

```text
Redmine plugin
      |
      v
Integration Core
```

## 6. Deep links

Redmine issue canonical URL:

```text
/issues/{issue_id}
```

## 7. Redmine fork policy

Avoid maintaining a full Redmine fork solely for Stage 1 integration where a plugin/theme mechanism is sufficient.

Existing theme work in an upstream fork should be extracted cleanly rather than mixing unrelated upstream changes into Stage 1.

## 8. Version policy

Do not couple Stage 1 identity/linking work to a major Redmine upgrade unless separately approved.

## 9. Search security

A shared Integration Core Redmine service account must not become a mechanism for exposing issues to users who lack native Redmine permission.

Broad issue search should remain disabled or restricted until authorization-safe delegated discovery is implemented.