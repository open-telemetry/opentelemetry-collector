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

	"gonum.org/v1/gonum/graph"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/service/internal/components"
	"go.opentelemetry.io/collector/service/internal/fanoutconsumer"
)

type componentNodeID int64

func (n componentNodeID) ID() int64 {
	return int64(n)
}

func newComponentNodeID(parts ...string) componentNodeID {
	h := fnv.New64a()
	for _, part := range parts {
		_, _ = h.Write([]byte(part))
	}
	return componentNodeID(int64(h.Sum64()))
}

type componentNode interface {
	ComponentID() component.ID
	component.Component
	graph.Node
}

var _ componentNode = &receiverNode{}

// A receiver instance can be shared by multiple pipelines of the same type.
// Therefore, componentNodeID is derived from "pipeline type" and "component ID".
type receiverNode struct {
	componentNodeID
	componentID  component.ID
	pipelineType component.DataType
	cfg          component.Config
	factory      component.ReceiverFactory
	component.Component
}

func newReceiverNodeID(pipelineType component.DataType, recvID component.ID) componentNodeID {
	return newComponentNodeID("receiver", string(pipelineType), recvID.String())
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
	set := component.ReceiverCreateSettings{TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = receiverLogger(set.TelemetrySettings.Logger, n.componentID, n.pipelineType)
	components.LogStabilityLevel(set.TelemetrySettings.Logger, getReceiverStabilityLevel(n.factory, n.pipelineType))

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

var _ componentNode = &processorNode{}

// Every processor instance is unique to one pipeline.
// Therefore, componentNodeID is derived from "pipeline ID" and "component ID".
type processorNode struct {
	componentNodeID
	componentID component.ID
	pipelineID  component.ID
	cfg         component.Config
	factory     component.ProcessorFactory
	component.Component
}

func (n *processorNode) ComponentID() component.ID {
	return n.componentID
}

func newProcessorNodeID(pipelineID, procID component.ID) componentNodeID {
	return newComponentNodeID("processor", pipelineID.String(), procID.String())
}

func (n *processorNode) build(
	ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	next baseConsumer,
) error {
	set := component.ProcessorCreateSettings{TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = processorLogger(set.TelemetrySettings.Logger, n.componentID, n.pipelineID)
	components.LogStabilityLevel(set.TelemetrySettings.Logger, getProcessorStabilityLevel(n.factory, n.pipelineID.Type()))

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

var _ componentNode = &exporterNode{}

// An exporter instance can be shared by multiple pipelines of the same type.
// Therefore, componentNodeID is derived from "pipeline type" and "component ID".
type exporterNode struct {
	componentNodeID
	componentID  component.ID
	pipelineType component.DataType
	cfg          component.Config
	factory      component.ExporterFactory
	component.Component
}

func (n *exporterNode) ComponentID() component.ID {
	return n.componentID
}

func newExporterNodeID(pipelineType component.DataType, exprID component.ID) componentNodeID {
	return newComponentNodeID("exporter", string(pipelineType), exprID.String())
}

func (n *exporterNode) build(
	ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
) error {
	set := component.ExporterCreateSettings{TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = exporterLogger(set.TelemetrySettings.Logger, n.componentID, n.pipelineType)
	components.LogStabilityLevel(set.TelemetrySettings.Logger, getExporterStabilityLevel(n.factory, n.pipelineType))

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

var _ componentNode = &connectorNode{}

// A connector instance connects one pipeline type to one other pipeline type.
// Therefore, componentNodeID is derived from "exporter pipeline type", "receiver pipeline type", and "component ID".
type connectorNode struct {
	componentNodeID
	componentID      component.ID
	exprPipelineType component.DataType
	rcvrPipelineType component.DataType
	cfg              component.Config
	factory          component.ConnectorFactory
	component.Component
}

func (n *connectorNode) ComponentID() component.ID {
	return n.componentID
}

type connectorConsumerTraces struct {
	component.Component
	consumer.Traces
}
type connectorConsumerMetrics struct {
	component.Component
	consumer.Metrics
}
type connectorConsumerLogs struct {
	component.Component
	consumer.Logs
}

func newConnectorNodeID(exprPipelineType, rcvrPipelineType component.DataType, connID component.ID) componentNodeID {
	return newComponentNodeID("connector", connID.String(), string(exprPipelineType), string(rcvrPipelineType))
}

func (n *connectorNode) build(
	ctx context.Context,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	nexts []baseConsumer,
) error {
	set := component.ConnectorCreateSettings{TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = connectorLogger(set.TelemetrySettings.Logger, n.componentID, n.exprPipelineType, n.rcvrPipelineType)
	components.LogStabilityLevel(set.TelemetrySettings.Logger, getConnectorStabilityLevel(n.factory, n.exprPipelineType, n.rcvrPipelineType))

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
			conn, err := n.factory.CreateTracesToTracesConnector(ctx, set, n.cfg, fanoutConsumer)
			if err != nil {
				return err
			}
			cons, consErr := consumer.NewTraces(conn.ConsumeTracesToTraces)
			if consErr != nil {
				return consErr
			}
			n.Component = connectorConsumerTraces{Component: conn, Traces: cons}
		case component.DataTypeMetrics:
			conn, err := n.factory.CreateMetricsToTracesConnector(ctx, set, n.cfg, fanoutConsumer)
			if err != nil {
				return err
			}
			cons, consErr := consumer.NewMetrics(conn.ConsumeMetricsToTraces)
			if consErr != nil {
				return consErr
			}
			n.Component = connectorConsumerMetrics{Component: conn, Metrics: cons}
		case component.DataTypeLogs:
			conn, err := n.factory.CreateLogsToTracesConnector(ctx, set, n.cfg, fanoutConsumer)
			if err != nil {
				return err
			}
			cons, consErr := consumer.NewLogs(conn.ConsumeLogsToTraces)
			if consErr != nil {
				return err
			}
			n.Component = connectorConsumerLogs{Component: conn, Logs: cons}
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
			conn, err := n.factory.CreateTracesToMetricsConnector(ctx, set, n.cfg, fanoutConsumer)
			if err != nil {
				return err
			}
			cons, consErr := consumer.NewTraces(conn.ConsumeTracesToMetrics)
			if consErr != nil {
				return err
			}
			n.Component = connectorConsumerTraces{Component: conn, Traces: cons}
		case component.DataTypeMetrics:
			conn, err := n.factory.CreateMetricsToMetricsConnector(ctx, set, n.cfg, fanoutConsumer)
			if err != nil {
				return err
			}
			cons, consErr := consumer.NewMetrics(conn.ConsumeMetricsToMetrics)
			if consErr != nil {
				return err
			}
			n.Component = connectorConsumerMetrics{Component: conn, Metrics: cons}
		case component.DataTypeLogs:
			conn, err := n.factory.CreateLogsToMetricsConnector(ctx, set, n.cfg, fanoutConsumer)
			if err != nil {
				return err
			}
			cons, consErr := consumer.NewLogs(conn.ConsumeLogsToMetrics)
			if consErr != nil {
				return err
			}
			n.Component = connectorConsumerLogs{Component: conn, Logs: cons}
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
			conn, err := n.factory.CreateTracesToLogsConnector(ctx, set, n.cfg, fanoutConsumer)
			if err != nil {
				return err
			}
			cons, consErr := consumer.NewTraces(conn.ConsumeTracesToLogs)
			if consErr != nil {
				return err
			}
			n.Component = connectorConsumerTraces{Component: conn, Traces: cons}
		case component.DataTypeMetrics:
			conn, err := n.factory.CreateMetricsToLogsConnector(ctx, set, n.cfg, fanoutConsumer)
			if err != nil {
				return err
			}
			cons, consErr := consumer.NewMetrics(conn.ConsumeMetricsToLogs)
			if consErr != nil {
				return err
			}
			n.Component = connectorConsumerMetrics{Component: conn, Metrics: cons}
		case component.DataTypeLogs:
			conn, err := n.factory.CreateLogsToLogsConnector(ctx, set, n.cfg, fanoutConsumer)
			if err != nil {
				return err
			}
			cons, consErr := consumer.NewLogs(conn.ConsumeLogsToLogs)
			if consErr != nil {
				return consErr
			}
			n.Component = connectorConsumerLogs{Component: conn, Logs: cons}
		}
	}
	return nil
}

// If a pipeline has any processors, a fan-in is added before the first one.
// The main purpose of this node is to present aggregated capabilities to receivers,
// such as whether the pipeline mutates data.
// The componentNodeID is derived from "pipeline ID".
type fanInNode struct {
	componentNodeID
	pipelineID component.ID
	baseConsumer
	consumer.Capabilities
}

func newFanInNode(pipelineID component.ID) *fanInNode {
	return &fanInNode{
		componentNodeID: newComponentNodeID("fanin_to_processors", pipelineID.String()),
		pipelineID:      pipelineID,
		Capabilities:    consumer.Capabilities{},
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
// Therefore, componentNodeID is derived from "pipeline ID".
type fanOutNode struct {
	componentNodeID
	pipelineID component.ID
	baseConsumer
}

func newFanOutNode(pipelineID component.ID) *fanOutNode {
	return &fanOutNode{
		componentNodeID: newComponentNodeID("fanout_to_exporters", pipelineID.String()),
		pipelineID:      pipelineID,
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
