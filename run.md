# Running Documi locally

This guide covers the current Documi bootstrap on a local machine: installation, web-only testing, the Go API, PostgreSQL/pgvector, Temporal, Keycloak, Python workers, the Tauri desktop shell, verification, shutdown, and common errors.

> **Current product status:** this repository is an architecture bootstrap with a working native Tiptap preview and editor-provider/WOPI foundations. Native saves are preview-only until the persistence-backed Documents/version service exists. Collabora, ONLYOFFICE, Google, and LibreOffice save-back fail closed or remain unavailable rather than bypassing permissions and versions.

## Local verification status

Last exercised on macOS on 2026-08-26:

| Check | Result | Observation |
| --- | --- | --- |
| `pnpm bootstrap` | Passed | Frozen pnpm, Go, and uv dependencies were already current |
| Live Vite server | Passed | HTML and transformed React/AppShell module served from `http://localhost:5173` |
| Browser click-through | Not run | No controllable browser was connected; navigation remains covered by its unit test |
| Live Go API | Passed | All three endpoints returned expected JSON and graceful shutdown completed |
| Python document worker | Passed | Started, emitted expected bootstrap-idle warning, and stopped cleanly |
| Python AI worker | Passed | Started, emitted expected bootstrap-idle warning, and stopped cleanly |
| Go Temporal worker | Environment-blocked | Correctly reported connection refused because Temporal was not running |
| `pnpm check` | Passed | Formatting, lint, type checks, tests, and builds all succeeded |
| Docker infrastructure | Environment-blocked | Docker is not installed on the verification machine |
| Database migration | Environment-blocked | A different local PostgreSQL owns port 5432 and has no `documi` role |
| Desktop frontend build | Passed | TypeScript and Vite production build succeeded |
| Native Tauri build | Environment-blocked | Xcode is installed, but Rust/Cargo are not installed |
| Native editor package | Passed | Type check, native-format tests, web build, and desktop frontend build passed |
| Editor API/WOPI unit tests | Passed | Provider resolution, air-gap availability, WOPI permission/read-only/save-version boundaries, and signing tests passed |
| Live editor API | Passed | Availability returned normalized providers; missing WOPI token returned `401` |
| Collabora container | Environment-blocked | Docker is not installed; optional profile was not started |

The sections below remain the canonical process. Environment-blocked items should be rerun after their prerequisites are installed; they are not recorded as successful tests.

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
- Go 1.23 or newer
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

Create local configuration without overwriting an existing `.env`:

```sh
test -f .env || cp .env.example .env
test -f .env && echo ".env is ready"
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

The bootstrap UI currently uses local placeholder content. It does not yet fetch documents or users from the Go API.

### Use a different web port

If port 5173 is already in use:

```sh
pnpm --filter @documi/web dev -- --port 5174
```

Then open `http://localhost:5174`.

## 5. Run and test the Go API without Docker

The bootstrap API does not require PostgreSQL for its three current endpoints.

Copying `.env` does **not** automatically export it into local Go or Python processes. On macOS/Linux, load it into the current terminal first:

```sh
if [ ! -f .env ]; then
  echo "Missing .env; copy .env.example first"
  exit 1
fi
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

When the API is launched through pnpm, pnpm may also print `ELIFECYCLE` or an exit-code message after `Ctrl+C`. If the API logged `api stopped` immediately before it, the application shutdown itself was graceful; the wrapper is reporting the interrupt signal.

### Test air-gapped configuration

```sh
AIR_GAPPED=true DOCUMI_HTTP_ADDRESS=:8080 pnpm dev:api
```

The system-info response should then contain `"airGapped":true`.

### Test document editors locally

Run the Go API in one terminal, then the web app in another:

```sh
set -a
. ./.env
set +a
pnpm dev:api
```

```sh
pnpm --filter @documi/web dev
```

Open `http://localhost:5173`, choose **Documents**, then **Documi Document**. The browser route changes to `/documents/local-preview/edit` and the Tiptap editor supports headings, lists, task lists, code blocks, tables, undo, and redo. **Save** currently confirms only the local preview; it does not claim to create a durable document version.

Verify provider availability and the WOPI authorization boundary:

```sh
curl -fsS http://localhost:8080/api/v1/document-editors
curl -i http://localhost:8080/api/v1/wopi/files/example
```

The first response includes `native` as available and disabled reasons for optional providers. The second must return `401 Unauthorized` because no short-lived editor token was supplied.

### Optional Collabora profile

Collabora runs in a separate container; browser users do not install LibreOffice. Start infrastructure, the API, and CODE with:

```sh
COLLABORA_ENABLED=true \
COLLABORA_URL=http://collabora:9980 \
COLLABORA_PUBLIC_URL=http://localhost:9980 \
docker compose -f deployments/compose/docker-compose.yml \
  --profile application --profile collabora up --build
```

