# Document editor architecture

## Ownership and boundaries

The `documents` module owns identity, workspace, permissions, versions, metadata, storage references, and audit events. An `EditorProvider` owns creation/open mechanics, capability reporting, provider configuration, and its protocol adapter. Providers never retrieve bytes directly from filesystem paths and never replace a version outside the Documents application service.

```text
Document request
  -> Documents permission policy
  -> EditorProviderResolver
  -> capability + family + availability
  -> provider adapter
  -> Documents service for content/version/audit
```

Stable provider IDs are `native`, `collabora`, `onlyoffice`, `libreoffice-desktop`, `google-docs`, `google-sheets`, and `google-slides`. Selection policy is centralized in the Go resolver. React consumes the normalized availability API and does not infer installed software or choose an editor from filename extensions.

Capabilities currently describe `create`, `edit`, `view`, `convert`, `export`, `collaborative`, `browser`, `desktop`, `offline`, `external`, and `external-process`, with native, text, spreadsheet, presentation, and PDF families. Availability is distinct from capability: Collabora can support browser editing while being disabled or missing its Documents gateway.

## Managed and external documents

A managed document has Documi-controlled content and versions. Relevant metadata is `editor_provider`, `storage_mode=managed`, and, for native content, `native_format=documi-json-v1`.

An external document is a Documi identity and permission/audit reference whose live content remains with a third party. It uses `storage_mode=external`, `external_provider`, `external_file_id`, and an official external URL. Documi does not claim that Google-hosted content is offline or air-gapped. OAuth credentials are referenced through the connections and SecretStore boundary, never embedded in document metadata.

## Native editor

`packages/editor` owns the Tiptap implementation and a versioned JSON envelope. It supports the lightweight Documi format in browser and desktop, including the existing Tiptap extension model for headings, paragraphs, lists, task lists, tables, images, links, code, and future Documi extensions for mentions, comments, templates, variables, and AI actions. It is not a DOCX rendering engine.

In a Tauri local workspace the editor creates real records, reloads them after restart, and performs explicit immutable saves through fixed commands. SQLite holds metadata, versions, idempotency, and append-only audit events; `documi-json-v1` bytes are limited to 5 MiB and stored in application-controlled files. In server workspaces the same React gateway uses authenticated PostgreSQL-backed APIs, and every native save checks the base version before committing metadata, audit, and the immutable storage reference.

## Collabora Online and WOPI

Collabora is an optional separate server/container. Browser users do not install LibreOffice. The target flow is:

```text
authorized user -> Documi editor session -> short-lived token
  -> Collabora iframe -> Documi WOPI host -> Documents service
  -> StorageProvider / new document version / audit
```

The WOPI foundation implements `CheckFileInfo`, `GetFile`, `PutFile`, `LOCK`, `REFRESH_LOCK`, `GET_LOCK`, `UNLOCK`, and atomic unlock/relock dispatch beneath `/api/v1/wopi/files/{id}`. HMAC claims are bound to a persisted session, document, actor, mode, and base version, with a maximum ten-hour lifetime. Every call validates the unexpired session row and re-checks current access; denied documents are non-enumerating 404 responses. Read-only sessions cannot lock or save. Filenames, locks, and body sizes are bounded, paths are never returned, and saves call `SaveNewVersion` with an expected base version and lock.

Discovery parsing resolves view/edit action URLs from `${COLLABORA_URL}/hosting/discovery` with a five-second timeout, a bounded response, same-origin redirects, and a five-minute cache. When PostgreSQL, document storage, configuration, checksum-verified Office templates, and discovery are ready, the running composition issues persisted sessions and uses the Documents service for WOPI content, 30-minute locks, immutable saves, conflict detection, and generic/provider audit events. If any required dependency is absent, availability and session creation fail closed. The React iframe receives sessions through the authenticated document gateway; it never creates an unauthenticated side client.

## Managed Office creation templates

New DOCX, XLSX, and PPTX documents are copied from known application assets, never assembled as ad-hoc ZIP files. The source bytes are text-encoded for repository review, decoded in memory, and SHA-256 checked before use. A checksum mismatch makes Office creation and Collabora availability fail closed.

| Family | Upstream immutable source | SHA-256 |
| --- | --- | --- |
| Text | `python-openxml/python-docx` commit `e45454602b53e8e572b179ccf1c91093ec9f4ed7`, `doc-word-default-blank.docx` | `560b0e299a592f6b7d1ec9abab98648f1d92bf8fa684ce544a0424fdde798903` |
| Spreadsheet | `dtjohnson/xlsx-populate` commit `7480a02575c9140c0e7995623ea192c88c1886d3`, `blank/blank.xlsx` | `32ce7d6f125c2cdde13a61c94c6fd0f05e96dea7ab73c0f7ee6b67b6dd37cf04` |
| Presentation | `office-kit/pptx` commit `51b1f90b91ef9ea30b49bc38e5500c8acc344412`, `test/fixtures/minimal/blank.pptx` | `dd4746eb40918140f90310247c04e3a3aa7232689de2e2f37670a10e562a4454` |

