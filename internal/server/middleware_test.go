package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// installSpanRecorder swaps the global tracer provider for one backed by an
// in-memory recorder, restoring the original when the test ends. The
// middleware resolves its tracer through the global provider, so this must
// run before otelMiddleware() is called.
func installSpanRecorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	t.Cleanup(func() { otel.SetTracerProvider(previous) })
	return recorder
}

func newOtelTestServer(handler echo.HandlerFunc) *echo.Echo {
	e := echo.New()
	e.Use(otelMiddleware("test-service"))
	e.GET("/things/:id", handler)
	return e
}

func TestOtelMiddleware(t *testing.T) {
	t.Run("successful request records a server span with route pattern", func(t *testing.T) {
		recorder := installSpanRecorder(t)
		e := newOtelTestServer(func(c *echo.Context) error {
			return (*c).NoContent(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/things/42", http.NoBody)
		e.ServeHTTP(httptest.NewRecorder(), req)

		spans := recorder.Ended()
		require.Len(t, spans, 1)
		span := spans[0]

		// The span name must use the route pattern, not the raw URL —
		// per-ID span names would explode cardinality.
		assert.Equal(t, "GET /things/:id", span.Name())
		assert.Equal(t, trace.SpanKindServer, span.SpanKind())
		assert.Contains(t, span.Attributes(),
			attribute.Int("http.response.status_code", http.StatusNoContent))
		assert.Contains(t, span.Attributes(), attribute.String("url.path", "/things/42"))
		assert.Equal(t, codes.Unset, span.Status().Code)
	})

	t.Run("handler error marks the span as errored", func(t *testing.T) {
		recorder := installSpanRecorder(t)
		e := newOtelTestServer(func(*echo.Context) error {
			return errors.New("boom")
		})

		req := httptest.NewRequest(http.MethodGet, "/things/42", http.NoBody)
		e.ServeHTTP(httptest.NewRecorder(), req)

		spans := recorder.Ended()
		require.Len(t, spans, 1)
		span := spans[0]

		assert.Equal(t, codes.Error, span.Status().Code)
		require.Len(t, span.Events(), 1)
		assert.Equal(t, "exception", span.Events()[0].Name)
	})

	t.Run("incoming traceparent is adopted as parent", func(t *testing.T) {
		recorder := installSpanRecorder(t)
		// The global default propagator is a no-op; production wiring installs
		// TraceContext in setupOTel. Mirror that here, restoring afterwards.
		previous := otel.GetTextMapPropagator()
		otel.SetTextMapPropagator(propagation.TraceContext{})
		t.Cleanup(func() { otel.SetTextMapPropagator(previous) })

		e := newOtelTestServer(func(c *echo.Context) error {
			return (*c).NoContent(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/things/42", http.NoBody)
		req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
		e.ServeHTTP(httptest.NewRecorder(), req)

		spans := recorder.Ended()
		require.Len(t, spans, 1)
		assert.Equal(t,
			"4bf92f3577b34da6a3ce929d0e0e4736",
			spans[0].SpanContext().TraceID().String())
	})
}
