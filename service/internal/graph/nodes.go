// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/internal/components"
	"go.opentelemetry.io/collector/service/internal/fanoutconsumer"
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
	pipelineType component.DataType
	component.Component
}

func newReceiverNode(pipelineType component.DataType, recvID component.ID) *receiverNode {
	return &receiverNode{
		nodeID:       newNodeID(receiverSeed, string(pipelineType), recvID.String()),
		componentID:  recvID,
		pipelineType: pipelineType,
	}
}

func (n *receiverNode) buildComponent(ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *receiver.Builder,
	nexts []baseConsumer,
) error {
	set := receiver.CreateSettings{ID: n.componentID, TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ReceiverLogger(tel.Logger, n.componentID, n.pipelineType)
	var err error
	switch n.pipelineType {
	case component.DataTypeTraces:
		var consumers []consumer.Traces
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Traces))
		}
		n.Component, err = builder.CreateTraces(ctx, set, fanoutconsumer.NewTraces(consumers))
	case component.DataTypeMetrics:
		var consumers []consumer.Metrics
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Metrics))
		}
		n.Component, err = builder.CreateMetrics(ctx, set, fanoutconsumer.NewMetrics(consumers))
	case component.DataTypeLogs:
		var consumers []consumer.Logs
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Logs))
		}
		n.Component, err = builder.CreateLogs(ctx, set, fanoutconsumer.NewLogs(consumers))
	default:
		return fmt.Errorf("error creating receiver %q for data type %q is not supported", set.ID, n.pipelineType)
	}
	if err != nil {
		return fmt.Errorf("failed to create %q receiver for data type %q: %w", set.ID, n.pipelineType, err)
	}
	return nil
}

var _ consumerNode = &processorNode{}

// Every processor instance is unique to one pipeline.
// Therefore, nodeID is derived from "pipeline ID" and "component ID".
type processorNode struct {
	nodeID
	componentID component.ID
	pipelineID  component.ID
	component.Component
}

func newProcessorNode(pipelineID, procID component.ID) *processorNode {
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
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *processor.Builder,
	next baseConsumer,
) error {
	set := processor.CreateSettings{ID: n.componentID, TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ProcessorLogger(set.TelemetrySettings.Logger, n.componentID, n.pipelineID)
	var err error
	switch n.pipelineID.Type() {
	case component.DataTypeTraces:
		n.Component, err = builder.CreateTraces(ctx, set, next.(consumer.Traces))
	case component.DataTypeMetrics:
		n.Component, err = builder.CreateMetrics(ctx, set, next.(consumer.Metrics))
	case component.DataTypeLogs:
		n.Component, err = builder.CreateLogs(ctx, set, next.(consumer.Logs))
	default:
		return fmt.Errorf("error creating processor %q in pipeline %q, data type %q is not supported", set.ID, n.pipelineID, n.pipelineID.Type())
	}
	if err != nil {
		return fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, n.pipelineID, err)
	}
	return nil
}

var _ consumerNode = &exporterNode{}

// An exporter instance can be shared by multiple pipelines of the same type.
// Therefore, nodeID is derived from "pipeline type" and "component ID".
type exporterNode struct {
	nodeID
	componentID  component.ID
	pipelineType component.DataType
	component.Component
}

func newExporterNode(pipelineType component.DataType, exprID component.ID) *exporterNode {
	return &exporterNode{
		nodeID:       newNodeID(exporterSeed, string(pipelineType), exprID.String()),
		componentID:  exprID,
		pipelineType: pipelineType,
	}
}

func (n *exporterNode) getConsumer() baseConsumer {
	return n.Component.(baseConsumer)
}

