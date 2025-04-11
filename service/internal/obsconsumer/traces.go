// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var _ consumer.Traces = Traces{}

func NewTraces(consumer consumer.Traces, itemCounter metric.Int64Counter, opts ...Option) Traces {
	o := options{}
	for _, opt := range opts {
		opt.apply(&o)
	}
	return Traces{
		consumer:        consumer,
		itemCounter:     itemCounter,
		compiledOptions: o.compile(),
	}
}

type Traces struct {
	consumer    consumer.Traces
	itemCounter metric.Int64Counter
	compiledOptions
}

func (c Traces) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
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

func (c Traces) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
