// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proctelemetry // import "go.opentelemetry.io/collector/service/internal/proctelemetry"

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/bridge/opencensus"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/collector/obsreport"
	semconv "go.opentelemetry.io/collector/semconv/v1.18.0"
	"go.opentelemetry.io/collector/service/telemetry"
)

const (

	// gRPC Instrumentation Name
	GRPCInstrumentation = "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	// http Instrumentation Name
	HTTPInstrumentation = "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	// supported metric readers
	pullMetricReader = "pull"

	// supported exporters
	prometheueExporter = "prometheus"
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

func InitMetricReader(ctx context.Context, name string, reader any, asyncErrorChannel chan error) (sdkmetric.Reader, *http.Server, error) {
	readerType := strings.Split(name, "/")[0]
	switch readerType {
	case pullMetricReader:
		r, ok := reader.(telemetry.PullMetricReader)
		if !ok {
			return nil, nil, fmt.Errorf("invalid metric reader configuration: %v", reader)
		}
		return initExporter(ctx, r.Exporter, asyncErrorChannel)
	default:
		return nil, nil, fmt.Errorf("unsupported metric reader type: %s", readerType)
	}
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

func InitPrometheusServer(registry *prometheus.Registry, address string, asyncErrorChannel chan error) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	server := &http.Server{
		Addr:    address,
		Handler: mux,
	}
	go func() {
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			asyncErrorChannel <- serveErr
		}
	}()
	return server
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

func initPrometheusExporter(prometheusConfig telemetry.Prometheus, asyncErrorChannel chan error) (sdkmetric.Reader, *http.Server, error) {
	promRegistry := prometheus.NewRegistry()
	if prometheusConfig.Host == nil {
		return nil, nil, fmt.Errorf("host must be specified")
	}
	if prometheusConfig.Port == nil {
		return nil, nil, fmt.Errorf("port must be specified")
	}
	wrappedRegisterer := prometheus.WrapRegistererWithPrefix("otelcol_", promRegistry)
	// We can remove `otelprom.WithoutUnits()` when the otel-go start exposing prometheus metrics using the OpenMetrics format
	// which includes metric units that prometheusreceiver uses to trim unit's suffixes from metric names.
	// https://github.com/open-telemetry/opentelemetry-go/issues/3468
	exporter, err := otelprom.New(
		otelprom.WithRegisterer(wrappedRegisterer),
		otelprom.WithoutUnits(),
		// Disabled for the moment until this becomes stable, and we are ready to break backwards compatibility.
		otelprom.WithoutScopeInfo())
	if err != nil {
		return nil, nil, fmt.Errorf("error creating otel prometheus exporter: %w", err)
	}

	exporter.RegisterProducer(opencensus.NewMetricProducer())
	return exporter, InitPrometheusServer(promRegistry, fmt.Sprintf("%s:%d", *prometheusConfig.Host, *prometheusConfig.Port), asyncErrorChannel), nil
}

func initExporter(_ context.Context, exporters telemetry.MetricExporter, asyncErrorChannel chan error) (sdkmetric.Reader, *http.Server, error) {
	for exporterType, exporter := range exporters {
		switch exporterType {
		case prometheueExporter:
			e, ok := exporter.(telemetry.Prometheus)
			if !ok {
				return nil, nil, fmt.Errorf("prometheus exporter invalid: %v", exporter)
			}
			return initPrometheusExporter(e, asyncErrorChannel)
		default:
			return nil, nil, fmt.Errorf("unsupported metric exporter type: %s", exporterType)
		}
	}
	return nil, nil, fmt.Errorf("no valid exporter: %v", exporters)
}
