# Documi — Initial Architecture and Repository Bootstrap

You are acting as the principal full-stack/platform engineer responsible for bootstrapping **Documi**, a serious open-source, self-hosted business operations platform.

Do not treat this as a demo, hackathon application, or "chat with PDFs" application.

Your task is to establish the repository, architecture, development environment, documentation, and minimum working skeleton on which the product will be built.

---

# 1. Product vision

Documi combines:

- Documents
- Structured company data
- Forms
- Human tasks and approvals
- Business workflows and automation
- Permissions and auditing
- Integrations
- Private/local AI
- Permission-aware RAG

The product should eventually allow an administrator to configure a process such as:

Supplier document arrives  
→ store document internally  
→ create immutable document version  
→ parse/OCR document  
→ extract structured information using AI  
→ compare information against company rules  
→ request human approval when required  
→ update structured company data  
→ move/archive the document  
→ record every important operation in the audit log  
→ allow authorized employees to search and ask questions about the information later.

AI is NOT the center of the architecture.

The core business system is:

DOCUMENTS  
+ STRUCTURED DATA  
+ PEOPLE / TASKS  
+ WORKFLOWS  
↓  
PRIVATE AI / RAG

Documi must be capable of running entirely inside a company's infrastructure.

Air-gapped deployments are a first-class architectural requirement.

---

# 2. Technology stack

Use this architecture unless there is a technically unavoidable reason not to.

## Frontend

- React
- TypeScript
- Vite
- Tailwind CSS
- shared component/design system
- Tiptap for native Documi documents

Package manager:

- pnpm

Use pnpm workspaces for TypeScript applications and packages.

## Desktop

- Tauri v2
- React + TypeScript frontend
- Rust only for desktop/native integration
- SQLite for local metadata/database later
- sqlite-vec for local vectors later
- local filesystem
- local AI through Ollama or llama.cpp

The Tauri project must use the conventional `src-tauri` structure.

## Core backend

Language:

- Go

Architecture:

- modular monolith
- domain-oriented modules
- REST API initially
- WebSocket/SSE only when realtime functionality requires it

Do NOT create one microservice per domain.

Separate executable processes may share the same Go module.

Initial Go binaries:

- API server
- Temporal/business worker
- migration utility if required

## Server database

- PostgreSQL
- pgvector

PostgreSQL is the canonical server database.

Do not introduce a separate vector database.

## Document and AI processing

Language:

- Python

Dependency/project management:

- uv
- pyproject.toml
- uv.lock

One Python project should initially contain multiple worker entry points rather than creating separate repositories.

Responsibilities include:

- Docling parsing
- Tesseract OCR
- document normalization
- semantic chunking
- embeddings
- retrieval
- reranking later
- structured extraction
- classification
- AI provider adapters

## Workflow engine

- Temporal

Business workflows must be durable.

Separate the concepts of:

- workflow definitions
- activities
- worker registration
- Temporal client/runtime integration

Do not implement long-running approvals using ordinary background jobs.

## Authentication

Server/company mode:

- Keycloak
- OIDC

Design authentication behind an abstraction so LDAP, Active Directory, SAML or other providers can be introduced later.

## AI providers

Never hard-code the application to one AI runtime.

Define an AI provider abstraction supporting eventually:

- Ollama
- llama.cpp
- vLLM
- OpenAI-compatible endpoints
- custom HTTP provider
- disabled AI

Model licenses and application licenses must remain separate.

Do not commit model weights to the source repository.

## Storage

Create a storage-provider abstraction.

Initial providers:

- local filesystem
- mounted network filesystem

Future provider:

- S3-compatible storage

For SMB/NFS, assume the operating system mounts the filesystem and Documi accesses the mounted path.

Do not implement an SMB protocol client in v0.1.

## Deployment

Initial:

- Docker Compose

Later:

- Kubernetes

Kubernetes must NOT be required for local development or the first production version.

---

# 3. Required repository architecture

Create this high-level structure:

