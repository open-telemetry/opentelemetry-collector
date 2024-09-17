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

// TracesRouterAndConsumer feeds the first consumer.Traces in each of the specified pipelines.
//
// Deprecated: [v0.110.0] Use TracesRouterAndConsumerWithPipelineIDs instead.
type TracesRouterAndConsumer interface {
	consumer.Traces
	Consumer(...component.ID) (consumer.Traces, error)
	PipelineIDs() []component.ID
	privateFunc()
}

// TracesRouterAndConsumerWithPipelineIDs feeds the first consumer.Traces in each of the specified pipelines.
type TracesRouterAndConsumerWithPipelineIDs interface {
	consumer.Traces
	Consumer(...pipeline.ID) (consumer.Traces, error)
	PipelineIDs() []pipeline.ID
	privateFunc()
}

type tracesRouter struct {
	consumer.Traces
	internal.BaseRouter[consumer.Traces]
}

// Deprecated: [v0.110.0] Use NewTracesRouterWithPipelineIDs instead
func NewTracesRouter(cm map[component.ID]consumer.Traces) TracesRouterAndConsumer {
	consumers := make([]consumer.Traces, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &tracesRouter{
		Traces:     fanoutconsumer.NewTraces(consumers),
		BaseRouter: internal.NewBaseRouter(fanoutconsumer.NewTraces, cm),
	}
}

func (r *tracesRouter) privateFunc() {}

type tracesRouterPipelineIDs struct {
	consumer.Traces
	internal.BaseRouterWithPipelineIDs[consumer.Traces]
}

func NewTracesRouterWithPipelineIDs(cm map[pipeline.ID]consumer.Traces) TracesRouterAndConsumerWithPipelineIDs {
	consumers := make([]consumer.Traces, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &tracesRouterPipelineIDs{
		Traces:                    fanoutconsumer.NewTraces(consumers),
		BaseRouterWithPipelineIDs: internal.NewBaseRouterWithPipelineIDs(fanoutconsumer.NewTraces, cm),
	}
}

func (r *tracesRouterPipelineIDs) privateFunc() {}
