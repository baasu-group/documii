package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment       string
	HTTPAddress       string
	LogLevel          string
	DatabaseURL       string
	StorageRoot       string
	TemporalAddress   string
	TemporalNamespace string
	AirGapped         bool
	ShutdownTimeout   time.Duration
	Editors           EditorConfig
	Auth              AuthConfig
	PublicURL         string
}

type AuthConfig struct{ Mode, Issuer, DiscoveryURL, Audience, ClientID, DevelopmentSubject, DevelopmentEmail, DevelopmentName string }

type EditorConfig struct {
	TokenSecret         string
	CollaboraEnabled    bool
	CollaboraURL        string
	CollaboraPublicURL  string
	OnlyOfficeEnabled   bool
	OnlyOfficeURL       string
	OnlyOfficeJWTSecret string
}

func Load() (Config, error) {
	environment := envOrDefault("DOCUMI_ENV", "development")
	authMode := "oidc"
	if environment == "development" || environment == "test" {
		authMode = "development"
	}
	airGapped, err := strconv.ParseBool(envOrDefault("AIR_GAPPED", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("parse AIR_GAPPED: %w", err)
	}

	collaboraEnabled, err := booleanEnvironment("COLLABORA_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	onlyOfficeEnabled, err := booleanEnvironment("ONLYOFFICE_ENABLED", false)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		Environment:       environment,
		HTTPAddress:       envOrDefault("DOCUMI_HTTP_ADDRESS", ":8080"),
		LogLevel:          envOrDefault("DOCUMI_LOG_LEVEL", "info"),
		DatabaseURL:       os.Getenv("DOCUMI_DATABASE_URL"),
		StorageRoot:       envOrDefault("DOCUMI_STORAGE_ROOT", "./.data/storage"),
		TemporalAddress:   envOrDefault("DOCUMI_TEMPORAL_ADDRESS", "localhost:7233"),
		TemporalNamespace: envOrDefault("DOCUMI_TEMPORAL_NAMESPACE", "default"),
		AirGapped:         airGapped,
		ShutdownTimeout:   10 * time.Second,
		PublicURL:         envOrDefault("DOCUMI_PUBLIC_URL", "http://localhost:8080"),
		Auth:              AuthConfig{Mode: envOrDefault("DOCUMI_AUTH_MODE", authMode), Issuer: os.Getenv("DOCUMI_OIDC_ISSUER"), DiscoveryURL: os.Getenv("DOCUMI_OIDC_DISCOVERY_URL"), Audience: envOrDefault("DOCUMI_OIDC_AUDIENCE", "documi-api"), ClientID: envOrDefault("DOCUMI_OIDC_CLIENT_ID", "documi-web"), DevelopmentSubject: envOrDefault("DOCUMI_DEV_SUBJECT", "local-admin"), DevelopmentEmail: envOrDefault("DOCUMI_DEV_EMAIL", "admin@documi.local"), DevelopmentName: envOrDefault("DOCUMI_DEV_NAME", "Local administrator")},
		Editors: EditorConfig{
			TokenSecret:         os.Getenv("DOCUMI_EDITOR_TOKEN_SECRET"),
			CollaboraEnabled:    collaboraEnabled,
			CollaboraURL:        os.Getenv("COLLABORA_URL"),
			CollaboraPublicURL:  os.Getenv("COLLABORA_PUBLIC_URL"),
			OnlyOfficeEnabled:   onlyOfficeEnabled,
			OnlyOfficeURL:       os.Getenv("ONLYOFFICE_URL"),
			OnlyOfficeJWTSecret: os.Getenv("ONLYOFFICE_JWT_SECRET"),
		},
	}

	if strings.TrimSpace(config.HTTPAddress) == "" {
		return Config{}, fmt.Errorf("DOCUMI_HTTP_ADDRESS must not be empty")
	}
	if strings.TrimSpace(config.TemporalAddress) == "" {
		return Config{}, fmt.Errorf("DOCUMI_TEMPORAL_ADDRESS must not be empty")
	}
	if err := requireHTTPURL("DOCUMI_PUBLIC_URL", config.PublicURL); err != nil {
		return Config{}, err
	}
	if err := config.Editors.Validate(); err != nil {
		return Config{}, err
	}
	if config.Auth.Mode != "oidc" && config.Auth.Mode != "development" {
		return Config{}, fmt.Errorf("DOCUMI_AUTH_MODE must be oidc or development")
	}
	if config.Auth.Mode == "development" && config.Environment != "development" && config.Environment != "test" {
		return Config{}, fmt.Errorf("development authentication is forbidden outside development/test")
	}
	if config.Environment == "production" && strings.TrimSpace(config.DatabaseURL) == "" {
		return Config{}, fmt.Errorf("DOCUMI_DATABASE_URL is required in production")
	}
	if config.Auth.Mode == "oidc" {
		if err := requireHTTPURL("DOCUMI_OIDC_ISSUER", config.Auth.Issuer); err != nil {
			return Config{}, err
		}
		if config.Auth.Audience == "" {
			return Config{}, fmt.Errorf("DOCUMI_OIDC_AUDIENCE is required")
		}
		if config.Auth.DiscoveryURL == "" {
			config.Auth.DiscoveryURL = config.Auth.Issuer
		}
		if err := requireHTTPURL("DOCUMI_OIDC_DISCOVERY_URL", config.Auth.DiscoveryURL); err != nil {
			return Config{}, err
		}
	}
	return config, nil
}

func (config EditorConfig) Validate() error {
	if config.CollaboraEnabled {
		if err := requireHTTPURL("COLLABORA_URL", config.CollaboraURL); err != nil {
			return err
		}
		if err := requireHTTPURL("COLLABORA_PUBLIC_URL", config.CollaboraPublicURL); err != nil {
			return err
		}
		if len(config.TokenSecret) < 32 {
			return fmt.Errorf("DOCUMI_EDITOR_TOKEN_SECRET must contain at least 32 characters when Collabora is enabled")
		}
	}
	if config.OnlyOfficeEnabled {
		if err := requireHTTPURL("ONLYOFFICE_URL", config.OnlyOfficeURL); err != nil {
			return err
		}
		if len(config.OnlyOfficeJWTSecret) < 32 {
			return fmt.Errorf("ONLYOFFICE_JWT_SECRET must contain at least 32 characters when ONLYOFFICE is enabled")
		}
	}
	return nil
}

func requireHTTPURL(name, value string) error {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("%s must be an absolute HTTP(S) URL", name)
	}
	return nil
}

func booleanEnvironment(name string, fallback bool) (bool, error) {
	value := envOrDefault(name, strconv.FormatBool(fallback))
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}

func envOrDefault(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}
