package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/your-org/your-project/internal/config"
)

const (
	defaultDomain      = "localhost"
	defaultServiceName = "your-project"
	defaultTransport   = "grpc"
	envPort            = "PORT"
	envSamplingRatio   = "OTEL_SAMPLING_RATIO"
	envExportInterval  = "OTEL_METRIC_EXPORT_INTERVAL"
)

func TestLoadFrom(t *testing.T) {
	tests := []struct {
		name    string
		vars    map[string]string
		want    config.Config
		wantErr bool
	}{
		{
			name: "defaults when map is empty",
			vars: map[string]string{},
			want: config.Config{Domain: defaultDomain, Port: 8080, ServiceName: defaultServiceName, OTelTransport: defaultTransport, OTelSamplingRatio: 1.0, OTelExportInterval: 15 * time.Second},
		},
		{
			name: "custom PORT",
			vars: map[string]string{envPort: "9090"},
			want: config.Config{Domain: defaultDomain, Port: 9090, ServiceName: defaultServiceName, OTelTransport: defaultTransport, OTelSamplingRatio: 1.0, OTelExportInterval: 15 * time.Second},
		},
		{
			name: "custom DOMAIN",
			vars: map[string]string{"DOMAIN": "example.com"},
			want: config.Config{Domain: "example.com", Port: 8080, ServiceName: defaultServiceName, OTelTransport: defaultTransport, OTelSamplingRatio: 1.0, OTelExportInterval: 15 * time.Second},
		},
		{
			name: "custom FRONTEND_ORIGIN",
			vars: map[string]string{"FRONTEND_ORIGIN": "https://frontend.example.com"},
			want: config.Config{Domain: defaultDomain, FrontendOrigin: "https://frontend.example.com", Port: 8080, ServiceName: defaultServiceName, OTelTransport: defaultTransport, OTelSamplingRatio: 1.0, OTelExportInterval: 15 * time.Second},
		},
		{
			name: "all custom values",
			vars: map[string]string{
				envPort:           "3000",
				"DOMAIN":          "api.example.com",
				"FRONTEND_ORIGIN": "https://app.example.com",
			},
			want: config.Config{Domain: "api.example.com", FrontendOrigin: "https://app.example.com", Port: 3000, ServiceName: defaultServiceName, OTelTransport: defaultTransport, OTelSamplingRatio: 1.0, OTelExportInterval: 15 * time.Second},
		},
		{
			name: "custom OTel endpoint and service name",
			vars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "collector:4317",
				"OTEL_SERVICE_NAME":           "my-svc",
			},
			want: config.Config{Domain: defaultDomain, Port: 8080, OTelEndpoint: "collector:4317", OTelTransport: defaultTransport, ServiceName: "my-svc", OTelSamplingRatio: 1.0, OTelExportInterval: 15 * time.Second},
		},
		{
			name: "custom OTel sampling ratio and export interval",
			vars: map[string]string{
				envSamplingRatio:  "0.25",
				envExportInterval: "30s",
			},
			want: config.Config{Domain: defaultDomain, Port: 8080, ServiceName: defaultServiceName, OTelTransport: defaultTransport, OTelSamplingRatio: 0.25, OTelExportInterval: 30 * time.Second},
		},
		{
			name: "http transport",
			vars: map[string]string{"OTEL_EXPORTER_OTLP_PROTOCOL": "http"},
			want: config.Config{Domain: defaultDomain, Port: 8080, ServiceName: defaultServiceName, OTelTransport: "http", OTelSamplingRatio: 1.0, OTelExportInterval: 15 * time.Second},
		},
		{
			name:    "invalid PORT returns error",
			vars:    map[string]string{envPort: "not-a-number"},
			wantErr: true,
		},
		{
			name:    "PORT zero is invalid",
			vars:    map[string]string{envPort: "0"},
			wantErr: true,
		},
		{
			name:    "PORT negative is invalid",
			vars:    map[string]string{envPort: "-1"},
			wantErr: true,
		},
		{
			name:    "PORT above 65535 is invalid",
			vars:    map[string]string{envPort: "65536"},
			wantErr: true,
		},
		{
			name:    "sampling ratio below 0 is invalid",
			vars:    map[string]string{envSamplingRatio: "-0.1"},
			wantErr: true,
		},
		{
			name:    "sampling ratio above 1 is invalid",
			vars:    map[string]string{envSamplingRatio: "1.1"},
			wantErr: true,
		},
		{
			name:    "zero export interval is invalid",
			vars:    map[string]string{envExportInterval: "0s"},
			wantErr: true,
		},
		{
			name:    "negative export interval is invalid",
			vars:    map[string]string{envExportInterval: "-1s"},
			wantErr: true,
		},
		{
			name:    "invalid transport is invalid",
			vars:    map[string]string{"OTEL_EXPORTER_OTLP_PROTOCOL": "udp"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := config.LoadFrom(tc.vars)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
