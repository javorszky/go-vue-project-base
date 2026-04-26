// Package main is the entry point for the server binary.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/your-org/your-project/internal/config"
	"github.com/your-org/your-project/internal/server"
)

// Injected at build time via -ldflags:
//
//	-X main.gitSHA=$(git rev-parse HEAD)
//	-X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)
var (
	gitSHA    = "unknown"
	buildTime = "unknown"
)

func main() {
	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.New(cfg, gitSHA, buildTime).Start(ctx); err != nil {
		return fmt.Errorf("run server: %w", err)
	}
	return nil
}