The pinned development image is available on port `9980`. In the current repository, the API deliberately reports Collabora unavailable with reason `document_service_unavailable`: the WOPI protocol and security boundary exist, but the authenticated Documents/version gateway and editor-session issuance do not. This profile validates deployment wiring only; it does not yet provide an editable Office document.

Common Collabora errors:

- `docker: command not found`: install/start Docker, or continue with the native editor.
- API exits with a token-secret error: set `DOCUMI_EDITOR_TOKEN_SECRET` to at least 32 random characters.
- Collabora loads but cannot reach WOPI: confirm the API and Collabora share the Compose network and that `COLLABORA_URL` uses `http://collabora:9980` inside Compose.
- Provider says `document_service_unavailable`: expected in this foundation; do not work around it by exposing storage paths.

### Desktop and optional LibreOffice

Install Rust/Tauri prerequisites, then run the API and desktop frontend:

```sh
pnpm dev:api
```

```sh
pnpm --filter @documi/desktop tauri dev
```

Open **Settings → Document Editors**. The Tauri command checks standard installation locations and the normal `soffice`/`libreoffice` executable lookup. If missing, Documi remains usable and displays the optional-install message. Install LibreOffice yourself only if you want this feature; Documi does not install it. Opening a controlled working copy and **Save back to Documi** are not implemented yet.

### Google Workspace configuration

Google Workspace is intentionally unavailable in this build. The connector, encrypted SecretStore, OAuth callback, and import/export services do not yet exist, so there are no safe Google client-secret environment variables to configure. Do not put refresh tokens in document metadata or `.env`. With `AIR_GAPPED=true`, the availability API returns `air_gapped` for all Google editor providers.

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

If another local PostgreSQL already owns port 5432, edit `.env` before starting Compose. Change both values together, for example:

```dotenv
DOCUMI_POSTGRES_PORT=55432
DOCUMI_DATABASE_URL=postgres://documi:documi_dev_only@localhost:55432/documi?sslmode=disable
```

Compose maps the selected host port to PostgreSQL's internal port 5432. The API and migration utility use `DOCUMI_DATABASE_URL`, so its port must match.

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

These credentials are for local development only. The imported `documi` realm provides the public `documi-web` client but deliberately contains no test users. Authentication is not yet enforced by the bootstrap API/UI.

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

The migration enables `pgvector`, creates the migration bookkeeping table, and creates a minimal installation marker table. It does not create the future product domain schema.

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

## 10. Run the desktop application

First confirm the native environment:

```sh
pnpm --filter @documi/desktop tauri info
```

The output should show installed versions of `rustc`, Cargo, and the platform webview/native build tools. Then run:

```sh
pnpm dev:desktop
```

Tauri starts the Vite desktop frontend at port 1420 and opens the native Documi window. The desktop currently renders the same shared shell as the web app; SQLite, sqlite-vec, local document persistence, native file operations, and local AI are deferred.

To verify only the desktop frontend without Rust:

```sh
pnpm --filter @documi/desktop build
```

This validates TypeScript and builds the desktop web assets, but it does not compile the Rust native application.

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
- Temporal development activity wiring.
- Python health and worker startup.
- Disabled AI provider failing closed.

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

### PostgreSQL reports `role "documi" does not exist`

If this appears before Documi's Compose PostgreSQL has started, you are probably connecting to a different PostgreSQL already installed on the machine. Identify the listener without changing it:

```sh
lsof -nP -iTCP:5432 -sTCP:LISTEN
```

Do not create Documi roles in an unrelated database just to hide the conflict. Either stop that local server while testing Documi or configure `DOCUMI_POSTGRES_PORT` and `DOCUMI_DATABASE_URL` to use another host port as shown in **Start server infrastructure with Docker**.

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

Use the exact host printed by Vite. On a machine where `localhost` resolves to IPv6, `http://localhost:5173` can work while `http://127.0.0.1:5173` does not unless Vite is explicitly configured to bind IPv4.

### Browser page is blank or does not update

Check the terminal and browser developer console first. Restart Vite with a forced dependency refresh:

```sh
pnpm --filter @documi/web dev -- --force
```

Then hard-refresh the browser.

### Node prints `ExperimentalWarning: Type Stripping`

This was observed with the odd-numbered Node.js 23 release while loading TypeScript build configuration. It did not fail the build, but the recommended fix is to use an LTS Node release such as Node.js 22 or a newer supported LTS version.

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

### The UI does not upload documents or sign in

That behavior has not been implemented yet. The bootstrap UI is intended to validate shared web/desktop architecture and navigation. The next milestone is authenticated workspace context plus immutable document upload/storage/version/audit.

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
