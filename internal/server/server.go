// Package server configures and runs the Echo HTTP server.
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/your-org/your-project/internal/config"
)

// Server wraps the Echo instance and its configuration.
type Server struct {
	echo *echo.Echo
	cfg  config.Config
}

// New creates and configures a Server.
func New(cfg config.Config) *Server {
	e := echo.New()

	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())

	// CORS is only needed in decoupled deployments where the frontend and
	// backend run on different origins. In embedded mode they share an origin
	// so no CORS headers are required.
	if cfg.FrontendOrigin != "" {
		e.Use(middleware.CORS(cfg.FrontendOrigin))
	}

	v1 := e.Group("/api/v1")
	v1.GET("/health", healthHandler)

	registerStatic(e)

	return &Server{echo: e, cfg: cfg}
}

// Start runs the server until ctx is cancelled, then shuts down gracefully.
func (s *Server) Start(ctx context.Context) error {
	sc := echo.StartConfig{
		Address:         fmt.Sprintf(":%d", s.cfg.Port),
		GracefulTimeout: 10 * time.Second,
	}
	if err := sc.Start(ctx, s.echo); err != nil {
		return fmt.Errorf("start server: %w", err)
	}
	return nil
}

// Handler returns the underlying http.Handler, useful for testing routes
// without starting a real listener.
func (s *Server) Handler() http.Handler {
	return s.echo
}

func healthHandler(c *echo.Context) error {
	if err := c.JSON(http.StatusOK, map[string]string{"status": "ok"}); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}
