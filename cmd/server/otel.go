package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	logGlobal "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"github.com/your-org/your-project/internal/config"
)

const (
	otelShutdownTimeout = 10 * time.Second
	otelConnectTimeout  = 5 * time.Second
	otelTransportHTTP   = "http"
)

// setupOTel initialises the three OTel signal providers (trace, metric, log),
// registers them as globals, and bridges the global slog logger into the OTel
// log pipeline. It returns a shutdown function that flushes all providers.
// The returned shutdown uses context.WithoutCancel so it still runs after ctx
// is cancelled by the signal handler.
func setupOTel(ctx context.Context, cfg config.Config) (func(), error) {
	res, err := sdkresource.Merge(
		sdkresource.Default(),
		sdkresource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("build otel resource: %w", err)
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if cfg.OTelEndpoint != "" {
		if connectErr := checkOTelConnectivity(cfg.OTelEndpoint, cfg.OTelTransport); connectErr != nil {
			return nil, connectErr
		}
	}

	tp, err := buildTracerProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("build tracer provider: %w", err)
	}

	mp, err := buildMeterProvider(ctx, cfg, res)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("build meter provider: %w", err), tp.Shutdown(ctx))
	}

	lp, err := buildLoggerProvider(ctx, cfg, res)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("build logger provider: %w", err), tp.Shutdown(ctx), mp.Shutdown(ctx))
	}

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	logGlobal.SetLoggerProvider(lp)

	otelHandler := otelslog.NewHandler(cfg.ServiceName)
	handler, stderrLogger := buildSlogHandler(cfg.OTelEndpoint, otelHandler)
	slog.SetDefault(slog.New(handler))

	return func() {
		flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), otelShutdownTimeout)
		defer cancel()
		if err := tp.Shutdown(flushCtx); err != nil {
			slog.Error("tracer provider shutdown", "error", err)
		}
		if err := mp.Shutdown(flushCtx); err != nil {
			slog.Error("meter provider shutdown", "error", err)
		}
		if err := lp.Shutdown(flushCtx); err != nil {
			stderrLogger.Error("logger provider shutdown", "error", err)
		}
	}, nil
}

// buildSlogHandler returns the slog handler to install as the global default and
// a stderr-backed fallback logger for use after the OTel log provider shuts down.
//
// Must not use slog.Default().Handler() (defaultHandler) as the stderr fallback:
// slog.SetDefault installs the returned handler as log.Default()'s writer via
// handlerWriter, so defaultHandler → log.Default().output (acquires mu) →
// handlerWriter.Write → this handler → defaultHandler → re-acquire mu →
// self-deadlock on the same goroutine. JSONHandler writes to os.Stderr directly.
func buildSlogHandler(endpoint string, otelHandler slog.Handler) (slog.Handler, *slog.Logger) {
	stderrHandler := slog.NewJSONHandler(os.Stderr, nil)
	if endpoint != "" {
		// Prod: fan-out to stderr AND the OTel bridge. asyncHandler wraps the
		// bridge so a slow or unreachable collector cannot block the request path.
		return newMultiHandler(stderrHandler, newAsyncHandler(otelHandler)), slog.New(stderrHandler)
	}
	// Dev: OTel bridge only — the stdoutlog exporter writes to stdout and
	// is always available, so no fallback is needed.
	return otelHandler, slog.New(stderrHandler)
}

// checkOTelConnectivity verifies the OTLP endpoint is reachable before the
// server starts. The probe is transport-specific:
//   - grpc: TCP dial only — gRPC uses HTTP/2 framing over TCP, so a successful
//     TCP connection confirms the port is open. A non-gRPC service on the same
//     port would pass this check, but that misconfiguration surfaces quickly
//     when the first export attempt fails.
//   - http: HTTP HEAD to /v1/traces — confirms an HTTP server is listening and
//     responding, not just that the port is open.
func checkOTelConnectivity(endpoint, transport string) error {
	if transport == otelTransportHTTP {
		return checkOTelHTTP(endpoint)
	}
	return checkOTelTCP(endpoint)
}

func checkOTelTCP(endpoint string) error {
	conn, err := net.DialTimeout("tcp", endpoint, otelConnectTimeout)
	if err != nil {
		return fmt.Errorf("otel endpoint %q unreachable: %w", endpoint, err)
	}
	if err := conn.Close(); err != nil {
		return fmt.Errorf("close otel probe connection: %w", err)
	}
	return nil
}

func checkOTelHTTP(endpoint string) error {
	client := &http.Client{Timeout: otelConnectTimeout}
	resp, err := client.Head("http://" + endpoint + "/v1/traces")
	if err != nil {
		return fmt.Errorf("otel http endpoint %q unreachable: %w", endpoint, err)
	}
	_ = resp.Body.Close()
	return nil
}

