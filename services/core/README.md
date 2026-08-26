# Documi core

One Go module builds the API, Temporal business worker, and migration utility. Run commands from this directory with Go 1.23+:

```sh
go run ./cmd/api
go run ./cmd/worker
go run ./cmd/migrate --dir ../../database/migrations
go test ./...
```

The API is intentionally independent from PostgreSQL during bootstrap so liveness can be tested before the first product schema exists. Dependency-aware readiness is introduced with persistence-backed features.
