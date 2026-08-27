package main

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/documi/documi/services/core/internal/modules/administration"
	"github.com/documi/documi/services/core/internal/modules/documents"
	"github.com/documi/documi/services/core/internal/modules/documents/editors"
	"github.com/documi/documi/services/core/internal/modules/documents/editors/collabora"
	"github.com/documi/documi/services/core/internal/modules/documents/templates"
	"github.com/documi/documi/services/core/internal/modules/permissions"
	"github.com/documi/documi/services/core/internal/modules/storage"
	"github.com/documi/documi/services/core/internal/modules/system"
	platformauth "github.com/documi/documi/services/core/internal/platform/auth"
	"github.com/documi/documi/services/core/internal/platform/config"
	"github.com/documi/documi/services/core/internal/platform/httpserver"
	"github.com/documi/documi/services/core/internal/platform/logging"
	"github.com/documi/documi/services/core/internal/platform/shutdown"
	"github.com/jackc/pgx/v5/pgxpool"
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
	var authenticator platformauth.Authenticator
	if configValue.Auth.Mode == "oidc" {
		value, err := platformauth.NewOIDCAuthenticator(ctx, configValue.Auth.Issuer, configValue.Auth.DiscoveryURL, configValue.Auth.Audience)
		if err != nil {
			logger.Error("configure OIDC", "error", err)
			os.Exit(1)
		}
		authenticator = value
	} else {
		authenticator = platformauth.DevelopmentAuthenticator{Subject: platformauth.Subject{ID: configValue.Auth.DevelopmentSubject, Issuer: "documi-development", ExternalSubject: configValue.Auth.DevelopmentSubject, Email: configValue.Auth.DevelopmentEmail, DisplayName: configValue.Auth.DevelopmentName}}
	}
	var documentHandler *documents.Handler
	var documentService *documents.Service
	var databasePool *pgxpool.Pool
	if configValue.DatabaseURL != "" {
		databasePool, err = pgxpool.New(ctx, configValue.DatabaseURL)
		if err != nil {
			logger.Error("configure PostgreSQL", "error", err)
			os.Exit(1)
		}
		defer databasePool.Close()
		if err := databasePool.Ping(ctx); err != nil {
			logger.Error("connect PostgreSQL", "error", err)
			os.Exit(1)
		}
		authenticator = platformauth.NewPersistentAuthenticator(authenticator, databasePool)
		filesystem, err := storage.NewFilesystem(configValue.StorageRoot, documents.MaxDocumentBytes)
		if err != nil {
			logger.Error("configure document storage", "error", err)
			os.Exit(1)
		}
		authorizer := permissions.NewAuthorizer(databasePool)
		documentService = documents.NewService(databasePool, filesystem, authorizer)
		documentHandler = documents.NewHandler(documentService)
	}

	editorTokenSecret := []byte(configValue.Editors.TokenSecret)
	if len(editorTokenSecret) < 32 {
		editorTokenSecret = make([]byte, 32)
		if _, err := rand.Read(editorTokenSecret); err != nil {
			logger.Error("create ephemeral editor token secret", "error", err)
			os.Exit(1)
		}
	}
	tokenService, err := collabora.NewTokenService(editorTokenSecret, 10*time.Hour)
	if err != nil {
		logger.Error("configure WOPI token service", "error", err)
		os.Exit(1)
	}
	var sessionService editors.SessionService = editors.UnavailableSessionService{}
	var wopiGateway collabora.DocumentGateway = collabora.UnavailableGateway{}
	collaboraReady := false
	collaboraReason := "document_service_unavailable"
	if configValue.Editors.CollaboraEnabled && documentService != nil {
		templateErr := templates.Ready()
		if templateErr != nil {
			collaboraReason = "office_templates_unavailable"
			logger.Error("Office templates unavailable", "error", templateErr)
		} else {
		discovery, discoveryErr := collabora.NewDiscovery(configValue.Editors.CollaboraURL)
		if discoveryErr == nil {
			discoveryErr = discovery.Healthy(ctx)
		}
		if discoveryErr != nil {
			collaboraReason = "discovery_unavailable"
			logger.Warn("Collabora discovery unavailable", "error", discoveryErr)
		} else {
			persistentSessions, sessionErr := collabora.NewPersistentSessionService(databasePool, documentService, tokenService, discovery, configValue.PublicURL, configValue.Editors.CollaboraPublicURL)
			if sessionErr != nil {
				logger.Error("configure Collabora sessions", "error", sessionErr)
				os.Exit(1)
			}
			sessionService = persistentSessions
			wopiGateway = collabora.NewPersistentGateway(documentService)
			collaboraReady = true
			collaboraReason = ""
		}
		}
	}
	wopiHandler, err := collabora.NewHandler(tokenService, wopiGateway, documents.MaxDocumentBytes)
	if err != nil {
		logger.Error("configure WOPI host", "error", err)
		os.Exit(1)
	}
	editorProviders := editors.DefaultProviders(editors.ProviderConfiguration{
		AirGapped:                   configValue.AirGapped,
		CollaboraEnabled:            configValue.Editors.CollaboraEnabled,
		CollaboraReady:              collaboraReady,
		CollaboraUnavailableReason:  collaboraReason,
		OnlyOfficeEnabled:           configValue.Editors.OnlyOfficeEnabled,
		OnlyOfficeUnavailableReason: "document_service_unavailable",
	})
	editorRegistry, err := editors.NewRegistry(editorProviders...)
	if err != nil {
		logger.Error("register document editor providers", "error", err)
		os.Exit(1)
	}
	editorResolver := editors.NewResolver(editorRegistry, editors.DefaultPolicy())
	if databasePool != nil {
		editorResolver.WithPolicyStore(editors.NewPostgresPolicyStore(databasePool))
	}
	editorService := editors.NewService(editorRegistry, editorResolver)
	if documentService != nil {
		documentService.SetEditorSelector(documents.EditorSelectorFunc(func(ctx context.Context, organizationID, family string) (string, error) {
			status, err := editorService.ResolveForOrganization(ctx, organizationID, editors.DocumentFamily(family), editors.CapabilityCreate)
			return string(status.ID), err
		}))
	}
	editorHandler := editors.NewHandler(editorService, sessionService)
	var administrationHandler *administration.Handler
	if databasePool != nil {
		administrationService := administration.NewService(databasePool, permissions.NewAuthorizer(databasePool), func(ctx context.Context, family, providerID string) error {
			status, err := editorService.Provider(ctx, editors.ProviderID(providerID))
			if err != nil {
				return err
			}
			for _, supported := range status.Families {
				if string(supported) == family {
					return nil
				}
			}
			return editors.ErrProviderUnavailable
		})
		administrationHandler = administration.NewHandler(administrationService)
	}
	var readiness httpserver.Readiness = httpserver.StaticReadiness(true)
	if databasePool != nil {
		readiness = httpserver.ReadinessFunc(func() bool {
			checkContext, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			return databasePool.Ping(checkContext) == nil
		})
	}

	server := &http.Server{
		Addr: configValue.HTTPAddress,
		Handler: httpserver.NewRouter(logger, httpserver.Routes{
			System:         system.NewHandler(systemService),
			Editors:        editorHandler,
			WOPI:           wopiHandler,
			Authenticator:  authenticator,
			AuthConfig:     map[string]any{"mode": configValue.Auth.Mode, "issuer": configValue.Auth.Issuer, "audience": configValue.Auth.Audience, "clientId": configValue.Auth.ClientID},
			Documents:      documentHandler,
			Administration: administrationHandler,
		}, readiness),
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
