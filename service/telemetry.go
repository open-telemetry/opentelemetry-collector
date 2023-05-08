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

package service // import "go.opentelemetry.io/collector/service"

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	ocprom "contrib.go.opencensus.io/exporter/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	ocmetric "go.opencensus.io/metric"
	"go.opencensus.io/metric/metricproducer"
	"go.opencensus.io/stats/view"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/bridge/opencensus"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/obsreport"
	semconv "go.opentelemetry.io/collector/semconv/v1.18.0"
	"go.opentelemetry.io/collector/service/internal/proctelemetry"
	"go.opentelemetry.io/collector/service/telemetry"
)

const (
	zapKeyTelemetryAddress = "address"
	zapKeyTelemetryLevel   = "level"

	// supported trace propagators
	traceContextPropagator = "tracecontext"
	b3Propagator           = "b3"

	// gRPC Instrumentation Name
	grpcInstrumentation = "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	// http Instrumentation Name
	httpInstrumentation = "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	errUnsupportedPropagator = errors.New("unsupported trace propagator")

	// grpcUnacceptableKeyValues is a list of high cardinality grpc attributes that should be filtered out.
	grpcUnacceptableKeyValues = []attribute.KeyValue{
		attribute.String(semconv.AttributeNetSockPeerAddr, ""),
		attribute.String(semconv.AttributeNetSockPeerPort, ""),
		attribute.String(semconv.AttributeNetSockPeerName, ""),
	}

	// httpUnacceptableKeyValues is a list of high cardinality http attributes that should be filtered out.
	httpUnacceptableKeyValues = []attribute.KeyValue{
		attribute.String(semconv.AttributeNetHostName, ""),
		attribute.String(semconv.AttributeNetHostPort, ""),
	}
)

type telemetryInitializer struct {
	views      []*view.View
	ocRegistry *ocmetric.Registry
	mp         metric.MeterProvider
	servers    []*http.Server

	useOtel                bool
	disableHighCardinality bool
}

func newColTelemetry(useOtel bool, disableHighCardinality bool) *telemetryInitializer {
	return &telemetryInitializer{
		mp:                     noop.NewMeterProvider(),
		useOtel:                useOtel,
		disableHighCardinality: disableHighCardinality,
	}
}

func (tel *telemetryInitializer) init(ctx context.Context, res *resource.Resource, settings component.TelemetrySettings, cfg telemetry.Config, asyncErrorChannel chan error) error {
	if cfg.Metrics.Level == configtelemetry.LevelNone || (cfg.Metrics.Address == "" && len(cfg.Metrics.Readers) == 0) {
		settings.Logger.Info(
			"Skipping telemetry setup.",
			zap.String(zapKeyTelemetryAddress, cfg.Metrics.Address),
			zap.String(zapKeyTelemetryLevel, cfg.Metrics.Level.String()),
		)
		return nil
	}

	settings.Logger.Info("Setting up own telemetry...")

	if tp, err := textMapPropagatorFromConfig(cfg.Traces.Propagators); err == nil {
		otel.SetTextMapPropagator(tp)
	} else {
		return err
	}

	if !tel.useOtel {
		// when still using opencensus, return here as there's no need to configure readers below
		_, err := tel.initPrometheus(res, settings.Logger, cfg.Metrics.Address, cfg.Metrics.Level, asyncErrorChannel)
		return err
	}

	if len(cfg.Metrics.Address) > 0 {
		settings.Logger.Warn("service.telemetry.metrics.address is being deprecated in favour of service.telemetry.metrics.metric_readers")
		// TODO: account for multiple readers trying to use the same port
		host, port, err := net.SplitHostPort(cfg.Metrics.Address)
		if err != nil {
			return err
		}
		portInt, err := strconv.Atoi(port)
		if err != nil {
			return err
		}
		cfg.Metrics.Readers = append(cfg.Metrics.Readers, telemetry.MetricReader{
			Type: "prometheus",
			Args: telemetry.MetricReaderArgs{
				Host: &host,
				Port: &portInt,
			},
		})
	}
	readers := []sdkmetric.Option{}
	for _, reader := range cfg.Metrics.Readers {
		r, err := tel.initMetricReader(ctx, res, reader, settings.Logger, cfg, asyncErrorChannel)
		if err != nil {
			return err
		}
		readers = append(readers, sdkmetric.WithReader(r))
	}
	var err error
	tel.mp, err = proctelemetry.InitOpenTelemetry(res, readers)

	return err
}

func (tel *telemetryInitializer) initMetricReader(ctx context.Context, res *resource.Resource, reader telemetry.MetricReader, logger *zap.Logger, cfg telemetry.Config, asyncErrorChannel chan error) (sdkmetric.Reader, error) {
	switch reader.Type {
	case proctelemetry.PrometheusMetricReader:
		address := fmt.Sprintf("%s:%d", *reader.Args.Host, *reader.Args.Port)
		return tel.initPrometheus(res, logger, address, cfg.Metrics.Level, asyncErrorChannel)
	case proctelemetry.PeriodMetricReader:
		return proctelemetry.InitPeriodicReader(ctx, reader)
	}
	return nil, fmt.Errorf("unsupported metric reader type: %s", reader.Type)
}

