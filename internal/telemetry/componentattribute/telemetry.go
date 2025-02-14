// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute

import (
	"go.opentelemetry.io/collector/component"
)

func RemoveInstanceAttributes(ts *component.TelemetrySettings, fields ...string) {
	ts.InstanceAttributes = RemoveAttributes(ts.InstanceAttributes)
	UpdateInstanceAttributes(ts)
}

func UpdateInstanceAttributes(ts *component.TelemetrySettings) {
	ts.Logger = LoggerWithAttributes(ts.Logger, ts.InstanceAttributes)
	ts.TracerProvider = TracerProviderWithAttributes(ts.TracerProvider, ts.InstanceAttributes)
	ts.MeterProvider = MeterProviderWithAttributes(ts.MeterProvider, ts.InstanceAttributes)
}
