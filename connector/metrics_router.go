// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"fmt"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
)

// Deprecated: [v0.92.0] use MetricsRouterAndConsumer.
type MetricsRouter interface {
	Consumer(...component.ID) (consumer.Metrics, error)
	PipelineIDs() []component.ID
}

// MetricsRouterAndConsumer feeds the first consumer.Metrics in each of the specified pipelines.
type MetricsRouterAndConsumer interface {
	consumer.Metrics
	Consumer(...component.ID) (consumer.Metrics, error)
	PipelineIDs() []component.ID
	privateFunc()
}

type metricsRouter struct {
	consumer.Metrics
	consumers map[component.ID]consumer.Metrics
}

func NewMetricsRouter(cm map[component.ID]consumer.Metrics) MetricsRouterAndConsumer {
	consumers := make([]consumer.Metrics, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &metricsRouter{
		Metrics:   fanoutconsumer.NewMetrics(consumers),
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
	return fanoutconsumer.NewMetrics(consumers), nil
}

func (r *metricsRouter) privateFunc() {}
