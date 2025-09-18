// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/internal/telemetry"

import (
	"slices"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// IMPORTANT: This struct is reexported as part of the public API of
// go.opentelemetry.io/collector/component, a stable module.
// DO NOT MAKE BREAKING CHANGES TO EXPORTED FIELDS.
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

	// Extra attributes added to instrumentation scopes
	extraAttributes attribute.Set
}

// The publicization of this API is tracked in https://github.com/open-telemetry/opentelemetry-collector/issues/12405

func (t TelemetrySettings) Attributes() attribute.Set {
	return t.extraAttributes
}

func WithAttributeSet(ts TelemetrySettings, attrs attribute.Set) TelemetrySettings {
	ts.extraAttributes = attrs
	return ts
}

func WithoutAttributes(ts TelemetrySettings, fields ...string) TelemetrySettings {
	ts.extraAttributes = removeAttributes(ts, fields...)
	return ts
}

func removeAttributes(t TelemetrySettings, fields ...string) attribute.Set {
	attrs, _ := attribute.NewSetWithFiltered(t.extraAttributes.ToSlice(), func(kv attribute.KeyValue) bool {
		return !slices.Contains(fields, string(kv.Key))
	})
	return attrs
}
