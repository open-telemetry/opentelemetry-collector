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
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/service/internal/components"
)

const exporterSeed = "exporter"

var _ consumerNode = &exporterNode{}

// An exporter instance can be shared by multiple pipelines of the same type.
// Therefore, nodeID is derived from "pipeline type" and "component ID".
type exporterNode struct {
	nodeID
	componentID  component.ID
	pipelineType component.DataType
	component.Component
}

func newExporterNode(pipelineID component.ID, exprID component.ID) *exporterNode {
	return &exporterNode{
		nodeID:       newNodeID(exporterSeed, string(pipelineID.Type()), exprID.String()),
		componentID:  exprID,
		pipelineType: pipelineID.Type(),
	}
}

func (n *exporterNode) getConsumer() baseConsumer {
	return n.Component.(baseConsumer)
}

func buildExporter(
	ctx context.Context,
	componentID component.ID,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *exporter.Builder,
	pipelineID component.ID,
) (exp component.Component, err error) {
	set := exporter.CreateSettings{ID: componentID, TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ExporterLogger(set.TelemetrySettings.Logger, componentID, pipelineID.Type())
	switch pipelineID.Type() {
	case component.DataTypeTraces:
		exp, err = builder.CreateTraces(ctx, set)
	case component.DataTypeMetrics:
		exp, err = builder.CreateMetrics(ctx, set)
	case component.DataTypeLogs:
		exp, err = builder.CreateLogs(ctx, set)
	default:
		return nil, fmt.Errorf("error creating exporter %q in pipeline %q, data type %q is not supported", set.ID, pipelineID, pipelineID.Type())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create %q exporter, in pipeline %q: %w", set.ID, pipelineID, err)
	}
	return exp, nil
}
