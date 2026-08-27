# Running Documi locally

This guide covers the current Documi bootstrap on a local machine: installation, web-only testing, the Go API, PostgreSQL/pgvector, Temporal, Keycloak, Python workers, the Tauri desktop shell, verification, shutdown, and common errors.

> **Current product status:** local Tauri workspaces and PostgreSQL server workspaces have separate native document gateways with immutable versions. Server authentication, hierarchy-aware grants, audit, validated Office uploads, and Collabora session/WOPI persistence are composed. Collabora remains partial until blank Office templates and a real-container browser test are completed.

## 1. Choose what you want to test

| Test level | Required tools | What works |
| --- | --- | --- |
| Web shell only | Node.js and pnpm | React UI and all initial navigation areas |
| API only | Go | `/health`, `/ready`, and `/api/v1/system/info` |
| Automated checks | Node.js, pnpm, Go, Python, uv | Formatting, linting, tests, type checks, and builds |
| Server infrastructure | Docker Compose | PostgreSQL/pgvector, Temporal, Temporal UI, and Keycloak |
| Full bootstrap | All of the above | UI, API, infrastructure, Go worker, and Python workers |
| Desktop shell | All frontend tools, Rust, and native Tauri prerequisites | Native window using the shared React application |

For the fastest first test, use **Web-only quick start**. Docker, Go, Python, Rust, and AI models are not required for that path.

## 2. Prerequisites

Recommended versions:

- Node.js 22 LTS or newer supported LTS release
- pnpm 10 or newer
- Go 1.25 or newer
- Python 3.12 or newer
- uv
- Docker Desktop or Docker Engine with Compose v2
- Rust 1.77.2 or newer for the desktop application only

Check what is installed:

```sh
node --version
pnpm --version
go version
python3 --version
uv --version
docker --version
docker compose version
rustc --version
cargo --version
```

Missing Go, Python/uv, Docker, or Rust does not prevent web-only testing.

### macOS installation notes

Install Node.js using your preferred version manager or Homebrew, then enable/install pnpm. Install the remaining optional tools only when you need those runtimes:

```sh
brew install node pnpm go uv
brew install --cask docker
```

For Tauri, install Apple's command-line tools and Rust through `rustup`:

```sh
xcode-select --install
```

Then follow the Rust installer at `https://rustup.rs/`. The full Tauri prerequisites are documented at `https://v2.tauri.app/start/prerequisites/`.

Avoid installing the full Homebrew `rust` formula on a nearly full disk: it can pull a large LLVM dependency. The minimal `rustup` profile is generally smaller.

### Windows notes

Use WSL2 or Git Bash for repository root scripts because they currently call POSIX `sh` scripts. Docker Desktop should be running with WSL integration if you use WSL. Building the native Windows Tauri application also requires Rust and Microsoft C++ Build Tools/WebView2 as described in the Tauri prerequisites.

### Linux notes

Install Node.js, pnpm, Go, uv, and Docker using your distribution or official installers. A Tauri native build additionally needs the WebKitGTK and system development packages listed by Tauri for your distribution.

## 3. Initial repository setup

From a terminal, enter the repository:

```sh
cd /Users/basantashrestha/Development/documii/documii
```

Create local configuration:

```sh
cp .env.example .env
```

The values in `.env.example` are safe development placeholders. Do not use its passwords in a public or production deployment, and never commit `.env`.

Install locked dependencies:

```sh
pnpm bootstrap
```

`pnpm bootstrap` performs:

1. `pnpm install --frozen-lockfile` for all TypeScript workspaces.
2. `go mod download` when Go is installed.
3. `uv sync --frozen` when uv is installed.

Expected final output includes:

```text
bootstrap: complete for installed toolchains
```

If Go or uv is not installed, bootstrap reports that it skipped that optional toolchain. That is acceptable for web-only testing.

## 4. Web-only quick start

Run:

```sh
pnpm dev
```

Expected output includes a URL similar to:

```text
Local: http://localhost:5173/
```

Open `http://localhost:5173` in a browser. You should see the Documi shell with:

- Home
- Documents
- Data
- Forms
- Tasks
- Automations
- AI
- Integrations
- Settings

Click each navigation item and verify that its placeholder page loads. Stop the server with `Ctrl+C`.

The web UI can render the shared document workflow, but its server gateway is not persistence-backed yet. Use the Tauri path for durable local documents.

### Use a different web port

If port 5173 is already in use:

```sh
pnpm --filter @documi/web dev -- --port 5174
```

Then open `http://localhost:5174`.

## 5. Run and test the Go API without Docker

