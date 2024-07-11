// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/telemetry"

import (
	"context"
	"net"
	"strconv"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/contrib/config"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

func newMeterProvider(ctx context.Context, cfg Config) (metric.MeterProvider, error) {
	if cfg.Metrics.Level == configtelemetry.LevelNone || (cfg.Metrics.Address == "" && len(cfg.Metrics.Readers) == 0) {
		return noop.NewMeterProvider(), nil
	}
	if len(cfg.Metrics.Address) != 0 {
		host, port, err := net.SplitHostPort(cfg.Metrics.Address)
		if err != nil {
			return nil, err
		}
		portInt, err := strconv.Atoi(port)
		if err != nil {
			return nil, err
		}
		if cfg.Metrics.Readers == nil {
			cfg.Metrics.Readers = []config.MetricReader{}
		}
		cfg.Metrics.Readers = append(cfg.Metrics.Readers, config.MetricReader{
			Pull: &config.PullMetricReader{
				Exporter: config.MetricExporter{
					Prometheus: &config.Prometheus{
						Host:              &host,
						Port:              &portInt,
						WithoutScopeInfo:  ptr(true),
						WithoutTypeSuffix: ptr(true),
						WithoutUnits:      ptr(true),
						WithResourceConstantLabels: &config.IncludeExclude{
							Included: []string{"*"},
						},
					},
				},
			},
		})
	}
	sdk, err := config.NewSDK(
		config.WithContext(ctx),
		config.WithOpenTelemetryConfiguration(
			config.OpenTelemetryConfiguration{
				MeterProvider: &config.MeterProvider{
					Readers: cfg.Metrics.Readers,
				},
			},
		),
	)

	if err != nil {
		return nil, err
	}

	return sdk.MeterProvider(), nil
}

func ptr[T any](val T) *T {
	return &val
}
