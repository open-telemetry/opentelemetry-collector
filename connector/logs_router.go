// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"go.opentelemetry.io/collector/connector/internal"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
	"go.opentelemetry.io/collector/pipeline"
)

// LogsRouterAndConsumer feeds the first consumer.Logs in each of the specified pipelines.
type LogsRouterAndConsumer interface {
	consumer.Logs
	Consumer(...pipeline.ID) (consumer.Logs, error)
	PipelineIDs() []pipeline.ID
	privateFunc()
}

type logsRouter struct {
	consumer.Logs
	internal.BaseRouter[consumer.Logs]
}

func NewLogsRouter(cm map[pipeline.ID]consumer.Logs) LogsRouterAndConsumer {
	consumers := make([]consumer.Logs, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &logsRouter{
		Logs:       fanoutconsumer.NewLogs(consumers),
		BaseRouter: internal.NewBaseRouter(fanoutconsumer.NewLogs, cm),
	}
}

func (r *logsRouter) privateFunc() {}
