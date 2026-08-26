package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/documi/documi/services/core/internal/platform/config"
	"github.com/documi/documi/services/core/internal/platform/database"
)

func main() {
	directory := flag.String("dir", "../../database/migrations", "directory containing *.up.sql files")
	flag.Parse()
	configValue, err := config.Load()
	if err != nil {
		slog.Error("load configuration", "error", err)
		os.Exit(1)
	}
	if configValue.DatabaseURL == "" {
		slog.Error("DOCUMI_DATABASE_URL is required")
		os.Exit(1)
	}
	if err := database.Migrate(context.Background(), configValue.DatabaseURL, *directory); err != nil {
		slog.Error("migrate database", "error", err)
		os.Exit(1)
	}
	slog.Info("database migrations are current")
}
