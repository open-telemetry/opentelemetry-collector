// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentprofiles"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectorprofiles"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/pipeline"
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
	baseConsumer
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
	return n.baseConsumer
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
		capability := consumer.Capabilities{MutatesData: false}
		consumers := make(map[pipeline.ID]consumer.Traces, len(nexts))
		for _, next := range nexts {
			consumers[next.(*capabilitiesNode).pipelineID] = next.(consumer.Traces)
			capability.MutatesData = capability.MutatesData || next.Capabilities().MutatesData
		}
		next := connector.NewTracesRouter(consumers)

		switch n.exprPipelineType {
		case pipeline.SignalTraces:
			conn, err := builder.CreateTracesToTraces(ctx, set, next)
			if err != nil {
				return err
			}
			n.Component = conn
			// When connecting pipelines of the same data type, the connector must
			// inherit the capabilities of pipelines in which it is acting as a receiver.
			// Since the incoming and outgoing data types are the same, we must also consider
			// that the connector itself may MutatesData.
			capability.MutatesData = capability.MutatesData || conn.Capabilities().MutatesData
			n.baseConsumer = capabilityconsumer.NewTraces(conn, capability)
		case pipeline.SignalMetrics:
			conn, err := builder.CreateMetricsToTraces(ctx, set, next)
			if err != nil {
				return err
			}
			n.Component, n.baseConsumer = conn, conn
		case pipeline.SignalLogs:
			conn, err := builder.CreateLogsToTraces(ctx, set, next)
			if err != nil {
				return err
			}
			n.Component, n.baseConsumer = conn, conn
		case componentprofiles.SignalProfiles:
			conn, err := builder.CreateProfilesToTraces(ctx, set, next)
			if err != nil {
				return err
			}
			n.Component, n.baseConsumer = conn, conn
		}

	case pipeline.SignalMetrics:
		capability := consumer.Capabilities{MutatesData: false}
		consumers := make(map[pipeline.ID]consumer.Metrics, len(nexts))
		for _, next := range nexts {
			consumers[next.(*capabilitiesNode).pipelineID] = next.(consumer.Metrics)
			capability.MutatesData = capability.MutatesData || next.Capabilities().MutatesData
		}
		next := connector.NewMetricsRouter(consumers)

		switch n.exprPipelineType {
		case pipeline.SignalTraces:
			conn, err := builder.CreateTracesToMetrics(ctx, set, next)
			if err != nil {
				return err
			}
			n.Component, n.baseConsumer = conn, conn
		case pipeline.SignalMetrics:
			conn, err := builder.CreateMetricsToMetrics(ctx, set, next)
			if err != nil {
				return err
			}
			n.Component = conn
			// When connecting pipelines of the same data type, the connector must
			// inherit the capabilities of pipelines in which it is acting as a receiver.
			// Since the incoming and outgoing data types are the same, we must also consider
			// that the connector itself may MutatesData.
			capability.MutatesData = capability.MutatesData || conn.Capabilities().MutatesData
			n.baseConsumer = capabilityconsumer.NewMetrics(conn, capability)
		case pipeline.SignalLogs:
			conn, err := builder.CreateLogsToMetrics(ctx, set, next)
			if err != nil {
				return err
			}
			n.Component, n.baseConsumer = conn, conn
		case componentprofiles.SignalProfiles:
			conn, err := builder.CreateProfilesToMetrics(ctx, set, next)
			if err != nil {
				return err
			}
			n.Component, n.baseConsumer = conn, conn
		}
	case pipeline.SignalLogs:
		capability := consumer.Capabilities{MutatesData: false}
		consumers := make(map[pipeline.ID]consumer.Logs, len(nexts))
		for _, next := range nexts {
			consumers[next.(*capabilitiesNode).pipelineID] = next.(consumer.Logs)
			capability.MutatesData = capability.MutatesData || next.Capabilities().MutatesData
		}
		next := connector.NewLogsRouter(consumers)

		switch n.exprPipelineType {
		case pipeline.SignalTraces:
			conn, err := builder.CreateTracesToLogs(ctx, set, next)
			if err != nil {
				return err
			}
			n.Component, n.baseConsumer = conn, conn
		case pipeline.SignalMetrics:
			conn, err := builder.CreateMetricsToLogs(ctx, set, next)
			if err != nil {
				return err
			}
			n.Component, n.baseConsumer = conn, conn
		case pipeline.SignalLogs:
			conn, err := builder.CreateLogsToLogs(ctx, set, next)
			if err != nil {
				return err
			}
			n.Component = conn
			// When connecting pipelines of the same data type, the connector must
			// inherit the capabilities of pipelines in which it is acting as a receiver.
			// Since the incoming and outgoing data types are the same, we must also consider
			// that the connector itself may MutatesData.
			capability.MutatesData = capability.MutatesData || conn.Capabilities().MutatesData
			n.baseConsumer = capabilityconsumer.NewLogs(conn, capability)
		case componentprofiles.SignalProfiles:
			conn, err := builder.CreateProfilesToLogs(ctx, set, next)
			if err != nil {
				return err
			}
			n.Component, n.baseConsumer = conn, conn
		}
	case componentprofiles.SignalProfiles:
		capability := consumer.Capabilities{MutatesData: false}
		consumers := make(map[pipeline.ID]consumerprofiles.Profiles, len(nexts))
		for _, next := range nexts {
			consumers[next.(*capabilitiesNode).pipelineID] = next.(consumerprofiles.Profiles)
			capability.MutatesData = capability.MutatesData || next.Capabilities().MutatesData
		}
		next := connectorprofiles.NewProfilesRouter(consumers)

		switch n.exprPipelineType {
		case pipeline.SignalTraces:
			conn, err := builder.CreateTracesToProfiles(ctx, set, next)
			if err != nil {
				return err
			}
			n.Component, n.baseConsumer = conn, conn
		case pipeline.SignalMetrics:
			conn, err := builder.CreateMetricsToProfiles(ctx, set, next)
			if err != nil {
				return err
			}
			n.Component, n.baseConsumer = conn, conn
		case pipeline.SignalLogs:
			conn, err := builder.CreateLogsToProfiles(ctx, set, next)
			if err != nil {
				return err
			}
			n.Component, n.baseConsumer = conn, conn
		case componentprofiles.SignalProfiles:
			conn, err := builder.CreateProfilesToProfiles(ctx, set, next)
			if err != nil {
				return err
			}
			n.Component = conn
			// When connecting pipelines of the same data type, the connector must
			// inherit the capabilities of pipelines in which it is acting as a receiver.
			// Since the incoming and outgoing data types are the same, we must also consider
			// that the connector itself may MutatesData.
			capability.MutatesData = capability.MutatesData || conn.Capabilities().MutatesData
			n.baseConsumer = capabilityconsumer.NewProfiles(conn, capability)
		}
	}
	return nil
}
