// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestNewFactory_CreateDefaultConfig(t *testing.T) {
	var config component.Config = new(struct{})
	factory := NewFactory(func() component.Config { return config })
	require.NotNil(t, factory)
	assert.Equal(t, config, factory.CreateDefaultConfig())
}

func TestNewFactory_Defaults(t *testing.T) {
	factory := NewFactory(nil)
	require.NotNil(t, factory)

	res, err := factory.CreateResource(context.Background(), Settings{}, nil)
	require.NoError(t, err)
	assert.Equal(t, pcommon.NewResource(), res)

	logger, loggerShutdownFunc, err := factory.CreateLogger(context.Background(), LoggerSettings{}, nil)
	require.NoError(t, err)
	assert.Equal(t, zap.NewNop(), logger)
	assert.Nil(t, loggerShutdownFunc)

	meterProvider, err := factory.CreateMeterProvider(context.Background(), MeterSettings{}, nil)
	require.NoError(t, err)
	assert.Equal(t, noopMeterProvider{MeterProvider: noopmetric.NewMeterProvider()}, meterProvider)

	tracerProvider, err := factory.CreateTracerProvider(context.Background(), TracerSettings{}, nil)
	require.NoError(t, err)
	assert.Equal(t, noopTracerProvider{TracerProvider: nooptrace.NewTracerProvider()}, tracerProvider)
}

func TestNewFactory_Options(t *testing.T) {
	var called []string
	factory := NewFactory(nil, factoryOptionFunc(func(*factory) {
		called = append(called, "option1")
	}), factoryOptionFunc(func(*factory) {
		called = append(called, "option2")
	}))
	require.NotNil(t, factory)
	assert.Equal(t, []string{"option1", "option2"}, called)
}

func TestNewFactory_CreateResource(t *testing.T) {
	type contextKey struct{}
	var config component.Config = new(struct{})
	settings := Settings{
		BuildInfo: component.NewDefaultBuildInfo(),
	}
	ctx := context.WithValue(context.Background(), contextKey{}, 123)

	dummyResource := pcommon.NewResource()
	dummyResource.Attributes().PutStr("what", "ever")
	factory := NewFactory(nil, WithCreateResource(
		func(ctx context.Context, set Settings, cfg component.Config) (pcommon.Resource, error) {
			assert.Equal(t, 123, ctx.Value(contextKey{}))
			assert.Equal(t, settings, set)
			assert.Equal(t, config, cfg)
			return dummyResource, errors.New("not implemented")
		},
	))
	require.NotNil(t, factory)

	resource, err := factory.CreateResource(ctx, settings, config)
	require.EqualError(t, err, "not implemented")
	assert.Equal(t, dummyResource, resource)
}

func TestNewFactory_CreateLogger(t *testing.T) {
	type contextKey struct{}
	var config component.Config = new(struct{})
	settings := LoggerSettings{
		Settings: Settings{BuildInfo: component.NewDefaultBuildInfo()},
	}
	ctx := context.WithValue(context.Background(), contextKey{}, 123)

	shutdownCalled := false
	dummyLogger := new(zap.Logger)
	factory := NewFactory(nil, WithCreateLogger(
		func(ctx context.Context, set LoggerSettings, cfg component.Config) (*zap.Logger, component.ShutdownFunc, error) {
			assert.Equal(t, 123, ctx.Value(contextKey{}))
			assert.Equal(t, settings, set)
			assert.Equal(t, config, cfg)
			shutdownFunc := func(context.Context) error {
				shutdownCalled = true
				return nil
			}
			return dummyLogger, shutdownFunc, errors.New("not implemented")
		},
	))
	require.NotNil(t, factory)

	logger, shutdownFunc, err := factory.CreateLogger(ctx, settings, config)
	require.EqualError(t, err, "not implemented")
	assert.Equal(t, dummyLogger, logger)
	require.NoError(t, shutdownFunc(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestNewFactory_CreateMeterProvider(t *testing.T) {
	type contextKey struct{}
	var config component.Config = new(struct{})
	settings := MeterSettings{
		Settings: Settings{BuildInfo: component.NewDefaultBuildInfo()},
	}
	ctx := context.WithValue(context.Background(), contextKey{}, 123)

	var dummyMeterProvider struct{ MeterProvider }
	factory := NewFactory(nil, WithCreateMeterProvider(
		func(ctx context.Context, set MeterSettings, cfg component.Config) (MeterProvider, error) {
			assert.Equal(t, 123, ctx.Value(contextKey{}))
			assert.Equal(t, settings, set)
			assert.Equal(t, config, cfg)
			return &dummyMeterProvider, errors.New("not implemented")
		},
	))
	require.NotNil(t, factory)

	meterProvider, err := factory.CreateMeterProvider(ctx, settings, config)
	require.EqualError(t, err, "not implemented")
	assert.Equal(t, &dummyMeterProvider, meterProvider)
}

func TestNewFactory_CreateTracerProvider(t *testing.T) {
	type contextKey struct{}
	var config component.Config = new(struct{})
	settings := TracerSettings{
		Settings: Settings{BuildInfo: component.NewDefaultBuildInfo()},
	}
	ctx := context.WithValue(context.Background(), contextKey{}, 123)

	var dummyTracerProvider struct{ TracerProvider }
	factory := NewFactory(nil, WithCreateTracerProvider(
		func(ctx context.Context, set TracerSettings, cfg component.Config) (TracerProvider, error) {
			assert.Equal(t, 123, ctx.Value(contextKey{}))
			assert.Equal(t, settings, set)
			assert.Equal(t, config, cfg)
			return &dummyTracerProvider, errors.New("not implemented")
		},
	))
	require.NotNil(t, factory)

	tracerProvider, err := factory.CreateTracerProvider(ctx, settings, config)
	require.EqualError(t, err, "not implemented")
	assert.Equal(t, &dummyTracerProvider, tracerProvider)
}
