// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gonum.org/v1/gonum/graph/simple"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/status"
	"go.opentelemetry.io/collector/service/internal/testcomponents"
	"go.opentelemetry.io/collector/service/pipelines"
)

// TestReloadNoChange verifies that when no configs change,
// no components are restarted.
func TestReloadNoChange(t *testing.T) {
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
	}
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Capture original component references and state.
	// Note: the test component factory caches component instances by config, so
	// Stopped() may already be true from a previous test run that used the same
	// config (empty struct pointers are deduplicated by Go). We capture the state
	// before the update and verify it doesn't change.
	pipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	originalReceiver := pipe.receivers[newReceiverNode(pipeline.SignalTraces, component.MustNewID("examplereceiver")).ID()].(*receiverNode).Component
	originalProcessor := pipe.processors[0].(*processorNode).Component
	receiverStoppedBefore := originalReceiver.(*testcomponents.ExampleReceiver).Stopped()
	processorStoppedBefore := originalProcessor.(*testcomponents.ExampleProcessor).Stopped()

	// Perform the update with unchanged configs.
	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, procCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, nil, nil,
		host)
	require.NoError(t, err)
	assert.False(t, reloaded, "no changes should mean no reload")

	// With no changes, components should be the same instances (not restarted).
	newReceiver := pipe.receivers[newReceiverNode(pipeline.SignalTraces, component.MustNewID("examplereceiver")).ID()].(*receiverNode).Component
	newProcessor := pipe.processors[0].(*processorNode).Component

	assert.Same(t, originalReceiver, newReceiver, "receiver should be the same instance (no change)")
	assert.Same(t, originalProcessor, newProcessor, "processor should be the same instance (no change)")
	assert.Equal(t, receiverStoppedBefore, originalReceiver.(*testcomponents.ExampleReceiver).Stopped(),
		"receiver Stopped() state should not change")
	assert.Equal(t, processorStoppedBefore, originalProcessor.(*testcomponents.ExampleProcessor).Stopped(),
		"processor Stopped() state should not change")

	// Data still flows through the pipeline.
	allReceivers := pg.getReceivers()
	for _, c := range allReceivers[pipeline.SignalTraces] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
	}
	for _, e := range pg.GetExporters()[pipeline.SignalTraces] {
		assert.NotEmpty(t, e.(*testcomponents.ExampleExporter).Traces)
	}

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadNilHost verifies that Reload returns an error when host is nil.
func TestReloadNilHost(t *testing.T) {
	pg := &Graph{
		componentGraph: simple.NewDirectedGraph(),
		pipelines:      make(map[pipeline.ID]*pipelineNodes),
		instanceIDs:    make(map[int64]*componentstatus.InstanceID),
		telemetry:      componenttest.NewNopTelemetrySettings(),
	}
	_, err := pg.Reload(context.Background(), &Settings{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.EqualError(t, err, "host cannot be nil")
}

// TestReloadReceiverOnlyChange verifies that when only receiver config changes,
// receivers are rebuilt but processors remain untouched.
func TestReloadReceiverOnlyChange(t *testing.T) {
	type namedCfg struct{ Name string }
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): &namedCfg{Name: "original"},
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
	}
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Capture original component references and state.
	pipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	recvNodeID := newReceiverNode(pipeline.SignalTraces, component.MustNewID("examplereceiver")).ID()
	originalReceiverNode := pipe.receivers[recvNodeID].(*receiverNode)
	originalProcessor := pipe.processors[0].(*processorNode).Component
	originalExporter := pipe.exporters[newExporterNode(pipeline.SignalTraces, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component
	processorStoppedBefore := originalProcessor.(*testcomponents.ExampleProcessor).Stopped()
	exporterStoppedBefore := originalExporter.(*testcomponents.ExampleExporter).Stopped()

	// Change only the receiver config (processors unchanged).
	newRcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): &namedCfg{Name: "changed"},
	}

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, newRcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, procCfgs, // processors unchanged
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs, // exporters unchanged
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, nil, nil,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "receiver change should trigger reload")

	// Receiver node should be rebuilt (new node created).
	newReceiverNodePtr := pipe.receivers[recvNodeID].(*receiverNode)
	assert.NotSame(t, originalReceiverNode, newReceiverNodePtr, "receiverNode should be a new instance after config change")
	newReceiver := newReceiverNodePtr.Component
	assert.True(t, originalReceiverNode.Component.(*testcomponents.ExampleReceiver).Stopped(), "old receiver should be stopped")
	assert.True(t, newReceiver.(*testcomponents.ExampleReceiver).Started(), "new receiver should be started")

	// Processor should be the same instance (not rebuilt, only receiver changed).
	newProcessor := pipe.processors[0].(*processorNode).Component
	assert.Same(t, originalProcessor, newProcessor, "processor should be the same instance (no processor change)")
	assert.Equal(t, processorStoppedBefore, originalProcessor.(*testcomponents.ExampleProcessor).Stopped(),
		"processor Stopped() state should not change")

	// Exporter should be the same instance (not restarted).
	newExporter := pipe.exporters[newExporterNode(pipeline.SignalTraces, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component
	assert.Same(t, originalExporter, newExporter, "exporter should be the same instance")
	assert.Equal(t, exporterStoppedBefore, originalExporter.(*testcomponents.ExampleExporter).Stopped(),
		"exporter Stopped() state should not change")

	// Data still flows through the pipeline.
	allReceivers := pg.getReceivers()
	for _, c := range allReceivers[pipeline.SignalTraces] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
	}
	for _, e := range pg.GetExporters()[pipeline.SignalTraces] {
		assert.NotEmpty(t, e.(*testcomponents.ExampleExporter).Traces)
	}

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadSharedReceiverWithNewPipeline verifies that when a new pipeline is added
// that shares a receiver with an existing pipeline, the receiver is rebuilt.
func TestReloadSharedReceiverWithNewPipeline(t *testing.T) {
	// Start with one traces pipeline.
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"):                   testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("exampleprocessor", "second"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"):                   testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("exampleexporter", "second"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
	}
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Get the receiver's instanceID and verify it has one pipeline.
	tracePipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	var receiverNodeID int64
	for nodeID, node := range tracePipe.receivers {
		if _, ok := node.(*receiverNode); ok {
			receiverNodeID = nodeID
			break
		}
	}
	originalInstanceID := pg.instanceIDs[receiverNodeID]
	pipelineCount := 0
	originalInstanceID.AllPipelineIDs(func(_ pipeline.ID) bool {
		pipelineCount++
		return true
	})
	assert.Equal(t, 1, pipelineCount, "receiver should be in 1 pipeline initially")

	// Add a second traces pipeline that shares the same receiver.
	newPipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
		pipeline.NewIDWithName(pipeline.SignalTraces, "second"): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")}, // Same receiver!
			Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "second")},
			Exporters:  []component.ID{component.MustNewIDWithName("exampleexporter", "second")},
		},
	}
	set.PipelineConfigs = newPipelineCfgs

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, procCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, nil,
		nil,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "adding pipeline should trigger reload")

	// Verify the new pipeline exists.
	_, exists := pg.pipelines[pipeline.NewIDWithName(pipeline.SignalTraces, "second")]
	assert.True(t, exists, "second traces pipeline should exist")

	// Verify the receiver is now shared and its instanceID reflects both pipelines.
	tracePipe = pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	for nodeID, node := range tracePipe.receivers {
		if _, ok := node.(*receiverNode); ok {
			receiverNodeID = nodeID
			break
		}
	}
	newInstanceID := pg.instanceIDs[receiverNodeID]
	pipelineCount = 0
	newInstanceID.AllPipelineIDs(func(_ pipeline.ID) bool {
		pipelineCount++
		return true
	})
	assert.Equal(t, 2, pipelineCount, "receiver should be in 2 pipelines after reload")

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadProcessorChange verifies that when a processor config changes,
// processors and receivers are rebuilt but exporters remain untouched.
func TestReloadProcessorChange(t *testing.T) {
	type namedCfg struct{ Name string }
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): &namedCfg{Name: "original"},
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
	}
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Capture original component references and state.
	// Note: the test component factory caches component instances by config, so
	// Stopped() may already be true from a previous test run. We capture the state
	// before the update and verify it doesn't change for components that shouldn't
	// be touched.
	pipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	recvNodeID := newReceiverNode(pipeline.SignalTraces, component.MustNewID("examplereceiver")).ID()
	originalReceiverNode := pipe.receivers[recvNodeID].(*receiverNode)
	originalProcessor := pipe.processors[0].(*processorNode).Component
	originalExporter := pipe.exporters[newExporterNode(pipeline.SignalTraces, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component
	exporterStoppedBefore := originalExporter.(*testcomponents.ExampleExporter).Stopped()

	// Change the processor config.
	newProcCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): &namedCfg{Name: "changed"},
	}

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, newProcCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, nil, nil,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "processor change should trigger reload")

	// Processor should be a new instance (rebuilt due to config change).
	newProcessor := pipe.processors[0].(*processorNode).Component
	assert.NotSame(t, originalProcessor, newProcessor, "processor should be new instance after config change")
	assert.True(t, originalProcessor.(*testcomponents.ExampleProcessor).Stopped(), "old processor should be stopped")
	assert.True(t, newProcessor.(*testcomponents.ExampleProcessor).Started(), "new processor should be started")

	// Receiver node should be rebuilt (new node created).
	// Note: The test factory caches receivers by config, so the Component may be the same instance.
	// We verify that the receiverNode itself was recreated by checking the node pointer is different.
	newReceiverNode := pipe.receivers[recvNodeID].(*receiverNode)
	assert.NotSame(t, originalReceiverNode, newReceiverNode, "receiverNode should be a new instance after pipeline rebuild")
	newReceiver := newReceiverNode.Component
	// The receiver was stopped and restarted.
	assert.True(t, newReceiver.(*testcomponents.ExampleReceiver).Stopped(), "receiver should have been stopped during rebuild")
	assert.True(t, newReceiver.(*testcomponents.ExampleReceiver).Started(), "receiver should have been started after rebuild")

	// Exporter should be the same instance (not restarted).
	newExporter := pipe.exporters[newExporterNode(pipeline.SignalTraces, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component
	assert.Same(t, originalExporter, newExporter, "exporter should be the same instance")
	assert.Equal(t, exporterStoppedBefore, originalExporter.(*testcomponents.ExampleExporter).Stopped(),
		"exporter Stopped() state should not change")

	// Data still flows through the pipeline.
	allReceivers := pg.getReceivers()
	for _, c := range allReceivers[pipeline.SignalTraces] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
	}
	for _, e := range pg.GetExporters()[pipeline.SignalTraces] {
		assert.NotEmpty(t, e.(*testcomponents.ExampleExporter).Traces)
	}

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadMetricsPipeline verifies that processor changes in a metrics pipeline
// correctly rebuild only the affected components.
func TestReloadMetricsPipeline(t *testing.T) {
	type namedCfg struct{ Name string }
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): &namedCfg{Name: "original"},
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
	}
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalMetrics): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	pipe := pg.pipelines[pipeline.NewID(pipeline.SignalMetrics)]
	originalProcessor := pipe.processors[0].(*processorNode).Component

	// Change the processor config.
	newProcCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): &namedCfg{Name: "changed"},
	}

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, newProcCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, nil, nil,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "processor change should trigger reload")

	// Processor should be a new instance.
	newProcessor := pipe.processors[0].(*processorNode).Component
	assert.NotSame(t, originalProcessor, newProcessor, "processor should be new instance after config change")
	assert.True(t, originalProcessor.(*testcomponents.ExampleProcessor).Stopped(), "old processor should be stopped")
	assert.True(t, newProcessor.(*testcomponents.ExampleProcessor).Started(), "new processor should be started")

	// Data still flows through the pipeline.
	allReceivers := pg.getReceivers()
	for _, c := range allReceivers[pipeline.SignalMetrics] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeMetrics(context.Background(), testdata.GenerateMetrics(1)))
	}
	for _, e := range pg.GetExporters()[pipeline.SignalMetrics] {
		assert.NotEmpty(t, e.(*testcomponents.ExampleExporter).Metrics)
	}

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadLogsPipeline verifies that processor changes in a logs pipeline
// correctly rebuild only the affected components.
func TestReloadLogsPipeline(t *testing.T) {
	type namedCfg struct{ Name string }
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): &namedCfg{Name: "original"},
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
	}
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalLogs): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	pipe := pg.pipelines[pipeline.NewID(pipeline.SignalLogs)]
	originalProcessor := pipe.processors[0].(*processorNode).Component

	// Change the processor config.
	newProcCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): &namedCfg{Name: "changed"},
	}

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, newProcCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, nil, nil,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "processor change should trigger reload")

	// Processor should be a new instance.
	newProcessor := pipe.processors[0].(*processorNode).Component
	assert.NotSame(t, originalProcessor, newProcessor, "processor should be new instance after config change")
	assert.True(t, originalProcessor.(*testcomponents.ExampleProcessor).Stopped(), "old processor should be stopped")
	assert.True(t, newProcessor.(*testcomponents.ExampleProcessor).Started(), "new processor should be started")

	// Data still flows through the pipeline.
	allReceivers := pg.getReceivers()
	for _, c := range allReceivers[pipeline.SignalLogs] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeLogs(context.Background(), testdata.GenerateLogs(1)))
	}
	for _, e := range pg.GetExporters()[pipeline.SignalLogs] {
		assert.NotEmpty(t, e.(*testcomponents.ExampleExporter).Logs)
	}

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadProfilesPipeline verifies that processor changes in a profiles pipeline
// correctly rebuild only the affected components.
func TestReloadProfilesPipeline(t *testing.T) {
	type namedCfg struct{ Name string }
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): &namedCfg{Name: "original"},
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
	}
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(xpipeline.SignalProfiles): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	pipe := pg.pipelines[pipeline.NewID(xpipeline.SignalProfiles)]
	originalProcessor := pipe.processors[0].(*processorNode).Component

	// Change the processor config.
	newProcCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): &namedCfg{Name: "changed"},
	}

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, newProcCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, nil, nil,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "processor change should trigger reload")

	// Processor should be a new instance.
	newProcessor := pipe.processors[0].(*processorNode).Component
	assert.NotSame(t, originalProcessor, newProcessor, "processor should be new instance after config change")
	assert.True(t, originalProcessor.(*testcomponents.ExampleProcessor).Stopped(), "old processor should be stopped")
	assert.True(t, newProcessor.(*testcomponents.ExampleProcessor).Started(), "new processor should be started")

	// Data still flows through the pipeline.
	allReceivers := pg.getReceivers()
	for _, c := range allReceivers[xpipeline.SignalProfiles] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeProfiles(context.Background(), testdata.GenerateProfiles(1)))
	}
	for _, e := range pg.GetExporters()[xpipeline.SignalProfiles] {
		assert.NotEmpty(t, e.(*testcomponents.ExampleExporter).Profiles)
	}

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadMultiPipelineOnlyOneAffected verifies that when only one pipeline
// is affected by a change, other pipelines remain completely untouched.
func TestReloadMultiPipelineOnlyOneAffected(t *testing.T) {
	type namedCfg struct{ Name string }
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"):                    testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("examplereceiver", "metrics"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"):                    &namedCfg{Name: "traces-original"},
		component.MustNewIDWithName("exampleprocessor", "metrics"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"):                    testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("exampleexporter", "metrics"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
	}
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
		pipeline.NewID(pipeline.SignalMetrics): {
			Receivers:  []component.ID{component.MustNewIDWithName("examplereceiver", "metrics")},
			Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "metrics")},
			Exporters:  []component.ID{component.MustNewIDWithName("exampleexporter", "metrics")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Capture original component references.
	tracesPipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	metricsPipe := pg.pipelines[pipeline.NewID(pipeline.SignalMetrics)]

	tracesRecvNodeID := newReceiverNode(pipeline.SignalTraces, component.MustNewID("examplereceiver")).ID()
	originalTracesReceiverNode := tracesPipe.receivers[tracesRecvNodeID].(*receiverNode)
	originalTracesProcessor := tracesPipe.processors[0].(*processorNode).Component
	originalTracesExporter := tracesPipe.exporters[newExporterNode(pipeline.SignalTraces, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component

	originalMetricsReceiver := metricsPipe.receivers[newReceiverNode(pipeline.SignalMetrics, component.MustNewIDWithName("examplereceiver", "metrics")).ID()].(*receiverNode).Component
	originalMetricsProcessor := metricsPipe.processors[0].(*processorNode).Component
	originalMetricsExporter := metricsPipe.exporters[newExporterNode(pipeline.SignalMetrics, component.MustNewIDWithName("exampleexporter", "metrics")).ID()].(*exporterNode).Component

	// Capture stopped state for metrics pipeline components (they should not change).
	metricsReceiverStoppedBefore := originalMetricsReceiver.(*testcomponents.ExampleReceiver).Stopped()
	metricsProcessorStoppedBefore := originalMetricsProcessor.(*testcomponents.ExampleProcessor).Stopped()
	metricsExporterStoppedBefore := originalMetricsExporter.(*testcomponents.ExampleExporter).Stopped()

	// Change only the traces processor config.
	newProcCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"):                    &namedCfg{Name: "traces-changed"},
		component.MustNewIDWithName("exampleprocessor", "metrics"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, newProcCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, nil, nil,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "processor change should trigger reload")

	// Traces processor should be a new instance (rebuilt due to config change).
	newTracesProcessor := tracesPipe.processors[0].(*processorNode).Component
	assert.NotSame(t, originalTracesProcessor, newTracesProcessor, "traces processor should be new instance after config change")
	assert.True(t, originalTracesProcessor.(*testcomponents.ExampleProcessor).Stopped(), "old traces processor should be stopped")
	assert.True(t, newTracesProcessor.(*testcomponents.ExampleProcessor).Started(), "new traces processor should be started")

	// Traces receiver node should be rebuilt (new node created).
	// Note: The test factory caches receivers by config, so the Component may be the same instance.
	// We verify that the receiverNode itself was recreated by checking the node pointer is different.
	newTracesReceiverNode := tracesPipe.receivers[tracesRecvNodeID].(*receiverNode)
	assert.NotSame(t, originalTracesReceiverNode, newTracesReceiverNode, "traces receiverNode should be a new instance after pipeline rebuild")
	newTracesReceiver := newTracesReceiverNode.Component
	// The receiver was stopped and restarted.
	assert.True(t, newTracesReceiver.(*testcomponents.ExampleReceiver).Stopped(), "traces receiver should have been stopped during rebuild")
	assert.True(t, newTracesReceiver.(*testcomponents.ExampleReceiver).Started(), "traces receiver should have been started after rebuild")

	// Traces exporter should be the same instance (downstream of change).
	newTracesExporter := tracesPipe.exporters[newExporterNode(pipeline.SignalTraces, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component
	assert.Same(t, originalTracesExporter, newTracesExporter, "traces exporter should be the same instance")

	// METRICS PIPELINE SHOULD BE COMPLETELY UNTOUCHED.
	newMetricsReceiver := metricsPipe.receivers[newReceiverNode(pipeline.SignalMetrics, component.MustNewIDWithName("examplereceiver", "metrics")).ID()].(*receiverNode).Component
	newMetricsProcessor := metricsPipe.processors[0].(*processorNode).Component
	newMetricsExporter := metricsPipe.exporters[newExporterNode(pipeline.SignalMetrics, component.MustNewIDWithName("exampleexporter", "metrics")).ID()].(*exporterNode).Component

	assert.Same(t, originalMetricsReceiver, newMetricsReceiver, "metrics receiver should be the same instance (pipeline not affected)")
	assert.Same(t, originalMetricsProcessor, newMetricsProcessor, "metrics processor should be the same instance (pipeline not affected)")
	assert.Same(t, originalMetricsExporter, newMetricsExporter, "metrics exporter should be the same instance (pipeline not affected)")

	// Verify stopped state didn't change for metrics pipeline.
	assert.Equal(t, metricsReceiverStoppedBefore, originalMetricsReceiver.(*testcomponents.ExampleReceiver).Stopped(),
		"metrics receiver Stopped() state should not change")
	assert.Equal(t, metricsProcessorStoppedBefore, originalMetricsProcessor.(*testcomponents.ExampleProcessor).Stopped(),
		"metrics processor Stopped() state should not change")
	assert.Equal(t, metricsExporterStoppedBefore, originalMetricsExporter.(*testcomponents.ExampleExporter).Stopped(),
		"metrics exporter Stopped() state should not change")

	// Data still flows through both pipelines.
	allReceivers := pg.getReceivers()
	for _, c := range allReceivers[pipeline.SignalTraces] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
	}
	for _, c := range allReceivers[pipeline.SignalMetrics] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeMetrics(context.Background(), testdata.GenerateMetrics(1)))
	}
	for _, e := range pg.GetExporters()[pipeline.SignalTraces] {
		assert.NotEmpty(t, e.(*testcomponents.ExampleExporter).Traces)
	}
	for _, e := range pg.GetExporters()[pipeline.SignalMetrics] {
		assert.NotEmpty(t, e.(*testcomponents.ExampleExporter).Metrics)
	}

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadExporterChange verifies that when an exporter config changes,
// the exporter and upstream components (processors, receivers) are rebuilt.
func TestReloadExporterChange(t *testing.T) {
	type namedCfg struct{ Name string }
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): &namedCfg{Name: "original"},
	}
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Capture original component references.
	pipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	recvNodeID := newReceiverNode(pipeline.SignalTraces, component.MustNewID("examplereceiver")).ID()
	originalReceiverNode := pipe.receivers[recvNodeID].(*receiverNode)
	originalProcessor := pipe.processors[0].(*processorNode).Component
	originalExporter := pipe.exporters[newExporterNode(pipeline.SignalTraces, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component

	// Change the exporter config.
	newExpCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): &namedCfg{Name: "changed"},
	}

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, procCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, newExpCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, nil, nil,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "exporter change should trigger reload")

	// Exporter should be a new instance (rebuilt due to config change).
	newExporter := pipe.exporters[newExporterNode(pipeline.SignalTraces, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component
	assert.NotSame(t, originalExporter, newExporter, "exporter should be new instance after config change")
	assert.True(t, originalExporter.(*testcomponents.ExampleExporter).Stopped(), "old exporter should be stopped")
	assert.True(t, newExporter.(*testcomponents.ExampleExporter).Started(), "new exporter should be started")

	// Processor should be a new instance (rebuilt because processors store a reference
	// to the fanOutNode consumer, which needs to be updated when exporter changes).
	newProcessor := pipe.processors[0].(*processorNode).Component
	assert.NotSame(t, originalProcessor, newProcessor, "processor should be new instance (consumer reference updated)")
	assert.True(t, originalProcessor.(*testcomponents.ExampleProcessor).Stopped(), "old processor should be stopped")
	assert.True(t, newProcessor.(*testcomponents.ExampleProcessor).Started(), "new processor should be started")

	// Receiver node should be rebuilt (new node created).
	// Note: The test factory caches receivers by config, so the Component may be the same instance.
	// We verify that the receiverNode itself was recreated by checking the node pointer is different.
	newReceiverNodePtr := pipe.receivers[recvNodeID].(*receiverNode)
	assert.NotSame(t, originalReceiverNode, newReceiverNodePtr, "receiverNode should be a new instance after pipeline rebuild")
	newReceiver := newReceiverNodePtr.Component
	// The receiver was stopped and restarted.
	assert.True(t, newReceiver.(*testcomponents.ExampleReceiver).Stopped(), "receiver should have been stopped during rebuild")
	assert.True(t, newReceiver.(*testcomponents.ExampleReceiver).Started(), "receiver should have been started after rebuild")

	// Data still flows through the pipeline.
	allReceivers := pg.getReceivers()
	for _, c := range allReceivers[pipeline.SignalTraces] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
	}
	for _, e := range pg.GetExporters()[pipeline.SignalTraces] {
		assert.NotEmpty(t, e.(*testcomponents.ExampleExporter).Traces)
	}

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadExporterChangeMetrics verifies that exporter changes in a metrics pipeline
// correctly rebuild the exporter and upstream components.
func TestReloadExporterChangeMetrics(t *testing.T) {
	type namedCfg struct{ Name string }
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): &namedCfg{Name: "original"},
	}
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalMetrics): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	pipe := pg.pipelines[pipeline.NewID(pipeline.SignalMetrics)]
	originalExporter := pipe.exporters[newExporterNode(pipeline.SignalMetrics, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component

	// Change the exporter config.
	newExpCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): &namedCfg{Name: "changed"},
	}

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, procCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, newExpCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, nil, nil,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "exporter change should trigger reload")

	// Exporter should be a new instance.
	newExporter := pipe.exporters[newExporterNode(pipeline.SignalMetrics, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component
	assert.NotSame(t, originalExporter, newExporter, "exporter should be new instance after config change")

	// Data still flows through the pipeline.
	allReceivers := pg.getReceivers()
	for _, c := range allReceivers[pipeline.SignalMetrics] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeMetrics(context.Background(), testdata.GenerateMetrics(1)))
	}
	for _, e := range pg.GetExporters()[pipeline.SignalMetrics] {
		assert.NotEmpty(t, e.(*testcomponents.ExampleExporter).Metrics)
	}

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadExporterChangeLogs verifies that exporter changes in a logs pipeline
// correctly rebuild the exporter and upstream components.
func TestReloadExporterChangeLogs(t *testing.T) {
	type namedCfg struct{ Name string }
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): &namedCfg{Name: "original"},
	}
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalLogs): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	pipe := pg.pipelines[pipeline.NewID(pipeline.SignalLogs)]
	originalExporter := pipe.exporters[newExporterNode(pipeline.SignalLogs, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component

	// Change the exporter config.
	newExpCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): &namedCfg{Name: "changed"},
	}

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, procCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, newExpCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, nil, nil,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "exporter change should trigger reload")

	// Exporter should be a new instance.
	newExporter := pipe.exporters[newExporterNode(pipeline.SignalLogs, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component
	assert.NotSame(t, originalExporter, newExporter, "exporter should be new instance after config change")

	// Data still flows through the pipeline.
	allReceivers := pg.getReceivers()
	for _, c := range allReceivers[pipeline.SignalLogs] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeLogs(context.Background(), testdata.GenerateLogs(1)))
	}
	for _, e := range pg.GetExporters()[pipeline.SignalLogs] {
		assert.NotEmpty(t, e.(*testcomponents.ExampleExporter).Logs)
	}

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadExporterChangeProfiles verifies that exporter changes in a profiles pipeline
// correctly rebuild the exporter and upstream components.
func TestReloadExporterChangeProfiles(t *testing.T) {
	type namedCfg struct{ Name string }
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): &namedCfg{Name: "original"},
	}
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(xpipeline.SignalProfiles): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	pipe := pg.pipelines[pipeline.NewID(xpipeline.SignalProfiles)]
	originalExporter := pipe.exporters[newExporterNode(xpipeline.SignalProfiles, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component

	// Change the exporter config.
	newExpCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): &namedCfg{Name: "changed"},
	}

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, procCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, newExpCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, nil, nil,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "exporter change should trigger reload")

	// Exporter should be a new instance.
	newExporter := pipe.exporters[newExporterNode(xpipeline.SignalProfiles, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component
	assert.NotSame(t, originalExporter, newExporter, "exporter should be new instance after config change")

	// Data still flows through the pipeline.
	allReceivers := pg.getReceivers()
	for _, c := range allReceivers[xpipeline.SignalProfiles] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeProfiles(context.Background(), testdata.GenerateProfiles(1)))
	}
	for _, e := range pg.GetExporters()[xpipeline.SignalProfiles] {
		assert.NotEmpty(t, e.(*testcomponents.ExampleExporter).Profiles)
	}

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadExporterListChange verifies that adding an exporter to a pipeline's
// exporter list causes the pipeline to be rebuilt.
func TestReloadExporterListChange(t *testing.T) {
	// Start with one exporter.
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"):                   testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("exampleexporter", "second"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
	}
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Save original components.
	tracePipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	originalProcessor := tracePipe.processors[0].(*processorNode).Component
	assert.Len(t, tracePipe.exporters, 1, "should have 1 exporter initially")

	// Add second exporter to pipeline.
	newPipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter"), component.MustNewIDWithName("exampleexporter", "second")},
		},
	}
	set.PipelineConfigs = newPipelineCfgs

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, procCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, nil,
		nil,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "exporter list change should trigger reload")

	// Verify second exporter was added.
	tracePipe = pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	assert.Len(t, tracePipe.exporters, 2, "should have 2 exporters after reload")

	// Processor should have been rebuilt due to fanout change.
	newProcessor := tracePipe.processors[0].(*processorNode).Component
	assert.NotSame(t, originalProcessor, newProcessor, "processor should be rebuilt")

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadAddNewExporterToExistingPipeline verifies that adding a new exporter
// to an existing pipeline correctly rebuilds the pipeline.
func TestReloadAddNewExporterToExistingPipeline(t *testing.T) {
	// Tests adding a new exporter to an existing pipeline's exporter list.
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): struct{}{},
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): struct{}{},
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): struct{}{},
	}

	// Start with one exporter.
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Add a second exporter to the pipeline.
	newExpCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"):                   struct{}{},
		component.MustNewIDWithName("exampleexporter", "second"): struct{}{},
	}
	newPipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters: []component.ID{
				component.MustNewID("exampleexporter"),
				component.MustNewIDWithName("exampleexporter", "second"),
			},
		},
	}
	set.PipelineConfigs = newPipelineCfgs

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, procCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, newExpCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, nil,
		nil,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "adding exporter should trigger reload")

	// Verify the pipeline now has 2 exporters.
	tracePipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	exporterCount := 0
	for _, node := range tracePipe.exporters {
		if _, ok := node.(*exporterNode); ok {
			exporterCount++
		}
	}
	assert.Equal(t, 2, exporterCount, "pipeline should have 2 exporters after reload")

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadRemoveExporterFromPipeline verifies that removing an exporter from a
// pipeline causes the pipeline chain to be rebuilt.
func TestReloadRemoveExporterFromPipeline(t *testing.T) {
	// Tests that removing an exporter from a pipeline's exporter list causes
	// the pipeline chain to be rebuilt (receiver, processors rebuild to get
	// new consumer references through the updated fanout).
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): struct{}{},
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): struct{}{},
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"):                   struct{}{},
		component.MustNewIDWithName("exampleexporter", "second"): struct{}{},
	}

	// Start with two exporters.
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters: []component.ID{
				component.MustNewID("exampleexporter"),
				component.MustNewIDWithName("exampleexporter", "second"),
			},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Capture original component references.
	tracePipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	originalReceiver := tracePipe.receivers[newReceiverNode(pipeline.SignalTraces, component.MustNewID("examplereceiver")).ID()].(*receiverNode).Component
	originalProcessor := tracePipe.processors[0].(*processorNode).Component

	// Remove the second exporter from the pipeline.
	newPipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}
	set.PipelineConfigs = newPipelineCfgs

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, procCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, nil,
		nil,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "removing exporter should trigger reload")

	// The pipeline chain should be rebuilt because fanout changed.
	// Original components should be stopped.
	assert.True(t, originalReceiver.(*testcomponents.ExampleReceiver).Stopped(),
		"receiver should be stopped when exporter list changes")
	assert.True(t, originalProcessor.(*testcomponents.ExampleProcessor).Stopped(),
		"processor should be stopped when exporter list changes")

	// Pipeline still works with rebuilt components.
	allReceivers := pg.getReceivers()
	require.Contains(t, allReceivers, pipeline.SignalTraces)
	for _, c := range allReceivers[pipeline.SignalTraces] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
	}
	for _, e := range pg.GetExporters()[pipeline.SignalTraces] {
		assert.NotEmpty(t, e.(*testcomponents.ExampleExporter).Traces)
	}

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadConnectorChange verifies that when a connector config changes,
// the connector and upstream components in the source pipeline are rebuilt.
func TestReloadConnectorChange(t *testing.T) {
	type namedCfg struct{ Name string }
	connID := component.MustNewID("exampleconnector")
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
	}
	connCfgs := map[component.ID]component.Config{
		connID: &namedCfg{Name: "original"},
	}
	// traces → connector → traces (same signal type via mockforward or exampleconnector).
	pipelineCfgs := pipelines.Config{
		pipeline.NewIDWithName(pipeline.SignalTraces, "in"): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{connID},
		},
		pipeline.NewIDWithName(pipeline.SignalTraces, "out"): {
			Receivers:  []component.ID{connID},
			Processors: []component.ID{},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}

	connFactories := map[component.Type]connector.Factory{
		testcomponents.ExampleConnectorFactory.Type(): testcomponents.ExampleConnectorFactory,
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(connCfgs, connFactories),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Capture original component references.
	pipeIn := pg.pipelines[pipeline.NewIDWithName(pipeline.SignalTraces, "in")]
	pipeOut := pg.pipelines[pipeline.NewIDWithName(pipeline.SignalTraces, "out")]
	recvNodeID := newReceiverNode(pipeline.SignalTraces, component.MustNewID("examplereceiver")).ID()
	originalReceiverNode := pipeIn.receivers[recvNodeID].(*receiverNode)
	originalProcessor := pipeIn.processors[0].(*processorNode).Component
	originalExporter := pipeOut.exporters[newExporterNode(pipeline.SignalTraces, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component
	exporterStoppedBefore := originalExporter.(*testcomponents.ExampleExporter).Stopped()

	// Find the connector node.
	var originalConnector component.Component
	for _, node := range pipeIn.exporters {
		if cn, ok := node.(*connectorNode); ok {
			originalConnector = cn.Component
			break
		}
	}
	require.NotNil(t, originalConnector, "connector should exist in pipeline")

	// Change the connector config.
	newConnCfgs := map[component.ID]component.Config{
		connID: &namedCfg{Name: "changed"},
	}

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, procCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		connCfgs, newConnCfgs, connFactories,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "connector change should trigger reload")

	// Connector should be a new instance (rebuilt due to config change).
	var newConnector component.Component
	for _, node := range pipeIn.exporters {
		if cn, ok := node.(*connectorNode); ok {
			newConnector = cn.Component
			break
		}
	}
	require.NotNil(t, newConnector, "connector should exist after reload")
	assert.NotSame(t, originalConnector, newConnector, "connector should be new instance after config change")
	assert.True(t, originalConnector.(*testcomponents.ExampleConnector).Stopped(), "old connector should be stopped")
	assert.True(t, newConnector.(*testcomponents.ExampleConnector).Started(), "new connector should be started")

	// Processor in source pipeline should be rebuilt (consumer reference updated).
	newProcessor := pipeIn.processors[0].(*processorNode).Component
	assert.NotSame(t, originalProcessor, newProcessor, "processor should be new instance (consumer reference updated)")
	assert.True(t, originalProcessor.(*testcomponents.ExampleProcessor).Stopped(), "old processor should be stopped")
	assert.True(t, newProcessor.(*testcomponents.ExampleProcessor).Started(), "new processor should be started")

	// Exporter in destination pipeline should be the same instance (not rebuilt).
	newExporter := pipeOut.exporters[newExporterNode(pipeline.SignalTraces, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component
	assert.Same(t, originalExporter, newExporter, "exporter should be the same instance")
	assert.Equal(t, exporterStoppedBefore, originalExporter.(*testcomponents.ExampleExporter).Stopped(),
		"exporter Stopped() state should not change")

	// Receiver node should be rebuilt (new node created).
	// Note: The test factory caches receivers by config, so the Component may be the same instance.
	// We verify that the receiverNode itself was recreated by checking the node pointer is different.
	newReceiverNodePtr := pipeIn.receivers[recvNodeID].(*receiverNode)
	assert.NotSame(t, originalReceiverNode, newReceiverNodePtr, "receiverNode should be a new instance after pipeline rebuild")
	newReceiver := newReceiverNodePtr.Component
	// The receiver was stopped and restarted.
	assert.True(t, newReceiver.(*testcomponents.ExampleReceiver).Stopped(), "receiver should have been stopped during rebuild")
	assert.True(t, newReceiver.(*testcomponents.ExampleReceiver).Started(), "receiver should have been started after rebuild")

	// Data still flows through the pipeline.
	allReceivers := pg.getReceivers()
	for _, c := range allReceivers[pipeline.SignalTraces] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
	}
	for _, e := range pg.GetExporters()[pipeline.SignalTraces] {
		assert.NotEmpty(t, e.(*testcomponents.ExampleExporter).Traces)
	}

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadConnectorAddition verifies that adding a connector between two
// existing pipelines correctly creates and wires up the connector.
func TestReloadConnectorAddition(t *testing.T) {
	// Start with two independent pipelines: traces and logs.
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"):                 testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("examplereceiver", "logs"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"):                 testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("exampleprocessor", "logs"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"):                 testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("exampleexporter", "logs"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
	}
	connCfgs := map[component.ID]component.Config{}
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
		pipeline.NewID(pipeline.SignalLogs): {
			Receivers:  []component.ID{component.MustNewIDWithName("examplereceiver", "logs")},
			Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "logs")},
			Exporters:  []component.ID{component.MustNewIDWithName("exampleexporter", "logs")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(connCfgs, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Save original components to verify which ones get rebuilt.
	tracePipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	logsPipe := pg.pipelines[pipeline.NewID(pipeline.SignalLogs)]
	originalTraceProcessor := tracePipe.processors[0].(*processorNode).Component
	originalLogsProcessor := logsPipe.processors[0].(*processorNode).Component

	// Now add a connector from traces to logs.
	newConnCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleconnector"): testcomponents.ExampleConnectorFactory.CreateDefaultConfig(),
	}
	newPipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			// Add connector as exporter.
			Exporters: []component.ID{component.MustNewID("exampleexporter"), component.MustNewID("exampleconnector")},
		},
		pipeline.NewID(pipeline.SignalLogs): {
			// Add connector as receiver.
			Receivers:  []component.ID{component.MustNewIDWithName("examplereceiver", "logs"), component.MustNewID("exampleconnector")},
			Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "logs")},
			Exporters:  []component.ID{component.MustNewIDWithName("exampleexporter", "logs")},
		},
	}
	set.PipelineConfigs = newPipelineCfgs

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, procCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		connCfgs, newConnCfgs,
		map[component.Type]connector.Factory{testcomponents.ExampleConnectorFactory.Type(): testcomponents.ExampleConnectorFactory},
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "connector addition should trigger reload")

	// Verify the connector exists in both pipelines.
	tracePipe = pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	logsPipe = pg.pipelines[pipeline.NewID(pipeline.SignalLogs)]

	// Traces pipeline should have the connector as exporter.
	hasConnectorAsExporter := false
	for _, node := range tracePipe.exporters {
		if cn, ok := node.(*connectorNode); ok && cn.componentID == component.MustNewID("exampleconnector") {
			hasConnectorAsExporter = true
			break
		}
	}
	assert.True(t, hasConnectorAsExporter, "traces pipeline should have connector as exporter")

	// Logs pipeline should have the connector as receiver.
	hasConnectorAsReceiver := false
	for _, node := range logsPipe.receivers {
		if cn, ok := node.(*connectorNode); ok && cn.componentID == component.MustNewID("exampleconnector") {
			hasConnectorAsReceiver = true
			break
		}
	}
	assert.True(t, hasConnectorAsReceiver, "logs pipeline should have connector as receiver")

	// Traces pipeline processor should have been rebuilt (due to fanout change).
	newTraceProcessor := tracePipe.processors[0].(*processorNode).Component
	assert.NotSame(t, originalTraceProcessor, newTraceProcessor, "traces processor should be rebuilt")
	assert.True(t, originalTraceProcessor.(*testcomponents.ExampleProcessor).Stopped(), "old traces processor should be stopped")

	// Logs pipeline processor should NOT have been rebuilt.
	newLogsProcessor := logsPipe.processors[0].(*processorNode).Component
	assert.Same(t, originalLogsProcessor, newLogsProcessor, "logs processor should NOT be rebuilt")

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadConnectorRemoval verifies that removing a connector between two
// pipelines correctly shuts down and removes the connector.
func TestReloadConnectorRemoval(t *testing.T) {
	// Start with two pipelines connected by a connector.
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"):                 testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("examplereceiver", "logs"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"):                 testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("exampleprocessor", "logs"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"):                 testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("exampleexporter", "logs"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
	}
	connCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleconnector"): testcomponents.ExampleConnectorFactory.CreateDefaultConfig(),
	}
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter"), component.MustNewID("exampleconnector")},
		},
		pipeline.NewID(pipeline.SignalLogs): {
			Receivers:  []component.ID{component.MustNewIDWithName("examplereceiver", "logs"), component.MustNewID("exampleconnector")},
			Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "logs")},
			Exporters:  []component.ID{component.MustNewIDWithName("exampleexporter", "logs")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(
			connCfgs,
			map[component.Type]connector.Factory{
				testcomponents.ExampleConnectorFactory.Type(): testcomponents.ExampleConnectorFactory,
			},
		),
		PipelineConfigs: pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Verify connector exists initially.
	tracePipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	hasConnector := false
	for _, node := range tracePipe.exporters {
		if _, ok := node.(*connectorNode); ok {
			hasConnector = true
			break
		}
	}
	require.True(t, hasConnector, "traces pipeline should have connector initially")

	// Save original components.
	originalTraceProcessor := tracePipe.processors[0].(*processorNode).Component

	// Now remove the connector.
	newConnCfgs := map[component.ID]component.Config{}
	newPipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
		pipeline.NewID(pipeline.SignalLogs): {
			Receivers:  []component.ID{component.MustNewIDWithName("examplereceiver", "logs")},
			Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "logs")},
			Exporters:  []component.ID{component.MustNewIDWithName("exampleexporter", "logs")},
		},
	}
	set.PipelineConfigs = newPipelineCfgs

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, procCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		connCfgs, newConnCfgs,
		map[component.Type]connector.Factory{testcomponents.ExampleConnectorFactory.Type(): testcomponents.ExampleConnectorFactory},
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "connector removal should trigger reload")

	// Verify connector is removed.
	tracePipe = pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	logsPipe := pg.pipelines[pipeline.NewID(pipeline.SignalLogs)]

	hasConnectorAsExporter := false
	for _, node := range tracePipe.exporters {
		if _, ok := node.(*connectorNode); ok {
			hasConnectorAsExporter = true
			break
		}
	}
	assert.False(t, hasConnectorAsExporter, "traces pipeline should not have connector")

	hasConnectorAsReceiver := false
	for _, node := range logsPipe.receivers {
		if _, ok := node.(*connectorNode); ok {
			hasConnectorAsReceiver = true
			break
		}
	}
	assert.False(t, hasConnectorAsReceiver, "logs pipeline should not have connector")

	// Traces pipeline processor should have been rebuilt.
	newTraceProcessor := tracePipe.processors[0].(*processorNode).Component
	assert.NotSame(t, originalTraceProcessor, newTraceProcessor, "traces processor should be rebuilt")

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadWithConnectorBetweenPipelines verifies that when a connector bridges
// two pipelines of different signal types, changing the connector rebuilds it properly.
func TestReloadWithConnectorBetweenPipelines(t *testing.T) {
	type namedCfg struct{ Name string }
	connID := component.MustNewID("exampleconnector")
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
	}
	connCfgs := map[component.ID]component.Config{
		connID: &namedCfg{Name: "original"},
	}
	// traces → connector → metrics (different signal types).
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{connID},
		},
		pipeline.NewID(pipeline.SignalMetrics): {
			Receivers:  []component.ID{connID},
			Processors: []component.ID{},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}

	connFactories := map[component.Type]connector.Factory{
		testcomponents.ExampleConnectorFactory.Type(): testcomponents.ExampleConnectorFactory,
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(connCfgs, connFactories),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Capture original component references.
	tracesPipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	metricsPipe := pg.pipelines[pipeline.NewID(pipeline.SignalMetrics)]

	recvNodeID := newReceiverNode(pipeline.SignalTraces, component.MustNewID("examplereceiver")).ID()
	originalReceiverNode := tracesPipe.receivers[recvNodeID].(*receiverNode)
	originalProcessor := tracesPipe.processors[0].(*processorNode).Component
	originalExporter := metricsPipe.exporters[newExporterNode(pipeline.SignalMetrics, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component
	exporterStoppedBefore := originalExporter.(*testcomponents.ExampleExporter).Stopped()

	// Find the connector node (it appears as exporter in traces pipeline).
	var originalConnector component.Component
	for _, node := range tracesPipe.exporters {
		if cn, ok := node.(*connectorNode); ok {
			originalConnector = cn.Component
			break
		}
	}
	require.NotNil(t, originalConnector, "connector should exist in traces pipeline as exporter")

	// Change the connector config.
	newConnCfgs := map[component.ID]component.Config{
		connID: &namedCfg{Name: "changed"},
	}

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, procCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		connCfgs, newConnCfgs, connFactories,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "connector change should trigger reload")

	// Connector should be a new instance (rebuilt due to config change).
	var newConnector component.Component
	for _, node := range tracesPipe.exporters {
		if cn, ok := node.(*connectorNode); ok {
			newConnector = cn.Component
			break
		}
	}
	require.NotNil(t, newConnector, "connector should exist after reload")
	assert.NotSame(t, originalConnector, newConnector, "connector should be new instance after config change")
	assert.True(t, originalConnector.(*testcomponents.ExampleConnector).Stopped(), "old connector should be stopped")
	assert.True(t, newConnector.(*testcomponents.ExampleConnector).Started(), "new connector should be started")

	// Processor in traces pipeline should be rebuilt (consumer reference updated).
	newProcessor := tracesPipe.processors[0].(*processorNode).Component
	assert.NotSame(t, originalProcessor, newProcessor, "processor should be new instance (consumer reference updated)")
	assert.True(t, originalProcessor.(*testcomponents.ExampleProcessor).Stopped(), "old processor should be stopped")
	assert.True(t, newProcessor.(*testcomponents.ExampleProcessor).Started(), "new processor should be started")

	// Exporter in metrics pipeline should be the same instance (not rebuilt).
	newExporter := metricsPipe.exporters[newExporterNode(pipeline.SignalMetrics, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component
	assert.Same(t, originalExporter, newExporter, "exporter should be the same instance")
	assert.Equal(t, exporterStoppedBefore, originalExporter.(*testcomponents.ExampleExporter).Stopped(),
		"exporter Stopped() state should not change")

	// Receiver node should be rebuilt (new node created).
	// Note: The test factory caches receivers by config, so the Component may be the same instance.
	// We verify that the receiverNode itself was recreated by checking the node pointer is different.
	newReceiverNodePtr := tracesPipe.receivers[recvNodeID].(*receiverNode)
	assert.NotSame(t, originalReceiverNode, newReceiverNodePtr, "receiverNode should be a new instance after pipeline rebuild")
	newReceiver := newReceiverNodePtr.Component
	// The receiver was stopped and restarted.
	assert.True(t, newReceiver.(*testcomponents.ExampleReceiver).Stopped(), "receiver should have been stopped during rebuild")
	assert.True(t, newReceiver.(*testcomponents.ExampleReceiver).Started(), "receiver should have been started after rebuild")

	// Data still flows through the pipeline (traces → connector → metrics).
	allReceivers := pg.getReceivers()
	for _, c := range allReceivers[pipeline.SignalTraces] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
	}
	// The connector converts traces to metrics, so check metrics exporter.
	for _, e := range pg.GetExporters()[pipeline.SignalMetrics] {
		assert.NotEmpty(t, e.(*testcomponents.ExampleExporter).Metrics, "metrics should flow through connector")
	}

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadWithConnectorAsReceiverInPipeline verifies that when a pipeline has
// a connector as a receiver, it is properly handled during reload.
func TestReloadWithConnectorAsReceiverInPipeline(t *testing.T) {
	// Tests that when a pipeline has a connector as a receiver,
	// the connector is properly skipped in receiver-related logic.
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): struct{}{},
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"):                   struct{}{},
		component.MustNewIDWithName("exampleprocessor", "second"): struct{}{},
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): struct{}{},
	}
	connCfgs := map[component.ID]component.Config{
		component.MustNewIDWithName("exampleconnector", "con"): struct{}{},
	}

	// Setup: traces -> connector -> logs pipeline.
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "con")},
		},
		pipeline.NewID(pipeline.SignalLogs): {
			Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "con")}, // Connector as receiver
			Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "second")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(
			connCfgs,
			map[component.Type]connector.Factory{
				testcomponents.ExampleConnectorFactory.Type(): testcomponents.ExampleConnectorFactory,
			},
		),
		PipelineConfigs: pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Change processor config in logs pipeline.
	// This should affect the logs pipeline but the connector should be handled correctly.
	newProcCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"):                   struct{}{},
		component.MustNewIDWithName("exampleprocessor", "second"): &struct{ Modified bool }{Modified: true},
	}
	set.ProcessorBuilder = builders.NewProcessor(
		newProcCfgs,
		map[component.Type]processor.Factory{
			testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
		},
	)

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, newProcCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		connCfgs, connCfgs,
		map[component.Type]connector.Factory{testcomponents.ExampleConnectorFactory.Type(): testcomponents.ExampleConnectorFactory},
		host)
	require.NoError(t, err)
	assert.True(t, reloaded)

	// Verify both pipelines still exist.
	assert.Len(t, pg.pipelines, 2)
	_, exists := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	assert.True(t, exists)
	_, exists = pg.pipelines[pipeline.NewID(pipeline.SignalLogs)]
	assert.True(t, exists)

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadAddNewConnectorBetweenExistingPipelines verifies that adding a new
// connector between two existing pipelines correctly wires up both pipelines.
func TestReloadAddNewConnectorBetweenExistingPipelines(t *testing.T) {
	// Tests adding a new connector between two existing pipelines.
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"):                   struct{}{},
		component.MustNewIDWithName("examplereceiver", "second"): struct{}{},
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"):                   struct{}{},
		component.MustNewIDWithName("exampleprocessor", "second"): struct{}{},
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"):                   struct{}{},
		component.MustNewIDWithName("exampleexporter", "second"): struct{}{},
	}

	// Start with two separate pipelines (no connector).
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
		pipeline.NewID(pipeline.SignalLogs): {
			Receivers:  []component.ID{component.MustNewIDWithName("examplereceiver", "second")},
			Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "second")},
			Exporters:  []component.ID{component.MustNewIDWithName("exampleexporter", "second")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Add a connector between the traces and logs pipelines.
	connCfgs := map[component.ID]component.Config{
		component.MustNewIDWithName("exampleconnector", "con"): struct{}{},
	}
	newPipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters: []component.ID{
				component.MustNewID("exampleexporter"),
				component.MustNewIDWithName("exampleconnector", "con"), // Add connector
			},
		},
		pipeline.NewID(pipeline.SignalLogs): {
			Receivers: []component.ID{
				component.MustNewIDWithName("examplereceiver", "second"),
				component.MustNewIDWithName("exampleconnector", "con"), // Connector as receiver
			},
			Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "second")},
			Exporters:  []component.ID{component.MustNewIDWithName("exampleexporter", "second")},
		},
	}
	set.PipelineConfigs = newPipelineCfgs

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, procCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, connCfgs,
		map[component.Type]connector.Factory{testcomponents.ExampleConnectorFactory.Type(): testcomponents.ExampleConnectorFactory},
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "adding connector should trigger reload")

	// Verify the traces pipeline has the connector as exporter.
	tracePipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	hasConnectorAsExporter := false
	for _, node := range tracePipe.exporters {
		if _, ok := node.(*connectorNode); ok {
			hasConnectorAsExporter = true
			break
		}
	}
	assert.True(t, hasConnectorAsExporter, "traces pipeline should have connector as exporter")

	// Verify the logs pipeline has the connector as receiver.
	logsPipe := pg.pipelines[pipeline.NewID(pipeline.SignalLogs)]
	hasConnectorAsReceiver := false
	for _, node := range logsPipe.receivers {
		if _, ok := node.(*connectorNode); ok {
			hasConnectorAsReceiver = true
			break
		}
	}
	assert.True(t, hasConnectorAsReceiver, "logs pipeline should have connector as receiver")

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadPipelineAddition verifies that adding a new pipeline during reload
// correctly creates and starts all components in the new pipeline.
func TestReloadPipelineAddition(t *testing.T) {
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
	}
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Capture original component references.
	pipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	originalReceiver := pipe.receivers[newReceiverNode(pipeline.SignalTraces, component.MustNewID("examplereceiver")).ID()].(*receiverNode).Component
	originalProcessor := pipe.processors[0].(*processorNode).Component
	originalExporter := pipe.exporters[newExporterNode(pipeline.SignalTraces, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component
	originalReceiverStopped := originalReceiver.(*testcomponents.ExampleReceiver).Stopped()
	originalProcessorStopped := originalProcessor.(*testcomponents.ExampleProcessor).Stopped()
	originalExporterStopped := originalExporter.(*testcomponents.ExampleExporter).Stopped()

	// Add a new metrics pipeline.
	newPipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
		pipeline.NewID(pipeline.SignalMetrics): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}
	set.PipelineConfigs = newPipelineCfgs

	// Perform the reload.
	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, procCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, nil, nil,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "pipeline addition should trigger reload")

	// Verify the new pipeline was added.
	require.Contains(t, pg.pipelines, pipeline.NewID(pipeline.SignalMetrics))
	newPipe := pg.pipelines[pipeline.NewID(pipeline.SignalMetrics)]
	assert.NotNil(t, newPipe.capabilitiesNode)
	assert.NotNil(t, newPipe.fanOutNode)
	assert.Len(t, newPipe.processors, 1)

	// Verify original traces pipeline components are unchanged.
	tracePipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	// The original receiver/processor/exporter references may change due to shared nodes being rebuilt,
	// but we can verify they weren't stopped unnecessarily.
	traceReceiver := tracePipe.receivers[newReceiverNode(pipeline.SignalTraces, component.MustNewID("examplereceiver")).ID()].(*receiverNode).Component
	traceProcessor := tracePipe.processors[0].(*processorNode).Component
	traceExporter := tracePipe.exporters[newExporterNode(pipeline.SignalTraces, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component

	// Original components should not have been stopped if they were untouched.
	assert.Equal(t, originalReceiverStopped, originalReceiver.(*testcomponents.ExampleReceiver).Stopped(),
		"original receiver Stopped() should not change")
	assert.Equal(t, originalProcessorStopped, originalProcessor.(*testcomponents.ExampleProcessor).Stopped(),
		"original processor Stopped() should not change")
	assert.Equal(t, originalExporterStopped, originalExporter.(*testcomponents.ExampleExporter).Stopped(),
		"original exporter Stopped() should not change")

	// Current references should be valid components.
	assert.NotNil(t, traceReceiver)
	assert.NotNil(t, traceProcessor)
	assert.NotNil(t, traceExporter)

	// Data flows through both pipelines.
	allReceivers := pg.getReceivers()
	for _, c := range allReceivers[pipeline.SignalTraces] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
	}
	for _, c := range allReceivers[pipeline.SignalMetrics] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeMetrics(context.Background(), testdata.GenerateMetrics(1)))
	}

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadPipelineRemoval verifies that removing a pipeline during reload
// correctly shuts down and removes all components in the removed pipeline.
func TestReloadPipelineRemoval(t *testing.T) {
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
	}
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
		pipeline.NewID(pipeline.SignalMetrics): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Verify both pipelines exist.
	require.Contains(t, pg.pipelines, pipeline.NewID(pipeline.SignalTraces))
	require.Contains(t, pg.pipelines, pipeline.NewID(pipeline.SignalMetrics))

	// Capture traces pipeline component references.
	tracePipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	originalTraceReceiver := tracePipe.receivers[newReceiverNode(pipeline.SignalTraces, component.MustNewID("examplereceiver")).ID()].(*receiverNode).Component
	originalTraceProcessor := tracePipe.processors[0].(*processorNode).Component
	originalTraceExporter := tracePipe.exporters[newExporterNode(pipeline.SignalTraces, component.MustNewID("exampleexporter")).ID()].(*exporterNode).Component
	originalTraceReceiverStopped := originalTraceReceiver.(*testcomponents.ExampleReceiver).Stopped()
	originalTraceProcessorStopped := originalTraceProcessor.(*testcomponents.ExampleProcessor).Stopped()
	originalTraceExporterStopped := originalTraceExporter.(*testcomponents.ExampleExporter).Stopped()

	// Remove the metrics pipeline.
	newPipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}
	set.PipelineConfigs = newPipelineCfgs

	// Perform the reload.
	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, procCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, nil, nil,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "pipeline removal should trigger reload")

	// Verify the metrics pipeline was removed.
	require.NotContains(t, pg.pipelines, pipeline.NewID(pipeline.SignalMetrics))
	require.Contains(t, pg.pipelines, pipeline.NewID(pipeline.SignalTraces))

	// Verify traces pipeline still works.
	tracePipe = pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	assert.NotNil(t, tracePipe.capabilitiesNode)
	assert.NotNil(t, tracePipe.fanOutNode)

	// Traces and metrics pipelines have different signal types, so their nodes
	// are separate. When the metrics pipeline is removed:
	// - Metrics-specific nodes are shut down
	// - Traces nodes are NOT affected (different signal type = different nodes)
	//
	// ExampleReceiver factory shares instances across signal types (same config = same instance),
	// so the traces receiver instance is stopped when the metrics receiver is shut down.
	// ExampleExporter factory does NOT share - each signal type gets a new instance.
	// Processors are always per-pipeline and not shared.
	_ = originalTraceReceiverStopped
	_ = originalTraceProcessorStopped
	_ = originalTraceExporterStopped
	assert.True(t, originalTraceReceiver.(*testcomponents.ExampleReceiver).Stopped(),
		"shared receiver instance stopped due to factory sharing")
	assert.False(t, originalTraceProcessor.(*testcomponents.ExampleProcessor).Stopped(),
		"traces processor not affected by metrics pipeline removal")
	assert.False(t, originalTraceExporter.(*testcomponents.ExampleExporter).Stopped(),
		"traces exporter not affected (ExampleExporter doesn't share instances)")

	// Data flows through the remaining traces pipeline (with rebuilt components).
	allReceivers := pg.getReceivers()
	require.Contains(t, allReceivers, pipeline.SignalTraces)
	for _, c := range allReceivers[pipeline.SignalTraces] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
	}
	for _, e := range pg.GetExporters()[pipeline.SignalTraces] {
		assert.NotEmpty(t, e.(*testcomponents.ExampleExporter).Traces)
	}

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadPipelineRemovalWithSharedReceiver verifies that when removing a pipeline
// that shares a receiver with another pipeline, the shared receiver is rebuilt.
func TestReloadPipelineRemovalWithSharedReceiver(t *testing.T) {
	// Tests that when removing a pipeline, a shared receiver's instanceID
	// is updated to remove the pipeline from its list (usedElsewhere=true path).
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): struct{}{},
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"):                   struct{}{},
		component.MustNewIDWithName("exampleprocessor", "second"): struct{}{},
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"):                   struct{}{},
		component.MustNewIDWithName("exampleexporter", "second"): struct{}{},
	}

	// Start with two pipelines sharing a receiver.
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
		pipeline.NewIDWithName(pipeline.SignalTraces, "second"): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")}, // Same receiver!
			Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "second")},
			Exporters:  []component.ID{component.MustNewIDWithName("exampleexporter", "second")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Get receiver's initial instanceID (should have 2 pipelines).
	tracePipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	var receiverNodeID int64
	for nodeID, node := range tracePipe.receivers {
		if _, ok := node.(*receiverNode); ok {
			receiverNodeID = nodeID
			break
		}
	}
	initialInstanceID := pg.instanceIDs[receiverNodeID]
	pipelineCount := 0
	initialInstanceID.AllPipelineIDs(func(_ pipeline.ID) bool {
		pipelineCount++
		return true
	})
	assert.Equal(t, 2, pipelineCount, "receiver should be in 2 pipelines initially")

	// Remove the second pipeline.
	newPipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}
	set.PipelineConfigs = newPipelineCfgs

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, procCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, nil,
		nil,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "removing pipeline should trigger reload")

	// Verify the second pipeline no longer exists.
	_, exists := pg.pipelines[pipeline.NewIDWithName(pipeline.SignalTraces, "second")]
	assert.False(t, exists, "second traces pipeline should not exist")

	// Verify the receiver now has only 1 pipeline in its instanceID.
	tracePipe = pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	for nodeID, node := range tracePipe.receivers {
		if _, ok := node.(*receiverNode); ok {
			receiverNodeID = nodeID
			break
		}
	}
	newInstanceID := pg.instanceIDs[receiverNodeID]
	pipelineCount = 0
	newInstanceID.AllPipelineIDs(func(_ pipeline.ID) bool {
		pipelineCount++
		return true
	})
	assert.Equal(t, 1, pipelineCount, "receiver should be in 1 pipeline after reload")

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadPipelineRemovalWithSharedExporter verifies that when removing a pipeline
// that shares an exporter with another pipeline, the shared exporter's instanceID is updated.
func TestReloadPipelineRemovalWithSharedExporter(t *testing.T) {
	// Tests that when removing a pipeline, a shared exporter's instanceID
	// is updated to remove the pipeline from its list (usedElsewhere=true path).
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"):                   struct{}{},
		component.MustNewIDWithName("examplereceiver", "second"): struct{}{},
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"):                   struct{}{},
		component.MustNewIDWithName("exampleprocessor", "second"): struct{}{},
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): struct{}{},
	}

	// Start with two pipelines sharing an exporter.
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
		pipeline.NewIDWithName(pipeline.SignalTraces, "second"): {
			Receivers:  []component.ID{component.MustNewIDWithName("examplereceiver", "second")},
			Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "second")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")}, // Same exporter!
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Get exporter's initial instanceID (should have 2 pipelines).
	tracePipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	var exporterNodeID int64
	for nodeID, node := range tracePipe.exporters {
		if _, ok := node.(*exporterNode); ok {
			exporterNodeID = nodeID
			break
		}
	}
	initialInstanceID := pg.instanceIDs[exporterNodeID]
	pipelineCount := 0
	initialInstanceID.AllPipelineIDs(func(_ pipeline.ID) bool {
		pipelineCount++
		return true
	})
	assert.Equal(t, 2, pipelineCount, "exporter should be in 2 pipelines initially")

	// Remove the second pipeline.
	newPipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}
	set.PipelineConfigs = newPipelineCfgs

	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, rcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, procCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, expCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, nil,
		nil,
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "removing pipeline should trigger reload")

	// Verify the second pipeline no longer exists.
	_, exists := pg.pipelines[pipeline.NewIDWithName(pipeline.SignalTraces, "second")]
	assert.False(t, exists, "second traces pipeline should not exist")

	// Verify the exporter now has only 1 pipeline in its instanceID.
	tracePipe = pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	for nodeID, node := range tracePipe.exporters {
		if _, ok := node.(*exporterNode); ok {
			exporterNodeID = nodeID
			break
		}
	}
	newInstanceID := pg.instanceIDs[exporterNodeID]
	pipelineCount = 0
	newInstanceID.AllPipelineIDs(func(_ pipeline.ID) bool {
		pipelineCount++
		return true
	})
	assert.Equal(t, 1, pipelineCount, "exporter should be in 1 pipeline after reload")

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadReplaceAllPipelinesWithConnector verifies that when ALL pipelines are replaced
// (not just some), connectors between the old pipelines are properly removed.
// This tests the case where pipelinesToRemove contains all pipelines that use a connector.
func TestReloadReplaceAllPipelinesWithConnector(t *testing.T) {
	// Start with two pipelines connected by a connector: traces -> connector -> logs.
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"):                 testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("examplereceiver", "logs"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"):                 testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("exampleprocessor", "logs"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"):                 testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("exampleexporter", "logs"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
	}
	connCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleconnector"): testcomponents.ExampleConnectorFactory.CreateDefaultConfig(),
	}
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter"), component.MustNewID("exampleconnector")},
		},
		pipeline.NewID(pipeline.SignalLogs): {
			Receivers:  []component.ID{component.MustNewIDWithName("examplereceiver", "logs"), component.MustNewID("exampleconnector")},
			Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "logs")},
			Exporters:  []component.ID{component.MustNewIDWithName("exampleexporter", "logs")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(
			connCfgs,
			map[component.Type]connector.Factory{
				testcomponents.ExampleConnectorFactory.Type(): testcomponents.ExampleConnectorFactory,
			},
		),
		PipelineConfigs: pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Verify connector exists initially.
	require.Len(t, pg.pipelines, 2)
	tracePipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	hasConnector := false
	for _, node := range tracePipe.exporters {
		if _, ok := node.(*connectorNode); ok {
			hasConnector = true
			break
		}
	}
	require.True(t, hasConnector, "traces pipeline should have connector initially")

	// Replace ALL pipelines with completely different ones (no connector).
	// This is the key scenario: both traces and logs pipelines are removed,
	// and new metrics and logs/new pipelines are added.
	newRcvrCfgs := map[component.ID]component.Config{
		component.MustNewIDWithName("examplereceiver", "new1"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("examplereceiver", "new2"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	newProcCfgs := map[component.ID]component.Config{
		component.MustNewIDWithName("exampleprocessor", "new1"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("exampleprocessor", "new2"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}
	newExpCfgs := map[component.ID]component.Config{
		component.MustNewIDWithName("exampleexporter", "new1"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("exampleexporter", "new2"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
	}
	newPipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalMetrics): {
			Receivers:  []component.ID{component.MustNewIDWithName("examplereceiver", "new1")},
			Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "new1")},
			Exporters:  []component.ID{component.MustNewIDWithName("exampleexporter", "new1")},
		},
		pipeline.NewIDWithName(pipeline.SignalLogs, "new"): {
			Receivers:  []component.ID{component.MustNewIDWithName("examplereceiver", "new2")},
			Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "new2")},
			Exporters:  []component.ID{component.MustNewIDWithName("exampleexporter", "new2")},
		},
	}
	set.PipelineConfigs = newPipelineCfgs

	// Perform the reload - this should not panic.
	// Note: Reload automatically updates set.*Builder internally based on the factory parameters.
	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, newRcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, newProcCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, newExpCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		connCfgs, nil,
		map[component.Type]connector.Factory{testcomponents.ExampleConnectorFactory.Type(): testcomponents.ExampleConnectorFactory},
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "replacing all pipelines should trigger reload")

	// Verify old pipelines are gone and new ones exist.
	require.NotContains(t, pg.pipelines, pipeline.NewID(pipeline.SignalTraces))
	require.NotContains(t, pg.pipelines, pipeline.NewID(pipeline.SignalLogs))
	require.Contains(t, pg.pipelines, pipeline.NewID(pipeline.SignalMetrics))
	require.Contains(t, pg.pipelines, pipeline.NewIDWithName(pipeline.SignalLogs, "new"))

	// Verify no connectors exist in any pipeline.
	for pipelineID, pipe := range pg.pipelines {
		for _, node := range pipe.exporters {
			if _, ok := node.(*connectorNode); ok {
				t.Errorf("pipeline %s should not have connector as exporter", pipelineID)
			}
		}
		for _, node := range pipe.receivers {
			if _, ok := node.(*connectorNode); ok {
				t.Errorf("pipeline %s should not have connector as receiver", pipelineID)
			}
		}
	}

	// Verify data flows through new pipelines.
	allReceivers := pg.getReceivers()
	require.Contains(t, allReceivers, pipeline.SignalMetrics)
	for _, c := range allReceivers[pipeline.SignalMetrics] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeMetrics(context.Background(), testdata.GenerateMetrics(1)))
	}
	for _, e := range pg.GetExporters()[pipeline.SignalMetrics] {
		assert.NotEmpty(t, e.(*testcomponents.ExampleExporter).Metrics)
	}

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}

