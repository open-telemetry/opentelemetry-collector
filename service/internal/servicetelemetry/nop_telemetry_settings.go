// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package servicetelemetry // import "go.opentelemetry.io/collector/service/internal/servicetelemetry"

import (
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/service/internal/status"
)

// NewNopTelemetrySettings returns a new nop settings for Create* functions.
func NewNopTelemetrySettings() TelemetrySettings {
	return TelemetrySettings{
		Logger:         zap.NewNop(),
		TracerProvider: nooptrace.NewTracerProvider(),
		MeterProvider:  noopmetric.NewMeterProvider(),
		MetricsLevel:   configtelemetry.LevelNone,
		Resource:       pcommon.NewResource(),
		Status:         status.NewReporter(func(*component.InstanceID, *component.StatusEvent) {}, func(err error) {}),
	}
}
