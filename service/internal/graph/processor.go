// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentprofiles"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/components"
)

const processorSeed = "processor"

var _ consumerNode = (*processorNode)(nil)

// Every processor instance is unique to one pipeline.
// Therefore, nodeID is derived from "pipeline ID" and "component ID".
type processorNode struct {
	nodeID
	componentID component.ID
	pipelineID  pipeline.ID
	component.Component
}

func newProcessorNode(pipelineID pipeline.ID, procID component.ID) *processorNode {
	return &processorNode{
		nodeID:      newNodeID(processorSeed, pipelineID.String(), procID.String()),
		componentID: procID,
		pipelineID:  pipelineID,
	}
}

func (n *processorNode) getConsumer() baseConsumer {
	return n.Component.(baseConsumer)
}

func (n *processorNode) buildComponent(ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *builders.ProcessorBuilder,
	next baseConsumer,
) error {
	tel.Logger = components.ProcessorLogger(tel.Logger, n.componentID, n.pipelineID)
	set := processor.Settings{ID: n.componentID, TelemetrySettings: tel, BuildInfo: info}
	var err error
	switch n.pipelineID.Signal() {
	case pipeline.SignalTraces:
		n.Component, err = builder.CreateTraces(ctx, set, next.(consumer.Traces))
	case pipeline.SignalMetrics:
		n.Component, err = builder.CreateMetrics(ctx, set, next.(consumer.Metrics))
	case pipeline.SignalLogs:
		n.Component, err = builder.CreateLogs(ctx, set, next.(consumer.Logs))
	case componentprofiles.SignalProfiles:
		n.Component, err = builder.CreateProfiles(ctx, set, next.(consumerprofiles.Profiles))
	default:
		return fmt.Errorf("error creating processor %q in pipeline %q, data type %q is not supported", set.ID, n.pipelineID.String(), n.pipelineID.Signal())
	}
	if err != nil {
		return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
	}
	return nil
}
