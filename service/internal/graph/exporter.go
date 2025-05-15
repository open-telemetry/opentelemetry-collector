// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/service/internal/attribute"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/metadata"
	"go.opentelemetry.io/collector/service/internal/obsconsumer"
)

var _ consumerNode = (*exporterNode)(nil)

// An exporter instance can be shared by multiple pipelines of the same type.
// Therefore, nodeID is derived from "pipeline type" and "component ID".
type exporterNode struct {
	attribute.Attributes
	componentID  component.ID
	pipelineType pipeline.Signal
	component.Component
	consumer baseConsumer
}

func newExporterNode(pipelineType pipeline.Signal, exprID component.ID) *exporterNode {
	return &exporterNode{
		Attributes:   attribute.Exporter(pipelineType, exprID),
		componentID:  exprID,
		pipelineType: pipelineType,
	}
}

func (n *exporterNode) getConsumer() baseConsumer {
	return n.consumer
}

func (n *exporterNode) buildComponent(
	ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *builders.ExporterBuilder,
) error {
	set := exporter.Settings{
		ID:                n.componentID,
		TelemetrySettings: telemetry.WithAttributeSet(tel, *n.Set()),
		BuildInfo:         info,
	}

	tb, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return err
	}

	switch n.pipelineType {
	case pipeline.SignalTraces:
		n.Component, err = builder.CreateTraces(ctx, set)
		if err != nil {
			return fmt.Errorf("failed to create %q exporter for data type %q: %w", set.ID, n.pipelineType, err)
		}
		n.consumer = obsconsumer.NewTraces(n.Component.(consumer.Traces), tb.ExporterConsumedItems)
	case pipeline.SignalMetrics:
		n.Component, err = builder.CreateMetrics(ctx, set)
		if err != nil {
			return fmt.Errorf("failed to create %q exporter for data type %q: %w", set.ID, n.pipelineType, err)
		}
		n.consumer = obsconsumer.NewMetrics(n.Component.(consumer.Metrics), tb.ExporterConsumedItems)
	case pipeline.SignalLogs:
		n.Component, err = builder.CreateLogs(ctx, set)
		if err != nil {
			return fmt.Errorf("failed to create %q exporter for data type %q: %w", set.ID, n.pipelineType, err)
		}
		n.consumer = obsconsumer.NewLogs(n.Component.(consumer.Logs), tb.ExporterConsumedItems)
	case xpipeline.SignalProfiles:
		n.Component, err = builder.CreateProfiles(ctx, set)
		if err != nil {
			return fmt.Errorf("failed to create %q exporter for data type %q: %w", set.ID, n.pipelineType, err)
		}
		n.consumer = obsconsumer.NewProfiles(n.Component.(xconsumer.Profiles), tb.ExporterConsumedItems)
	default:
		return fmt.Errorf("error creating exporter %q for data type %q is not supported", set.ID, n.pipelineType)
	}
	return nil
}
