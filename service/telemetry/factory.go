// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"time"

	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/featuregate"
)

var useLocalHostAsDefaultMetricsAddressFeatureGate = featuregate.GlobalRegistry().MustRegister(
	"telemetry.UseLocalHostAsDefaultMetricsAddress",
	featuregate.StageBeta,
	featuregate.WithRegisterFromVersion("v0.111.0"),
	featuregate.WithRegisterDescription("controls whether default Prometheus metrics server use localhost as the default host for their endpoints"),
)

// Settings holds configuration for building Telemetry.
type Settings struct {
	BuildInfo         component.BuildInfo
	AsyncErrorChannel chan error
	ZapOptions        []zap.Option
	SDK               *config.SDK
}

// Factory is factory interface for telemetry.
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
type Factory interface {
	// CreateDefaultConfig creates the default configuration for the telemetry.
	// TODO: Should we just inherit from component.Factory?
	CreateDefaultConfig() component.Config

	// CreateLogger creates a logger.
	CreateLogger(ctx context.Context, set Settings, cfg component.Config) (*zap.Logger, log.LoggerProvider, error)

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
		withLogger(func(_ context.Context, set Settings, cfg component.Config) (*zap.Logger, log.LoggerProvider, error) {
			c := *cfg.(*Config)
			return newLogger(set, c)
		}),
		withTracerProvider(func(_ context.Context, set Settings, cfg component.Config) (trace.TracerProvider, error) {
			c := *cfg.(*Config)
			return newTracerProvider(set, c)
		}),
		withMeterProvider(func(_ context.Context, set Settings, cfg component.Config) (metric.MeterProvider, error) {
			c := *cfg.(*Config)
			return newMeterProvider(set, c)
		}),
	)
}

func createDefaultConfig() component.Config {
	metricsHost := "localhost"
	if !useLocalHostAsDefaultMetricsAddressFeatureGate.IsEnabled() {
		metricsHost = ""
	}

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
			Level: configtelemetry.LevelNormal,
			MeterProvider: config.MeterProvider{
				Readers: []config.MetricReader{
					{
						Pull: &config.PullMetricReader{Exporter: config.PullMetricExporter{Prometheus: &config.Prometheus{
							WithoutScopeInfo:  newPtr(true),
							WithoutUnits:      newPtr(true),
							WithoutTypeSuffix: newPtr(true),
							Host:              &metricsHost,
							Port:              newPtr(8888),
							WithResourceConstantLabels: &config.IncludeExclude{
								Included: []string{},
							},
						}}},
					},
				},
			},
		},
	}
}

func newPtr[T int | string | bool](str T) *T {
	return &str
}
