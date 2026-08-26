# Development tooling

Root commands coordinate native package managers rather than replacing them. pnpm owns JavaScript dependencies and workspace ordering; Go modules own the core; uv owns Python resolution and environments; Cargo owns Tauri native dependencies.

`pnpm bootstrap` installs the lockfile and prepares any installed backend toolchains. `pnpm check` runs formatting checks, lint, type checks, tests, and builds. Scripts report a missing optional tool locally, while CI installs all three main toolchains. Use Compose only for integration and runtime dependencies; unit tests should remain fast and hermetic.
