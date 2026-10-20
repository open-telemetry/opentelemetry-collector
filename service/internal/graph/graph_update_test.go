// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/status"
	"go.opentelemetry.io/collector/service/internal/testcomponents"
	"go.opentelemetry.io/collector/service/pipelines"
)

var (
	procA = component.MustNewIDWithName("exampleprocessor", "a")
	procB = component.MustNewIDWithName("exampleprocessor", "b")
	procC = component.MustNewIDWithName("exampleprocessor", "c")

	tracesID = pipeline.NewID(pipeline.SignalTraces)
	logsID   = pipeline.NewID(pipeline.SignalLogs)

	testProcessorFactories = map[component.Type]processor.Factory{
		testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
	}
)

// testReceiver is a distinct instance per build, so tests can tell whether it was rebuilt.
type testReceiver struct {
	starts, stops int
	traces        consumer.Traces
	logs          consumer.Logs
}

func (r *testReceiver) Start(context.Context, component.Host) error { r.starts++; return nil }
func (r *testReceiver) Shutdown(context.Context) error              { r.stops++; return nil }

var (
	testReceiverType               = component.MustNewType("testreceiver")
	processorTestReceiverFactories = map[component.Type]receiver.Factory{
		testReceiverType: receiver.NewFactory(testReceiverType,
			func() component.Config { return &struct{ _ int }{} },
			receiver.WithTraces(func(_ context.Context, _ receiver.Settings, _ component.Config, next consumer.Traces) (receiver.Traces, error) {
				return &testReceiver{traces: next}, nil
			}, component.StabilityLevelDevelopment),
			receiver.WithLogs(func(_ context.Context, _ receiver.Settings, _ component.Config, next consumer.Logs) (receiver.Logs, error) {
				return &testReceiver{logs: next}, nil
			}, component.StabilityLevelDevelopment),
		),
	}
	testReceiverID  = component.NewID(testReceiverType)
	testReceiverID2 = component.NewIDWithName(testReceiverType, "2")
)

func testProcessorCfgs() map[component.ID]component.Config {
	cfgs := map[component.ID]component.Config{}
	for _, id := range []component.ID{procA, procB, procC} {
		cfgs[id] = testcomponents.ExampleProcessorFactory.CreateDefaultConfig()
	}
	return cfgs
}

func testReceiverCfgs() map[component.ID]component.Config {
	return map[component.ID]component.Config{
		testReceiverID:  processorTestReceiverFactories[testReceiverType].CreateDefaultConfig(),
		testReceiverID2: processorTestReceiverFactories[testReceiverType].CreateDefaultConfig(),
	}
}

func processorTestPipelines(traces, logs []component.ID) pipelines.Config {
	cfg := pipelines.Config{
		tracesID: {
			Receivers:  []component.ID{testReceiverID},
			Processors: traces,
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}
	if logs != nil {
		cfg[logsID] = &pipelines.PipelineConfig{
			Receivers:  []component.ID{testReceiverID},
			Processors: logs,
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		}
	}
	return cfg
}

func buildProcessorTestGraph(t *testing.T, pipelineCfgs pipelines.Config, connCfgs map[component.ID]component.Config) (*Graph, *Host, Settings) {
	t.Helper()
	connFactories := map[component.Type]connector.Factory{}
	if len(connCfgs) > 0 {
		connFactories[testcomponents.ExampleConnectorFactory.Type()] = testcomponents.ExampleConnectorFactory
	}
	set := Settings{
		Telemetry:        componenttest.NewNopTelemetrySettings(),
		BuildInfo:        component.NewDefaultBuildInfo(),
		ReceiverBuilder:  builders.NewReceiver(testReceiverCfgs(), processorTestReceiverFactories),
		ProcessorBuilder: builders.NewProcessor(testProcessorCfgs(), testProcessorFactories),
		ExporterBuilder: builders.NewExporter(
			map[component.ID]component.Config{
				component.MustNewID("exampleexporter"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
			},
			map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		),
		ConnectorBuilder: builders.NewConnector(connCfgs, connFactories),
		PipelineConfigs:  pipelineCfgs,
	}
	pg, err := Build(context.Background(), set)
	require.NoError(t, err)
	host := &Host{
		Receivers:  set.ReceiverBuilder,
		Processors: set.ProcessorBuilder,
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))
	t.Cleanup(func() { _ = pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()) })
	return pg, host, set
}

