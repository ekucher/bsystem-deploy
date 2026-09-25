# Integration Core — Stage 1 Live Audit

**Repository:** `ekucher/bsystem-integration-core`  
**Branch:** `main`  
**Audited SHA:** `dd2c7f90c887183db9392fdf6db6b980c2adccbf`  
**Audit date:** 2026-09-25

## 1. Exact-head verification

The audited `main` SHA is:

```text
dd2c7f90c887183db9392fdf6db6b980c2adccbf
```

GitHub Actions for this exact SHA were verified:

```text
CI       completed / success
Security completed / success
```

No Stage 1 implementation changes are included in this SHA. It is the baseline.

## 2. Executive conclusion

Stage 1 SHOULD NOT create a new integration service or a parallel object-ID layer.

The existing Integration Core already has the correct reusable foundations:

- immutable Global IDs;
- `(source, entity_type, source_id)` mappings;
- human identity mapping from Authentik `sub`;
- service identities with `SVC-*`;
- durable audit primitives;
- Redmine adapter;
- Outline adapter;
- adapter health/resilience;
- permission-aware search;
- machine and human API separation;
- OpenAPI contract tests;
- PostgreSQL persistence.

The missing Stage 1 primitive is a **generic cross-system relationship store/API**.

Therefore the Stage 1 Core work is an extension:

```text
existing Core
+ QA object registration
+ generic relationships
+ relationship audit
+ safe actor propagation
+ authorization-safe object resolution
```

not a rewrite.

## 3. Global IDs are the canonical object identity layer

Current design:

```text
global_entities
  global_id
  entity_type
  source
  source_id
  tenant_id
  metadata
```

with a unique constraint on:

```text
(source, entity_type, source_id)
```

This is directly compatible with Stage 1 cross-system relationships.

### Existing relevant families

```text
task      -> TSK-*
test_case -> TST-*
bug       -> BUG-*
document  -> DOC-*
user      -> USR-*
service   -> SVC-*
```

Current Redmine issues are mapped as:

```text
entity_type = task
source      = redmine
source_id   = Redmine issue ID
global_id   = TSK-*
```

Current Outline documents are mapped as:

```text
entity_type = document
source      = outline
source_id   = Outline document UUID
global_id   = DOC-*
```

### QA implications

Recommended Stage 1 mappings:

```text
QA Test Case
entity_type = test_case
source      = qa
global_id   = TST-*

QA Bug
entity_type = bug
source      = qa
global_id   = BUG-*

QA Task
entity_type = task
source      = qa
global_id   = TSK-*
```

The same `TSK-*` family may contain Redmine and QA tasks because the authoritative
mapping retains `source` and `source_id`.

### Missing type: Requirement

The current Global ID counter set has no `requirement` family.

Stage 1 SHOULD add:

```text
entity_type = requirement
prefix      = REQ
```

through a new append-only migration.

Do not edit migration `001_init.sql`.

## 4. Reads do not allocate

The existing Core intentionally separates:

```text
LookupGlobalEntity()
```

from:

```text
CreateGlobalEntity()
```

A read does not mint an ID.

This property MUST be retained by the relationship API.

Listing a relationship whose referenced object no longer has a valid mapping
must not silently create or remap an object.

## 5. No generic relationship store currently exists

The audited migrations through `018_module_administration.sql` contain no
generic cross-system relationship table.

Existing occurrences of domain-specific "relations" elsewhere in the platform
must not be treated as the Stage 1 relationship graph.

A new migration is therefore required.

## 6. Proposed relationship persistence

Recommended append-only migration:

```text
019_relationships.sql
```

Conceptual table:

```sql
CREATE TABLE relationships (
    id              BIGSERIAL PRIMARY KEY,
    source_global_id TEXT NOT NULL REFERENCES global_entities(global_id),
    target_global_id TEXT NOT NULL REFERENCES global_entities(global_id),
    relation_type   TEXT NOT NULL,
    created_by      TEXT NOT NULL,
    created_by_service TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CHECK (source_global_id <> target_global_id),
    UNIQUE (source_global_id, target_global_id, relation_type)
);
```

