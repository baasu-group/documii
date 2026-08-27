# ADR 0012: Local and server document persistence

## Status

Accepted

## Context

Documi Desktop must support a local workspace without a server, while company workspaces require centralized PostgreSQL authorization, audit, and WOPI access. Automatic synchronization would create conflicting authorities and is explicitly deferred.

## Decision

Local and server workspaces implement the same document/version contracts through separate adapters. Local workspaces use embedded SQLite plus application-controlled files behind fixed Tauri commands. Server workspaces use the Go Documents service, PostgreSQL, and a configured StorageProvider. React selects a `DocumentGateway` from workspace mode and contains no persistence logic.

Every save supplies a base version and creates an immutable version. Local and server IDs and histories never merge automatically. Shared contract fixtures must exercise both adapters to limit semantic drift.

## Consequences

Desktop remains independently useful and server governance remains authoritative. Rust and Go have separate persistence adapters, but cross-language contracts and conformance tests define observable behavior. Any future import or synchronization feature must be explicit and receive its own ADR.

## Alternatives considered

A bundled local Go HTTP server adds operational complexity to the desktop application. Storing native content in database blobs contradicts the storage boundary. Treating SQLite as an offline PostgreSQL replica creates multi-master synchronization before the product has a safe conflict model.
