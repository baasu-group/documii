# Database migrations

Forward migrations use `<six-digit-version>_<name>.up.sql`; optional development rollback files use `.down.sql`. Production changes move forward with a corrective migration rather than editing an applied file. Run from `services/core` with `go run ./cmd/migrate --dir ../../database/migrations`.
