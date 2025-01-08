// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/service/internal/attribute"
	"go.opentelemetry.io/collector/service/internal/builders"
)

var _ consumerNode = (*exporterNode)(nil)

// An exporter instance can be shared by multiple pipelines of the same type.
// Therefore, nodeID is derived from "pipeline type" and "component ID".
type exporterNode struct {
	*attribute.Attributes
	componentID   component.ID
	pipelineTypes map[pipeline.Signal]struct{}
	component.Component
}

func newExporterNode(pipelineType pipeline.Signal, exprID component.ID) *exporterNode {
	return &exporterNode{
		Attributes:    attribute.Exporter(pipelineType, exprID),
		componentID:   exprID,
		pipelineTypes: map[pipeline.Signal]struct{}{pipelineType: {}},
	}
}

func newSharedExporterNode(exprID component.ID) *exporterNode {
	return &exporterNode{
		Attributes:    attribute.SharedExporter(exprID),
		componentID:   exprID,
		pipelineTypes: map[pipeline.Signal]struct{}{},
	}
}

func (n *exporterNode) getConsumer() baseConsumer {
	return n.Component.(baseConsumer)
}

func (n *exporterNode) withSignalType(pipelineType pipeline.Signal) {
	n.pipelineTypes[pipelineType] = struct{}{}
}

func (n *exporterNode) buildComponent(
	ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *builders.ExporterBuilder,
) error {
	tel.Logger = n.Attributes.Logger(tel.Logger)
	set := exporter.Settings{ID: n.componentID, TelemetrySettings: tel, BuildInfo: info}
	var err error
	for signal := range n.pipelineTypes {
		switch signal {
		case pipeline.SignalTraces:
			n.Component, err = builder.CreateTraces(ctx, set)
		case pipeline.SignalMetrics:
			n.Component, err = builder.CreateMetrics(ctx, set)
		case pipeline.SignalLogs:
			n.Component, err = builder.CreateLogs(ctx, set)
		case xpipeline.SignalProfiles:
			n.Component, err = builder.CreateProfiles(ctx, set)
		default:
			return fmt.Errorf("error creating exporter %q for data type %q is not supported", set.ID, signal)
		}
		if err != nil {
			return fmt.Errorf("failed to create %q exporter for data type %q: %w", set.ID, signal, err)
		}
	}
	return nil
}
