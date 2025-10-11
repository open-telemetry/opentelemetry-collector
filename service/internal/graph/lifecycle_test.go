// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gonum.org/v1/gonum/graph/simple"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/status"
	"go.opentelemetry.io/collector/service/pipelines"
)

func TestGraphStartStop(t *testing.T) {
	t.Run("with_internal_telemetry", func(t *testing.T) {
		setObsConsumerGateForTest(t, true)
		testGraphStartStop(t)
	})
	t.Run("without_internal_telemetry", func(t *testing.T) {
		setObsConsumerGateForTest(t, false)
		testGraphStartStop(t)
	})
}

func testGraphStartStop(t *testing.T) {
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
			require.NoError(t, pg.ShutdownAll(ctx, status.NewNopStatusReporter()))
			for _, edge := range tt.edges {
				assert.Less(t, ctx.order[edge[0]], ctx.order[edge[1]])
			}
		})
	}
}

func TestGraphStartStopCycle(t *testing.T) {
	t.Run("with_internal_telemetry", func(t *testing.T) {
		setObsConsumerGateForTest(t, true)
		testGraphStartStopCycle(t)
	})
	t.Run("without_internal_telemetry", func(t *testing.T) {
		setObsConsumerGateForTest(t, false)
		testGraphStartStopCycle(t)
	})
}

func testGraphStartStopCycle(t *testing.T) {
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
	require.ErrorContains(t, err, `topo: no topological ordering: cyclic components`)

	err = pg.ShutdownAll(context.Background(), status.NewNopStatusReporter())
	assert.ErrorContains(t, err, `topo: no topological ordering: cyclic components`)
}

func TestGraphStartStopComponentError(t *testing.T) {
	t.Run("with_internal_telemetry", func(t *testing.T) {
		setObsConsumerGateForTest(t, true)
		testGraphStartStopComponentError(t)
	})
	t.Run("without_internal_telemetry", func(t *testing.T) {
		setObsConsumerGateForTest(t, false)
		testGraphStartStopComponentError(t)
	})
}

func testGraphStartStopComponentError(t *testing.T) {
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
	require.ErrorIs(t, pg.StartAll(context.Background(), &Host{Reporter: status.NewReporter(func(*componentstatus.InstanceID, *componentstatus.Event) {}, func(error) {})}), r1.startErr)
	assert.EqualError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()), "bar")
}

// This includes all tests from the previous implementation, plus a new one
// relevant only to the new graph-based implementation.
func TestGraphFailToStartAndShutdown(t *testing.T) {
	t.Run("with_internal_telemetry", func(t *testing.T) {
		setObsConsumerGateForTest(t, true)
		testGraphFailToStartAndShutdown(t)
	})
	t.Run("without_internal_telemetry", func(t *testing.T) {
		setObsConsumerGateForTest(t, false)
		testGraphFailToStartAndShutdown(t)
	})
}

