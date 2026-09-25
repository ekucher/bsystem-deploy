# Identity Provisioning and Binding

## 1. Problem

Redmine, QA and Outline already contain local users.

Those local users are referenced by domain data and history.

Examples:

```text
Redmine user
  -> issue author
  -> assignee
  -> journal author
  -> project membership

QA user
  -> created_by
  -> updated_by
  -> comments
  -> project_members
  -> notifications
  -> history

Outline user
  -> document author
  -> document revisions
  -> collection/group membership
```

Therefore Stage 1 must bind identities, not replace them.

## 2. Canonical identity graph

```text
                    Authentik identity
                         stable sub
                             |
          +------------------+------------------+
          |                  |                  |
          v                  v                  v
    Redmine user.id      QA users.id      Outline user.id
```

The local identifiers remain authoritative inside each application.

## 3. Binding invariant

```text
Central identity != application-local identity
```

A single person may legitimately have:

```text
Authentik sub = X
Redmine user.id = 42
QA user.id = 7
Outline user.id = UUID-Y
```

## 4. Migration rule

Existing local IDs MUST NOT be renumbered or recreated solely to enable SSO.

## 5. Matching policy

Recommended one-time migration priority:

1. explicit administrator-confirmed mapping;
2. verified unique email match;
3. controlled unique username match only when organization policy guarantees equivalence;
4. otherwise manual binding.

Automatic matching MUST NOT silently bind ambiguous accounts.

## 6. QA schema change

Recommended additive migration:

```text
users.authentik_subject
```

Properties:

- nullable during migration;
- unique once populated;
- immutable after binding except through controlled administrative recovery.

Existing:

```text
users.id
```

must remain unchanged.

## 7. Redmine binding

The Redmine OIDC integration must map Authentik identity to the existing Redmine user wherever possible.

A new Redmine user must not be created if an approved binding to an existing Redmine user already exists.

## 8. Outline binding

The same rule applies to Outline.

Existing authorship and document history must remain tied to the existing Outline local user.

## 9. Provisioning modes

Two supported models may coexist:

### Pre-provisioned

Application admin creates/configures the local user before first SSO login and binds the Authentik identity.

Preferred where project membership/roles must be assigned before access.

### Controlled first-login provisioning

A local user is created on first SSO login only if no approved existing binding exists.

The resulting local role must default to the least privilege appropriate for that application.

## 10. Acceptance requirements

- no duplicate account for an already-bound person;
- all historical data still resolves to the original local user;
- disabling Authentik access prevents new authentication without deleting historical application records;
- application-local roles remain unchanged by SSO migration unless explicitly migrated.