Unit tests verify package integrity and reject macro parts. Office creation uses the same create capability, opaque storage keys, immutable initial version, idempotency, provider resolver, and audit transaction as uploads.

## ONLYOFFICE

The adapter foundation provides an isolated HS256 signer/verifier for server-created configurations and callbacks. It is independently configured and disabled by default. A production adapter must validate callback identity, token, document key, replay state, current permissions, expected version, download URL allowlists, and payload size before creating a version and audit event. The callback/session integration is not implemented.

## LibreOffice Desktop

LibreOffice Desktop is a desktop capability, not a web editor. Tauri exposes only `check_libreoffice`; it examines known macOS/Windows locations and the normal executable lookup (`soffice`/`libreoffice`) and returns installed/version/executable status. Absence is non-fatal and the native editor remains available.

The future open flow must download authorized content to an application-controlled path such as `editor-workspace/<document-id>/<version-id>/<safe-name>`. The frontend must never provide an arbitrary executable or server storage path. Explicit **Save back to Documi** will upload the controlled working copy through the Documents service, enforce `can_edit`, create a version, detect version conflicts, and audit the operation. Automatic multi-master synchronization is excluded.

## LibreOffice headless conversion

`DocumentConverter` is a worker-facing boundary for controlled DOCX/ODT/XLSX/ODS/PPTX/ODP to PDF conversion. The Go API never shells out to LibreOffice. A runtime implementation belongs in an optional isolated worker/container with argument arrays, generated filenames, isolated temporary directories, file-size and concurrency limits, a subprocess timeout, macro restrictions, and reliable cleanup. No runtime converter is implemented yet.

## Google Workspace

Google Docs, Sheets, and Slides are optional external providers. The intended flow creates a file through an existing Google connection, then creates a Documi external reference. Opening uses the official Google editor URL. Explicit import exports a DOCX/XLSX/PPTX/PDF snapshot into Documi storage and creates a managed document version; explicit export uses Drive conversion. Automatic or bidirectional synchronization is excluded.

Google providers report unavailable today because the connector and SecretStore are not implemented. With `AIR_GAPPED=true`, the server reports `air_gapped` and future connector operations must reject requests even if a client displays stale UI.

## Permission, version, and audit rules

All providers call Documents policy before operations: `can_read` for open, `can_edit` for edit/save/replace, and `can_download` for download/export. Provider tokens narrow authority; they never grant authority absent from current policy. Save callbacks re-check access and base version. Successful flows append provider-specific and generic events without tokens or content, including `document.editor.opened/saved`, `document.collabora.opened/saved`, `document.onlyoffice.opened/saved`, `document.libreoffice.opened/saved`, `document.external.created/opened`, `document.google.created/imported/exported`, and `document.imported/exported`.

Folder, group, membership, additive grant, effective-access, editor-policy, and cursor-audit administration live in an application service beside Documents rather than in HTTP handlers. Mutations require the relevant `manage_permissions` or `manage_members` capability, validate tenant hierarchy, append audit in the same transaction, preserve the last organization Owner, and return non-enumerating errors. React consumes these operations through the shared administration gateway.

## Configuration and air-gapped operation

Environment configuration is:

- `DOCUMI_EDITOR_TOKEN_SECRET`
- `COLLABORA_ENABLED`, `COLLABORA_URL`, `COLLABORA_PUBLIC_URL`
- `ONLYOFFICE_ENABLED`, `ONLYOFFICE_URL`, `ONLYOFFICE_JWT_SECRET`
- `AIR_GAPPED`

Secrets require at least 32 characters when their provider is enabled. URLs are absolute HTTP(S) values and are never hard-coded in application code. Deployment-level egress controls and an explicit host allowlist remain required before provider-initiated fetching is enabled. Native editing works without external services. Private-network Collabora/ONLYOFFICE and local LibreOffice remain compatible with air-gapped mode when fully configured.

## Implementation status

| Provider | Status | Current behavior |
| --- | --- | --- |
| Native | Working | server workflow is integration-tested; all native Rust tests, including file-backed SQLite restart persistence, strict Clippy, and a macOS no-bundle Tauri release build pass locally |
| Collabora | Partial | managed Office creation, persisted sessions/WOPI/locks/save versions, and a scheduled real-container OIDC/browser acceptance are implemented; the acceptance has not run on this workstation |
| LibreOffice Desktop | Foundation only | desktop detection command and missing-state UI; open/save-back deferred |
| LibreOffice Headless | Foundation only | converter interface only |
| Google Workspace | Foundation only | normalized disabled/air-gap availability only |
| ONLYOFFICE | Foundation only | provider/configuration and JWT signer only |
