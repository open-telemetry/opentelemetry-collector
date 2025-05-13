// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var (
	_               consumer.Traces = tracesItemCounter{}
	_               consumer.Traces = tracesSizeCounter{}
	tracesMarshaler                 = &ptrace.ProtoMarshaler{}
)

func WithTracesItemCounter(itemCounter *metric.Int64Counter) Option {
	return func(opts *options) {
		opts.tracesItemCounter = itemCounter
	}
}

func WithTracesSizeCounter(sizeCounter *metric.Int64Counter) Option {
	return func(opts *options) {
		opts.tracesSizeCounter = sizeCounter
	}
}

func NewTraces(cons consumer.Traces, opts ...Option) consumer.Traces {
	if !telemetry.NewPipelineTelemetryGate.IsEnabled() {
		return cons
	}

	o := options{}
	for _, opt := range opts {
		opt(&o)
	}
	if o.tracesItemCounter == nil && o.tracesSizeCounter == nil {
		return cons
	}

	copts := o.compile()
	if o.tracesItemCounter != nil {
		cons = tracesItemCounter{
			consumer:        cons,
			itemCounter:     *o.tracesItemCounter,
			compiledOptions: copts,
		}
	}
	if o.tracesSizeCounter != nil {
		cons = tracesSizeCounter{
			consumer:        cons,
			sizeCounter:     *o.tracesSizeCounter,
			compiledOptions: copts,
		}
	}
	return cons
}

type tracesItemCounter struct {
	consumer    consumer.Traces
	itemCounter metric.Int64Counter
	compiledOptions
}

func (c tracesItemCounter) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	// Measure before calling ConsumeTraces because the data may be mutated downstream
	itemCount := td.SpanCount()
	err := c.consumer.ConsumeTraces(ctx, td)
	if err == nil {
		c.itemCounter.Add(ctx, int64(itemCount), c.withSuccessAttrs)
	} else {
		c.itemCounter.Add(ctx, int64(itemCount), c.withFailureAttrs)
	}
	return err
}

func (c tracesItemCounter) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}

type tracesSizeCounter struct {
	consumer    consumer.Traces
	sizeCounter metric.Int64Counter
	compiledOptions
}

func (c tracesSizeCounter) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	// Measure before calling ConsumeTraces because the data may be mutated downstream
	byteCount := tracesMarshaler.TracesSize(td)
	err := c.consumer.ConsumeTraces(ctx, td)
	if err == nil {
		c.sizeCounter.Add(ctx, int64(byteCount), c.withSuccessAttrs)
	} else {
		c.sizeCounter.Add(ctx, int64(byteCount), c.withFailureAttrs)
	}
	return err
}

func (c tracesSizeCounter) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
