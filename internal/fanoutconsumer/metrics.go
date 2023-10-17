// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fanoutconsumer // import "go.opentelemetry.io/collector/internal/fanoutconsumer"

import (
	"context"
	"fmt"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// NewMetrics wraps multiple metrics consumers in a single one.
// It fanouts the incoming data to all the consumers, and does smart routing:
//   - Clones only to the consumer that needs to mutate the data.
//   - If all consumers needs to mutate the data one will get the original mutable data.
func NewMetrics(mcs []consumer.Metrics) consumer.Metrics {
	// Don't wrap if there is only one non-mutating consumer.
	if len(mcs) == 1 && !mcs[0].Capabilities().MutatesData {
		return mcs[0]
	}

	mc := &metricsConsumer{}
	for i := 0; i < len(mcs); i++ {
		if mcs[i].Capabilities().MutatesData {
			mc.mutable = append(mc.mutable, mcs[i])
		} else {
			mc.readonly = append(mc.readonly, mcs[i])
		}
	}
	return mc
}

type metricsConsumer struct {
	mutable  []consumer.Metrics
	readonly []consumer.Metrics
}

func (msc *metricsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeMetrics exports the pmetric.Metrics to all consumers wrapped by the current one.
func (msc *metricsConsumer) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	var errs error

	if len(msc.mutable) > 0 {
		// Clone the data before sending to all mutating consumers except the last one.
		for i := 0; i < len(msc.mutable)-1; i++ {
			errs = multierr.Append(errs, msc.mutable[i].ConsumeMetrics(ctx, cloneMetrics(md)))
		}
		// Send data as is to the last mutating consumer only if there are no other non-mutating consumers and the
		// data is mutable. Never share the same data between a mutating and a non-mutating consumer since the
		// non-mutating consumer may process data async and the mutating consumer may change the data before that.
		lastConsumer := msc.mutable[len(msc.mutable)-1]
		if len(msc.readonly) == 0 && !md.IsReadOnly() {
			errs = multierr.Append(errs, lastConsumer.ConsumeMetrics(ctx, md))
		} else {
			errs = multierr.Append(errs, lastConsumer.ConsumeMetrics(ctx, cloneMetrics(md)))
		}
	}

	// Mark the data as read-only if it will be sent to more than one read-only consumer.
	if len(msc.readonly) > 1 && !md.IsReadOnly() {
		md.MarkReadOnly()
	}
	for _, mc := range msc.readonly {
		errs = multierr.Append(errs, mc.ConsumeMetrics(ctx, md))
	}

	return errs
}

func cloneMetrics(md pmetric.Metrics) pmetric.Metrics {
	clonedMetrics := pmetric.NewMetrics()
	md.CopyTo(clonedMetrics)
	return clonedMetrics
}

var _ connector.MetricsRouter = (*metricsRouter)(nil)

type metricsRouter struct {
	consumer.Metrics
	consumers map[component.ID]consumer.Metrics
}

func NewMetricsRouter(cm map[component.ID]consumer.Metrics) consumer.Metrics {
	consumers := make([]consumer.Metrics, 0, len(cm))
	for _, consumer := range cm {
		consumers = append(consumers, consumer)
	}
	return &metricsRouter{
		Metrics:   NewMetrics(consumers),
		consumers: cm,
	}
}

func (r *metricsRouter) PipelineIDs() []component.ID {
	ids := make([]component.ID, 0, len(r.consumers))
	for id := range r.consumers {
		ids = append(ids, id)
	}
	return ids
}

func (r *metricsRouter) Consumer(pipelineIDs ...component.ID) (consumer.Metrics, error) {
	if len(pipelineIDs) == 0 {
		return nil, fmt.Errorf("missing consumers")
	}
	consumers := make([]consumer.Metrics, 0, len(pipelineIDs))
	var errors error
	for _, pipelineID := range pipelineIDs {
		c, ok := r.consumers[pipelineID]
		if ok {
			consumers = append(consumers, c)
		} else {
			errors = multierr.Append(errors, fmt.Errorf("missing consumer: %q", pipelineID))
		}
	}
	if errors != nil {
		// TODO potentially this could return a NewMetrics with the valid consumers
		return nil, errors
	}
	return NewMetrics(consumers), nil
}
