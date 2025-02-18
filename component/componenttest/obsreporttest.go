// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/collector/component"
)

type TestTelemetry struct {
	*Telemetry
	id component.ID
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
