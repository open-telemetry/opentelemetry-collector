// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"context"

	otelattr "go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/xconnector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/service/internal/attribute"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/capabilityconsumer"
	"go.opentelemetry.io/collector/service/internal/metadata"
)

const pipelineIDAttrKey = "otelcol.pipeline.id.output"

var _ consumerNode = (*connectorNode)(nil)

type connectorNode struct {
	attribute.Attributes
	componentID      component.ID
	exprPipelineType pipeline.Signal
	rcvrPipelineType pipeline.Signal
	component.Component
	consumer baseConsumer
}

func newConnectorNode(exprPipelineType, rcvrPipelineType pipeline.Signal, connID component.ID) *connectorNode {
	return &connectorNode{
		Attributes:       attribute.Connector(exprPipelineType, rcvrPipelineType, connID),
		componentID:      connID,
		exprPipelineType: exprPipelineType,
		rcvrPipelineType: rcvrPipelineType,
	}
}

func (n *connectorNode) getConsumer() baseConsumer {
	return n.consumer
}

func (n *connectorNode) buildComponent(
	ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *builders.ConnectorBuilder,
	nexts []baseConsumer,
) error {
	set := connector.Settings{
		ID:                n.componentID,
		TelemetrySettings: telemetry.WithAttributeSet(tel, *n.Set()),
		BuildInfo:         info,
	}

	switch n.rcvrPipelineType {
	case pipeline.SignalTraces:
		return n.buildTraces(ctx, set, builder, nexts)
	case pipeline.SignalMetrics:
		return n.buildMetrics(ctx, set, builder, nexts)
	case pipeline.SignalLogs:
		return n.buildLogs(ctx, set, builder, nexts)
	case xpipeline.SignalProfiles:
		return n.buildProfiles(ctx, set, builder, nexts)
	}
	return nil
}

func (n *connectorNode) buildTraces(
	ctx context.Context,
	set connector.Settings,
	builder *builders.ConnectorBuilder,
	nexts []baseConsumer,
) error {
	consumers := make(map[pipeline.ID]consumer.Traces, len(nexts))
	for _, next := range nexts {
		pipelineAttrs := otelattr.String(pipelineIDAttrKey, next.(*capabilitiesNode).pipelineID.String())
		routeSet := otelattr.NewSet(append(n.Set().ToSlice(), pipelineAttrs)...)
		tb, err := metadata.NewTelemetryBuilder(telemetry.WithAttributeSet(set.TelemetrySettings, routeSet))
		if err != nil {
			return err
		}
		consumers[next.(*capabilitiesNode).pipelineID] = obsConsumerTraces{
			Traces:      next.(consumer.Traces),
			itemCounter: tb.ConnectorProducedItems,
		}
	}
	next := connector.NewTracesRouter(consumers)

	tb, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return err
	}

	switch n.exprPipelineType {
	case pipeline.SignalTraces:
		n.Component, err = builder.CreateTracesToTraces(ctx, set, next)
		if err != nil {
			return err
		}
		n.consumer = obsConsumerTraces{
			// Connectors which might pass along data must inherit capabilities of all nexts
			Traces: capabilityconsumer.NewTraces(
				n.Component.(consumer.Traces),
				aggregateCap(n.Component.(consumer.Traces), nexts),
			),
			itemCounter: tb.ConnectorConsumedItems,
		}
	case pipeline.SignalMetrics:
		n.Component, err = builder.CreateMetricsToTraces(ctx, set, next)
		if err != nil {
			return err
		}
		n.consumer = obsConsumerMetrics{
			Metrics:     n.Component.(consumer.Metrics),
			itemCounter: tb.ConnectorConsumedItems,
		}
	case pipeline.SignalLogs:
		n.Component, err = builder.CreateLogsToTraces(ctx, set, next)
		if err != nil {
			return err
		}
		n.consumer = obsConsumerLogs{
			Logs:        n.Component.(consumer.Logs),
			itemCounter: tb.ConnectorConsumedItems,
		}
	case xpipeline.SignalProfiles:
		n.Component, err = builder.CreateProfilesToTraces(ctx, set, next)
		if err != nil {
			return err
		}
		n.consumer = obsConsumerProfiles{
			Profiles:    n.Component.(xconsumer.Profiles),
			itemCounter: tb.ConnectorConsumedItems,
		}
	}
	return nil
}

