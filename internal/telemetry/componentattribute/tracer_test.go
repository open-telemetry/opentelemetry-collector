// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"

	"go.opentelemetry.io/collector/internal/telemetry/componentattribute"
)

// Emulate a TracerProvider from a non-official SDK
type customTracerProvider struct {
	embedded.TracerProvider
	tp *sdkTrace.TracerProvider
}

func (ctp customTracerProvider) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	return ctp.tp.Tracer(name, options...)
}

var _ trace.TracerProvider = customTracerProvider{}

func TestTPWA(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp1 := sdkTrace.NewTracerProvider(sdkTrace.WithSpanProcessor(sdkTrace.NewSimpleSpanProcessor(exporter)))

	// Add attribute set
	extraAttrs2 := attribute.NewSet(
		attribute.String("extrakey2", "extraval2"),
	)
	tp2 := componentattribute.TracerProviderWithAttributes(tp1, extraAttrs2)

	// Replace attribute set
	extraAttrs3 := attribute.NewSet(
		attribute.String("extrakey3", "extraval3"),
	)
	tp3 := componentattribute.TracerProviderWithAttributes(tp2, extraAttrs3)

	// The same thing, but with a non-official SDK Provider
	tp4 := &customTracerProvider{tp: tp1}
	tp5 := componentattribute.TracerProviderWithAttributes(tp4, extraAttrs2)
	tp6 := componentattribute.TracerProviderWithAttributes(tp5, extraAttrs3)

	noAttrs := attribute.NewSet()
	// Add a standard attribute on top of the extra attributes
	attrs4 := attribute.NewSet(
		attribute.String("stdkey", "stdval"),
	)
	expAttrs4 := attribute.NewSet(append(extraAttrs3.ToSlice(), attrs4.ToSlice()...)...)
	// Overwrite the extra attribute
	attrs5 := attribute.NewSet(
		attribute.String("extrakey3", "customval"),
	)

	tests := []struct {
		tp       trace.TracerProvider
		attrs    attribute.Set
		expAttrs attribute.Set
		name     string
	}{
		{tp: tp1, attrs: noAttrs, expAttrs: noAttrs, name: "no extra attributes"},
		{tp: tp2, attrs: noAttrs, expAttrs: extraAttrs2, name: "set extra attributes"},
		{tp: tp3, attrs: noAttrs, expAttrs: extraAttrs3, name: "reset extra attributes"},
		{tp: tp3, attrs: attrs4, expAttrs: expAttrs4, name: "merge attributes"},
		{tp: tp3, attrs: attrs5, expAttrs: attrs5, name: "overwrite extra attribute"},
		{tp: tp4, attrs: noAttrs, expAttrs: noAttrs, name: "no extra attributes, non-official SDK"},
		{tp: tp5, attrs: noAttrs, expAttrs: extraAttrs2, name: "set extra attributes, non-official SDK"},
		{tp: tp6, attrs: noAttrs, expAttrs: extraAttrs3, name: "reset extra attributes, non-official SDK"},
	}

	for i, test := range tests {
		tracerName := fmt.Sprintf("testtracer%d", i+1)
		_, span := test.tp.Tracer(
			tracerName,
			trace.WithInstrumentationAttributes(test.attrs.ToSlice()...),
		).Start(context.Background(), "testspan")
		span.End()
	}

	spans := exporter.GetSpans()
	require.LessOrEqual(t, len(spans), len(tests))

	for i, test := range tests {
		t.Run(test.name+"/check", func(t *testing.T) {
			tracerName := fmt.Sprintf("testtracer%d", i+1)
			i := slices.IndexFunc(spans, func(s tracetest.SpanStub) bool {
				return s.InstrumentationScope.Name == tracerName
			})
			assert.NotEqual(t, -1, i)
			span := spans[i]
			assert.Equal(t, "testspan", span.Name)
			assert.Equal(t, test.expAttrs, span.InstrumentationScope.Attributes)
		})
	}
}
