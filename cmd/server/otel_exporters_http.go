package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/your-org/your-project/internal/config"
)

func buildHTTPExporters(ctx context.Context, cfg config.Config) (exporterSet, error) {
	te, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.OTelEndpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return exporterSet{}, fmt.Errorf("create http trace exporter: %w", err)
	}
	me, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(cfg.OTelEndpoint),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return exporterSet{}, errors.Join(fmt.Errorf("create http metric exporter: %w", err), te.Shutdown(ctx))
	}
	le, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(cfg.OTelEndpoint),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		return exporterSet{}, errors.Join(fmt.Errorf("create http log exporter: %w", err), te.Shutdown(ctx), me.Shutdown(ctx))
	}
	return exporterSet{
		tracer: te,
		reader: sdkmetric.NewPeriodicReader(me, sdkmetric.WithInterval(cfg.OTelExportInterval)),
		logger: le,
	}, nil
}

// checkOTelHTTP verifies the HTTP endpoint is serving. An HTTP HEAD to /v1/traces
// confirms a real HTTP server is listening, not just that the port is open.
func checkOTelHTTP(endpoint string) error {
	client := &http.Client{Timeout: otelConnectTimeout}
	resp, err := client.Head("http://" + endpoint + "/v1/traces")
	if err != nil {
		return fmt.Errorf("otel http endpoint %q unreachable: %w", endpoint, err)
	}
	_ = resp.Body.Close()
	return nil
}