func (n *connectorNode) buildMetrics(
	ctx context.Context,
	set connector.Settings,
	builder *builders.ConnectorBuilder,
	nexts []baseConsumer,
) error {
	consumers := make(map[pipeline.ID]consumer.Metrics, len(nexts))
	for _, next := range nexts {
		pipelineAttrs := otelattr.String(pipelineIDAttrKey, next.(*capabilitiesNode).pipelineID.String())
		routeSet := otelattr.NewSet(append(n.Set().ToSlice(), pipelineAttrs)...)
		tb, err := metadata.NewTelemetryBuilder(telemetry.WithAttributeSet(set.TelemetrySettings, routeSet))
		if err != nil {
			return err
		}
		consumers[next.(*capabilitiesNode).pipelineID] = obsConsumerMetrics{
			Metrics:     next.(consumer.Metrics),
			itemCounter: tb.ConnectorProducedItems,
		}
	}
	next := connector.NewMetricsRouter(consumers)

	tb, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return err
	}

	switch n.exprPipelineType {
	case pipeline.SignalMetrics:
		n.Component, err = builder.CreateMetricsToMetrics(ctx, set, next)
		if err != nil {
			return err
		}
		n.consumer = obsConsumerMetrics{
			// Connectors which might pass along data must inherit capabilities of all nexts
			Metrics: capabilityconsumer.NewMetrics(
				n.Component.(consumer.Metrics),
				aggregateCap(n.Component.(consumer.Metrics), nexts),
			),
			itemCounter: tb.ConnectorConsumedItems,
		}
	case pipeline.SignalTraces:
		n.Component, err = builder.CreateTracesToMetrics(ctx, set, next)
		if err != nil {
			return err
		}
		n.consumer = obsConsumerTraces{
			Traces:      n.Component.(consumer.Traces),
			itemCounter: tb.ConnectorConsumedItems,
		}
	case pipeline.SignalLogs:
		n.Component, err = builder.CreateLogsToMetrics(ctx, set, next)
		if err != nil {
			return err
		}
		n.consumer = obsConsumerLogs{
			Logs:        n.Component.(consumer.Logs),
			itemCounter: tb.ConnectorConsumedItems,
		}
	case xpipeline.SignalProfiles:
		n.Component, err = builder.CreateProfilesToMetrics(ctx, set, next)
		if err != nil {
			return err
		}
		n.consumer = obsConsumerProfiles{
			Profiles:    n.Component.(xconsumer.Profiles),
			itemCounter: tb.ConnectorConsumedItems,
		}
	}
	return nil
}

func (n *connectorNode) buildLogs(
	ctx context.Context,
	set connector.Settings,
	builder *builders.ConnectorBuilder,
	nexts []baseConsumer,
) error {
	consumers := make(map[pipeline.ID]consumer.Logs, len(nexts))
	for _, next := range nexts {
		pipelineAttrs := otelattr.String(pipelineIDAttrKey, next.(*capabilitiesNode).pipelineID.String())
		routeSet := otelattr.NewSet(append(n.Set().ToSlice(), pipelineAttrs)...)
		tb, err := metadata.NewTelemetryBuilder(telemetry.WithAttributeSet(set.TelemetrySettings, routeSet))
		if err != nil {
			return err
		}
		consumers[next.(*capabilitiesNode).pipelineID] = obsConsumerLogs{
			Logs:        next.(consumer.Logs),
			itemCounter: tb.ConnectorProducedItems,
		}
	}
	next := connector.NewLogsRouter(consumers)

	tb, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return err
	}

	switch n.exprPipelineType {
	case pipeline.SignalLogs:
		n.Component, err = builder.CreateLogsToLogs(ctx, set, next)
		if err != nil {
			return err
		}
		n.consumer = obsConsumerLogs{
			// Connectors which might pass along data must inherit capabilities of all nexts
			Logs: capabilityconsumer.NewLogs(
				n.Component.(consumer.Logs),
				aggregateCap(n.Component.(consumer.Logs), nexts),
			),
			itemCounter: tb.ConnectorConsumedItems,
		}
	case pipeline.SignalTraces:
		n.Component, err = builder.CreateTracesToLogs(ctx, set, next)
		if err != nil {
			return err
		}
		n.consumer = obsConsumerTraces{
			Traces:      n.Component.(consumer.Traces),
			itemCounter: tb.ConnectorConsumedItems,
		}
	case pipeline.SignalMetrics:
		n.Component, err = builder.CreateMetricsToLogs(ctx, set, next)
		if err != nil {
			return err
		}
		n.consumer = obsConsumerMetrics{
			Metrics:     n.Component.(consumer.Metrics),
			itemCounter: tb.ConnectorConsumedItems,
		}
	case xpipeline.SignalProfiles:
		n.Component, err = builder.CreateProfilesToLogs(ctx, set, next)
		if err != nil {
			return err
		}
		n.consumer = obsConsumerProfiles{
			Profiles:    n.Component.(xconsumer.Profiles),
			itemCounter: tb.ConnectorConsumedItems,
		}
	}
	return nil
}

