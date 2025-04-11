// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
)

var _ consumer.Logs = Logs{}

func NewLogs(consumer consumer.Logs, itemCounter metric.Int64Counter, opts ...Option) Logs {
	o := options{}
	for _, opt := range opts {
		opt.apply(&o)
	}
	return Logs{
		consumer:        consumer,
		itemCounter:     itemCounter,
		compiledOptions: o.compile(),
	}
}

type Logs struct {
	consumer    consumer.Logs
	itemCounter metric.Int64Counter
	compiledOptions
}

func (c Logs) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	// Measure before calling ConsumeLogs because the data may be mutated downstream
	itemCount := ld.LogRecordCount()
	err := c.consumer.ConsumeLogs(ctx, ld)
	if err == nil {
		c.itemCounter.Add(ctx, int64(itemCount), c.withSuccessAttrs)
	} else {
		c.itemCounter.Add(ctx, int64(itemCount), c.withFailureAttrs)
	}
	return err
}

func (c Logs) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
