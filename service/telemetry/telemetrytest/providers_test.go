// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetrytest // import "go.opentelemetry.io/collector/service/telemetry/telemetrytest"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log"
	nooplog "go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/service/telemetry"
)

func TestWithResource(t *testing.T) {
	res := pcommon.NewResource()
	res.Attributes().PutStr("key", "value")

	factory := telemetry.NewFactory(nil, WithResource(res))
	createdResource, err := factory.CreateResource(context.Background(), telemetry.Settings{}, nil)
	require.NoError(t, err)
	assert.Equal(t, res, createdResource)
}

func TestWithLogger(t *testing.T) {
	test := func(t *testing.T, provider log.LoggerProvider, expectedProvider telemetry.LoggerProvider) {
		logger := zap.NewNop()
		factory := telemetry.NewFactory(nil, WithLogger(logger, provider))
		createdLogger, createdProvider, err := factory.CreateLogger(context.Background(), telemetry.LoggerSettings{}, nil)
		require.NoError(t, err)
		assert.Same(t, logger, createdLogger)
		assert.Equal(t, expectedProvider, createdProvider)
	}
	t.Run("Without Shutdown method", func(t *testing.T) {
		provider := nooplog.NewLoggerProvider()
		test(t, provider, ShutdownLoggerProvider{LoggerProvider: provider})
	})
	t.Run("With Shutdown method", func(t *testing.T) {
		provider := new(struct{ telemetry.LoggerProvider })
		test(t, provider, provider)
	})
}

func TestWithMeterProvider(t *testing.T) {
	test := func(t *testing.T, provider metric.MeterProvider, expected telemetry.MeterProvider) {
		factory := telemetry.NewFactory(nil, WithMeterProvider(provider))
		createdProvider, err := factory.CreateMeterProvider(context.Background(), telemetry.MeterSettings{}, nil)
		require.NoError(t, err)
		assert.Equal(t, expected, createdProvider)
	}
	t.Run("Without Shutdown method", func(t *testing.T) {
		provider := noopmetric.NewMeterProvider()
		test(t, provider, ShutdownMeterProvider{MeterProvider: provider})
	})
	t.Run("With Shutdown method", func(t *testing.T) {
		provider := new(struct{ telemetry.MeterProvider })
		test(t, provider, provider)
	})
}

func TestWithTracerProvider(t *testing.T) {
	test := func(t *testing.T, provider trace.TracerProvider, expected telemetry.TracerProvider) {
		factory := telemetry.NewFactory(nil, WithTracerProvider(provider))
		createdProvider, err := factory.CreateTracerProvider(context.Background(), telemetry.TracerSettings{}, nil)
		require.NoError(t, err)
		assert.Equal(t, expected, createdProvider)
	}
	t.Run("Without Shutdown method", func(t *testing.T) {
		provider := nooptrace.NewTracerProvider()
		test(t, provider, ShutdownTracerProvider{TracerProvider: provider})
	})
	t.Run("With Shutdown method", func(t *testing.T) {
		provider := new(struct{ telemetry.TracerProvider })
		test(t, provider, provider)
	})
}

/*
// WithMeterProvider returns a telemetry.FactoryOption that configures the
// factory's CreateMeterProvider method to return provider. If provider does
// not implement the Shutdown method, it will be wrapped with
// ShutdownMeterProvider with a no-op shutdown func.
func WithMeterProvider(provider metric.MeterProvider) telemetry.FactoryOption {
	return telemetry.WithCreateMeterProvider(
		func(context.Context, telemetry.MeterSettings, component.Config) (
			telemetry.MeterProvider, error,
		) {
			withShutdown, ok := provider.(telemetry.MeterProvider)
			if !ok {
				withShutdown = ShutdownMeterProvider{MeterProvider: provider}
			}
			return withShutdown, nil
		},
	)
}

// WithTracerProvider returns a telemetry.FactoryOption that configures the
// factory's CreateTracerProvider method to return provider. If provider does
// not implement the Shutdown method, it will be wrapped with
// ShutdownTracerProvider with a no-op shutdown func.
func WithTracerProvider(provider trace.TracerProvider) telemetry.FactoryOption {
	return telemetry.WithCreateTracerProvider(
		func(context.Context, telemetry.TracerSettings, component.Config) (
			telemetry.TracerProvider, error,
		) {
			withShutdown, ok := provider.(telemetry.TracerProvider)
			if !ok {
				withShutdown = ShutdownTracerProvider{TracerProvider: provider}
			}
			return withShutdown, nil
		},
	)
}
*/
