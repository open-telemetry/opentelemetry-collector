// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proctelemetry // import "go.opentelemetry.io/collector/service/internal/proctelemetry"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/service/telemetry"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/resource"
)

const (
	// supported trace propagators
	traceContextPropagator = "tracecontext"
	b3Propagator           = "b3"

	// supported exporters
	stdoutmetricExporter   = "console"
	otlpmetricgrpcExporter = "otlp"

	// supported metric readers
	PrometheusMetricReader = "prometheus"
	PeriodMetricReader     = "periodic"
)

func InitExporter(ctx context.Context, exporterType string, args any) (sdkmetric.Exporter, error) {
	switch exporterType {
	case stdoutmetricExporter:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return stdoutmetric.New(
			stdoutmetric.WithEncoder(enc),
		)
	case otlpmetricgrpcExporter:
		opts := []otlpmetricgrpc.Option{}

		// for k, v := range args {
		// 	switch k {
		// 	case "endpoint":
		// 		opts = append(opts, otlpmetricgrpc.WithEndpoint(v))

		// 	// otlpmetricgrpc.WithAggregationSelector()
		// 	// otlpmetricgrpc.WithCompressor()
		// 	// otlpmetricgrpc.WithDialOption()
		// 	// otlpmetricgrpc.WithGRPCConn()
		// 	// otlpmetricgrpc.WithInsecure()
		// 	// otlpmetricgrpc.WithReconnectionPeriod()
		// 	// otlpmetricgrpc.WithRetry()
		// 	// otlpmetricgrpc.WithServiceConfig()
		// 	// otlpmetricgrpc.WithTLSCredentials()
		// 	// otlpmetricgrpc.WithTemporalitySelector()
		// 	// otlpmetricgrpc.WithTimeout()
		// 	case "headers":
		// 		opts = append(opts, otlpmetricgrpc.WithHeaders(v))
		// 	}
		// }
		return otlpmetricgrpc.New(ctx, opts...)
	default:
		return nil, fmt.Errorf("unsupported metric exporter type: %s", exporterType)
	}
}

// InitReader initializes the metric reader from the configuration.
func InitPeriodicReader(ctx context.Context, reader telemetry.MetricReader) (sdkmetric.Reader, error) {
	if len(reader.Args.Exporter) == 0 {
		return nil, errors.New("no exporter configured")
	}

	for name, args := range reader.Args.Exporter {
		exp, err := InitExporter(ctx, name, args)
		if err != nil {
			return nil, err
		}
		return sdkmetric.NewPeriodicReader(exp), nil
	}
	return nil, errors.New("unexpected exporter configuration")
}

func InitOpenTelemetry(res *resource.Resource, options []sdkmetric.Option) (*sdkmetric.MeterProvider, error) {
	opts := []sdkmetric.Option{
		sdkmetric.WithResource(res),
		sdkmetric.WithView(batchViews()...),
	}

	opts = append(opts, options...)
	return sdkmetric.NewMeterProvider(
		opts...,
	), nil
}

func batchViews() []sdkmetric.View {
	return []sdkmetric.View{
		sdkmetric.NewView(
			sdkmetric.Instrument{Name: obsreport.BuildProcessorCustomMetricName("batch", "batch_send_size")},
			sdkmetric.Stream{Aggregation: aggregation.ExplicitBucketHistogram{
				Boundaries: []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000, 100000},
			}},
		),
		sdkmetric.NewView(
			sdkmetric.Instrument{Name: obsreport.BuildProcessorCustomMetricName("batch", "batch_send_size_bytes")},
			sdkmetric.Stream{Aggregation: aggregation.ExplicitBucketHistogram{
				Boundaries: []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000,
					100_000, 200_000, 300_000, 400_000, 500_000, 600_000, 700_000, 800_000, 900_000,
					1000_000, 2000_000, 3000_000, 4000_000, 5000_000, 6000_000, 7000_000, 8000_000, 9000_000},
			}},
		),
	}
}
