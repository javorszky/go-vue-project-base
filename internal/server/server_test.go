package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/your-org/your-project/internal/config"
	"github.com/your-org/your-project/internal/server"
)

func newHandler(frontendOrigin string) http.Handler {
	return server.New(config.Config{
		Domain:         "localhost",
		Port:           8080,
		FrontendOrigin: frontendOrigin,
	}, "", "").Handler()
}

func TestHealthEndpoint(t *testing.T) {
	tests := []struct {
		name   string
		method string
		want   int
	}{
		{name: "GET returns 200", method: http.MethodGet, want: http.StatusOK},
		{name: "POST returns 405", method: http.MethodPost, want: http.StatusMethodNotAllowed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/v1/health", http.NoBody)
			rec := httptest.NewRecorder()
			newHandler("").ServeHTTP(rec, req)

			assert.Equal(t, tc.want, rec.Code)
		})
	}
}

func TestHealthEndpoint_body(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", http.NoBody)
	rec := httptest.NewRecorder()
	newHandler("").ServeHTTP(rec, req)

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "ok", body["status"])
}

func TestStatusEndpoint(t *testing.T) {
	tests := []struct {
		name   string
		method string
		want   int
	}{
		{name: "GET returns 200", method: http.MethodGet, want: http.StatusOK},
		{name: "POST returns 405", method: http.MethodPost, want: http.StatusMethodNotAllowed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/v1/status", http.NoBody)
			rec := httptest.NewRecorder()
			newHandler("").ServeHTTP(rec, req)

			assert.Equal(t, tc.want, rec.Code)
		})
	}
}

func TestStatusEndpoint_body(t *testing.T) {
	h := server.New(config.Config{
		Domain: "localhost",
		Port:   8080,
	}, "abc123", "2026-04-25T12:00:00Z").Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", http.NoBody)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Status    string `json:"status"`
		GitSHA    string `json:"git_sha"`
		BuildTime string `json:"build_time"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "ok", body.Status)
	assert.Equal(t, "abc123", body.GitSHA)
	assert.Equal(t, "2026-04-25T12:00:00Z", body.BuildTime)
}

func TestCORSHeaders(t *testing.T) {
	const origin = "https://example.com"

	tests := []struct {
		name           string
		frontendOrigin string
		requestOrigin  string
		wantHeader     string
	}{
		{
			name:           "ACAO header set when origin matches configured origin",
			frontendOrigin: origin,
			requestOrigin:  origin,
			wantHeader:     origin,
		},
		{
			name:           "no ACAO header when FrontendOrigin not configured",
			frontendOrigin: "",
			requestOrigin:  "https://attacker.com",
			wantHeader:     "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/health", http.NoBody)
			req.Header.Set("Origin", tc.requestOrigin)
			rec := httptest.NewRecorder()
			newHandler(tc.frontendOrigin).ServeHTTP(rec, req)

			assert.Equal(t, tc.wantHeader, rec.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}
