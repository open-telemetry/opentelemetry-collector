// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/internal/attribute"
)

var _ consumerNode = (*capabilitiesNode)(nil)

// Every pipeline has a "virtual" capabilities node immediately after the receiver(s).
// There are two purposes for this node:
// 1. Present aggregated capabilities to receivers, such as whether the pipeline mutates data.
// 2. Present a consistent "first consumer" for each pipeline.
// The nodeID is derived from "pipeline ID".
type capabilitiesNode struct {
	attribute.Attributes
	pipelineID pipeline.ID
	baseConsumer
	consumer.ConsumeTracesFunc
	consumer.ConsumeMetricsFunc
	consumer.ConsumeLogsFunc
	xconsumer.ConsumeProfilesFunc
}

func newCapabilitiesNode(pipelineID pipeline.ID) *capabilitiesNode {
	return &capabilitiesNode{
		Attributes: attribute.Capabilities(pipelineID),
		pipelineID: pipelineID,
	}
}

func (n *capabilitiesNode) getConsumer() baseConsumer {
	return n
}
