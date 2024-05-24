// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/cmetric"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
)

// MetricsRouterAndConsumer feeds the first cmetric.Metrics in each of the specified pipelines.
type MetricsRouterAndConsumer interface {
	cmetric.Metrics
	Consumer(...component.ID) (cmetric.Metrics, error)
	PipelineIDs() []component.ID
	privateFunc()
}

type metricsRouter struct {
	cmetric.Metrics
	baseRouter[cmetric.Metrics]
}

func NewMetricsRouter(cm map[component.ID]cmetric.Metrics) MetricsRouterAndConsumer {
	consumers := make([]cmetric.Metrics, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &metricsRouter{
		Metrics:    fanoutconsumer.NewMetrics(consumers),
		baseRouter: newBaseRouter(fanoutconsumer.NewMetrics, cm),
	}
}

func (r *metricsRouter) privateFunc() {}
