// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/service/internal/components"
	"go.opentelemetry.io/collector/service/internal/fanoutconsumer"
)

const connectorSeed = "connector"

var _ consumerNode = &connectorNode{}

// A connector instance connects one pipeline type to one other pipeline type.
// Therefore, nodeID is derived from "exporter pipeline type", "receiver pipeline type", and "component ID".
type connectorNode struct {
	nodeID
	componentID      component.ID
	exprPipelineType component.DataType
	rcvrPipelineType component.DataType
	component.Component
}

func newConnectorNode(exprPipelineType, rcvrPipelineType component.DataType, connID component.ID) *connectorNode {
	return &connectorNode{
		nodeID:           newNodeID(connectorSeed, connID.String(), string(exprPipelineType), string(rcvrPipelineType)),
		componentID:      connID,
		exprPipelineType: exprPipelineType,
		rcvrPipelineType: rcvrPipelineType,
	}
}

func (n *connectorNode) getConsumer() baseConsumer {
	return n.Component.(baseConsumer)
}

func buildConnector(
	ctx context.Context,
	componentID component.ID,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *connector.Builder,
	exprPipelineType component.Type,
	rcvrPipelineType component.Type,
	nexts []baseConsumer,
) (conn component.Component, err error) {
	set := connector.CreateSettings{ID: componentID, TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ConnectorLogger(set.TelemetrySettings.Logger, componentID, exprPipelineType, rcvrPipelineType)

	switch rcvrPipelineType {
	case component.DataTypeTraces:
		next := nexts[0].(consumer.Traces)
		if len(nexts) > 1 {
			consumers := make(map[component.ID]consumer.Traces, len(nexts))
			for _, next := range nexts {
				consumers[next.(*capabilitiesNode).pipelineID] = next.(consumer.Traces)
			}
			next = fanoutconsumer.NewTracesRouter(consumers)
		}
		switch exprPipelineType {
		case component.DataTypeTraces:
			conn, err = builder.CreateTracesToTraces(ctx, set, next)
		case component.DataTypeMetrics:
			conn, err = builder.CreateMetricsToTraces(ctx, set, next)
		case component.DataTypeLogs:
			conn, err = builder.CreateLogsToTraces(ctx, set, next)
		}
	case component.DataTypeMetrics:
		next := nexts[0].(consumer.Metrics)
		if len(nexts) > 1 {
			consumers := make(map[component.ID]consumer.Metrics, len(nexts))
			for _, next := range nexts {
				consumers[next.(*capabilitiesNode).pipelineID] = next.(consumer.Metrics)
			}
			next = fanoutconsumer.NewMetricsRouter(consumers)
		}
		switch exprPipelineType {
		case component.DataTypeTraces:
			conn, err = builder.CreateTracesToMetrics(ctx, set, next)
		case component.DataTypeMetrics:
			conn, err = builder.CreateMetricsToMetrics(ctx, set, next)
		case component.DataTypeLogs:
			conn, err = builder.CreateLogsToMetrics(ctx, set, next)
		}
	case component.DataTypeLogs:
		next := nexts[0].(consumer.Logs)
		if len(nexts) > 1 {
			consumers := make(map[component.ID]consumer.Logs, len(nexts))
			for _, next := range nexts {
				consumers[next.(*capabilitiesNode).pipelineID] = next.(consumer.Logs)
			}
			next = fanoutconsumer.NewLogsRouter(consumers)
		}
		switch exprPipelineType {
		case component.DataTypeTraces:
			conn, err = builder.CreateTracesToLogs(ctx, set, next)
		case component.DataTypeMetrics:
			conn, err = builder.CreateMetricsToLogs(ctx, set, next)
		case component.DataTypeLogs:
			conn, err = builder.CreateLogsToLogs(ctx, set, next)
		}
	}
	return
}
