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
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/service/internal/resource"
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
}

type Providers struct {
	Logger         *zap.Logger
	LoggerProvider log.LoggerProvider
	MeterProvider  metric.MeterProvider
	TracerProvider trace.TracerProvider
}

// Factory is factory interface for telemetry.
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
type Factory interface {
	// CreateDefaultConfig creates the default configuration for the telemetry.
	// TODO: Should we just inherit from component.Factory?
	CreateDefaultConfig() component.Config

	// CreateResource creates a pcommon.Resource representing the collector,
	// based on the provided settings and configuration. This may be used by
	// components in their internal telemetry.
	//
	// TODO what is the resource used for? Should it be injected into all the
	// internal telemetry, and be encapsulated?
	CreateResource(context.Context, Settings, component.Config) (pcommon.Resource, error)

	// CreateProviders creates the logger, meter, and tracer providers for
	// the collector's internal telemetry.
	//
	// TODO should CreateProviders be CreateTelemetrySettings instead,
	// and return a component.TelemetrySettings?
	CreateProviders(context.Context, Settings, component.Config) (Providers, error)

	// unexportedFactoryFunc is used to prevent external implementations of Factory.
	unexportedFactoryFunc()
}

// NewFactory creates a new Factory.
func NewFactory() Factory {
	return newFactory(createDefaultConfig,
		withResource(func(ctx context.Context, set Settings, cfg component.Config) (pcommon.Resource, error) {
			c := cfg.(*telemetryConfig)
			res := resource.New(set.BuildInfo, c.Resource)
			pcommonRes := pcommon.NewResource()
			for _, keyValue := range res.Attributes() {
				pcommonRes.Attributes().PutStr(string(keyValue.Key), keyValue.Value.AsString())
			}
			return pcommonRes, nil
		}),
		withProviders(func(ctx context.Context, set Settings, cfg component.Config) (Providers, error) {
			c := *cfg.(*telemetryConfig)
			return newProviders(ctx, set, c)
		}),
	)
}

func createDefaultConfig() component.Config {
	metricsHost := "localhost"
	if !useLocalHostAsDefaultMetricsAddressFeatureGate.IsEnabled() {
		metricsHost = ""
	}

	return &telemetryConfig{
		Logs: logsConfig{
			Level:       zapcore.InfoLevel,
			Development: false,
			Encoding:    "console",
			Sampling: &logsSamplingConfig{
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
		Metrics: metricsConfig{
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
