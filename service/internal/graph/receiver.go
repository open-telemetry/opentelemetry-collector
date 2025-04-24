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
	set := receiver.Settings{ID: n.componentID, TelemetrySettings: tel, BuildInfo: info}
	if telemetry.NewPipelineTelemetryGate.IsEnabled() {
		set.TelemetrySettings = telemetry.WithAttributeSet(set.TelemetrySettings, *n.Set())
	}
	var err error
	switch n.pipelineType {
	case pipeline.SignalTraces:
		var consumers []consumer.Traces
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Traces))
		}
		n.Component, err = builder.CreateTraces(ctx, set, fanoutconsumer.NewTraces(consumers))
	case pipeline.SignalMetrics:
		var consumers []consumer.Metrics
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Metrics))
		}
		n.Component, err = builder.CreateMetrics(ctx, set, fanoutconsumer.NewMetrics(consumers))
	case pipeline.SignalLogs:
		var consumers []consumer.Logs
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Logs))
		}
		n.Component, err = builder.CreateLogs(ctx, set, fanoutconsumer.NewLogs(consumers))
	case xpipeline.SignalProfiles:
		var consumers []xconsumer.Profiles
		for _, next := range nexts {
			consumers = append(consumers, next.(xconsumer.Profiles))
		}
		n.Component, err = builder.CreateProfiles(ctx, set, fanoutconsumer.NewProfiles(consumers))
	default:
		return fmt.Errorf("error creating receiver %q for data type %q is not supported", set.ID, n.pipelineType)
	}
	if err != nil {
		return fmt.Errorf("failed to create %q receiver for data type %q: %w", set.ID, n.pipelineType, err)
	}
	return nil
}