The health, system, auth-configuration, and editor-availability endpoints can start without PostgreSQL. Server workspace/document routes are registered only when `DOCUMI_DATABASE_URL` is configured and reachable.

Copying `.env` does **not** automatically export it into local Go or Python processes. On macOS/Linux, load it into the current terminal first:

```sh
set -a
. ./.env
set +a
```

Then run:

```sh
pnpm dev:api
```

The API listens on `http://localhost:8080` by default. In a second terminal, test it:

```sh
curl -fsS http://localhost:8080/health
curl -fsS http://localhost:8080/ready
curl -fsS http://localhost:8080/api/v1/system/info
```

Expected responses:

```json
{"status":"ok"}
{"status":"ready"}
{"name":"Documi","version":"dev","environment":"development","airGapped":false}
```

Stop the API with `Ctrl+C`. A graceful stop writes an `api stopped` structured log entry.

### Test air-gapped configuration

```sh
AIR_GAPPED=true DOCUMI_HTTP_ADDRESS=:8080 pnpm dev:api
```

The system-info response should then contain `"airGapped":true`.

## 6. Start server infrastructure with Docker

Make sure Docker Desktop or the Docker daemon is running, then start the default infrastructure profile:

```sh
pnpm infra:up
```

This starts:

| Service | Address | Purpose |
| --- | --- | --- |
| PostgreSQL + pgvector | `localhost:5432` | Canonical server database |
| Temporal | `localhost:7233` | Durable workflow engine |
| Temporal UI | `http://localhost:8082` | Workflow inspection |
| Keycloak | `http://localhost:8081` | Development OIDC provider |

Check container state:

```sh
docker compose -f deployments/compose/docker-compose.yml ps
```

Wait until PostgreSQL and Temporal are healthy. Inspect logs when necessary:

```sh
docker compose -f deployments/compose/docker-compose.yml logs -f postgres
docker compose -f deployments/compose/docker-compose.yml logs -f temporal
docker compose -f deployments/compose/docker-compose.yml logs -f keycloak
```

Press `Ctrl+C` to stop following logs; this does not stop the containers.

### Keycloak development login

Open `http://localhost:8081`. With the unchanged `.env.example` development values, the administrative login is:

```text
Username: admin
Password: admin_dev_only
```

These credentials are for local development only. The imported `documi` realm provides the public `documi-web` client but deliberately contains no test users. The API protects `/api/v1/me`, document routes, and editor-session creation. Development mode supplies the configured development subject; OIDC mode validates issuer, signature, audience, expiry, and subject. PostgreSQL identity persistence and document grant enforcement are composed whenever `DOCUMI_DATABASE_URL` is configured and migrations are current.

## 7. Apply the bootstrap database migration

First load `.env` into the terminal:

```sh
set -a
. ./.env
set +a
```

After PostgreSQL is healthy, run:

```sh
cd services/core
go run ./cmd/migrate --dir ../../database/migrations
cd ../..
```

Expected output:

```text
database migrations are current
```

Migration `000001` enables `pgvector`; `000002` creates identity, organization/workspace/folder, additive grants, storage, document/version, immutable audit, idempotency, editor-session, and WOPI-lock tables. With `DOCUMI_DATABASE_URL` set, the API composes the server document gateway and maps validated identities to internal users.

You can inspect the database through the container:

```sh
docker compose -f deployments/compose/docker-compose.yml exec postgres \
  psql -U documi -d documi -c '\dx'

docker compose -f deployments/compose/docker-compose.yml exec postgres \
  psql -U documi -d documi -c '\dt'
```

## 8. Run the full bootstrap in separate terminals

Keep Docker infrastructure running, then open separate terminals in the repository root. Load `.env` in every terminal that runs a local Go or Python process:

```sh
set -a
. ./.env
set +a
```

Terminal 1 — API:

```sh
pnpm dev:api
```

Terminal 2 — Go Temporal worker:

```sh
pnpm dev:core-worker
```

Expected output includes:

```text
Temporal worker starting
```

Terminal 3 — document worker:

```sh
pnpm dev:document-worker
```

Terminal 4 — AI worker:

```sh
pnpm dev:ai-worker
```

The Python workers currently validate configuration, report startup, and wait for shutdown. This warning is expected:

```text
bootstrap worker is idle; Temporal activity polling is introduced with the first pipeline
```

Terminal 5 — web application:

```sh
pnpm dev
```

Open these pages:

- Documi web: `http://localhost:5173`
- API health: `http://localhost:8080/health`
- Temporal UI: `http://localhost:8082`
- Keycloak: `http://localhost:8081`

Stop local processes with `Ctrl+C` in each terminal.

