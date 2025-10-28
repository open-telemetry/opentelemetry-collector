// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pdata/plog"
)

var (
	_             consumer.Logs = obsLogs{}
	logsMarshaler               = &plog.ProtoMarshaler{}
)

func NewLogs(cons consumer.Logs, set Settings, opts ...Option) consumer.Logs {
	if !telemetry.NewPipelineTelemetryGate.IsEnabled() {
		return cons
	}

	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	consumerSet := Settings{
		ItemCounter: set.ItemCounter,
		SizeCounter: set.SizeCounter,
		Logger:      set.Logger.With(telemetry.ToZapFields(o.staticDataPointAttributes)...),
	}

	return obsLogs{
		consumer:        cons,
		set:             consumerSet,
		compiledOptions: o.compile(),
	}
}

type obsLogs struct {
	consumer consumer.Logs
	set      Settings
	compiledOptions
}

// ConsumeLogs measures telemetry before calling ConsumeLogs because the data may be mutated downstream
func (c obsLogs) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	// Use a pointer to so that deferred function can depend on the result of ConsumeLogs
	attrs := &c.withSuccessAttrs

	itemCount := ld.LogRecordCount()
	defer func() {
		c.set.ItemCounter.Add(ctx, int64(itemCount), *attrs)
	}()

	if isEnabled(ctx, c.set.SizeCounter) {
		byteCount := int64(logsMarshaler.LogsSize(ld))
		defer func() {
			c.set.SizeCounter.Add(ctx, byteCount, *attrs)
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
		if c.set.Logger.Core().Enabled(zap.DebugLevel) {
			c.set.Logger.Debug("Logs pipeline component had an error", zap.Error(err), zap.Int("item count", itemCount))
		}
	}
	return err
}

func (c obsLogs) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
