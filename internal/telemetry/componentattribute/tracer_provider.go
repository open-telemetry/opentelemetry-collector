// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute // import "go.opentelemetry.io/collector/internal/telemetry/componentattribute"

import (
	"slices"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type tracerProviderWithAttributes struct {
	trace.TracerProvider
	attrs []attribute.KeyValue
}

// TracerProviderWithAttributes creates a TracerProvider with a new set of injected instrumentation scope attributes.
func TracerProviderWithAttributes(tp trace.TracerProvider, attrs attribute.Set) trace.TracerProvider {
	if tpwa, ok := tp.(tracerProviderWithAttributes); ok {
		tp = tpwa.TracerProvider
	}
	return tracerProviderWithAttributes{
		TracerProvider: tp,
		attrs:          attrs.ToSlice(),
	}
}

func tracerWithAttributes(tp trace.TracerProvider, attrs []attribute.KeyValue, name string, opts ...trace.TracerOption) trace.Tracer {
	conf := trace.NewTracerConfig(opts...)
	attrSet := conf.InstrumentationAttributes()
	// prepend our attributes so they can be overwritten
	newAttrs := append(slices.Clone(attrs), attrSet.ToSlice()...)
	// append our attribute set option to overwrite the old one
	opts = append(opts, trace.WithInstrumentationAttributes(newAttrs...))
	return tp.Tracer(name, opts...)
}

func (tpwa tracerProviderWithAttributes) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	return tracerWithAttributes(tpwa.TracerProvider, tpwa.attrs, name, options...)
}

func (tpwa tracerProviderWithAttributes) Unwrap() trace.TracerProvider {
	return tpwa.TracerProvider
}
