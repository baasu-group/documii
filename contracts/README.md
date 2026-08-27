# Cross-language contracts

Files here are authoritative. Go, Python, and TypeScript adapters must be generated from or validated against them as the API grows; handwritten types may temporarily mirror only the bootstrap endpoints and must be checked for drift. Schema changes follow compatibility review and explicit versioning.

`contracts/editors/` standardizes persisted editor IDs, capability/status responses, managed-versus-external editor metadata, and the versioned native Documi JSON envelope. Provider configuration and secrets are intentionally absent from public contracts.
