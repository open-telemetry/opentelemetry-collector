// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"context"
	"errors"
	"hash/fnv"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentprofiles"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectorprofiles"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterprofiles"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorprofiles"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverprofiles"
	"go.opentelemetry.io/collector/service/internal/testcomponents"
	"go.opentelemetry.io/collector/service/pipelines"
)

var _ component.Component = (*testNode)(nil)

type testNode struct {
	id          component.ID
	startErr    error
	shutdownErr error
}

// ID satisfies the graph.Node interface, allowing
// testNode to be used in a simple.DirectedGraph
func (n *testNode) ID() int64 {
	h := fnv.New64a()
	h.Write([]byte(n.id.String()))
	return int64(h.Sum64())
}

func (n *testNode) Start(ctx context.Context, _ component.Host) error {
	if n.startErr != nil {
		return n.startErr
	}
	if cwo, ok := ctx.(*contextWithOrder); ok {
		cwo.record(n.id)
	}
	return nil
}

func (n *testNode) Shutdown(ctx context.Context) error {
	if n.shutdownErr != nil {
		return n.shutdownErr
	}
	if cwo, ok := ctx.(*contextWithOrder); ok {
		cwo.record(n.id)
	}
	return nil
}

type contextWithOrder struct {
	context.Context
	sync.Mutex
	next  int
	order map[component.ID]int
}

func (c *contextWithOrder) record(id component.ID) {
	c.Lock()
	c.order[id] = c.next
	c.next++
	c.Unlock()
}

func (g *Graph) getReceivers() map[pipeline.Signal]map[component.ID]component.Component {
	receiversMap := make(map[pipeline.Signal]map[component.ID]component.Component)
	receiversMap[pipeline.SignalTraces] = make(map[component.ID]component.Component)
	receiversMap[pipeline.SignalMetrics] = make(map[component.ID]component.Component)
	receiversMap[pipeline.SignalLogs] = make(map[component.ID]component.Component)
	receiversMap[componentprofiles.SignalProfiles] = make(map[component.ID]component.Component)

	for _, pg := range g.pipelines {
		for _, rcvrNode := range pg.receivers {
			rcvrOrConnNode := g.componentGraph.Node(rcvrNode.ID())
			rcvrNode, ok := rcvrOrConnNode.(*receiverNode)
			if !ok {
				continue
			}
			receiversMap[rcvrNode.pipelineType][rcvrNode.componentID] = rcvrNode.Component
		}
	}
	return receiversMap
}

// Calculates the expected number of receiver and exporter instances in the specified pipeline.
//
// Expect one instance of each receiver and exporter, unless it is a connector.
//
// For Connectors:
// - Let E equal the number of pipeline types in which the connector is used as an exporter.
// - Let R equal the number of pipeline types in which the connector is used as a receiver.
//
// Within the graph as a whole, we expect E*R instances, i.e. one per combination of data types.
//
// However, within an individual pipeline, we expect:
// - E instances of the connector as a receiver.
// - R instances of the connector as an exporter.
func expectedInstances(m pipelines.Config, pID pipeline.ID) (int, int) {
	exConnectorType := component.MustNewType("exampleconnector")
	var r, e int
	for _, rID := range m[pID].Receivers {
		if rID.Type() != exConnectorType {
			r++
			continue
		}

		// This is a connector. Count the pipeline types where it is an exporter.
		typeMap := map[pipeline.Signal]bool{}
		for pID, pCfg := range m {
			for _, eID := range pCfg.Exporters {
				if eID == rID {
					typeMap[pID.Signal()] = true
				}
			}
		}
		r += len(typeMap)
	}
	for _, eID := range m[pID].Exporters {
		if eID.Type() != exConnectorType {
			e++
			continue
		}

		// This is a connector. Count the pipeline types where it is a receiver.
		typeMap := map[pipeline.Signal]bool{}
		for pID, pCfg := range m {
			for _, rID := range pCfg.Receivers {
				if rID == eID {
					typeMap[pID.Signal()] = true
				}
			}
		}
		e += len(typeMap)
	}
	return r, e
}

// connector needs to be unwrapped to access component as ExampleConnector
func unwrapExampleConnector(c *connectorNode) *testcomponents.ExampleConnector {
	switch ct := c.Component.(type) {
	case componentTraces: // consumes traces, emits traces
		return ct.Component.(*testcomponents.ExampleConnector)
	case connector.Traces: // consumes traces, emits something else
		return ct.(*testcomponents.ExampleConnector)
	case componentMetrics: // consumes metrics, emits metrics
		return ct.Component.(*testcomponents.ExampleConnector)
	case connector.Metrics: // consumes metrics, emits something else
		return ct.(*testcomponents.ExampleConnector)
	case componentLogs: // consumes logs, emits logs
		return ct.Component.(*testcomponents.ExampleConnector)
	case connector.Logs: // consumes logs, emits something else
		return ct.(*testcomponents.ExampleConnector)
	case componentProfiles: // consumes profiles, emits profiles
		return ct.Component.(*testcomponents.ExampleConnector)
	case connectorprofiles.Profiles: // consumes profiles, emits something else
		return ct.(*testcomponents.ExampleConnector)
	}
	return nil
}

