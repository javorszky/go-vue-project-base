package server

import (
	"io/fs"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// registerStatic serves the compiled Vue SPA from the given filesystem
// (ui.FS in production; tests inject an fstest.MapFS so they don't depend
// on a built frontend being present in internal/ui/dist).
//
// Phase 1 → Phase 2 migration: delete this file and remove its call in New().
// No other server code changes are needed.
func registerStatic(e *echo.Echo, fsys fs.FS) {
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		HTML5:      true,
		Root:       "dist",
		Filesystem: fsys,
		Skipper: func(c *echo.Context) bool {
			return strings.HasPrefix((*c).Request().URL.Path, "/api/")
		},
	}))
}