func (n *exporterNode) buildComponent(
	ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *exporter.Builder,
) error {
	set := exporter.CreateSettings{ID: n.componentID, TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ExporterLogger(set.TelemetrySettings.Logger, n.componentID, n.pipelineType)
	var err error
	switch n.pipelineType {
	case component.DataTypeTraces:
		n.Component, err = builder.CreateTraces(ctx, set)
	case component.DataTypeMetrics:
		n.Component, err = builder.CreateMetrics(ctx, set)
	case component.DataTypeLogs:
		n.Component, err = builder.CreateLogs(ctx, set)
	default:
		return fmt.Errorf("error creating exporter %q for data type %q is not supported", set.ID, n.pipelineType)
	}
	if err != nil {
		return fmt.Errorf("failed to create %q exporter for data type %q: %w", set.ID, n.pipelineType, err)
	}
	return nil
}

var _ consumerNode = &connectorNode{}

// A connector instance connects one pipeline type to one other pipeline type.
// Therefore, nodeID is derived from "exporter pipeline type", "receiver pipeline type", and "component ID".
type connectorNode struct {
	nodeID
	componentID      component.ID
	exprPipelineType component.DataType
	rcvrPipelineType component.DataType
	component.Component
}

func newConnectorNode(exprPipelineType, rcvrPipelineType component.DataType, connID component.ID) *connectorNode {
	return &connectorNode{
		nodeID:           newNodeID(connectorSeed, connID.String(), string(exprPipelineType), string(rcvrPipelineType)),
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
	builder *connector.Builder,
	nexts []baseConsumer,
) error {
	set := connector.CreateSettings{ID: n.componentID, TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ConnectorLogger(set.TelemetrySettings.Logger, n.componentID, n.exprPipelineType, n.rcvrPipelineType)

	var err error
	switch n.rcvrPipelineType {
	case component.DataTypeTraces:
		next := nexts[0].(consumer.Traces)
		if len(nexts) > 1 {
			consumers := make(map[component.ID]consumer.Traces, len(nexts))
			for _, next := range nexts {
				consumers[next.(*capabilitiesNode).pipelineID] = next.(consumer.Traces)
			}
			next = fanoutconsumer.NewTracesRouter(consumers)
		}
		switch n.exprPipelineType {
		case component.DataTypeTraces:
			n.Component, err = builder.CreateTracesToTraces(ctx, set, next)
		case component.DataTypeMetrics:
			n.Component, err = builder.CreateMetricsToTraces(ctx, set, next)
		case component.DataTypeLogs:
			n.Component, err = builder.CreateLogsToTraces(ctx, set, next)
		}
	case component.DataTypeMetrics:
		next := nexts[0].(consumer.Metrics)
		if len(nexts) > 1 {
			consumers := make(map[component.ID]consumer.Metrics, len(nexts))
			for _, next := range nexts {
				consumers[next.(*capabilitiesNode).pipelineID] = next.(consumer.Metrics)
			}
			next = fanoutconsumer.NewMetricsRouter(consumers)
		}
		switch n.exprPipelineType {
		case component.DataTypeTraces:
			n.Component, err = builder.CreateTracesToMetrics(ctx, set, next)
		case component.DataTypeMetrics:
			n.Component, err = builder.CreateMetricsToMetrics(ctx, set, next)
		case component.DataTypeLogs:
			n.Component, err = builder.CreateLogsToMetrics(ctx, set, next)
		}
	case component.DataTypeLogs:
		next := nexts[0].(consumer.Logs)
		if len(nexts) > 1 {
			consumers := make(map[component.ID]consumer.Logs, len(nexts))
			for _, next := range nexts {
				consumers[next.(*capabilitiesNode).pipelineID] = next.(consumer.Logs)
			}
			next = fanoutconsumer.NewLogsRouter(consumers)
		}
		switch n.exprPipelineType {
		case component.DataTypeTraces:
			n.Component, err = builder.CreateTracesToLogs(ctx, set, next)
		case component.DataTypeMetrics:
			n.Component, err = builder.CreateMetricsToLogs(ctx, set, next)
		case component.DataTypeLogs:
			n.Component, err = builder.CreateLogsToLogs(ctx, set, next)
		}
	}
	return err
}

var _ consumerNode = &capabilitiesNode{}

// Every pipeline has a "virtual" capabilities node immediately after the receiver(s).
// There are two purposes for this node:
// 1. Present aggregated capabilities to receivers, such as whether the pipeline mutates data.
// 2. Present a consistent "first consumer" for each pipeline.
// The nodeID is derived from "pipeline ID".
type capabilitiesNode struct {
	nodeID
	pipelineID component.ID
	baseConsumer
	consumer.ConsumeTracesFunc
	consumer.ConsumeMetricsFunc
	consumer.ConsumeLogsFunc
}

func newCapabilitiesNode(pipelineID component.ID) *capabilitiesNode {
	return &capabilitiesNode{
		nodeID:     newNodeID(capabilitiesSeed, pipelineID.String()),
		pipelineID: pipelineID,
	}
}

func (n *capabilitiesNode) getConsumer() baseConsumer {
	return n
}

var _ consumerNode = &fanOutNode{}

// Each pipeline has one fan-out node before exporters.
// Therefore, nodeID is derived from "pipeline ID".
type fanOutNode struct {
	nodeID
	pipelineID component.ID
	baseConsumer
}

func newFanOutNode(pipelineID component.ID) *fanOutNode {
	return &fanOutNode{
		nodeID:     newNodeID(fanOutToExporters, pipelineID.String()),
		pipelineID: pipelineID,
	}
}

func (n *fanOutNode) getConsumer() baseConsumer {
	return n.baseConsumer
}