func testGraphFailToStartAndShutdown(t *testing.T) {
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
			set.PipelineConfigs = pipelines.Config{
				pipeline.NewID(dt): {
					Receivers:  []component.ID{component.MustNewID("nop"), component.MustNewID("err")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				},
			}
			pipelines, err := Build(context.Background(), set)
			require.NoError(t, err)
			require.Error(t, pipelines.StartAll(context.Background(), &Host{Reporter: status.NewReporter(func(*componentstatus.InstanceID, *componentstatus.Event) {}, func(error) {})}))
			assert.Error(t, pipelines.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
		})

		t.Run(dt.String()+"/processor", func(t *testing.T) {
			set.PipelineConfigs = pipelines.Config{
				pipeline.NewID(dt): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop"), component.MustNewID("err")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				},
			}
			pipelines, err := Build(context.Background(), set)
			require.NoError(t, err)
			require.Error(t, pipelines.StartAll(context.Background(), &Host{Reporter: status.NewReporter(func(*componentstatus.InstanceID, *componentstatus.Event) {}, func(error) {})}))
			assert.Error(t, pipelines.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
		})

		t.Run(dt.String()+"/exporter", func(t *testing.T) {
			set.PipelineConfigs = pipelines.Config{
				pipeline.NewID(dt): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewID("nop"), component.MustNewID("err")},
				},
			}
			pipelines, err := Build(context.Background(), set)
			require.NoError(t, err)
			require.Error(t, pipelines.StartAll(context.Background(), &Host{Reporter: status.NewReporter(func(*componentstatus.InstanceID, *componentstatus.Event) {}, func(error) {})}))
			assert.Error(t, pipelines.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
		})

		for _, dt2 := range dataTypes {
			t.Run(dt.String()+"/"+dt2.String()+"/connector", func(t *testing.T) {
				set.PipelineConfigs = pipelines.Config{
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
				assert.Error(t, pipelines.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
			})
		}
	}
}

func TestStatusReportedOnStartupShutdown(t *testing.T) {
	t.Run("with_internal_telemetry", func(t *testing.T) {
		setObsConsumerGateForTest(t, true)
		testStatusReportedOnStartupShutdown(t)
	})
	t.Run("without_internal_telemetry", func(t *testing.T) {
		setObsConsumerGateForTest(t, false)
		testStatusReportedOnStartupShutdown(t)
	})
}

func testStatusReportedOnStartupShutdown(t *testing.T) {
	rNoErr := &testNode{id: component.MustNewIDWithName("r_no_err", "1")}
	rStErr := &testNode{id: component.MustNewIDWithName("r_st_err", "1"), startErr: assert.AnError}
	rSdErr := &testNode{id: component.MustNewIDWithName("r_sd_err", "1"), shutdownErr: assert.AnError}

	eNoErr := &testNode{id: component.MustNewIDWithName("e_no_err", "1")}
	eStErr := &testNode{id: component.MustNewIDWithName("e_st_err", "1"), startErr: assert.AnError}
	eSdErr := &testNode{id: component.MustNewIDWithName("e_sd_err", "1"), shutdownErr: assert.AnError}

	instanceIDs := map[*testNode]*componentstatus.InstanceID{
		rNoErr: componentstatus.NewInstanceID(rNoErr.id, component.KindReceiver),
		rStErr: componentstatus.NewInstanceID(rStErr.id, component.KindReceiver),
		rSdErr: componentstatus.NewInstanceID(rSdErr.id, component.KindReceiver),
		eNoErr: componentstatus.NewInstanceID(eNoErr.id, component.KindExporter),
		eStErr: componentstatus.NewInstanceID(eStErr.id, component.KindExporter),
		eSdErr: componentstatus.NewInstanceID(eSdErr.id, component.KindExporter),
	}

	// compare two maps of status events ignoring timestamp
	assertEqualStatuses := func(t *testing.T, evMap1, evMap2 map[*componentstatus.InstanceID][]*componentstatus.Event) {
		assert.Len(t, evMap2, len(evMap1))
		for id, evts1 := range evMap1 {
			evts2 := evMap2[id]
			assert.Len(t, evts2, len(evts1))
			for i := range evts1 {
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
					componentstatus.NewEvent(componentstatus.StatusStopping),
					componentstatus.NewEvent(componentstatus.StatusStopped),
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

			e0, e1 := tt.edge[0], tt.edge[1]
			pg.instanceIDs = map[int64]*componentstatus.InstanceID{
				e0.ID(): instanceIDs[e0],
				e1.ID(): instanceIDs[e1],
			}
			pg.componentGraph.SetEdge(simple.Edge{F: e0, T: e1})

			require.ErrorIs(t, pg.StartAll(context.Background(), &Host{Reporter: rep}), tt.startupErr)
			assert.Equal(t, tt.shutdownErr, pg.ShutdownAll(context.Background(), rep))
			assertEqualStatuses(t, tt.expectedStatuses, actualStatuses)
		})
	}
}
