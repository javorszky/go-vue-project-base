package main

import (
	"fmt"

	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/your-org/your-project/internal/config"
)

func buildStdoutExporters(cfg config.Config) (exporterSet, error) {
	te, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return exporterSet{}, fmt.Errorf("create stdout trace exporter: %w", err)
	}
	me, err := stdoutmetric.New(stdoutmetric.WithPrettyPrint())
	if err != nil {
		return exporterSet{}, fmt.Errorf("create stdout metric exporter: %w", err)
	}
	le, err := stdoutlog.New()
	if err != nil {
		return exporterSet{}, fmt.Errorf("create stdout log exporter: %w", err)
	}
	return exporterSet{
		tracer: te,
		reader: sdkmetric.NewPeriodicReader(me, sdkmetric.WithInterval(cfg.OTelExportInterval)),
		logger: le,
	}, nil
}
