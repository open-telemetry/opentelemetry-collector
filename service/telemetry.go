// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/internal/collector/telemetry"
	"github.com/open-telemetry/opentelemetry-collector/obsreport"
	"github.com/open-telemetry/opentelemetry-collector/processor"
	"github.com/open-telemetry/opentelemetry-collector/processor/batchprocessor"
	"github.com/open-telemetry/opentelemetry-collector/processor/queuedprocessor"
	"github.com/open-telemetry/opentelemetry-collector/processor/samplingprocessor/tailsamplingprocessor"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

var (
	// AppTelemetry is application's own telemetry.
	AppTelemetry = &appTelemetry{}
)

type appTelemetry struct {
	views []*view.View
}

func (tel *appTelemetry) init(asyncErrorChannel chan<- error, ballastSizeBytes uint64, logger *zap.Logger) error {
	level, err := telemetry.GetLevel()
	if err != nil {
		return errors.Wrap(err, "failed to parse metrics level")
	}

	if level == telemetry.None {
		return nil
	}

	port := telemetry.GetMetricsPort()

	var views []*view.View
	views = append(views, obsreport.Configure(telemetry.UseLegacyMetrics(), telemetry.UseNewMetrics())...)
	views = append(views, processor.MetricViews(level)...)
	views = append(views, queuedprocessor.MetricViews(level)...)
	views = append(views, batchprocessor.MetricViews()...)
	views = append(views, tailsamplingprocessor.SamplingProcessorMetricViews(level)...)
	processMetricsViews := telemetry.NewProcessMetricsViews(ballastSizeBytes)
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
			sanitizePrometheusKey(conventions.AttributeServiceInstance): instanceID,
		}
	}

	pe, err := prometheus.NewExporter(opts)
	if err != nil {
		return err
	}

	view.RegisterExporter(pe)

	logger.Info(
		"Serving Prometheus metrics",
		zap.Int("port", port),
		zap.Bool("legacy_metrics", telemetry.UseLegacyMetrics()),
		zap.Bool("new_metrics", telemetry.UseNewMetrics()),
		zap.Int8("level", int8(level)), // TODO: make it human friendly
		zap.String(conventions.AttributeServiceInstance, instanceID),
	)

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", pe)
		serveErr := http.ListenAndServe(":"+strconv.Itoa(port), mux)
		if serveErr != nil && serveErr != http.ErrServerClosed {
			asyncErrorChannel <- serveErr
		}
	}()

	return nil
}

func (tel *appTelemetry) shutdown() {
	view.Unregister(tel.views...)
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
