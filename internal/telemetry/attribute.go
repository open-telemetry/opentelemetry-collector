// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/internal/telemetry/componentattribute"

import (
	"slices"

	"go.opentelemetry.io/otel/attribute"
)

const (
	ComponentKindKey = "otelcol.component.kind"
	ComponentIDKey   = "otelcol.component.id"
	PipelineIDKey    = "otelcol.pipeline.id"
	SignalKey        = "otelcol.signal"
	SignalOutputKey  = "otelcol.signal.output"
)

func RemoveAttributes(t TelemetrySettings, fields ...string) attribute.Set {
	attrs, _ := attribute.NewSetWithFiltered(t.extraAttributes.ToSlice(), func(kv attribute.KeyValue) bool {
		return !slices.Contains(fields, string(kv.Key))
	})
	return attrs
}
