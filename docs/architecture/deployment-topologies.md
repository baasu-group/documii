# Deployment topologies and operating systems

Documi supports two separate document authorities. Selecting one is explicit; neither silently synchronizes into the other.

## Local desktop authority

Documi Desktop runs on Windows, macOS, and Linux without a server. Tauri selects the platform application-data directory, bundled SQLite stores metadata, and immutable content is stored below `documents/<workspace>/<document>/<version>/`. React receives identifiers and content, never filesystem paths. Local files remain local to that operating-system user.

Typical application-data roots differ by OS and must never be hard-coded:

- Windows: the Tauri-resolved roaming/local application-data location.
- macOS: the Tauri-resolved Application Support location.
- Linux: the Tauri-resolved XDG data location.

The Rust CI matrix runs formatting, Clippy, and tests on Windows, macOS, and Linux. Native packaging still requires each platform's normal signing and webview prerequisites.

## Company server authority on a private network

Windows, macOS, and Linux browsers/desktops connect to the Documi HTTPS API. They do **not** receive PostgreSQL credentials and never connect directly to the database.

```text
Browser/Desktop
      |
      | HTTPS / OIDC
      v
Documi API + WOPI host
      |                 \
      | PostgreSQL       \ controlled filesystem/NAS mount
      v                   v
private database       immutable document bytes
```

PostgreSQL may run on the same server or a separate LAN database host. Set `DOCUMI_DATABASE_URL` only on the API/admin/migration processes. Restrict port 5432 to application and administrative hosts. PostgreSQL requires pgvector because migration `000001` enables it.

Storage may be a local disk or an SMB/NFS share mounted by the server OS. `DOCUMI_STORAGE_ROOT` is the local mount path seen by the API. Documi does not embed SMB credentials or implement the SMB protocol. The mount must support same-directory rename and normal file durability semantics; otherwise use reliable local/server storage until another StorageProvider is implemented.

## Optional private-network editors

Collabora and later ONLYOFFICE run as separate services. `COLLABORA_URL` is the API-to-Collabora discovery address, `COLLABORA_PUBLIC_URL` is reachable by client browsers, and `DOCUMI_PUBLIC_URL` is reachable by the Collabora server for WOPI. In a Compose-only network the latter may be `http://api:8080`; production normally uses an internal HTTPS DNS name.

Air-gapped deployments can use native editing, privately hosted Collabora/ONLYOFFICE, and local LibreOffice. Google providers remain rejected server-side.

## Limits and fallback

- A local SQLite workspace is single-device authority; there is no automatic LAN synchronization.
- A server workspace requires API reachability. If the LAN is unavailable, use a separate local workspace and explicitly import later; do not copy SQLite rows into PostgreSQL.
- PostgreSQL is optional for purely local desktop use and required for company/server workspaces.
- Collabora requires a reachable separate service and supported Office content. Native documents continue working when it is disabled.
- Network-mounted storage behavior varies by appliance. Validate atomic rename, permissions, capacity, backups, and restore before production use.
