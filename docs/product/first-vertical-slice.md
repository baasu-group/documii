# First production vertical slice

Implement the invoice journey in thin, demonstrable increments:

1. OIDC-authenticated workspace request context.
2. Stream upload to a filesystem storage provider while hashing.
3. Transactionally create document, immutable version metadata, and audit event.
4. Commit an outbox event and start a version-pinned Temporal ingestion workflow.
5. Parse/OCR/normalize, then persist provenance-rich chunks and embeddings.
6. Extract a typed invoice record with evidence and create a manager approval task.
7. Approve/reject durably, update record state, archive through storage policy, and audit each decision.
8. Query only the caller's authorized candidate set and return version/page citations.

The recommended next change implements steps 1–3 with negative tenant/path tests. Do not build extraction before source integrity and authorization are real.
