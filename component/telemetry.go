// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component // import "go.opentelemetry.io/collector/component"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/internal/telemetry/componentattribute"
)

// TelemetrySettings provides components with APIs to report telemetry.
type TelemetrySettings = telemetry.TelemetrySettings

// ContextWithAttributes returns a derived context that points to the parent Context,
// with the given attributes associated. These attributes will be added to any spans
// and synchronous metric measurements made within the context.
func ContextWithAttributes(ctx context.Context, attrs ...attribute.KeyValue) context.Context {
	return componentattribute.ContextWithAttributes(ctx, attrs...)
}
