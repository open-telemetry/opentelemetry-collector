// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var (
	_                consumer.Metrics = obsMetrics{}
	metricsMarshaler                  = &pmetric.ProtoMarshaler{}
)

func NewMetrics(cons consumer.Metrics, set Settings, opts ...Option) consumer.Metrics {
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

	return obsMetrics{
		consumer:        cons,
		set:             consumerSet,
		compiledOptions: o.compile(),
	}
}

type obsMetrics struct {
	consumer consumer.Metrics
	set      Settings
	compiledOptions
}

// ConsumeMetrics measures telemetry before calling ConsumeMetrics because the data may be mutated downstream
func (c obsMetrics) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	// Use a pointer to so that deferred function can depend on the result of ConsumeMetrics
	attrs := &c.withSuccessAttrs

	itemCount := md.DataPointCount()
	defer func() {
		c.set.ItemCounter.Add(ctx, int64(itemCount), *attrs)
	}()

	if isEnabled(ctx, c.set.SizeCounter) {
		byteCount := int64(metricsMarshaler.MetricsSize(md))
		defer func() {
			c.set.SizeCounter.Add(ctx, byteCount, *attrs)
		}()
	}

	err := c.consumer.ConsumeMetrics(ctx, md)
	if err != nil {
		if consumererror.IsDownstream(err) {
			attrs = &c.withRefusedAttrs
		} else {
			attrs = &c.withFailureAttrs
			err = consumererror.NewDownstream(err)
		}
		if c.set.Logger.Core().Enabled(zap.DebugLevel) {
			c.set.Logger.Debug("Metrics pipeline component had an error", zap.Error(err), zap.Int("item count", itemCount))
		}
	}
	return err
}

func (c obsMetrics) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
