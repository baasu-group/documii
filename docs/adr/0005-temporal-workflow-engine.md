# ADR 0005: Temporal workflow engine

## Status
Accepted

## Context
Approvals and document processing can wait, retry, and resume over long periods.

## Decision
Use Temporal for durable execution while keeping the public Documi workflow DSL independent.

## Consequences
Workers follow determinism and idempotency rules; Temporal is a server deployment dependency.

## Alternatives considered
Database-polled jobs cannot safely represent long-running human workflows; embedding Temporal's model as the public DSL would couple the product contract to one engine.
