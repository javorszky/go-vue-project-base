package server

import (
	"net/http"
	"os"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// New creates and configures an Echo instance with middleware and routes.
func New() *echo.Echo {
	e := echo.New()

	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())

	// CORS is only needed in decoupled deployments where the frontend and
	// backend run on different origins. In embedded mode they share an origin
	// so no CORS headers are required.
	if origin := os.Getenv("FRONTEND_ORIGIN"); origin != "" {
		e.Use(middleware.CORS(origin))
	}

	v1 := e.Group("/api/v1")
	v1.GET("/health", healthHandler)

	registerStatic(e)

	return e
}

func healthHandler(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
