// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"go.opentelemetry.io/collector/connector/internal"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
	"go.opentelemetry.io/collector/pipeline"
)

// MetricsRouterAndConsumer feeds the first consumer.Metrics in each of the specified pipelines.
type MetricsRouterAndConsumer interface {
	consumer.Metrics
	Consumer(...pipeline.ID) (consumer.Metrics, error)
	PipelineIDs() []pipeline.ID
	privateFunc()
}

type metricsRouter struct {
	consumer.Metrics
	internal.BaseRouter[consumer.Metrics]
}

func NewMetricsRouter(cm map[pipeline.ID]consumer.Metrics) MetricsRouterAndConsumer {
	consumers := make([]consumer.Metrics, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &metricsRouter{
		Metrics:    fanoutconsumer.NewMetrics(consumers),
		BaseRouter: internal.NewBaseRouter(fanoutconsumer.NewMetrics, cm),
	}
}

func (r *metricsRouter) privateFunc() {}
