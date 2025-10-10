// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute // import "go.opentelemetry.io/collector/service/internal/componentattribute"

import (
	"slices"

	"go.opentelemetry.io/otel/attribute"
	traceSdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// This is a copy of tracerProviderWithAttributes, but using the official SDK's
// concrete TracerProvider struct instead of the API's interface.
// This enables components like the zpagesextension to access SDK-only methods,
// such as for registering span processors.
//
// TODO: Remove once zpages trace functionality is moved into the service.
type tracerProviderWithAttributesSdk struct {
	*traceSdk.TracerProvider
	attrs []attribute.KeyValue
}

func (tpwa tracerProviderWithAttributesSdk) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	conf := trace.NewTracerConfig(options...)
	attrSet := conf.InstrumentationAttributes()
	// prepend our attributes so they can be overwritten
	newAttrs := append(slices.Clone(tpwa.attrs), attrSet.ToSlice()...)
	// append our attribute set option to overwrite the old one
	options = append(options, trace.WithInstrumentationAttributes(newAttrs...))
	return tpwa.TracerProvider.Tracer(name, options...)
}

func (tpwa tracerProviderWithAttributesSdk) DropInjectedAttributes(droppedAttrs ...string) trace.TracerProvider {
	return tracerProviderWithAttributesSdk{
		TracerProvider: tpwa.TracerProvider,
		attrs: slices.DeleteFunc(slices.Clone(tpwa.attrs), func(kv attribute.KeyValue) bool {
			return slices.Contains(droppedAttrs, string(kv.Key))
		}),
	}
}
