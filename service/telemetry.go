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
	"net/http"
	"strings"
	"sync"
	"unicode"

	ocprom "contrib.go.opencensus.io/exporter/prometheus"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	ocmetric "go.opencensus.io/metric"
	"go.opencensus.io/metric/metricproducer"
	"go.opencensus.io/stats/view"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	semconv "go.opentelemetry.io/collector/semconv/v1.5.0"
	"go.opentelemetry.io/collector/service/telemetry"
)

const (
	zapKeyTelemetryAddress = "address"
	zapKeyTelemetryLevel   = "level"

	// useOtelForInternalMetricsfeatureGateID is the feature gate ID that controls whether the collector uses open
	// telemetrySettings for internal metrics.
	useOtelForInternalMetricsfeatureGateID = "telemetry.useOtelForInternalMetrics"

	// supported trace propagators
	traceContextPropagator = "tracecontext"
	b3Propagator           = "b3"
)

var (
	errUnsupportedPropagator = errors.New("unsupported trace propagator")
)

type telemetryInitializer struct {
	registry *featuregate.Registry
	views    []*view.View

	ocRegistry *ocmetric.Registry
	mp         metric.MeterProvider

	server     *http.Server
	doInitOnce sync.Once
}

func init() {
	// register feature gate
	registerInternalMetricFeatureGate(featuregate.GetRegistry())
}

// registerInternalMetricFeatureGate registers the Internal Metric feature gate to the passed in registry
func registerInternalMetricFeatureGate(registry *featuregate.Registry) {
	registry.MustRegister(featuregate.Gate{
		ID:          useOtelForInternalMetricsfeatureGateID,
		Description: "controls whether the collector to uses OpenTelemetry for internal metrics",
		Enabled:     false,
	})
}

func newColTelemetry(registry *featuregate.Registry) *telemetryInitializer {
	return &telemetryInitializer{
		registry: registry,
		mp:       metric.NewNoopMeterProvider(),
	}
}

func (tel *telemetryInitializer) init(buildInfo component.BuildInfo, logger *zap.Logger, cfg telemetry.Config, asyncErrorChannel chan error) error {
	var err error
	tel.doInitOnce.Do(
		func() {
			err = tel.initOnce(buildInfo, logger, cfg, asyncErrorChannel)
		},
	)
	return err
}

func (tel *telemetryInitializer) initOnce(buildInfo component.BuildInfo, logger *zap.Logger, cfg telemetry.Config, asyncErrorChannel chan error) error {
	if cfg.Metrics.Level == configtelemetry.LevelNone || cfg.Metrics.Address == "" {
		logger.Info(
			"Skipping telemetry setup.",
			zap.String(zapKeyTelemetryAddress, cfg.Metrics.Address),
			zap.String(zapKeyTelemetryLevel, cfg.Metrics.Level.String()),
		)
		return nil
	}

	logger.Info("Setting up own telemetry...")

	// Construct telemetry attributes from build info and config's resource attributes.
	telAttrs := buildTelAttrs(buildInfo, cfg)

	if tp, err := textMapPropagatorFromConfig(cfg.Traces.Propagators); err == nil {
		otel.SetTextMapPropagator(tp)
	} else {
		return err
	}

	var pe http.Handler
	var err error
	if tel.registry.IsEnabled(useOtelForInternalMetricsfeatureGateID) {
		pe, err = tel.initOpenTelemetry(telAttrs)
	} else {
		pe, err = tel.initOpenCensus(cfg, telAttrs)
	}
	if err != nil {
		return err
	}

	logger.Info(
		"Serving Prometheus metrics",
		zap.String(zapKeyTelemetryAddress, cfg.Metrics.Address),
		zap.String(zapKeyTelemetryLevel, cfg.Metrics.Level.String()),
	)

	mux := http.NewServeMux()
	mux.Handle("/metrics", pe)

	tel.server = &http.Server{
		Addr:    cfg.Metrics.Address,
		Handler: mux,
	}

	go func() {
		if serveErr := tel.server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			asyncErrorChannel <- serveErr
		}
	}()

	return nil
}

func buildTelAttrs(buildInfo component.BuildInfo, cfg telemetry.Config) map[string]string {
	telAttrs := map[string]string{}

	for k, v := range cfg.Resource {
		// nil value indicates that the attribute should not be included in the telemetry.
		if v != nil {
			telAttrs[k] = *v
		}
	}

	if _, ok := cfg.Resource[semconv.AttributeServiceName]; !ok {
		// AttributeServiceName is not specified in the config. Use the default service name.
		telAttrs[semconv.AttributeServiceName] = buildInfo.Command
	}

	if _, ok := cfg.Resource[semconv.AttributeServiceInstanceID]; !ok {
		// AttributeServiceInstanceID is not specified in the config. Auto-generate one.
		instanceUUID, _ := uuid.NewRandom()
		instanceID := instanceUUID.String()
		telAttrs[semconv.AttributeServiceInstanceID] = instanceID
	}

	if _, ok := cfg.Resource[semconv.AttributeServiceVersion]; !ok {
		// AttributeServiceVersion is not specified in the config. Use the actual
		// build version.
		telAttrs[semconv.AttributeServiceVersion] = buildInfo.Version
	}

	return telAttrs
}

func (tel *telemetryInitializer) initOpenCensus(cfg telemetry.Config, telAttrs map[string]string) (http.Handler, error) {
	tel.ocRegistry = ocmetric.NewRegistry()
	metricproducer.GlobalManager().AddProducer(tel.ocRegistry)

	var views []*view.View
	obsMetrics := obsreportconfig.Configure(cfg.Metrics.Level)
	views = append(views, batchprocessor.MetricViews()...)
	views = append(views, obsMetrics.Views...)

	tel.views = views
	if err := view.Register(views...); err != nil {
		return nil, err
	}

	// Until we can use a generic metrics exporter, default to Prometheus.
	opts := ocprom.Options{
		Namespace: "otelcol",
	}

	opts.ConstLabels = make(map[string]string)

	for k, v := range telAttrs {
		opts.ConstLabels[sanitizePrometheusKey(k)] = v
	}

	pe, err := ocprom.NewExporter(opts)
	if err != nil {
		return nil, err
	}

	view.RegisterExporter(pe)
	return pe, nil
}

func (tel *telemetryInitializer) initOpenTelemetry(attrs map[string]string) (http.Handler, error) {
	// Initialize the ocRegistry, still used by the process metrics.
	tel.ocRegistry = ocmetric.NewRegistry()

	var resAttrs []attribute.KeyValue
	for k, v := range attrs {
		resAttrs = append(resAttrs, attribute.String(k, v))
	}

	res, err := resource.New(context.Background(), resource.WithAttributes(resAttrs...))
	if err != nil {
		return nil, fmt.Errorf("error creating otel resources: %w", err)
	}

	exporter := otelprom.New()
	tel.mp = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(exporter),
	)

	registry := prometheus.NewRegistry()

	wrappedRegisterer := prometheus.WrapRegistererWithPrefix("otelcol_", registry)
	if err := wrappedRegisterer.Register(exporter.Collector); err != nil {
		return nil, fmt.Errorf("failed to register prometheus collector: %w", err)
	}

	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{}), nil
}

func (tel *telemetryInitializer) shutdown() error {
	metricproducer.GlobalManager().DeleteProducer(tel.ocRegistry)

	view.Unregister(tel.views...)

	if tel.server != nil {
		return tel.server.Close()
	}

	return nil
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
