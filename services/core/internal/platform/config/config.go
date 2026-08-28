package config

import (
	"encoding/base64"
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
	Notifications     NotificationConfig
	Integrations      IntegrationConfig
	Secrets           SecretConfig
}

type AuthConfig struct {
	Mode, Issuer, DiscoveryURL, Audience, ClientID, DevelopmentSubject, DevelopmentEmail, DevelopmentName, PlatformGroupClaim string
	PlatformAdminGroups                                                                                                       []string
	DevelopmentPlatformAdmin                                                                                                  bool
}

type EditorConfig struct {
	TokenSecret         string
	CollaboraEnabled    bool
	CollaboraURL        string
	CollaboraPublicURL  string
	OnlyOfficeEnabled   bool
	OnlyOfficeURL       string
	OnlyOfficeJWTSecret string
}
type NotificationConfig struct {
	SMTPEnabled                                                 bool
	SMTPAddress, SMTPHost, SMTPUsername, SMTPPassword, SMTPFrom string
}
type IntegrationConfig struct {
	GoogleEnabled                                         bool
	GoogleClientID, GoogleClientSecret, GoogleRedirectURL string
}
type SecretConfig struct {
	MasterKey  []byte
	KeyVersion int
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
	smtpEnabled, err := booleanEnvironment("DOCUMI_SMTP_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	developmentPlatformAdmin, err := booleanEnvironment("DOCUMI_DEV_PLATFORM_ADMIN", false)
	if err != nil {
		return Config{}, err
	}
	googleEnabled, err := booleanEnvironment("GOOGLE_WORKSPACE_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	secretVersion, err := strconv.Atoi(envOrDefault("DOCUMI_SECRET_KEY_VERSION", "1"))
	if err != nil || secretVersion < 1 {
		return Config{}, fmt.Errorf("DOCUMI_SECRET_KEY_VERSION must be a positive integer")
	}
	var masterKey []byte
	if encoded := strings.TrimSpace(os.Getenv("DOCUMI_SECRET_MASTER_KEY")); encoded != "" {
		masterKey, err = base64.StdEncoding.DecodeString(encoded)
		if err != nil || len(masterKey) != 32 {
			return Config{}, fmt.Errorf("DOCUMI_SECRET_MASTER_KEY must be base64 for exactly 32 bytes")
		}
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
		Auth:              AuthConfig{Mode: envOrDefault("DOCUMI_AUTH_MODE", authMode), Issuer: os.Getenv("DOCUMI_OIDC_ISSUER"), DiscoveryURL: os.Getenv("DOCUMI_OIDC_DISCOVERY_URL"), Audience: envOrDefault("DOCUMI_OIDC_AUDIENCE", "documi-api"), ClientID: envOrDefault("DOCUMI_OIDC_CLIENT_ID", "documi-web"), DevelopmentSubject: envOrDefault("DOCUMI_DEV_SUBJECT", "local-admin"), DevelopmentEmail: envOrDefault("DOCUMI_DEV_EMAIL", "admin@documi.local"), DevelopmentName: envOrDefault("DOCUMI_DEV_NAME", "Local administrator"), PlatformGroupClaim: envOrDefault("DOCUMI_PLATFORM_GROUP_CLAIM", "groups"), PlatformAdminGroups: splitList(os.Getenv("DOCUMI_PLATFORM_ADMIN_GROUPS")), DevelopmentPlatformAdmin: developmentPlatformAdmin},
		Editors: EditorConfig{
			TokenSecret:         os.Getenv("DOCUMI_EDITOR_TOKEN_SECRET"),
			CollaboraEnabled:    collaboraEnabled,
			CollaboraURL:        os.Getenv("COLLABORA_URL"),
			CollaboraPublicURL:  os.Getenv("COLLABORA_PUBLIC_URL"),
			OnlyOfficeEnabled:   onlyOfficeEnabled,
			OnlyOfficeURL:       os.Getenv("ONLYOFFICE_URL"),
			OnlyOfficeJWTSecret: os.Getenv("ONLYOFFICE_JWT_SECRET"),
		},
		Notifications: NotificationConfig{SMTPEnabled: smtpEnabled, SMTPAddress: os.Getenv("DOCUMI_SMTP_ADDRESS"), SMTPHost: os.Getenv("DOCUMI_SMTP_HOST"), SMTPUsername: os.Getenv("DOCUMI_SMTP_USERNAME"), SMTPPassword: os.Getenv("DOCUMI_SMTP_PASSWORD"), SMTPFrom: os.Getenv("DOCUMI_SMTP_FROM")},
		Integrations:  IntegrationConfig{GoogleEnabled: googleEnabled, GoogleClientID: os.Getenv("GOOGLE_OAUTH_CLIENT_ID"), GoogleClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"), GoogleRedirectURL: os.Getenv("GOOGLE_OAUTH_REDIRECT_URL")},
		Secrets:       SecretConfig{MasterKey: masterKey, KeyVersion: secretVersion},
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
	if config.Notifications.SMTPEnabled && (strings.TrimSpace(config.Notifications.SMTPAddress) == "" || strings.TrimSpace(config.Notifications.SMTPHost) == "" || !strings.Contains(config.Notifications.SMTPFrom, "@")) {
		return Config{}, fmt.Errorf("DOCUMI SMTP address, host, and from are required when SMTP is enabled")
	}
	if config.Integrations.GoogleEnabled {
		if config.AirGapped {
			return Config{}, fmt.Errorf("Google Workspace cannot be enabled when AIR_GAPPED=true")
		}
		if strings.TrimSpace(config.Integrations.GoogleClientID) == "" || strings.TrimSpace(config.Integrations.GoogleClientSecret) == "" {
			return Config{}, fmt.Errorf("Google OAuth client ID and secret are required when Google Workspace is enabled")
		}
		if len(config.Secrets.MasterKey) != 32 {
			return Config{}, fmt.Errorf("DOCUMI_SECRET_MASTER_KEY is required when Google Workspace is enabled")
		}
		if err := requireHTTPURL("GOOGLE_OAUTH_REDIRECT_URL", config.Integrations.GoogleRedirectURL); err != nil {
			return Config{}, err
		}
	}
	return config, nil
}

func splitList(value string) []string {
	var out []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
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
