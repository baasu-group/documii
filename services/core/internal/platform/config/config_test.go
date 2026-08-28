package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("AIR_GAPPED", "true")
	t.Setenv("DOCUMI_HTTP_ADDRESS", ":9090")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !got.AirGapped || got.HTTPAddress != ":9090" {
		t.Fatalf("Load() = %#v", got)
	}
}

func TestLoadRejectsEnabledCollaboraWithoutSecureConfiguration(t *testing.T) {
	t.Setenv("COLLABORA_ENABLED", "true")
	t.Setenv("COLLABORA_URL", "http://collabora:9980")
	t.Setenv("COLLABORA_PUBLIC_URL", "http://localhost:9980")
	t.Setenv("DOCUMI_EDITOR_TOKEN_SECRET", "short")

	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted a short editor token secret")
	}
}

func TestLoadAcceptsDisabledExternalEditorsWithoutSecrets(t *testing.T) {
	t.Setenv("COLLABORA_ENABLED", "false")
	t.Setenv("ONLYOFFICE_ENABLED", "false")
	t.Setenv("DOCUMI_EDITOR_TOKEN_SECRET", "")
	t.Setenv("ONLYOFFICE_JWT_SECRET", "")

	if _, err := Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadRejectsGoogleInAirGap(t *testing.T) {
	t.Setenv("AIR_GAPPED", "true")
	t.Setenv("GOOGLE_WORKSPACE_ENABLED", "true")
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "client")
	t.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "secret")
	t.Setenv("GOOGLE_OAUTH_REDIRECT_URL", "http://localhost/callback")
	if _, err := Load(); err == nil {
		t.Fatal("Load accepted Google Workspace in air-gapped mode")
	}
}
func TestLoadRejectsGoogleWithoutEncryptedSecretKey(t *testing.T) {
	t.Setenv("AIR_GAPPED", "false")
	t.Setenv("GOOGLE_WORKSPACE_ENABLED", "true")
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "client")
	t.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "secret")
	t.Setenv("GOOGLE_OAUTH_REDIRECT_URL", "http://localhost/callback")
	t.Setenv("DOCUMI_SECRET_MASTER_KEY", "")
	if _, err := Load(); err == nil {
		t.Fatal("Load accepted Google without SecretStore master key")
	}
}
