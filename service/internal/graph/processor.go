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
		producedOpts := []obsconsumer.Option{
			obsconsumer.WithTracesItemCounter(&tb.ProcessorProducedItems),
		}
		if isEnabled(tb.ProcessorProducedSize) {
			producedOpts = append(producedOpts, obsconsumer.WithTracesSizeCounter(&tb.ProcessorProducedSize))
		}
		n.Component, err = builder.CreateTraces(ctx, set,
			obsconsumer.NewTraces(next.(consumer.Traces), producedOpts...))
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}

		consumedOpts := []obsconsumer.Option{
			obsconsumer.WithTracesItemCounter(&tb.ProcessorConsumedItems),
		}
		if isEnabled(tb.ProcessorConsumedSize) {
			consumedOpts = append(consumedOpts, obsconsumer.WithTracesSizeCounter(&tb.ProcessorConsumedSize))
		}
		n.consumer = obsconsumer.NewTraces(n.Component.(consumer.Traces), consumedOpts...)
	case pipeline.SignalMetrics:
		producedOpts := []obsconsumer.Option{
			obsconsumer.WithMetricsItemCounter(&tb.ProcessorProducedItems),
		}
		if isEnabled(tb.ProcessorProducedSize) {
			producedOpts = append(producedOpts, obsconsumer.WithMetricsSizeCounter(&tb.ProcessorProducedSize))
		}
		n.Component, err = builder.CreateMetrics(ctx, set,
			obsconsumer.NewMetrics(next.(consumer.Metrics), producedOpts...))
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}

		consumedOpts := []obsconsumer.Option{
			obsconsumer.WithMetricsItemCounter(&tb.ProcessorConsumedItems),
		}
		if isEnabled(tb.ProcessorConsumedSize) {
			consumedOpts = append(consumedOpts, obsconsumer.WithMetricsSizeCounter(&tb.ProcessorConsumedSize))
		}
		n.consumer = obsconsumer.NewMetrics(n.Component.(consumer.Metrics), consumedOpts...)
	case pipeline.SignalLogs:
		producedOpts := []obsconsumer.Option{
			obsconsumer.WithLogsSizeCounter(&tb.ProcessorProducedItems),
		}
		if isEnabled(tb.ProcessorProducedSize) {
			producedOpts = append(producedOpts, obsconsumer.WithLogsSizeCounter(&tb.ProcessorProducedSize))
		}
		n.Component, err = builder.CreateLogs(ctx, set,
			obsconsumer.NewLogs(next.(consumer.Logs), producedOpts...))
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}

		consumedOpts := []obsconsumer.Option{
			obsconsumer.WithLogsSizeCounter(&tb.ProcessorConsumedItems),
		}
		if isEnabled(tb.ProcessorConsumedSize) {
			consumedOpts = append(consumedOpts, obsconsumer.WithLogsSizeCounter(&tb.ProcessorConsumedSize))
		}
		n.consumer = obsconsumer.NewLogs(n.Component.(consumer.Logs), consumedOpts...)
	case xpipeline.SignalProfiles:
		producedOpts := []obsconsumer.Option{
			obsconsumer.WithProfilesItemCounter(&tb.ProcessorProducedItems),
		}
		if isEnabled(tb.ProcessorProducedSize) {
			producedOpts = append(producedOpts, obsconsumer.WithProfilesSizeCounter(&tb.ProcessorProducedSize))
		}
		n.Component, err = builder.CreateProfiles(ctx, set,
			obsconsumer.NewProfiles(next.(xconsumer.Profiles), producedOpts...))
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}

		consumedOpts := []obsconsumer.Option{
			obsconsumer.WithProfilesItemCounter(&tb.ProcessorConsumedItems),
		}
		if isEnabled(tb.ProcessorConsumedSize) {
			consumedOpts = append(consumedOpts, obsconsumer.WithProfilesSizeCounter(&tb.ProcessorConsumedSize))
		}
		n.consumer = obsconsumer.NewProfiles(n.Component.(xconsumer.Profiles), consumedOpts...)
	default:
		return fmt.Errorf("error creating processor %q in pipeline %q, data type %q is not supported", set.ID, n.pipelineID.String(), n.pipelineID.Signal())
	}
	return nil
}
