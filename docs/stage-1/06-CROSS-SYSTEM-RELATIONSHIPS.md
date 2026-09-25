# Cross-System Relationships

## 1. Goal

Allow users to create meaningful links among Redmine, QA and Outline objects while preserving source-system ownership.

## 2. Stage 1 resource types

### Redmine

```text
issue
```

### QA

```text
requirement
test_case
bug
task
```

### Outline

```text
document
```

## 3. Canonical object reference

```json
{
  "system": "qa",
  "type": "test_case",
  "external_id": "183"
}
```

Optional presentation metadata:

```json
{
  "display_id": "TC-128",
  "title": "Перевірка створення РЖ",
  "canonical_url": "https://qa.example/testcases/183"
}
```

## 4. Stable identity rule

A human-readable code such as `TC-128` may be shown to users, but integration identity should use an immutable source identifier where available.

## 5. Relationship record

Conceptual model:

```json
{
  "source": {
    "system": "qa",
    "type": "test_case",
    "external_id": "183"
  },
  "target": {
    "system": "redmine",
    "type": "issue",
    "external_id": "5026"
  },
  "relation_type": "tests",
  "created_by": "USR-or-actor-reference",
  "created_at": "..."
}
```

## 6. Relation vocabulary

Minimum Stage 1 vocabulary:

```text
related-to

tests
tested-by

validates
validated-by

documents
documented-by

implements
implemented-by

depends-on
required-by

blocks
blocked-by

references
referenced-by
```

## 7. Inverse relations

Relations should have canonical inverse semantics.

Example:

```text
QA TC-128 tests Redmine #5026
Redmine #5026 tested-by QA TC-128
```

One stored relationship is preferable to two independently managed rows.

## 8. Native UX

Canonical native panel:

```text
Пов'язані об'єкти

Redmine
  #5026  Додати новий вид РЖ

Documentation
  Functional Requirements

[+ Додати зв'язок]
```

## 9. Create flow

```text
+ Add relationship
      ↓
select target system
      ↓
identify target object
      ↓
select semantic relation
      ↓
create
```

For the first secure MVP, target identification may be limited to:

- pasted canonical URL;
- exact issue/document/object ID;
- authorization-safe search where available.

## 10. Delete semantics

Deleting a relationship deletes only the relationship.

```text
Delete relation
!= Delete Redmine issue
!= Delete QA object
!= Delete Outline document
```

## 11. Relationship ownership

Integration Core is the single source of truth for cross-system relationships after migration.

No native application should remain an independent authoritative cross-system relationship database.