func (tel *telemetryInitializer) initPrometheus(res *resource.Resource, logger *zap.Logger, address string, level configtelemetry.Level, asyncErrorChannel chan error) (sdkmetric.Reader, error) {
	promRegistry := prometheus.NewRegistry()
	if tel.useOtel {
		if err := tel.initOpenTelemetry(res, promRegistry); err != nil {
			return nil, err
		}
	} else {
		if err := tel.initOpenCensus(level, res, promRegistry); err != nil {
			return nil, err
		}
	}

	logger.Info(
		"Serving Prometheus metrics",
		zap.String(zapKeyTelemetryAddress, address),
		zap.String(zapKeyTelemetryLevel, level.String()),
	)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(promRegistry, promhttp.HandlerOpts{}))
	server := &http.Server{
		Addr:    address,
		Handler: mux,
	}
	tel.servers = append(tel.servers, server)
	go func() {
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			asyncErrorChannel <- serveErr
		}
	}()
	return otelprom.New(
		otelprom.WithRegisterer(promRegistry),
		otelprom.WithoutUnits(),
		// Disabled for the moment until this becomes stable, and we are ready to break backwards compatibility.
		otelprom.WithoutScopeInfo())
}

func (tel *telemetryInitializer) initOpenCensus(level configtelemetry.Level, res *resource.Resource, promRegistry *prometheus.Registry) error {
	tel.ocRegistry = ocmetric.NewRegistry()
	metricproducer.GlobalManager().AddProducer(tel.ocRegistry)

	tel.views = obsreportconfig.AllViews(level)
	if err := view.Register(tel.views...); err != nil {
		return err
	}

	// Until we can use a generic metrics exporter, default to Prometheus.
	opts := ocprom.Options{
		Namespace: "otelcol",
		Registry:  promRegistry,
	}

	opts.ConstLabels = make(map[string]string)
	for _, keyValue := range res.Attributes() {
		opts.ConstLabels[sanitizePrometheusKey(string(keyValue.Key))] = keyValue.Value.AsString()
	}

	pe, err := ocprom.NewExporter(opts)
	if err != nil {
		return err
	}

	view.RegisterExporter(pe)
	return nil
}

func (tel *telemetryInitializer) initOpenTelemetry(res *resource.Resource, promRegistry *prometheus.Registry) error {
	// Initialize the ocRegistry, still used by the process metrics.
	tel.ocRegistry = ocmetric.NewRegistry()
	metricproducer.GlobalManager().AddProducer(tel.ocRegistry)

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
		return fmt.Errorf("error creating otel prometheus exporter: %w", err)
	}

	exporter.RegisterProducer(opencensus.NewMetricProducer())
	views := batchViews()
	if tel.disableHighCardinality {
		views = append(views, sdkmetric.NewView(sdkmetric.Instrument{
			Scope: instrumentation.Scope{
				Name: grpcInstrumentation,
			},
		}, sdkmetric.Stream{
			AttributeFilter: cardinalityFilter(grpcUnacceptableKeyValues...),
		}))
		views = append(views, sdkmetric.NewView(sdkmetric.Instrument{
			Scope: instrumentation.Scope{
				Name: httpInstrumentation,
			},
		}, sdkmetric.Stream{
			AttributeFilter: cardinalityFilter(httpUnacceptableKeyValues...),
		}))
	}
	tel.mp = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(exporter),
		sdkmetric.WithView(views...),
	)

	return nil
}

func (tel *telemetryInitializer) shutdown() error {
	metricproducer.GlobalManager().DeleteProducer(tel.ocRegistry)
	view.Unregister(tel.views...)

	var errs error
	for _, server := range tel.servers {
		if server != nil {
			errs = multierr.Append(errs, server.Close())
		}
	}
	return errs
}

func cardinalityFilter(kvs ...attribute.KeyValue) attribute.Filter {
	filter := attribute.NewSet(kvs...)
	return func(kv attribute.KeyValue) bool {
		return !filter.HasValue(kv.Key)
	}
}

func sanitizePrometheusKey(str string) string {
	runeFilterMap := func(r rune) rune {
		if unicode.IsDigit(r) || unicode.IsLetter(r) || r == '_' {
			return r
		}
		return '_'
	}
	return strings.Map(runeFilterMap, str)
}

func textMapPropagatorFromConfig(props []string) (propagation.TextMapPropagator, error) {
	var textMapPropagators []propagation.TextMapPropagator
	for _, prop := range props {
		switch prop {
		case traceContextPropagator:
			textMapPropagators = append(textMapPropagators, propagation.TraceContext{})
		case b3Propagator:
			textMapPropagators = append(textMapPropagators, b3.New())
		default:
			return nil, errUnsupportedPropagator
		}
	}
	return propagation.NewCompositeTextMapPropagator(textMapPropagators...), nil
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
