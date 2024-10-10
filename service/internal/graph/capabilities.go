// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/pipeline"
)

const capabilitiesSeed = "capabilities"

var _ consumerNode = (*capabilitiesNode)(nil)

// Every pipeline has a "virtual" capabilities node immediately after the receiver(s).
// There are two purposes for this node:
// 1. Present aggregated capabilities to receivers, such as whether the pipeline mutates data.
// 2. Present a consistent "first consumer" for each pipeline.
// The nodeID is derived from "pipeline ID".
type capabilitiesNode struct {
	nodeID
	pipelineID pipeline.ID
	baseConsumer
	consumer.ConsumeTracesFunc
	consumer.ConsumeMetricsFunc
	consumer.ConsumeLogsFunc
	consumerprofiles.ConsumeProfilesFunc
}

func newCapabilitiesNode(pipelineID pipeline.ID) *capabilitiesNode {
	return &capabilitiesNode{
		nodeID:     newNodeID(capabilitiesSeed, pipelineID.String()),
		pipelineID: pipelineID,
	}
}

func (n *capabilitiesNode) getConsumer() baseConsumer {
	return n
}
