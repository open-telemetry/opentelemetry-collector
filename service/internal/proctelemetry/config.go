// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proctelemetry // import "go.opentelemetry.io/collector/service/internal/proctelemetry"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/config"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/bridge/opencensus"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/collector/processor/processorhelper"
	semconv "go.opentelemetry.io/collector/semconv/v1.18.0"
)

const (

	// gRPC Instrumentation Name
	GRPCInstrumentation = "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	// http Instrumentation Name
	HTTPInstrumentation = "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	// supported protocols
	protocolProtobufHTTP = "http/protobuf"
	protocolProtobufGRPC = "grpc/protobuf"
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

	errNoValidMetricExporter = errors.New("no valid metric exporter")
	errNoValidSpanExporter   = errors.New("no valid span exporter")
)

func InitMetricReader(ctx context.Context, reader config.MetricReader, asyncErrorChannel chan error) (sdkmetric.Reader, *http.Server, error) {
	if reader.Pull != nil {
		return initPullExporter(reader.Pull.Exporter, asyncErrorChannel)
	}
	if reader.Periodic != nil {
		opts := []sdkmetric.PeriodicReaderOption{}
		if reader.Periodic.Interval != nil {
			opts = append(opts, sdkmetric.WithInterval(time.Duration(*reader.Periodic.Interval)*time.Millisecond))
		}

		if reader.Periodic.Timeout != nil {
			opts = append(opts, sdkmetric.WithTimeout(time.Duration(*reader.Periodic.Timeout)*time.Millisecond))
		}
		return initPeriodicExporter(ctx, reader.Periodic.Exporter, opts...)
	}
	return nil, nil, fmt.Errorf("unsupported metric reader type %v", reader)
}

func InitSpanProcessor(ctx context.Context, processor config.SpanProcessor) (sdktrace.SpanProcessor, error) {
	if processor.Batch != nil {
		if processor.Batch.Exporter.Console != nil {
			exp, err := stdouttrace.New(
				stdouttrace.WithPrettyPrint(),
			)
			if err != nil {
				return nil, err
			}
			return initBatchSpanProcessor(processor.Batch, exp)
		}
		if processor.Batch.Exporter.OTLP != nil {
			var err error
			var exp sdktrace.SpanExporter
			switch processor.Batch.Exporter.OTLP.Protocol {
			case protocolProtobufHTTP:
				exp, err = initOTLPHTTPSpanExporter(ctx, processor.Batch.Exporter.OTLP)
			case protocolProtobufGRPC:
				exp, err = initOTLPgRPCSpanExporter(ctx, processor.Batch.Exporter.OTLP)
			default:
				return nil, fmt.Errorf("unsupported protocol %q", processor.Batch.Exporter.OTLP.Protocol)
			}
			if err != nil {
				return nil, err
			}
			return initBatchSpanProcessor(processor.Batch, exp)
		}
		return nil, errNoValidSpanExporter
	}
	return nil, fmt.Errorf("unsupported span processor type %v", processor)
}

