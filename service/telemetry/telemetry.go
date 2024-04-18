// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"fmt"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/service/telemetry/internal"
)

// Telemetry is the service telemetry.
// Deprecated: [v0.99.0] Use Factory.
type Telemetry struct {
	logger         *zap.Logger
	tracerProvider trace.TracerProvider
}

func (t *Telemetry) TracerProvider() trace.TracerProvider {
	return t.tracerProvider
}

func (t *Telemetry) Logger() *zap.Logger {
	return t.logger
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	// TODO: Sync logger.
	if tp, ok := t.tracerProvider.(*sdktrace.TracerProvider); ok {
		return multierr.Combine(
			tp.Shutdown(ctx),
		)
	}
	// should this return an error?
	return nil
}

// Settings holds configuration for building Telemetry.
type Settings = internal.CreateSettings

// New creates a new Telemetry from Config.
// Deprecated: [v0.99.0] Use NewFactory.
func New(ctx context.Context, set Settings, cfg Config) (*Telemetry, error) {
	f := NewFactory()
	logger, err := f.CreateLogger(ctx, set, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to created logger: %w", err)
	}

	tracerProvider, err := f.CreateTracerProvider(ctx, set, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracer provider: %w", err)
	}

	return &Telemetry{
		logger:         logger,
		tracerProvider: tracerProvider,
	}, nil
}
