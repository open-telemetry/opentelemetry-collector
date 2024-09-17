// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector/internal"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
	"go.opentelemetry.io/collector/pipeline"
)

// MetricsRouterAndConsumer feeds the first consumer.Metrics in each of the specified pipelines.
//
// Deprecated: [v0.110.0] Use MetricsRouterAndConsumerWithPipelineIDs instead
type MetricsRouterAndConsumer interface {
	consumer.Metrics
	Consumer(...component.ID) (consumer.Metrics, error)
	PipelineIDs() []component.ID
	privateFunc()
}

// MetricsRouterAndConsumerWithPipelineIDs feeds the first consumer.Metrics in each of the specified pipelines.
type MetricsRouterAndConsumerWithPipelineIDs interface {
	consumer.Metrics
	Consumer(...pipeline.ID) (consumer.Metrics, error)
	PipelineIDs() []pipeline.ID
	privateFunc()
}

type metricsRouter struct {
	consumer.Metrics
	internal.BaseRouter[consumer.Metrics]
}

// Deprecated: [v0.110.0] Use NewMetricsRouterWithPipelineIDs instead
func NewMetricsRouter(cm map[component.ID]consumer.Metrics) MetricsRouterAndConsumer {
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

type metricsRouterPipelineIDs struct {
	consumer.Metrics
	internal.BaseRouterWithPipelineIDs[consumer.Metrics]
}

func NewMetricsRouterWithPipelineIDs(cm map[pipeline.ID]consumer.Metrics) MetricsRouterAndConsumerWithPipelineIDs {
	consumers := make([]consumer.Metrics, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &metricsRouterPipelineIDs{
		Metrics:                   fanoutconsumer.NewMetrics(consumers),
		BaseRouterWithPipelineIDs: internal.NewBaseRouterWithPipelineIDs(fanoutconsumer.NewMetrics, cm),
	}
}

func (r *metricsRouterPipelineIDs) privateFunc() {}
