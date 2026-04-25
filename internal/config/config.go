// Package config loads and validates runtime configuration from environment
// variables. All other packages receive a Config value; they must never call
// os.Getenv directly.
package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config holds all runtime configuration for the application, parsed from
// environment variables at startup. Add new fields here as the app grows.
type Config struct {
	Domain         string `env:"DOMAIN"          envDefault:"localhost"`
	FrontendOrigin string `env:"FRONTEND_ORIGIN"`
	Port           int    `env:"PORT"            envDefault:"8080"`
}

// Load parses the process environment into Config. Call once at startup and
// fail fast before serving traffic.
func Load() (Config, error) {
	return parse(env.Options{})
}

// LoadFrom parses cfg from an explicit map instead of the process environment.
// Use this in tests to avoid os.Setenv and the cleanup it requires.
func LoadFrom(vars map[string]string) (Config, error) {
	return parse(env.Options{Environment: vars})
}

//nolint:gocritic // hugeParam: Options is large but parse is called once at startup; copying is acceptable.
func parse(opts env.Options) (Config, error) {
	cfg, err := env.ParseAsWithOptions[Config](opts)
	if err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}
