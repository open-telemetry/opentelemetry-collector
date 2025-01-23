// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	"context"
	"errors"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
)

type Telemetry struct {
	Reader        *sdkmetric.ManualReader
	SpanRecorder  *tracetest.SpanRecorder
	meterProvider *sdkmetric.MeterProvider
	traceProvider *sdktrace.TracerProvider
}

func NewTelemetry() Telemetry {
	reader := sdkmetric.NewManualReader()
	spanRecorder := new(tracetest.SpanRecorder)
	return Telemetry{
		Reader:        reader,
		SpanRecorder:  spanRecorder,
		meterProvider: sdkmetric.NewMeterProvider(sdkmetric.WithResource(resource.Empty()), sdkmetric.WithReader(reader)),
		traceProvider: sdktrace.NewTracerProvider(sdktrace.WithResource(resource.Empty()), sdktrace.WithSpanProcessor(spanRecorder)),
	}
}

func (tt *Telemetry) NewTelemetrySettings() component.TelemetrySettings {
	set := NewNopTelemetrySettings()
	set.MeterProvider = tt.meterProvider
	set.MetricsLevel = configtelemetry.LevelDetailed
	set.TracerProvider = tt.traceProvider
	return set
}

func (tt *Telemetry) Shutdown(ctx context.Context) error {
	return errors.Join(
		tt.meterProvider.Shutdown(ctx),
		tt.traceProvider.Shutdown(ctx),
	)
}
