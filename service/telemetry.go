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
	"go.opentelemetry.io/collector/internal/version"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	semconv "go.opentelemetry.io/collector/semconv/v1.5.0"
	"go.opentelemetry.io/collector/service/featuregate"
	telemetry2 "go.opentelemetry.io/collector/service/internal/telemetry"
)

// collectorTelemetry is collector's own telemetry.
var collectorTelemetry collectorTelemetryExporter = newColTelemetry(featuregate.GetRegistry())

// AddCollectorVersionTag indicates if the collector version tag should be added to all telemetry metrics
const AddCollectorVersionTag = true

const (
	zapKeyTelemetryAddress = "address"
	zapKeyTelemetryLevel   = "level"

	// useOtelForInternalMetricsfeatureGateID is the feature gate ID that controls whether the collector uses open
	// telemetry for internal metrics.
	useOtelForInternalMetricsfeatureGateID = "telemetry.useOtelForInternalMetrics"
)

type collectorTelemetryExporter interface {
	init(col *Collector) error
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

func (tel *colTelemetry) init(col *Collector) error {
	var err error
	tel.doInitOnce.Do(
		func() {
			err = tel.initOnce(col)
		},
	)
	if err != nil {
		return fmt.Errorf("failed to initialize telemetry: %w", err)
	}
	return nil
}

func (tel *colTelemetry) initOnce(col *Collector) error {
	logger := col.telemetry.Logger
	cfg := col.service.config.Telemetry

	level := cfg.Metrics.Level
	metricsAddr := cfg.Metrics.Address

	if level == configtelemetry.LevelNone || metricsAddr == "" {
		logger.Info(
			"Skipping telemetry setup.",
			zap.String(zapKeyTelemetryAddress, metricsAddr),
			zap.String(zapKeyTelemetryLevel, level.String()),
		)
		return nil
	}

	logger.Info("Setting up own telemetry...")

	instanceUUID, _ := uuid.NewRandom()
	instanceID := instanceUUID.String()

	var pe http.Handler
	if tel.registry.IsEnabled(useOtelForInternalMetricsfeatureGateID) {
		otelHandler, err := tel.initOpenTelemetry(col)
		if err != nil {
			return err
		}
		pe = otelHandler
	} else {
		ocHandler, err := tel.initOpenCensus(col, instanceID)
		if err != nil {
			return err
		}
		pe = ocHandler
	}

	logger.Info(
		"Serving Prometheus metrics",
		zap.String(zapKeyTelemetryAddress, metricsAddr),
		zap.String(zapKeyTelemetryLevel, level.String()),
		zap.String(semconv.AttributeServiceInstanceID, instanceID),
		zap.String(semconv.AttributeServiceVersion, version.Version),
	)

	mux := http.NewServeMux()
	mux.Handle("/metrics", pe)

	tel.server = &http.Server{
		Addr:    metricsAddr,
		Handler: mux,
	}

	go func() {
		if serveErr := tel.server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			col.asyncErrorChannel <- serveErr
		}
	}()

	return nil
}

func (tel *colTelemetry) initOpenCensus(col *Collector, instanceID string) (http.Handler, error) {
	processMetricsViews, err := telemetry2.NewProcessMetricsViews(getBallastSize(col.service.host))
	if err != nil {
		return nil, err
	}

	var views []*view.View
	obsMetrics := obsreportconfig.Configure(col.service.config.Telemetry.Metrics.Level)
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

	opts.ConstLabels[sanitizePrometheusKey(semconv.AttributeServiceInstanceID)] = instanceID

	if AddCollectorVersionTag {
		opts.ConstLabels[sanitizePrometheusKey(semconv.AttributeServiceVersion)] = version.Version
	}

	pe, err := prometheus.NewExporter(opts)
	if err != nil {
		return nil, err
	}

	view.RegisterExporter(pe)
	return pe, nil
}

func (tel *colTelemetry) initOpenTelemetry(col *Collector) (http.Handler, error) {
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

	col.telemetry.MeterProvider = pe.MeterProvider()
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
