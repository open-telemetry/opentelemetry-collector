// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute // import "go.opentelemetry.io/collector/service/internal/componentattribute"

import (
	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/internal/telemetry"
)

func TelemetrySettingsWithAttributes(ts component.TelemetrySettings, attrSet attribute.Set) component.TelemetrySettings {
	attrs := attrSet.ToSlice()
	ts.Logger = LoggerWithAttributes(ts.Logger, attrs)
	ts.TracerProvider = tracerProviderWithAttributes{
		TracerProvider: ts.TracerProvider,
		attrs:          attrs,
	}
	if telemetry.NewPipelineTelemetryGate.IsEnabled() {
		ts.MeterProvider = meterProviderWithAttributes{
			MeterProvider: ts.MeterProvider,
			attrs:         attrs,
		}
	}
	return ts
}
