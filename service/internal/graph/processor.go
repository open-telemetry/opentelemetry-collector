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
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/service/internal/components"
)

const processorSeed = "processor"

var _ consumerNode = &processorNode{}

// Every processor instance is unique to one pipeline.
// Therefore, nodeID is derived from "pipeline ID" and "component ID".
type processorNode struct {
	nodeID
	componentID component.ID
	pipelineID  component.ID
	component.Component
}

func newProcessorNode(pipelineID, procID component.ID) *processorNode {
	return &processorNode{
		nodeID:      newNodeID(processorSeed, pipelineID.String(), procID.String()),
		componentID: procID,
		pipelineID:  pipelineID,
	}
}

func (n *processorNode) getConsumer() baseConsumer {
	return n.Component.(baseConsumer)
}

func buildProcessor(ctx context.Context,
	componentID component.ID,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *processor.Builder,
	pipelineID component.ID,
	next baseConsumer,
) (proc component.Component, err error) {
	set := processor.CreateSettings{ID: componentID, TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ProcessorLogger(set.TelemetrySettings.Logger, componentID, pipelineID)
	switch pipelineID.Type() {
	case component.DataTypeTraces:
		proc, err = builder.CreateTraces(ctx, set, next.(consumer.Traces))
	case component.DataTypeMetrics:
		proc, err = builder.CreateMetrics(ctx, set, next.(consumer.Metrics))
	case component.DataTypeLogs:
		proc, err = builder.CreateLogs(ctx, set, next.(consumer.Logs))
	default:
		return nil, fmt.Errorf("error creating processor %q in pipeline %q, data type %q is not supported", set.ID, pipelineID, pipelineID.Type())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, pipelineID, err)
	}
	return proc, nil
}
