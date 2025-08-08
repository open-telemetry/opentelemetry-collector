// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"
	"time"

	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/service/telemetry"
)

var useLocalHostAsDefaultMetricsAddressFeatureGate = featuregate.GlobalRegistry().MustRegister(
	"telemetry.UseLocalHostAsDefaultMetricsAddress",
	featuregate.StageBeta,
	featuregate.WithRegisterFromVersion("v0.111.0"),
	featuregate.WithRegisterDescription("controls whether default Prometheus metrics server use localhost as the default host for their endpoints"),
)

// Factory is factory interface for telemetry.
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
//
// NOTE This API is experimental and will change soon - use at your own risk.
// See https://github.com/open-telemetry/opentelemetry-collector/issues/4970
type Factory interface {
	// CreateDefaultConfig creates the default configuration for the telemetry.
	// TODO: Should we just inherit from component.Factory?
	CreateDefaultConfig() component.Config

	// CreateLogger creates a logger.
	CreateLogger(context.Context, telemetry.Settings, component.Config) (*zap.Logger, log.LoggerProvider, error)

	// CreateTracerProvider creates a TracerProvider.
	CreateTracerProvider(context.Context, telemetry.Settings, component.Config) (trace.TracerProvider, error)

	// CreateMeterProvider creates a MeterProvider.
	CreateMeterProvider(context.Context, telemetry.Settings, component.Config) (metric.MeterProvider, error)

	// unexportedFactoryFunc is used to prevent external implementations of Factory.
	unexportedFactoryFunc()
}

// NewFactory creates a new Factory.
//
// NOTE This API is experimental and will change soon - use at your own risk.
// See https://github.com/open-telemetry/opentelemetry-collector/issues/4970
//
// TODO remove the parameters once the factory is fully self-contained
// and is responsible for creating the SDK and resource itself.
func NewFactory(sdk *config.SDK, res *resource.Resource) Factory {
	return newFactory(createDefaultConfig,
		withLogger(func(_ context.Context, set telemetry.Settings, cfg component.Config) (*zap.Logger, log.LoggerProvider, error) {
			c := *cfg.(*Config)
			return newLogger(set, c, sdk, res)
		}),
		withTracerProvider(func(_ context.Context, _ telemetry.Settings, cfg component.Config) (trace.TracerProvider, error) {
			c := *cfg.(*Config)
			return newTracerProvider(c, sdk)
		}),
		withMeterProvider(func(_ context.Context, _ telemetry.Settings, cfg component.Config) (metric.MeterProvider, error) {
			c := *cfg.(*Config)
			return newMeterProvider(c, sdk)
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
							WithoutScopeInfo:  ptr(true),
							WithoutUnits:      ptr(true),
							WithoutTypeSuffix: ptr(true),
							Host:              &metricsHost,
							Port:              ptr(8888),
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

func ptr[T any](v T) *T {
	return &v
}
