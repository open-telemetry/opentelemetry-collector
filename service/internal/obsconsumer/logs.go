// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pdata/plog"
)

var (
	_             consumer.Logs = logsItemCounter{}
	_             consumer.Logs = logsSizeCounter{}
	logsMarshaler               = &plog.ProtoMarshaler{}
)

func WithLogsItemCounter(itemCounter *metric.Int64Counter) Option {
	return func(opts *options) {
		opts.logsItemCounter = itemCounter
	}
}

func WithLogsSizeCounter(sizeCounter *metric.Int64Counter) Option {
	return func(opts *options) {
		opts.logsSizeCounter = sizeCounter
	}
}

func NewLogs(cons consumer.Logs, opts ...Option) consumer.Logs {
	if !telemetry.NewPipelineTelemetryGate.IsEnabled() {
		return cons
	}

	o := options{}
	for _, opt := range opts {
		opt(&o)
	}
	if o.logsItemCounter == nil && o.logsSizeCounter == nil {
		return cons
	}

	copts := o.compile()
	if o.logsItemCounter != nil {
		cons = logsItemCounter{
			consumer:        cons,
			itemCounter:     *o.logsItemCounter,
			compiledOptions: copts,
		}
	}
	if o.logsSizeCounter != nil {
		cons = logsSizeCounter{
			consumer:        cons,
			sizeCounter:     *o.logsSizeCounter,
			compiledOptions: copts,
		}
	}
	return cons
}

type logsItemCounter struct {
	consumer    consumer.Logs
	itemCounter metric.Int64Counter
	compiledOptions
}

func (c logsItemCounter) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
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

func (c logsItemCounter) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}

type logsSizeCounter struct {
	consumer    consumer.Logs
	sizeCounter metric.Int64Counter
	compiledOptions
}

func (c logsSizeCounter) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	// Measure before calling ConsumeLogs because the data may be mutated downstream
	byteCount := logsMarshaler.LogsSize(ld)
	err := c.consumer.ConsumeLogs(ctx, ld)
	if err == nil {
		c.sizeCounter.Add(ctx, int64(byteCount), c.withSuccessAttrs)
	} else {
		c.sizeCounter.Add(ctx, int64(byteCount), c.withFailureAttrs)
	}
	return err
}

func (c logsSizeCounter) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
