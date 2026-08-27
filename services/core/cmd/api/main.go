package main

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/documi/documi/services/core/internal/modules/documents/editors"
	"github.com/documi/documi/services/core/internal/modules/documents/editors/collabora"
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
	editorProviders := editors.DefaultProviders(editors.ProviderConfiguration{
		AirGapped:                   configValue.AirGapped,
		CollaboraEnabled:            configValue.Editors.CollaboraEnabled,
		CollaboraUnavailableReason:  "document_service_unavailable",
		OnlyOfficeEnabled:           configValue.Editors.OnlyOfficeEnabled,
		OnlyOfficeUnavailableReason: "document_service_unavailable",
	})
	editorRegistry, err := editors.NewRegistry(editorProviders...)
	if err != nil {
		logger.Error("register document editor providers", "error", err)
		os.Exit(1)
	}
	editorService := editors.NewService(editorRegistry, editors.NewResolver(editorRegistry, editors.DefaultPolicy()))
	editorHandler := editors.NewHandler(editorService, editors.UnavailableSessionService{})

	editorTokenSecret := []byte(configValue.Editors.TokenSecret)
	if len(editorTokenSecret) < 32 {
		editorTokenSecret = make([]byte, 32)
		if _, err := rand.Read(editorTokenSecret); err != nil {
			logger.Error("create ephemeral editor token secret", "error", err)
			os.Exit(1)
		}
	}
	tokenService, err := collabora.NewTokenService(editorTokenSecret, 5*time.Minute)
	if err != nil {
		logger.Error("configure WOPI token service", "error", err)
		os.Exit(1)
	}
	wopiHandler, err := collabora.NewHandler(tokenService, collabora.UnavailableGateway{}, 100<<20)
	if err != nil {
		logger.Error("configure WOPI host", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr: configValue.HTTPAddress,
		Handler: httpserver.NewRouter(logger, httpserver.Routes{
			System:  system.NewHandler(systemService),
			Editors: editorHandler,
			WOPI:    wopiHandler,
		}, httpserver.StaticReadiness(true)),
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
