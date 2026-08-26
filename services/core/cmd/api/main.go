package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/documi/documi/services/core/internal/modules/system"
	"github.com/documi/documi/services/core/internal/platform/config"
	"github.com/documi/documi/services/core/internal/platform/httpserver"
	"github.com/documi/documi/services/core/internal/platform/logging"
	"github.com/documi/documi/services/core/internal/platform/shutdown"
)

var version = "dev"

func main() {
	configValue, err := config.Load()
	if err != nil {
		slog.Error("load configuration", "error", err)
		os.Exit(1)
	}
	logger := logging.New(configValue.LogLevel)
	ctx, stop := shutdown.Context(context.Background())
	defer stop()

	systemService := system.NewService(version, configValue.Environment, configValue.AirGapped)
	server := &http.Server{
		Addr:              configValue.HTTPAddress,
		Handler:           httpserver.NewRouter(logger, system.NewHandler(systemService), httpserver.StaticReadiness(true)),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("api listening", "address", server.Addr, "environment", configValue.Environment)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), configValue.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			logger.Error("graceful shutdown", "error", err)
			os.Exit(1)
		}
		logger.Info("api stopped")
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api failed", "error", err)
			os.Exit(1)
		}
	}
}
