# Documi agent guide

## Before changing code

1. Read `ARCHITECTURE.md` before making architectural changes.
2. Read the ADRs relevant to the subsystem being changed.
3. Check for a more specific `AGENTS.md` in the directory you are editing.
4. Inspect the working tree and preserve unrelated user changes.

## Architectural guardrails

- Preserve domain module boundaries. Keep core business logic out of Go HTTP handlers.
- Prefer the simplest architecture consistent with the product vision.
- Do not create a microservice without an accepted ADR.
- Do not add an infrastructure dependency without showing why PostgreSQL, the filesystem, Temporal, or existing infrastructure cannot meet the requirement.
- Never retrieve unauthorized chunks and filter afterward. Authorization constrains the RAG/search candidate set before retrieval.
- Do not add required public-cloud dependencies. Air-gapped operation must remain possible.
- Keep product logic reusable between web and desktop; platform behavior belongs behind an interface.
- Keep AI providers and storage providers replaceable at their owning domain boundary.
- Treat audit events as append-only at the application level.
- Keep cross-language contracts in `contracts/`; do not maintain contradictory handwritten copies.
- Never commit secrets, customer data, model weights, generated build output, or local databases.
- Tests are required for business-critical behavior and authorization boundaries.

## When architecture changes

Update `ARCHITECTURE.md` and create or amend an ADR in the same change. Do not silently contradict the documented architecture. Run the relevant root checks before handing work off.
