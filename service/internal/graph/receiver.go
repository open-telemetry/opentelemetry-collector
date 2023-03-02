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
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/internal/components"
	"go.opentelemetry.io/collector/service/internal/fanoutconsumer"
)

const receiverSeed = "receiver"

// A receiver instance can be shared by multiple pipelines of the same type.
// Therefore, nodeID is derived from "pipeline type" and "component ID".
type receiverNode struct {
	nodeID
	componentID  component.ID
	pipelineType component.DataType
	component.Component
}

func newReceiverNode(pipelineID component.ID, recvID component.ID) *receiverNode {
	return &receiverNode{
		nodeID:       newNodeID(receiverSeed, string(pipelineID.Type()), recvID.String()),
		componentID:  recvID,
		pipelineType: pipelineID.Type(),
	}
}

func buildReceiver(ctx context.Context,
	componentID component.ID,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *receiver.Builder,
	pipelineID component.ID,
	nexts []baseConsumer,
) (recv component.Component, err error) {
	set := receiver.CreateSettings{ID: componentID, TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ReceiverLogger(tel.Logger, componentID, pipelineID.Type())
	switch pipelineID.Type() {
	case component.DataTypeTraces:
		var consumers []consumer.Traces
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Traces))
		}
		recv, err = builder.CreateTraces(ctx, set, fanoutconsumer.NewTraces(consumers))
	case component.DataTypeMetrics:
		var consumers []consumer.Metrics
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Metrics))
		}
		recv, err = builder.CreateMetrics(ctx, set, fanoutconsumer.NewMetrics(consumers))
	case component.DataTypeLogs:
		var consumers []consumer.Logs
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Logs))
		}
		recv, err = builder.CreateLogs(ctx, set, fanoutconsumer.NewLogs(consumers))
	default:
		return nil, fmt.Errorf("error creating receiver %q in pipeline %q, data type %q is not supported", set.ID, pipelineID, pipelineID.Type())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create %q receiver, in pipeline %q: %w", set.ID, pipelineID, err)
	}
	return recv, nil
}
