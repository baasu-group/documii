# ADR 0010: Polyglot monorepo

## Status
Accepted

## Context
The product benefits from TypeScript UI tooling, Go core services, Python document/AI libraries, and Rust desktop integration.

## Decision
Keep all runtimes, contracts, deployments, and documentation in one repository with language-native package management.

## Consequences
Cross-language changes and CI stay coordinated; contributors need multiple toolchains and explicit contract ownership.

## Alternatives considered
Separate repositories increase drift; forcing one language sacrifices key ecosystem advantages.
