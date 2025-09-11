// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/service/telemetry"
)

type otelconfTelemetry struct {
	resource       pcommon.Resource
	logger         *zap.Logger
	loggerProvider telemetry.LoggerProvider
	meterProvider  telemetry.MeterProvider
	tracerProvider telemetry.TracerProvider
}

func createProviders(
	ctx context.Context, set telemetry.Settings, cfg component.Config,
) (_ telemetry.Providers, resultErr error) {
	res, err := createResource(ctx, set, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	logger, loggerProvider, err := createLogger(ctx, set, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger provider: %w", err)
	}
	defer func() {
		if resultErr != nil {
			// Log the error before shutting down the logger provider
			logger.Error("error creating telemetry providers", zap.Error(resultErr))
			resultErr = multierr.Append(resultErr, loggerProvider.Shutdown(ctx))
		}
	}()

	meterProvider, err := createMeterProvider(ctx, set, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create meter provider: %w", err)
	}
	defer func() {
		if resultErr != nil {
			resultErr = multierr.Append(resultErr, meterProvider.Shutdown(ctx))
		}
	}()

	tracerProvider, err := createTracerProvider(ctx, set, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracer provider: %w", err)
	}

	return &otelconfTelemetry{
		resource:       res,
		logger:         logger,
		loggerProvider: loggerProvider,
		meterProvider:  meterProvider,
		tracerProvider: tracerProvider,
	}, nil
}

func (t *otelconfTelemetry) Resource() pcommon.Resource {
	return t.resource
}

func (t *otelconfTelemetry) Logger() *zap.Logger {
	return t.logger
}

func (t *otelconfTelemetry) LoggerProvider() log.LoggerProvider {
	return t.loggerProvider
}

func (t *otelconfTelemetry) MeterProvider() metric.MeterProvider {
	return t.meterProvider
}

func (t *otelconfTelemetry) TracerProvider() trace.TracerProvider {
	return t.tracerProvider
}

func (t *otelconfTelemetry) Shutdown(ctx context.Context) error {
	var errs error
	if t.loggerProvider != nil {
		if err := t.loggerProvider.Shutdown(ctx); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("failed to shutdown logger provider: %w", err))
		}
		t.loggerProvider = nil
	}
	if t.meterProvider != nil {
		if err := t.meterProvider.Shutdown(ctx); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("failed to shutdown meter provider: %w", err))
		}
		t.meterProvider = nil
	}
	if t.tracerProvider != nil {
		if err := t.tracerProvider.Shutdown(ctx); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("failed to shutdown tracer provider: %w", err))
		}
		t.tracerProvider = nil
	}
	return errs
}
