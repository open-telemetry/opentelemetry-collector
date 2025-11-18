// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute // import "go.opentelemetry.io/collector/service/internal/componentattribute"

import (
	"slices"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type tracerProviderWithAttributes struct {
	trace.TracerProvider
	attrs []attribute.KeyValue
}

func (tpwa tracerProviderWithAttributes) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	conf := trace.NewTracerConfig(options...)
	attrSet := conf.InstrumentationAttributes()
	// prepend our attributes so they can be overwritten
	newAttrs := append(slices.Clone(tpwa.attrs), attrSet.ToSlice()...)
	// append our attribute set option to overwrite the old one
	options = append(options, trace.WithInstrumentationAttributes(newAttrs...))
	return tpwa.TracerProvider.Tracer(name, options...)
}

func (tpwa tracerProviderWithAttributes) Unwrap() trace.TracerProvider {
	return tpwa.TracerProvider
}

func (tpwa tracerProviderWithAttributes) DropInjectedAttributes(droppedAttrs ...string) trace.TracerProvider {
	return tracerProviderWithAttributes{
		TracerProvider: tpwa.TracerProvider,
		attrs: slices.DeleteFunc(slices.Clone(tpwa.attrs), func(kv attribute.KeyValue) bool {
			return slices.Contains(droppedAttrs, string(kv.Key))
		}),
	}
}
