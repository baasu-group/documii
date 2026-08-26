# Contributing to Documi

Read `ARCHITECTURE.md`, root `AGENTS.md`, and relevant ADRs before changing a subsystem. Discuss architecture-changing work in an issue and capture accepted decisions in an ADR.

## Development workflow

1. Create a focused branch and preserve domain boundaries.
2. Add or update tests with behavior.
3. Run `pnpm check` with all relevant toolchains installed.
4. Update contracts and documentation in the same change when behavior crosses runtimes.
5. Explain security, migration, compatibility, and air-gap impact in the pull request.

Commits must not contain secrets, customer data, model weights, generated build output, or dependencies. Use synthetic fixtures with clear provenance. By participating, you agree to the Code of Conduct.
