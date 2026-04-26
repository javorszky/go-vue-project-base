package main

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/your-org/your-project/internal/config"
)

func buildGRPCExporters(ctx context.Context, cfg config.Config) (exporterSet, error) {
	te, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OTelEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return exporterSet{}, fmt.Errorf("create grpc trace exporter: %w", err)
	}
	me, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(cfg.OTelEndpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return exporterSet{}, errors.Join(fmt.Errorf("create grpc metric exporter: %w", err), te.Shutdown(ctx))
	}
	le, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(cfg.OTelEndpoint),
		otlploggrpc.WithInsecure(),
	)
	if err != nil {
		return exporterSet{}, errors.Join(fmt.Errorf("create grpc log exporter: %w", err), te.Shutdown(ctx), me.Shutdown(ctx))
	}
	return exporterSet{
		tracer: te,
		reader: sdkmetric.NewPeriodicReader(me, sdkmetric.WithInterval(cfg.OTelExportInterval)),
		logger: le,
	}, nil
}

// checkOTelGRPC verifies the endpoint is accepting gRPC traffic, not merely
// that the port is open. It dials with h2c (insecure HTTP/2) and polls the
// connection state machine until Ready or TransientFailure.
//
// A plain TCP listener on the same port fails this check because the HTTP/2
// settings handshake never completes, leaving the client in Connecting until
// the deadline fires.
func checkOTelGRPC(endpoint string) error {
	conn, err := grpc.NewClient(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("otel grpc endpoint %q unreachable: %w", endpoint, err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), otelConnectTimeout)
	defer cancel()

	conn.Connect()
	for {
		s := conn.GetState()
		if s == connectivity.Ready {
			return nil
		}
		if s == connectivity.TransientFailure || s == connectivity.Shutdown {
			return fmt.Errorf("otel grpc endpoint %q unreachable: connection failed", endpoint)
		}
		if !conn.WaitForStateChange(ctx, s) {
			return fmt.Errorf("otel grpc endpoint %q unreachable: %w", endpoint, ctx.Err())
		}
	}
}
