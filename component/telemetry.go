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

// TelemetrySettings provides components with APIs to report telemetry.
//
// Note: there is a service version of this struct, servicetelemetry.TelemetrySettings, that mirrors
// this struct with the exception of ReportComponentStatus. When adding or removing anything from
// this struct consider whether or not the same should be done for the service version.
type TelemetrySettings struct {
	TracerProvider        trace.TracerProvider
	MeterProvider         metric.MeterProvider
	Resource              pcommon.Resource
	Logger                *zap.Logger
	ReportComponentStatus StatusFunc
	MetricsLevel          configtelemetry.Level
}

// Deprecated: [0.91.0] Use TelemetrySettings directly
type TelemetrySettingsBase[T any] struct {
	TracerProvider        trace.TracerProvider
	MeterProvider         metric.MeterProvider
	Resource              pcommon.Resource
	ReportComponentStatus T
	Logger                *zap.Logger
	MetricsLevel          configtelemetry.Level
}
