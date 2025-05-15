// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/service/internal/attribute"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/metadata"
	"go.opentelemetry.io/collector/service/internal/obsconsumer"
)

var _ consumerNode = (*processorNode)(nil)

// Every processor instance is unique to one pipeline.
// Therefore, nodeID is derived from "pipeline ID" and "component ID".
type processorNode struct {
	attribute.Attributes
	componentID component.ID
	pipelineID  pipeline.ID
	component.Component
	consumer baseConsumer
}

func newProcessorNode(pipelineID pipeline.ID, procID component.ID) *processorNode {
	return &processorNode{
		Attributes:  attribute.Processor(pipelineID, procID),
		componentID: procID,
		pipelineID:  pipelineID,
	}
}

func (n *processorNode) getConsumer() baseConsumer {
	return n.consumer
}

func (n *processorNode) buildComponent(ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *builders.ProcessorBuilder,
	next baseConsumer,
) error {
	set := processor.Settings{
		ID:                n.componentID,
		TelemetrySettings: telemetry.WithAttributeSet(tel, *n.Set()),
		BuildInfo:         info,
	}

	tb, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return err
	}

	switch n.pipelineID.Signal() {
	case pipeline.SignalTraces:
		n.Component, err = builder.CreateTraces(ctx, set,
			obsconsumer.NewTraces(next.(consumer.Traces), tb.ProcessorProducedItems))
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}
		n.consumer = obsconsumer.NewTraces(n.Component.(consumer.Traces), tb.ProcessorConsumedItems)
	case pipeline.SignalMetrics:
		n.Component, err = builder.CreateMetrics(ctx, set,
			obsconsumer.NewMetrics(next.(consumer.Metrics), tb.ProcessorProducedItems))
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}
		n.consumer = obsconsumer.NewMetrics(n.Component.(consumer.Metrics), tb.ProcessorConsumedItems)
	case pipeline.SignalLogs:
		n.Component, err = builder.CreateLogs(ctx, set,
			obsconsumer.NewLogs(next.(consumer.Logs), tb.ProcessorProducedItems))
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}
		n.consumer = obsconsumer.NewLogs(n.Component.(consumer.Logs), tb.ProcessorConsumedItems)
	case xpipeline.SignalProfiles:
		n.Component, err = builder.CreateProfiles(ctx, set,
			obsconsumer.NewProfiles(next.(xconsumer.Profiles), tb.ProcessorProducedItems))
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}
		n.consumer = obsconsumer.NewProfiles(n.Component.(xconsumer.Profiles), tb.ProcessorConsumedItems)
	default:
		return fmt.Errorf("error creating processor %q in pipeline %q, data type %q is not supported", set.ID, n.pipelineID.String(), n.pipelineID.Signal())
	}
	return nil
}
