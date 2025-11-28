// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
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

// NewFactory creates a new telemetry.Factory that uses otelconf
// to configure opentelemetry-go SDK telemetry providers.
func NewFactory() telemetry.Factory {
	return telemetry.NewFactory(
		createDefaultConfig,
		telemetry.WithCreateResource(createResource),
		telemetry.WithCreateLogger(createLogger),
		telemetry.WithCreateMeterProvider(createMeterProvider),
		telemetry.WithCreateTracerProvider(createTracerProvider),
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
			OutputPaths:        []string{"stderr"},
			ErrorOutputPaths:   []string{"stderr"},
			DisableCaller:      false,
			DisableStacktrace:  false,
			InitialFields:      map[string]any(nil),
			DisableZapResource: false,
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
