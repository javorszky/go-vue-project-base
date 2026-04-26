// Package config loads and validates runtime configuration from environment
// variables. All other packages receive a Config value; they must never call
// os.Getenv directly.
package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config holds all runtime configuration for the application, parsed from
// environment variables at startup. Add new fields here as the app grows.
type Config struct {
	Domain             string        `env:"DOMAIN"                        envDefault:"localhost"`
	FrontendOrigin     string        `env:"FRONTEND_ORIGIN"`
	OTelEndpoint       string        `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	OTelTransport      string        `env:"OTEL_EXPORTER_OTLP_PROTOCOL"   envDefault:"grpc"`
	ServiceName        string        `env:"OTEL_SERVICE_NAME"             envDefault:"hoplink"`
	OTelExportInterval time.Duration `env:"OTEL_METRIC_EXPORT_INTERVAL"   envDefault:"15s"`
	OTelSamplingRatio  float64       `env:"OTEL_SAMPLING_RATIO"           envDefault:"1.0"`
	Port               int           `env:"PORT"                          envDefault:"8080"`
}

// Load parses the process environment into Config. Call once at startup and
// fail fast before serving traffic.
func Load() (Config, error) {
	return parse(&env.Options{})
}

// LoadFrom parses cfg from an explicit map instead of the process environment.
// Use this in tests to avoid os.Setenv and the cleanup it requires.
func LoadFrom(vars map[string]string) (Config, error) {
	return parse(&env.Options{Environment: vars})
}

func (c Config) validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("config: PORT must be between 1 and 65535, got %d", c.Port)
	}
	if c.OTelSamplingRatio < 0 || c.OTelSamplingRatio > 1 {
		return fmt.Errorf("config: OTEL_SAMPLING_RATIO must be between 0.0 and 1.0, got %g", c.OTelSamplingRatio)
	}
	if c.OTelExportInterval <= 0 {
		return fmt.Errorf("config: OTEL_METRIC_EXPORT_INTERVAL must be positive, got %s", c.OTelExportInterval)
	}
	if c.OTelTransport != "grpc" && c.OTelTransport != "http" {
		return fmt.Errorf("config: OTEL_EXPORTER_OTLP_PROTOCOL must be 'grpc' or 'http', got %q", c.OTelTransport)
	}
	return nil
}

func parse(opts *env.Options) (Config, error) {
	cfg, err := env.ParseAsWithOptions[Config](*opts)
	if err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
