// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gonum.org/v1/gonum/graph/simple"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentprofiles"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectorprofiles"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterprofiles"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorprofiles"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverprofiles"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/status"
	"go.opentelemetry.io/collector/service/internal/status/statustest"
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
	return int64(newNodeID(n.id.String()))
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

func TestGraphStartStop(t *testing.T) {
	testCases := []struct {
		name  string
		edges [][2]component.ID
	}{
		{
			name: "single",
			edges: [][2]component.ID{
				{component.MustNewIDWithName("r", "1"), component.MustNewIDWithName("p", "1")},
				{component.MustNewIDWithName("r", "2"), component.MustNewIDWithName("p", "1")},
				{component.MustNewIDWithName("p", "1"), component.MustNewIDWithName("p", "2")},
				{component.MustNewIDWithName("p", "2"), component.MustNewIDWithName("e", "1")},
				{component.MustNewIDWithName("p", "1"), component.MustNewIDWithName("e", "2")},
			},
		},
		{
			name: "multi",
			edges: [][2]component.ID{
				// Pipeline 1
				{component.MustNewIDWithName("r", "1"), component.MustNewIDWithName("p", "1")},
				{component.MustNewIDWithName("r", "2"), component.MustNewIDWithName("p", "1")},
				{component.MustNewIDWithName("p", "1"), component.MustNewIDWithName("p", "2")},
				{component.MustNewIDWithName("p", "2"), component.MustNewIDWithName("e", "1")},
				{component.MustNewIDWithName("p", "1"), component.MustNewIDWithName("e", "2")},

				// Pipeline 2, shares r1 and e2
				{component.MustNewIDWithName("r", "1"), component.MustNewIDWithName("p", "3")},
				{component.MustNewIDWithName("p", "3"), component.MustNewIDWithName("e", "2")},
			},
		},
		{
			name: "connected",
			edges: [][2]component.ID{
				// Pipeline 1
				{component.MustNewIDWithName("r", "1"), component.MustNewIDWithName("p", "1")},
				{component.MustNewIDWithName("r", "2"), component.MustNewIDWithName("p", "1")},
				{component.MustNewIDWithName("p", "1"), component.MustNewIDWithName("p", "2")},
				{component.MustNewIDWithName("p", "2"), component.MustNewIDWithName("e", "1")},
				{component.MustNewIDWithName("p", "1"), component.MustNewIDWithName("c", "1")},

				// Pipeline 2, shares r1 and c1
				{component.MustNewIDWithName("r", "1"), component.MustNewIDWithName("p", "3")},
				{component.MustNewIDWithName("p", "3"), component.MustNewIDWithName("c", "1")},

				// Pipeline 3, emits to e2 and c2
				{component.MustNewIDWithName("c", "1"), component.MustNewIDWithName("e", "2")},
				{component.MustNewIDWithName("c", "1"), component.MustNewIDWithName("c", "2")},

				// Pipeline 4, also emits to e2
				{component.MustNewIDWithName("c", "2"), component.MustNewIDWithName("e", "2")},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &contextWithOrder{
				Context: context.Background(),
				order:   map[component.ID]int{},
			}

			pg := &Graph{componentGraph: simple.NewDirectedGraph()}
			pg.telemetry = componenttest.NewNopTelemetrySettings()
			pg.instanceIDs = make(map[int64]*componentstatus.InstanceID)

			for _, edge := range tt.edges {
				f, t := &testNode{id: edge[0]}, &testNode{id: edge[1]}
				pg.instanceIDs[f.ID()] = &componentstatus.InstanceID{}
				pg.instanceIDs[t.ID()] = &componentstatus.InstanceID{}
				pg.componentGraph.SetEdge(simple.Edge{F: f, T: t})
			}

			require.NoError(t, pg.StartAll(ctx, &Host{Reporter: status.NewReporter(func(*componentstatus.InstanceID, *componentstatus.Event) {}, func(error) {})}))
			for _, edge := range tt.edges {
				assert.Greater(t, ctx.order[edge[0]], ctx.order[edge[1]])
			}

			ctx.order = map[component.ID]int{}
			require.NoError(t, pg.ShutdownAll(ctx, statustest.NewNopStatusReporter()))
			for _, edge := range tt.edges {
				assert.Less(t, ctx.order[edge[0]], ctx.order[edge[1]])
			}
		})
	}
}

func TestGraphStartStopCycle(t *testing.T) {
	pg := &Graph{componentGraph: simple.NewDirectedGraph()}

	r1 := &testNode{id: component.MustNewIDWithName("r", "1")}
	p1 := &testNode{id: component.MustNewIDWithName("p", "1")}
	c1 := &testNode{id: component.MustNewIDWithName("c", "1")}
	e1 := &testNode{id: component.MustNewIDWithName("e", "1")}

	pg.instanceIDs = map[int64]*componentstatus.InstanceID{
		r1.ID(): {},
		p1.ID(): {},
		c1.ID(): {},
		e1.ID(): {},
	}

	pg.componentGraph.SetEdge(simple.Edge{F: r1, T: p1})
	pg.componentGraph.SetEdge(simple.Edge{F: p1, T: c1})
	pg.componentGraph.SetEdge(simple.Edge{F: c1, T: e1})
	pg.componentGraph.SetEdge(simple.Edge{F: c1, T: p1}) // loop back

	err := pg.StartAll(context.Background(), &Host{Reporter: status.NewReporter(func(*componentstatus.InstanceID, *componentstatus.Event) {}, func(error) {})})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `topo: no topological ordering: cyclic components`)

	err = pg.ShutdownAll(context.Background(), statustest.NewNopStatusReporter())
	require.Error(t, err)
	assert.Contains(t, err.Error(), `topo: no topological ordering: cyclic components`)
}

func TestGraphStartStopComponentError(t *testing.T) {
	pg := &Graph{componentGraph: simple.NewDirectedGraph()}
	pg.telemetry = componenttest.NewNopTelemetrySettings()
	r1 := &testNode{
		id:       component.MustNewIDWithName("r", "1"),
		startErr: errors.New("foo"),
	}
	e1 := &testNode{
		id:          component.MustNewIDWithName("e", "1"),
		shutdownErr: errors.New("bar"),
	}
	pg.instanceIDs = map[int64]*componentstatus.InstanceID{
		r1.ID(): {},
		e1.ID(): {},
	}
	pg.componentGraph.SetEdge(simple.Edge{
		F: r1,
		T: e1,
	})
	require.EqualError(t, pg.StartAll(context.Background(), &Host{Reporter: status.NewReporter(func(*componentstatus.InstanceID, *componentstatus.Event) {}, func(error) {})}), "foo")
	assert.EqualError(t, pg.ShutdownAll(context.Background(), statustest.NewNopStatusReporter()), "bar")
}

