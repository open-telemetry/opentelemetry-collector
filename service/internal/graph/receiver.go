// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/internal/attribute"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/metadata"
	"go.opentelemetry.io/collector/service/internal/obsconsumer"
)

// A receiver instance can be shared by multiple pipelines of the same type.
// Therefore, nodeID is derived from "pipeline type" and "component ID".
type receiverNode struct {
	attribute.Attributes
	componentID  component.ID
	pipelineType pipeline.Signal
	component.Component
}

func newReceiverNode(pipelineType pipeline.Signal, recvID component.ID) *receiverNode {
	return &receiverNode{
		Attributes:   attribute.Receiver(pipelineType, recvID),
		componentID:  recvID,
		pipelineType: pipelineType,
	}
}

func (n *receiverNode) buildComponent(ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *builders.ReceiverBuilder,
	nexts []baseConsumer,
) error {
	set := receiver.Settings{
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
		var consumers []consumer.Traces
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Traces))
		}

		consumedOpts := []obsconsumer.Option{
			obsconsumer.WithTracesItemCounter(&tb.ReceiverProducedItems),
		}
		if isEnabled(tb.ReceiverProducedSize) {
			consumedOpts = append(consumedOpts, obsconsumer.WithTracesSizeCounter(&tb.ReceiverProducedSize))
		}
		n.Component, err = builder.CreateTraces(ctx, set,
			obsconsumer.NewTraces(fanoutconsumer.NewTraces(consumers), consumedOpts...))
	case pipeline.SignalMetrics:
		var consumers []consumer.Metrics
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Metrics))
		}

		consumedOpts := []obsconsumer.Option{
			obsconsumer.WithMetricsItemCounter(&tb.ReceiverProducedItems),
		}
		if isEnabled(tb.ReceiverProducedSize) {
			consumedOpts = append(consumedOpts, obsconsumer.WithMetricsSizeCounter(&tb.ReceiverProducedSize))
		}
		n.Component, err = builder.CreateMetrics(ctx, set,
			obsconsumer.NewMetrics(fanoutconsumer.NewMetrics(consumers), consumedOpts...))
	case pipeline.SignalLogs:
		var consumers []consumer.Logs
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Logs))
		}

		consumedOpts := []obsconsumer.Option{
			obsconsumer.WithLogsSizeCounter(&tb.ReceiverProducedItems),
		}
		if isEnabled(tb.ReceiverProducedSize) {
			consumedOpts = append(consumedOpts, obsconsumer.WithLogsSizeCounter(&tb.ReceiverProducedSize))
		}
		n.Component, err = builder.CreateLogs(ctx, set,
			obsconsumer.NewLogs(fanoutconsumer.NewLogs(consumers), consumedOpts...))
	case xpipeline.SignalProfiles:
		var consumers []xconsumer.Profiles
		for _, next := range nexts {
			consumers = append(consumers, next.(xconsumer.Profiles))
		}

		consumedOpts := []obsconsumer.Option{
			obsconsumer.WithProfilesItemCounter(&tb.ReceiverProducedItems),
		}
		if isEnabled(tb.ReceiverProducedSize) {
			consumedOpts = append(consumedOpts, obsconsumer.WithProfilesSizeCounter(&tb.ReceiverProducedSize))
		}
		n.Component, err = builder.CreateProfiles(ctx, set,
			obsconsumer.NewProfiles(fanoutconsumer.NewProfiles(consumers), consumedOpts...))
	default:
		return fmt.Errorf("error creating receiver %q for data type %q is not supported", set.ID, n.pipelineType)
	}
	if err != nil {
		return fmt.Errorf("failed to create %q receiver for data type %q: %w", set.ID, n.pipelineType, err)
	}
	return nil
}
