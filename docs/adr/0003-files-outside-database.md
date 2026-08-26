# ADR 0003: Store file bytes outside the database

## Status
Accepted

## Context
Documents can be large and must work with local disks and company-mounted storage.

## Decision
Store document metadata in PostgreSQL and immutable version bytes through a storage provider.

## Consequences
Backup procedures must coordinate two stores; database size and file streaming remain manageable.

## Alternatives considered
PostgreSQL byte columns were rejected as the default; custom distributed storage is unnecessary.
