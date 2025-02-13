package componentattribute

import (
	"context"
	"slices"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type tracerProviderWithAttributes struct {
	trace.TracerProvider
	option trace.SpanStartOption
}

func TracerProviderWithAttributes(tp trace.TracerProvider, attrs attribute.Set) trace.TracerProvider {
	if tpwa, ok := tp.(tracerProviderWithAttributes); ok {
		tp = tpwa.TracerProvider
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

func (twa tracerWithAttributes) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	opts = slices.Insert(opts, 0, twa.option)
	return twa.Tracer.Start(ctx, spanName, opts...)
}