## 9. Optional: run application processes in Compose

The default `pnpm infra:up` command starts infrastructure only. To build and run the Go API and Go worker containers too:

```sh
docker compose -f deployments/compose/docker-compose.yml \
  --profile application up --build
```

The containerized API is available at `http://localhost:8080`. Stop the foreground command with `Ctrl+C`, then remove its containers with the normal shutdown command in the next section.

### Optional Collabora foundation

Start the API and Collabora profiles together:

```sh
docker compose -f deployments/compose/docker-compose.yml \
  --profile application --profile collabora up --build
```

Verify the Collabora service itself:

```sh
curl -fsS http://localhost:9980/hosting/discovery
```

The API reports Collabora available only when it can reach discovery and the PostgreSQL document/storage gateway is ready. Supported Office/OpenDocument uploads can then receive persisted sessions and WOPI locks/saves. Blank Office template creation is not implemented yet, so this is not a complete browser acceptance test.

### Bootstrap the first server owner

After applying migrations, load `.env`, then run:

```sh
cd services/core
go run ./cmd/admin \
  --organization "Example Company" \
  --workspace "Main Workspace" \
  --issuer "http://localhost:8081/realms/documi" \
  --subject "REPLACE_WITH_OIDC_SUBJECT" \
  --email "owner@example.test" \
  --display-name "Example Owner"
cd ../..
```

This is idempotent for the issuer/subject and names and never guesses the first user. After bootstrap, that identity can use the PostgreSQL-backed server document routes.

For `DOCUMI_AUTH_MODE=development`, use `--issuer documi-development` and set `--subject` to `DOCUMI_DEV_SUBJECT` (default `local-admin`). Development auth is rejected outside development/test environments.

### Use PostgreSQL on another private-network server

PostgreSQL is optional for a purely local desktop workspace. For a company workspace it may run on another LAN host; only the API, migration, and admin processes receive its URL:

```sh
DOCUMI_DATABASE_URL='postgres://documi:REPLACE@db.internal.example:5432/documi?sslmode=verify-full'
DOCUMI_STORAGE_ROOT='/srv/documi/documents'
```

On Windows Server, `DOCUMI_STORAGE_ROOT` can be a drive or a company-approved mounted SMB path. On Linux it can be an NFS/SMB mount such as `/mnt/documi`; on macOS development it can be a mounted volume. Mount and authenticate the share through the operating system—do not put SMB credentials in document metadata.

Requirements:

1. Install a supported PostgreSQL version plus pgvector on the database host.
2. Permit port 5432 only from API/migration/admin hosts.
3. Require TLS outside an isolated development machine.
4. Run migrations from one controlled release process.
5. Run the API near the database/storage; clients connect only to API HTTPS.
6. Verify same-directory atomic rename, permissions, backups, and restore on any network share.

PowerShell can set process-local values as follows:

```powershell
$env:DOCUMI_DATABASE_URL = "postgres://documi:REPLACE@db.internal.example:5432/documi?sslmode=verify-full"
$env:DOCUMI_STORAGE_ROOT = "D:\Documi\Documents"
go run ./cmd/api
```

See [deployment topologies](docs/architecture/deployment-topologies.md) for the authority and platform boundaries.

## 10. Run the desktop application

First confirm the native environment:

```sh
pnpm --filter @documi/desktop tauri info
```

The output should show installed versions of `rustc`, Cargo, and the platform webview/native build tools. Then run:

```sh
pnpm dev:desktop
```

Tauri starts the Vite desktop frontend at port 1420 and opens the native Documi window. Open **Documents**, create a Documi document, edit it, and choose the explicit save action. Close and restart the application: the document and current version should still appear. Metadata lives in `documi.sqlite3`; content lives under `documents/<workspace-id>/<document-id>/<version-id>/document.documi.json` inside the platform application-data directory. React cannot choose or receive those paths.

To verify only the desktop frontend without Rust:

```sh
pnpm --filter @documi/desktop build
```

This validates TypeScript and builds the desktop web assets, but it does not compile the Rust native application or test SQLite. Run `cargo test`, `cargo fmt --check`, and `cargo clippy -- -D warnings` from `apps/desktop/src-tauri` before accepting native changes.

## 11. Run tests and quality checks

Run the complete repository check:

```sh
pnpm check
```

It runs, in order:

1. Prettier, gofmt, and Ruff formatting checks.
2. ESLint, `go vet`, and Ruff linting.
3. TypeScript and Go compilation/type checks.
4. TypeScript, Go, and Python tests.
5. Web/desktop frontend, Go, and Python worker builds/smoke checks.

You can also run individual stages:

