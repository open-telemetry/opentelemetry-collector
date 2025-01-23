// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelinit // import "go.opentelemetry.io/collector/service/telemetry/internal/otelinit"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/config"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc/credentials"

	semconv "go.opentelemetry.io/collector/semconv/v1.18.0"
)

const (

	// gRPC Instrumentation Name
	GRPCInstrumentation = "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	// http Instrumentation Name
	HTTPInstrumentation = "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	// supported protocols
	protocolProtobufHTTP     = "http/protobuf"
	protocolProtobufGRPC     = "grpc/protobuf"
	defaultReadHeaderTimeout = 10 * time.Second
)

var (
	// GRPCUnacceptableKeyValues is a list of high cardinality grpc attributes that should be filtered out.
	GRPCUnacceptableKeyValues = attribute.NewSet(
		attribute.String(semconv.AttributeNetSockPeerAddr, ""),
		attribute.String(semconv.AttributeNetSockPeerPort, ""),
		attribute.String(semconv.AttributeNetSockPeerName, ""),
	)

	// HTTPUnacceptableKeyValues is a list of high cardinality http attributes that should be filtered out.
	HTTPUnacceptableKeyValues = attribute.NewSet(
		attribute.String(semconv.AttributeNetHostName, ""),
		attribute.String(semconv.AttributeNetHostPort, ""),
	)

	errNoValidMetricExporter = errors.New("no valid metric exporter")
)

func InitMetricReader(ctx context.Context, reader config.MetricReader, asyncErrorChannel chan error, serverWG *sync.WaitGroup) (sdkmetric.Reader, *http.Server, error) {
	if reader.Pull != nil {
		return initPullExporter(reader.Pull.Exporter, asyncErrorChannel, serverWG)
	}
	if reader.Periodic != nil {
		var opts []sdkmetric.PeriodicReaderOption
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

func InitOpenTelemetry(res *resource.Resource, options []sdkmetric.Option, disableHighCardinality bool) (*sdkmetric.MeterProvider, error) {
	opts := []sdkmetric.Option{
		sdkmetric.WithResource(res),
		sdkmetric.WithView(disableHighCardinalityViews(disableHighCardinality)...),
	}

	opts = append(opts, options...)
	return sdkmetric.NewMeterProvider(
		opts...,
	), nil
}

func InitPrometheusServer(registry *prometheus.Registry, address string, asyncErrorChannel chan error, serverWG *sync.WaitGroup) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	server := &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
	}

	serverWG.Add(1)
	go func() {
		defer serverWG.Done()
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			select {
			case asyncErrorChannel <- serveErr:
			case <-time.After(1 * time.Second):
			}
		}
	}()
	return server
}

func disableHighCardinalityViews(disableHighCardinality bool) []sdkmetric.View {
	if !disableHighCardinality {
		return nil
	}
	return []sdkmetric.View{
		sdkmetric.NewView(
			sdkmetric.Instrument{Scope: instrumentation.Scope{Name: GRPCInstrumentation}},
			sdkmetric.Stream{
				AttributeFilter: cardinalityFilter(GRPCUnacceptableKeyValues),
			}),
		sdkmetric.NewView(
			sdkmetric.Instrument{Scope: instrumentation.Scope{Name: HTTPInstrumentation}},
			sdkmetric.Stream{
				AttributeFilter: cardinalityFilter(HTTPUnacceptableKeyValues),
			}),
	}
}

func cardinalityFilter(filter attribute.Set) attribute.Filter {
	return func(kv attribute.KeyValue) bool {
		return !filter.HasValue(kv.Key)
	}
}

