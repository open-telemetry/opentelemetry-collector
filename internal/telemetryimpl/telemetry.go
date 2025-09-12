// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetryimpl // import "go.opentelemetry.io/collector/internal/telemetryimpl"

import (
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/telemetry"
)

var NewPipelineTelemetryGate = featuregate.GlobalRegistry().MustRegister(
	"telemetry.newPipelineTelemetry",
	featuregate.StageAlpha,
	featuregate.WithRegisterFromVersion("v0.123.0"),
	featuregate.WithRegisterReferenceURL("https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/rfcs/component-universal-telemetry.md"),
	featuregate.WithRegisterDescription("Injects component-identifying scope attributes in internal Collector metrics"),
)

func InitializeWithAttributes(ts internaltelemetry.TelemetrySettings) internaltelemetry.TelemetrySettings {
	ts.Logger = ZapLoggerWithAttributes(ts.Logger, ts.Attributes())
	ts.TracerProvider = TracerProviderWithAttributes(ts.TracerProvider, ts.Attributes())
	if NewPipelineTelemetryGate.IsEnabled() {
		ts.MeterProvider = MeterProviderWithAttributes(ts.MeterProvider, ts.Attributes())
	}
	return ts
}
