// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/service/internal/metadata"
)

var (
	_             consumer.Logs = obsLogs{}
	logsMarshaler               = &plog.ProtoMarshaler{}
)

func NewLogs(cons consumer.Logs, set Settings, opts ...Option) consumer.Logs {
	if !metadata.TelemetryNewPipelineTelemetryFeatureGate.IsEnabled() {
		return cons
	}

	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	consumerSet := Settings{
		ItemCounter:     set.ItemCounter,
		SizeCounter:     set.SizeCounter,
		BodySizeCounter: set.BodySizeCounter,
		Logger:          set.Logger.With(telemetry.ToZapFields(o.staticDataPointAttributes)...),
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

	if isEnabled(ctx, c.set.BodySizeCounter) {
		bodySize := logBodySize(ld)
		defer func() {
			c.set.BodySizeCounter.Add(ctx, bodySize, *attrs)
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

func logBodySize(ld plog.Logs) int64 {
	var total int64
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		rl := ld.ResourceLogs().At(i)
		for j := 0; j < rl.ScopeLogs().Len(); j++ {
			sl := rl.ScopeLogs().At(j)
			for k := 0; k < sl.LogRecords().Len(); k++ {
				total += valueSize(sl.LogRecords().At(k).Body())
			}
		}
	}
	return total
}

// valueSize estimates the byte size of a pcommon.Value without allocating.
// For string and bytes values it returns the exact length. For numeric and
// boolean types it returns the fixed in-memory size. For maps and slices it
// recursively sums leaf sizes (including map keys). This avoids the
// json.Marshal path that AsString() takes for structured types.
func valueSize(v pcommon.Value) int64 {
	switch v.Type() {
	case pcommon.ValueTypeStr:
		return int64(len(v.Str()))
	case pcommon.ValueTypeBytes:
		return int64(v.Bytes().Len())
	case pcommon.ValueTypeInt:
		return 8
	case pcommon.ValueTypeDouble:
		return 8
	case pcommon.ValueTypeBool:
		return 1
	case pcommon.ValueTypeMap:
		var size int64
		v.Map().Range(func(k string, val pcommon.Value) bool {
			size += int64(len(k)) + valueSize(val)
			return true
		})
		return size
	case pcommon.ValueTypeSlice:
		var size int64
		sl := v.Slice()
		for i := 0; i < sl.Len(); i++ {
			size += valueSize(sl.At(i))
		}
		return size
	default:
		return 0
	}
}
