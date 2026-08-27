# End-to-end tests

`collabora.spec.mjs` is the real browser delivery test. It requires the Compose `application`, `collabora`, and `acceptance` profiles plus the web dev server. It logs into Keycloak through authorization code + PKCE, creates a checksum-pinned DOCX, verifies the Collabora iframe, performs protocol locking/save with the browser-issued WOPI token, then verifies the immutable version and audit events through the authenticated API.

Run it using the exact setup in `run.md`, or dispatch `.github/workflows/collabora-acceptance.yml`. It is intentionally separate from normal unit tests because it downloads and starts large third-party containers and a browser runtime.
