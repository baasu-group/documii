# Docker Compose development environment

From the repository root:

```sh
cp .env.example .env
pnpm infra:up
docker compose -f deployments/compose/docker-compose.yml --profile application up --build
```

The default profile starts PostgreSQL/pgvector, Temporal and its UI, and Keycloak. The optional `application` profile builds the Go API and worker. Defaults are development-only and must be replaced in production.