Actual FK/delete policy MUST be reviewed before implementation because Global IDs
are designed to be immutable and non-reused.

Recommended indexes:

```text
(source_global_id)
(target_global_id)
(relation_type)
(created_at)
```

## 7. Canonical relation direction

Store one canonical edge.

Do not separately store both:

```text
A tests B
B tested-by A
```

The inverse should be derived from a controlled vocabulary.

Recommended Stage 1 vocabulary:

```text
related-to        <-> related-to
tests             <-> tested-by
validates         <-> validated-by
documents         <-> documented-by
implements        <-> implemented-by
depends-on        <-> required-by
blocks            <-> blocked-by
references        <-> referenced-by
```

## 8. Relationship API placement

The current Core deliberately separates:

```text
/api/v1/*
```

for human principals from:

```text
/api/service/v1/*
```

for service principals.

Native application integrations are server-side machine clients, but
relationship mutations are usually initiated by a human.

Therefore Stage 1 should expose the canonical storage operation through the
**machine API** and carry a trusted actor assertion supplied by the authenticated
native backend.

Conceptual surface:

```text
GET    /api/service/v1/relationships?object_id={GLOBAL_ID}
POST   /api/service/v1/relationships
DELETE /api/service/v1/relationships/{id}
```

The native application backend:

1. authenticates its user locally;
2. performs its own native authorization;
3. authenticates to Core as its service identity;
4. supplies actor context;
5. Core records both the acting service and asserted end-user identity.

A browser MUST NOT be permitted to invent the actor assertion directly.

## 9. Actor propagation is a Stage 1 security boundary

Existing service calls identify the machine as `SVC-*`.

That is insufficient for relationship audit because the business operation was
performed by a human.

The relationship audit model must distinguish:

```text
service actor:
SVC-qa

end-user actor:
Authentik sub / USR-* where resolvable
local app user id where useful
```

Recommended actor data:

```json
{
  "service_id": "SVC-00000X",
  "user_subject": "authentik-sub",
  "user_global_id": "USR-00000Y",
  "local_user_id": "7"
}
```

The transport format is an implementation decision, but it MUST be:

- server-to-server only;
- protected from browser spoofing;
- bound to the authenticated service;
- included in audit.

## 10. Audit primitives are reusable

The Core already has:

```text
audit_events
```

with:

- subject;
- `global_user_id`;
- action;
- resource type/id;
- request ID;
- source IP;
- metadata;
- timestamp.

Stage 1 SHOULD reuse this audit subsystem.

New actions should use stable names such as:

```text
relationship.created
relationship.deleted
relationship.read
```

Relationship mutation audit should be treated as security-relevant enough that
the failure policy is explicitly chosen.

Recommendation:

```text
create/delete relationship:
fail-closed if the audit record is the only durable provenance of who changed it
```

Prefer performing relationship mutation and its mandatory audit record in the
same PostgreSQL transaction.

## 11. Search is already designed to fail closed

The existing search model is useful but not yet sufficient for native
application authorization.

Important existing properties:

- search documents carry permissions and scope;
- empty permissions make a document invalid/invisible;
- results are narrowed by the provider and re-authorized by Core;
- provider candidate totals are not returned because they may disclose hidden
  records;
- authorization-store failures fail the request rather than returning a
  misleading/unsafe partial result.

This is a strong primitive.

However, the current permission model is **Core platform RBAC**, not necessarily
Redmine/QA/Outline native resource authorization.

Therefore Stage 1 MUST NOT assume current Core search automatically solves
native app delegated authorization.

## 12. Search Stage 1 decision

For the first secure implementation:

```text
exact object ID / canonical URL
```

is allowed before:

```text
broad federated native-object search
```

unless native authorization can be proven for every returned object.

The existing search engine can later be extended with QA entity types and
application-derived authorization metadata.

## 13. Existing search entity types require extension

Current searchable entity types include:

