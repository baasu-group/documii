# Security policy

Documi is pre-release. Do not use the bootstrap with production or sensitive data.

## Reporting a vulnerability

Do not open a public issue for a suspected vulnerability. Use the repository host's private security advisory feature and include affected revision, impact, reproduction steps, and any suggested mitigation. Maintainers will acknowledge a complete report as soon as practical and coordinate disclosure after a fix is available.

## Security invariants

- Authorization constrains search/RAG before retrieval.
- Document versions are immutable records; audit history is append-only at application level.
- Secrets, tokens, model weights, customer data, and private keys are never committed or logged.
- AI and outbound integrations are optional; air-gapped mode has no required public dependency.
- Filesystem keys must remain under the configured storage root.

Development passwords in `.env.example` and Compose are unsafe outside local development.
