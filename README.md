# Documi

Documi is an open-source, self-hosted business operations platform for versioned documents, structured data, forms, human approvals, durable workflows, integrations, auditing, and permission-aware private AI.

This repository is an **architecture bootstrap**, not a production release. It proves the monorepo boundaries, shared web/desktop shell, core health API, worker composition roots, configuration, deployment dependencies, contracts, and quality workflow. Start with [ARCHITECTURE.md](ARCHITECTURE.md) before changing a subsystem.

## Repository at a glance

```text
apps/             React/Vite web and Tauri v2 desktop bootstraps
packages/         shared application shell, UI, contracts-facing TypeScript packages
services/core/    Go API, Temporal worker, and migration utility
services/ai/      uv-managed Python document and AI worker project
contracts/        authoritative OpenAPI and JSON Schema contracts
database/         PostgreSQL migration convention
deployments/      Compose, Keycloak, Temporal, air-gap, and future Kubernetes guidance
docs/             ADRs and engineering documentation
```

## Prerequisites

- Node.js 22+ and pnpm 10+
- Go 1.23+
- Python 3.12+ and `uv`
- Rust 1.77.2+ plus the [Tauri v2 system prerequisites](https://v2.tauri.app/start/prerequisites/) for native desktop builds
- Docker with Compose v2 for server dependencies

Only Node/pnpm is needed to run the browser shell. Root checks run every installed toolchain and CI runs all supported toolchains.

## First setup

```sh
cp .env.example .env
pnpm bootstrap
pnpm dev
```

Open `http://localhost:5173`. The browser app renders product navigation for Home, Documents, Data, Forms, Tasks, Automations, AI, Integrations, and Settings.

## Development infrastructure

```sh
pnpm infra:up
```

This starts PostgreSQL with pgvector (`localhost:5432`), Temporal (`localhost:7233`), Temporal UI (`http://localhost:8082`), and Keycloak (`http://localhost:8081`). Values in `.env.example` are local-only placeholders.

Run processes in separate terminals:

```sh
pnpm dev:api
pnpm dev:core-worker
pnpm dev:document-worker
pnpm dev:ai-worker
```

The API exposes:

```text
GET http://localhost:8080/health
GET http://localhost:8080/ready
GET http://localhost:8080/api/v1/system/info
```

Run the desktop shell after installing Tauri platform prerequisites:

```sh
pnpm dev:desktop
```

## Quality commands

```sh
pnpm format
pnpm lint
pnpm typecheck
pnpm test
pnpm build
pnpm check
```

The scripts are intentionally visible under `scripts/`. Missing optional toolchains are reported locally rather than hiding what was skipped; CI provisions them all.

## Security and air-gapped operation

Never commit `.env`, credentials, customer data, model weights, or local databases. AI defaults to the disabled provider and no model is downloaded automatically. See [SECURITY.md](SECURITY.md) and [air-gap deployment guidance](deployments/airgap/README.md).

## Project status

The next milestone is the first thin production slice: authenticated upload → filesystem storage → immutable document/version metadata → append-only audit. Parsing, chunking, extraction, approval, and permission-aware search then extend that same record rather than forming isolated demos.

Documi is licensed under Apache License 2.0. See [CONTRIBUTING.md](CONTRIBUTING.md) and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
