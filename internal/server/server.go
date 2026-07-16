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
	"github.com/your-org/your-project/internal/ui"
)

// The timeout and size limits mirror deploy/Caddyfile (read_header 5s,
// read_body 30s, write 30s, idle 2m, max_header_size 64KB) so the Go server
// is equally protected when it is exposed directly in embedded mode, without
// Caddy in front.
const (
	bodyLimitBytes    = 10 * 1024 * 1024 // 10 MiB
	gracefulTimeout   = 10 * time.Second
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 2 * time.Minute
	maxHeaderBytes    = 64 * 1024 // 64 KiB
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

	// Security headers, mirroring the deploy/Caddyfile header block so
	// embedded-mode deployments (no Caddy in front) get the same protection.
	// Deliberately omitted here:
	//   - HSTS: this process serves plain HTTP; browsers ignore the header
	//     over http://, and Caddy sets it at the TLS edge.
	//   - CSP: a useful policy is app-specific and a template default of
	//     'self' breaks the first external font or API a consumer adds;
	//     set one in deploy/Caddyfile (or here) once the app's needs are known.
	//   - X-XSS-Protection: deprecated; modern browsers ignore it.
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		ContentTypeNosniff: "nosniff",
		XFrameOptions:      "DENY",
		ReferrerPolicy:     "strict-origin-when-cross-origin",
	}))
	// Permissions-Policy is not part of echo's SecureConfig; set it directly.
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			(*c).Response().Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
			return next(c)
		}
	})

	e.Use(otelMiddleware(cfg.ServiceName))
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

	e.Use(middleware.BodyLimit(bodyLimitBytes))

	// CORS is only needed in decoupled deployments where the frontend and
	// backend run on different origins. In embedded mode they share an origin
	// so no CORS headers are required.
	if cfg.FrontendOrigin != "" {
		e.Use(middleware.CORS(cfg.FrontendOrigin))
	}

	v1 := e.Group("/api/v1")
	v1.GET("/health", healthHandler)
	v1.GET("/status", statusHandler(gitSHA, buildTime))

	registerStatic(e, ui.FS)

	return &Server{echo: e, addr: fmt.Sprintf(":%d", cfg.Port)}
}

// Start runs the server until ctx is cancelled, then shuts down gracefully.
func (s *Server) Start(ctx context.Context) error {
	sc := echo.StartConfig{
		Address:         s.addr,
		GracefulTimeout: gracefulTimeout,
		BeforeServeFunc: func(srv *http.Server) error {
			srv.ReadHeaderTimeout = readHeaderTimeout
			srv.ReadTimeout = readTimeout
			srv.WriteTimeout = writeTimeout
			srv.IdleTimeout = idleTimeout
			srv.MaxHeaderBytes = maxHeaderBytes
			return nil
		},
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

// healthHandler is a liveness probe: it returns 200 as long as the process
// responds. It does not check dependencies. When the first backing service
// lands, split into /livez (always 200) and /readyz (checks deps) and
// deprecate this endpoint.
func healthHandler(c *echo.Context) error {
	if err := c.JSON(http.StatusOK, map[string]string{"status": "ok"}); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}
