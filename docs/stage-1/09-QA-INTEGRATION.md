# QA Stage 1 Integration

## 1. Canonical baseline

Repository:

```text
bsystem-code/qa
```

Audited branch:

```text
main
```

Audited SHA:

```text
803a49f11c14c1ce181b1be41f8c626c21cc1407
```

## 2. Existing authentication

Current implementation uses:

- local `username/password`;
- `password_hash`;
- local `sessions`;
- `qa_session` cookie;
- `requireUser()`;
- `requireAccount()`;
- native roles and project membership.

## 3. Target authentication

Replace only authentication establishment:

```text
Authentik
   ↓
QA OIDC callback
   ↓
bind Authentik sub to existing QA user
   ↓
create QA local session
   ↓
existing native authorization continues
```

## 4. Preserve

MUST preserve:

- `users.id`;
- QA roles;
- project memberships;
- `ai_access`;
- history;
- comments;
- ownership;
- `requireUser()`;
- `requireAccount()`;
- `requirePermission()`;
- existing native permission checks.

## 5. Add

Recommended field:

```text
users.authentik_subject
```

It should be unique and stable after migration.

## 6. Deprecate

After SSO cutover:

- local password login for ordinary users;
- password-change UI for ordinary users;
- password creation as part of normal QA user provisioning.

## 7. Existing Redmine integration

The current `main` already includes:

```text
app/api/redmine/issues/route.js
app/api/redmine/links/route.js
app/api/redmine/settings/route.js
app/api/redmine/test/route.js

components/RedmineLinks.js
components/RedmineSettings.js

lib/redmine.js
lib/links.js
lib/secrets.js
```

This is valuable Stage 1 work and should be reused.

## 8. Existing local relation state

Current Redmine-specific state includes concepts/tables such as:

```text
user_redmine
bug_redmine_issues
test_case_redmine_issues
task_redmine_issues
```

These should not remain authoritative after Stage 1 migration.

## 9. UI reuse

`RedmineLinks` should be generalized rather than discarded.

Target:

```text
RelatedObjects
```

Conceptually:

```jsx
<RelatedObjects
  source={{
    system: "qa",
    type: "test_case",
    id: dbId
  }}
/>
```

## 10. Stage 1 QA object set

Supported:

- Requirement
- Test Case
- Bug
- Task

Checklist can remain out of the initial cross-system scope.

## 11. RedmineSettings

Per-user Redmine URL/API-key configuration should be deprecated once platform-level SSO/integration is operational.

The old configuration must not be dropped until migration and rollback are complete.

## 12. Existing link migration

Existing QA↔Redmine links must be migrated to Integration Core.

Recommended sequence:

1. inventory/count legacy links;
2. export;
3. import objects and relationships;
4. verify counts and mappings;
5. switch reads to Integration Core;
6. stop legacy writes;
7. freeze old tables read-only;
8. remove later after observation.

Avoid long-lived dual-write.

## 13. Branch strategy

Recommended Stage 1 branches:

```text
stage1/qa-authentik-sso
stage1/qa-cross-system-links
```

Create them only from a verified current `origin/main`.