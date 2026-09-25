# Stage 1 Architecture

## 1. Architecture style

Stage 1 uses a hub-and-spoke integration architecture without a user-facing BHUB frontend.

```text
                         Authentik
                       /    |    \
                      /     |     \
                     v      v      v
                Redmine     QA    Outline
                     \      |      /
                      \     |     /
                       \    |    /
                    Integration Core
```

## 2. Responsibility boundaries

### Authentik

Owns:

- central identity;
- authentication;
- MFA;
- password/credential lifecycle;
- central SSO session;
- application-level access entitlement;
- OIDC/OAuth2 provider behavior.

Does not own:

- Redmine project roles;
- QA project membership;
- QA workflow permissions;
- Outline document/collection permissions;
- object relationships.

### Redmine

Owns:

- Redmine local user records;
- projects;
- roles;
- memberships;
- issues;
- journals/comments;
- assignments;
- Redmine-specific administration.

### QA

Owns:

- QA local user records;
- QA roles;
- project memberships;
- requirements;
- test cases;
- bugs;
- tasks;
- checklists;
- application history and audit data;
- QA-specific administration.

### Outline

Owns:

- Outline local user records;
- documents;
- document revisions;
- collections;
- groups;
- sharing rules;
- Outline-native administration.

### Integration Core

Owns:

- cross-system object references;
- Global IDs / stable mappings where applicable;
- cross-system relationship records;
- relation-type vocabulary;
- integration audit;
- adapter registry;
- canonical link metadata;
- limited cached metadata where permitted;
- integration health.

## 3. Data ownership rule

Integration Core MUST NOT become a shadow database of Redmine, QA or Outline.

It may store only the minimum metadata required to:

- identify objects;
- render useful link labels;
- open canonical deep links;
- maintain relationships;
- audit operations.

## 4. Runtime independence

Redmine, QA and Outline must remain independently addressable.

### Native application public URLs

Production hostnames are supplied through deployment variables:

```text
QA      -> ${QA_PUBLIC_URL}
Redmine -> ${REDMINE_PUBLIC_URL}
Wiki    -> ${OUTLINE_PUBLIC_URL}
```

Safe test/example values:

```text
QA_PUBLIC_URL=https://qa.example
REDMINE_PUBLIC_URL=https://redmine.example
OUTLINE_PUBLIC_URL=https://kb.example
```

Real PROD values belong only in the untracked deployment environment.

BHUB availability is irrelevant in Stage 1.

Integration Core outage should degrade relationship functionality but MUST NOT make the native application unusable for its own domain work.

## 5. Level B — Native Applications

Stage 1 uses Level B:

```text
Native application UI
+ shared SSO
+ shared cross-system relationship infrastructure
```

There is no Level A BHUB shell in Stage 1.

## 6. Failure isolation

| Failure | Expected effect |
|---|---|
| Authentik unavailable | new SSO entry may fail; existing app sessions may continue per app policy |
| Integration Core unavailable | related-object panels degrade; native domain work remains available |
| Redmine unavailable | Redmine links fail/open unavailable; QA and Outline remain available |
| QA unavailable | Redmine/Outline remain available |
| Outline unavailable | Redmine/QA remain available |

## 7. Canonical URLs

Each object type must define a canonical deep-link function independent of Integration Core storage.

Examples using the safe documentation domains:

```text
Redmine issue -> ${REDMINE_PUBLIC_URL}/issues/{id}
QA test case  -> ${QA_PUBLIC_URL}/testcases/{id}
QA bug        -> ${QA_PUBLIC_URL}/bugs/{id}
QA task       -> ${QA_PUBLIC_URL}/tasks/{id}
Outline/Wiki  -> ${OUTLINE_PUBLIC_URL}/<native-document-path>
```

Example configuration:

```dotenv
QA_PUBLIC_URL=https://qa.example
REDMINE_PUBLIC_URL=https://redmine.example
OUTLINE_PUBLIC_URL=https://kb.example
```

The exact Outline document path remains native to Outline and must not be reconstructed from mutable document titles.
