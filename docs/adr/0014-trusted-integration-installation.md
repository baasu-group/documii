# ADR 0014: Trusted integration installation

Status: accepted

## Decision

Documi integrations use a versioned manifest contract but first-party Go and Rust adapters remain compile-time registered. A manifest declares capabilities, deployment targets, safe configuration, write-only secrets, outbound network requirements, air-gap compatibility, and health behavior. Documi does not load arbitrary provider code.

Google Workspace is an optional external connector. LibreOffice Desktop is an explicitly installed local capability; Collabora remains the embedded browser Office editor. Secret material is encrypted through a SecretStore whose master key is supplied by deployment configuration and is never returned by APIs, logs, audit, or reports.

## Consequences

- `AIR_GAPPED=true` rejects Google installation and operations on the server, not merely in the UI.
- Organization and personal connections have separate ownership records.
- The LibreOffice extension receives only opaque working-copy identifiers and no Documi credential or storage path.
- Dynamic third-party code loading requires a future security ADR.
