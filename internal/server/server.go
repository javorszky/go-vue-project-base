// Package server configures and runs the Echo HTTP server.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/your-org/your-project/internal/config"
)

// Server wraps the Echo instance and the address it will listen on.
type Server struct {
	echo *echo.Echo
	addr string
}

// New creates and configures a Server.
func New(cfg config.Config, gitSHA, buildTime string) *Server {
	e := echo.New()

	e.Use(middleware.Recover())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod:   true,
		LogURI:      true,
		LogStatus:   true,
		LogLatency:  true,
		HandleError: true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error == nil {
				slog.LogAttrs((*c).Request().Context(), slog.LevelInfo, "request",
					slog.String("method", v.Method),
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.Duration("latency", v.Latency),
				)
			} else {
				slog.LogAttrs((*c).Request().Context(), slog.LevelError, "request",
					slog.String("method", v.Method),
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.Duration("latency", v.Latency),
					slog.String("error", v.Error.Error()),
				)
			}
			return nil
		},
	}))

	// CORS is only needed in decoupled deployments where the frontend and
	// backend run on different origins. In embedded mode they share an origin
	// so no CORS headers are required.
	if cfg.FrontendOrigin != "" {
		e.Use(middleware.CORS(cfg.FrontendOrigin))
	}

	v1 := e.Group("/api/v1")
	v1.GET("/health", healthHandler)
	v1.GET("/status", statusHandler(gitSHA, buildTime))

	registerStatic(e)

	return &Server{echo: e, addr: fmt.Sprintf(":%d", cfg.Port)}
}

// Start runs the server until ctx is cancelled, then shuts down gracefully.
func (s *Server) Start(ctx context.Context) error {
	sc := echo.StartConfig{
		Address:         s.addr,
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