func TestConnectorPipelinesGraph(t *testing.T) {
	tests := []struct {
		name                string
		pipelineConfigs     pipelines.ConfigWithPipelineID
		expectedPerExporter int // requires symmetry in Pipelines
	}{
		{
			name: "pipelines_simple.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("traces"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("metrics"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("logs"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("profiles"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_simple_mutate.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("traces"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("metrics"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("logs"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("profiles"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_simple_multi_proc.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("traces"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor"), component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("metrics"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor"), component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("logs"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor"), component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("profiles"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor"), component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_simple_no_proc.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("traces"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("metrics"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("logs"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("profiles"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_multi.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("traces"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver"), component.MustNewIDWithName("examplereceiver", "1")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate"), component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter"), component.MustNewIDWithName("exampleexporter", "1")},
				},
				pipeline.MustNewID("metrics"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver"), component.MustNewIDWithName("examplereceiver", "1")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate"), component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter"), component.MustNewIDWithName("exampleexporter", "1")},
				},
				pipeline.MustNewID("logs"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver"), component.MustNewIDWithName("examplereceiver", "1")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate"), component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter"), component.MustNewIDWithName("exampleexporter", "1")},
				},
				pipeline.MustNewID("profiles"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver"), component.MustNewIDWithName("examplereceiver", "1")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate"), component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter"), component.MustNewIDWithName("exampleexporter", "1")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_multi_no_proc.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("traces"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver"), component.MustNewIDWithName("examplereceiver", "1")},
					Exporters: []component.ID{component.MustNewID("exampleexporter"), component.MustNewIDWithName("exampleexporter", "1")},
				},
				pipeline.MustNewID("metrics"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver"), component.MustNewIDWithName("examplereceiver", "1")},
					Exporters: []component.ID{component.MustNewID("exampleexporter"), component.MustNewIDWithName("exampleexporter", "1")},
				},
				pipeline.MustNewID("logs"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver"), component.MustNewIDWithName("examplereceiver", "1")},
					Exporters: []component.ID{component.MustNewID("exampleexporter"), component.MustNewIDWithName("exampleexporter", "1")},
				},
				pipeline.MustNewID("profiles"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver"), component.MustNewIDWithName("examplereceiver", "1")},
					Exporters: []component.ID{component.MustNewID("exampleexporter"), component.MustNewIDWithName("exampleexporter", "1")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "multi_pipeline_receivers_and_exporters.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("traces"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewIDWithName("traces", "1"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("metrics"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewIDWithName("metrics", "1"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("logs"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewIDWithName("logs", "1"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("profiles"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewIDWithName("profiles", "1"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_conn_simple_traces.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("traces", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.MustNewIDWithName("traces", "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_simple_metrics.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("metrics", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.MustNewIDWithName("metrics", "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_simple_logs.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("logs", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.MustNewIDWithName("logs", "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_simple_profiles.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("profiles", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.MustNewIDWithName("profiles", "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_fork_merge_traces.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("traces", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.MustNewIDWithName("traces", "type0"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
				},
				pipeline.MustNewIDWithName("traces", "type1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
				},
				pipeline.MustNewIDWithName("traces", "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_conn_fork_merge_metrics.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("metrics", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.MustNewIDWithName("metrics", "type0"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
				},
				pipeline.MustNewIDWithName("metrics", "type1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
				},
				pipeline.MustNewIDWithName("metrics", "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_conn_fork_merge_logs.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("logs", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.MustNewIDWithName("logs", "type0"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
				},
				pipeline.MustNewIDWithName("logs", "type1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
				},
				pipeline.MustNewIDWithName("logs", "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_conn_fork_merge_profiles.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("profiles", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.MustNewIDWithName("profiles", "type0"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
				},
				pipeline.MustNewIDWithName("profiles", "type1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
				},
				pipeline.MustNewIDWithName("profiles", "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_conn_translate_from_traces.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("traces"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleconnector")},
				},
				pipeline.MustNewID("metrics"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("logs"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("profiles"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_translate_from_metrics.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("metrics"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleconnector")},
				},
				pipeline.MustNewID("traces"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("logs"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("profiles"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_translate_from_logs.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("logs"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleconnector")},
				},
				pipeline.MustNewID("traces"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("metrics"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("profiles"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_translate_from_profiles.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("traces"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleconnector")},
				},
				pipeline.MustNewID("metrics"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("logs"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewID("profiles"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_matrix_immutable.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("traces", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleconnector")},
				},
				pipeline.MustNewIDWithName("metrics", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleconnector")},
				},
				pipeline.MustNewIDWithName("logs", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleconnector")},
				},
				pipeline.MustNewIDWithName("profiles", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleconnector")},
				},
				pipeline.MustNewIDWithName("traces", "out"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewIDWithName("metrics", "out"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewIDWithName("logs", "out"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewIDWithName("profiles", "out"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 4,
		},
		{
			name: "pipelines_conn_matrix_mutable.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("traces", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.MustNewIDWithName("metrics", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.MustNewIDWithName("logs", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.MustNewIDWithName("profiles", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.MustNewIDWithName("traces", "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewIDWithName("metrics", "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewIDWithName("logs", "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewIDWithName("profiles", "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 4,
		},
		{
			name: "pipelines_conn_lanes.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("traces", "in"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("mockforward")},
				},
				pipeline.MustNewIDWithName("traces", "out"): {
					Receivers: []component.ID{component.MustNewID("mockforward")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewIDWithName("metrics", "in"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("mockforward")},
				},
				pipeline.MustNewIDWithName("metrics", "out"): {
					Receivers: []component.ID{component.MustNewID("mockforward")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewIDWithName("logs", "in"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("mockforward")},
				},
				pipeline.MustNewIDWithName("logs", "out"): {
					Receivers: []component.ID{component.MustNewID("mockforward")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewIDWithName("profiles", "in"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("mockforward")},
				},
				pipeline.MustNewIDWithName("profiles", "out"): {
					Receivers: []component.ID{component.MustNewID("mockforward")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_mutate_traces.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("traces", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.MustNewIDWithName("traces", "out0"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewIDWithName("traces", "middle"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "mutate")},
				},
				pipeline.MustNewIDWithName("traces", "out1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_conn_mutate_metrics.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("metrics", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.MustNewIDWithName("metrics", "out0"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewIDWithName("metrics", "middle"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "mutate")},
				},
				pipeline.MustNewIDWithName("metrics", "out1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_conn_mutate_logs.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("logs", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.MustNewIDWithName("logs", "out0"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewIDWithName("logs", "middle"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "mutate")},
				},
				pipeline.MustNewIDWithName("logs", "out1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_conn_mutate_profiles.yaml",
			pipelineConfigs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("profiles", "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.MustNewIDWithName("profiles", "out0"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.MustNewIDWithName("profiles", "middle"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "mutate")},
				},
				pipeline.MustNewIDWithName("profiles", "out1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build the pipeline
			set := Settings{
				Telemetry: componenttest.NewNopTelemetrySettings(),
				BuildInfo: component.NewDefaultBuildInfo(),
				ReceiverBuilder: builders.NewReceiver(
					map[component.ID]component.Config{
						component.MustNewID("examplereceiver"):              testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
						component.MustNewIDWithName("examplereceiver", "1"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
					},
					map[component.Type]receiver.Factory{
						testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
					},
				),
				ProcessorBuilder: builders.NewProcessor(
					map[component.ID]component.Config{
						component.MustNewID("exampleprocessor"):                   testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
						component.MustNewIDWithName("exampleprocessor", "mutate"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
					},
					map[component.Type]processor.Factory{
						testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
					},
				),
				ExporterBuilder: builders.NewExporter(
					map[component.ID]component.Config{
						component.MustNewID("exampleexporter"):              testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
						component.MustNewIDWithName("exampleexporter", "1"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
					},
					map[component.Type]exporter.Factory{
						testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
					},
				),
				ConnectorBuilder: builders.NewConnector(
					map[component.ID]component.Config{
						component.MustNewID("exampleconnector"):                           testcomponents.ExampleConnectorFactory.CreateDefaultConfig(),
						component.MustNewIDWithName("exampleconnector", "merge"):          testcomponents.ExampleConnectorFactory.CreateDefaultConfig(),
						component.MustNewIDWithName("exampleconnector", "mutate"):         testcomponents.ExampleConnectorFactory.CreateDefaultConfig(),
						component.MustNewIDWithName("exampleconnector", "inherit_mutate"): testcomponents.ExampleConnectorFactory.CreateDefaultConfig(),
						component.MustNewID("mockforward"):                                testcomponents.MockForwardConnectorFactory.CreateDefaultConfig(),
					},
					map[component.Type]connector.Factory{
						testcomponents.ExampleConnectorFactory.Type():     testcomponents.ExampleConnectorFactory,
						testcomponents.MockForwardConnectorFactory.Type(): testcomponents.MockForwardConnectorFactory,
					},
				),
				PipelineConfigs: tt.pipelineConfigs,
			}

			pg, err := Build(context.Background(), set)
			require.NoError(t, err)

			assert.Equal(t, len(tt.pipelineConfigs), len(pg.pipelines))

			require.NoError(t, pg.StartAll(context.Background(), &Host{Reporter: status.NewReporter(func(*componentstatus.InstanceID, *componentstatus.Event) {}, func(error) {})}))

			mutatingPipelines := make(map[pipeline.ID]bool, len(tt.pipelineConfigs))

			// Check each pipeline individually, ensuring that all components are started
			// and that they have observed no signals yet.
			for pipelineID, pipelineCfg := range tt.pipelineConfigs {
				pipeline, ok := pg.pipelines[pipelineID]
				require.True(t, ok, "expected to find pipeline: %s", pipelineID.String())

				// Determine independently if the capabilities node should report MutateData as true
				var expectMutatesData bool
				for _, expr := range pipelineCfg.Exporters {
					if strings.Contains(expr.Name(), "mutate") {
						expectMutatesData = true
					}
				}
				for _, proc := range pipelineCfg.Processors {
					if proc.Name() == "mutate" {
						expectMutatesData = true
					}
				}
				assert.Equal(t, expectMutatesData, pipeline.capabilitiesNode.getConsumer().Capabilities().MutatesData)
				mutatingPipelines[pipelineID] = expectMutatesData

				expectedReceivers, expectedExporters := expectedInstances(tt.pipelineConfigs, pipelineID)
				require.Len(t, pipeline.receivers, expectedReceivers)
				require.Equal(t, len(pipelineCfg.Processors), len(pipeline.processors))
				require.Len(t, pipeline.exporters, expectedExporters)

				for _, n := range pipeline.exporters {
					switch c := n.(type) {
					case *exporterNode:
						e := c.Component.(*testcomponents.ExampleExporter)
						require.True(t, e.Started())
						require.Empty(t, e.Traces)
						require.Empty(t, e.Metrics)
						require.Empty(t, e.Logs)
						require.Empty(t, e.Profiles)
					case *connectorNode:
						// connector needs to be unwrapped to access component as ExampleConnector
						switch ct := c.Component.(type) {
						case connector.Traces:
							require.True(t, ct.(*testcomponents.ExampleConnector).Started())
						case connector.Metrics:
							require.True(t, ct.(*testcomponents.ExampleConnector).Started())
						case connector.Logs:
							require.True(t, ct.(*testcomponents.ExampleConnector).Started())
						case connectorprofiles.Profiles:
							require.True(t, ct.(*testcomponents.ExampleConnector).Started())
						}
					default:
						require.Fail(t, fmt.Sprintf("unexpected type %T", c))
					}
				}

				for _, n := range pipeline.processors {
					require.True(t, n.Component.(*testcomponents.ExampleProcessor).Started())
				}

				for _, n := range pipeline.receivers {
					switch c := n.(type) {
					case *receiverNode:
						require.True(t, c.Component.(*testcomponents.ExampleReceiver).Started())
					case *connectorNode:
						// connector needs to be unwrapped to access component as ExampleConnector
						switch ct := c.Component.(type) {
						case connector.Traces:
							require.True(t, ct.(*testcomponents.ExampleConnector).Started())
						case connector.Metrics:
							require.True(t, ct.(*testcomponents.ExampleConnector).Started())
						case connector.Logs:
							require.True(t, ct.(*testcomponents.ExampleConnector).Started())
						case connectorprofiles.Profiles:
							require.True(t, ct.(*testcomponents.ExampleConnector).Started())
						}
					default:
						require.Fail(t, fmt.Sprintf("unexpected type %T", c))
					}
				}
			}

			// Check that Connectors are correctly inheriting mutability from downstream Pipelines
			for expPipelineID, expPipeline := range pg.pipelines {
				for _, exp := range expPipeline.exporters {
					expConn, ok := exp.(*connectorNode)
					if !ok {
						continue
					}
					if expConn.getConsumer().Capabilities().MutatesData {
						continue
					}
					// find all the Pipelines of the same type where this connector is a receiver
					var inheritMutatesData bool
					for recPipelineID, recPipeline := range pg.pipelines {
						if recPipelineID == expPipelineID || recPipelineID.Signal() != expPipelineID.Signal() {
							continue
						}
						for _, rec := range recPipeline.receivers {
							recConn, ok := rec.(*connectorNode)
							if !ok || recConn.ID() != expConn.ID() {
								continue
							}
							inheritMutatesData = inheritMutatesData || mutatingPipelines[recPipelineID]
						}
					}
					assert.Equal(t, inheritMutatesData, expConn.getConsumer().Capabilities().MutatesData)
				}
			}

			// Push data into the Pipelines. The list of Receivers is retrieved directly from the overall
			// component graph because we do not want to duplicate signal inputs to Receivers that are
			// shared between Pipelines. The `allReceivers` function also excludes Connectors, which we do
			// not want to directly inject with signals.
			allReceivers := pg.getReceivers()
			for _, c := range allReceivers[pipeline.SignalTraces] {
				tracesReceiver := c.(*testcomponents.ExampleReceiver)
				require.NoError(t, tracesReceiver.ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
			}
			for _, c := range allReceivers[pipeline.SignalMetrics] {
				metricsReceiver := c.(*testcomponents.ExampleReceiver)
				require.NoError(t, metricsReceiver.ConsumeMetrics(context.Background(), testdata.GenerateMetrics(1)))
			}
			for _, c := range allReceivers[pipeline.SignalLogs] {
				logsReceiver := c.(*testcomponents.ExampleReceiver)
				require.NoError(t, logsReceiver.ConsumeLogs(context.Background(), testdata.GenerateLogs(1)))
			}
			for _, c := range allReceivers[componentprofiles.SignalProfiles] {
				profilesReceiver := c.(*testcomponents.ExampleReceiver)
				require.NoError(t, profilesReceiver.ConsumeProfiles(context.Background(), testdata.GenerateProfiles(1)))
			}

			// Shut down the entire component graph
			require.NoError(t, pg.ShutdownAll(context.Background(), statustest.NewNopStatusReporter()))

			// Check each pipeline individually, ensuring that all components are stopped.
			for pipelineID := range tt.pipelineConfigs {
				pipeline, ok := pg.pipelines[pipelineID]
				require.True(t, ok, "expected to find pipeline: %s", pipelineID.String())

				for _, n := range pipeline.receivers {
					switch c := n.(type) {
					case *receiverNode:
						require.True(t, c.Component.(*testcomponents.ExampleReceiver).Stopped())
					case *connectorNode:
						// connector needs to be unwrapped to access component as ExampleConnector
						switch ct := c.Component.(type) {
						case connector.Traces:
							require.True(t, ct.(*testcomponents.ExampleConnector).Stopped())
						case connector.Metrics:
							require.True(t, ct.(*testcomponents.ExampleConnector).Stopped())
						case connector.Logs:
							require.True(t, ct.(*testcomponents.ExampleConnector).Stopped())
						case connectorprofiles.Profiles:
							require.True(t, ct.(*testcomponents.ExampleConnector).Stopped())
						}
					default:
						require.Fail(t, fmt.Sprintf("unexpected type %T", c))
					}
				}

				for _, n := range pipeline.processors {
					require.True(t, n.Component.(*testcomponents.ExampleProcessor).Stopped())
				}

				for _, n := range pipeline.exporters {
					switch c := n.(type) {
					case *exporterNode:
						e := c.Component.(*testcomponents.ExampleExporter)
						require.True(t, e.Stopped())
					case *connectorNode:
						// connector needs to be unwrapped to access component as ExampleConnector
						switch ct := c.Component.(type) {
						case connector.Traces:
							require.True(t, ct.(*testcomponents.ExampleConnector).Stopped())
						case connector.Metrics:
							require.True(t, ct.(*testcomponents.ExampleConnector).Stopped())
						case connector.Logs:
							require.True(t, ct.(*testcomponents.ExampleConnector).Stopped())
						case connectorprofiles.Profiles:
							require.True(t, ct.(*testcomponents.ExampleConnector).Stopped())
						}
					default:
						require.Fail(t, fmt.Sprintf("unexpected type %T", c))
					}
				}
			}

			// Get the list of Exporters directly from the overall component graph. Like Receivers,
			// exclude Connectors and validate each exporter once regardless of sharing between Pipelines.
			allExporters := pg.GetExporters()
			for _, e := range allExporters[pipeline.SignalTraces] {
				tracesExporter := e.(*testcomponents.ExampleExporter)
				assert.Len(t, tracesExporter.Traces, tt.expectedPerExporter)
				expectedMutable := testdata.GenerateTraces(1)
				expectedReadOnly := testdata.GenerateTraces(1)
				expectedReadOnly.MarkReadOnly()
				for i := 0; i < tt.expectedPerExporter; i++ {
					if tracesExporter.Traces[i].IsReadOnly() {
						assert.EqualValues(t, expectedReadOnly, tracesExporter.Traces[i])
					} else {
						assert.EqualValues(t, expectedMutable, tracesExporter.Traces[i])
					}
				}
			}
			for _, e := range allExporters[pipeline.SignalMetrics] {
				metricsExporter := e.(*testcomponents.ExampleExporter)
				assert.Len(t, metricsExporter.Metrics, tt.expectedPerExporter)
				expectedMutable := testdata.GenerateMetrics(1)
				expectedReadOnly := testdata.GenerateMetrics(1)
				expectedReadOnly.MarkReadOnly()
				for i := 0; i < tt.expectedPerExporter; i++ {
					if metricsExporter.Metrics[i].IsReadOnly() {
						assert.EqualValues(t, expectedReadOnly, metricsExporter.Metrics[i])
					} else {
						assert.EqualValues(t, expectedMutable, metricsExporter.Metrics[i])
					}
				}
			}
			for _, e := range allExporters[pipeline.SignalLogs] {
				logsExporter := e.(*testcomponents.ExampleExporter)
				assert.Len(t, logsExporter.Logs, tt.expectedPerExporter)
				expectedMutable := testdata.GenerateLogs(1)
				expectedReadOnly := testdata.GenerateLogs(1)
				expectedReadOnly.MarkReadOnly()
				for i := 0; i < tt.expectedPerExporter; i++ {
					if logsExporter.Logs[i].IsReadOnly() {
						assert.EqualValues(t, expectedReadOnly, logsExporter.Logs[i])
					} else {
						assert.EqualValues(t, expectedMutable, logsExporter.Logs[i])
					}
				}
			}
			for _, e := range allExporters[componentprofiles.SignalProfiles] {
				profilesExporter := e.(*testcomponents.ExampleExporter)
				assert.Len(t, profilesExporter.Profiles, tt.expectedPerExporter)
				expectedMutable := testdata.GenerateProfiles(1)
				expectedReadOnly := testdata.GenerateProfiles(1)
				expectedReadOnly.MarkReadOnly()
				for i := 0; i < tt.expectedPerExporter; i++ {
					if profilesExporter.Profiles[i].IsReadOnly() {
						assert.EqualValues(t, expectedReadOnly, profilesExporter.Profiles[i])
					} else {
						assert.EqualValues(t, expectedMutable, profilesExporter.Profiles[i])
					}
				}
			}
		})
	}
}

func TestConnectorRouter(t *testing.T) {
	rcvrID := component.MustNewID("examplereceiver")
	routeTracesID := component.MustNewIDWithName("examplerouter", "traces")
	routeMetricsID := component.MustNewIDWithName("examplerouter", "metrics")
	routeLogsID := component.MustNewIDWithName("examplerouter", "logs")
	routeProfilesID := component.MustNewIDWithName("examplerouter", "profiles")
	expRightID := component.MustNewIDWithName("exampleexporter", "right")
	expLeftID := component.MustNewIDWithName("exampleexporter", "left")

	tracesInID := pipeline.MustNewIDWithName("traces", "in")
	tracesRightID := pipeline.MustNewIDWithName("traces", "right")
	tracesLeftID := pipeline.MustNewIDWithName("traces", "left")

	metricsInID := pipeline.MustNewIDWithName("metrics", "in")
	metricsRightID := pipeline.MustNewIDWithName("metrics", "right")
	metricsLeftID := pipeline.MustNewIDWithName("metrics", "left")

	logsInID := pipeline.MustNewIDWithName("logs", "in")
	logsRightID := pipeline.MustNewIDWithName("logs", "right")
	logsLeftID := pipeline.MustNewIDWithName("logs", "left")

	profilesInID := pipeline.MustNewIDWithName("profiles", "in")
	profilesRightID := pipeline.MustNewIDWithName("profiles", "right")
	profilesLeftID := pipeline.MustNewIDWithName("profiles", "left")

	ctx := context.Background()
	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			map[component.ID]component.Config{
				rcvrID: testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
			},
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			map[component.ID]component.Config{
				expRightID: testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
				expLeftID:  testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
			},
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(
			map[component.ID]component.Config{
				routeTracesID: testcomponents.ExampleRouterConfig{
					Traces: &testcomponents.LeftRightConfig{
						Right: tracesRightID,
						Left:  tracesLeftID,
					},
				},
				routeMetricsID: testcomponents.ExampleRouterConfig{
					Metrics: &testcomponents.LeftRightConfig{
						Right: metricsRightID,
						Left:  metricsLeftID,
					},
				},
				routeLogsID: testcomponents.ExampleRouterConfig{
					Logs: &testcomponents.LeftRightConfig{
						Right: logsRightID,
						Left:  logsLeftID,
					},
				},
				routeProfilesID: testcomponents.ExampleRouterConfig{
					Profiles: &testcomponents.LeftRightConfig{
						Right: profilesRightID,
						Left:  profilesLeftID,
					},
				},
			},
			map[component.Type]connector.Factory{
				testcomponents.ExampleRouterFactory.Type(): testcomponents.ExampleRouterFactory,
			},
		),
		PipelineConfigs: pipelines.ConfigWithPipelineID{
			tracesInID: {
				Receivers: []component.ID{rcvrID},
				Exporters: []component.ID{routeTracesID},
			},
			tracesRightID: {
				Receivers: []component.ID{routeTracesID},
				Exporters: []component.ID{expRightID},
			},
			tracesLeftID: {
				Receivers: []component.ID{routeTracesID},
				Exporters: []component.ID{expLeftID},
			},
			metricsInID: {
				Receivers: []component.ID{rcvrID},
				Exporters: []component.ID{routeMetricsID},
			},
			metricsRightID: {
				Receivers: []component.ID{routeMetricsID},
				Exporters: []component.ID{expRightID},
			},
			metricsLeftID: {
				Receivers: []component.ID{routeMetricsID},
				Exporters: []component.ID{expLeftID},
			},
			logsInID: {
				Receivers: []component.ID{rcvrID},
				Exporters: []component.ID{routeLogsID},
			},
			logsRightID: {
				Receivers: []component.ID{routeLogsID},
				Exporters: []component.ID{expRightID},
			},
			logsLeftID: {
				Receivers: []component.ID{routeLogsID},
				Exporters: []component.ID{expLeftID},
			},
			profilesInID: {
				Receivers: []component.ID{rcvrID},
				Exporters: []component.ID{routeProfilesID},
			},
			profilesRightID: {
				Receivers: []component.ID{routeProfilesID},
				Exporters: []component.ID{expRightID},
			},
			profilesLeftID: {
				Receivers: []component.ID{routeProfilesID},
				Exporters: []component.ID{expLeftID},
			},
		},
	}

	pg, err := Build(ctx, set)
	require.NoError(t, err)

	allReceivers := pg.getReceivers()
	allExporters := pg.GetExporters()

	assert.Equal(t, len(set.PipelineConfigs), len(pg.pipelines))

	// Get a handle for the traces receiver and both Exporters
	tracesReceiver := allReceivers[pipeline.SignalTraces][rcvrID].(*testcomponents.ExampleReceiver)
	tracesRight := allExporters[pipeline.SignalTraces][expRightID].(*testcomponents.ExampleExporter)
	tracesLeft := allExporters[pipeline.SignalTraces][expLeftID].(*testcomponents.ExampleExporter)

	// Consume 1, validate it went right
	require.NoError(t, tracesReceiver.ConsumeTraces(ctx, testdata.GenerateTraces(1)))
	assert.Len(t, tracesRight.Traces, 1)
	assert.Empty(t, tracesLeft.Traces)

	// Consume 1, validate it went left
	require.NoError(t, tracesReceiver.ConsumeTraces(ctx, testdata.GenerateTraces(1)))
	assert.Len(t, tracesRight.Traces, 1)
	assert.Len(t, tracesLeft.Traces, 1)

	// Consume 3, validate 2 went right, 1 went left
	assert.NoError(t, tracesReceiver.ConsumeTraces(ctx, testdata.GenerateTraces(1)))
	assert.NoError(t, tracesReceiver.ConsumeTraces(ctx, testdata.GenerateTraces(1)))
	assert.NoError(t, tracesReceiver.ConsumeTraces(ctx, testdata.GenerateTraces(1)))
	assert.Len(t, tracesRight.Traces, 3)
	assert.Len(t, tracesLeft.Traces, 2)

	// Get a handle for the metrics receiver and both Exporters
	metricsReceiver := allReceivers[pipeline.SignalMetrics][rcvrID].(*testcomponents.ExampleReceiver)
	metricsRight := allExporters[pipeline.SignalMetrics][expRightID].(*testcomponents.ExampleExporter)
	metricsLeft := allExporters[pipeline.SignalMetrics][expLeftID].(*testcomponents.ExampleExporter)

	// Consume 1, validate it went right
	require.NoError(t, metricsReceiver.ConsumeMetrics(ctx, testdata.GenerateMetrics(1)))
	assert.Len(t, metricsRight.Metrics, 1)
	assert.Empty(t, metricsLeft.Metrics)

	// Consume 1, validate it went left
	require.NoError(t, metricsReceiver.ConsumeMetrics(ctx, testdata.GenerateMetrics(1)))
	assert.Len(t, metricsRight.Metrics, 1)
	assert.Len(t, metricsLeft.Metrics, 1)

	// Consume 3, validate 2 went right, 1 went left
	assert.NoError(t, metricsReceiver.ConsumeMetrics(ctx, testdata.GenerateMetrics(1)))
	assert.NoError(t, metricsReceiver.ConsumeMetrics(ctx, testdata.GenerateMetrics(1)))
	assert.NoError(t, metricsReceiver.ConsumeMetrics(ctx, testdata.GenerateMetrics(1)))
	assert.Len(t, metricsRight.Metrics, 3)
	assert.Len(t, metricsLeft.Metrics, 2)

	// Get a handle for the logs receiver and both Exporters
	logsReceiver := allReceivers[pipeline.SignalLogs][rcvrID].(*testcomponents.ExampleReceiver)
	logsRight := allExporters[pipeline.SignalLogs][expRightID].(*testcomponents.ExampleExporter)
	logsLeft := allExporters[pipeline.SignalLogs][expLeftID].(*testcomponents.ExampleExporter)

	// Consume 1, validate it went right
	require.NoError(t, logsReceiver.ConsumeLogs(ctx, testdata.GenerateLogs(1)))
	assert.Len(t, logsRight.Logs, 1)
	assert.Empty(t, logsLeft.Logs)

	// Consume 1, validate it went left
	require.NoError(t, logsReceiver.ConsumeLogs(ctx, testdata.GenerateLogs(1)))
	assert.Len(t, logsRight.Logs, 1)
	assert.Len(t, logsLeft.Logs, 1)

	// Consume 3, validate 2 went right, 1 went left
	assert.NoError(t, logsReceiver.ConsumeLogs(ctx, testdata.GenerateLogs(1)))
	assert.NoError(t, logsReceiver.ConsumeLogs(ctx, testdata.GenerateLogs(1)))
	assert.NoError(t, logsReceiver.ConsumeLogs(ctx, testdata.GenerateLogs(1)))
	assert.Len(t, logsRight.Logs, 3)
	assert.Len(t, logsLeft.Logs, 2)

	// Get a handle for the profiles receiver and both Exporters
	profilesReceiver := allReceivers[componentprofiles.SignalProfiles][rcvrID].(*testcomponents.ExampleReceiver)
	profilesRight := allExporters[componentprofiles.SignalProfiles][expRightID].(*testcomponents.ExampleExporter)
	profilesLeft := allExporters[componentprofiles.SignalProfiles][expLeftID].(*testcomponents.ExampleExporter)

	// Consume 1, validate it went right
	require.NoError(t, profilesReceiver.ConsumeProfiles(ctx, testdata.GenerateProfiles(1)))
	assert.Len(t, profilesRight.Profiles, 1)
	assert.Empty(t, profilesLeft.Profiles)

	// Consume 1, validate it went left
	require.NoError(t, profilesReceiver.ConsumeProfiles(ctx, testdata.GenerateProfiles(1)))
	assert.Len(t, profilesRight.Profiles, 1)
	assert.Len(t, profilesLeft.Profiles, 1)

	// Consume 3, validate 2 went right, 1 went left
	assert.NoError(t, profilesReceiver.ConsumeProfiles(ctx, testdata.GenerateProfiles(1)))
	assert.NoError(t, profilesReceiver.ConsumeProfiles(ctx, testdata.GenerateProfiles(1)))
	assert.NoError(t, profilesReceiver.ConsumeProfiles(ctx, testdata.GenerateProfiles(1)))
	assert.Len(t, profilesRight.Profiles, 3)
	assert.Len(t, profilesLeft.Profiles, 2)

}

func TestGraphBuildErrors(t *testing.T) {
	nopReceiverFactory := receivertest.NewNopFactory()
	nopProcessorFactory := processortest.NewNopFactory()
	nopExporterFactory := exportertest.NewNopFactory()
	nopConnectorFactory := connectortest.NewNopFactory()
	mfConnectorFactory := testcomponents.MockForwardConnectorFactory
	badReceiverFactory := newBadReceiverFactory()
	badProcessorFactory := newBadProcessorFactory()
	badExporterFactory := newBadExporterFactory()
	badConnectorFactory := newBadConnectorFactory()

	tests := []struct {
		name          string
		receiverCfgs  map[component.ID]component.Config
		processorCfgs map[component.ID]component.Config
		exporterCfgs  map[component.ID]component.Config
		connectorCfgs map[component.ID]component.Config
		pipelineCfgs  pipelines.ConfigWithPipelineID
		expected      string
	}{
		{
			name: "not_supported_exporter_logs",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): badExporterFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("logs"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
			},
			expected: "failed to create \"bf\" exporter for data type \"logs\": telemetry type is not supported",
		},
		{
			name: "not_supported_exporter_metrics",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): badExporterFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("metrics"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
			},
			expected: "failed to create \"bf\" exporter for data type \"metrics\": telemetry type is not supported",
		},
		{
			name: "not_supported_exporter_traces",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): badExporterFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("traces"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
			},
			expected: "failed to create \"bf\" exporter for data type \"traces\": telemetry type is not supported",
		},
		{
			name: "not_supported_exporter_profiles",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): badExporterFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("profiles"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
			},
			expected: "failed to create \"bf\" exporter for data type \"profiles\": telemetry type is not supported",
		},
		{
			name: "not_supported_processor_logs",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			processorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): badProcessorFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("logs"): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("bf")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "failed to create \"bf\" processor, in pipeline \"logs\": telemetry type is not supported",
		},
		{
			name: "not_supported_processor_metrics",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			processorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): badProcessorFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("metrics"): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("bf")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "failed to create \"bf\" processor, in pipeline \"metrics\": telemetry type is not supported",
		},
		{
			name: "not_supported_processor_traces",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			processorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): badProcessorFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("traces"): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("bf")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "failed to create \"bf\" processor, in pipeline \"traces\": telemetry type is not supported",
		},
		{
			name: "not_supported_processor_profiles",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			processorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): badProcessorFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("profiles"): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("bf")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "failed to create \"bf\" processor, in pipeline \"profiles\": telemetry type is not supported",
		},
		{
			name: "not_supported_receiver_logs",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): badReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("logs"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "failed to create \"bf\" receiver for data type \"logs\": telemetry type is not supported",
		},
		{
			name: "not_supported_receiver_metrics",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): badReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("metrics"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "failed to create \"bf\" receiver for data type \"metrics\": telemetry type is not supported",
		},
		{
			name: "not_supported_receiver_traces",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): badReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("traces"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "failed to create \"bf\" receiver for data type \"traces\": telemetry type is not supported",
		},
		{
			name: "not_supported_receiver_profiles",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): badReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("profiles"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "failed to create \"bf\" receiver for data type \"profiles\": telemetry type is not supported",
		},
		{
			name: "not_supported_connector_traces_traces.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("traces", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.MustNewIDWithName("traces", "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in traces pipeline but not used in any supported receiver pipeline",
		},
		{
			name: "not_supported_connector_traces_metrics.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("traces", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.MustNewIDWithName("metrics", "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in traces pipeline but not used in any supported receiver pipeline",
		},
		{
			name: "not_supported_connector_traces_logs.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("traces", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.MustNewIDWithName("logs", "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in traces pipeline but not used in any supported receiver pipeline",
		},
		{
			name: "not_supported_connector_traces_profiles.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("traces", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.MustNewIDWithName("profiles", "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in traces pipeline but not used in any supported receiver pipeline",
		},
		{
			name: "not_supported_connector_metrics_traces.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("metrics", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.MustNewIDWithName("traces", "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in metrics pipeline but not used in any supported receiver pipeline",
		},
		{
			name: "not_supported_connector_metrics_metrics.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("metrics", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.MustNewIDWithName("metrics", "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in metrics pipeline but not used in any supported receiver pipeline",
		},
		{
			name: "not_supported_connector_metrics_logs.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("metrics", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.MustNewIDWithName("logs", "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in metrics pipeline but not used in any supported receiver pipeline",
		},
		{
			name: "not_supported_connector_metrics_profiles.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("metrics", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.MustNewIDWithName("profiles", "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in metrics pipeline but not used in any supported receiver pipeline",
		},
		{
			name: "not_supported_connector_logs_traces.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("logs", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.MustNewIDWithName("traces", "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in logs pipeline but not used in any supported receiver pipeline",
		},
		{
			name: "not_supported_connector_logs_metrics.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("logs", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.MustNewIDWithName("metrics", "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in logs pipeline but not used in any supported receiver pipeline",
		},
		{
			name: "not_supported_connector_logs_logs.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("logs", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.MustNewIDWithName("logs", "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in logs pipeline but not used in any supported receiver pipeline",
		},
		{
			name: "not_supported_connector_logs_profiles.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("logs", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.MustNewIDWithName("profiles", "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in logs pipeline but not used in any supported receiver pipeline",
		},
		{
			name: "not_supported_connector_profiles_traces.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("profiles", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.MustNewIDWithName("traces", "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in profiles pipeline but not used in any supported receiver pipeline",
		},
		{
			name: "not_supported_connector_profiles_metrics.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("profiles", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.MustNewIDWithName("metrics", "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in profiles pipeline but not used in any supported receiver pipeline",
		},
		{
			name: "not_supported_connector_profiles_logs.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("profiles", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.MustNewIDWithName("logs", "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in profiles pipeline but not used in any supported receiver pipeline",
		},
		{
			name: "not_supported_connector_profiles_profiles.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("profiles", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.MustNewIDWithName("profiles", "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in profiles pipeline but not used in any supported receiver pipeline",
		},
		{
			name: "orphaned-connector-use-as-exporter",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewIDWithName("nop", "conn"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("metrics", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewIDWithName("nop", "conn")},
				},
			},
			expected: `connector "nop/conn" used as exporter in metrics pipeline but not used in any supported receiver pipeline`,
		},
		{
			name: "orphaned-connector-use-as-receiver",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewIDWithName("nop", "conn"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("traces", "out"): {
					Receivers: []component.ID{component.MustNewIDWithName("nop", "conn")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: `connector "nop/conn" used as receiver in traces pipeline but not used in any supported exporter pipeline`,
		},
		{
			name: "partially-orphaned-connector-use-as-exporter",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewID("mockforward"): mfConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("traces", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("mockforward")},
				},
				pipeline.MustNewIDWithName("traces", "out"): {
					Receivers: []component.ID{component.MustNewID("mockforward")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
				pipeline.MustNewIDWithName("metrics", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("mockforward")},
				},
			},
			expected: `connector "mockforward" used as exporter in metrics pipeline but not used in any supported receiver pipeline`,
		},
		{
			name: "partially-orphaned-connector-use-as-receiver",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewID("mockforward"): mfConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("metrics", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("mockforward")},
				},
				pipeline.MustNewIDWithName("metrics", "out"): {
					Receivers: []component.ID{component.MustNewID("mockforward")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
				pipeline.MustNewIDWithName("traces", "out"): {
					Receivers: []component.ID{component.MustNewID("mockforward")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: `connector "mockforward" used as receiver in traces pipeline but not used in any supported exporter pipeline`,
		},
		{
			name: "not_allowed_simple_cycle_traces.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			processorCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewIDWithName("nop", "conn"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("traces"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn")},
				},
			},
			expected: `cycle detected: ` +
				`connector "nop/conn" (traces to traces) -> ` +
				`processor "nop" in pipeline "traces" -> ` +
				`connector "nop/conn" (traces to traces)`,
		},
		{
			name: "not_allowed_simple_cycle_metrics.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			processorCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewIDWithName("nop", "conn"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("metrics"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn")},
				},
			},
			expected: `cycle detected: ` +
				`connector "nop/conn" (metrics to metrics) -> ` +
				`processor "nop" in pipeline "metrics" -> ` +
				`connector "nop/conn" (metrics to metrics)`,
		},
		{
			name: "not_allowed_simple_cycle_logs.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			processorCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewIDWithName("nop", "conn"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("logs"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn")},
				},
			},
			expected: `cycle detected: ` +
				`connector "nop/conn" (logs to logs) -> ` +
				`processor "nop" in pipeline "logs" -> ` +
				`connector "nop/conn" (logs to logs)`,
		},
		{
			name: "not_allowed_simple_cycle_profiles.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			processorCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewIDWithName("nop", "conn"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("profiles"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn")},
				},
			},
			expected: `cycle detected: ` +
				`connector "nop/conn" (profiles to profiles) -> ` +
				`processor "nop" in pipeline "profiles" -> ` +
				`connector "nop/conn" (profiles to profiles)`,
		},
		{
			name: "not_allowed_deep_cycle_traces.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			processorCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewIDWithName("nop", "conn"):  nopConnectorFactory.CreateDefaultConfig(),
				component.MustNewIDWithName("nop", "conn1"): nopConnectorFactory.CreateDefaultConfig(),
				component.MustNewIDWithName("nop", "conn2"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("traces", "in"): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn")},
				},
				pipeline.MustNewIDWithName("traces", "1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn1")},
				},
				pipeline.MustNewIDWithName("traces", "2"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn1")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn2"), component.MustNewIDWithName("nop", "conn")},
				},
				pipeline.MustNewIDWithName("traces", "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn2")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				},
			},
			expected: `cycle detected: ` +
				`connector "nop/conn1" (traces to traces) -> ` +
				`processor "nop" in pipeline "traces/2" -> ` +
				`connector "nop/conn" (traces to traces) -> ` +
				`processor "nop" in pipeline "traces/1" -> ` +
				`connector "nop/conn1" (traces to traces)`,
		},
		{
			name: "not_allowed_deep_cycle_metrics.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			processorCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewIDWithName("nop", "conn"):  nopConnectorFactory.CreateDefaultConfig(),
				component.MustNewIDWithName("nop", "conn1"): nopConnectorFactory.CreateDefaultConfig(),
				component.MustNewIDWithName("nop", "conn2"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("metrics", "in"): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn")},
				},
				pipeline.MustNewIDWithName("metrics", "1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn1")},
				},
				pipeline.MustNewIDWithName("metrics", "2"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn1")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn2"), component.MustNewIDWithName("nop", "conn")},
				},
				pipeline.MustNewIDWithName("metrics", "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn2")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				},
			},
			expected: `cycle detected: ` +
				`connector "nop/conn1" (metrics to metrics) -> ` +
				`processor "nop" in pipeline "metrics/2" -> ` +
				`connector "nop/conn" (metrics to metrics) -> ` +
				`processor "nop" in pipeline "metrics/1" -> ` +
				`connector "nop/conn1" (metrics to metrics)`,
		},
		{
			name: "not_allowed_deep_cycle_logs.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			processorCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewIDWithName("nop", "conn"):  nopConnectorFactory.CreateDefaultConfig(),
				component.MustNewIDWithName("nop", "conn1"): nopConnectorFactory.CreateDefaultConfig(),
				component.MustNewIDWithName("nop", "conn2"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("logs", "in"): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn")},
				},
				pipeline.MustNewIDWithName("logs", "1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn1")},
				},
				pipeline.MustNewIDWithName("logs", "2"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn1")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn2"), component.MustNewIDWithName("nop", "conn")},
				},
				pipeline.MustNewIDWithName("logs", "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn2")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				},
			},
			expected: `cycle detected: ` +
				`connector "nop/conn1" (logs to logs) -> ` +
				`processor "nop" in pipeline "logs/2" -> ` +
				`connector "nop/conn" (logs to logs) -> ` +
				`processor "nop" in pipeline "logs/1" -> ` +
				`connector "nop/conn1" (logs to logs)`,
		},
		{
			name: "not_allowed_deep_cycle_profiles.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			processorCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewIDWithName("nop", "conn"):  nopConnectorFactory.CreateDefaultConfig(),
				component.MustNewIDWithName("nop", "conn1"): nopConnectorFactory.CreateDefaultConfig(),
				component.MustNewIDWithName("nop", "conn2"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("profiles", "in"): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn")},
				},
				pipeline.MustNewIDWithName("profiles", "1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn1")},
				},
				pipeline.MustNewIDWithName("profiles", "2"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn1")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn2"), component.MustNewIDWithName("nop", "conn")},
				},
				pipeline.MustNewIDWithName("profiles", "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn2")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				},
			},
			expected: `cycle detected: ` +
				`connector "nop/conn1" (profiles to profiles) -> ` +
				`processor "nop" in pipeline "profiles/2" -> ` +
				`connector "nop/conn" (profiles to profiles) -> ` +
				`processor "nop" in pipeline "profiles/1" -> ` +
				`connector "nop/conn1" (profiles to profiles)`,
		},
		{
			name: "not_allowed_deep_cycle_multi_signal.yaml",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			processorCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewIDWithName("nop", "fork"):      nopConnectorFactory.CreateDefaultConfig(),
				component.MustNewIDWithName("nop", "count"):     nopConnectorFactory.CreateDefaultConfig(),
				component.MustNewIDWithName("nop", "forkagain"): nopConnectorFactory.CreateDefaultConfig(),
				component.MustNewIDWithName("nop", "rawlog"):    nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("traces", "in"): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "fork")},
				},
				pipeline.MustNewIDWithName("traces", "copy1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "fork")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "count")},
				},
				pipeline.MustNewIDWithName("traces", "copy2"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "fork")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "forkagain")},
				},
				pipeline.MustNewIDWithName("traces", "copy2a"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "forkagain")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "count")},
				},
				pipeline.MustNewIDWithName("traces", "copy2b"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "forkagain")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "rawlog")},
				},
				pipeline.MustNewIDWithName("metrics", "count"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "count")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				},
				pipeline.MustNewIDWithName("logs", "raw"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "rawlog")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "fork")}, // cannot loop back to "nop/fork"
				},
			},
			expected: `cycle detected: ` +
				`connector "nop/rawlog" (traces to logs) -> ` +
				`processor "nop" in pipeline "logs/raw" -> ` +
				`connector "nop/fork" (logs to traces) -> ` +
				`processor "nop" in pipeline "traces/copy2" -> ` +
				`connector "nop/forkagain" (traces to traces) -> ` +
				`processor "nop" in pipeline "traces/copy2b" -> ` +
				`connector "nop/rawlog" (traces to logs)`,
		},
		{
			name: "unknown_exporter_config",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("traces"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("nop"), component.MustNewIDWithName("nop", "1")},
				},
			},
			expected: "failed to create \"nop/1\" exporter for data type \"traces\": exporter \"nop/1\" is not configured",
		},
		{
			name: "unknown_exporter_factory",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("unknown"): nopReceiverFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("traces"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("unknown")},
				},
			},
			expected: "failed to create \"unknown\" exporter for data type \"traces\": exporter factory not available for: \"unknown\"",
		},
		{
			name: "unknown_processor_config",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			processorCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("metrics"): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop"), component.MustNewIDWithName("nop", "1")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "failed to create \"nop/1\" processor, in pipeline \"metrics\": processor \"nop/1\" is not configured",
		},
		{
			name: "unknown_processor_factory",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			processorCfgs: map[component.ID]component.Config{
				component.MustNewID("unknown"): nopProcessorFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("metrics"): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("unknown")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "failed to create \"unknown\" processor, in pipeline \"metrics\": processor factory not available for: \"unknown\"",
		},
		{
			name: "unknown_receiver_config",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("logs"): {
					Receivers: []component.ID{component.MustNewID("nop"), component.MustNewIDWithName("nop", "1")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "failed to create \"nop/1\" receiver for data type \"logs\": receiver \"nop/1\" is not configured",
		},
		{
			name: "unknown_receiver_factory",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("unknown"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewID("logs"): {
					Receivers: []component.ID{component.MustNewID("unknown")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "failed to create \"unknown\" receiver for data type \"logs\": receiver factory not available for: \"unknown\"",
		},
		{
			name: "unknown_connector_factory",
			receiverCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			exporterCfgs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			connectorCfgs: map[component.ID]component.Config{
				component.MustNewID("unknown"): nopConnectorFactory.CreateDefaultConfig(),
			},
			pipelineCfgs: pipelines.ConfigWithPipelineID{
				pipeline.MustNewIDWithName("traces", "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("unknown")},
				},
				pipeline.MustNewIDWithName("traces", "out"): {
					Receivers: []component.ID{component.MustNewID("unknown")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector factory not available for: \"unknown\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := Settings{
				BuildInfo: component.NewDefaultBuildInfo(),
				Telemetry: componenttest.NewNopTelemetrySettings(),
				ReceiverBuilder: builders.NewReceiver(
					tt.receiverCfgs,
					map[component.Type]receiver.Factory{
						nopReceiverFactory.Type(): nopReceiverFactory,
						badReceiverFactory.Type(): badReceiverFactory,
					}),
				ProcessorBuilder: builders.NewProcessor(
					tt.processorCfgs,
					map[component.Type]processor.Factory{
						nopProcessorFactory.Type(): nopProcessorFactory,
						badProcessorFactory.Type(): badProcessorFactory,
					}),
				ExporterBuilder: builders.NewExporter(
					tt.exporterCfgs,
					map[component.Type]exporter.Factory{
						nopExporterFactory.Type(): nopExporterFactory,
						badExporterFactory.Type(): badExporterFactory,
					}),
				ConnectorBuilder: builders.NewConnector(
					tt.connectorCfgs,
					map[component.Type]connector.Factory{
						nopConnectorFactory.Type(): nopConnectorFactory,
						badConnectorFactory.Type(): badConnectorFactory,
						mfConnectorFactory.Type():  mfConnectorFactory,
					}),
				PipelineConfigs: tt.pipelineCfgs,
			}
			_, err := Build(context.Background(), set)
			assert.EqualError(t, err, tt.expected)
		})
	}
}

// This includes all tests from the previous implementation, plus a new one
// relevant only to the new graph-based implementation.
func TestGraphFailToStartAndShutdown(t *testing.T) {
	errReceiverFactory := newErrReceiverFactory()
	errProcessorFactory := newErrProcessorFactory()
	errExporterFactory := newErrExporterFactory()
	errConnectorFactory := newErrConnectorFactory()
	nopReceiverFactory := receivertest.NewNopFactory()
	nopProcessorFactory := processortest.NewNopFactory()
	nopExporterFactory := exportertest.NewNopFactory()
	nopConnectorFactory := connectortest.NewNopFactory()

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			map[component.ID]component.Config{
				component.NewID(nopReceiverFactory.Type()): nopReceiverFactory.CreateDefaultConfig(),
				component.NewID(errReceiverFactory.Type()): errReceiverFactory.CreateDefaultConfig(),
			},
			map[component.Type]receiver.Factory{
				nopReceiverFactory.Type(): nopReceiverFactory,
				errReceiverFactory.Type(): errReceiverFactory,
			}),
		ProcessorBuilder: builders.NewProcessor(
			map[component.ID]component.Config{
				component.NewID(nopProcessorFactory.Type()): nopProcessorFactory.CreateDefaultConfig(),
				component.NewID(errProcessorFactory.Type()): errProcessorFactory.CreateDefaultConfig(),
			},
			map[component.Type]processor.Factory{
				nopProcessorFactory.Type(): nopProcessorFactory,
				errProcessorFactory.Type(): errProcessorFactory,
			}),
		ExporterBuilder: builders.NewExporter(
			map[component.ID]component.Config{
				component.NewID(nopExporterFactory.Type()): nopExporterFactory.CreateDefaultConfig(),
				component.NewID(errExporterFactory.Type()): errExporterFactory.CreateDefaultConfig(),
			},
			map[component.Type]exporter.Factory{
				nopExporterFactory.Type(): nopExporterFactory,
				errExporterFactory.Type(): errExporterFactory,
			}),
		ConnectorBuilder: builders.NewConnector(
			map[component.ID]component.Config{
				component.NewIDWithName(nopConnectorFactory.Type(), "conn"): nopConnectorFactory.CreateDefaultConfig(),
				component.NewIDWithName(errConnectorFactory.Type(), "conn"): errConnectorFactory.CreateDefaultConfig(),
			},
			map[component.Type]connector.Factory{
				nopConnectorFactory.Type(): nopConnectorFactory,
				errConnectorFactory.Type(): errConnectorFactory,
			}),
	}

	dataTypes := []pipeline.Signal{pipeline.SignalTraces, pipeline.SignalMetrics, pipeline.SignalLogs}
	for _, dt := range dataTypes {
		t.Run(dt.String()+"/receiver", func(t *testing.T) {
			set.PipelineConfigs = pipelines.ConfigWithPipelineID{
				pipeline.NewID(dt): {
					Receivers:  []component.ID{component.MustNewID("nop"), component.MustNewID("err")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				},
			}
			pipelines, err := Build(context.Background(), set)
			require.NoError(t, err)
			require.Error(t, pipelines.StartAll(context.Background(), &Host{Reporter: status.NewReporter(func(*componentstatus.InstanceID, *componentstatus.Event) {}, func(error) {})}))
			assert.Error(t, pipelines.ShutdownAll(context.Background(), statustest.NewNopStatusReporter()))
		})

		t.Run(dt.String()+"/processor", func(t *testing.T) {
			set.PipelineConfigs = pipelines.ConfigWithPipelineID{
				pipeline.NewID(dt): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop"), component.MustNewID("err")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				},
			}
			pipelines, err := Build(context.Background(), set)
			require.NoError(t, err)
			require.Error(t, pipelines.StartAll(context.Background(), &Host{Reporter: status.NewReporter(func(*componentstatus.InstanceID, *componentstatus.Event) {}, func(error) {})}))
			assert.Error(t, pipelines.ShutdownAll(context.Background(), statustest.NewNopStatusReporter()))
		})

		t.Run(dt.String()+"/exporter", func(t *testing.T) {
			set.PipelineConfigs = pipelines.ConfigWithPipelineID{
				pipeline.NewID(dt): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewID("nop"), component.MustNewID("err")},
				},
			}
			pipelines, err := Build(context.Background(), set)
			require.NoError(t, err)
			require.Error(t, pipelines.StartAll(context.Background(), &Host{Reporter: status.NewReporter(func(*componentstatus.InstanceID, *componentstatus.Event) {}, func(error) {})}))
			assert.Error(t, pipelines.ShutdownAll(context.Background(), statustest.NewNopStatusReporter()))
		})

		for _, dt2 := range dataTypes {
			t.Run(dt.String()+"/"+dt2.String()+"/connector", func(t *testing.T) {
				set.PipelineConfigs = pipelines.ConfigWithPipelineID{
					pipeline.NewIDWithName(dt, "in"): {
						Receivers:  []component.ID{component.MustNewID("nop")},
						Processors: []component.ID{component.MustNewID("nop")},
						Exporters:  []component.ID{component.MustNewID("nop"), component.MustNewIDWithName("err", "conn")},
					},
					pipeline.NewIDWithName(dt2, "out"): {
						Receivers:  []component.ID{component.MustNewID("nop"), component.MustNewIDWithName("err", "conn")},
						Processors: []component.ID{component.MustNewID("nop")},
						Exporters:  []component.ID{component.MustNewID("nop")},
					},
				}
				pipelines, err := Build(context.Background(), set)
				require.NoError(t, err)
				require.Error(t, pipelines.StartAll(context.Background(), &Host{Reporter: status.NewReporter(func(*componentstatus.InstanceID, *componentstatus.Event) {}, func(error) {})}))
				assert.Error(t, pipelines.ShutdownAll(context.Background(), statustest.NewNopStatusReporter()))
			})
		}
	}
}

func TestStatusReportedOnStartupShutdown(t *testing.T) {

	rNoErr := &testNode{id: component.MustNewIDWithName("r_no_err", "1")}
	rStErr := &testNode{id: component.MustNewIDWithName("r_st_err", "1"), startErr: assert.AnError}
	rSdErr := &testNode{id: component.MustNewIDWithName("r_sd_err", "1"), shutdownErr: assert.AnError}

	eNoErr := &testNode{id: component.MustNewIDWithName("e_no_err", "1")}
	eStErr := &testNode{id: component.MustNewIDWithName("e_st_err", "1"), startErr: assert.AnError}
	eSdErr := &testNode{id: component.MustNewIDWithName("e_sd_err", "1"), shutdownErr: assert.AnError}

	instanceIDs := map[*testNode]*componentstatus.InstanceID{
		rNoErr: componentstatus.NewInstanceIDWithPipelineIDs(rNoErr.id, component.KindReceiver),
		rStErr: componentstatus.NewInstanceIDWithPipelineIDs(rStErr.id, component.KindReceiver),
		rSdErr: componentstatus.NewInstanceIDWithPipelineIDs(rSdErr.id, component.KindReceiver),
		eNoErr: componentstatus.NewInstanceIDWithPipelineIDs(eNoErr.id, component.KindExporter),
		eStErr: componentstatus.NewInstanceIDWithPipelineIDs(eStErr.id, component.KindExporter),
		eSdErr: componentstatus.NewInstanceIDWithPipelineIDs(eSdErr.id, component.KindExporter),
	}

	// compare two maps of status events ignoring timestamp
	assertEqualStatuses := func(t *testing.T, evMap1, evMap2 map[*componentstatus.InstanceID][]*componentstatus.Event) {
		assert.Equal(t, len(evMap1), len(evMap2))
		for id, evts1 := range evMap1 {
			evts2 := evMap2[id]
			assert.Equal(t, len(evts1), len(evts2))
			for i := 0; i < len(evts1); i++ {
				ev1 := evts1[i]
				ev2 := evts2[i]
				assert.Equal(t, ev1.Status(), ev2.Status())
				assert.Equal(t, ev1.Err(), ev2.Err())
			}
		}

	}

	for _, tt := range []struct {
		name             string
		edge             [2]*testNode
		expectedStatuses map[*componentstatus.InstanceID][]*componentstatus.Event
		startupErr       error
		shutdownErr      error
	}{
		{
			name: "successful startup/shutdown",
			edge: [2]*testNode{rNoErr, eNoErr},
			expectedStatuses: map[*componentstatus.InstanceID][]*componentstatus.Event{
				instanceIDs[rNoErr]: {
					componentstatus.NewEvent(componentstatus.StatusStarting),
					componentstatus.NewEvent(componentstatus.StatusOK),
					componentstatus.NewEvent(componentstatus.StatusStopping),
					componentstatus.NewEvent(componentstatus.StatusStopped),
				},
				instanceIDs[eNoErr]: {
					componentstatus.NewEvent(componentstatus.StatusStarting),
					componentstatus.NewEvent(componentstatus.StatusOK),
					componentstatus.NewEvent(componentstatus.StatusStopping),
					componentstatus.NewEvent(componentstatus.StatusStopped),
				},
			},
		},
		{
			name: "early startup error",
			edge: [2]*testNode{rNoErr, eStErr},
			expectedStatuses: map[*componentstatus.InstanceID][]*componentstatus.Event{
				instanceIDs[eStErr]: {
					componentstatus.NewEvent(componentstatus.StatusStarting),
					componentstatus.NewPermanentErrorEvent(assert.AnError),
				},
			},
			startupErr: assert.AnError,
		},
		{
			name: "late startup error",
			edge: [2]*testNode{rStErr, eNoErr},
			expectedStatuses: map[*componentstatus.InstanceID][]*componentstatus.Event{
				instanceIDs[rStErr]: {
					componentstatus.NewEvent(componentstatus.StatusStarting),
					componentstatus.NewPermanentErrorEvent(assert.AnError),
				},
				instanceIDs[eNoErr]: {
					componentstatus.NewEvent(componentstatus.StatusStarting),
					componentstatus.NewEvent(componentstatus.StatusOK),
					componentstatus.NewEvent(componentstatus.StatusStopping),
					componentstatus.NewEvent(componentstatus.StatusStopped),
				},
			},
			startupErr: assert.AnError,
		},
		{
			name: "early shutdown error",
			edge: [2]*testNode{rSdErr, eNoErr},
			expectedStatuses: map[*componentstatus.InstanceID][]*componentstatus.Event{
				instanceIDs[rSdErr]: {
					componentstatus.NewEvent(componentstatus.StatusStarting),
					componentstatus.NewEvent(componentstatus.StatusOK),
					componentstatus.NewEvent(componentstatus.StatusStopping),
					componentstatus.NewPermanentErrorEvent(assert.AnError),
				},
				instanceIDs[eNoErr]: {
					componentstatus.NewEvent(componentstatus.StatusStarting),
					componentstatus.NewEvent(componentstatus.StatusOK),
					componentstatus.NewEvent(componentstatus.StatusStopping),
					componentstatus.NewEvent(componentstatus.StatusStopped),
				},
			},
			shutdownErr: assert.AnError,
		},
		{
			name: "late shutdown error",
			edge: [2]*testNode{rNoErr, eSdErr},
			expectedStatuses: map[*componentstatus.InstanceID][]*componentstatus.Event{
				instanceIDs[rNoErr]: {
					componentstatus.NewEvent(componentstatus.StatusStarting),
					componentstatus.NewEvent(componentstatus.StatusOK),
					componentstatus.NewEvent(componentstatus.StatusStopping),
					componentstatus.NewEvent(componentstatus.StatusStopped),
				},
				instanceIDs[eSdErr]: {
					componentstatus.NewEvent(componentstatus.StatusStarting),
					componentstatus.NewEvent(componentstatus.StatusOK),
					componentstatus.NewEvent(componentstatus.StatusStopping),
					componentstatus.NewPermanentErrorEvent(assert.AnError),
				},
			},
			shutdownErr: assert.AnError,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pg := &Graph{componentGraph: simple.NewDirectedGraph()}
			pg.telemetry = componenttest.NewNopTelemetrySettings()

			actualStatuses := make(map[*componentstatus.InstanceID][]*componentstatus.Event)
			rep := status.NewReporter(func(id *componentstatus.InstanceID, ev *componentstatus.Event) {
				actualStatuses[id] = append(actualStatuses[id], ev)
			}, func(error) {
			})

			rep.Ready()

			e0, e1 := tt.edge[0], tt.edge[1]
			pg.instanceIDs = map[int64]*componentstatus.InstanceID{
				e0.ID(): instanceIDs[e0],
				e1.ID(): instanceIDs[e1],
			}
			pg.componentGraph.SetEdge(simple.Edge{F: e0, T: e1})

			assert.Equal(t, tt.startupErr, pg.StartAll(context.Background(), &Host{Reporter: rep}))
			assert.Equal(t, tt.shutdownErr, pg.ShutdownAll(context.Background(), rep))
			assertEqualStatuses(t, tt.expectedStatuses, actualStatuses)
		})
	}
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
func expectedInstances(m pipelines.ConfigWithPipelineID, pID pipeline.ID) (int, int) {
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
	return receiver.NewFactory(component.MustNewType("err"),
		func() component.Config { return &struct{}{} },
		receiver.WithTraces(func(context.Context, receiver.Settings, component.Config, consumer.Traces) (receiver.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		receiver.WithLogs(func(context.Context, receiver.Settings, component.Config, consumer.Logs) (receiver.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		receiver.WithMetrics(func(context.Context, receiver.Settings, component.Config, consumer.Metrics) (receiver.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		receiverprofiles.WithProfiles(func(context.Context, receiver.Settings, component.Config, consumerprofiles.Profiles) (receiverprofiles.Profiles, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func newErrProcessorFactory() processor.Factory {
	return processor.NewFactory(component.MustNewType("err"),
		func() component.Config { return &struct{}{} },
		processor.WithTraces(func(context.Context, processor.Settings, component.Config, consumer.Traces) (processor.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		processor.WithLogs(func(context.Context, processor.Settings, component.Config, consumer.Logs) (processor.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		processor.WithMetrics(func(context.Context, processor.Settings, component.Config, consumer.Metrics) (processor.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		processorprofiles.WithProfiles(func(context.Context, processor.Settings, component.Config, consumerprofiles.Profiles) (processorprofiles.Profiles, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func newErrExporterFactory() exporter.Factory {
	return exporter.NewFactory(component.MustNewType("err"),
		func() component.Config { return &struct{}{} },
		exporter.WithTraces(func(context.Context, exporter.Settings, component.Config) (exporter.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		exporter.WithLogs(func(context.Context, exporter.Settings, component.Config) (exporter.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		exporter.WithMetrics(func(context.Context, exporter.Settings, component.Config) (exporter.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		exporterprofiles.WithProfiles(func(context.Context, exporter.Settings, component.Config) (exporterprofiles.Profiles, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func newErrConnectorFactory() connector.Factory {
	return connector.NewFactory(component.MustNewType("err"), func() component.Config {
		return &struct{}{}
	},
		connector.WithTracesToTraces(func(context.Context, connector.Settings, component.Config, consumer.Traces) (connector.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connector.WithTracesToMetrics(func(context.Context, connector.Settings, component.Config, consumer.Metrics) (connector.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connector.WithTracesToLogs(func(context.Context, connector.Settings, component.Config, consumer.Logs) (connector.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connectorprofiles.WithTracesToProfiles(func(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connector.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),

		connector.WithMetricsToTraces(func(context.Context, connector.Settings, component.Config, consumer.Traces) (connector.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connector.WithMetricsToMetrics(func(context.Context, connector.Settings, component.Config, consumer.Metrics) (connector.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connector.WithMetricsToLogs(func(context.Context, connector.Settings, component.Config, consumer.Logs) (connector.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connectorprofiles.WithMetricsToProfiles(func(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connector.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),

		connector.WithLogsToTraces(func(context.Context, connector.Settings, component.Config, consumer.Traces) (connector.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connector.WithLogsToMetrics(func(context.Context, connector.Settings, component.Config, consumer.Metrics) (connector.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connector.WithLogsToLogs(func(context.Context, connector.Settings, component.Config, consumer.Logs) (connector.Logs, error) {
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