func processorComponents(pg *Graph, pid pipeline.ID) []*testcomponents.ExampleProcessor {
	var out []*testcomponents.ExampleProcessor
	for _, n := range pg.pipelines[pid].processors {
		out = append(out, n.(*processorNode).Component.(*testcomponents.ExampleProcessor))
	}
	return out
}

func receiverComponent(pg *Graph, pid pipeline.ID) *testReceiver {
	for _, n := range pg.pipelines[pid].receivers {
		return n.(*receiverNode).Component.(*testReceiver)
	}
	return nil
}

func sendTraces(t *testing.T, pg *Graph) {
	t.Helper()
	require.NoError(t, receiverComponent(pg, tracesID).traces.ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
}

func exporterComponent(pg *Graph) *testcomponents.ExampleExporter {
	for _, e := range pg.GetExporters()[pipeline.SignalTraces] {
		return e.(*testcomponents.ExampleExporter)
	}
	return nil
}

func updateProcessors(t *testing.T, pg *Graph, host *Host, set Settings, newPipelines pipelines.Config, changed ...component.ID) (bool, error) {
	t.Helper()
	changedSet := map[component.ID]bool{}
	for _, id := range changed {
		changedSet[id] = true
	}
	set.PipelineConfigs = newPipelines
	return pg.Update(context.Background(), set, UpdateRequest{
		ChangedProcessors:  changedSet,
		NewProcessorCfgs:   testProcessorCfgs(),
		ProcessorFactories: testProcessorFactories,
		NewReceiverCfgs:    testReceiverCfgs(),
		ReceiverFactories:  processorTestReceiverFactories,
	}, host)
}

func TestUpdate_ProcessorsNilHost(t *testing.T) {
	pg := &Graph{}
	applied, err := pg.Update(context.Background(), Settings{}, UpdateRequest{}, nil)
	require.EqualError(t, err, "host cannot be nil")
	assert.False(t, applied)
}

func TestUpdate_ProcessorsNoChange(t *testing.T) {
	cfg := processorTestPipelines([]component.ID{procA}, nil)
	pg, host, set := buildProcessorTestGraph(t, cfg, nil)
	before := processorComponents(pg, tracesID)
	rcvr := receiverComponent(pg, tracesID)

	// A changed processor that no pipeline uses affects nothing.
	applied, err := updateProcessors(t, pg, host, set, cfg, procC)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Same(t, before[0], processorComponents(pg, tracesID)[0])
	assert.Same(t, rcvr, receiverComponent(pg, tracesID))
	assert.Zero(t, rcvr.stops)
}

func TestUpdate_ProcessorsConfigChange(t *testing.T) {
	cfg := processorTestPipelines([]component.ID{procA, procB}, []component.ID{procC})
	pg, host, set := buildProcessorTestGraph(t, cfg, nil)

	tracesProcs := processorComponents(pg, tracesID)
	tracesRcvr := receiverComponent(pg, tracesID)
	logsProcs := processorComponents(pg, logsID)
	logsRcvr := receiverComponent(pg, logsID)
	exp := exporterComponent(pg)
	expStoppedBefore := exp.Stopped()

	// Changing the last processor rebuilds all processors in the pipeline.
	applied, err := updateProcessors(t, pg, host, set, cfg, procB)
	require.NoError(t, err)
	assert.True(t, applied)

	newProcs := processorComponents(pg, tracesID)
	require.Len(t, newProcs, 2)
	for i := range newProcs {
		assert.NotSame(t, tracesProcs[i], newProcs[i])
		assert.True(t, tracesProcs[i].Stopped())
		assert.True(t, newProcs[i].Started())
	}
	assert.NotSame(t, tracesRcvr, receiverComponent(pg, tracesID))
	assert.Equal(t, 1, tracesRcvr.stops)
	assert.Equal(t, 1, receiverComponent(pg, tracesID).starts)

	// Receivers are keyed by signal, so the logs pipeline is untouched.
	assert.Same(t, logsProcs[0], processorComponents(pg, logsID)[0])
	assert.Same(t, logsRcvr, receiverComponent(pg, logsID))
	assert.Zero(t, logsRcvr.stops)
	assert.Equal(t, expStoppedBefore, exp.Stopped())

	// Data flows through the rebuilt pipeline.
	exportedBefore := len(exporterComponent(pg).Traces)
	sendTraces(t, pg)
	assert.Len(t, exporterComponent(pg).Traces, exportedBefore+1)
}

func TestUpdate_ProcessorsKeepsUnchangedDownstreamProcessors(t *testing.T) {
	cfg := processorTestPipelines([]component.ID{procA, procB}, nil)
	pg, host, set := buildProcessorTestGraph(t, cfg, nil)
	before := processorComponents(pg, tracesID)

	// The unchanged trailing processor keeps running.
	applied, err := updateProcessors(t, pg, host, set, cfg, procA)
	require.NoError(t, err)
	assert.True(t, applied)

	after := processorComponents(pg, tracesID)
	assert.NotSame(t, before[0], after[0])
	assert.True(t, before[0].Stopped())
	assert.Same(t, before[1], after[1])
	assert.False(t, before[1].Stopped())

	exportedBefore := len(exporterComponent(pg).Traces)
	sendTraces(t, pg)
	assert.Len(t, exporterComponent(pg).Traces, exportedBefore+1)
}

func TestUpdate_ProcessorsListChanges(t *testing.T) {
	tests := []struct {
		name      string
		oldProcs  []component.ID
		newProcs  []component.ID
		wantKept  int // number of trailing processors expected to be reused
		wantCount int
	}{
		{"add_first", []component.ID{procB}, []component.ID{procA, procB}, 1, 2},
		{"add_last", []component.ID{procA}, []component.ID{procA, procB}, 0, 2},
		{"remove_first", []component.ID{procA, procB}, []component.ID{procB}, 1, 1},
		{"remove_last", []component.ID{procA, procB}, []component.ID{procA}, 0, 1},
		{"remove_all", []component.ID{procA}, nil, 0, 0},
		{"add_to_empty", nil, []component.ID{procA}, 0, 1},
		{"reorder", []component.ID{procA, procB}, []component.ID{procB, procA}, 0, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, host, set := buildProcessorTestGraph(t, processorTestPipelines(tt.oldProcs, nil), nil)
			before := processorComponents(pg, tracesID)

			applied, err := updateProcessors(t, pg, host, set, processorTestPipelines(tt.newProcs, nil))
			require.NoError(t, err)
			assert.True(t, applied)

			after := processorComponents(pg, tracesID)
			require.Len(t, after, tt.wantCount)
			for i := 1; i <= tt.wantKept; i++ {
				assert.Same(t, before[len(before)-i], after[len(after)-i])
			}
			for i := 0; i < len(after)-tt.wantKept; i++ {
				assert.True(t, after[i].Started())
			}

			// Data reaches the exporter.
			exportedBefore := len(exporterComponent(pg).Traces)
			sendTraces(t, pg)
			assert.Len(t, exporterComponent(pg).Traces, exportedBefore+1)
		})
	}
}

