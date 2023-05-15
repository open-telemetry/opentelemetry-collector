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
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/collector/obsreport"
	semconv "go.opentelemetry.io/collector/semconv/v1.18.0"
	"go.opentelemetry.io/collector/service/telemetry"
)

const (
	// supported exporters
	consoleExporter      = "console"
	otlpExporter         = "otlp"
	protocolProtobufHTTP = "http/protobuf"
	protocolProtobufGRPC = "grpc/protobuf"
	compressionGzip      = "gzip"

	// supported metric readers
	PrometheusMetricReader = "prometheus"
	PeriodMetricReader     = "periodic"

	// gRPC Instrumentation Name
	GRPCInstrumentation = "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	// http Instrumentation Name
	HTTPInstrumentation = "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	// GRPCUnacceptableKeyValues is a list of high cardinality grpc attributes that should be filtered out.
	GRPCUnacceptableKeyValues = []attribute.KeyValue{
		attribute.String(semconv.AttributeNetSockPeerAddr, ""),
		attribute.String(semconv.AttributeNetSockPeerPort, ""),
		attribute.String(semconv.AttributeNetSockPeerName, ""),
	}

	// HTTPUnacceptableKeyValues is a list of high cardinality http attributes that should be filtered out.
	HTTPUnacceptableKeyValues = []attribute.KeyValue{
		attribute.String(semconv.AttributeNetHostName, ""),
		attribute.String(semconv.AttributeNetHostPort, ""),
	}
)

func toStringMap(in map[string]interface{}) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}

func newOTLPGRPCMetricExporter(ctx context.Context, args map[string]interface{}) (sdkmetric.Exporter, error) {
	opts := []otlpmetricgrpc.Option{}
	for k, v := range args {
		switch k {
		case "endpoint":
			opts = append(opts, otlpmetricgrpc.WithEndpoint(fmt.Sprintf("%s", v)))
		case "certificate":
		case "client_key":
		case "client_certificate":
		case "compression":
			opts = append(opts, otlpmetricgrpc.WithCompressor(fmt.Sprintf("%s", v)))
		case "timeout":
			timeout, ok := v.(int)
			if !ok {
				return nil, fmt.Errorf("invalid timeout for otlp exporter: %s", v)
			}
			opts = append(opts, otlpmetricgrpc.WithTimeout(time.Millisecond*time.Duration(timeout)))
		case "headers":
			headers, ok := v.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid headers for otlp exporter: %s", v)
			}
			opts = append(opts, otlpmetricgrpc.WithHeaders(toStringMap(headers)))
			// otlpmetricgrpc.WithInsecure()
			// otlpmetricgrpc.WithReconnectionPeriod()
			// otlpmetricgrpc.WithTLSCredentials()
			//
		}
	}
	return otlpmetricgrpc.New(ctx, opts...)
}

func newOTLPHTTPMetricExporter(ctx context.Context, args map[string]interface{}) (sdkmetric.Exporter, error) {
	opts := []otlpmetrichttp.Option{}
	for k, v := range args {
		switch k {
		case "endpoint":
			opts = append(opts, otlpmetrichttp.WithEndpoint(fmt.Sprintf("%s", v)))
		case "certificate":
		case "client_key":
		case "client_certificate":
		case "compression":
			switch fmt.Sprintf("%s", v) {
			case compressionGzip:
				opts = append(opts, otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression))
			}
		case "timeout":
			timeout, ok := v.(int)
			if !ok {
				return nil, fmt.Errorf("invalid timeout for otlp exporter: %s", v)
			}
			opts = append(opts, otlpmetrichttp.WithTimeout(time.Millisecond*time.Duration(timeout)))
		case "headers":
			headers, ok := v.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid headers for otlp exporter: %s", v)
			}
			opts = append(opts, otlpmetrichttp.WithHeaders(toStringMap(headers)))
			// otlpmetricgrpc.WithInsecure()
			// otlpmetricgrpc.WithReconnectionPeriod()
			// otlpmetricgrpc.WithTLSCredentials()
			//
		}
	}
	return otlpmetrichttp.New(ctx, opts...)
}

func InitExporter(ctx context.Context, exporterType string, args any) (sdkmetric.Exporter, error) {
	switch exporterType {
	case consoleExporter:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return stdoutmetric.New(
			stdoutmetric.WithEncoder(enc),
		)
	case otlpExporter:
		switch t := args.(type) {
		case map[string]interface{}:
			switch t["protocol"] {
			case protocolProtobufGRPC:
				return newOTLPGRPCMetricExporter(ctx, t)
			case protocolProtobufHTTP:
				return newOTLPHTTPMetricExporter(ctx, t)
			default:
				return nil, fmt.Errorf("unsupported protocol for otlp exporter: %s", t["protocol"])
			}
		default:
			return nil, fmt.Errorf("invalid args for otlp exporter: %v", args)
		}
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

func InitOpenTelemetry(res *resource.Resource, options []sdkmetric.Option, disableHighCardinality bool) (*sdkmetric.MeterProvider, error) {
	opts := []sdkmetric.Option{
		sdkmetric.WithResource(res),
		sdkmetric.WithView(batchViews(disableHighCardinality)...),
	}

	opts = append(opts, options...)
	return sdkmetric.NewMeterProvider(
		opts...,
	), nil
}

func batchViews(disableHighCardinality bool) []sdkmetric.View {
	views := []sdkmetric.View{
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
	if disableHighCardinality {
		views = append(views, sdkmetric.NewView(sdkmetric.Instrument{
			Scope: instrumentation.Scope{
				Name: GRPCInstrumentation,
			},
		}, sdkmetric.Stream{
			AttributeFilter: cardinalityFilter(GRPCUnacceptableKeyValues...),
		}))
		views = append(views, sdkmetric.NewView(sdkmetric.Instrument{
			Scope: instrumentation.Scope{
				Name: HTTPInstrumentation,
			},
		}, sdkmetric.Stream{
			AttributeFilter: cardinalityFilter(HTTPUnacceptableKeyValues...),
		}))
	}
	return views
}

func cardinalityFilter(kvs ...attribute.KeyValue) attribute.Filter {
	filter := attribute.NewSet(kvs...)
	return func(kv attribute.KeyValue) bool {
		return !filter.HasValue(kv.Key)
	}
}
