// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var _ consumer.Metrics = Metrics{}

func NewMetrics(consumer consumer.Metrics, itemCounter metric.Int64Counter, opts ...Option) Metrics {
	o := options{}
	for _, opt := range opts {
		opt.apply(&o)
	}
	return Metrics{
		consumer:        consumer,
		itemCounter:     itemCounter,
		compiledOptions: o.compile(),
	}
}

type Metrics struct {
	consumer    consumer.Metrics
	itemCounter metric.Int64Counter
	compiledOptions
}

func (c Metrics) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	// Measure before calling ConsumeMetrics because the data may be mutated downstream
	itemCount := md.DataPointCount()
	err := c.consumer.ConsumeMetrics(ctx, md)
	if err == nil {
		c.itemCounter.Add(ctx, int64(itemCount), c.withSuccessAttrs)
	} else {
		c.itemCounter.Add(ctx, int64(itemCount), c.withFailureAttrs)
	}
	return err
}

func (c Metrics) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
