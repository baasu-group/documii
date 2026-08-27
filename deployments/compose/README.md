# Docker Compose development environment

From the repository root:

```sh
cp .env.example .env
pnpm infra:up
docker compose -f deployments/compose/docker-compose.yml --profile application up --build
```

The default profile starts PostgreSQL/pgvector, Temporal and its UI, and Keycloak. The optional `application` profile builds the Go API, migration/admin binaries, and worker; migrations complete before the API starts. OIDC is the application-profile default. Defaults are development-only and must be replaced in production.

The optional real editor acceptance stack is:

```sh
COLLABORA_ENABLED=true docker compose -f deployments/compose/docker-compose.yml \
  --profile application --profile collabora --profile acceptance up --build
```

The `acceptance` profile bootstraps only the fixed development Keycloak subject declared in `deployments/keycloak/documi-realm.json`. It is not a production bootstrap mechanism.

For ordinary Documi + Collabora development without the acceptance account:

```sh
COLLABORA_ENABLED=true docker compose -f deployments/compose/docker-compose.yml \
  --profile application --profile collabora up --build
```
