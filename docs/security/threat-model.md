# Bootstrap threat model

Primary protected assets are document bytes, metadata, structured records, identities/ACLs, secrets, audit history, embeddings, and retrieved content. Principal boundaries are browser/desktop → API, API → identity/database/storage/Temporal, and workers → storage/database/AI provider.

Early high-risk cases are cross-organization object access, filesystem path traversal, confused workspace context, retrieved confidential chunks, prompt injection reaching privileged tools, secret leakage through logs/events, and mutable document/audit history. The first data slice must add explicit negative tests for each boundary it activates. TLS termination, production OIDC, encrypted secrets, malware scanning, rate limiting, and backup/restore procedures remain release blockers, not bootstrap claims.
