package main

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckOTelConnectivity(t *testing.T) {
	t.Run("grpc passes when endpoint is listening", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { _ = ln.Close() })

		require.NoError(t, checkOTelConnectivity(ln.Addr().String(), "grpc"))
	})

	t.Run("grpc fails when nothing is listening", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		addr := ln.Addr().String()
		_ = ln.Close()

		err = checkOTelConnectivity(addr, "grpc")
		require.Error(t, err)
		assert.ErrorContains(t, err, "unreachable")
	})

	t.Run("http passes when endpoint is listening", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(srv.Close)

		// srv.URL is "http://host:port" — strip the scheme for the endpoint.
		endpoint := srv.Listener.Addr().String()
		require.NoError(t, checkOTelConnectivity(endpoint, "http"))
	})

	t.Run("http fails when nothing is listening", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		addr := ln.Addr().String()
		_ = ln.Close()

		err = checkOTelConnectivity(addr, "http")
		require.Error(t, err)
		assert.ErrorContains(t, err, "unreachable")
	})
}

func TestBuildSlogHandler(t *testing.T) {
	// Use a discard handler as a stand-in for the OTel bridge.
	otelBridge := slog.NewTextHandler(io.Discard, nil)

	t.Run("dev mode returns the otel handler unchanged", func(t *testing.T) {
		handler, logger := buildSlogHandler("", otelBridge)
		assert.Equal(t, otelBridge, handler)
		assert.NotNil(t, logger)
	})

	t.Run("prod mode wraps in a multiHandler", func(t *testing.T) {
		handler, logger := buildSlogHandler("localhost:4317", otelBridge)
		_, ok := handler.(multiHandler)
		assert.True(t, ok, "expected multiHandler, got %T", handler)
		assert.NotNil(t, logger)
	})

	t.Run("fallback logger writes to stderr not otel bridge", func(t *testing.T) {
		// The returned logger must be backed by a JSONHandler (os.Stderr), not
		// the otel bridge, to avoid the log.std.mu re-entrant deadlock.
		_, logger := buildSlogHandler("localhost:4317", otelBridge)
		_, ok := logger.Handler().(*slog.JSONHandler)
		assert.True(t, ok, "fallback logger handler should be *slog.JSONHandler, got %T", logger.Handler())
	})
}
