# Tenant isolation acceptance matrix

What each actor must be able to reach in a real stage environment, and what
must be refused. This is the checklist an acceptance walks through.

Every cell is **derived from the platform's own configuration** — the seeded
role/permission grants in `003_rbac_scopes.sql` onward, and the permission and
resource scope declared per route in `cmd/server/routes.go` — not from an
assumption about what a role called "Manager" ought to do. Where the code and
this table disagree, the code is right and this document is a bug.

## Legend

| | |
| --- | --- |
| **ALLOW** | the actor reaches the resource |
| **DENY** | refused: `403` for a missing permission, `404` for an addressed resource the actor may not learn exists |
| **SCOPED** | permitted only inside an explicit scope grant; without a matching grant, collections are empty and detail reads are `404` |
| **N/A** | the surface does not exist for this actor |
| **BLOCKED** | cannot be accepted until the owner supplies authoritative data |

The distinction between DENY and SCOPED is the one that matters. DENY is a
property of the role. SCOPED is a property of the role *and* the data, and it
is the half that cannot be fully accepted yet.

## Permissions each role actually holds

| Role | Permissions |
| --- | --- |
| Administrator | `*` (every permission) |
| Manager | `crm.client.read`, `projects.task.read`, `qa.report.read`, `support.incident.read`, `operations.server.read`, `wiki.document.read` |
| Developer | `projects.task.read`, `projects.task.edit`, `qa.testcase.read`, `development.repo.read`, `development.pr.write`, `operations.server.read`, `wiki.document.read`, `wiki.document.edit` |
| QA | `projects.task.read`, `qa.testcase.read`, `qa.testcase.execute`, `qa.bug.write`, `wiki.document.read` |
| Support | `crm.client.read`, `projects.task.read`, `support.incident.read`, `support.incident.write`, `operations.server.read`, `wiki.document.read` |
| DevOps | `operations.server.read`, `operations.server.manage`, `development.repo.read`, `wiki.document.read`, `wiki.document.edit` |
| Customer | `portal.read`, `support.incident.read`, `wiki.document.read` — **and is scope-confined** |
| Service Core | `adapters.read`, `events.publish`, `global_ids.read`, `notifications.publish`, `operations.report`, `search.index` |
| unmapped user | none |

Customer is the only role in `DefaultConfinedRoles`. Every access it makes must
be justified by an explicit scope grant; holding the permission is not enough.

## Matrix — human API (`/api/v1/*`)

| Actor | clients | contacts | projects | issues | documents | servers | incidents | notifications | search | AI |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Administrator | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW |
| Manager | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW (read) | ALLOW | ALLOW | DENY |
| Developer | DENY | DENY | ALLOW | ALLOW | ALLOW | ALLOW | DENY | ALLOW | ALLOW | DENY |
| QA | DENY | DENY | ALLOW | ALLOW | ALLOW | DENY | DENY | ALLOW | ALLOW | DENY |
| Support | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW (read+write) | ALLOW | ALLOW | DENY |
| DevOps | DENY | DENY | DENY | DENY | ALLOW | ALLOW | DENY | ALLOW | ALLOW | DENY |
| Customer | **BLOCKED** | **BLOCKED** | DENY | DENY | SCOPED | DENY | **BLOCKED** | ALLOW | ALLOW | DENY |
| Service Core | N/A | N/A | N/A | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| unmapped user | DENY | DENY | DENY | DENY | DENY | DENY | DENY | ALLOW (empty) | ALLOW (empty) | DENY |

Three results in that table are worth reading twice.

**AI is DENY for every role including Manager.** `ai.query` is granted to no
role at all; only Administrator reaches it, through the `*` wildcard. That is
deliberate: who may spend money on a model is an owner decision, and the
migration says so in a comment rather than picking one. If an acceptance
expects a Manager to use the AI gateway, the grant has to be made first — as a
decision, not as a fix.

**Notifications and search require no permission.** They are filtered by
audience and by what the caller can already reach, not gated by a permission.
An unmapped user therefore gets `200` with an empty list rather than `403` —
correct, and worth confirming rather than reading as a hole.

**Customer rows for clients, contacts and incidents are BLOCKED, not DENY.**
Today they deny, because a scope-confined role with no grants denies. That is
the safe behaviour and it is testable now. What cannot be tested is the other
half — that a customer *does* see their own client and their own incidents —
because no authoritative mapping exists saying which client a customer owns.
Inventing one would mean inventing the answer to the question being accepted.

## Matrix — machine API (`/api/service/v1/*`)

| Actor | adapters | adapters/health | events | notifications | search/documents | servers | operations/events |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Service Core | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW |
| Administrator (human token) | DENY | DENY | DENY | DENY | DENY | DENY | DENY |
| every other human actor | DENY | DENY | DENY | DENY | DENY | DENY | DENY |
| unmapped / anonymous | DENY | DENY | DENY | DENY | DENY | DENY | DENY |

A human token never reaches the machine API, and an Administrator is no
exception. The two surfaces are separated by identity kind, not by permission,
so no amount of privilege crosses the line. Verify this rather than assuming
it: it is the property most likely to be quietly relaxed for convenience.

## Detail reads and existence disclosure

For every addressed resource (`/{id}`), a caller who may not see it gets
**`404`, not `403`**. The two are deliberately indistinguishable, so that
probing ids cannot be used to enumerate what exists.

Scope is evaluated **before** the resource is read, so a refusal cannot depend
on whether the record exists. A contract test asserts this ordering by reading
the handler source, and it was added after a real defect: `getServer` looked the
server up first, which made a refusal for a real id distinguishable from one for
an invented id — an enumeration oracle.

Resource scopes by route:

| Route | Scope kind |
| --- | --- |
| `/api/v1/clients/{id}` | client |
| `/api/v1/contacts/{id}` | resource |
| `/api/v1/projects/{id}` | project |
| `/api/v1/issues/{id}` | resource |
| `/api/v1/documents/{id}` | resource |
| `/api/v1/servers/{id}` | client |
| `/api/v1/incidents/{id}` | client |

## Acceptance procedure

For each actor, sign in as a user in exactly one BSYSTEM group and walk the row:

1. `GET /api/v1/me` — confirm the resolved role is the one expected. A wrong
   role here makes every later result meaningless.
2. For each ALLOW cell — the collection returns `200`.
3. For each DENY cell — the collection returns `403` naming the missing
   permission and nothing else. The body must not say whether any record exists.
4. For each SCOPED cell — without a grant, the collection is `200` **and
   empty**; a detail read of a known id is `404`.
5. For each detail read the actor may not see — `404`, never `403`.
6. With a service token, repeat step 2 against `/api/service/v1/*`, then
   confirm the same token is rejected on `/api/v1/*`.

An `X-Request-ID` on every call makes each result traceable in the audit trail,
which is the difference between "it denied me" and evidence.

## What stays BLOCKED, and why it cannot be worked around

The authoritative customer ownership mapping — which client each customer
account owns — does not exist in any system BSYSTEM can read. Until the owner
provides it:

- customer ALLOW cases cannot be accepted, only their DENY counterparts;
- no scope grant for a real customer can be written, because writing one means
  asserting an ownership fact;
- guessing from a name, an email domain or an account label would be inventing
  exactly the mapping the platform exists to get right.

Unknown ownership denies. That is the platform behaving correctly, not a defect
to route around, and it is why these cells say BLOCKED rather than DENY: the
behaviour is known, the acceptance is not yet possible.
