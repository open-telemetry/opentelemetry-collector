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

package obsreporttest // import "go.opentelemetry.io/collector/obsreport/obsreporttest"

import (
	"context"

	ocprom "contrib.go.opencensus.io/exporter/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"go.opencensus.io/stats/view"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

const (
	// Names used by the metrics and labels are hard coded here in order to avoid
	// inadvertent changes: at this point changing metric names and labels should
	// be treated as a breaking changing and requires a good justification.
	// Changes to metric names or labels can break alerting, dashboards, etc
	// that are used to monitor the Collector in production deployments.
	// DO NOT SWITCH THE VARIABLES BELOW TO SIMILAR ONES DEFINED ON THE PACKAGE.
	receiverTag  = "receiver"
	scraperTag   = "scraper"
	transportTag = "transport"
	exporterTag  = "exporter"
	processorTag = "processor"
)

type TestTelemetry struct {
	component.TelemetrySettings
	id           component.ID
	SpanRecorder *tracetest.SpanRecorder
	views        []*view.View

	otelPrometheusChecker *prometheusChecker
	meterProvider         *sdkmetric.MeterProvider
	ocExporter            *ocprom.Exporter
}

// ToExporterCreateSettings returns an exporter.CreateSettings with configured TelemetrySettings.
func (tts *TestTelemetry) ToExporterCreateSettings() exporter.CreateSettings {
	set := exportertest.NewNopCreateSettings()
	set.TelemetrySettings = tts.TelemetrySettings
	set.ID = tts.id
	return set
}

// ToProcessorCreateSettings returns a processor.CreateSettings with configured TelemetrySettings.
func (tts *TestTelemetry) ToProcessorCreateSettings() processor.CreateSettings {
	set := processortest.NewNopCreateSettings()
	set.TelemetrySettings = tts.TelemetrySettings
	set.ID = tts.id
	return set
}

// ToReceiverCreateSettings returns a receiver.CreateSettings with configured TelemetrySettings.
func (tts *TestTelemetry) ToReceiverCreateSettings() receiver.CreateSettings {
	set := receivertest.NewNopCreateSettings()
	set.TelemetrySettings = tts.TelemetrySettings
	set.ID = tts.id
	return set
}

// CheckExporterTraces checks that for the current exported values for trace exporter metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (tts *TestTelemetry) CheckExporterTraces(sentSpans, sendFailedSpans int64) error {
	return tts.otelPrometheusChecker.checkExporterTraces(tts.id, sentSpans, sendFailedSpans)
}

// CheckExporterMetrics checks that for the current exported values for metrics exporter metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (tts *TestTelemetry) CheckExporterMetrics(sentMetricsPoints, sendFailedMetricsPoints int64) error {
	return tts.otelPrometheusChecker.checkExporterMetrics(tts.id, sentMetricsPoints, sendFailedMetricsPoints)
}

// CheckExporterLogs checks that for the current exported values for logs exporter metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (tts *TestTelemetry) CheckExporterLogs(sentLogRecords, sendFailedLogRecords int64) error {
	return tts.otelPrometheusChecker.checkExporterLogs(tts.id, sentLogRecords, sendFailedLogRecords)
}

// CheckProcessorTraces checks that for the current exported values for trace exporter metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (tts *TestTelemetry) CheckProcessorTraces(acceptedSpans, refusedSpans, droppedSpans int64) error {
	return tts.otelPrometheusChecker.checkProcessorTraces(tts.id, acceptedSpans, refusedSpans, droppedSpans)
}

// CheckProcessorMetrics checks that for the current exported values for metrics exporter metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (tts *TestTelemetry) CheckProcessorMetrics(acceptedMetricPoints, refusedMetricPoints, droppedMetricPoints int64) error {
	return tts.otelPrometheusChecker.checkProcessorMetrics(tts.id, acceptedMetricPoints, refusedMetricPoints, droppedMetricPoints)
}

// CheckProcessorLogs checks that for the current exported values for logs exporter metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (tts *TestTelemetry) CheckProcessorLogs(acceptedLogRecords, refusedLogRecords, droppedLogRecords int64) error {
	return tts.otelPrometheusChecker.checkProcessorLogs(tts.id, acceptedLogRecords, refusedLogRecords, droppedLogRecords)
}

// CheckReceiverTraces checks that for the current exported values for trace receiver metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (tts *TestTelemetry) CheckReceiverTraces(protocol string, acceptedSpans, droppedSpans int64) error {
	return tts.otelPrometheusChecker.checkReceiverTraces(tts.id, protocol, acceptedSpans, droppedSpans)
}

// CheckReceiverLogs checks that for the current exported values for logs receiver metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (tts *TestTelemetry) CheckReceiverLogs(protocol string, acceptedLogRecords, droppedLogRecords int64) error {
	return tts.otelPrometheusChecker.checkReceiverLogs(tts.id, protocol, acceptedLogRecords, droppedLogRecords)
}

// CheckReceiverMetrics checks that for the current exported values for metrics receiver metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (tts *TestTelemetry) CheckReceiverMetrics(protocol string, acceptedMetricPoints, droppedMetricPoints int64) error {
	return tts.otelPrometheusChecker.checkReceiverMetrics(tts.id, protocol, acceptedMetricPoints, droppedMetricPoints)
}

// Shutdown unregisters any views and shuts down the SpanRecorder
func (tts *TestTelemetry) Shutdown(ctx context.Context) error {
	view.Unregister(tts.views...)
	view.UnregisterExporter(tts.ocExporter)
	var errs error
	errs = multierr.Append(errs, tts.SpanRecorder.Shutdown(ctx))
	if tts.meterProvider != nil {
		errs = multierr.Append(errs, tts.meterProvider.Shutdown(ctx))
	}
	return errs
}

// SetupTelemetry does setup the testing environment to check the metrics recorded by receivers, producers or exporters.
// The caller must pass the ID of the component that intends to test, so the CreateSettings and Check methods will use.
// The caller should defer a call to Shutdown the returned TestTelemetry.
func SetupTelemetry(id component.ID) (TestTelemetry, error) {
	sr := new(tracetest.SpanRecorder)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	settings := TestTelemetry{
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		id:                id,
		SpanRecorder:      sr,
	}
	settings.TelemetrySettings.TracerProvider = tp
	settings.TelemetrySettings.MetricsLevel = configtelemetry.LevelNormal
	settings.views = obsreportconfig.AllViews(configtelemetry.LevelNormal)
	err := view.Register(settings.views...)
	if err != nil {
		return settings, err
	}

	promReg := prometheus.NewRegistry()

	settings.ocExporter, err = ocprom.NewExporter(ocprom.Options{Registry: promReg})
	if err != nil {
		return settings, err
	}
	view.RegisterExporter(settings.ocExporter)

	exp, err := otelprom.New(otelprom.WithRegisterer(promReg), otelprom.WithoutUnits(), otelprom.WithoutScopeInfo())
	if err != nil {
		return settings, err
	}

	settings.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource.Empty()),
		sdkmetric.WithReader(exp),
	)
	settings.TelemetrySettings.MeterProvider = settings.meterProvider

	settings.otelPrometheusChecker = &prometheusChecker{promHandler: settings.ocExporter}

	return settings, nil
}

// CheckScraperMetrics checks that for the current exported values for metrics scraper metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func CheckScraperMetrics(tts TestTelemetry, receiver component.ID, scraper component.ID, scrapedMetricPoints, erroredMetricPoints int64) error {
	return tts.otelPrometheusChecker.checkScraperMetrics(receiver, scraper, scrapedMetricPoints, erroredMetricPoints)
}
