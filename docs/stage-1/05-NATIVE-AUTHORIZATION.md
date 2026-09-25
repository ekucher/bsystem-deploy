# Native Application Authorization

## 1. Principle

Stage 1 centralizes authentication, not detailed authorization.

```text
Authentik:
Who is the user?
May the user enter this application?

Application:
What may the user do inside this application?
```

## 2. Redmine

Redmine remains authoritative for:

- project membership;
- project roles;
- issue visibility;
- issue editing;
- trackers;
- workflows;
- administration;
- native resource permissions.

## 3. QA

QA remains authoritative for:

- `admin`;
- `tester`;
- `developer`;
- `project_manager`;
- `viewer`;
- project membership;
- project isolation;
- `ai_access`;
- workflow permissions;
- QA administration.

Existing backend guards such as:

```text
requireUser()
requireAccount()
requirePermission()
can()
```

should remain the authorization boundary.

## 4. Outline

Outline remains authoritative for:

- collections;
- groups;
- sharing;
- document visibility;
- document editing;
- workspace settings.

## 5. Authorization and relationships

A relationship does not grant access.

Example:

```text
QA TC-128
  -> documented-by
Outline DOC-X
```

This relationship MUST NOT cause a QA user to receive Outline document permission.

Opening the link still requires Outline to authorize the user.

## 6. Metadata disclosure

Even metadata can be protected.

The platform must treat the following as potentially sensitive:

- object existence;
- object title;
- object status;
- project/collection name;
- search result membership.

Integration APIs must not disclose these fields when the caller is not authorized to see the underlying object.

## 7. Default policy

When authorization cannot be safely determined:

```text
deny metadata disclosure
allow only the least revealing interaction possible
```

For example, Stage 1 may accept a pasted object URL/ID without implementing broad federated search until delegated authorization is solved.