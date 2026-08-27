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
