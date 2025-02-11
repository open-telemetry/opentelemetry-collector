// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
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

// Deprecated: [v0.120.0] use the metadatatest.AssertEqualMetric series of functions instead.
// CheckReceiverTraces checks that for the current exported values for trace receiver metrics match given values.
func (tts *TestTelemetry) CheckReceiverTraces(protocol string, acceptedSpans, droppedSpans int64) error {
	return checkReceiver(tts.Telemetry, tts.id, "spans", protocol, acceptedSpans, droppedSpans)
}

// Deprecated: [v0.120.0] use the metadatatest.AssertEqualMetric series of functions instead.
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
