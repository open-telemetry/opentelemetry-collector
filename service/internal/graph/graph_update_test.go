// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
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
		tagProcessorFactory.Type(): tagProcessorFactory,
	}
)

// tagAttribute is the resource attribute that tagProcessor appends its name to.
const tagAttribute = "processors"

// tagProcessor marks the data it handles by appending its component name to
// the tagAttribute resource attribute, so tests can tell which processors
// modified the data and in what order.
type tagProcessor struct {
	tag              string
	nextTraces       consumer.Traces
	nextLogs         consumer.Logs
	started, stopped bool
}

func (p *tagProcessor) Start(context.Context, component.Host) error { p.started = true; return nil }
func (p *tagProcessor) Shutdown(context.Context) error              { p.stopped = true; return nil }
func (p *tagProcessor) Started() bool                               { return p.started }
func (p *tagProcessor) Stopped() bool                               { return p.stopped }
func (*tagProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (p *tagProcessor) tagResource(attrs pcommon.Map) {
	if v, ok := attrs.Get(tagAttribute); ok {
		v.Slice().AppendEmpty().SetStr(p.tag)
		return
	}
	attrs.PutEmptySlice(tagAttribute).AppendEmpty().SetStr(p.tag)
}

func (p *tagProcessor) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	for _, rs := range td.ResourceSpans().All() {
		p.tagResource(rs.Resource().Attributes())
	}
	return p.nextTraces.ConsumeTraces(ctx, td)
}

func (p *tagProcessor) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	for _, rl := range ld.ResourceLogs().All() {
		p.tagResource(rl.Resource().Attributes())
	}
	return p.nextLogs.ConsumeLogs(ctx, ld)
}

