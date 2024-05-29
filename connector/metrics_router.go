// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumermetrics"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
)

// MetricsRouterAndConsumer feeds the first consumermetrics.Metrics in each of the specified pipelines.
type MetricsRouterAndConsumer interface {
	consumermetrics.Metrics
	Consumer(...component.ID) (consumermetrics.Metrics, error)
	PipelineIDs() []component.ID
	privateFunc()
}

type metricsRouter struct {
	consumermetrics.Metrics
	baseRouter[consumermetrics.Metrics]
}

func NewMetricsRouter(cm map[component.ID]consumermetrics.Metrics) MetricsRouterAndConsumer {
	consumers := make([]consumermetrics.Metrics, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &metricsRouter{
		Metrics:    fanoutconsumer.NewMetrics(consumers),
		baseRouter: newBaseRouter(fanoutconsumer.NewMetrics, cm),
	}
}

func (r *metricsRouter) privateFunc() {}
