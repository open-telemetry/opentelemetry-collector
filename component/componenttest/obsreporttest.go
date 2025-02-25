// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/collector/component"
)

// Deprecated: [v0.121.0] use the componenttest.Telemetry instead.
type TestTelemetry struct {
	*Telemetry
	id component.ID
}

// Deprecated: [v0.121.0] use the NewTelemetrySettings from componenttest.Telemetry.
func (tts *TestTelemetry) TelemetrySettings() component.TelemetrySettings {
	return tts.NewTelemetrySettings()
}

// Deprecated: [v0.121.0] use the componenttest.NewTelemetry instead.
func SetupTelemetry(id component.ID) (TestTelemetry, error) {
	return TestTelemetry{
		Telemetry: NewTelemetry(
			WithMetricOptions(sdkmetric.WithResource(resource.Empty())),
			WithTraceOptions(sdktrace.WithResource(resource.Empty()))),
		id: id,
	}, nil
}
