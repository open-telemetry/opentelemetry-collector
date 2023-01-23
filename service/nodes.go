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

// nolint:unused
package service // import "go.opentelemetry.io/collector/service"

import (
	"hash/fnv"
	"strings"

	"gonum.org/v1/gonum/graph"

	"go.opentelemetry.io/collector/component"
)

type nodeID int64

func (n nodeID) ID() int64 {
	return int64(n)
}

func newNodeID(parts ...string) nodeID {
	h := fnv.New64a()
	h.Write([]byte(strings.Join(parts, "|")))
	return nodeID(h.Sum64())
}

type componentNode interface {
	component.Component
	graph.Node
}

var _ componentNode = &receiverNode{}

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
		nodeID:       newNodeID("receiver", string(pipelineID.Type()), recvID.String()),
		componentID:  recvID,
		pipelineType: pipelineID.Type(),
	}
}

var _ componentNode = &processorNode{}

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
		nodeID:      newNodeID("processor", pipelineID.String(), procID.String()),
		componentID: procID,
		pipelineID:  pipelineID,
	}
}

var _ componentNode = &exporterNode{}

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
		nodeID:       newNodeID("exporter", string(pipelineID.Type()), exprID.String()),
		componentID:  exprID,
		pipelineType: pipelineID.Type(),
	}
}

var _ componentNode = &connectorNode{}

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
		nodeID:           newNodeID("connector", connID.String(), string(exprPipelineType), string(rcvrPipelineType)),
		componentID:      connID,
		exprPipelineType: exprPipelineType,
		rcvrPipelineType: rcvrPipelineType,
	}
}
