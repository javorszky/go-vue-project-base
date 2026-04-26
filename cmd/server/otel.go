package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
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

const otelShutdownTimeout = 10 * time.Second

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

	// Save the pre-bridge logger so the shutdown closure can log lp's own
	// shutdown error after the provider (and therefore the bridge) is gone.
	preOTelLogger := slog.Default()
	slog.SetDefault(otelslog.NewLogger(cfg.ServiceName))

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
			preOTelLogger.Error("logger provider shutdown", "error", err)
		}
	}, nil
}

func buildTracerProvider(ctx context.Context, cfg config.Config, res *sdkresource.Resource) (*sdktrace.TracerProvider, error) {
	var exporter sdktrace.SpanExporter
	var err error
	if cfg.OTelEndpoint == "" {
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	} else {
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
	if cfg.OTelEndpoint == "" {
		exporter, err := stdoutmetric.New(stdoutmetric.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("create stdout metric exporter: %w", err)
		}
		reader = sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(cfg.OTelExportInterval))
	} else {
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

func buildLoggerProvider(ctx context.Context, cfg config.Config, res *sdkresource.Resource) (*sdklog.LoggerProvider, error) {
	var exporter sdklog.Exporter
	var err error
	if cfg.OTelEndpoint == "" {
		exporter, err = stdoutlog.New()
	} else {
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