// TestReloadAddNewPipelinesWithConnector verifies that when adding new pipelines
// that are connected by a new connector, the build order is correct.
// The connector needs its downstream capabilitiesNode to be built first.
func TestReloadAddNewPipelinesWithConnector(t *testing.T) {
	// Start with a simple metrics pipeline (no connector).
	rcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	procCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}
	expCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
	}
	pipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalMetrics): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
	}

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			rcvrCfgs,
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			procCfgs,
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
			},
		),
		ExporterBuilder: builders.NewExporter(
			expCfgs,
			map[component.Type]exporter.Factory{
				testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
			},
		),
		ConnectorBuilder: builders.NewConnector(nil, nil),
		PipelineConfigs:  pipelineCfgs,
	}

	pg, err := Build(context.Background(), set)
	require.NoError(t, err)

	host := &Host{
		Reporter: status.NewReporter(
			func(*componentstatus.InstanceID, *componentstatus.Event) {},
			func(error) {},
		),
	}
	require.NoError(t, pg.StartAll(context.Background(), host))

	// Verify initial state.
	require.Len(t, pg.pipelines, 1)
	require.Contains(t, pg.pipelines, pipeline.NewID(pipeline.SignalMetrics))

	// Add two new pipelines connected by a connector: traces -> connector -> logs.
	// This tests that the build order is correct when both the source and
	// destination pipelines are new.
	newRcvrCfgs := map[component.ID]component.Config{
		component.MustNewID("examplereceiver"):                   testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("examplereceiver", "traces"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("examplereceiver", "logs"):   testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
	}
	newProcCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleprocessor"):                   testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("exampleprocessor", "traces"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("exampleprocessor", "logs"):   testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
	}
	newExpCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleexporter"):                   testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("exampleexporter", "traces"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
		component.MustNewIDWithName("exampleexporter", "logs"):   testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
	}
	newConnCfgs := map[component.ID]component.Config{
		component.MustNewID("exampleconnector"): testcomponents.ExampleConnectorFactory.CreateDefaultConfig(),
	}
	newPipelineCfgs := pipelines.Config{
		pipeline.NewID(pipeline.SignalMetrics): {
			Receivers:  []component.ID{component.MustNewID("examplereceiver")},
			Processors: []component.ID{component.MustNewID("exampleprocessor")},
			Exporters:  []component.ID{component.MustNewID("exampleexporter")},
		},
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.MustNewIDWithName("examplereceiver", "traces")},
			Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "traces")},
			Exporters:  []component.ID{component.MustNewIDWithName("exampleexporter", "traces"), component.MustNewID("exampleconnector")},
		},
		pipeline.NewID(pipeline.SignalLogs): {
			Receivers:  []component.ID{component.MustNewIDWithName("examplereceiver", "logs"), component.MustNewID("exampleconnector")},
			Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "logs")},
			Exporters:  []component.ID{component.MustNewIDWithName("exampleexporter", "logs")},
		},
	}
	set.PipelineConfigs = newPipelineCfgs

	// Perform the reload - this should not panic due to build order issues.
	// Note: Reload automatically updates set.*Builder internally based on the factory parameters.
	reloaded, err := pg.Reload(context.Background(), &set,
		rcvrCfgs, newRcvrCfgs,
		map[component.Type]receiver.Factory{testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory},
		procCfgs, newProcCfgs,
		map[component.Type]processor.Factory{testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory},
		expCfgs, newExpCfgs,
		map[component.Type]exporter.Factory{testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory},
		nil, newConnCfgs,
		map[component.Type]connector.Factory{testcomponents.ExampleConnectorFactory.Type(): testcomponents.ExampleConnectorFactory},
		host)
	require.NoError(t, err)
	assert.True(t, reloaded, "adding new pipelines with connector should trigger reload")

	// Verify all pipelines exist.
	require.Len(t, pg.pipelines, 3)
	require.Contains(t, pg.pipelines, pipeline.NewID(pipeline.SignalMetrics))
	require.Contains(t, pg.pipelines, pipeline.NewID(pipeline.SignalTraces))
	require.Contains(t, pg.pipelines, pipeline.NewID(pipeline.SignalLogs))

	// Verify connector exists.
	tracePipe := pg.pipelines[pipeline.NewID(pipeline.SignalTraces)]
	logsPipe := pg.pipelines[pipeline.NewID(pipeline.SignalLogs)]

	hasConnectorAsExporter := false
	for _, node := range tracePipe.exporters {
		if _, ok := node.(*connectorNode); ok {
			hasConnectorAsExporter = true
			break
		}
	}
	assert.True(t, hasConnectorAsExporter, "traces pipeline should have connector as exporter")

	hasConnectorAsReceiver := false
	for _, node := range logsPipe.receivers {
		if _, ok := node.(*connectorNode); ok {
			hasConnectorAsReceiver = true
			break
		}
	}
	assert.True(t, hasConnectorAsReceiver, "logs pipeline should have connector as receiver")

	// Verify data flows through all pipelines.
	allReceivers := pg.getReceivers()

	// Test traces pipeline with connector to logs.
	require.Contains(t, allReceivers, pipeline.SignalTraces)
	for _, c := range allReceivers[pipeline.SignalTraces] {
		require.NoError(t, c.(*testcomponents.ExampleReceiver).ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
	}
	// The traces data should flow to traces exporter.
	for _, e := range pg.GetExporters()[pipeline.SignalTraces] {
		assert.NotEmpty(t, e.(*testcomponents.ExampleExporter).Traces)
	}
	// And via connector to logs exporter.
	for _, e := range pg.GetExporters()[pipeline.SignalLogs] {
		assert.NotEmpty(t, e.(*testcomponents.ExampleExporter).Logs, "logs exporter should have data from connector")
	}

	require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))
}
