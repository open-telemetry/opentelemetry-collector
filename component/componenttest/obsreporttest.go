// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
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
	ts           component.TelemetrySettings
	id           component.ID
	SpanRecorder *tracetest.SpanRecorder

	prometheusChecker *prometheusChecker
	meterProvider     *sdkmetric.MeterProvider
}

// CheckExporterTraces checks that for the current exported values for trace exporter metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (tts *TestTelemetry) CheckExporterTraces(sentSpans, sendFailedSpans int64) error {
	return tts.prometheusChecker.checkExporterTraces(tts.id, sentSpans, sendFailedSpans)
}

// CheckExporterMetrics checks that for the current exported values for metrics exporter metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (tts *TestTelemetry) CheckExporterMetrics(sentMetricsPoints, sendFailedMetricsPoints int64) error {
	return tts.prometheusChecker.checkExporterMetrics(tts.id, sentMetricsPoints, sendFailedMetricsPoints)
}

func (tts *TestTelemetry) CheckExporterEnqueueFailedMetrics(enqueueFailed int64) error {
	return tts.prometheusChecker.checkExporterEnqueueFailed(tts.id, "metric_points", enqueueFailed)
}

func (tts *TestTelemetry) CheckExporterEnqueueFailedTraces(enqueueFailed int64) error {
	return tts.prometheusChecker.checkExporterEnqueueFailed(tts.id, "spans", enqueueFailed)
}

func (tts *TestTelemetry) CheckExporterEnqueueFailedLogs(enqueueFailed int64) error {
	return tts.prometheusChecker.checkExporterEnqueueFailed(tts.id, "log_records", enqueueFailed)
}

// CheckExporterLogs checks that for the current exported values for logs exporter metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (tts *TestTelemetry) CheckExporterLogs(sentLogRecords, sendFailedLogRecords int64) error {
	return tts.prometheusChecker.checkExporterLogs(tts.id, sentLogRecords, sendFailedLogRecords)
}

func (tts *TestTelemetry) CheckExporterMetricGauge(metric string, val int64, extraAttrs ...attribute.KeyValue) error {
	attrs := attributesForExporterMetrics(tts.id)
	attrs = append(attrs, extraAttrs...)
	return tts.prometheusChecker.checkGauge(metric, val, attrs)
}

// CheckProcessorTraces checks that for the current exported values for trace exporter metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (tts *TestTelemetry) CheckProcessorTraces(acceptedSpans, refusedSpans, droppedSpans, insertedSpans int64) error {
	return tts.prometheusChecker.checkProcessorTraces(tts.id, acceptedSpans, refusedSpans, droppedSpans, insertedSpans)
}

// CheckProcessorMetrics checks that for the current exported values for metrics exporter metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (tts *TestTelemetry) CheckProcessorMetrics(acceptedMetricPoints, refusedMetricPoints, droppedMetricPoints, insertedMetricPoints int64) error {
	return tts.prometheusChecker.checkProcessorMetrics(tts.id, acceptedMetricPoints, refusedMetricPoints, droppedMetricPoints, insertedMetricPoints)
}

// CheckProcessorLogs checks that for the current exported values for logs exporter metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (tts *TestTelemetry) CheckProcessorLogs(acceptedLogRecords, refusedLogRecords, droppedLogRecords, insertedLogRecords int64) error {
	return tts.prometheusChecker.checkProcessorLogs(tts.id, acceptedLogRecords, refusedLogRecords, droppedLogRecords, insertedLogRecords)
}

// CheckReceiverTraces checks that for the current exported values for trace receiver metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (tts *TestTelemetry) CheckReceiverTraces(protocol string, acceptedSpans, droppedSpans int64) error {
	return tts.prometheusChecker.checkReceiverTraces(tts.id, protocol, acceptedSpans, droppedSpans)
}

// CheckReceiverLogs checks that for the current exported values for logs receiver metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (tts *TestTelemetry) CheckReceiverLogs(protocol string, acceptedLogRecords, droppedLogRecords int64) error {
	return tts.prometheusChecker.checkReceiverLogs(tts.id, protocol, acceptedLogRecords, droppedLogRecords)
}

// CheckReceiverMetrics checks that for the current exported values for metrics receiver metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (tts *TestTelemetry) CheckReceiverMetrics(protocol string, acceptedMetricPoints, droppedMetricPoints int64) error {
	return tts.prometheusChecker.checkReceiverMetrics(tts.id, protocol, acceptedMetricPoints, droppedMetricPoints)
}

// CheckScraperMetrics checks that for the current exported values for metrics scraper metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (tts *TestTelemetry) CheckScraperMetrics(receiver component.ID, scraper component.ID, scrapedMetricPoints, erroredMetricPoints int64) error {
	return tts.prometheusChecker.checkScraperMetrics(receiver, scraper, scrapedMetricPoints, erroredMetricPoints)
}

// Shutdown unregisters any views and shuts down the SpanRecorder
func (tts *TestTelemetry) Shutdown(ctx context.Context) error {
	var errs error
	errs = multierr.Append(errs, tts.SpanRecorder.Shutdown(ctx))
	if tts.meterProvider != nil {
		errs = multierr.Append(errs, tts.meterProvider.Shutdown(ctx))
	}
	return errs
}

// TelemetrySettings returns the TestTelemetry's TelemetrySettings
func (tts *TestTelemetry) TelemetrySettings() component.TelemetrySettings {
	return tts.ts
}

// SetupTelemetry does setup the testing environment to check the metrics recorded by receivers, producers or exporters.
// The caller must pass the ID of the component that intends to test, so the CreateSettings and Check methods will use.
// The caller should defer a call to Shutdown the returned TestTelemetry.
func SetupTelemetry(id component.ID) (TestTelemetry, error) {
	sr := new(tracetest.SpanRecorder)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	settings := TestTelemetry{
		ts:           NewNopTelemetrySettings(),
		id:           id,
		SpanRecorder: sr,
	}
	settings.ts.TracerProvider = tp
	settings.ts.MetricsLevel = configtelemetry.LevelNormal

	promRegOtel := prometheus.NewRegistry()

	exp, err := otelprom.New(otelprom.WithRegisterer(promRegOtel), otelprom.WithoutUnits(), otelprom.WithoutScopeInfo(), otelprom.WithoutCounterSuffixes())
	if err != nil {
		return settings, err
	}

	settings.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource.Empty()),
		sdkmetric.WithReader(exp),
	)
	settings.ts.MeterProvider = settings.meterProvider

	settings.prometheusChecker = &prometheusChecker{
		otelHandler: promhttp.HandlerFor(promRegOtel, promhttp.HandlerOpts{}),
	}

	return settings, nil
}