```text
documi/
├── apps/
│   ├── web/
│   └── desktop/
│
├── packages/
│   ├── app-shell/
│   ├── ui/
│   ├── editor/
│   ├── api-client/
│   ├── shared-types/
│   ├── workflow-designer/
│   └── config/
│
├── services/
│   ├── core/
│   └── ai/
│
├── contracts/
│   ├── openapi/
│   ├── events/
│   ├── workflows/
│   ├── plugins/
│   └── ai/
│
├── database/
│   ├── migrations/
│   ├── seeds/
│   └── fixtures/
│
├── plugins/
│   ├── builtin/
│   └── examples/
│
├── deployments/
│   ├── compose/
│   ├── keycloak/
│   ├── temporal/
│   ├── airgap/
│   └── kubernetes/
│
├── tests/
│   ├── e2e/
│   ├── integration/
│   ├── security/
│   └── fixtures/
│
├── docs/
│   ├── adr/
│   ├── architecture/
│   ├── product/
│   ├── development/
│   └── security/
│
├── scripts/
├── .github/
│
├── ARCHITECTURE.md
├── AGENTS.md
├── README.md
├── CONTRIBUTING.md
├── SECURITY.md
├── CODE_OF_CONDUCT.md
├── package.json
├── pnpm-workspace.yaml
├── .editorconfig
├── .gitignore
├── .env.example
└── LICENSE
```

Do not create empty directories purely for appearance unless they contain at least a README explaining their purpose or a required initial implementation.

---

# 4. Go backend architecture

Create:

```text
services/core/
├── cmd/
│   ├── api/
│   ├── worker/
│   └── migrate/
│
├── internal/
│   ├── modules/
│   └── platform/
│
├── tests/
├── go.mod
└── Dockerfile
```

Domain modules should eventually include:

```text
auth
organizations
users
groups
workspaces
folders
documents
storage
permissions
collections
forms
workflows
tasks
approvals
connectors
audit
events
search
ai
secrets
notifications
```

Cross-cutting platform concerns belong under:

```text
internal/platform/
├── config/
├── database/
├── http/
├── logging/
├── telemetry/
├── temporal/
├── auth/
├── crypto/
└── shutdown/
```

Do NOT use a repository-wide architecture like:

```text
handlers/
services/
repositories/
models/
```

Instead organize code by domain.

A domain module may look like:

```text
documents/
├── domain.go
├── service.go
├── repository.go
├── repository_postgres.go
├── handler.go
├── events.go
├── permissions.go
└── service_test.go
```

Keep domain interfaces close to the domain that owns them.

Keep infrastructure implementation details behind interfaces where replacement is an explicit product requirement.

---

# 5. Python AI architecture

Create:

```text
services/ai/
├── src/
│   └── documi_ai/
│       ├── api/
│       ├── workers/
│       ├── ingestion/
│       ├── parsers/
│       ├── ocr/
│       ├── normalization/
│       ├── chunking/
│       ├── embeddings/
│       ├── retrieval/
│       ├── reranking/
│       ├── extraction/
│       ├── classification/
│       ├── providers/
│       ├── security/
│       ├── contracts/
│       ├── config/
│       └── observability/
│
├── tests/
├── pyproject.toml
├── uv.lock
└── Dockerfile
```

Create separate executable/CLI entry points for at least:

- document worker
- AI worker

They may share the same Python codebase and container image initially.

Do not create separate Python repositories.

---

# 6. Frontend architecture

Use:

```text
apps/web
apps/desktop
packages/app-shell
packages/ui
packages/editor
packages/api-client
packages/shared-types
```

`apps/web` and `apps/desktop` should primarily contain platform-specific bootstrapping.

Shared product functionality belongs in `packages/app-shell`.

Feature organization should align with product navigation:

```text
documents
data
forms
tasks
automations
ai
integrations
settings
```

Do not duplicate the entire React application between web and desktop.

Desktop-specific filesystem/native behavior must live behind a platform abstraction.

---

# 7. Contracts

Treat cross-language contracts as first-class source files.

Create:

```text
contracts/openapi/
contracts/events/
contracts/workflows/
contracts/plugins/
contracts/ai/
```

REST APIs should eventually be described through OpenAPI.

Define a standard event envelope approximately containing:

```json
{
  "id": "evt_...",
  "type": "document.created",
  "timestamp": "...",
  "organization_id": "...",
  "workspace_id": "...",
  "actor_id": "...",
  "resource_id": "...",
  "data": {}
}
```

Do not maintain independent contradictory representations of API contracts in TypeScript, Go and Python.

Prepare the architecture for generated clients/types later.

---

# 8. Core data model

