package server

import (
	"github.com/labstack/echo/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
)

// otelMiddleware creates a per-request span, extracts W3C trace context from
// incoming headers, and records the HTTP method, path, and response status.
// It is written for Echo v5 which uses *echo.Context (not the v4 interface).
// The otelecho contrib package targets Echo v4 and cannot be used here.
func otelMiddleware(serviceName string) echo.MiddlewareFunc {
	tracer := otel.Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := (*c).Request()
			ctx := propagator.Extract(req.Context(), propagation.HeaderCarrier(req.Header))

			spanName := req.Method + " " + (*c).Path()
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(req.Method),
					semconv.URLPath(req.URL.Path),
				),
			)
			defer span.End()

			(*c).SetRequest(req.WithContext(ctx))

			err := next(c)

			if resp, unwrapErr := echo.UnwrapResponse((*c).Response()); unwrapErr == nil {
				span.SetAttributes(semconv.HTTPResponseStatusCode(resp.Status))
			}
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}
			return err
		}
	}
}