```text
client
contact
project
issue
document
```

Stage 1 relationship UX may require:

```text
requirement
test_case
bug
task
```

If federated search is enabled for those entities, update:

- `internal/search/search.go`;
- provider validation/tests;
- OpenAPI schemas;
- indexing publishers;
- authorization metadata contract.

This is not required for exact-ID/URL linking MVP.

## 14. Adapters

Current production adapter capabilities:

```text
redmine:
  projects.read
  issues.read

outline:
  documents.read
  documents.search
```

These are reusable for Stage 1 object resolution.

There is currently no QA adapter registered in the audited adapter registry.

Stage 1 has two viable patterns:

### Preferred for QA-owned objects

QA backend registers/updates its own Global ID/object metadata through a
service endpoint.

### Alternative

Add a Core QA adapter.

Because QA is a BSYSTEM-controlled native application, push/register through a
service API is simpler and avoids a new polling/read adapter unless Core needs
independent QA reads.

## 15. Service identities

Current machine API requires membership in:

```text
BSYSTEM-Services
```

and allocates stable `SVC-*` IDs from Authentik `sub`.

This is directly reusable.

Stage 1 should create/define machine identities conceptually as:

```text
SVC-qa
SVC-redmine
SVC-outline
```

The exact Authentik subjects/names are deployment configuration.

## 16. Service permissions required

Current service-role capabilities include:

```text
adapters.read
events.publish
global_ids.read
notifications.publish
operations.report
search.index
```

Stage 1 should add least-privilege permissions rather than reuse `*`.

Recommended:

```text
relationships.read
relationships.write
objects.register
objects.read
```

Potentially split write permissions by use case if needed.

## 17. Human Core RBAC is not native app RBAC

The current Core has persistent platform RBAC and scope grants.

It includes concepts such as:

```text
projects.task.read
qa.testcase.read
wiki.document.read
```

For Stage 1 these are **not** authoritative for Redmine, QA or Outline
fine-grained permissions.

They may remain for future platform/HUB behavior, but Stage 1 native application
authorization remains:

```text
Redmine -> Redmine
QA      -> QA
Outline -> Outline
```

Do not delete current Core RBAC merely to implement Stage 1.

## 18. API and contract-test integration

The repository's OpenAPI file is:

```text
docs/openapi.yaml
```

and it is contract-tested against `cmd/server/routes.go`.

Any Stage 1 relationship endpoint therefore MUST update:

- route inventory;
- OpenAPI;
- route/OpenAPI contract tests;
- negative authorization tests.

This is desirable and should be retained.

## 19. Recommended files for Stage 1 implementation

Likely additions:

```text
internal/platformdb/migrations/019_relationships.sql
internal/platformdb/relationships.go
internal/platformdb/relationships_db_test.go

internal/relationships/
  relationships.go
  relationships_test.go

cmd/server/relationships.go
cmd/server/relationships_test.go
```

Likely modifications:

```text
cmd/server/routes.go
docs/openapi.yaml
docs/API.md
docs/GLOBAL-IDS.md
docs/AUDIT.md
docs/AUTHORIZATION.md
internal/platformdb/migrations/<next>-stage1-permissions.sql
internal/search/search.go        # only if federated search is enabled
```

## 20. What MUST NOT be built

Do not build:

- a second Global ID table;
- a second audit subsystem;
- a separate Stage 1 integration service;
- direct QA↔Redmine↔Outline relation databases;
- an unfiltered shared-service global search;
- BHUB frontend dependencies.

## 21. Stage 1 Core implementation verdict

```text
Global IDs             REUSE
Identity mappings      REUSE
Service identities     REUSE
Audit                  REUSE + extend actions/policy
Redmine adapter        REUSE
Outline adapter        REUSE
Search                 REUSE selectively / do not over-trust
Platform RBAC          PRESERVE, but not native-app authority
Relationship store     ADD
Relationship API       ADD
QA object registration ADD
Requirement Global ID  ADD
Actor propagation      ADD
Stage 1 service perms  ADD
```