Document in ARCHITECTURE.md that the important long-term entities include:

```text
organizations
users
groups
memberships

workspaces
folders

documents
document_versions
document_metadata
document_permissions
document_links

storage_objects

collections
collection_fields
collection_records

forms
form_submissions

workflows
workflow_versions
workflow_runs
workflow_steps

tasks
approvals

connections
secrets

ai_models
ai_providers

document_chunks
embeddings

audit_events
notifications
webhooks
```

Do not implement all of these during the bootstrap.

Document them as the intended domain model.

---

# 9. Document architecture

Documents are important business records.

Never design important documents to simply overwrite previous content.

Architect for:

```text
document
    ↓
document versions
```

Each stored file/version should eventually have:

- document ID
- version ID
- storage provider
- storage key
- content hash / SHA-256
- size
- MIME type
- creator
- timestamps

Document storage bytes must not be stored directly inside PostgreSQL unless explicitly justified later.

PostgreSQL stores metadata.

A storage provider stores the bytes.

---

# 10. Permission architecture

Permission hierarchy:

```text
Organization
    ↓
Workspace
    ↓
Folder
    ↓
Document
```

Support roles eventually such as:

- Owner
- Admin
- Manager
- Editor
- Contributor
- Viewer
- Guest

Also allow explicit ACLs.

This is a non-negotiable architecture rule:

RAG/search must apply document authorization DURING retrieval.

NEVER:

```text
retrieve confidential chunks
→ then check permission
```

Instead:

```text
user permission context
→ authorized candidate set
→ keyword/vector retrieval
→ reranking
→ LLM
```

AI must never become a mechanism for bypassing ordinary document permissions.

---

# 11. RAG architecture

Document ingestion lifecycle:

```text
upload
↓
validation
↓
document version
↓
content hash
↓
indexing event/job
↓
format detection
↓
Docling
↓
Tesseract when OCR is needed
↓
normalized representation
↓
semantic chunks
↓
embeddings
↓
pgvector
```

Chunks should eventually carry metadata such as:

```text
organization_id
workspace_id
document_id
version_id

page
heading
section

start_offset
end_offset

content
embedding

permission information

content_hash
metadata
```

Answers must eventually return citations.

Citation quality is a core product capability.

---

# 12. Storage abstraction

Define conceptually:

```go
type StorageProvider interface {
    Put(...)
    Get(...)
    Delete(...)
    Move(...)
    Exists(...)
}
```

Initial implementation:

```text
LocalStorage
MountedFilesystemStorage
```

Future:

```text
S3Storage
```

Do not overbuild distributed object storage during bootstrap.

---

# 13. Workflow architecture

A workflow conceptually follows:

```text
TRIGGER
↓
CONDITION
↓
ACTION
↓
ACTION
↓
HUMAN TASK
↓
ACTION
```

The visual workflow designer should eventually edit a declarative Documi workflow representation.

Temporal is the durable execution engine.

Do not make Temporal's internal representation the public Documi workflow definition.

Maintain separation between:

```text
Documi Workflow DSL
        ↓
compiler/runtime adapter
        ↓
Temporal workflow
```

---

# 14. Audit architecture

Important events must eventually be audited, including:

```text
login

document.open
document.download
document.create
document.update
document.delete

permission.update

ai.query

workflow.start
workflow.approve
workflow.reject

record.create
record.update
```

Audit events must be append-only at the application level.

Never build functionality that silently rewrites audit history.

---

# 15. Air-gapped mode

The architecture must support:

```text
AIR_GAPPED=true
```

Air-gapped operation means the product must be capable of functioning without:

- telemetry
- CDNs
- Google Fonts
- externally hosted application assets
- automatic external model downloads
- external API dependencies
- required internet connectivity

Do not introduce a mandatory SaaS dependency.

---

# 16. Deployment profiles

Document two product deployment profiles.

## Local desktop

```text
Documi Desktop
├── Tauri
├── SQLite
├── sqlite-vec
├── local filesystem
└── Ollama / llama.cpp
```

No server should be required for a purely local workspace.

## Company server

```text
Browser/Desktop
       ↓
Documi API
       ↓
PostgreSQL + pgvector
Temporal
Keycloak
workers
local/private AI
company filesystem/NAS
```

Do NOT implement automatic local/server multi-master synchronization during this bootstrap.

