// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentprofiles"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectorprofiles"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/capabilityconsumer"
	"go.opentelemetry.io/collector/service/internal/components"
	"go.opentelemetry.io/collector/service/internal/graph/internal/metadata"
)

const (
	receiverSeed      = "receiver"
	processorSeed     = "processor"
	exporterSeed      = "exporter"
	connectorSeed     = "connector"
	capabilitiesSeed  = "capabilities"
	fanOutToExporters = "fanout_to_exporters"
)

// baseConsumer redeclared here since not public in consumer package. May consider to make that public.
type baseConsumer interface {
	Capabilities() consumer.Capabilities
}

type nodeID int64

func (n nodeID) ID() int64 {
	return int64(n)
}

func newNodeID(parts ...string) nodeID {
	h := fnv.New64a()
	h.Write([]byte(strings.Join(parts, "|")))
	return nodeID(h.Sum64())
}

type consumerNode interface {
	getConsumer() baseConsumer
}

// A receiver instance can be shared by multiple pipelines of the same type.
// Therefore, nodeID is derived from "pipeline type" and "component ID".
type receiverNode struct {
	nodeID
	componentID  component.ID
	pipelineType pipeline.Signal
	component.Component
}

func newReceiverNode(pipelineType pipeline.Signal, recvID component.ID) *receiverNode {
	return &receiverNode{
		nodeID:       newNodeID(receiverSeed, pipelineType.String(), recvID.String()),
		componentID:  recvID,
		pipelineType: pipelineType,
	}
}

func (n *receiverNode) buildComponent(ctx context.Context,
	tb *metadata.TelemetryBuilder,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *builders.ReceiverBuilder,
	nexts []baseConsumer,
) error {
	tel.Logger = components.ReceiverLogger(tel.Logger, n.componentID, n.pipelineType)
	set := receiver.Settings{ID: n.componentID, TelemetrySettings: tel, BuildInfo: info}
	var err error
	switch n.pipelineType {
	case pipeline.SignalTraces:
		var consumers []consumer.Traces
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Traces))
		}
		n.Component, err = builder.CreateTraces(ctx, set, obsTracesConsumer{
			Traces:      fanoutconsumer.NewTraces(consumers),
			itemCounter: tb.ReceiverOutgoingItems,
			// TODO add attributes
		})
	case pipeline.SignalMetrics:
		var consumers []consumer.Metrics
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Metrics))
		}
		n.Component, err = builder.CreateMetrics(ctx, set, obsMetricsConsumer{
			Metrics:     fanoutconsumer.NewMetrics(consumers),
			itemCounter: tb.ReceiverOutgoingItems,
			// TODO add attributes
		})
	case pipeline.SignalLogs:
		var consumers []consumer.Logs
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Logs))
		}
		n.Component, err = builder.CreateLogs(ctx, set, obsLogsConsumer{
			Logs:        fanoutconsumer.NewLogs(consumers),
			itemCounter: tb.ReceiverOutgoingItems,
			// TODO add attributes
		})
	case componentprofiles.SignalProfiles:
		var consumers []consumerprofiles.Profiles
		for _, next := range nexts {
			consumers = append(consumers, next.(consumerprofiles.Profiles))
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

var _ consumerNode = (*processorNode)(nil)

// Every processor instance is unique to one pipeline.
// Therefore, nodeID is derived from "pipeline ID" and "component ID".
type processorNode struct {
	nodeID
	componentID component.ID
	pipelineID  pipeline.ID
	component.Component
}

func newProcessorNode(pipelineID pipeline.ID, procID component.ID) *processorNode {
	return &processorNode{
		nodeID:      newNodeID(processorSeed, pipelineID.String(), procID.String()),
		componentID: procID,
		pipelineID:  pipelineID,
	}
}

func (n *processorNode) getConsumer() baseConsumer {
	return n.Component.(baseConsumer)
}

func (n *processorNode) buildComponent(ctx context.Context,
	tb *metadata.TelemetryBuilder,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *builders.ProcessorBuilder,
	next baseConsumer,
) error {
	tel.Logger = components.ProcessorLogger(tel.Logger, n.componentID, n.pipelineID)
	set := processor.Settings{ID: n.componentID, TelemetrySettings: tel, BuildInfo: info}

	switch n.pipelineID.Signal() {
	case pipeline.SignalTraces:
		c, err := builder.CreateTraces(ctx, set, obsTracesConsumer{
			Traces:      next.(consumer.Traces),
			itemCounter: tb.ProcessorOutgoingItems,
			// TODO add attributes
		})
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}
		n.Component = obsTracesProcessor{
			Traces:      c.(processor.Traces),
			itemCounter: tb.ProcessorIncomingItems,
			// TODO add attributes
		}
	case pipeline.SignalMetrics:
		c, err := builder.CreateMetrics(ctx, set, obsMetricsConsumer{
			Metrics:     next.(consumer.Metrics),
			itemCounter: tb.ProcessorOutgoingItems,
			// TODO add attributes
		})
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}
		n.Component = obsMetricsProcessor{
			Metrics:     c.(processor.Metrics),
			itemCounter: tb.ProcessorIncomingItems,
			// TODO add attributes
		}
	case pipeline.SignalLogs:
		c, err := builder.CreateLogs(ctx, set, obsLogsConsumer{
			Logs:        next.(consumer.Logs),
			itemCounter: tb.ProcessorOutgoingItems,
			// TODO add attributes
		})
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}
		n.Component = obsLogsProcessor{
			Logs:        c.(processor.Logs),
			itemCounter: tb.ProcessorIncomingItems,
			// TODO add attributes
		}
	case componentprofiles.SignalProfiles:
		c, err := builder.CreateProfiles(ctx, set, next.(consumerprofiles.Profiles))
		if err != nil {
			return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID.String(), err)
		}
		n.Component = c
	default:
		return fmt.Errorf("error creating processor %q in pipeline %q, data type %q is not supported", set.ID, n.pipelineID.String(), n.pipelineID.Signal())
	}
	return nil
}

