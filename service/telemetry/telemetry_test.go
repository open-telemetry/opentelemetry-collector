// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	nooplog "go.opentelemetry.io/otel/log/noop"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestNewFactory(t *testing.T) {
	type contextKey struct{}
	var config component.Config = new(struct{})
	settings := Settings{BuildInfo: component.NewDefaultBuildInfo()}
	ctx := context.WithValue(context.Background(), contextKey{}, 123)

	var providers struct {
		Providers
	}
	createProvidersFunc := func(ctx context.Context, set Settings, cfg component.Config) (Providers, error) {
		assert.Equal(t, 123, ctx.Value(contextKey{}))
		assert.Equal(t, settings, set)
		assert.Equal(t, config, cfg)
		return &providers, errors.New("not implemented")
	}
	factory := NewFactory(func() component.Config { return config }, createProvidersFunc)
	require.NotNil(t, factory)

	assert.Equal(t, config, factory.CreateDefaultConfig())
	result, err := factory.CreateProviders(ctx, settings, config)
	assert.Equal(t, &providers, result)
	assert.EqualError(t, err, "not implemented")
}

func TestNewFactory_Nop(t *testing.T) {
	// If the createProvidersFunc is nil, the factory will return
	// a Providers implementation with no-op providers.
	factory := NewFactory(nil, nil)
	require.NotNil(t, factory)

	providers, err := factory.CreateProviders(context.Background(), Settings{}, nil)
	require.NoError(t, err)
	require.NotNil(t, providers)

	assert.Equal(t, pcommon.NewResource(), providers.Resource())
	assert.Equal(t, zap.NewNop(), providers.Logger())
	assert.Equal(t, nooplog.NewLoggerProvider(), providers.LoggerProvider())
	assert.Equal(t, noopmetric.NewMeterProvider(), providers.MeterProvider())
	assert.Equal(t, nooptrace.NewTracerProvider(), providers.TracerProvider())
	assert.NoError(t, providers.Shutdown(context.Background()))
}

func TestNewFactory_Options(t *testing.T) {
	var called []string
	factory := NewFactory(nil, nil, factoryOptionFunc(func(*factory) {
		called = append(called, "option1")
	}), factoryOptionFunc(func(*factory) {
		called = append(called, "option2")
	}))
	require.NotNil(t, factory)
	assert.Equal(t, []string{"option1", "option2"}, called)
}
