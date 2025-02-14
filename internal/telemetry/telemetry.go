package telemetry

import (
	"go.opentelemetry.io/collector/internal/telemetry/componentattribute"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type TelemetrySettings struct {
	// Logger that the factory can use during creation and can pass to the created
	// component to be used later as well.
	Logger *zap.Logger

	// TracerProvider that the factory can pass to other instrumented third-party libraries.
	TracerProvider trace.TracerProvider

	// MeterProvider that the factory can pass to other instrumented third-party libraries.
	MeterProvider metric.MeterProvider

	// Resource contains the resource attributes for the collector's telemetry.
	Resource pcommon.Resource

	// Extra attributes added to spans, metric points, and log records
	extraAttributes attribute.Set
}

// Once this API is ready to be made public:
// - the two functions should be made methods of TelemetrySettings
// - the three declarations should be moved back into `component/telemetry.go`

func WithoutAttributes(ts TelemetrySettings, fields ...string) TelemetrySettings {
	return WithAttributeSet(ts, componentattribute.RemoveAttributes(ts.extraAttributes, fields...))
}

func WithAttributeSet(ts TelemetrySettings, attrs attribute.Set) TelemetrySettings {
	ts.extraAttributes = attrs
	ts.Logger = componentattribute.LoggerWithAttributes(ts.Logger, ts.extraAttributes)
	ts.TracerProvider = componentattribute.TracerProviderWithAttributes(ts.TracerProvider, ts.extraAttributes)
	ts.MeterProvider = componentattribute.MeterProviderWithAttributes(ts.MeterProvider, ts.extraAttributes)
	return ts
}
