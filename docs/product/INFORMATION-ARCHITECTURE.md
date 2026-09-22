# BSYSTEM Information Architecture v1.0

## Primary navigation

```text
Home
Clients
Projects
Tasks
QA
Knowledge Base
Files
Products
Installations
Support
  Incidents
  Requests
Reports
AI
Administration
```

Navigation is permission-driven, but hiding a menu item never replaces backend authorization.

## Entity screen pattern

Business entities should converge on:

```text
List -> Detail -> Related entities -> Actions
```

Create/Edit dialogs exist only when BSYSTEM has a normalized write capability. Do not recreate every native upstream screen by default.

## Clients

Client detail:

```text
Overview | Contacts | Projects | Products | Support | Documents | Files
```

## Projects

Project detail:

```text
Overview | Tasks | QA | Releases | Documents | Files | Activity
```

## Tasks

Task list supports cross-domain context such as project, client, bug/test/release relations and status.

Task detail can link to QA entities by Global ID without duplicating the bug into Redmine.

## QA

```text
Overview | Test Cases | Test Runs | Bugs | Releases
```

## Knowledge Base

Knowledge Base is the BSYSTEM presentation of Outline-backed documents plus authorized relationships to clients/projects/products/incidents.

Suggested navigation:

```text
Knowledge Base
  All documents
  Spaces / Collections
  Recent
  Favorites
  Related to me
```

## Products

Product catalog/fleet view:

```text
Products
  BRAVO
  VETCONTROL
  ...
```

A product view aggregates its installations, versions, health and operational attention state.

## Installations

Installation is the primary Operations screen:

```text
Overview | Health | Backups | Maintenance | Updates | Events | Incidents
```

Backup and maintenance are not mandatory top-level sidebar modules. Fleet dashboards link to filtered installation views.

## Support

```text
Support
  Incidents
  Requests
```

Customer portal exposes only client-scoped support data/actions permitted by policy.

## Administration

Target sections:

```text
Administration
  Users
  Roles & Permissions
  Access Scopes
  Service Identities
  Integrations
  Products
  Reference Data
  Audit
  System / Health
```

authentik remains the identity provider; BSYSTEM administration is the unified control plane and must not duplicate password/identity semantics outside the supported authentik integration.

## Global dashboard

The Home dashboard should aggregate actionable state rather than duplicate every module:

- my/open tasks;
- QA attention;
- active incidents;
- product/installations health;
- backup attention;
- maintenance attention;
- recent activity;
- notifications.

Each card drills into the canonical list with filters.
