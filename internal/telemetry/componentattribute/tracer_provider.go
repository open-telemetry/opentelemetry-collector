// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute // import "go.opentelemetry.io/collector/internal/telemetry/componentattribute"

import (
	"context"
	"slices"

	"go.opentelemetry.io/otel/attribute"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type tracerProviderWithAttributes struct {
	trace.TracerProvider
	option trace.SpanStartOption
}

// Necessary for components that use SDK-only methods, such as zpagesextension
type tracerProviderWithAttributesSdk struct {
	*sdkTrace.TracerProvider
	option trace.SpanStartOption
}

func TracerProviderWithAttributes(tp trace.TracerProvider, attrs attribute.Set) trace.TracerProvider {
	if tpwa, ok := tp.(tracerProviderWithAttributesSdk); ok {
		tp = tpwa.TracerProvider
	} else if tpwa, ok := tp.(tracerProviderWithAttributes); ok {
		tp = tpwa.TracerProvider
	}
	if tpSdk, ok := tp.(*sdkTrace.TracerProvider); ok {
		return tracerProviderWithAttributesSdk{
			TracerProvider: tpSdk,
			option:         trace.WithAttributes(attrs.ToSlice()...),
		}
	}
	return tracerProviderWithAttributes{
		TracerProvider: tp,
		option:         trace.WithAttributes(attrs.ToSlice()...),
	}
}

type tracerWithAttributes struct {
	trace.Tracer
	option trace.SpanStartOption
}

func (tpwa tracerProviderWithAttributes) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	return tracerWithAttributes{
		Tracer: tpwa.TracerProvider.Tracer(name, options...),
		option: tpwa.option,
	}
}

func (tpwa tracerProviderWithAttributesSdk) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	return tracerWithAttributes{
		Tracer: tpwa.TracerProvider.Tracer(name, options...),
		option: tpwa.option,
	}
}

func (twa tracerWithAttributes) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	opts = slices.Insert(opts, 0, twa.option)
	return twa.Tracer.Start(ctx, spanName, opts...)
}
