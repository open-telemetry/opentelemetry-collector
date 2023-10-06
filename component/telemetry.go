// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component // import "go.opentelemetry.io/collector/component"

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

type TelemetrySettingsBase[T any] struct {
	// Logger that the factory can use during creation and can pass to the created
	// component to be used later as well.
	Logger *zap.Logger

	// TracerProvider that the factory can pass to other instrumented third-party libraries.
	TracerProvider trace.TracerProvider

	// MeterProvider that the factory can pass to other instrumented third-party libraries.
	MeterProvider metric.MeterProvider

	// MetricsLevel controls the level of detail for metrics emitted by the collector.
	// Experimental: *NOTE* this field is experimental and may be changed or removed.
	MetricsLevel configtelemetry.Level

	// Resource contains the resource attributes for the collector's telemetry.
	Resource pcommon.Resource

	// ReportComponentStatus allows a component to report runtime changes in status. The service
	// will automatically report status for a component during startup and shutdown. Components can
	// use this method to report status after start and before shutdown. ReportComponentStatus
	// will only return errors if the API used incorrectly. The two scenarios where an error will
	// be returned are:
	//
	//   - An illegal state transition
	//   - Calling this method before component startup
	//
	// If the API is being used properly, these errors are safe to ignore.
	ReportComponentStatus T
}

// TelemetrySettings and servicetelemetry.Settings differ in the method signature for
// ReportComponentStatus
type TelemetrySettings TelemetrySettingsBase[StatusFunc]
