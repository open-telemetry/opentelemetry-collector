// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component // import "go.opentelemetry.io/collector/component"

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/featuregate"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

var TelemetryDisableOldFormatMetrics = featuregate.GlobalRegistry().MustRegister(
	"telemetry.disableOldFormatMetrics",
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription("When enabled, disables all metrics which use _ delimiter."),
	featuregate.WithRegisterReferenceURL("https://github.com/open-telemetry/opentelemetry-collector/issues/12909"),
	featuregate.WithRegisterFromVersion("v0.133.0"),
)

var TelemetryEnableNewFormatMetrics = featuregate.GlobalRegistry().MustRegister(
	"telemetry.enableNewFormatMetrics",
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription("When enabled, enables equivalent metrics to those which disableOldFormatMetrics disables (e.g. otelcol_exporter_send_failed_spans becomes otelcol.exporter.send.failed.spans)."),
	featuregate.WithRegisterReferenceURL("https://github.com/open-telemetry/opentelemetry-collector/issues/12909"),
	featuregate.WithRegisterFromVersion("v0.133.0"),
)

// TelemetrySettings provides components with APIs to report telemetry.
type TelemetrySettings struct {
	// Logger that the factory can use during creation and can pass to the created
	// component to be used later as well.
	Logger *zap.Logger

	// TracerProvider that the factory can pass to other instrumented third-party libraries.
	//
	// The service may wrap this provider for attribute injection. The wrapper may implement an
	// additional `Unwrap() trace.TracerProvider` method to grant access to the underlying SDK.
	TracerProvider trace.TracerProvider

	// MeterProvider that the factory can pass to other instrumented third-party libraries.
	MeterProvider metric.MeterProvider

	// Resource contains the resource attributes for the collector's telemetry.
	Resource pcommon.Resource

	// prevent unkeyed literal initialization
	_ struct{}
}
