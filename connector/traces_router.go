// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertraces"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
)

// TracesRouterAndConsumer feeds the first consumertraces.Traces in each of the specified pipelines.
type TracesRouterAndConsumer interface {
	consumertraces.Traces
	Consumer(...component.ID) (consumertraces.Traces, error)
	PipelineIDs() []component.ID
	privateFunc()
}

type tracesRouter struct {
	consumertraces.Traces
	baseRouter[consumertraces.Traces]
}

func NewTracesRouter(cm map[component.ID]consumertraces.Traces) TracesRouterAndConsumer {
	consumers := make([]consumertraces.Traces, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &tracesRouter{
		Traces:     fanoutconsumer.NewTraces(consumers),
		baseRouter: newBaseRouter(fanoutconsumer.NewTraces, cm),
	}
}

func (r *tracesRouter) privateFunc() {}
