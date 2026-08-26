# ADR 0002: PostgreSQL is the server source of truth

## Status
Accepted

## Context
Business metadata, authorization, audit, workflow references, and vectors need consistent server-side persistence.

## Decision
Use PostgreSQL as canonical server storage and pgvector for embeddings.

## Consequences
Transactions can cover metadata and outbox changes; PostgreSQL operation is required in server mode. No separate vector database is introduced.

## Alternatives considered
MySQL lacks the selected vector path; a separate vector store adds consistency and authorization risk; SQLite is reserved for local workspaces.
