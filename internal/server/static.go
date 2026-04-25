package server

import (
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/your-org/your-project/internal/ui"
)

// registerStatic serves the compiled Vue SPA via the embedded filesystem.
//
// Phase 1 → Phase 2 migration: delete this file and remove its call in New().
// No other server code changes are needed.
func registerStatic(e *echo.Echo) {
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		HTML5:      true,
		Root:       "dist",
		Filesystem: ui.FS,
	}))
}
