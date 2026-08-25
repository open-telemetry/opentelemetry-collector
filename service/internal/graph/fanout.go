// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/internal/attribute"
)

var _ consumerNode = (*fanOutNode)(nil)

// Each pipeline has one fan-out node before exporters.
// Therefore, nodeID is derived from "pipeline ID".
type fanOutNode struct {
	attribute.Attributes
	pipelineID pipeline.ID
	baseConsumer
}

func newFanOutNode(pipelineID pipeline.ID) *fanOutNode {
	return &fanOutNode{
		Attributes: attribute.Fanout(pipelineID),
		pipelineID: pipelineID,
	}
}

func (n *fanOutNode) getConsumer() baseConsumer {
	return n.baseConsumer
}
