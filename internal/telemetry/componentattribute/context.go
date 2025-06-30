// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute // import "go.opentelemetry.io/collector/internal/telemetry/componentattribute"

import (
	"context"
	"slices"

	"go.opentelemetry.io/otel/attribute"
)

type contextAttributesKey struct{}

// ContextWithAttributes returns a derived context that points to the parent Context,
// with the given attributes associated. These attributes will be added to any spans
// and synchronous metric measurements made within the context.
//
// Any existing attributes in the context will be preserved, and the new attributes
// will be added to them. Duplicates will be eliminated by taking the last value for
// each key.
//
// NOTE: attributes will NOT not be added to log records, as the collector does not
// currently support contextual logging. See:
// https://github.com/open-telemetry/opentelemetry-collector/issues/5962
func ContextWithAttributes(ctx context.Context, attrs ...attribute.KeyValue) context.Context {
	if len(attrs) == 0 {
		return ctx
	}
	existing, ok := ctx.Value(contextAttributesKey{}).(attribute.Set)
	if ok {
		attrs = slices.Concat(existing.ToSlice(), attrs)
	}
	return context.WithValue(ctx, contextAttributesKey{}, attribute.NewSet(attrs...))
}

func attributesFromContext(ctx context.Context) (attribute.Set, bool) {
	set, ok := ctx.Value(contextAttributesKey{}).(attribute.Set)
	return set, ok
}
