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
- Go 1.25+
- Python 3.12+ and `uv`
- Rust 1.77.2+ plus the [Tauri v2 system prerequisites](https://v2.tauri.app/start/prerequisites/) for native desktop builds
- Docker with Compose v2 for server dependencies

Only Node/pnpm is needed to run the browser shell. Root checks run every installed toolchain and CI runs all supported toolchains.

## First setup

### Run this checkout locally

For the local landing page and the checked-in browser build, use the lightweight
server below. This route needs Go, Python 3, and PostgreSQL 15 or newer. The
workspace source and pnpm manifests are also present for normal development.

Start PostgreSQL and create a database. For a temporary Homebrew PostgreSQL 17
instance on macOS (run once per new temporary cluster):

```sh
/opt/homebrew/opt/postgresql@17/bin/initdb -D /tmp/documii-postgres17 -A trust --no-instructions
/opt/homebrew/opt/postgresql@17/bin/pg_ctl -D /tmp/documii-postgres17 \
  -o '-h 127.0.0.1 -p 55432' -l /tmp/documii-postgres17.log start
```

On later runs, use only the `pg_ctl ... start` command. This temporary cluster
uses trust authentication and is bound to localhost; use it only for local
development. If you already have PostgreSQL 15+ running, use its connection URL
in the commands below instead.

Apply migrations and create the development workspace:

```sh
cd services/core
export DOCUMI_DATABASE_URL='postgres://localhost:55432/postgres?sslmode=disable'
GOCACHE=/tmp/documii-go-build go run ./cmd/migrate --dir ../../database/migrations
GOCACHE=/tmp/documii-go-build go run ./cmd/admin \
  --organization 'Local Documi' --workspace 'Local Workspace' \
  --issuer documi-development --subject local-admin \
  --email admin@documi.local --display-name 'Local administrator' --platform-admin
```

From the same `services/core` directory, start the API in one terminal:

```sh
GOCACHE=/tmp/documii-go-build \
  DOCUMI_AUTH_MODE=development \
  DOCUMI_HTTP_ADDRESS=127.0.0.1:8080 \
  go run ./cmd/api
```

In another terminal, serve the browser build and proxy its API requests:

```sh
cd ../..
python3 scripts/serve-built-web.py
```

Open <http://127.0.0.1:5173> for the public platform landing page. **Sign in**
opens <http://127.0.0.1:5173/login>, then continues to the existing app at
<http://127.0.0.1:5173/app>. The public pages are maintained in `landing/`;
the workspace uses the checked-in browser build.
The workspace's **Log out** button returns to the landing page in development
mode. Development authentication has no login session to revoke; an OIDC
deployment signs out through its identity provider.
Check the session with
`curl http://127.0.0.1:5173/api/v1/session-context`. Stop the API and web server
with `Ctrl+C`; stop the temporary database with
`/opt/homebrew/opt/postgresql@17/bin/pg_ctl -D /tmp/documii-postgres17 stop`.
The browser build is a snapshot and will not reflect source edits until a new
build is made.

### Full source setup

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

Never commit `.env`, credentials, customer data, model weights, or local databases. AI defaults to the disabled provider and no model is downloaded automatically. 
See [SECURITY.md](SECURITY.md) and [air-gap deployment guidance](deployments/airgap/README.md).

## Project status

The next milestone is the first thin production slice: authenticated upload → filesystem storage → immutable document/version metadata → append-only audit. Parsing, chunking, extraction, approval, and permission-aware search then extend that same record rather than forming isolated demos.

Documii is licensed under Apache License 2.0. 
See [CONTRIBUTING.md](CONTRIBUTING.md) and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
