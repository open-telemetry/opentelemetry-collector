// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var (
	_                consumer.Metrics = metricsItemCounter{}
	_                consumer.Metrics = metricsSizeCounter{}
	metricsMarshaler                  = &pmetric.ProtoMarshaler{}
)

func WithMetricsItemCounter(itemCounter *metric.Int64Counter) Option {
	return func(opts *options) {
		opts.metricsItemCounter = itemCounter
	}
}

func WithMetricsSizeCounter(sizeCounter *metric.Int64Counter) Option {
	return func(opts *options) {
		opts.metricsSizeCounter = sizeCounter
	}
}

func NewMetrics(cons consumer.Metrics, opts ...Option) consumer.Metrics {
	if !telemetry.NewPipelineTelemetryGate.IsEnabled() {
		return cons
	}

	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	if o.metricsItemCounter == nil && o.metricsSizeCounter == nil {
		return cons
	}

	copts := o.compile()
	if o.metricsItemCounter != nil {
		cons = metricsItemCounter{
			consumer:        cons,
			itemCounter:     *o.metricsItemCounter,
			compiledOptions: copts,
		}
	}
	if o.metricsSizeCounter != nil {
		cons = metricsSizeCounter{
			consumer:        cons,
			sizeCounter:     *o.metricsSizeCounter,
			compiledOptions: copts,
		}
	}
	return cons
}

type metricsItemCounter struct {
	consumer    consumer.Metrics
	itemCounter metric.Int64Counter
	compiledOptions
}

func (c metricsItemCounter) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
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

func (c metricsItemCounter) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}

type metricsSizeCounter struct {
	consumer    consumer.Metrics
	sizeCounter metric.Int64Counter
	compiledOptions
}

func (c metricsSizeCounter) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	// Measure before calling ConsumeMetrics because the data may be mutated downstream
	byteCount := metricsMarshaler.MetricsSize(md)
	err := c.consumer.ConsumeMetrics(ctx, md)
	if err == nil {
		c.sizeCounter.Add(ctx, int64(byteCount), c.withSuccessAttrs)
	} else {
		c.sizeCounter.Add(ctx, int64(byteCount), c.withFailureAttrs)
	}
	return err
}

func (c metricsSizeCounter) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