func newBadReceiverFactory() receiver.Factory {
	return receiver.NewFactory(component.MustNewType("bf"), func() component.Config {
		return &struct{}{}
	})
}

func newBadProcessorFactory() processor.Factory {
	return processor.NewFactory(component.MustNewType("bf"), func() component.Config {
		return &struct{}{}
	})
}

func newBadExporterFactory() exporter.Factory {
	return exporter.NewFactory(component.MustNewType("bf"), func() component.Config {
		return &struct{}{}
	})
}

func newBadConnectorFactory() connector.Factory {
	return connector.NewFactory(component.MustNewType("bf"), func() component.Config {
		return &struct{}{}
	})
}

func newErrReceiverFactory() receiver.Factory {
	return receiverprofiles.NewFactory(component.MustNewType("err"),
		func() component.Config { return &struct{}{} },
		receiverprofiles.WithTraces(func(context.Context, receiver.Settings, component.Config, consumer.Traces) (receiver.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		receiverprofiles.WithLogs(func(context.Context, receiver.Settings, component.Config, consumer.Logs) (receiver.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		receiverprofiles.WithMetrics(func(context.Context, receiver.Settings, component.Config, consumer.Metrics) (receiver.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		receiverprofiles.WithProfiles(func(context.Context, receiver.Settings, component.Config, consumerprofiles.Profiles) (receiverprofiles.Profiles, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func newErrProcessorFactory() processor.Factory {
	return processorprofiles.NewFactory(component.MustNewType("err"),
		func() component.Config { return &struct{}{} },
		processorprofiles.WithTraces(func(context.Context, processor.Settings, component.Config, consumer.Traces) (processor.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		processorprofiles.WithLogs(func(context.Context, processor.Settings, component.Config, consumer.Logs) (processor.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		processorprofiles.WithMetrics(func(context.Context, processor.Settings, component.Config, consumer.Metrics) (processor.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		processorprofiles.WithProfiles(func(context.Context, processor.Settings, component.Config, consumerprofiles.Profiles) (processorprofiles.Profiles, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func newErrExporterFactory() exporter.Factory {
	return exporterprofiles.NewFactory(component.MustNewType("err"),
		func() component.Config { return &struct{}{} },
		exporterprofiles.WithTraces(func(context.Context, exporter.Settings, component.Config) (exporter.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		exporterprofiles.WithLogs(func(context.Context, exporter.Settings, component.Config) (exporter.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		exporterprofiles.WithMetrics(func(context.Context, exporter.Settings, component.Config) (exporter.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		exporterprofiles.WithProfiles(func(context.Context, exporter.Settings, component.Config) (exporterprofiles.Profiles, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func newErrConnectorFactory() connector.Factory {
	return connectorprofiles.NewFactory(component.MustNewType("err"), func() component.Config {
		return &struct{}{}
	},
		connectorprofiles.WithTracesToTraces(func(context.Context, connector.Settings, component.Config, consumer.Traces) (connector.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connectorprofiles.WithTracesToMetrics(func(context.Context, connector.Settings, component.Config, consumer.Metrics) (connector.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connectorprofiles.WithTracesToLogs(func(context.Context, connector.Settings, component.Config, consumer.Logs) (connector.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connectorprofiles.WithTracesToProfiles(func(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connector.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),

		connectorprofiles.WithMetricsToTraces(func(context.Context, connector.Settings, component.Config, consumer.Traces) (connector.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connectorprofiles.WithMetricsToMetrics(func(context.Context, connector.Settings, component.Config, consumer.Metrics) (connector.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connectorprofiles.WithMetricsToLogs(func(context.Context, connector.Settings, component.Config, consumer.Logs) (connector.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connectorprofiles.WithMetricsToProfiles(func(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connector.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),

		connectorprofiles.WithLogsToTraces(func(context.Context, connector.Settings, component.Config, consumer.Traces) (connector.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connectorprofiles.WithLogsToMetrics(func(context.Context, connector.Settings, component.Config, consumer.Metrics) (connector.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connectorprofiles.WithLogsToLogs(func(context.Context, connector.Settings, component.Config, consumer.Logs) (connector.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connectorprofiles.WithLogsToProfiles(func(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connector.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),

		connectorprofiles.WithProfilesToTraces(func(context.Context, connector.Settings, component.Config, consumer.Traces) (connectorprofiles.Profiles, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connectorprofiles.WithProfilesToMetrics(func(context.Context, connector.Settings, component.Config, consumer.Metrics) (connectorprofiles.Profiles, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connectorprofiles.WithProfilesToLogs(func(context.Context, connector.Settings, component.Config, consumer.Logs) (connectorprofiles.Profiles, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connectorprofiles.WithProfilesToProfiles(func(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connectorprofiles.Profiles, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
	)
}

type errComponent struct {
	consumertest.Consumer
}

func (e errComponent) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (e errComponent) Start(context.Context, component.Host) error {
	return errors.New("my error")
}

func (e errComponent) Shutdown(context.Context) error {
	return errors.New("my error")
}
