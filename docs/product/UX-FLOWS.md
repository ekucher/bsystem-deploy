# BSYSTEM UX Flows v1.0

## 1. Create a task

```text
Projects -> Project -> New task
                    |
                    +-- title / description
                    +-- assignee / priority / dates
                    +-- optional relations
                        +-- BUG-*
                        +-- DOC-*
                        +-- INC-*
```

HUB submits the normalized request. For Redmine-owned tasks, Core writes through the Redmine adapter.

## 2. Link a Redmine task to a QA bug

```text
TSK-1842
  Related
    + Add relation
      Type: fixes / relates to
      Entity: BUG-391
```

The relation is stored by BSYSTEM using Global IDs. Neither Redmine nor QA must duplicate the other record.

## 3. Installation operations

```text
Client -> Products -> BRAVO installation
                   -> Overview
                   -> Health
                   -> Backups
                   -> Maintenance
                   -> Updates
                   -> Events
                   -> Incidents
```

## 4. Backup failure to incident

```text
RUN-* backup.failed
      |
      v
Installation attention state
      |
      +--> operator opens execution
      |
      +--> Create incident
             client       prefilled
             installation prefilled
             product      prefilled
             execution    linked
             severity     policy/user selected
```

No client/server identifiers are manually recopied.

## 5. Fleet dashboard

```text
Operations summary
  Installations: healthy / attention / unavailable
  Backups: current / stale / failed
  Maintenance: successful / warning / failed
  Updates: current / available / action required
  Incidents: active by severity
```

Selecting a metric opens Installations or Incidents with the corresponding filter.

## 6. Customer support portal

Customer:

```text
Home
My products
My requests
My incidents (only policy-approved visibility)
Knowledge Base
Files (only scoped shares)
Profile
```

The customer cannot infer another tenant through search counts, relations, identifiers or errors.

## 7. Administration

Administrator manages human accounts through the existing Core/authentik administration boundary, then assigns BSYSTEM roles/scopes. Identity lifecycle and authorization lifecycle are related but distinct.

## UX invariants

- one corporate Design System;
- consistent list/detail/dialog patterns;
- deep links use BSYSTEM Global IDs where possible;
- normalized errors provide a useful next action;
- permission-hidden actions are also denied by backend;
- destructive/high-impact actions require explicit confirmation;
- native upstream UI is an escape hatch, not the default user journey.
