package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

// White-box test with an injected filesystem: the real embedded ui.FS only
// contains built assets after `npm run build`, and the test suite must pass
// on a fresh clone where internal/ui/dist holds nothing but .gitkeep.
func TestRegisterStaticSPAFallback(t *testing.T) {
	fsys := fstest.MapFS{
		"dist/index.html": &fstest.MapFile{
			Data: []byte("<!doctype html><html><body>spa</body></html>"),
		},
	}

	e := echo.New()
	registerStatic(e, fsys)

	tests := []struct {
		name            string
		path            string
		wantContentType string
		wantStatus      int
	}{
		{
			name:            "root serves index",
			path:            "/",
			wantStatus:      http.StatusOK,
			wantContentType: "text/html",
		},
		{
			name:            "unknown frontend path falls back to SPA index",
			path:            "/some-spa-route",
			wantStatus:      http.StatusOK,
			wantContentType: "text/html",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, http.NoBody)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
			assert.Contains(t, rec.Header().Get("Content-Type"), tc.wantContentType)
		})
	}
}