func InitTracerProvider(res *resource.Resource, options []sdktrace.TracerProviderOption) (*sdktrace.TracerProvider, error) {
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
	}

	opts = append(opts, options...)
	return sdktrace.NewTracerProvider(opts...), nil
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
			sdkmetric.Instrument{Name: processorhelper.BuildCustomMetricName("batch", "batch_send_size")},
			sdkmetric.Stream{Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
				Boundaries: []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000, 100000},
			}},
		),
		sdkmetric.NewView(
			sdkmetric.Instrument{Name: processorhelper.BuildCustomMetricName("batch", "batch_send_size_bytes")},
			sdkmetric.Stream{Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
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

func initPrometheusExporter(prometheusConfig *config.Prometheus, asyncErrorChannel chan error) (sdkmetric.Reader, *http.Server, error) {
	promRegistry := prometheus.NewRegistry()
	if prometheusConfig.Host == nil {
		return nil, nil, fmt.Errorf("host must be specified")
	}
	if prometheusConfig.Port == nil {
		return nil, nil, fmt.Errorf("port must be specified")
	}
	exporter, err := otelprom.New(
		otelprom.WithRegisterer(promRegistry),
		// https://github.com/open-telemetry/opentelemetry-collector/issues/8043
		otelprom.WithoutUnits(),
		// Disabled for the moment until this becomes stable, and we are ready to break backwards compatibility.
		otelprom.WithoutScopeInfo(),
		otelprom.WithProducer(opencensus.NewMetricProducer()),
		// This allows us to produce metrics that are backwards compatible w/ opencensus
		otelprom.WithoutCounterSuffixes(),
		otelprom.WithNamespace("otelcol"),
		otelprom.WithResourceAsConstantLabels(attribute.NewDenyKeysFilter()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating otel prometheus exporter: %w", err)
	}

	return exporter, InitPrometheusServer(promRegistry, fmt.Sprintf("%s:%d", *prometheusConfig.Host, *prometheusConfig.Port), asyncErrorChannel), nil
}

func initPullExporter(exporter config.MetricExporter, asyncErrorChannel chan error) (sdkmetric.Reader, *http.Server, error) {
	if exporter.Prometheus != nil {
		return initPrometheusExporter(exporter.Prometheus, asyncErrorChannel)
	}
	return nil, nil, errNoValidMetricExporter
}

func initPeriodicExporter(ctx context.Context, exporter config.MetricExporter, opts ...sdkmetric.PeriodicReaderOption) (sdkmetric.Reader, *http.Server, error) {
	if exporter.Console != nil {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		exp, err := stdoutmetric.New(
			stdoutmetric.WithEncoder(enc),
		)
		if err != nil {
			return nil, nil, err
		}
		return sdkmetric.NewPeriodicReader(exp, opts...), nil, nil
	}
	if exporter.OTLP != nil {
		var err error
		var exp sdkmetric.Exporter
		switch exporter.OTLP.Protocol {
		case protocolProtobufHTTP:
			exp, err = initOTLPHTTPExporter(ctx, exporter.OTLP)
		case protocolProtobufGRPC:
			exp, err = initOTLPgRPCExporter(ctx, exporter.OTLP)
		default:
			return nil, nil, fmt.Errorf("unsupported protocol %s", exporter.OTLP.Protocol)
		}
		if err != nil {
			return nil, nil, err
		}
		return sdkmetric.NewPeriodicReader(exp, opts...), nil, nil
	}
	return nil, nil, errNoValidMetricExporter
}

func normalizeEndpoint(endpoint string) string {
	if !strings.HasPrefix(endpoint, "https://") && !strings.HasPrefix(endpoint, "http://") {
		return fmt.Sprintf("http://%s", endpoint)
	}
	return endpoint
}

func initOTLPgRPCExporter(ctx context.Context, otlpConfig *config.OTLPMetric) (sdkmetric.Exporter, error) {
	opts := []otlpmetricgrpc.Option{}

	if len(otlpConfig.Endpoint) > 0 {
		u, err := url.ParseRequestURI(normalizeEndpoint(otlpConfig.Endpoint))
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlpmetricgrpc.WithEndpoint(u.Host))
		if u.Scheme == "http" {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
	}

	if otlpConfig.Compression != nil {
		switch *otlpConfig.Compression {
		case "gzip":
			opts = append(opts, otlpmetricgrpc.WithCompressor(*otlpConfig.Compression))
		case "none":
			break
		default:
			return nil, fmt.Errorf("unsupported compression %q", *otlpConfig.Compression)
		}
	}
	if otlpConfig.Timeout != nil {
		opts = append(opts, otlpmetricgrpc.WithTimeout(time.Millisecond*time.Duration(*otlpConfig.Timeout)))
	}
	if len(otlpConfig.Headers) > 0 {
		opts = append(opts, otlpmetricgrpc.WithHeaders(otlpConfig.Headers))
	}

	return otlpmetricgrpc.New(ctx, opts...)
}

func initOTLPHTTPExporter(ctx context.Context, otlpConfig *config.OTLPMetric) (sdkmetric.Exporter, error) {
	opts := []otlpmetrichttp.Option{}

	if len(otlpConfig.Endpoint) > 0 {
		u, err := url.ParseRequestURI(normalizeEndpoint(otlpConfig.Endpoint))
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlpmetrichttp.WithEndpoint(u.Host))

		if u.Scheme == "http" {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		if len(u.Path) > 0 {
			opts = append(opts, otlpmetrichttp.WithURLPath(u.Path))
		}
	}
	if otlpConfig.Compression != nil {
		switch *otlpConfig.Compression {
		case "gzip":
			opts = append(opts, otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression))
		case "none":
			opts = append(opts, otlpmetrichttp.WithCompression(otlpmetrichttp.NoCompression))
		default:
			return nil, fmt.Errorf("unsupported compression %q", *otlpConfig.Compression)
		}
	}
	if otlpConfig.Timeout != nil {
		opts = append(opts, otlpmetrichttp.WithTimeout(time.Millisecond*time.Duration(*otlpConfig.Timeout)))
	}
	if len(otlpConfig.Headers) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(otlpConfig.Headers))
	}

	return otlpmetrichttp.New(ctx, opts...)
}

