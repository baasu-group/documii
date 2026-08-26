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
