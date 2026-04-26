package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
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

// exporterSet groups the three signal exporters/readers produced by a single
// transport factory. reader is a pre-wrapped PeriodicReader so buildMeterProvider
// does not need to know the export interval.
type exporterSet struct {
	tracer sdktrace.SpanExporter
	reader sdkmetric.Reader
	logger sdklog.Exporter
}

// buildExporters is the single dispatch point for exporter construction.
// All transport-specific logic lives in the three otel_exporters_*.go files.
func buildExporters(ctx context.Context, cfg config.Config) (exporterSet, error) {
	switch {
	case cfg.OTelEndpoint == "":
		return buildStdoutExporters(cfg)
	case cfg.OTelTransport == otelTransportHTTP:
		return buildHTTPExporters(ctx, cfg)
	default:
		return buildGRPCExporters(ctx, cfg)
	}
}

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

	exporters, err := buildExporters(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("build exporters: %w", err)
	}

	tp := buildTracerProvider(exporters.tracer, res, cfg.OTelSamplingRatio)
	mp := buildMeterProvider(exporters.reader, res)
	lp := buildLoggerProvider(exporters.logger, res)

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

// checkOTelConnectivity dispatches to a transport-appropriate probe.
// grpc uses a protocol-level check (see otel_exporters_grpc.go);
// http uses an HTTP HEAD request (see otel_exporters_http.go).
func checkOTelConnectivity(endpoint, transport string) error {
	if transport == otelTransportHTTP {
		return checkOTelHTTP(endpoint)
	}
	return checkOTelGRPC(endpoint)
}

func buildTracerProvider(exporter sdktrace.SpanExporter, res *sdkresource.Resource, ratio float64) *sdktrace.TracerProvider {
	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))),
	)
}

func buildMeterProvider(reader sdkmetric.Reader, res *sdkresource.Resource) *sdkmetric.MeterProvider {
	return sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)
}

func buildLoggerProvider(exporter sdklog.Exporter, res *sdkresource.Resource) *sdklog.LoggerProvider {
	return sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)
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
