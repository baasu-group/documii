# Temporal development profile

Compose runs Temporal backed by the same PostgreSQL server but separate Temporal-owned databases. Documi application tables never write directly to those databases. The UI is available at `http://localhost:8082`; the worker connects to `localhost:7233`.
