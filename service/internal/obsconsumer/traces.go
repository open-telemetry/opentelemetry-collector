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
	_               consumer.Traces = obsTraces{}
	tracesMarshaler                 = &ptrace.ProtoMarshaler{}
)

func NewTraces(cons consumer.Traces, itemCounter metric.Int64Counter, sizeCounter metric.Int64Counter, opts ...Option) consumer.Traces {
	if !telemetry.NewPipelineTelemetryGate.IsEnabled() {
		return cons
	}

	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	return obsTraces{
		consumer:        cons,
		itemCounter:     itemCounter,
		sizeCounter:     sizeCounter,
		compiledOptions: o.compile(),
	}
}

type obsTraces struct {
	consumer    consumer.Traces
	itemCounter metric.Int64Counter
	sizeCounter metric.Int64Counter
	compiledOptions
}

// ConsumeTraces measures telemetry before calling ConsumeTraces because the data may be mutated downstream
func (c obsTraces) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	// Use a pointer to so that deferred function can depend on the result of ConsumeTraces
	attrs := &c.withSuccessAttrs

	itemCount := td.SpanCount()
	defer func() {
		c.itemCounter.Add(ctx, int64(itemCount), *attrs)
	}()

	if isEnabled(ctx, c.sizeCounter) {
		byteCount := int64(tracesMarshaler.TracesSize(td))
		defer func() {
			c.sizeCounter.Add(ctx, byteCount, *attrs)
		}()
	}

	err := c.consumer.ConsumeTraces(ctx, td)
	if err != nil {
		attrs = &c.withFailureAttrs
	}
	return err
}

func (c obsTraces) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