func buildTracerProvider(ctx context.Context, cfg config.Config, res *sdkresource.Resource) (*sdktrace.TracerProvider, error) {
	var exporter sdktrace.SpanExporter
	var err error
	switch {
	case cfg.OTelEndpoint == "":
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	case cfg.OTelTransport == otelTransportHTTP:
		exporter, err = otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(cfg.OTelEndpoint),
			otlptracehttp.WithInsecure(),
		)
	default:
		exporter, err = otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(cfg.OTelEndpoint),
			otlptracegrpc.WithInsecure(),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}
	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.OTelSamplingRatio))),
	), nil
}

func buildMeterProvider(ctx context.Context, cfg config.Config, res *sdkresource.Resource) (*sdkmetric.MeterProvider, error) {
	var reader sdkmetric.Reader
	switch {
	case cfg.OTelEndpoint == "":
		exporter, err := stdoutmetric.New(stdoutmetric.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("create stdout metric exporter: %w", err)
		}
		reader = sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(cfg.OTelExportInterval))
	case cfg.OTelTransport == otelTransportHTTP:
		exporter, err := otlpmetrichttp.New(ctx,
			otlpmetrichttp.WithEndpoint(cfg.OTelEndpoint),
			otlpmetrichttp.WithInsecure(),
		)
		if err != nil {
			return nil, fmt.Errorf("create otlp metric exporter: %w", err)
		}
		reader = sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(cfg.OTelExportInterval))
	default:
		exporter, err := otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(cfg.OTelEndpoint),
			otlpmetricgrpc.WithInsecure(),
		)
		if err != nil {
			return nil, fmt.Errorf("create otlp metric exporter: %w", err)
		}
		reader = sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(cfg.OTelExportInterval))
	}
	return sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	), nil
}

// asyncLogBufSize is the number of log records the asyncHandler can queue
// before it starts dropping. Sized to absorb short bursts while the
// collector recovers without unbounded memory growth.
const asyncLogBufSize = 512

// asyncHandler wraps a slog.Handler and processes records off the hot path via
// a buffered channel and a single background goroutine. Handle does a
// non-blocking channel send and returns immediately; records are dropped (not
// queued indefinitely) when the buffer is full, so the caller is never held up
// even when the underlying handler or collector is slow or unreachable.
type asyncHandler struct {
	inner slog.Handler
	ch    chan slog.Record
}

func newAsyncHandler(h slog.Handler) *asyncHandler {
	a := &asyncHandler{inner: h, ch: make(chan slog.Record, asyncLogBufSize)}
	go func() {
		for r := range a.ch {
			if err := a.inner.Handle(context.Background(), r); err != nil {
				slog.Error("async log handler", "error", err)
			}
		}
	}()
	return a
}

func (a *asyncHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return a.inner.Enabled(ctx, level)
}

func (a *asyncHandler) Handle(ctx context.Context, r slog.Record) error { //nolint:gocritic // slog.Handler interface mandates this signature
	if !a.inner.Enabled(ctx, r.Level) {
		return nil
	}
	select {
	case a.ch <- r.Clone():
	default: // buffer full — drop rather than block the caller
	}
	return nil
}

func (a *asyncHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return newAsyncHandler(a.inner.WithAttrs(attrs))
}

func (a *asyncHandler) WithGroup(name string) slog.Handler {
	return newAsyncHandler(a.inner.WithGroup(name))
}

// multiHandler fans out slog records to multiple handlers sequentially.
// Handlers that must not block the caller (e.g. the OTel bridge) should be
// wrapped in asyncHandler before being passed here.
type multiHandler struct{ handlers []slog.Handler }

func newMultiHandler(handlers ...slog.Handler) multiHandler {
	return multiHandler{handlers: handlers}
}

func (m multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m multiHandler) Handle(ctx context.Context, r slog.Record) error { //nolint:gocritic // slog.Handler interface mandates this signature
	var errs []error
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			errs = append(errs, h.Handle(ctx, r.Clone()))
		}
	}
	return errors.Join(errs...)
}

func (m multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return multiHandler{handlers: handlers}
}

func (m multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return multiHandler{handlers: handlers}
}

func buildLoggerProvider(ctx context.Context, cfg config.Config, res *sdkresource.Resource) (*sdklog.LoggerProvider, error) {
	var exporter sdklog.Exporter
	var err error
	switch {
	case cfg.OTelEndpoint == "":
		exporter, err = stdoutlog.New()
	case cfg.OTelTransport == otelTransportHTTP:
		exporter, err = otlploghttp.New(ctx,
			otlploghttp.WithEndpoint(cfg.OTelEndpoint),
			otlploghttp.WithInsecure(),
		)
	default:
		exporter, err = otlploggrpc.New(ctx,
			otlploggrpc.WithEndpoint(cfg.OTelEndpoint),
			otlploggrpc.WithInsecure(),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("create log exporter: %w", err)
	}
	return sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	), nil
}
