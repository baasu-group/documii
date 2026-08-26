# ADR 0006: AI provider abstraction

## Status
Accepted

## Context
Private and air-gapped deployments use different inference runtimes, and some disable AI.

## Decision
Define product-level AI provider protocols with adapters for local, compatible HTTP, custom, and disabled providers.

## Consequences
Capabilities and errors require normalization; no model weights or automatic downloads belong in the repository.

## Alternatives considered
Hard-coding Ollama or a public API was rejected because it would constrain deployment and licensing.