func TestUpdate_ProcessorsRequiresFullReload(t *testing.T) {
	t.Run("pipeline_set_changed", func(t *testing.T) {
		cfg := processorTestPipelines([]component.ID{procA}, nil)
		pg, host, set := buildProcessorTestGraph(t, cfg, nil)
		rcvr := receiverComponent(pg, tracesID)

		applied, err := updateProcessors(t, pg, host, set, processorTestPipelines([]component.ID{procA}, []component.ID{procA}), procA)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Zero(t, rcvr.stops, "graph must not be modified")
	})

	t.Run("fed_by_connector", func(t *testing.T) {
		connID := component.MustNewID("exampleconnector")
		in := pipeline.NewIDWithName(pipeline.SignalTraces, "in")
		out := pipeline.NewIDWithName(pipeline.SignalTraces, "out")
		mk := func(outProcs []component.ID) pipelines.Config {
			return pipelines.Config{
				in:  {Receivers: []component.ID{testReceiverID}, Exporters: []component.ID{connID}},
				out: {Receivers: []component.ID{connID}, Processors: outProcs, Exporters: []component.ID{component.MustNewID("exampleexporter")}},
			}
		}
		connCfgs := map[component.ID]component.Config{connID: testcomponents.ExampleConnectorFactory.CreateDefaultConfig()}
		pg, host, set := buildProcessorTestGraph(t, mk([]component.ID{procA}), connCfgs)
		rcvr := receiverComponent(pg, in)

		applied, err := updateProcessors(t, pg, host, set, mk([]component.ID{procB}))
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Zero(t, rcvr.stops, "graph must not be modified")

		// The pipeline feeding the connector can reload its own processors.
		applied, err = updateProcessors(t, pg, host, set, pipelines.Config{
			in:  {Receivers: []component.ID{testReceiverID}, Processors: []component.ID{procA}, Exporters: []component.ID{connID}},
			out: mk([]component.ID{procA})[out],
		})
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Equal(t, 1, rcvr.stops)
	})
}

