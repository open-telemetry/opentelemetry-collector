// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	"context"
	"errors"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
)

type TelemetryOption interface {
	apply(*telemetryOption)
}

type telemetryOption struct {
	metricOpts []sdkmetric.Option
	traceOpts  []sdktrace.TracerProviderOption
}

type telemetryOptionFunc func(*telemetryOption)

func (f telemetryOptionFunc) apply(o *telemetryOption) { f(o) }

func WithMetricOptions(opts ...sdkmetric.Option) TelemetryOption {
	return telemetryOptionFunc(func(to *telemetryOption) {
		to.metricOpts = append(to.metricOpts, opts...)
	})
}

func WithTraceOptions(opts ...sdktrace.TracerProviderOption) TelemetryOption {
	return telemetryOptionFunc(func(to *telemetryOption) {
		to.traceOpts = append(to.traceOpts, opts...)
	})
}

type Telemetry struct {
	Reader        *sdkmetric.ManualReader
	SpanRecorder  *tracetest.SpanRecorder
	meterProvider *sdkmetric.MeterProvider
	traceProvider *sdktrace.TracerProvider
}

func NewTelemetry(opts ...TelemetryOption) Telemetry {
	reader := sdkmetric.NewManualReader()
	spanRecorder := new(tracetest.SpanRecorder)
	tOpts := telemetryOption{
		metricOpts: []sdkmetric.Option{sdkmetric.WithReader(reader)},
		traceOpts:  []sdktrace.TracerProviderOption{sdktrace.WithSpanProcessor(spanRecorder)},
	}
	for _, opt := range opts {
		opt.apply(&tOpts)
	}
	return Telemetry{
		Reader:        reader,
		SpanRecorder:  spanRecorder,
		meterProvider: sdkmetric.NewMeterProvider(tOpts.metricOpts...),
		traceProvider: sdktrace.NewTracerProvider(tOpts.traceOpts...),
	}
}

func (tt *Telemetry) NewTelemetrySettings() component.TelemetrySettings {
	set := NewNopTelemetrySettings()
	set.MeterProvider = tt.meterProvider
	set.MetricsLevel = configtelemetry.LevelDetailed //nolint:staticcheck //SA1019
	set.TracerProvider = tt.traceProvider
	return set
}

func (tt *Telemetry) Shutdown(ctx context.Context) error {
	return errors.Join(
		tt.meterProvider.Shutdown(ctx),
		tt.traceProvider.Shutdown(ctx),
	)
}
