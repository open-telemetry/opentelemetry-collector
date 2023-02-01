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

package service // import "go.opentelemetry.io/collector/service"

import (
	"context"
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

func newReceiverNode(pipelineID component.ID, recvID component.ID) *receiverNode {
	return &receiverNode{
		nodeID:       newNodeID(receiverSeed, string(pipelineID.Type()), recvID.String()),
		componentID:  recvID,
		pipelineType: pipelineID.Type(),
	}
}

func (n *receiverNode) build(
	ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *receiver.Builder,
	nexts []baseConsumer,
) error {
	set := receiver.CreateSettings{ID: n.componentID, TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ReceiverLogger(set.TelemetrySettings.Logger, n.componentID, n.pipelineType)
	r, err := buildReceiver(ctx, set, builder, component.NewIDWithName(n.pipelineType, "*"), nexts)
	n.Component = r
	return err
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

func (n *processorNode) build(
	ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *processor.Builder,
	next baseConsumer,
) error {
	set := processor.CreateSettings{ID: n.componentID, TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ProcessorLogger(set.TelemetrySettings.Logger, n.componentID, n.pipelineID)
	p, err := buildProcessor(ctx, set, builder, n.pipelineID, next)
	n.Component = p
	return err
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

func newExporterNode(pipelineID component.ID, exprID component.ID) *exporterNode {
	return &exporterNode{
		nodeID:       newNodeID(exporterSeed, string(pipelineID.Type()), exprID.String()),
		componentID:  exprID,
		pipelineType: pipelineID.Type(),
	}
}

func (n *exporterNode) getConsumer() baseConsumer {
	return n.Component.(baseConsumer)
}

func (n *exporterNode) build(
	ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *exporter.Builder,
) error {
	set := exporter.CreateSettings{ID: n.componentID, TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ExporterLogger(set.TelemetrySettings.Logger, n.componentID, n.pipelineType)
	e, err := buildExporter(ctx, set, builder, component.NewIDWithName(n.pipelineType, "*"))
	n.Component = e
	return err
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

func (n *connectorNode) build(
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
		var consumers []consumer.Traces
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Traces))
		}
		fanoutConsumer := fanoutconsumer.NewTraces(consumers)
		switch n.exprPipelineType {
		case component.DataTypeTraces:
			n.Component, err = builder.CreateTracesToTraces(ctx, set, fanoutConsumer)
		case component.DataTypeMetrics:
			n.Component, err = builder.CreateMetricsToTraces(ctx, set, fanoutConsumer)
		case component.DataTypeLogs:
			n.Component, err = builder.CreateLogsToTraces(ctx, set, fanoutConsumer)
		}
	case component.DataTypeMetrics:
		var consumers []consumer.Metrics
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Metrics))
		}
		fanoutConsumer := fanoutconsumer.NewMetrics(consumers)
		switch n.exprPipelineType {
		case component.DataTypeTraces:
			n.Component, err = builder.CreateTracesToMetrics(ctx, set, fanoutConsumer)
		case component.DataTypeMetrics:
			n.Component, err = builder.CreateMetricsToMetrics(ctx, set, fanoutConsumer)
		case component.DataTypeLogs:
			n.Component, err = builder.CreateLogsToMetrics(ctx, set, fanoutConsumer)
		}
	case component.DataTypeLogs:
		var consumers []consumer.Logs
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Logs))
		}
		fanoutConsumer := fanoutconsumer.NewLogs(consumers)
		switch n.exprPipelineType {
		case component.DataTypeTraces:
			n.Component, err = builder.CreateTracesToLogs(ctx, set, fanoutConsumer)
		case component.DataTypeMetrics:
			n.Component, err = builder.CreateMetricsToLogs(ctx, set, fanoutConsumer)
		case component.DataTypeLogs:
			n.Component, err = builder.CreateLogsToLogs(ctx, set, fanoutConsumer)
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
	consumer.Capabilities
}

func newCapabilitiesNode(pipelineID component.ID) *capabilitiesNode {
	return &capabilitiesNode{
		nodeID:       newNodeID(capabilitiesSeed, pipelineID.String()),
		pipelineID:   pipelineID,
		Capabilities: consumer.Capabilities{},
	}
}

func (n *capabilitiesNode) getConsumer() baseConsumer {
	return n.baseConsumer
}

func (n *capabilitiesNode) build(next baseConsumer, processors []*processorNode) {
	n.baseConsumer = next
	for _, proc := range processors {
		n.Capabilities.MutatesData = n.Capabilities.MutatesData ||
			proc.Component.(baseConsumer).Capabilities().MutatesData
	}
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

func (n *fanOutNode) build(nexts []baseConsumer) {
	switch n.pipelineID.Type() {
	case component.DataTypeTraces:
		consumers := make([]consumer.Traces, 0, len(nexts))
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Traces))
		}
		n.baseConsumer = fanoutconsumer.NewTraces(consumers)
	case component.DataTypeMetrics:
		consumers := make([]consumer.Metrics, 0, len(nexts))
		for _, next := range nexts {

			consumers = append(consumers, next.(consumer.Metrics))
		}
		n.baseConsumer = fanoutconsumer.NewMetrics(consumers)
	case component.DataTypeLogs:
		consumers := make([]consumer.Logs, 0, len(nexts))
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Logs))
		}
		n.baseConsumer = fanoutconsumer.NewLogs(consumers)
	}
}
