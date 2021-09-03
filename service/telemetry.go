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

package service

import (
	"net/http"
	"strings"
	"sync"
	"unicode"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/google/uuid"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/collector/telemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	semconv "go.opentelemetry.io/collector/model/semconv/v1.5.0"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	telemetry2 "go.opentelemetry.io/collector/service/internal/telemetry"
)

// collectorTelemetry is collector's own telemetry.
var collectorTelemetry collectorTelemetryExporter = &colTelemetry{}

type collectorTelemetryExporter interface {
	init(asyncErrorChannel chan<- error, ballastSizeBytes uint64, logger *zap.Logger) error
	shutdown() error
}

type colTelemetry struct {
	views      []*view.View
	server     *http.Server
	doInitOnce sync.Once
}

func (tel *colTelemetry) init(asyncErrorChannel chan<- error, ballastSizeBytes uint64, logger *zap.Logger) (err error) {
	tel.doInitOnce.Do(
		func() {
			err = tel.initOnce(asyncErrorChannel, ballastSizeBytes, logger)
		},
	)
	return
}
func (tel *colTelemetry) initOnce(asyncErrorChannel chan<- error, ballastSizeBytes uint64, logger *zap.Logger) error {
	level := configtelemetry.GetMetricsLevelFlagValue()
	metricsAddr := telemetry.GetMetricsAddr()

	if level == configtelemetry.LevelNone || metricsAddr == "" {
		return nil
	}

	processMetricsViews, err := telemetry2.NewProcessMetricsViews(ballastSizeBytes)
	if err != nil {
		return err
	}

	var views []*view.View
	obsMetrics := obsreportconfig.Configure(level)
	views = append(views, batchprocessor.MetricViews()...)
	views = append(views, obsMetrics.Views...)
	views = append(views, processMetricsViews.Views()...)

	tel.views = views
	if err = view.Register(views...); err != nil {
		return err
	}

	processMetricsViews.StartCollection()

	// Until we can use a generic metrics exporter, default to Prometheus.
	opts := prometheus.Options{
		Namespace: telemetry.GetMetricsPrefix(),
	}

	var instanceID string
	if telemetry.GetAddInstanceID() {
		instanceUUID, _ := uuid.NewRandom()
		instanceID = instanceUUID.String()
		opts.ConstLabels = map[string]string{
			sanitizePrometheusKey(semconv.AttributeServiceInstanceID): instanceID,
		}
	}

	pe, err := prometheus.NewExporter(opts)
	if err != nil {
		return err
	}

	view.RegisterExporter(pe)

	logger.Info(
		"Serving Prometheus metrics",
		zap.String("address", metricsAddr),
		zap.Int8("level", int8(level)), // TODO: make it human friendly
		zap.String(semconv.AttributeServiceInstanceID, instanceID),
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
