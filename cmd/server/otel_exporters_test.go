package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/your-org/your-project/internal/config"
)

// The OTLP exporters connect lazily, so construction is safe to test without
// a collector. These are smoke tests for the config→builder dispatch and for
// each builder returning a complete exporterSet.
func TestBuildExporters(t *testing.T) {
	tests := []struct {
		name       string
		wantTracer string
		cfg        config.Config
	}{
		{
			name:       "no endpoint dispatches to stdout",
			cfg:        config.Config{OTelExportInterval: 15 * time.Second},
			wantTracer: "*stdouttrace.Exporter",
		},
		{
			name: "http transport dispatches to the http builder",
			cfg: config.Config{
				OTelEndpoint:       "localhost:4318",
				OTelTransport:      "http",
				OTelExportInterval: 15 * time.Second,
			},
			wantTracer: "*otlptrace.Exporter",
		},
		{
			name: "grpc transport dispatches to the grpc builder",
			cfg: config.Config{
				OTelEndpoint:       "localhost:4317",
				OTelTransport:      "grpc",
				OTelExportInterval: 15 * time.Second,
			},
			wantTracer: "*otlptrace.Exporter",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			set, err := buildExporters(ctx, tc.cfg)
			require.NoError(t, err)

			assert.Equal(t, tc.wantTracer, fmt.Sprintf("%T", set.tracer))
			assert.NotNil(t, set.reader)
			assert.NotNil(t, set.logger)

			// Nothing was exported, so shutdown must succeed without a collector.
			assert.NoError(t, set.tracer.Shutdown(ctx))
			assert.NoError(t, set.reader.Shutdown(ctx))
			assert.NoError(t, set.logger.Shutdown(ctx))
		})
	}
}
