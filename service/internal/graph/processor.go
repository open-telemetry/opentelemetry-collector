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
	set := processor.Settings{ID: n.componentID, TelemetrySettings: tel, BuildInfo: info}
	if telemetry.NewPipelineTelemetryGate.IsEnabled() {
		set.TelemetrySettings = telemetry.WithAttributeSet(set.TelemetrySettings, *n.Set())
	}
	tb, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return err
	}

	switch n.pipelineID.Signal() {
	case pipeline.SignalTraces:
		obsConsumer := obsConsumerTraces{
			Traces:      next.(consumer.Traces),
			itemCounter: tb.ProcessorProducedItems,
		}
		n.Component, err = builder.CreateTraces(ctx, set, obsConsumer)
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}
		n.consumer = obsConsumerTraces{
			Traces:      n.Component.(consumer.Traces),
			itemCounter: tb.ProcessorConsumedItems,
		}
	case pipeline.SignalMetrics:
		obsConsumer := obsConsumerMetrics{
			Metrics:     next.(consumer.Metrics),
			itemCounter: tb.ProcessorProducedItems,
		}
		n.Component, err = builder.CreateMetrics(ctx, set, obsConsumer)
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}
		n.consumer = obsConsumerMetrics{
			Metrics:     n.Component.(consumer.Metrics),
			itemCounter: tb.ProcessorConsumedItems,
		}
	case pipeline.SignalLogs:
		obsConsumer := obsConsumerLogs{
			Logs:        next.(consumer.Logs),
			itemCounter: tb.ProcessorProducedItems,
		}
		n.Component, err = builder.CreateLogs(ctx, set, obsConsumer)
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}
		n.consumer = obsConsumerLogs{
			Logs:        n.Component.(consumer.Logs),
			itemCounter: tb.ProcessorConsumedItems,
		}
	case xpipeline.SignalProfiles:
		obsConsumer := obsConsumerProfiles{
			Profiles:    next.(xconsumer.Profiles),
			itemCounter: tb.ProcessorProducedItems,
		}
		n.Component, err = builder.CreateProfiles(ctx, set, obsConsumer)
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}
		n.consumer = obsConsumerProfiles{
			Profiles:    n.Component.(xconsumer.Profiles),
			itemCounter: tb.ProcessorConsumedItems,
		}
	default:
		return fmt.Errorf("error creating processor %q in pipeline %q, data type %q is not supported", set.ID, n.pipelineID.String(), n.pipelineID.Signal())
	}
	return nil
}
