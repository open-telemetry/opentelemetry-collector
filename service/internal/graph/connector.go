// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/xconnector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/capabilityconsumer"
	"go.opentelemetry.io/collector/service/internal/components"
)

const connectorSeed = "connector"

var _ consumerNode = (*connectorNode)(nil)

// A connector instance connects one pipeline type to one other pipeline type.
// Therefore, nodeID is derived from "exporter pipeline type", "receiver pipeline type", and "component ID".
type connectorNode struct {
	nodeID
	componentID      component.ID
	exprPipelineType pipeline.Signal
	rcvrPipelineType pipeline.Signal
	component.Component
}

func newConnectorNode(exprPipelineType, rcvrPipelineType pipeline.Signal, connID component.ID) *connectorNode {
	return &connectorNode{
		nodeID:           newNodeID(connectorSeed, connID.String(), exprPipelineType.String(), rcvrPipelineType.String()),
		componentID:      connID,
		exprPipelineType: exprPipelineType,
		rcvrPipelineType: rcvrPipelineType,
	}
}

func (n *connectorNode) getConsumer() baseConsumer {
	return n.Component.(baseConsumer)
}

func (n *connectorNode) buildComponent(
	ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *builders.ConnectorBuilder,
	nexts []baseConsumer,
) error {
	tel.Logger = components.ConnectorLogger(tel.Logger, n.componentID, n.exprPipelineType, n.rcvrPipelineType)
	set := connector.Settings{ID: n.componentID, TelemetrySettings: tel, BuildInfo: info}
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
		consumers[next.(*capabilitiesNode).pipelineID] = next.(consumer.Traces)
	}
	next := connector.NewTracesRouter(consumers)

	var err error
	switch n.exprPipelineType {
	case pipeline.SignalTraces:
		var conn connector.Traces
		conn, err = builder.CreateTracesToTraces(ctx, set, next)
		if err != nil {
			return err
		}
		n.Component = componentTraces{
			Component: conn,
			Traces:    capabilityconsumer.NewTraces(conn, aggregateCap(conn, nexts)),
		}
		return nil
	case pipeline.SignalMetrics:
		n.Component, err = builder.CreateMetricsToTraces(ctx, set, next)
	case pipeline.SignalLogs:
		n.Component, err = builder.CreateLogsToTraces(ctx, set, next)
	case xpipeline.SignalProfiles:
		n.Component, err = builder.CreateProfilesToTraces(ctx, set, next)
	}
	return err
}

func (n *connectorNode) buildMetrics(
	ctx context.Context,
	set connector.Settings,
	builder *builders.ConnectorBuilder,
	nexts []baseConsumer,
) error {
	consumers := make(map[pipeline.ID]consumer.Metrics, len(nexts))
	for _, next := range nexts {
		consumers[next.(*capabilitiesNode).pipelineID] = next.(consumer.Metrics)
	}
	next := connector.NewMetricsRouter(consumers)

	var err error
	switch n.exprPipelineType {
	case pipeline.SignalMetrics:
		var conn connector.Metrics
		conn, err = builder.CreateMetricsToMetrics(ctx, set, next)
		if err != nil {
			return err
		}
		n.Component = componentMetrics{
			Component: conn,
			Metrics:   capabilityconsumer.NewMetrics(conn, aggregateCap(conn, nexts)),
		}
		return nil
	case pipeline.SignalTraces:
		n.Component, err = builder.CreateTracesToMetrics(ctx, set, next)
	case pipeline.SignalLogs:
		n.Component, err = builder.CreateLogsToMetrics(ctx, set, next)
	case xpipeline.SignalProfiles:
		n.Component, err = builder.CreateProfilesToMetrics(ctx, set, next)
	}
	return err
}

func (n *connectorNode) buildLogs(
	ctx context.Context,
	set connector.Settings,
	builder *builders.ConnectorBuilder,
	nexts []baseConsumer,
) error {
	consumers := make(map[pipeline.ID]consumer.Logs, len(nexts))
	for _, next := range nexts {
		consumers[next.(*capabilitiesNode).pipelineID] = next.(consumer.Logs)
	}
	next := connector.NewLogsRouter(consumers)

	var err error
	switch n.exprPipelineType {
	case pipeline.SignalLogs:
		var conn connector.Logs
		conn, err = builder.CreateLogsToLogs(ctx, set, next)
		if err != nil {
			return err
		}
		n.Component = componentLogs{
			Component: conn,
			Logs:      capabilityconsumer.NewLogs(conn, aggregateCap(conn, nexts)),
		}
		return nil
	case pipeline.SignalTraces:
		n.Component, err = builder.CreateTracesToLogs(ctx, set, next)
	case pipeline.SignalMetrics:
		n.Component, err = builder.CreateMetricsToLogs(ctx, set, next)
	case xpipeline.SignalProfiles:
		n.Component, err = builder.CreateProfilesToLogs(ctx, set, next)
	}
	return err
}

func (n *connectorNode) buildProfiles(
	ctx context.Context,
	set connector.Settings,
	builder *builders.ConnectorBuilder,
	nexts []baseConsumer,
) error {
	consumers := make(map[pipeline.ID]xconsumer.Profiles, len(nexts))
	for _, next := range nexts {
		consumers[next.(*capabilitiesNode).pipelineID] = next.(xconsumer.Profiles)
	}
	next := xconnector.NewProfilesRouter(consumers)

	var err error
	switch n.exprPipelineType {
	case xpipeline.SignalProfiles:
		var conn xconnector.Profiles
		conn, err = builder.CreateProfilesToProfiles(ctx, set, next)
		if err != nil {
			return err
		}
		n.Component = componentProfiles{
			Component: conn,
			Profiles:  capabilityconsumer.NewProfiles(conn, aggregateCap(conn, nexts)),
		}
		return nil
	case pipeline.SignalTraces:
		n.Component, err = builder.CreateTracesToProfiles(ctx, set, next)
	case pipeline.SignalMetrics:
		n.Component, err = builder.CreateMetricsToProfiles(ctx, set, next)
	case pipeline.SignalLogs:
		n.Component, err = builder.CreateLogsToProfiles(ctx, set, next)
	}
	return err
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
