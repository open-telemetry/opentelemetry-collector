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
	"flag"
	"log"
	"net/http"
	"strconv"

	"contrib.go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/internal/collector/telemetry"
	"github.com/open-telemetry/opentelemetry-collector/observability"
	"github.com/open-telemetry/opentelemetry-collector/processor"
	"github.com/open-telemetry/opentelemetry-collector/processor/batchprocessor"
	"github.com/open-telemetry/opentelemetry-collector/processor/queuedprocessor"
	"github.com/open-telemetry/opentelemetry-collector/processor/samplingprocessor/tailsamplingprocessor"
)

const (
	metricsPortCfg  = "metrics-port"
	metricsLevelCfg = "metrics-level"
)

var (
	// AppTelemetry is application's own telemetry.
	AppTelemetry = &appTelemetry{}

	// Command-line flags that control publication of telemetry data.
	metricsLevelPtr *string
	metricsPortPtr  *uint
)

type appTelemetry struct {
	views []*view.View
}

func telemetryFlags(flags *flag.FlagSet) {
	metricsLevelPtr = flags.String(metricsLevelCfg, "BASIC", "Output level of telemetry metrics (NONE, BASIC, NORMAL, DETAILED)")
	// At least until we can use a generic, i.e.: OpenCensus, metrics exporter we default to Prometheus at port 8888, if not otherwise specified.
	metricsPortPtr = flags.Uint(metricsPortCfg, 8888, "Port exposing collector telemetry.")
}

func (tel *appTelemetry) init(asyncErrorChannel chan<- error, ballastSizeBytes uint64, logger *zap.Logger) error {
	level, err := telemetry.ParseLevel(*metricsLevelPtr)
	if err != nil {
		log.Fatalf("Failed to parse metrics level: %v", err)
	}

	if level == telemetry.None {
		return nil
	}

	port := int(*metricsPortPtr)

	views := processor.MetricViews(level)
	views = append(views, queuedprocessor.MetricViews(level)...)
	views = append(views, batchprocessor.MetricViews(level)...)
	views = append(views, observability.AllViews...)
	views = append(views, tailsamplingprocessor.SamplingProcessorMetricViews(level)...)
	processMetricsViews := telemetry.NewProcessMetricsViews(ballastSizeBytes)
	views = append(views, processMetricsViews.Views()...)
	tel.views = views
	if err := view.Register(views...); err != nil {
		return err
	}

	processMetricsViews.StartCollection()

	// Until we can use a generic metrics exporter, default to Prometheus.
	opts := prometheus.Options{
		Namespace: "oc_collector",
	}
	pe, err := prometheus.NewExporter(opts)
	if err != nil {
		return err
	}

	view.RegisterExporter(pe)

	logger.Info("Serving Prometheus metrics", zap.Int("port", port))
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
