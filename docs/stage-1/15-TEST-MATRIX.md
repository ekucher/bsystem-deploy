# Stage 1 Test Matrix

## 1. Identity

| ID | Test | Expected |
|---|---|---|
| ID-01 | Existing Redmine user signs in through Authentik | Same Redmine user reused |
| ID-02 | Existing QA user signs in through Authentik | Same `users.id` reused |
| ID-03 | Existing Outline user signs in through Authentik | Same local user/history reused |
| ID-04 | Ambiguous account match | Login/provisioning blocked pending controlled resolution |
| ID-05 | Disabled Authentik account | New authentication denied |
| ID-06 | Existing valid app session after IdP interruption | Behavior matches documented app/session policy |
| ID-07 | Login to second app with live Authentik session | No credential re-entry subject to policy |

## 2. Authorization

| ID | Test | Expected |
|---|---|---|
| AUTHZ-01 | QA user outside project | Project data unavailable |
| AUTHZ-02 | Redmine user without project permission | Target issue denied by Redmine |
| AUTHZ-03 | Outline user outside collection | Document denied by Outline |
| AUTHZ-04 | Cross-link exists to inaccessible target | Link does not grant access |
| AUTHZ-05 | Search using shared integration credential | Must not reveal objects user cannot access |

## 3. Relationships

| ID | Test | Expected |
|---|---|---|
| REL-01 | Create QA Test Case -> Redmine Issue relation | Stored once in Integration Core |
| REL-02 | View relation from QA | Correct inverse/presentation |
| REL-03 | View relation from Redmine | Same relationship visible |
| REL-04 | Delete relation | Only relationship deleted |
| REL-05 | Target app unavailable | Panel degrades, native page remains usable |
| REL-06 | Integration Core unavailable | Native domain work remains usable |
| REL-07 | Duplicate relationship create | Idempotent or deterministic conflict |
| REL-08 | Invalid target ID | Safe validation error |

## 4. Migration

| ID | Test | Expected |
|---|---|---|
| MIG-01 | Export legacy QA↔Redmine links | Complete inventory |
| MIG-02 | Import to Core | Counts match expected |
| MIG-03 | Random sample resolution | Correct source and target objects |
| MIG-04 | QA reads switched to Core | UI shows existing links |
| MIG-05 | Legacy write attempt after cutover | Blocked/disabled |
| MIG-06 | Rollback | Previous read path can be restored during acceptance window |

## 5. Logout

| ID | Test | Expected |
|---|---|---|
| LOGOUT-01 | QA local logout | QA session ends; central session preserved unless explicitly central logout |
| LOGOUT-02 | Outline local logout | Outline session ends; Authentik session preserved |
| LOGOUT-03 | Redmine local logout | Behavior documented and does not unexpectedly destroy ecosystem session |
| LOGOUT-04 | Central logout | New app entry requires reauthentication |

## 6. Audit

| ID | Test | Expected |
|---|---|---|
| AUD-01 | Bind identity | Actor, target local user, Authentik subject recorded |
| AUD-02 | Create relationship | Actor/source/target/type/timestamp recorded |
| AUD-03 | Remove relationship | Audit record retained |
| AUD-04 | Failed privileged action | Failure is logged without secrets |

## 7. Security negatives

| ID | Test | Expected |
|---|---|---|
| SEC-01 | No auth | Protected API denied |
| SEC-02 | Expired/invalid OIDC token | Denied |
| SEC-03 | Wrong issuer/audience | Denied |
| SEC-04 | User attempts to bind to another person's local account | Denied/audited |
| SEC-05 | Search for unauthorized Redmine issue | No protected metadata disclosure |
| SEC-06 | Relationship to inaccessible Outline doc | No permission escalation |


## 8. Integration Core relationship store

| ID | Test | Expected |
|---|---|---|
| CORE-01 | Register same `(source,type,source_id)` concurrently | One stable Global ID returned |
| CORE-02 | Allocate QA Requirement | `REQ-*` allocated once |
| CORE-03 | QA task and Redmine issue both use task family | Distinct mappings preserved by source/source_id |
| CORE-04 | Create same relation twice | Idempotent result or deterministic conflict, no duplicate edge |
| CORE-05 | Create self relation | Refused |
| CORE-06 | Read relationship | Does not allocate missing object IDs |
| CORE-07 | Inverse relation read | Derived correctly from one stored edge |
| CORE-08 | Mutation audit failure | Mutation follows documented fail-closed policy |
| CORE-09 | Browser spoofs actor assertion | Refused/ignored |
| CORE-10 | Service asserts actor | Both service and user actor preserved in audit |
| CORE-11 | Service lacks `relationships.write` | Refused |
| CORE-12 | Core relationship outage | Native application domain function remains available |