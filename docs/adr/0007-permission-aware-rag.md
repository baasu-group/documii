# ADR 0007: Permission-aware RAG

## Status
Accepted

## Context
Retrieved text can disclose data even if the final answer is filtered.

## Decision
Constrain keyword/vector candidates by the caller's authorization inside retrieval, before reranking or inference.

## Consequences
Chunk metadata and queries carry tenant/resource scope; authorization/retrieval integration receives dedicated security tests.

## Alternatives considered
Post-retrieval filtering and prompt-only controls are insecure and rejected.
