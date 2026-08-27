# ADR 0011: Document editor provider architecture

## Status

Accepted

## Context

Documi needs a lightweight native editor, browser Office suites, an optional desktop Office application, conversion workers, and external Google Workspace references. These integrations differ in ownership, storage, security, deployment, offline support, and save behavior. Treating them as interchangeable UI widgets would let vendor callbacks bypass document permissions and immutable version policy.

## Decision

Documi owns document identity, workspace membership, permissions, metadata, storage references, immutable versions, and audit history. Editors are replaceable providers selected by a server-side capability resolver.

Persist stable provider IDs (`native`, `collabora`, `onlyoffice`, `libreoffice-desktop`, `google-docs`, `google-sheets`, and `google-slides`), never Go type names. Providers report operations, supported document families, and availability. The Documents application service remains the only path to content and new versions.

Managed documents have Documi-controlled content. External documents are explicit references to content owned by another system; their credentials live in the SecretStore, not document metadata. Air-gapped mode rejects public-internet providers server-side.

Collabora reaches documents only through authenticated WOPI endpoints. Editing requires a persisted, expiring document lock and every save compares the lock's base version before creating a new immutable version. Session action URLs come from bounded, same-origin Collabora discovery; the browser never constructs privileged WOPI URLs. ONLYOFFICE uses signed server-generated configuration and validated callbacks. Desktop LibreOffice uses a controlled working copy and explicit save-back. Headless LibreOffice runs in an isolated worker, not the API process. Google integration uses the existing connector/secrets boundary.

Editor preferences are organization policy resolved server-side by family, capability, and live availability. Managed Office creation uses reviewed blank application assets with immutable upstream provenance and pinned SHA-256 checks; template corruption disables the dependent creation path. Provider sessions flow through the same authenticated React document gateway as document mutations.

## Consequences

- New editors implement one provider boundary and do not alter the Documents aggregate.
- Availability and default selection remain centralized and testable.
- Provider callbacks must re-check current permission and create a document version.
- A provider may be configured yet unavailable until its document gateway is wired.
- External documents cannot claim air-gapped or Documi-managed storage guarantees.
- The interface can grow only around shared product capabilities; provider-specific protocols stay inside adapters.

## Alternatives considered

Hard-coded provider checks in handlers or React spread policy across layers. Storing provider-owned copies as the document source of truth weakened version and permission guarantees. Treating LibreOffice headless as a browser editor confused conversion with interactive editing. A universal editor API hid important managed/external and browser/desktop differences.