func initPrometheusExporter(prometheusConfig *config.Prometheus, asyncErrorChannel chan error, serverWG *sync.WaitGroup) (sdkmetric.Reader, *http.Server, error) {
	promRegistry := prometheus.NewRegistry()
	if prometheusConfig.Host == nil {
		return nil, nil, errors.New("host must be specified")
	}
	if prometheusConfig.Port == nil {
		return nil, nil, errors.New("port must be specified")
	}

	opts := []otelprom.Option{
		otelprom.WithRegisterer(promRegistry),
		// https://github.com/open-telemetry/opentelemetry-collector/issues/8043
		otelprom.WithoutUnits(),
		// Disabled for the moment until this becomes stable, and we are ready to break backwards compatibility.
		otelprom.WithoutScopeInfo(),
		// This allows us to produce metrics that are backwards compatible w/ opencensus
		otelprom.WithoutCounterSuffixes(),
		otelprom.WithResourceAsConstantLabels(attribute.NewDenyKeysFilter()),
	}
	exporter, err := otelprom.New(opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating otel prometheus exporter: %w", err)
	}

	return exporter, InitPrometheusServer(promRegistry, net.JoinHostPort(*prometheusConfig.Host, strconv.Itoa(*prometheusConfig.Port)), asyncErrorChannel, serverWG), nil
}

func initPullExporter(exporter config.MetricExporter, asyncErrorChannel chan error, serverWG *sync.WaitGroup) (sdkmetric.Reader, *http.Server, error) {
	if exporter.Prometheus != nil {
		return initPrometheusExporter(exporter.Prometheus, asyncErrorChannel, serverWG)
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
		return "http://" + endpoint
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
		} else if otlpConfig.Certificate != nil {
			creds, err := credentials.NewClientTLSFromFile(*otlpConfig.Certificate, "")
			if err != nil {
				return nil, fmt.Errorf("could not create client tls credentials: %w", err)
			}
			opts = append(opts, otlpmetricgrpc.WithTLSCredentials(creds))
		}
	}

	if otlpConfig.Compression != nil {
		switch *otlpConfig.Compression {
		case "gzip":
			opts = append(opts, otlpmetricgrpc.WithCompressor(*otlpConfig.Compression))
		case "none":
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
	if otlpConfig.TemporalityPreference != nil {
		switch *otlpConfig.TemporalityPreference {
		case "delta":
			opts = append(opts, otlpmetricgrpc.WithTemporalitySelector(temporalityPreferenceDelta))
		case "cumulative":
			opts = append(opts, otlpmetricgrpc.WithTemporalitySelector(temporalityPreferenceCumulative))
		case "lowmemory":
			opts = append(opts, otlpmetricgrpc.WithTemporalitySelector(temporalityPreferenceLowMemory))
		default:
			return nil, fmt.Errorf("unsupported temporality preference %q", *otlpConfig.TemporalityPreference)
		}
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
	if otlpConfig.TemporalityPreference != nil {
		switch *otlpConfig.TemporalityPreference {
		case "delta":
			opts = append(opts, otlpmetrichttp.WithTemporalitySelector(temporalityPreferenceDelta))
		case "cumulative":
			opts = append(opts, otlpmetrichttp.WithTemporalitySelector(temporalityPreferenceCumulative))
		case "lowmemory":
			opts = append(opts, otlpmetrichttp.WithTemporalitySelector(temporalityPreferenceLowMemory))
		default:
			return nil, fmt.Errorf("unsupported temporality preference %q", *otlpConfig.TemporalityPreference)
		}
	}

	return otlpmetrichttp.New(ctx, opts...)
}

func temporalityPreferenceCumulative(_ sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func temporalityPreferenceDelta(ik sdkmetric.InstrumentKind) metricdata.Temporality {
	switch ik {
	case sdkmetric.InstrumentKindCounter, sdkmetric.InstrumentKindObservableCounter, sdkmetric.InstrumentKindHistogram:
		return metricdata.DeltaTemporality
	case sdkmetric.InstrumentKindObservableUpDownCounter, sdkmetric.InstrumentKindUpDownCounter:
		return metricdata.CumulativeTemporality
	default:
		return metricdata.DeltaTemporality
	}
}

func temporalityPreferenceLowMemory(ik sdkmetric.InstrumentKind) metricdata.Temporality {
	switch ik {
	case sdkmetric.InstrumentKindCounter, sdkmetric.InstrumentKindHistogram:
		return metricdata.DeltaTemporality
	case sdkmetric.InstrumentKindObservableCounter, sdkmetric.InstrumentKindObservableUpDownCounter, sdkmetric.InstrumentKindUpDownCounter:
		return metricdata.CumulativeTemporality
	default:
		return metricdata.DeltaTemporality
	}
}
