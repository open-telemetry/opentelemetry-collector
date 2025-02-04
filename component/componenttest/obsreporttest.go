// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/collector/component"
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
)

type TestTelemetry struct {
	*Telemetry
	id component.ID
}

// Deprecated: [v0.119.0] use the metadatatest.AssertEqualMetric series of functions instead.
// CheckExporterTraces checks that for the current exported values for trace exporter metrics match given values.
func (tts *TestTelemetry) CheckExporterTraces(sentSpans, sendFailedSpans int64) error {
	return checkExporter(tts.Telemetry, tts.id, "spans", sentSpans, sendFailedSpans)
}

// Deprecated: [v0.119.0] use the metadatatest.AssertEqualMetric series of functions instead.
// CheckExporterMetrics checks that for the current exported values for metrics exporter metrics match given values.
func (tts *TestTelemetry) CheckExporterMetrics(sentMetricsPoints, sendFailedMetricsPoints int64) error {
	return checkExporter(tts.Telemetry, tts.id, "metric_points", sentMetricsPoints, sendFailedMetricsPoints)
}

// Deprecated: [v0.119.0] use the metadatatest.AssertEqualMetric series of functions instead.
func (tts *TestTelemetry) CheckExporterEnqueueFailedMetrics(enqueueFailed int64) error {
	return checkExporterEnqueueFailed(tts.Telemetry, tts.id, "metric_points", enqueueFailed)
}

// Deprecated: [v0.119.0] use the metadatatest.AssertEqualMetric series of functions instead.
func (tts *TestTelemetry) CheckExporterEnqueueFailedTraces(enqueueFailed int64) error {
	return checkExporterEnqueueFailed(tts.Telemetry, tts.id, "spans", enqueueFailed)
}

// Deprecated: [v0.119.0] use the metadatatest.AssertEqualMetric series of functions instead.
func (tts *TestTelemetry) CheckExporterEnqueueFailedLogs(enqueueFailed int64) error {
	return checkExporterEnqueueFailed(tts.Telemetry, tts.id, "log_records", enqueueFailed)
}

// Deprecated: [v0.119.0] use the metadatatest.AssertEqualMetric series of functions instead.
// CheckExporterLogs checks that for the current exported values for logs exporter metrics match given values.
func (tts *TestTelemetry) CheckExporterLogs(sentLogRecords, sendFailedLogRecords int64) error {
	return checkExporter(tts.Telemetry, tts.id, "log_records", sentLogRecords, sendFailedLogRecords)
}

// Deprecated: [v0.119.0] use the metadatatest.AssertEqualMetric series of functions instead.
func (tts *TestTelemetry) CheckExporterMetricGauge(metric string, val int64, extraAttrs ...attribute.KeyValue) error {
	attrs := attributesForExporterMetrics(tts.id, extraAttrs...)
	return checkIntGauge(tts.Telemetry, metric, val, attrs)
}

// CheckReceiverTraces checks that for the current exported values for trace receiver metrics match given values.
func (tts *TestTelemetry) CheckReceiverTraces(protocol string, acceptedSpans, droppedSpans int64) error {
	return checkReceiver(tts.Telemetry, tts.id, "spans", protocol, acceptedSpans, droppedSpans)
}

// Deprecated: [v0.119.0] use the metadatatest.AssertEqualMetric series of functions instead.
// CheckReceiverLogs checks that for the current exported values for logs receiver metrics match given values.
func (tts *TestTelemetry) CheckReceiverLogs(protocol string, acceptedLogRecords, droppedLogRecords int64) error {
	return checkReceiver(tts.Telemetry, tts.id, "log_records", protocol, acceptedLogRecords, droppedLogRecords)
}

// CheckReceiverMetrics checks that for the current exported values for metrics receiver metrics match given values.
func (tts *TestTelemetry) CheckReceiverMetrics(protocol string, acceptedMetricPoints, droppedMetricPoints int64) error {
	return checkReceiver(tts.Telemetry, tts.id, "metric_points", protocol, acceptedMetricPoints, droppedMetricPoints)
}

// TelemetrySettings returns the TestTelemetry's TelemetrySettings
func (tts *TestTelemetry) TelemetrySettings() component.TelemetrySettings {
	return tts.NewTelemetrySettings()
}

// SetupTelemetry sets up the testing environment to check the metrics recorded by receivers, producers, or exporters.
// The caller must pass the ID of the component being tested. The ID will be used by the CreateSettings and Check methods.
// The caller must defer a call to `Shutdown` on the returned TestTelemetry.
func SetupTelemetry(id component.ID) (TestTelemetry, error) {
	return TestTelemetry{
		Telemetry: NewTelemetry(
			WithMetricOptions(sdkmetric.WithResource(resource.Empty())),
			WithTraceOptions(sdktrace.WithResource(resource.Empty()))),
		id: id,
	}, nil
}
