// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/service/internal/resource"
)

// disableHighCardinalityMetricsfeatureGate is the feature gate that controls whether the collector should enable
// potentially high cardinality metrics. The gate will be removed when the collector allows for view configuration.
var disableHighCardinalityMetricsfeatureGate = featuregate.GlobalRegistry().MustRegister(
	"telemetry.disableHighCardinalityMetrics",
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription("controls whether the collector should enable potentially high"+
		"cardinality metrics. The gate will be removed when the collector allows for view configuration."))

// Settings holds configuration for building Telemetry.
type Settings struct {
	BuildInfo         component.BuildInfo
	AsyncErrorChannel chan error
	ZapOptions        []zap.Option
}

// Factory is factory interface for telemetry.
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
type Factory interface {
	// CreateDefaultConfig creates the default configuration for the telemetry.
	// TODO: Should we just inherit from component.Factory?
	CreateDefaultConfig() component.Config

	// CreateLogger creates a logger.
	CreateLogger(ctx context.Context, set Settings, cfg component.Config) (*zap.Logger, error)

	// CreateTracerProvider creates a TracerProvider.
	CreateTracerProvider(ctx context.Context, set Settings, cfg component.Config) (trace.TracerProvider, error)

	// CreateMeterProvider creates a MeterProvider.
	CreateMeterProvider(ctx context.Context, set Settings, cfg component.Config) (metric.MeterProvider, error)

	// unexportedFactoryFunc is used to prevent external implementations of Factory.
	unexportedFactoryFunc()
}

// NewFactory creates a new Factory.
func NewFactory() Factory {
	return newFactory(createDefaultConfig,
		withLogger(func(_ context.Context, set Settings, cfg component.Config) (*zap.Logger, error) {
			c := *cfg.(*Config)
			return newLogger(c.Logs, set.ZapOptions)
		}),
		withTracerProvider(func(ctx context.Context, set Settings, cfg component.Config) (trace.TracerProvider, error) {
			c := *cfg.(*Config)
			return newTracerProvider(ctx, set, c)
		}),
		withMeterProvider(func(_ context.Context, set Settings, cfg component.Config) (metric.MeterProvider, error) {
			c := *cfg.(*Config)
			disableHighCard := disableHighCardinalityMetricsfeatureGate.IsEnabled()
			return newMeterProvider(
				meterProviderSettings{
					res:               resource.New(set.BuildInfo, c.Resource),
					cfg:               c.Metrics,
					asyncErrorChannel: set.AsyncErrorChannel,
				},
				disableHighCard,
			)
		}),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		Logs: LogsConfig{
			Level:       zapcore.InfoLevel,
			Development: false,
			Encoding:    "console",
			Sampling: &LogsSamplingConfig{
				Enabled:    true,
				Tick:       10 * time.Second,
				Initial:    10,
				Thereafter: 100,
			},
			OutputPaths:       []string{"stderr"},
			ErrorOutputPaths:  []string{"stderr"},
			DisableCaller:     false,
			DisableStacktrace: false,
			InitialFields:     map[string]any(nil),
		},
		Metrics: MetricsConfig{
			Level:   configtelemetry.LevelNormal,
			Address: ":8888",
		},
	}
}
