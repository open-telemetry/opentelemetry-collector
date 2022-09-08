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

type nodeID int64

func (n nodeID) ID() int64 {
	return int64(n)
}

func newNodeID(parts ...string) nodeID {
	h := fnv.New64a()
	h.Write([]byte(strings.Join(parts, "|")))
	return nodeID(h.Sum64())
}

// A receiver instance can be shared by multiple pipelines of the same type.
// Therefore, nodeID is derived from "pipeline type" and "component ID".
type receiverNode struct {
	nodeID
	componentID  component.ID
	pipelineType component.DataType
	cfg          component.Config
	factory      receiver.Factory
	component.Component
}

func newReceiverNodeID(pipelineType component.DataType, recvID component.ID) nodeID {
	return newNodeID("receiver", string(pipelineType), recvID.String())
}

func (n *receiverNode) ComponentID() component.ID {
	return n.componentID
}

func (n *receiverNode) build(
	ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	nexts []baseConsumer,
) error {
	set := receiver.CreateSettings{ID: n.ComponentID(), TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ReceiverLogger(set.TelemetrySettings.Logger, n.componentID, n.pipelineType)

	var err error
	switch n.pipelineType {
	case component.DataTypeTraces:
		var consumers []consumer.Traces
		for _, next := range nexts {
			tracesConsumer, ok := next.(consumer.Traces)
			if !ok {
				return fmt.Errorf("next component is not a traces consumer: %s", n.componentID)
			}
			consumers = append(consumers, tracesConsumer)
		}
		n.Component, err = n.factory.CreateTracesReceiver(ctx, set, n.cfg, fanoutconsumer.NewTraces(consumers))
		if err != nil {
			return err
		}
	case component.DataTypeMetrics:
		var consumers []consumer.Metrics
		for _, next := range nexts {
			metricsConsumer, ok := next.(consumer.Metrics)
			if !ok {
				return fmt.Errorf("next component is not a metrics consumer: %s", n.componentID)
			}
			consumers = append(consumers, metricsConsumer)
		}
		n.Component, err = n.factory.CreateMetricsReceiver(ctx, set, n.cfg, fanoutconsumer.NewMetrics(consumers))
		if err != nil {
			return err
		}
	case component.DataTypeLogs:
		var consumers []consumer.Logs
		for _, next := range nexts {
			logsConsumer, ok := next.(consumer.Logs)
			if !ok {
				return fmt.Errorf("next component is not a logs consumer: %s", n.componentID)
			}
			consumers = append(consumers, logsConsumer)
		}
		n.Component, err = n.factory.CreateLogsReceiver(ctx, set, n.cfg, fanoutconsumer.NewLogs(consumers))
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("error creating receiver %q, data type %q is not supported", n.componentID, n.pipelineType)
	}
	return nil
}

// Every processor instance is unique to one pipeline.
// Therefore, nodeID is derived from "pipeline ID" and "component ID".
type processorNode struct {
	nodeID
	componentID component.ID
	pipelineID  component.ID
	cfg         component.Config
	factory     processor.Factory
	component.Component
}

func (n *processorNode) ComponentID() component.ID {
	return n.componentID
}

func newProcessorNodeID(pipelineID, procID component.ID) nodeID {
	return newNodeID("processor", pipelineID.String(), procID.String())
}

func (n *processorNode) build(
	ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	next baseConsumer,
) error {
	set := processor.CreateSettings{ID: n.ComponentID(), TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ProcessorLogger(set.TelemetrySettings.Logger, n.componentID, n.pipelineID)

	var err error
	switch n.pipelineID.Type() {
	case component.DataTypeTraces:
		tracesConsumer, ok := next.(consumer.Traces)
		if !ok {
			return fmt.Errorf("next component is not a traces consumer: %s", n.componentID)
		}
		n.Component, err = n.factory.CreateTracesProcessor(ctx, set, n.cfg, tracesConsumer)
		if err != nil {
			return err
		}
	case component.DataTypeMetrics:
		metricsConsumer, ok := next.(consumer.Metrics)
		if !ok {
			return fmt.Errorf("next component is not a metrics consumer: %s", n.componentID)
		}
		n.Component, err = n.factory.CreateMetricsProcessor(ctx, set, n.cfg, metricsConsumer)
		if err != nil {
			return err
		}
	case component.DataTypeLogs:
		logsConsumer, ok := next.(consumer.Logs)
		if !ok {
			return fmt.Errorf("next component is not a logs consumer: %s", n.componentID)
		}
		n.Component, err = n.factory.CreateLogsProcessor(ctx, set, n.cfg, logsConsumer)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("error creating processor %q, data type %q is not supported", n.componentID, n.pipelineID.Type())
	}
	return nil
}

// An exporter instance can be shared by multiple pipelines of the same type.
// Therefore, nodeID is derived from "pipeline type" and "component ID".
type exporterNode struct {
	nodeID
	componentID  component.ID
	pipelineType component.DataType
	cfg          component.Config
	factory      exporter.Factory
	component.Component
}

func (n *exporterNode) ComponentID() component.ID {
	return n.componentID
}

func newExporterNodeID(pipelineType component.DataType, exprID component.ID) nodeID {
	return newNodeID("exporter", string(pipelineType), exprID.String())
}

func (n *exporterNode) build(
	ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
) error {
	set := exporter.CreateSettings{ID: n.ComponentID(), TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ExporterLogger(set.TelemetrySettings.Logger, n.componentID, n.pipelineType)

	var err error
	switch n.pipelineType {
	case component.DataTypeTraces:
		n.Component, err = n.factory.CreateTracesExporter(ctx, set, n.cfg)
		if err != nil {
			return err
		}
	case component.DataTypeMetrics:
		n.Component, err = n.factory.CreateMetricsExporter(ctx, set, n.cfg)
		if err != nil {
			return err
		}
	case component.DataTypeLogs:
		n.Component, err = n.factory.CreateLogsExporter(ctx, set, n.cfg)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("error creating exporter %q, data type %q is not supported", n.componentID, n.pipelineType)
	}
	return nil
}

// A connector instance connects one pipeline type to one other pipeline type.
// Therefore, nodeID is derived from "exporter pipeline type", "receiver pipeline type", and "component ID".
type connectorNode struct {
	nodeID
	componentID      component.ID
	exprPipelineType component.DataType
	rcvrPipelineType component.DataType
	cfg              component.Config
	factory          connector.Factory
	component.Component
}

func (n *connectorNode) ComponentID() component.ID {
	return n.componentID
}

func newConnectorNodeID(exprPipelineType, rcvrPipelineType component.DataType, connID component.ID) nodeID {
	return newNodeID("connector", connID.String(), string(exprPipelineType), string(rcvrPipelineType))
}

func (n *connectorNode) build(
	ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	nexts []baseConsumer,
) error {
	set := connector.CreateSettings{ID: n.ComponentID(), TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ConnectorLogger(set.TelemetrySettings.Logger, n.componentID, n.exprPipelineType, n.rcvrPipelineType)

	var err error
	switch n.rcvrPipelineType {
	case component.DataTypeTraces:
		var consumers []consumer.Traces
		for _, next := range nexts {
			tracesConsumer, ok := next.(consumer.Traces)
			if !ok {
				return fmt.Errorf("next component is not a traces consumer: %s", n.componentID)
			}
			consumers = append(consumers, tracesConsumer)
		}
		fanoutConsumer := fanoutconsumer.NewTraces(consumers)
		switch n.exprPipelineType {
		case component.DataTypeTraces:
			n.Component, err = n.factory.CreateTracesToTraces(ctx, set, n.cfg, fanoutConsumer)
			if err != nil {
				return err
			}
		case component.DataTypeMetrics:
			n.Component, err = n.factory.CreateMetricsToTraces(ctx, set, n.cfg, fanoutConsumer)
			if err != nil {
				return err
			}
		case component.DataTypeLogs:
			n.Component, err = n.factory.CreateLogsToTraces(ctx, set, n.cfg, fanoutConsumer)
			if err != nil {
				return err
			}
		}
	case component.DataTypeMetrics:
		var consumers []consumer.Metrics
		for _, next := range nexts {
			metricsConsumer, ok := next.(consumer.Metrics)
			if !ok {
				return fmt.Errorf("next component is not a metrics consumer: %s", n.componentID)
			}
			consumers = append(consumers, metricsConsumer)
		}
		fanoutConsumer := fanoutconsumer.NewMetrics(consumers)
		switch n.exprPipelineType {
		case component.DataTypeTraces:
			n.Component, err = n.factory.CreateTracesToMetrics(ctx, set, n.cfg, fanoutConsumer)
			if err != nil {
				return err
			}
		case component.DataTypeMetrics:
			n.Component, err = n.factory.CreateMetricsToMetrics(ctx, set, n.cfg, fanoutConsumer)
			if err != nil {
				return err
			}
		case component.DataTypeLogs:
			n.Component, err = n.factory.CreateLogsToMetrics(ctx, set, n.cfg, fanoutConsumer)
			if err != nil {
				return err
			}
		}
	case component.DataTypeLogs:
		var consumers []consumer.Logs
		for _, next := range nexts {
			logsConsumer, ok := next.(consumer.Logs)
			if !ok {
				return fmt.Errorf("next component is not a logs consumer: %s", n.componentID)
			}
			consumers = append(consumers, logsConsumer)
		}
		fanoutConsumer := fanoutconsumer.NewLogs(consumers)
		switch n.exprPipelineType {
		case component.DataTypeTraces:
			n.Component, err = n.factory.CreateTracesToLogs(ctx, set, n.cfg, fanoutConsumer)
			if err != nil {
				return err
			}
		case component.DataTypeMetrics:
			n.Component, err = n.factory.CreateMetricsToLogs(ctx, set, n.cfg, fanoutConsumer)
			if err != nil {
				return err
			}
		case component.DataTypeLogs:
			n.Component, err = n.factory.CreateLogsToLogs(ctx, set, n.cfg, fanoutConsumer)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// If a pipeline has any processors, a fan-in is added before the first one.
// The main purpose of this node is to present aggregated capabilities to receivers,
// such as whether the pipeline mutates data.
// The nodeID is derived from "pipeline ID".
type fanInNode struct {
	nodeID
	pipelineID component.ID
	baseConsumer
	consumer.Capabilities
}

func newFanInNode(pipelineID component.ID) *fanInNode {
	return &fanInNode{
		nodeID:       newNodeID("fanin_to_processors", pipelineID.String()),
		pipelineID:   pipelineID,
		Capabilities: consumer.Capabilities{},
	}
}

func (n *fanInNode) build(nextConsumer baseConsumer, processors []*processorNode) {
	n.baseConsumer = nextConsumer
	for _, proc := range processors {
		n.Capabilities.MutatesData = n.Capabilities.MutatesData ||
			proc.Component.(baseConsumer).Capabilities().MutatesData
	}
}

// Each pipeline has one fan-out node before exporters.
// Therefore, nodeID is derived from "pipeline ID".
type fanOutNode struct {
	nodeID
	pipelineID component.ID
	baseConsumer
}

func newFanOutNode(pipelineID component.ID) *fanOutNode {
	return &fanOutNode{
		nodeID:     newNodeID("fanout_to_exporters", pipelineID.String()),
		pipelineID: pipelineID,
	}
}

func (n *fanOutNode) build(nextConsumers []baseConsumer) error {
	switch n.pipelineID.Type() {
	case component.DataTypeTraces:
		consumers := make([]consumer.Traces, 0, len(nextConsumers))
		for _, next := range nextConsumers {
			consumers = append(consumers, next.(consumer.Traces))
		}
		n.baseConsumer = fanoutconsumer.NewTraces(consumers)
	case component.DataTypeMetrics:
		consumers := make([]consumer.Metrics, 0, len(nextConsumers))
		for _, next := range nextConsumers {

			consumers = append(consumers, next.(consumer.Metrics))
		}
		n.baseConsumer = fanoutconsumer.NewMetrics(consumers)
	case component.DataTypeLogs:
		consumers := make([]consumer.Logs, 0, len(nextConsumers))
		for _, next := range nextConsumers {
			consumers = append(consumers, next.(consumer.Logs))
		}
		n.baseConsumer = fanoutconsumer.NewLogs(consumers)
	default:
		return fmt.Errorf("create fan-out exporter in pipeline %q, data type %q is not supported", n.pipelineID, n.pipelineID.Type())
	}
	return nil
}