func TestUpdate_ProcessorsBuildFailure(t *testing.T) {
	cfg := processorTestPipelines([]component.ID{procA}, nil)
	pg, host, set := buildProcessorTestGraph(t, cfg, nil)

	set.PipelineConfigs = processorTestPipelines([]component.ID{procA}, nil)
	// A missing factory fails the build.
	applied, err := pg.Update(context.Background(), set, UpdateRequest{
		ChangedProcessors:  map[component.ID]bool{procA: true},
		NewProcessorCfgs:   testProcessorCfgs(),
		ProcessorFactories: map[component.Type]processor.Factory{},
		NewReceiverCfgs:    testReceiverCfgs(),
		ReceiverFactories:  processorTestReceiverFactories,
	}, host)
	require.Error(t, err)
	assert.True(t, applied, "the graph was modified, so the caller must not reuse it")

	// Shutdown skips nodes without a component.
	require.NotPanics(t, func() {
		_ = pg.ShutdownAll(context.Background(), status.NewNopStatusReporter())
	})
}

func receiverNodeFor(t *testing.T, pg *Graph, pid pipeline.ID, id component.ID) *testReceiver {
	t.Helper()
	for _, n := range pg.pipelines[pid].receivers {
		if rn, ok := n.(*receiverNode); ok && rn.componentID == id {
			return rn.Component.(*testReceiver)
		}
	}
	require.Failf(t, "receiver not found", "%s in %s", id, pid)
	return nil
}

