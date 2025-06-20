// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var (
	_                consumer.Metrics = obsMetrics{}
	metricsMarshaler                  = &pmetric.ProtoMarshaler{}
)

func NewMetrics(cons consumer.Metrics, itemCounter metric.Int64Counter, sizeCounter metric.Int64Counter, opts ...Option) consumer.Metrics {
	if !telemetry.NewPipelineTelemetryGate.IsEnabled() {
		return cons
	}

	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	return obsMetrics{
		consumer:        cons,
		itemCounter:     itemCounter,
		sizeCounter:     sizeCounter,
		compiledOptions: o.compile(),
	}
}

type obsMetrics struct {
	consumer    consumer.Metrics
	itemCounter metric.Int64Counter
	sizeCounter metric.Int64Counter
	compiledOptions
}

// ConsumeMetrics measures telemetry before calling ConsumeMetrics because the data may be mutated downstream
func (c obsMetrics) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	// Use a pointer to so that deferred function can depend on the result of ConsumeMetrics
	attrs := &c.withSuccessAttrs

	itemCount := md.DataPointCount()
	defer func() {
		c.itemCounter.Add(ctx, int64(itemCount), *attrs)
	}()

	if isEnabled(ctx, c.sizeCounter) {
		byteCount := int64(metricsMarshaler.MetricsSize(md))
		defer func() {
			c.sizeCounter.Add(ctx, byteCount, *attrs)
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
	}
	return err
}

func (c obsMetrics) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
