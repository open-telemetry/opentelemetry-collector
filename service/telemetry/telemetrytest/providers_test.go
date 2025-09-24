// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetrytest // import "go.opentelemetry.io/collector/service/telemetry/telemetrytest"

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	logger := zap.NewNop()
	shutdownErr := errors.New("shutdown error")
	shutdown := func(context.Context) error {
		return shutdownErr
	}

	factory := telemetry.NewFactory(nil, WithLogger(logger, shutdown))
	createdLogger, createdShutdown, err := factory.CreateLogger(context.Background(), telemetry.LoggerSettings{}, nil)
	require.NoError(t, err)
	assert.Same(t, logger, createdLogger)
	assert.Same(t, shutdownErr, createdShutdown(t.Context()))
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