func initOTLPgRPCSpanExporter(ctx context.Context, otlpConfig *config.OTLP) (sdktrace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{}

	if len(otlpConfig.Endpoint) > 0 {
		u, err := url.ParseRequestURI(normalizeEndpoint(otlpConfig.Endpoint))
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlptracegrpc.WithEndpoint(u.Host))
		if u.Scheme == "http" {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
	}

	if otlpConfig.Compression != nil {
		switch *otlpConfig.Compression {
		case "gzip":
			opts = append(opts, otlptracegrpc.WithCompressor(*otlpConfig.Compression))
		case "none":
			break
		default:
			return nil, fmt.Errorf("unsupported compression %q", *otlpConfig.Compression)
		}
	}
	if otlpConfig.Timeout != nil && *otlpConfig.Timeout > 0 {
		opts = append(opts, otlptracegrpc.WithTimeout(time.Millisecond*time.Duration(*otlpConfig.Timeout)))
	}
	if len(otlpConfig.Headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(otlpConfig.Headers))
	}

	return otlptracegrpc.New(ctx, opts...)
}

func initOTLPHTTPSpanExporter(ctx context.Context, otlpConfig *config.OTLP) (sdktrace.SpanExporter, error) {
	opts := []otlptracehttp.Option{}

	if len(otlpConfig.Endpoint) > 0 {
		u, err := url.ParseRequestURI(normalizeEndpoint(otlpConfig.Endpoint))
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlptracehttp.WithEndpoint(u.Host))

		if u.Scheme == "http" {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if len(u.Path) > 0 {
			opts = append(opts, otlptracehttp.WithURLPath(u.Path))
		}
	}
	if otlpConfig.Compression != nil {
		switch *otlpConfig.Compression {
		case "gzip":
			opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
		case "none":
			opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.NoCompression))
		default:
			return nil, fmt.Errorf("unsupported compression %q", *otlpConfig.Compression)
		}
	}
	if otlpConfig.Timeout != nil && *otlpConfig.Timeout > 0 {
		opts = append(opts, otlptracehttp.WithTimeout(time.Millisecond*time.Duration(*otlpConfig.Timeout)))
	}
	if len(otlpConfig.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(otlpConfig.Headers))
	}

	return otlptracehttp.New(ctx, opts...)
}

func initBatchSpanProcessor(bsp *config.BatchSpanProcessor, exp sdktrace.SpanExporter) (sdktrace.SpanProcessor, error) {
	opts := []sdktrace.BatchSpanProcessorOption{}
	if bsp.ExportTimeout != nil {
		if *bsp.ExportTimeout < 0 {
			return nil, fmt.Errorf("invalid export timeout %d", *bsp.ExportTimeout)
		}
		opts = append(opts, sdktrace.WithExportTimeout(time.Millisecond*time.Duration(*bsp.ExportTimeout)))
	}
	if bsp.MaxExportBatchSize != nil {
		if *bsp.MaxExportBatchSize < 0 {
			return nil, fmt.Errorf("invalid batch size %d", *bsp.MaxExportBatchSize)
		}
		opts = append(opts, sdktrace.WithMaxExportBatchSize(*bsp.MaxExportBatchSize))
	}
	if bsp.MaxQueueSize != nil {
		if *bsp.MaxQueueSize < 0 {
			return nil, fmt.Errorf("invalid queue size %d", *bsp.MaxQueueSize)
		}
		opts = append(opts, sdktrace.WithMaxQueueSize(*bsp.MaxQueueSize))
	}
	if bsp.ScheduleDelay != nil {
		if *bsp.ScheduleDelay < 0 {
			return nil, fmt.Errorf("invalid schedule delay %d", *bsp.ScheduleDelay)
		}
		opts = append(opts, sdktrace.WithBatchTimeout(time.Millisecond*time.Duration(*bsp.ScheduleDelay)))
	}
	return sdktrace.NewBatchSpanProcessor(exp, opts...), nil

}
