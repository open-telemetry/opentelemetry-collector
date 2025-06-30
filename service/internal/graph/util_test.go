// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"context"
	"errors"
	"hash/fnv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/xconnector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/xexporter"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/xprocessor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/xreceiver"
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

	// The graph identifies nodes by an int64 ID, but fnv gives us a uint64.
	// It is safe to cast because the meaning of the number is irrelevant.
	// We only care that each node has a unique 64 bit ID, which is unaltered by this cast.
	return int64(h.Sum64()) // #nosec G115
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
	receiversMap[xpipeline.SignalProfiles] = make(map[component.ID]component.Component)

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
	return xreceiver.NewFactory(component.MustNewType("err"),
		func() component.Config { return &struct{}{} },
		xreceiver.WithTraces(func(context.Context, receiver.Settings, component.Config, consumer.Traces) (receiver.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		xreceiver.WithLogs(func(context.Context, receiver.Settings, component.Config, consumer.Logs) (receiver.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		xreceiver.WithMetrics(func(context.Context, receiver.Settings, component.Config, consumer.Metrics) (receiver.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		xreceiver.WithProfiles(func(context.Context, receiver.Settings, component.Config, xconsumer.Profiles) (xreceiver.Profiles, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func newErrProcessorFactory() processor.Factory {
	return xprocessor.NewFactory(component.MustNewType("err"),
		func() component.Config { return &struct{}{} },
		xprocessor.WithTraces(func(context.Context, processor.Settings, component.Config, consumer.Traces) (processor.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		xprocessor.WithLogs(func(context.Context, processor.Settings, component.Config, consumer.Logs) (processor.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		xprocessor.WithMetrics(func(context.Context, processor.Settings, component.Config, consumer.Metrics) (processor.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		xprocessor.WithProfiles(func(context.Context, processor.Settings, component.Config, xconsumer.Profiles) (xprocessor.Profiles, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func newErrExporterFactory() exporter.Factory {
	return xexporter.NewFactory(component.MustNewType("err"),
		func() component.Config { return &struct{}{} },
		xexporter.WithTraces(func(context.Context, exporter.Settings, component.Config) (exporter.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		xexporter.WithLogs(func(context.Context, exporter.Settings, component.Config) (exporter.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		xexporter.WithMetrics(func(context.Context, exporter.Settings, component.Config) (exporter.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		xexporter.WithProfiles(func(context.Context, exporter.Settings, component.Config) (xexporter.Profiles, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func newErrConnectorFactory() connector.Factory {
	return xconnector.NewFactory(component.MustNewType("err"), func() component.Config {
		return &struct{}{}
	},
		xconnector.WithTracesToTraces(func(context.Context, connector.Settings, component.Config, consumer.Traces) (connector.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		xconnector.WithTracesToMetrics(func(context.Context, connector.Settings, component.Config, consumer.Metrics) (connector.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		xconnector.WithTracesToLogs(func(context.Context, connector.Settings, component.Config, consumer.Logs) (connector.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		xconnector.WithTracesToProfiles(func(context.Context, connector.Settings, component.Config, xconsumer.Profiles) (connector.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),

		xconnector.WithMetricsToTraces(func(context.Context, connector.Settings, component.Config, consumer.Traces) (connector.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		xconnector.WithMetricsToMetrics(func(context.Context, connector.Settings, component.Config, consumer.Metrics) (connector.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		xconnector.WithMetricsToLogs(func(context.Context, connector.Settings, component.Config, consumer.Logs) (connector.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		xconnector.WithMetricsToProfiles(func(context.Context, connector.Settings, component.Config, xconsumer.Profiles) (connector.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),

		xconnector.WithLogsToTraces(func(context.Context, connector.Settings, component.Config, consumer.Traces) (connector.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		xconnector.WithLogsToMetrics(func(context.Context, connector.Settings, component.Config, consumer.Metrics) (connector.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		xconnector.WithLogsToLogs(func(context.Context, connector.Settings, component.Config, consumer.Logs) (connector.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		xconnector.WithLogsToProfiles(func(context.Context, connector.Settings, component.Config, xconsumer.Profiles) (connector.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),

		xconnector.WithProfilesToTraces(func(context.Context, connector.Settings, component.Config, consumer.Traces) (xconnector.Profiles, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		xconnector.WithProfilesToMetrics(func(context.Context, connector.Settings, component.Config, consumer.Metrics) (xconnector.Profiles, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		xconnector.WithProfilesToLogs(func(context.Context, connector.Settings, component.Config, consumer.Logs) (xconnector.Profiles, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		xconnector.WithProfilesToProfiles(func(context.Context, connector.Settings, component.Config, xconsumer.Profiles) (xconnector.Profiles, error) {
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

func setObsConsumerGateForTest(t *testing.T, enabled bool) {
	initial := telemetry.NewPipelineTelemetryGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(telemetry.NewPipelineTelemetryGate.ID(), enabled))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(telemetry.NewPipelineTelemetryGate.ID(), initial))
	})
}
