// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var (
	_               consumer.Traces = obsTraces{}
	tracesMarshaler                 = &ptrace.ProtoMarshaler{}
)

func NewTraces(cons consumer.Traces, set Settings, opts ...Option) consumer.Traces {
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

	return obsTraces{
		consumer:        cons,
		set:             consumerSet,
		compiledOptions: o.compile(),
	}
}

type obsTraces struct {
	consumer consumer.Traces
	set      Settings
	compiledOptions
}

// ConsumeTraces measures telemetry before calling ConsumeTraces because the data may be mutated downstream
func (c obsTraces) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	// Use a pointer to so that deferred function can depend on the result of ConsumeTraces
	attrs := &c.withSuccessAttrs

	itemCount := td.SpanCount()
	defer func() {
		c.set.ItemCounter.Add(ctx, int64(itemCount), *attrs)
	}()

	if isEnabled(ctx, c.set.SizeCounter) {
		byteCount := int64(tracesMarshaler.TracesSize(td))
		defer func() {
			c.set.SizeCounter.Add(ctx, byteCount, *attrs)
		}()
	}

	err := c.consumer.ConsumeTraces(ctx, td)
	if err != nil {
		if consumererror.IsDownstream(err) {
			attrs = &c.withRefusedAttrs
		} else {
			attrs = &c.withFailureAttrs
			err = consumererror.NewDownstream(err)
		}
		if c.set.Logger.Core().Enabled(zap.DebugLevel) {
			c.set.Logger.Debug("Traces pipeline component had an error", zap.Error(err), zap.Int("item count", itemCount))
		}
	}
	return err
}

func (c obsTraces) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