Initially a workspace belongs to either:

```text
Local Workspace
```

or:

```text
Server Workspace
```

Offline synchronization is explicitly deferred.

---

# 17. First vertical slice

The first real product milestone is:

```text
user signs in
↓
workspace exists
↓
invoice PDF uploaded
↓
stored in filesystem
↓
document metadata created
↓
document version created
↓
Docling parses document
↓
OCR when required
↓
chunks produced
↓
embedding generated
↓
pgvector indexed
↓
invoice fields extracted
↓
structured invoice record created
↓
approval task created
↓
manager approves/rejects
↓
record updated
↓
document moved/archived
↓
everything audited
↓
authorized users can later search/ask about it
```

Do NOT attempt to implement this entire workflow during the bootstrap.

However, repository boundaries must make this workflow natural to implement next.

---

# 18. ARCHITECTURE.md

Create a comprehensive root-level:

```text
ARCHITECTURE.md
```

This file is a source of truth for humans and AI coding agents.

It must contain:

1. Product vision
2. Product non-goals
3. Architectural principles
4. Technology stack
5. Repository map
6. Runtime processes
7. Domain model
8. Frontend architecture
9. Go backend architecture
10. Python AI architecture
11. Database architecture
12. Document/version architecture
13. Storage abstraction
14. Event architecture
15. Workflow/Temporal architecture
16. AI provider architecture
17. RAG ingestion architecture
18. RAG query architecture
19. Permission/security model
20. Audit architecture
21. Secrets architecture
22. Local desktop architecture
23. Server architecture
24. Air-gapped architecture
25. API principles
26. First production vertical slice
27. Testing strategy
28. Deployment strategy
29. Explicitly deferred features
30. ADR index
31. Current implementation status

Include Mermaid diagrams where useful.

Do not make ARCHITECTURE.md a marketing document.

It should be an engineering document that allows a new engineer or coding agent to understand the system without requiring previous conversation history.

---

# 19. AGENTS.md

Create root:

```text
AGENTS.md
```

It must instruct future coding agents:

1. Read `ARCHITECTURE.md` before making architectural changes.
2. Read the relevant ADRs before modifying a subsystem.
3. Preserve module boundaries.
4. Prefer the simplest architecture consistent with the product vision.
5. Do not create microservices without an ADR.
6. Do not add infrastructure dependencies without demonstrating why PostgreSQL, filesystem, Temporal or existing infrastructure cannot satisfy the requirement.
7. Do not bypass authorization in RAG/search.
8. Do not add required public-cloud dependencies.
9. Keep air-gapped operation possible.
10. Keep web and desktop product logic reusable.
11. Keep Go core business logic out of HTTP handlers.
12. Keep AI providers replaceable.
13. Keep storage providers replaceable.
14. Treat audit events as append-only.
15. When an architectural decision changes, update `ARCHITECTURE.md` and create/update an ADR.
16. Do not silently contradict architecture documentation.
17. Tests are required for business-critical behavior.

---

# 20. Architecture Decision Records

Create initial ADRs:

```text
docs/adr/0001-go-core-server.md
docs/adr/0002-postgresql-source-of-truth.md
docs/adr/0003-files-outside-database.md
docs/adr/0004-storage-provider-abstraction.md
docs/adr/0005-temporal-workflow-engine.md
docs/adr/0006-ai-provider-abstraction.md
docs/adr/0007-permission-aware-rag.md
docs/adr/0008-local-vs-server-workspaces.md
docs/adr/0009-modular-monolith.md
docs/adr/0010-polyglot-monorepo.md
```

Each ADR should contain:

```text
Status
Context
Decision
Consequences
Alternatives considered
```

Keep them concise.

---

# 21. Bootstrap implementation

For this first task, IMPLEMENT ONLY the foundation necessary to prove the architecture works.

At minimum:

## Root

Create:

- Git configuration files
- pnpm workspace
- root scripts
- environment example
- README
- ARCHITECTURE.md
- AGENTS.md
- ADRs
- documentation folders
- CI skeleton

## Web

Create a working React + TypeScript + Vite application.

Create the initial shell/navigation for:

- Home
- Documents
- Data
- Forms
- Tasks
- Automations
- AI
- Integrations
- Settings

Do not build the full features.

## Desktop

