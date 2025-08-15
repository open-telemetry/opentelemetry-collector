// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"
	"time"

	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
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

// Factory is factory interface for telemetry providers.
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
//
// NOTE This API is experimental and will change soon - use at your own risk.
// See https://github.com/open-telemetry/opentelemetry-collector/issues/4970
type Factory interface {
	// CreateDefaultConfig creates the default configuration for the telemetry.
	CreateDefaultConfig() component.Config

	// CreateProviders creates telemetry providers.
	CreateProviders(context.Context, telemetry.Settings, component.Config) (telemetry.Providers, error)

	// unexportedFactoryFunc is used to prevent external implementations of Factory.
	unexportedFactoryFunc()
}

// NewFactory creates a new Factory.
//
// NOTE This API is experimental and will change soon - use at your own risk.
// See https://github.com/open-telemetry/opentelemetry-collector/issues/4970
func NewFactory() Factory {
	return newFactory(createDefaultConfig, createProviders)
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
