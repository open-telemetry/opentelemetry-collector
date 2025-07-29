// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pdata/plog"
)

var (
	_             consumer.Logs = obsLogs{}
	logsMarshaler               = &plog.ProtoMarshaler{}
)

func NewLogs(cons consumer.Logs, itemCounter metric.Int64Counter, sizeCounter metric.Int64Counter, opts ...Option) consumer.Logs {
	if !telemetry.NewPipelineTelemetryGate.IsEnabled() {
		return cons
	}

	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	return obsLogs{
		consumer:        cons,
		itemCounter:     itemCounter,
		sizeCounter:     sizeCounter,
		compiledOptions: o.compile(),
	}
}

type obsLogs struct {
	consumer    consumer.Logs
	itemCounter metric.Int64Counter
	sizeCounter metric.Int64Counter
	compiledOptions
}

// ConsumeLogs measures telemetry before calling ConsumeLogs because the data may be mutated downstream
func (c obsLogs) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	// Use a pointer to so that deferred function can depend on the result of ConsumeLogs
	attrs := &c.withSuccessAttrs

	itemCount := ld.LogRecordCount()
	defer func() {
		c.itemCounter.Add(ctx, int64(itemCount), *attrs)
	}()

	if isEnabled(ctx, c.sizeCounter) {
		byteCount := int64(logsMarshaler.LogsSize(ld))
		defer func() {
			c.sizeCounter.Add(ctx, byteCount, *attrs)
		}()
	}

	err := c.consumer.ConsumeLogs(ctx, ld)
	if err != nil {
		if consumererror.IsDownstream(err) {
			attrs = &c.withRefusedAttrs
		} else {
			attrs = &c.withFailureAttrs
			err = consumererror.NewDownstream(err)
		}
	}
	return err
}

func (c obsLogs) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
