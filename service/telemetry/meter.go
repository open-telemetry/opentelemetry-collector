// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"net"
	"strconv"

	"go.opentelemetry.io/contrib/config"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
)

func ptr[T any](v T) *T {
	return &v
}

func addressToReader(cfg Config) (*config.MetricReader, error) {
	host, port, err := net.SplitHostPort(cfg.Metrics.Address)
	if err != nil {
		return nil, err
	}
	portInt, err := strconv.Atoi(port)
	if err != nil {
		return nil, err
	}
	return &config.MetricReader{
		Pull: &config.PullMetricReader{
			Exporter: config.MetricExporter{
				Prometheus: &config.Prometheus{
					Host:              &host,
					Port:              &portInt,
					WithoutUnits:      ptr(true),
					WithoutScopeInfo:  ptr(true),
					WithoutTypeSuffix: ptr(true),
					WithResourceConstantLabels: &config.IncludeExclude{
						Excluded: []string{},
					},
				},
			},
		},
	}, nil
}

func batchViews() []config.View {
	views := []config.View{
		{
			Selector: &config.ViewSelector{
				InstrumentName: ptr("processor_batch_batch_send_size"),
			},
			Stream: &config.ViewStream{
				Aggregation: &config.ViewStreamAggregation{
					ExplicitBucketHistogram: &config.ViewStreamAggregationExplicitBucketHistogram{
						Boundaries: []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000, 100000},
					},
				},
			},
		},
		{
			Selector: &config.ViewSelector{
				InstrumentName: ptr("processor_batch_batch_send_size_bytes"),
			},
			Stream: &config.ViewStream{
				Aggregation: &config.ViewStreamAggregation{
					ExplicitBucketHistogram: &config.ViewStreamAggregationExplicitBucketHistogram{
						Boundaries: []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000,
							100_000, 200_000, 300_000, 400_000, 500_000, 600_000, 700_000, 800_000, 900_000,
							1000_000, 2000_000, 3000_000, 4000_000, 5000_000, 6000_000, 7000_000, 8000_000, 9000_000},
					},
				},
			},
		},
	}
	if obsreportconfig.DisableHighCardinalityMetricsfeatureGate.IsEnabled() {
		views = append(views, config.View{
			Selector: &config.ViewSelector{
				MeterName: ptr("go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"),
			},
			Stream: &config.ViewStream{
				// TODO: add excluded attributes filter
				AttributeKeys: []string{"service.instance_id", "service.name", "service.version"},
			},
		})
		views = append(views, config.View{
			Selector: &config.ViewSelector{
				MeterName: ptr("go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"),
			},
			Stream: &config.ViewStream{
				// TODO: add excluded attributes filter
				AttributeKeys: []string{"service.instance.id", "service.name", "service.version"},
			},
		})
	}
	return views
}

func newMeterProvider(ctx context.Context, set Settings, cfg Config) (metric.MeterProvider, func(context.Context) error, error) {
	if cfg.Metrics.Level == configtelemetry.LevelNone || (cfg.Metrics.Address == "" && len(cfg.Metrics.Readers) == 0) {
		return noop.NewMeterProvider(), func(context.Context) error { return nil }, nil
	}

	sch := semconv.SchemaURL
	res := config.Resource{
		SchemaUrl:  &sch,
		Attributes: attributes(set, cfg),
	}

	if len(cfg.Metrics.Address) != 0 {
		if cfg.Metrics.Readers == nil {
			cfg.Metrics.Readers = []config.MetricReader{}
		}
		r, err := addressToReader(cfg)
		if err != nil {
			return noop.NewMeterProvider(), nil, err
		}
		cfg.Metrics.Readers = append(cfg.Metrics.Readers, *r)
	}

	if cfg.Metrics.Views == nil {
		cfg.Metrics.Views = []config.View{}
	}
	cfg.Metrics.Views = append(cfg.Metrics.Views, batchViews()...)

	sdk, err := config.NewSDK(
		config.WithContext(ctx),
		config.WithOpenTelemetryConfiguration(
			config.OpenTelemetryConfiguration{
				Resource: &res,
				MeterProvider: &config.MeterProvider{
					Readers: cfg.Metrics.Readers,
					Views:   cfg.Metrics.Views,
				},
			}),
	)
	return sdk.MeterProvider(), sdk.Shutdown, err
}
