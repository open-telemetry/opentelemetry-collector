// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentprofiles"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/components"
)

const exporterSeed = "exporter"

var _ consumerNode = (*exporterNode)(nil)

// An exporter instance can be shared by multiple pipelines of the same type.
// Therefore, nodeID is derived from "pipeline type" and "component ID".
type exporterNode struct {
	nodeID
	componentID  component.ID
	pipelineType pipeline.Signal
	component.Component
}

func newExporterNode(pipelineType pipeline.Signal, exprID component.ID) *exporterNode {
	return &exporterNode{
		nodeID:       newNodeID(exporterSeed, pipelineType.String(), exprID.String()),
		componentID:  exprID,
		pipelineType: pipelineType,
	}
}

func (n *exporterNode) getConsumer() baseConsumer {
	return n.Component.(baseConsumer)
}

func (n *exporterNode) buildComponent(
	ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *builders.ExporterBuilder,
) error {
	tel.Logger = components.ExporterLogger(tel.Logger, n.componentID, n.pipelineType)
	set := exporter.Settings{ID: n.componentID, TelemetrySettings: tel, BuildInfo: info}
	var err error
	switch n.pipelineType {
	case pipeline.SignalTraces:
		n.Component, err = builder.CreateTraces(ctx, set)
	case pipeline.SignalMetrics:
		n.Component, err = builder.CreateMetrics(ctx, set)
	case pipeline.SignalLogs:
		n.Component, err = builder.CreateLogs(ctx, set)
	case componentprofiles.SignalProfiles:
		n.Component, err = builder.CreateProfiles(ctx, set)
	default:
		return fmt.Errorf("error creating exporter %q for data type %q is not supported", set.ID, n.pipelineType)
	}
	if err != nil {
		return fmt.Errorf("failed to create %q exporter for data type %q: %w", set.ID, n.pipelineType, err)
	}
	return nil
}
