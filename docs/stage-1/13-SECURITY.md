# Stage 1 Security Model

## 1. Security boundaries

### Identity boundary
Authentik authenticates the person.

### Application authorization boundary
The native application authorizes its own resources.

### Integration boundary
Integration Core stores and serves relationship data under least privilege.

## 2. Primary risk: metadata disclosure

A shared service credential can often see more than an individual user.

Therefore this is unsafe by default:

```text
QA user
 -> Integration Core
 -> shared Redmine service account
 -> return all matching Redmine issues
```

If the service account can see more than the user, search results may leak protected issue existence/title/status/project.

## 3. Stage 1 safe discovery rule

Until delegated user-aware authorization is implemented, use the least revealing interaction.

Preferred MVP:

- exact object ID;
- pasted canonical URL;
- object validation;
- deep link;
- native application performs final access check.

Broad federated search is optional and must be authorization-safe.

## 4. Service identities

Use dedicated machine identities such as:

```text
SVC-redmine
SVC-qa
SVC-outline
```

Never use application admin credentials for ordinary integration.

## 5. Least privilege

Adapter/service accounts should receive only the permissions required for the integration.

## 6. Secrets

- no secrets committed to Git;
- no API keys in URLs;
- no access tokens in query strings/fragments;
- no plaintext application credentials in frontend code;
- production secrets provided through deployment secret mechanisms.

## 7. OIDC

- exact redirect URIs;
- HTTPS;
- issuer validation;
- audience validation where applicable;
- expiry validation;
- stable `sub`;
- secure cookie settings;
- CSRF/state/nonce protections appropriate to the library/framework.

## 8. Identity binding

Binding an Authentik subject to an existing local account is security-sensitive.

Requirements:

- unique binding;
- audit;
- administrator confirmation for ambiguous mappings;
- no silent takeover based only on weak username similarity.

## 9. Cross-system relationship permissions

Create/remove relation permissions must be defined.

At minimum:

- actor must be authenticated;
- actor must be allowed to operate in the source application;
- target object metadata must not be disclosed beyond authorization;
- operation must be audited.

## 10. Failure behavior

Security-sensitive failures should fail closed.

Examples:

- cannot verify identity -> deny;
- cannot safely verify target metadata visibility -> do not disclose metadata;
- relationship service unavailable -> native application continues without relationship functionality.