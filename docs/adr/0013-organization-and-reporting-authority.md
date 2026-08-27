# ADR 0013: Organization and reporting authority

Status: accepted

## Decision

Documi owns organization status, membership, invitations, authorization grants, report runs, generated report documents, and audit history. OIDC and optional Keycloak Organizations authenticate or provision identities; neither supplies Documi document permissions.

Platform administration is granted only to an exact bootstrapped issuer/subject or an exact allowlisted OIDC group claim. Platform authority can read bounded document metadata but never document bytes. Titles remain hidden unless the organization explicitly enables title visibility, and drill-down is audited.

Organization suspension is enforced in the server authorization path. Reports are curated, bounded, and generated through fixed renderer adapters. Exported reports become ordinary managed Documi documents with immutable versions and normal grants.

## Consequences

- Credentials are issued per person and are never shared.
- Local and server membership authorities remain separate.
- PostgreSQL aggregate/read models are sufficient; no analytics database or arbitrary report builder is introduced.
- Last-owner protection and invitation acceptance must be transactional.