var _ consumerNode = (*exporterNode)(nil)

// An exporter instance can be shared by multiple pipelines of the same type.
// Therefore, nodeID is derived from "pipeline type" and "component ID".
type exporterNode struct {
	nodeID
	componentID  component.ID
	pipelineType pipeline.Signal
	component.Component
}

func newExporterNode(pipelineType pipeline.Signal, exprID component.ID) *exporterNode {
	return &exporterNode{
		nodeID:       newNodeID(exporterSeed, pipelineType.String(), exprID.String()),
		componentID:  exprID,
		pipelineType: pipelineType,
	}
}

func (n *exporterNode) getConsumer() baseConsumer {
	return n.Component.(baseConsumer)
}

func (n *exporterNode) buildComponent(
	ctx context.Context,
	tb *metadata.TelemetryBuilder,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *builders.ExporterBuilder,
) error {
	tel.Logger = components.ExporterLogger(tel.Logger, n.componentID, n.pipelineType)
	set := exporter.Settings{ID: n.componentID, TelemetrySettings: tel, BuildInfo: info}

	switch n.pipelineType {
	case pipeline.SignalTraces:
		c, err := builder.CreateTraces(ctx, set)
		if err != nil {
			return fmt.Errorf("failed to create %q exporter for data type %q: %w", set.ID, n.pipelineType, err)
		}
		n.Component = obsTracesExporter{
			Traces:      c.(exporter.Traces),
			itemCounter: tb.ExporterIncomingItems,
			// TODO add attributes
		}
	case pipeline.SignalMetrics:
		c, err := builder.CreateMetrics(ctx, set)
		if err != nil {
			return fmt.Errorf("failed to create %q exporter for data type %q: %w", set.ID, n.pipelineType, err)
		}
		n.Component = obsMetricsExporter{
			Metrics:     c.(exporter.Metrics),
			itemCounter: tb.ExporterIncomingItems,
			// TODO add attributes
		}
	case pipeline.SignalLogs:
		c, err := builder.CreateLogs(ctx, set)
		if err != nil {
			return fmt.Errorf("failed to create %q exporter for data type %q: %w", set.ID, n.pipelineType, err)
		}
		n.Component = obsLogsExporter{
			Logs:        c.(exporter.Logs),
			itemCounter: tb.ExporterIncomingItems,
			// TODO add attributes
		}
	case componentprofiles.SignalProfiles:
		c, err := builder.CreateProfiles(ctx, set)
		if err != nil {
			return fmt.Errorf("failed to create %q exporter for data type %q: %w", set.ID, n.pipelineType, err)
		}
		n.Component = c
	default:
		return fmt.Errorf("error creating exporter %q for data type %q is not supported", set.ID, n.pipelineType)
	}

	return nil
}

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
	tb *metadata.TelemetryBuilder,
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
		next := obsTracesConsumer{
			Traces:      connector.NewTracesRouter(consumers),
			itemCounter: tb.ConnectorOutgoingItems,
			// TODO add attributes
		}

		switch n.exprPipelineType {
		case pipeline.SignalTraces:
			conn, err := builder.CreateTracesToTraces(ctx, set, next)
			if err != nil {
				return err
			}
			obsConn := obsTracesConnector{
				Traces:      conn,
				itemCounter: tb.ConnectorIncomingItems,
				// TODO add attributes
			}
			n.Component = obsConn
			// When connecting pipelines of the same data type, the connector must
			// inherit the capabilities of pipelines in which it is acting as a receiver.
			// Since the incoming and outgoing data types are the same, we must also consider
			// that the connector itself may MutatesData.
			capability.MutatesData = capability.MutatesData || obsConn.Capabilities().MutatesData
			n.baseConsumer = capabilityconsumer.NewTraces(obsConn, capability)
		case pipeline.SignalMetrics:
			conn, err := builder.CreateMetricsToTraces(ctx, set, next)
			if err != nil {
				return err
			}
			obsConn := obsMetricsConnector{
				Metrics:     conn,
				itemCounter: tb.ConnectorIncomingItems,
				// TODO add attributes
			}
			n.Component, n.baseConsumer = obsConn, obsConn
		case pipeline.SignalLogs:
			conn, err := builder.CreateLogsToTraces(ctx, set, next)
			if err != nil {
				return err
			}
			obsConn := obsLogsConnector{
				Logs:        conn,
				itemCounter: tb.ConnectorIncomingItems,
				// TODO add attributes
			}
			n.Component, n.baseConsumer = obsConn, obsConn
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
		next := obsMetricsConsumer{
			Metrics:     connector.NewMetricsRouter(consumers),
			itemCounter: tb.ConnectorOutgoingItems,
			// TODO add attributes
		}

		switch n.exprPipelineType {
		case pipeline.SignalTraces:
			conn, err := builder.CreateTracesToMetrics(ctx, set, next)
			if err != nil {
				return err
			}
			obsConn := obsTracesConnector{
				Traces:      conn,
				itemCounter: tb.ConnectorIncomingItems,
				// TODO add attributes
			}
			n.Component, n.baseConsumer = obsConn, obsConn
		case pipeline.SignalMetrics:
			conn, err := builder.CreateMetricsToMetrics(ctx, set, next)
			if err != nil {
				return err
			}
			obsConn := obsMetricsConnector{
				Metrics:     conn,
				itemCounter: tb.ConnectorIncomingItems,
				// TODO add attributes
			}
			n.Component = obsConn
			// When connecting pipelines of the same data type, the connector must
			// inherit the capabilities of pipelines in which it is acting as a receiver.
			// Since the incoming and outgoing data types are the same, we must also consider
			// that the connector itself may MutatesData.
			capability.MutatesData = capability.MutatesData || obsConn.Capabilities().MutatesData
			n.baseConsumer = capabilityconsumer.NewMetrics(obsConn, capability)
		case pipeline.SignalLogs:
			conn, err := builder.CreateLogsToMetrics(ctx, set, next)
			if err != nil {
				return err
			}
			obsConn := obsLogsConnector{
				Logs:        conn,
				itemCounter: tb.ConnectorIncomingItems,
				// TODO add attributes
			}
			n.Component, n.baseConsumer = obsConn, obsConn
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
		next := obsLogsConsumer{
			Logs:        connector.NewLogsRouter(consumers),
			itemCounter: tb.ConnectorOutgoingItems,
			// TODO add attributes
		}

		switch n.exprPipelineType {
		case pipeline.SignalTraces:
			conn, err := builder.CreateTracesToLogs(ctx, set, next)
			if err != nil {
				return err
			}
			obsConn := obsTracesConnector{
				Traces:      conn,
				itemCounter: tb.ConnectorIncomingItems,
				// TODO add attributes
			}
			n.Component, n.baseConsumer = obsConn, obsConn
		case pipeline.SignalMetrics:
			conn, err := builder.CreateMetricsToLogs(ctx, set, next)
			if err != nil {
				return err
			}
			obsConn := obsMetricsConnector{
				Metrics:     conn,
				itemCounter: tb.ConnectorIncomingItems,
				// TODO add attributes
			}
			n.Component, n.baseConsumer = obsConn, obsConn
		case pipeline.SignalLogs:
			conn, err := builder.CreateLogsToLogs(ctx, set, next)
			if err != nil {
				return err
			}
			obsConn := obsLogsConnector{
				Logs:        conn,
				itemCounter: tb.ConnectorIncomingItems,
				// TODO add attributes
			}
			n.Component = obsConn
			// When connecting pipelines of the same data type, the connector must
			// inherit the capabilities of pipelines in which it is acting as a receiver.
			// Since the incoming and outgoing data types are the same, we must also consider
			// that the connector itself may MutatesData.
			capability.MutatesData = capability.MutatesData || obsConn.Capabilities().MutatesData
			n.baseConsumer = capabilityconsumer.NewLogs(obsConn, capability)
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

var _ consumerNode = (*capabilitiesNode)(nil)

// Every pipeline has a "virtual" capabilities node immediately after the receiver(s).
// There are two purposes for this node:
// 1. Present aggregated capabilities to receivers, such as whether the pipeline mutates data.
// 2. Present a consistent "first consumer" for each pipeline.
// The nodeID is derived from "pipeline ID".
type capabilitiesNode struct {
	nodeID
	pipelineID pipeline.ID
	baseConsumer
	consumer.ConsumeTracesFunc
	consumer.ConsumeMetricsFunc
	consumer.ConsumeLogsFunc
	consumerprofiles.ConsumeProfilesFunc
}

func newCapabilitiesNode(pipelineID pipeline.ID) *capabilitiesNode {
	return &capabilitiesNode{
		nodeID:     newNodeID(capabilitiesSeed, pipelineID.String()),
		pipelineID: pipelineID,
	}
}

func (n *capabilitiesNode) getConsumer() baseConsumer {
	return n
}

var _ consumerNode = (*fanOutNode)(nil)

// Each pipeline has one fan-out node before exporters.
// Therefore, nodeID is derived from "pipeline ID".
type fanOutNode struct {
	nodeID
	pipelineID pipeline.ID
	baseConsumer
}

func newFanOutNode(pipelineID pipeline.ID) *fanOutNode {
	return &fanOutNode{
		nodeID:     newNodeID(fanOutToExporters, pipelineID.String()),
		pipelineID: pipelineID,
	}
}

func (n *fanOutNode) getConsumer() baseConsumer {
	return n.baseConsumer
}
