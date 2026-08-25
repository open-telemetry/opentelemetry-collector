// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/service/internal/attribute"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/componentattribute"
	"go.opentelemetry.io/collector/service/internal/metadata"
	"go.opentelemetry.io/collector/service/internal/obsconsumer"
	"go.opentelemetry.io/collector/service/internal/refconsumer"
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
		TelemetrySettings: componentattribute.TelemetrySettingsWithAttributes(tel, *n.Set()),
		BuildInfo:         info,
	}

	tb, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return err
	}

	producedSettings := obsconsumer.Settings{
		ItemCounter: tb.ProcessorProducedItems,
		SizeCounter: tb.ProcessorProducedSize,
		Logger:      set.Logger,
	}
	consumedSettings := obsconsumer.Settings{
		ItemCounter: tb.ProcessorConsumedItems,
		SizeCounter: tb.ProcessorConsumedSize,
		Logger:      set.Logger,
	}

	switch n.pipelineID.Signal() {
	case pipeline.SignalTraces:
		n.Component, err = builder.CreateTraces(ctx, set,
			obsconsumer.NewTraces(next.(consumer.Traces), producedSettings),
		)
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}
		n.consumer = obsconsumer.NewTraces(n.Component.(consumer.Traces), consumedSettings)
		n.consumer = refconsumer.NewTraces(n.consumer.(consumer.Traces))
	case pipeline.SignalMetrics:
		n.Component, err = builder.CreateMetrics(ctx, set,
			obsconsumer.NewMetrics(next.(consumer.Metrics), producedSettings))
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}
		n.consumer = obsconsumer.NewMetrics(n.Component.(consumer.Metrics), consumedSettings)
		n.consumer = refconsumer.NewMetrics(n.consumer.(consumer.Metrics))
	case pipeline.SignalLogs:
		n.Component, err = builder.CreateLogs(ctx, set,
			obsconsumer.NewLogs(next.(consumer.Logs), producedSettings))
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}
		n.consumer = obsconsumer.NewLogs(n.Component.(consumer.Logs), consumedSettings)
		n.consumer = refconsumer.NewLogs(n.consumer.(consumer.Logs))
	case xpipeline.SignalProfiles:
		n.Component, err = builder.CreateProfiles(ctx, set,
			obsconsumer.NewProfiles(next.(xconsumer.Profiles), producedSettings))
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}
		n.consumer = obsconsumer.NewProfiles(n.Component.(xconsumer.Profiles), consumedSettings)
		n.consumer = refconsumer.NewProfiles(n.consumer.(xconsumer.Profiles))
	default:
		return fmt.Errorf("error creating processor %q in pipeline %q, data type %q is not supported", set.ID, n.pipelineID.String(), n.pipelineID.Signal())
	}
	return nil
}
