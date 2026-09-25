# Stage 1 Acceptance Criteria

## Production endpoints

- [ ] `QA_PUBLIC_URL` is configured with the intended PROD QA origin.
- [ ] `REDMINE_PUBLIC_URL` is configured with the intended PROD Redmine origin.
- [ ] `OUTLINE_PUBLIC_URL` is configured with the intended PROD Wiki/Outline origin.
- [ ] Documentation/example configuration uses only reserved domains such as `qa.example`, `redmine.example`, and `kb.example`.
- [ ] OIDC redirect/origin configuration uses the intended canonical deployment variables.
- [ ] Cross-system deep links do not point to DEV/STAGE hosts in PROD.

## Identity / SSO

- [ ] Redmine uses Authentik for ordinary interactive authentication.
- [ ] QA uses Authentik for ordinary interactive authentication.
- [ ] Outline uses Authentik for ordinary interactive authentication.
- [ ] Existing Authentik session is reused across applications subject to policy.
- [ ] Direct canonical application URLs work without BHUB.
- [ ] No BHUB UI is required.
- [ ] No BHUB token is passed to native applications.

## Identity preservation

- [ ] Existing Redmine user IDs remain valid.
- [ ] Existing QA user IDs remain valid.
- [ ] Existing Outline user IDs remain valid.
- [ ] Existing Redmine authorship/assignments/history remain intact.
- [ ] Existing QA authorship/history/project memberships remain intact.
- [ ] Existing Outline article authorship/revision history remain intact.
- [ ] Existing users are not duplicated by SSO when a controlled binding exists.

## Native authorization

- [ ] Redmine roles/projects still determine Redmine permissions.
- [ ] QA roles/project memberships still determine QA permissions.
- [ ] Outline groups/collections still determine Outline permissions.
- [ ] Authentik is not used as a replacement for application-specific fine-grained RBAC.

## QA SSO migration

- [ ] QA maps stable Authentik `sub` to existing local user.
- [ ] Existing `requireUser()` / native backend permission checks remain effective.
- [ ] Local password UI is retired for ordinary users after cutover.
- [ ] Controlled rollback remains possible during acceptance window.

## Cross-system relationships

- [ ] Integration Core is authoritative for cross-system relations.
- [ ] Redmine issue can show related QA/Outline objects.
- [ ] QA object can show related Redmine/Outline objects.
- [ ] Outline document can show related Redmine/QA objects where supported.
- [ ] Relation can be created.
- [ ] Relation can be removed.
- [ ] Removing relation never removes source objects.
- [ ] Deep links open the exact target object.

## Legacy link migration

- [ ] Existing QA↔Redmine links are inventoried.
- [ ] Migrated relation count matches source count or all discrepancies are documented.
- [ ] Representative migrated relations resolve correctly.
- [ ] Legacy tables are no longer authoritative after cutover.
- [ ] Rollback path is documented.

## Security

- [ ] Cross-system search does not reveal unauthorized target objects.
- [ ] Relationship existence does not grant target-object access.
- [ ] Service credentials use least privilege.
- [ ] Secrets are not committed.
- [ ] Identity binding changes are audited.
- [ ] Relationship creation/removal is audited.

## Resilience

- [ ] Integration Core outage does not make Redmine unusable.
- [ ] Integration Core outage does not make QA unusable.
- [ ] Integration Core outage does not make Outline unusable.
- [ ] Native related-object UI shows a clear degraded state.

## Evidence

Stage 1 MUST NOT be declared complete without:

- exact deployed SHAs;
- exact configuration version;
- test results;
- migration verification;
- rollback evidence;
- unresolved-risk list.
