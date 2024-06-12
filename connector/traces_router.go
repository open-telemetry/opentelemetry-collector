// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
	"go.opentelemetry.io/collector/pipeline"
)

// TracesRouterAndConsumer feeds the first consumer.Traces in each of the specified pipelines.
type TracesRouterAndConsumer interface {
	consumer.Traces
	Consumer(...pipeline.PipelineID) (consumer.Traces, error)
	PipelineIDs() []pipeline.PipelineID
	privateFunc()
}

type tracesRouter struct {
	consumer.Traces
	baseRouter[consumer.Traces]
}

func NewTracesRouter(cm map[pipeline.PipelineID]consumer.Traces) TracesRouterAndConsumer {
	consumers := make([]consumer.Traces, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &tracesRouter{
		Traces:     fanoutconsumer.NewTraces(consumers),
		baseRouter: newBaseRouter(fanoutconsumer.NewTraces, cm),
	}
}

func (r *tracesRouter) privateFunc() {}
