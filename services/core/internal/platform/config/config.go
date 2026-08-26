package config

import (
	"fmt"
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
}

func Load() (Config, error) {
	airGapped, err := strconv.ParseBool(envOrDefault("AIR_GAPPED", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("parse AIR_GAPPED: %w", err)
	}

	config := Config{
		Environment:       envOrDefault("DOCUMI_ENV", "development"),
		HTTPAddress:       envOrDefault("DOCUMI_HTTP_ADDRESS", ":8080"),
		LogLevel:          envOrDefault("DOCUMI_LOG_LEVEL", "info"),
		DatabaseURL:       os.Getenv("DOCUMI_DATABASE_URL"),
		StorageRoot:       envOrDefault("DOCUMI_STORAGE_ROOT", "./.data/storage"),
		TemporalAddress:   envOrDefault("DOCUMI_TEMPORAL_ADDRESS", "localhost:7233"),
		TemporalNamespace: envOrDefault("DOCUMI_TEMPORAL_NAMESPACE", "default"),
		AirGapped:         airGapped,
		ShutdownTimeout:   10 * time.Second,
	}

	if strings.TrimSpace(config.HTTPAddress) == "" {
		return Config{}, fmt.Errorf("DOCUMI_HTTP_ADDRESS must not be empty")
	}
	if strings.TrimSpace(config.TemporalAddress) == "" {
		return Config{}, fmt.Errorf("DOCUMI_TEMPORAL_ADDRESS must not be empty")
	}
	return config, nil
}

func envOrDefault(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}