```sh
pnpm format
pnpm format:check
pnpm lint
pnpm typecheck
pnpm test
pnpm build
```

Important: root scripts report and skip an unavailable Go or uv toolchain. Read the output; a zero exit code with a “skipped” message is not verification of that runtime. CI provisions all supported toolchains.

Expected test coverage at bootstrap includes:

- Initial navigation areas.
- Web runtime smoke test.
- Go configuration parsing.
- Go liveness/readiness/system-info handlers.
- Document validation, storage containment, permission, version, and WOPI protocol behavior.
- Temporal development activity wiring.
- Python health and worker startup.
- Disabled AI provider failing closed.

### Verification recorded for this implementation

On 2026-08-27 the following completed successfully in this checkout:

```sh
pnpm check
cd services/core && go test ./...
cd services/core && go test -race ./internal/modules/documents/editors/collabora ./internal/modules/documents
```

The API was also started on `127.0.0.1:18080`; `/health`, `/ready`, `/api/v1/auth/config`, `/api/v1/me`, and `/api/v1/document-editors` returned successful JSON responses. Against an isolated PostgreSQL 17 cluster, owner bootstrap, authorized workspace listing, native create/save/history, stale-save conflict, storage/version counts, audit creation, and a second user's non-enumerating denial were exercised. The processes and isolated data were then stopped and removed.

Two environment limitations prevented complete runtime verification on this machine:

- `cargo`/`rustfmt` were not installed, so the Tauri Rust module, bundled SQLite migration, and restart-persistence flow were not compiled or executed here.
- `docker` was not installed, so Keycloak/OIDC and Collabora were not executed against their real containers. Migration `000002` and the native server authorization/version flow were instead validated with an isolated Homebrew PostgreSQL 17 cluster; migration `000001` requires pgvector and correctly failed on that stock server.

The repository-wide `go test -race ./...` build was also attempted, but the machine's nearly full disk exhausted the Go temporary build volume. The two changed persistence/WOPI packages passed with race instrumentation using the narrower command above; the complete race suite remains a CI or higher-capacity-machine check.

Treat those as required checks on a machine with the corresponding prerequisites; do not infer success from the frontend-only desktop build.

## 12. Stop and clean up

Stop foreground web/API/worker processes with `Ctrl+C`.

Stop Docker containers while keeping development data:

```sh
pnpm infra:down
```

To also delete PostgreSQL, Keycloak, and document Docker volumes, use the following only when you intentionally want a complete development reset:

```sh
docker compose -f deployments/compose/docker-compose.yml down -v
```

> **Warning:** `down -v` permanently deletes the local databases and all data stored in those Compose volumes. It cannot be undone unless you have a backup.

Generated frontend builds, Python environments, caches, and local runtime data are ignored by Git.

## 13. Troubleshooting common errors

### `pnpm: command not found`

Install pnpm 10 or enable it with your Node.js toolchain, then confirm:

```sh
pnpm --version
```

### Unsupported Node.js version

Use a supported LTS version, preferably Node.js 22 LTS or a newer LTS release. Odd-numbered Node releases can be unsupported by transitive build tools even when newer numerically. After changing Node, run:

```sh
pnpm install --frozen-lockfile
```

### Registry, DNS, or proxy errors during bootstrap

Messages such as `ENOTFOUND`, `failed to lookup address`, or `Could not resolve host` mean pnpm, Go, or uv cannot reach its package registry. Confirm internet/DNS access and configure your organization's approved proxy or internal package mirror. Then retry:

```sh
pnpm bootstrap
```

Do not bypass company TLS inspection or disable certificate verification. An air-gapped source deployment requires dependencies and artifacts to be mirrored in advance; `AIR_GAPPED=true` controls runtime behavior and cannot make missing build dependencies appear.

### `go`, `uv`, `docker`, `rustc`, or `cargo` is required

Install only the toolchain for the process you want to test. Web-only development requires none of these. Re-run the version checks from the prerequisites section afterward.

### `Cannot connect to the Docker daemon`

Start Docker Desktop or the system Docker service. Confirm with:

```sh
docker info
```

### A Compose container is `unhealthy` or restarting

Check its logs:

```sh
docker compose -f deployments/compose/docker-compose.yml ps
docker compose -f deployments/compose/docker-compose.yml logs --tail=200 SERVICE_NAME
```

Replace `SERVICE_NAME` with `postgres`, `temporal`, `temporal-ui`, or `keycloak`.

Temporal waits for PostgreSQL. On the first start it can take longer while creating its own databases.

### `connection refused` from the Temporal worker

Confirm Temporal is healthy and port 7233 is available:

```sh
docker compose -f deployments/compose/docker-compose.yml ps temporal
docker compose -f deployments/compose/docker-compose.yml logs --tail=200 temporal
```

Also verify that the local terminal loaded `.env` and that `DOCUMI_TEMPORAL_ADDRESS=localhost:7233`.

### PostgreSQL password authentication failed

Docker volumes retain the password used when PostgreSQL was first created. Changing `.env` does not rewrite an existing database password.

Either restore the original value or, for disposable development data only, perform the destructive volume reset documented in **Stop and clean up**, then start infrastructure again.

### Migration reports that `DOCUMI_DATABASE_URL` is required

The `.env` file exists but was not exported into the terminal. Load it and retry:

```sh
set -a
. ./.env
set +a
```

Confirm without printing credentials:

```sh
test -n "$DOCUMI_DATABASE_URL" && echo "database URL is loaded"
```

### `address already in use`

Check the relevant port on macOS/Linux:

```sh
lsof -nP -iTCP:5173 -sTCP:LISTEN
lsof -nP -iTCP:8080 -sTCP:LISTEN
lsof -nP -iTCP:8081 -sTCP:LISTEN
lsof -nP -iTCP:8082 -sTCP:LISTEN
lsof -nP -iTCP:5432 -sTCP:LISTEN
lsof -nP -iTCP:7233 -sTCP:LISTEN
```

Stop the conflicting process or change the appropriate port. For the web app, use the alternate-port command in the web-only section. For the API:

```sh
DOCUMI_HTTP_ADDRESS=:18080 pnpm dev:api
```

### Browser page is blank or does not update

Check the terminal and browser developer console first. Restart Vite with a forced dependency refresh:

```sh
pnpm --filter @documi/web dev -- --force
```

Then hard-refresh the browser.

### Tauri says Rust or Cargo is missing

Install Rust through `rustup`, restart the terminal, and verify:

```sh
rustc --version
cargo --version
pnpm --filter @documi/desktop tauri info
```

If installation fails with `No space left on device`, free disk space before retrying. Rust, native system packages, Cargo dependencies, and native build output can require several gigabytes.

### `No space left on device` during dependency installation

Check available space:

```sh
df -h .
```

Remove only caches or artifacts you recognize. Safe, regenerable candidates may include old Docker build cache, unused Docker images, package-manager download caches, and this repository's ignored `dist`, `.venv`, or `target` outputs. Do not delete databases or Docker volumes unless you intend to lose their data.

Useful inspection commands:

```sh
docker system df
du -sh node_modules services/ai/.venv apps/desktop/src-tauri/target 2>/dev/null
```

### Python worker prints an idle warning

This is expected in the bootstrap. The worker remains alive until `Ctrl+C`; real Temporal document/AI polling arrives with the first ingestion pipeline.

### AI features do not produce answers

The default configuration is:

```text
DOCUMI_AI_PROVIDER=disabled
```
Only the provider abstraction and fail-closed disabled implementation exist. Ollama, llama.cpp, vLLM, OpenAI-compatible, and custom HTTP adapters are future work. No model is downloaded automatically.

### A document saves in desktop but not in the web application

This indicates that the selected authority or API is unavailable. Tauri local workspaces use SQLite and controlled files; browser/server workspaces use PostgreSQL through the API. Their histories deliberately do not synchronize.

### `document_version_conflict` when saving

The open document is based on an older immutable version. Reopen it to load the current version, reapply the intended change, and save again. Documi does not overwrite either version automatically.

### Collabora is running but the editor is unavailable

Check `COLLABORA_ENABLED`, both Collabora URLs, `DOCUMI_PUBLIC_URL`, the editor-token secret, PostgreSQL migrations, discovery reachability, and whether the document is a supported Office type. The gateway remains fail-closed if discovery or document persistence is unavailable. Browsers must never construct WOPI URLs themselves.

## 14. Recommended first local test checklist

- [ ] `pnpm bootstrap` completes.
- [ ] `pnpm dev` opens the web shell at `http://localhost:5173`.
- [ ] Every navigation page opens.
- [ ] `pnpm dev:api` starts successfully.
- [ ] `/health`, `/ready`, and `/api/v1/system/info` return expected JSON.
- [ ] `pnpm infra:up` starts healthy PostgreSQL, Temporal, and Keycloak containers.
- [ ] The bootstrap migration reports that migrations are current.
- [ ] The Go Temporal worker connects.
- [ ] Both Python workers start and stop cleanly.
- [ ] `pnpm check` passes with no unexpected skipped toolchains.
- [ ] Optional: Tauri opens a native window after its prerequisites are installed.
