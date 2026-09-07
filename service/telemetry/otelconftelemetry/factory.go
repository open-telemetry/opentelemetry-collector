// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"time"

	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/service/telemetry"
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
							WithoutScopeInfo:  new(true),
							WithoutUnits:      new(true),
							WithoutTypeSuffix: new(true),
							Host:              new("localhost"),
							Port:              new(8888),
						}}},
					},
				},
			},
		},
		Resource: ResourceConfig{
			Resource: config.Resource{
				SchemaUrl: new(semconv.SchemaURL),
			},
		},
	}
}
