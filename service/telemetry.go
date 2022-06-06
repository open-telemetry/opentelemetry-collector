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
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"unicode"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/google/uuid"
	"go.opencensus.io/stats/view"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	"go.opentelemetry.io/otel/sdk/metric/export/aggregation"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	semconv "go.opentelemetry.io/collector/semconv/v1.5.0"
	"go.opentelemetry.io/collector/service/featuregate"
	"go.opentelemetry.io/collector/service/internal/telemetry"
)

// collectorTelemetry is collector's own telemetry.
var collectorTelemetry collectorTelemetryExporter = newColTelemetry(featuregate.GetRegistry())

const (
	zapKeyTelemetryAddress = "address"
	zapKeyTelemetryLevel   = "level"

	// useOtelForInternalMetricsfeatureGateID is the feature gate ID that controls whether the collector uses open
	// telemetry for internal metrics.
	useOtelForInternalMetricsfeatureGateID = "telemetry.useOtelForInternalMetrics"
)

type collectorTelemetryExporter interface {
	init(svc *service) error
	shutdown() error
}

type colTelemetry struct {
	registry   *featuregate.Registry
	views      []*view.View
	server     *http.Server
	doInitOnce sync.Once
}

func newColTelemetry(registry *featuregate.Registry) *colTelemetry {
	registry.MustRegister(featuregate.Gate{
		ID:          useOtelForInternalMetricsfeatureGateID,
		Description: "controls whether the collector to uses open telemetry for internal metrics",
		Enabled:     false,
	})
	return &colTelemetry{registry: registry}
}

func (tel *colTelemetry) init(svc *service) error {
	var err error
	tel.doInitOnce.Do(
		func() {
			err = tel.initOnce(svc)
		},
	)
	if err != nil {
		return fmt.Errorf("failed to initialize telemetry: %w", err)
	}
	return nil
}

func (tel *colTelemetry) initOnce(svc *service) error {
	telemetryConf := svc.config.Telemetry

	if telemetryConf.Metrics.Level == configtelemetry.LevelNone || telemetryConf.Metrics.Address == "" {
		svc.telemetry.Logger.Info(
			"Skipping telemetry setup.",
			zap.String(zapKeyTelemetryAddress, telemetryConf.Metrics.Address),
			zap.String(zapKeyTelemetryLevel, telemetryConf.Metrics.Level.String()),
		)
		return nil
	}

	svc.telemetry.Logger.Info("Setting up own telemetry...")

	// Construct telemetry attributes from resource attributes.
	telAttrs := map[string]string{}
	for k, v := range telemetryConf.Resource {
		// nil value indicates that the attribute should not be included in the telemetry.
		if v != nil {
			telAttrs[k] = *v
		}
	}

	if _, ok := telemetryConf.Resource[semconv.AttributeServiceInstanceID]; !ok {
		// AttributeServiceInstanceID is not specified in the config. Auto-generate one.
		instanceUUID, _ := uuid.NewRandom()
		instanceID := instanceUUID.String()
		telAttrs[semconv.AttributeServiceInstanceID] = instanceID
	}

	if _, ok := telemetryConf.Resource[semconv.AttributeServiceVersion]; !ok {
		// AttributeServiceVersion is not specified in the config. Use the actual
		// build version.
		telAttrs[semconv.AttributeServiceVersion] = svc.buildInfo.Version
	}

	var pe http.Handler
	if tel.registry.IsEnabled(useOtelForInternalMetricsfeatureGateID) {
		otelHandler, err := tel.initOpenTelemetry(svc)
		if err != nil {
			return err
		}
		pe = otelHandler
	} else {
		ocHandler, err := tel.initOpenCensus(svc, telAttrs)
		if err != nil {
			return err
		}
		pe = ocHandler
	}

	svc.telemetry.Logger.Info(
		"Serving Prometheus metrics",
		zap.String(zapKeyTelemetryAddress, telemetryConf.Metrics.Address),
		zap.String(zapKeyTelemetryLevel, telemetryConf.Metrics.Level.String()),
	)

	mux := http.NewServeMux()
	mux.Handle("/metrics", pe)

	tel.server = &http.Server{
		Addr:    telemetryConf.Metrics.Address,
		Handler: mux,
	}

	go func() {
		if serveErr := tel.server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			svc.host.asyncErrorChannel <- serveErr
		}
	}()

	return nil
}

func (tel *colTelemetry) initOpenCensus(svc *service, telAttrs map[string]string) (http.Handler, error) {
	telemetryConf := svc.config.Telemetry
	processMetricsViews, err := telemetry.NewProcessMetricsViews(getBallastSize(svc.host))
	if err != nil {
		return nil, err
	}

	var views []*view.View
	obsMetrics := obsreportconfig.Configure(telemetryConf.Metrics.Level)
	views = append(views, batchprocessor.MetricViews()...)
	views = append(views, obsMetrics.Views...)
	views = append(views, processMetricsViews.Views()...)

	tel.views = views
	if err = view.Register(views...); err != nil {
		return nil, err
	}

	processMetricsViews.StartCollection()

	// Until we can use a generic metrics exporter, default to Prometheus.
	opts := prometheus.Options{
		Namespace: "otelcol",
	}

	opts.ConstLabels = make(map[string]string)

	for k, v := range telAttrs {
		opts.ConstLabels[sanitizePrometheusKey(k)] = v
	}

	pe, err := prometheus.NewExporter(opts)
	if err != nil {
		return nil, err
	}

	view.RegisterExporter(pe)
	return pe, nil
}

func (tel *colTelemetry) initOpenTelemetry(svc *service) (http.Handler, error) {
	config := otelprometheus.Config{}
	c := controller.New(
		processor.NewFactory(
			selector.NewWithHistogramDistribution(
				histogram.WithExplicitBoundaries(config.DefaultHistogramBoundaries),
			),
			aggregation.CumulativeTemporalitySelector(),
			processor.WithMemory(true),
		),
	)

	pe, err := otelprometheus.New(config, c)
	if err != nil {
		return nil, err
	}

	svc.telemetry.MeterProvider = pe.MeterProvider()
	return pe, err
}

func (tel *colTelemetry) shutdown() error {
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
