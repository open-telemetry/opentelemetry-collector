// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/ctrace"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
)

// TracesRouterAndConsumer feeds the first ctrace.Traces in each of the specified pipelines.
type TracesRouterAndConsumer interface {
	ctrace.Traces
	Consumer(...component.ID) (ctrace.Traces, error)
	PipelineIDs() []component.ID
	privateFunc()
}

type tracesRouter struct {
	ctrace.Traces
	baseRouter[ctrace.Traces]
}

func NewTracesRouter(cm map[component.ID]ctrace.Traces) TracesRouterAndConsumer {
	consumers := make([]ctrace.Traces, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &tracesRouter{
		Traces:     fanoutconsumer.NewTraces(consumers),
		baseRouter: newBaseRouter(fanoutconsumer.NewTraces, cm),
	}
}

func (r *tracesRouter) privateFunc() {}