var tagProcessorFactory = processor.NewFactory(testcomponents.ExampleProcessorFactory.Type(),
	func() component.Config { return &struct{ _ int }{} },
	processor.WithTraces(func(_ context.Context, set processor.Settings, _ component.Config, next consumer.Traces) (processor.Traces, error) {
		return &tagProcessor{tag: set.ID.Name(), nextTraces: next}, nil
	}, component.StabilityLevelDevelopment),
	processor.WithLogs(func(_ context.Context, set processor.Settings, _ component.Config, next consumer.Logs) (processor.Logs, error) {
		return &tagProcessor{tag: set.ID.Name(), nextLogs: next}, nil
	}, component.StabilityLevelDevelopment),
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
		cfgs[id] = tagProcessorFactory.CreateDefaultConfig()
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

func processorComponents(pg *Graph, pid pipeline.ID) []*tagProcessor {
	var out []*tagProcessor
	for _, n := range pg.pipelines[pid].processors {
		out = append(out, n.(*processorNode).Component.(*tagProcessor))
	}
	return out
}

func receiverComponent(pg *Graph, pid pipeline.ID) *testReceiver {
	for _, n := range pg.pipelines[pid].receivers {
		return n.(*receiverNode).Component.(*testReceiver)
	}
	return nil
}

// processedBy sends traces through the given receiver consumer and returns the
// names of the processors that modified the data that reached the exporter, in
// order. It is empty when no processor touched the data.
func processedBy(t *testing.T, pg *Graph, c consumer.Traces) []string {
	t.Helper()
	exp := exporterComponent(pg)
	before := len(exp.Traces)
	require.NoError(t, c.ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
	require.Len(t, exp.Traces, before+1)

	v, ok := exp.Traces[before].ResourceSpans().At(0).Resource().Attributes().Get(tagAttribute)
	if !ok {
		return nil
	}
	var names []string
	for _, e := range v.Slice().All() {
		names = append(names, e.Str())
	}
	return names
}

// processorNames returns the names of the given processor IDs.
func processorNames(ids ...component.ID) []string {
	var names []string
	for _, id := range ids {
		names = append(names, id.Name())
	}
	return names
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

	// The logs receiver has the same ID, so it may share an instance and is
	// restarted too. The logs processors and the exporter keep running.
	assert.NotSame(t, logsRcvr, receiverComponent(pg, logsID))
	assert.Equal(t, 1, logsRcvr.stops)
	assert.Same(t, logsProcs[0], processorComponents(pg, logsID)[0])
	assert.False(t, logsProcs[0].Stopped())
	assert.Equal(t, expStoppedBefore, exp.Stopped())

	// Data flows through the rebuilt pipeline and is modified by both processors.
	assert.Equal(t, processorNames(procA, procB), processedBy(t, pg, receiverComponent(pg, tracesID).traces))
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

	// The new first processor and the kept second one both modify the data.
	assert.Equal(t, processorNames(procA, procB), processedBy(t, pg, receiverComponent(pg, tracesID).traces))
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

			// The data is modified by exactly the processors on the new list, in order.
			assert.Equal(t, processorNames(tt.newProcs...), processedBy(t, pg, receiverComponent(pg, tracesID).traces))
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
	mk := func(tracesRecv, tracesProcs []component.ID) pipelines.Config {
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

	t.Run("receiver_with_other_id_untouched", func(t *testing.T) {
		cfg := mk([]component.ID{testReceiverID}, []component.ID{procA})
		pg, host, set := buildProcessorTestGraph(t, cfg, nil)
		logsRcvr := receiverNodeFor(t, pg, logsID, testReceiverID2)

		applied, err := update(pg, host, set, cfg, nil, []component.ID{procA})
		require.NoError(t, err)
		assert.True(t, applied)

		assert.Zero(t, logsRcvr.stops)
		assert.Same(t, logsRcvr, receiverNodeFor(t, pg, logsID, testReceiverID2))
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

		assert.Equal(t, processorNames(procA), processedBy(t, pg, newRcvr.traces))
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

		// Both receivers feed the replaced processor, not the old one.
		assert.Equal(t, processorNames(procB), processedBy(t, pg, receiverNodeFor(t, pg, tracesID, testReceiverID2).traces))
		assert.Equal(t, processorNames(procB), processedBy(t, pg, receiverNodeFor(t, pg, tracesID, testReceiverID).traces))
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

// faultyProcessor fails Start or Shutdown on demand.
type faultyProcessor struct {
	consumer.Traces
	startErr, shutdownErr error
}

func (p *faultyProcessor) Start(context.Context, component.Host) error { return p.startErr }
func (p *faultyProcessor) Shutdown(context.Context) error              { return p.shutdownErr }
func (*faultyProcessor) Capabilities() consumer.Capabilities           { return consumer.Capabilities{} }

// faultyProcessorFactories returns factories whose processors fail with the given errors.
func faultyProcessorFactories(startErr, shutdownErr error) map[component.Type]processor.Factory {
	f := processor.NewFactory(testcomponents.ExampleProcessorFactory.Type(),
		func() component.Config { return &struct{ _ int }{} },
		processor.WithTraces(func(_ context.Context, _ processor.Settings, _ component.Config, next consumer.Traces) (processor.Traces, error) {
			return &faultyProcessor{Traces: next, startErr: startErr, shutdownErr: shutdownErr}, nil
		}, component.StabilityLevelDevelopment),
	)
	return map[component.Type]processor.Factory{f.Type(): f}
}

func TestUpdate_ProcessorLifecycleErrors(t *testing.T) {
	cfg := processorTestPipelines([]component.ID{procA}, nil)
	update := func(pg *Graph, host *Host, set Settings, factories map[component.Type]processor.Factory) (bool, error) {
		set.PipelineConfigs = cfg
		return pg.Update(context.Background(), set, UpdateRequest{
			ChangedProcessors:  map[component.ID]bool{procA: true},
			NewProcessorCfgs:   testProcessorCfgs(),
			ProcessorFactories: factories,
			NewReceiverCfgs:    testReceiverCfgs(),
			ReceiverFactories:  processorTestReceiverFactories,
		}, host)
	}

	t.Run("start_fails", func(t *testing.T) {
		pg, host, set := buildProcessorTestGraph(t, cfg, nil)

		applied, err := update(pg, host, set, faultyProcessorFactories(errors.New("start boom"), nil))
		require.ErrorContains(t, err, "failed to start processor")
		assert.True(t, applied)
	})

	t.Run("shutdown_fails", func(t *testing.T) {
		pg, host, set := buildProcessorTestGraph(t, cfg, nil)

		// The first reload installs a processor that fails to shut down.
		factories := faultyProcessorFactories(nil, errors.New("shutdown boom"))
		applied, err := update(pg, host, set, factories)
		require.NoError(t, err)
		require.True(t, applied)

		// The second reload shuts it down.
		applied, err = update(pg, host, set, factories)
		require.ErrorContains(t, err, "failed to shutdown processor")
		assert.True(t, applied)
	})
}

func TestUpdate_PipelineMissingFromGraph(t *testing.T) {
	pg, host, set := buildProcessorTestGraph(t, processorTestPipelines([]component.ID{procA}, nil), nil)

	// Same number of pipelines, but a different one.
	set.PipelineConfigs = pipelines.Config{
		logsID: {
			Receivers:  []component.ID{testReceiverID},
			Processors: []component.ID{procA},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}
	applied, err := pg.Update(context.Background(), set, UpdateRequest{}, host)
	require.NoError(t, err)
	assert.False(t, applied)
}

// A processor change restarts the receivers of its pipeline. A receiver that
// shares one instance across signals must then be rebuilt for every signal.
func TestUpdate_ProcessorChangeRebuildsSharedReceiverForAllSignals(t *testing.T) {
	var instances []*sharedReceiver
	factories := map[component.Type]receiver.Factory{sharedReceiverType: newSharedReceiverFactory(&instances)}
	sharedID := component.MustNewID("sharedreceiver")
	rcvrCfgs := map[component.ID]component.Config{sharedID: &sharedReceiverConfig{id: "v1"}}

	pg, host, set := receiverTestGraph(t, rcvrCfgs, factories, sharedReceiverPipelines(sharedID, nil))
	require.Len(t, instances, 1)

	// Only the traces pipeline loses its processor.
	updated := sharedReceiverPipelines(sharedID, nil)
	updated[pipeline.NewID(pipeline.SignalTraces)].Processors = nil
	set.PipelineConfigs = updated
	applied, err := pg.Update(context.Background(), set, UpdateRequest{
		NewReceiverCfgs:   rcvrCfgs,
		ReceiverFactories: factories,
		NewProcessorCfgs: map[component.ID]component.Config{
			component.MustNewID("exampleprocessor"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
		},
		ProcessorFactories: testProcessorFactories,
	}, host)
	require.NoError(t, err)
	require.True(t, applied)

	// Both signals run on one new instance, built for both signals before it started.
	require.Len(t, instances, 2)
	assert.EqualValues(t, 1, instances[0].stopped.Load())
	rebuilt := instances[1]
	assert.EqualValues(t, 1, rebuilt.started.Load())
	assert.Equal(t, 2, rebuilt.signalsAtStart)
	assert.Same(t, rebuilt, sharedReceiverNodeComponent(t, pg, pipeline.SignalTraces, sharedID))
	assert.Same(t, rebuilt, sharedReceiverNodeComponent(t, pg, pipeline.SignalMetrics, sharedID))

	require.NoError(t, rebuilt.ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
	require.NoError(t, rebuilt.ConsumeMetrics(context.Background(), testdata.GenerateMetrics(1)))
	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}
