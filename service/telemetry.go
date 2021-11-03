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
	"fmt"
	"net/http"
	"strings"
	"sync"
	"unicode"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/google/uuid"
	"go.opencensus.io/stats/view"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric/global"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/internal/version"
	semconv "go.opentelemetry.io/collector/model/semconv/v1.5.0"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	telemetry2 "go.opentelemetry.io/collector/service/internal/telemetry"
)

// collectorTelemetry is collector's own telemetry.
var collectorTelemetry collectorTelemetryExporter = &colTelemetry{}

// AddCollectorVersionTag indicates if the collector version tag should be added to all telemetry metrics
const AddCollectorVersionTag = true

const (
	zapKeyTelemetryAddress = "address"
	zapKeyTelemetryLevel   = "level"
)

type collectorTelemetryExporter interface {
	init(asyncErrorChannel chan<- error, ballastSizeBytes uint64, logger *zap.Logger) error
	shutdown() error
}

type colTelemetry struct {
	views      []*view.View
	server     *http.Server
	doInitOnce sync.Once
}

func (tel *colTelemetry) init(asyncErrorChannel chan<- error, ballastSizeBytes uint64, logger *zap.Logger) error {
	var err error
	tel.doInitOnce.Do(
		func() {
			err = tel.initOnce(asyncErrorChannel, ballastSizeBytes, logger)
		},
	)
	if err != nil {
		return fmt.Errorf("failed to initialize telemetry: %w", err)
	}
	return nil
}

func (tel *colTelemetry) initOnce(asyncErrorChannel chan<- error, ballastSizeBytes uint64, logger *zap.Logger) error {
	level := configtelemetry.GetMetricsLevelFlagValue()
	metricsAddr := getMetricsAddr()

	if level == configtelemetry.LevelNone || metricsAddr == "" {
		logger.Info(
			"Skipping telemetry setup.",
			zap.String(zapKeyTelemetryAddress, metricsAddr),
			zap.String(zapKeyTelemetryLevel, level.String()),
		)
		return nil
	}

	logger.Info("Setting up own telemetry...")

	var instanceID string

	if getAddInstanceID() {
		instanceUUID, _ := uuid.NewRandom()
		instanceID = instanceUUID.String()
	}

	var pe http.Handler
	if configtelemetry.UseOpenTelemetryForInternalMetrics {
		otelHandler, err := tel.initOpenTelemetry()
		if err != nil {
			return err
		}
		pe = otelHandler
	} else {
		ocHandler, err := tel.initOpenCensus(level, instanceID, ballastSizeBytes)
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
		serveErr := tel.server.ListenAndServe()
		if serveErr != nil && serveErr != http.ErrServerClosed {
			asyncErrorChannel <- serveErr
		}
	}()

	return nil
}

func (tel *colTelemetry) initOpenCensus(level configtelemetry.Level, instanceID string, ballastSizeBytes uint64) (http.Handler, error) {
	processMetricsViews, err := telemetry2.NewProcessMetricsViews(ballastSizeBytes)
	if err != nil {
		return nil, err
	}

	var views []*view.View
	obsMetrics := obsreportconfig.Configure(level)
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
		Namespace: getMetricsPrefix(),
	}

	opts.ConstLabels = make(map[string]string)

	if getAddInstanceID() {
		opts.ConstLabels[sanitizePrometheusKey(semconv.AttributeServiceInstanceID)] = instanceID
	}

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

func (tel *colTelemetry) initOpenTelemetry() (http.Handler, error) {
	config := otelprometheus.Config{}
	c := controller.New(
		processor.NewFactory(
			selector.NewWithHistogramDistribution(
				histogram.WithExplicitBoundaries(config.DefaultHistogramBoundaries),
			),
			export.CumulativeExportKindSelector(),
			processor.WithMemory(true),
		),
	)

	pe, err := otelprometheus.New(config, c)
	if err != nil {
		return nil, err
	}

	global.SetMeterProvider(pe.MeterProvider())
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
