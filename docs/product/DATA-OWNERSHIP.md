# BSYSTEM Data Ownership Matrix v1.0

## Rule

BSYSTEM presents one workspace but does not create multiple independent masters for the same data.

| Domain / data | Authoritative owner | BSYSTEM behavior |
| --- | --- | --- |
| Human identity | authentik | map to USR-*, authorize through Core |
| Client/account | EspoCRM | normalize and relate through CL-* |
| Contact | EspoCRM | normalize and relate through CT-* |
| Project | Redmine | normalize and relate through PR-* |
| Task/issue | Redmine | normalize and relate through TSK-* |
| QA test case | BSYSTEM QA | own TST-* |
| QA bug | BSYSTEM QA | own BUG-* |
| Wiki document | Outline | normalize and relate through DOC-* |
| File bytes/folders | Nextcloud | expose through files capability; do not duplicate by default |
| Product catalog | BSYSTEM | own APP-* |
| Installation | BSYSTEM Operations | own INST-* and upstream/telemetry references |
| Supported server facts | BSYSTEM Operations | own/derive SRV-* with provenance |
| Backup/maintenance/update/self-test execution | BSYSTEM Operations | own RUN-* from validated producers |
| Incident | BSYSTEM Support | own INC-* |
| Cross-system relation | Integration Core | own relation metadata between Global IDs |
| RBAC/scopes | Integration Core | authoritative authorization metadata |
| Audit | Integration Core | append audit evidence |
| Notification state | Integration Core | own recipient/read state |
| Search index | Derived | never become source of truth |
| AI output | Derived | never become source of truth unless an explicit validated write is executed |

## Duplication policy

Allowed duplication must have a declared purpose:

- immutable mapping;
- authorization metadata;
- audit evidence;
- event/outbox state;
- bounded cache/projection;
- search index;
- operational history owned by BSYSTEM.

A cache/projection must define invalidation or freshness semantics. A derived copy must not silently become editable as an independent master.

## UI rule

A HUB form that edits source-owned data writes through the corresponding normalized Core capability to the authoritative system. The user should not need to know which upstream engine stores the record.

## Failure rule

If an authoritative source is unavailable, BSYSTEM must not silently accept a write into a local shadow copy and imply that the upstream record was changed.
