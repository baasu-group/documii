# ADR 0004: Storage provider abstraction

## Status
Accepted

## Context
Deployments require local and mounted filesystems, with possible S3-compatible storage later.

## Decision
The storage domain owns a small Put/Get/Delete/Move/Exists interface and validates opaque keys.

## Consequences
Providers are replaceable and testable; capability differences must be expressed explicitly.

## Alternatives considered
Direct filesystem calls throughout domain code and an SMB client were rejected.
