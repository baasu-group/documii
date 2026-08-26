# ADR 0001: Go core server

## Status
Accepted

## Context
Documi needs a reliable, maintainable core for transactional business rules, HTTP APIs, and Temporal activities.

## Decision
Use one Go module for the core API, business worker, and migration executables.

## Consequences
Static binaries and strong concurrency/tooling are available; the team maintains a polyglot stack.

## Alternatives considered
TypeScript end-to-end and Python were rejected for the core because the selected operational and type-safety trade-offs favor Go.
