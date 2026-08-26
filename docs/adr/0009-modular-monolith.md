# ADR 0009: Domain-oriented modular monolith

## Status
Accepted

## Context
Documi has many domains but initially one team and tightly related transactions.

## Decision
Organize one Go application by domain modules with explicit internal boundaries. New services require an ADR and concrete isolation need.

## Consequences
Deployment and transactions remain simple; module discipline is enforced through package ownership and tests.

## Alternatives considered
One microservice per domain and repository-wide handler/service/repository layers were rejected.
