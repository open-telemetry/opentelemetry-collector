// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// NewNopTelemetrySettings returns a new nop telemetry settings for Create* functions.
func NewNopTelemetrySettings() component.TelemetrySettings {
	return component.TelemetrySettings{
		Logger: zap.NewNop(),
		LeveledMeterProvider: func(level configtelemetry.Level) metric.MeterProvider {
			return noopmetric.NewMeterProvider()
		},
		TracerProvider: nooptrace.NewTracerProvider(),
		MeterProvider:  noopmetric.NewMeterProvider(),
		MetricsLevel:   configtelemetry.LevelNone,
		Resource:       pcommon.NewResource(),
	}
}