func TestUpdate_ReceiversAndProcessorsTogether(t *testing.T) {
	mk := func(tracesRecv []component.ID, tracesProcs []component.ID) pipelines.Config {
		return pipelines.Config{
			tracesID: {
				Receivers:  tracesRecv,
				Processors: tracesProcs,
				Exporters:  []component.ID{component.MustNewID("exampleexporter")},
			},
			logsID: {
				Receivers:  []component.ID{testReceiverID2},
				Processors: []component.ID{procC},
				Exporters:  []component.ID{component.MustNewID("exampleexporter")},
			},
		}
	}
	update := func(pg *Graph, host *Host, set Settings, cfg pipelines.Config, changedRecv, changedProc []component.ID) (bool, error) {
		set.PipelineConfigs = cfg
		req := UpdateRequest{
			ChangedReceivers:   map[component.ID]bool{},
			ChangedProcessors:  map[component.ID]bool{},
			NewReceiverCfgs:    testReceiverCfgs(),
			ReceiverFactories:  processorTestReceiverFactories,
			NewProcessorCfgs:   testProcessorCfgs(),
			ProcessorFactories: testProcessorFactories,
		}
		for _, id := range changedRecv {
			req.ChangedReceivers[id] = true
		}
		for _, id := range changedProc {
			req.ChangedProcessors[id] = true
		}
		return pg.Update(context.Background(), set, req, host)
	}

	t.Run("changes_in_different_pipelines", func(t *testing.T) {
		cfg := mk([]component.ID{testReceiverID}, []component.ID{procA})
		pg, host, set := buildProcessorTestGraph(t, cfg, nil)
		tracesRcvr := receiverNodeFor(t, pg, tracesID, testReceiverID)
		logsRcvr := receiverNodeFor(t, pg, logsID, testReceiverID2)
		tracesProc := processorComponents(pg, tracesID)[0]
		logsProc := processorComponents(pg, logsID)[0]

		// Processor change in traces, receiver change in logs.
		applied, err := update(pg, host, set, cfg, []component.ID{testReceiverID2}, []component.ID{procA})
		require.NoError(t, err)
		assert.True(t, applied)

		// Traces: the processor and its receiver restart once.
		assert.NotSame(t, tracesProc, processorComponents(pg, tracesID)[0])
		assert.Equal(t, 1, tracesRcvr.stops)
		newTracesRcvr := receiverNodeFor(t, pg, tracesID, testReceiverID)
		assert.NotSame(t, tracesRcvr, newTracesRcvr)
		assert.Equal(t, 1, newTracesRcvr.starts)

		// Logs: only the receiver restarts.
		assert.Equal(t, 1, logsRcvr.stops)
		assert.NotSame(t, logsRcvr, receiverNodeFor(t, pg, logsID, testReceiverID2))
		assert.Same(t, logsProc, processorComponents(pg, logsID)[0])
		assert.False(t, logsProc.Stopped(), "logs processor must not be restarted")
	})

	t.Run("same_receiver_changes_with_its_pipeline_processor", func(t *testing.T) {
		cfg := mk([]component.ID{testReceiverID}, []component.ID{procA})
		pg, host, set := buildProcessorTestGraph(t, cfg, nil)
		oldRcvr := receiverNodeFor(t, pg, tracesID, testReceiverID)

		applied, err := update(pg, host, set, cfg, []component.ID{testReceiverID}, []component.ID{procA})
		require.NoError(t, err)
		assert.True(t, applied)

		// The receiver is restarted once, not once per reason.
		assert.Equal(t, 1, oldRcvr.stops)
		newRcvr := receiverNodeFor(t, pg, tracesID, testReceiverID)
		assert.NotSame(t, oldRcvr, newRcvr)
		assert.Equal(t, 1, newRcvr.starts)
		assert.Equal(t, 0, newRcvr.stops)

		exportedBefore := len(exporterComponent(pg).Traces)
		require.NoError(t, newRcvr.traces.ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
		assert.Len(t, exporterComponent(pg).Traces, exportedBefore+1)
	})

	t.Run("receiver_added_and_processor_replaced", func(t *testing.T) {
		pg, host, set := buildProcessorTestGraph(t, mk([]component.ID{testReceiverID}, []component.ID{procA}), nil)
		oldRcvr := receiverNodeFor(t, pg, tracesID, testReceiverID)

		newCfg := mk([]component.ID{testReceiverID, testReceiverID2}, []component.ID{procB})
		applied, err := update(pg, host, set, newCfg, nil, nil)
		require.NoError(t, err)
		assert.True(t, applied)

		// The existing receiver feeds a rebuilt pipeline, so it restarts.
		assert.Equal(t, 1, oldRcvr.stops)
		assert.Equal(t, 1, receiverNodeFor(t, pg, tracesID, testReceiverID).starts)
		assert.Equal(t, 1, receiverNodeFor(t, pg, tracesID, testReceiverID2).starts)
		assert.Len(t, processorComponents(pg, tracesID), 1)
		assert.True(t, processorComponents(pg, tracesID)[0].Started())

		exportedBefore := len(exporterComponent(pg).Traces)
		require.NoError(t, receiverNodeFor(t, pg, tracesID, testReceiverID2).traces.ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
		assert.Len(t, exporterComponent(pg).Traces, exportedBefore+1)
	})

	t.Run("receiver_removed_from_affected_pipeline", func(t *testing.T) {
		pg, host, set := buildProcessorTestGraph(t, mk([]component.ID{testReceiverID, testReceiverID2}, []component.ID{procA}), nil)
		removed := receiverNodeFor(t, pg, tracesID, testReceiverID2)
		kept := receiverNodeFor(t, pg, tracesID, testReceiverID)

		applied, err := update(pg, host, set, mk([]component.ID{testReceiverID}, []component.ID{procB}), nil, nil)
		require.NoError(t, err)
		assert.True(t, applied)

		assert.Equal(t, 1, removed.stops)
		assert.Equal(t, 1, kept.stops)
		assert.Equal(t, 1, receiverNodeFor(t, pg, tracesID, testReceiverID).starts)
		assert.Len(t, pg.pipelines[tracesID].receivers, 1)
	})
}