func (n *connectorNode) buildProfiles(
	ctx context.Context,
	set connector.Settings,
	builder *builders.ConnectorBuilder,
	nexts []baseConsumer,
) error {
	consumers := make(map[pipeline.ID]xconsumer.Profiles, len(nexts))
	for _, next := range nexts {
		pipelineAttrs := otelattr.String(pipelineIDAttrKey, next.(*capabilitiesNode).pipelineID.String())
		routeSet := otelattr.NewSet(append(n.Set().ToSlice(), pipelineAttrs)...)
		tb, err := metadata.NewTelemetryBuilder(telemetry.WithAttributeSet(set.TelemetrySettings, routeSet))
		if err != nil {
			return err
		}
		consumers[next.(*capabilitiesNode).pipelineID] = obsConsumerProfiles{
			Profiles:    next.(xconsumer.Profiles),
			itemCounter: tb.ConnectorProducedItems,
		}
	}
	next := xconnector.NewProfilesRouter(consumers)

	tb, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return err
	}

	switch n.exprPipelineType {
	case xpipeline.SignalProfiles:
		n.Component, err = builder.CreateProfilesToProfiles(ctx, set, next)
		if err != nil {
			return err
		}
		n.consumer = obsConsumerProfiles{
			// Connectors which might pass along data must inherit capabilities of all nexts
			Profiles: capabilityconsumer.NewProfiles(
				n.Component.(xconsumer.Profiles),
				aggregateCap(n.Component.(xconsumer.Profiles), nexts),
			),
			itemCounter: tb.ConnectorConsumedItems,
		}
	case pipeline.SignalTraces:
		n.Component, err = builder.CreateTracesToProfiles(ctx, set, next)
		if err != nil {
			return err
		}
		n.consumer = obsConsumerTraces{
			Traces:      n.Component.(consumer.Traces),
			itemCounter: tb.ConnectorConsumedItems,
		}
	case pipeline.SignalMetrics:
		n.Component, err = builder.CreateMetricsToProfiles(ctx, set, next)
		if err != nil {
			return err
		}
		n.consumer = obsConsumerMetrics{
			Metrics:     n.Component.(consumer.Metrics),
			itemCounter: tb.ConnectorConsumedItems,
		}
	case pipeline.SignalLogs:
		n.Component, err = builder.CreateLogsToProfiles(ctx, set, next)
		if err != nil {
			return err
		}
		n.consumer = obsConsumerLogs{
			Logs:        n.Component.(consumer.Logs),
			itemCounter: tb.ConnectorConsumedItems,
		}
	}
	return nil
}

// When connecting pipelines of the same data type, the connector must
// inherit the capabilities of pipelines in which it is acting as a receiver.
// Since the incoming and outgoing data types are the same, we must also consider
// that the connector itself may mutate the data and pass it along.
func aggregateCap(base baseConsumer, nexts []baseConsumer) consumer.Capabilities {
	capabilities := base.Capabilities()
	for _, next := range nexts {
		capabilities.MutatesData = capabilities.MutatesData || next.Capabilities().MutatesData
	}
	return capabilities
}
