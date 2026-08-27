# ADR 0008: Local versus server workspaces

## Status

Accepted

## Context

Desktop must work alone while company deployments require centralized governance.

## Decision

A workspace is explicitly Local or Server. Local uses on-device stores; Server uses the API and PostgreSQL. Automatic multi-master synchronization is deferred.

Local document metadata and immutable version history use bundled SQLite, while content uses application-controlled files. Server documents use PostgreSQL and the configured storage provider. Both implementations are reached through the shared document gateway contract; neither database replicates into the other.

## Consequences

Users choose a clear authority and desktop adapters support two modes without ambiguous merging.

## Alternatives considered

Requiring a server violates local use; implicit offline replication adds unacceptable bootstrap complexity.