Create a working Tauri v2 application using the shared React application packages.

Ensure `src-tauri` follows normal Tauri conventions.

Do not implement SQLite synchronization yet.

## Go server

Create a runnable Go API server with:

```text
GET /health
GET /ready
GET /api/v1/system/info
```

Use structured configuration and graceful shutdown.

Do not put business logic directly into HTTP handlers.

Create the skeleton module boundaries.

## Temporal worker

Create a runnable Go worker process with a clean registration structure.

A trivial development workflow/activity may be included solely to verify Temporal connectivity.

Do not implement production invoice automation yet.

## Python AI service/workers

Create the Python project with uv.

Create health/config functionality and entry points for:

```text
document-worker
ai-worker
```

Provider interfaces should exist, but only minimal/no-op implementations are required initially.

Do not download models automatically.

## PostgreSQL

Provide Docker Compose configuration.

Enable pgvector.

Create only foundation migrations needed to prove connectivity and establish migration conventions.

Do not create the entire future schema yet.

## Keycloak

Add development configuration/realm placeholders sufficient to document integration.

Do not spend excessive time on advanced SSO during bootstrap.

## Temporal

Add development Docker configuration and worker connectivity.

## Docker Compose

A developer should eventually be able to run a documented command and bring up the required server-side development infrastructure.

---

# 22. Development experience

Create clear commands for:

```text
bootstrap
dev
test
lint
format
build
```

Prefer commands that can be run from the repository root.

Document required local tools.

Do not hide important commands behind undocumented scripts.

---

# 23. Testing

Establish testing conventions for:

- TypeScript unit tests
- Go unit tests
- Python unit tests
- integration tests
- end-to-end tests

Do not write hundreds of placeholder tests.

At minimum verify:

- web application builds
- Go API starts
- Go health endpoint works
- Python package imports/runs
- architecture lint/build commands work

---

# 24. Security rules

Never commit:

- passwords
- secrets
- tokens
- private keys
- model files
- customer data

`.env.example` contains safe placeholders only.

Create `SECURITY.md`.

Prepare for an encrypted secret-store abstraction later.

---

# 25. Avoid premature infrastructure

During this bootstrap DO NOT introduce:

- Kafka
- Elasticsearch
- Redis unless demonstrated necessary
- separate vector database
- Kubernetes as a development requirement
- service mesh
- GraphQL
- multiple Go microservices
- separate repository per worker
- Dropbox-style offline sync
- mobile apps
- collaborative Office integration
- plugin marketplace
- custom OCR engine
- custom LLM runtime
- custom authentication protocol
- custom object-storage system

Keep the first architecture deliberately boring, understandable and operationally simple.

---

# 26. Implementation quality

Act as a principal engineer.

Do not merely generate folders.

Every created structure must have a reason.

Prefer:

- clear boundaries
- small interfaces
- explicit dependencies
- reproducible builds
- strong typing
- testability
- graceful failure
- observability
- security by default
- minimal infrastructure

Avoid:

- circular dependencies
- giant shared utility packages
- god objects
- framework-specific domain logic
- duplicated web/desktop product code
- hidden global state
- unnecessary abstractions
- speculative microservices

---

# 27. Before writing code

First:

1. Inspect the repository.
2. Create or update `ARCHITECTURE.md`.
3. Create `AGENTS.md`.
4. Create the ADRs.
5. Present the final planned repository tree.
6. Check for contradictions.
7. Then implement the bootstrap.

Do not ask for confirmation unless a decision is genuinely impossible to infer.

Use the architecture in this prompt as the default decision.

---

# 28. After implementation

Run all available:

- formatting
- linting
- type checking
- Go tests
- Python tests
- TypeScript tests
- builds

Fix errors rather than merely reporting them when the fix is within scope.

Then provide a final report containing:

## Created

What was created.

## Architecture

Major boundaries established.

## Running locally

Exact commands.

## Verification

What tests/builds were actually run.

## Known gaps

Anything intentionally deferred.

## Next milestone

Recommend the next implementation step toward:

```text
upload invoice
→ store
→ document version
→ process
→ index
→ extract
→ approve
→ audit
```

Most importantly:

**Leave the repository in a clean, runnable state where `ARCHITECTURE.md` captures the architecture well enough that a completely new engineering session can continue the project without access to this conversation.**