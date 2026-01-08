// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/obsconsumer"
	"go.opentelemetry.io/collector/service/internal/testcomponents"
	"go.opentelemetry.io/collector/service/pipelines"
)

func TestComponentInstrumentation(t *testing.T) {
	setObsConsumerGateForTest(t, true)
	// All IDs have a name to ensure the "otelcol.component.id" attribute is not just the type
	rcvrID := component.MustNewIDWithName("examplereceiver", "foo")
	procID := component.MustNewIDWithName("exampleprocessor", "bar")
	routerID := component.MustNewIDWithName("examplerouter", "baz")
	expRightID := component.MustNewIDWithName("exampleexporter", "right")
	expLeftID := component.MustNewIDWithName("exampleexporter", "left")

	tracesInID := pipeline.NewIDWithName(pipeline.SignalTraces, "in")
	tracesRightID := pipeline.NewIDWithName(pipeline.SignalTraces, "right")
	tracesLeftID := pipeline.NewIDWithName(pipeline.SignalTraces, "left")

	metricsInID := pipeline.NewIDWithName(pipeline.SignalMetrics, "in")
	metricsRightID := pipeline.NewIDWithName(pipeline.SignalMetrics, "right")
	metricsLeftID := pipeline.NewIDWithName(pipeline.SignalMetrics, "left")

	logsInID := pipeline.NewIDWithName(pipeline.SignalLogs, "in")
	logsRightID := pipeline.NewIDWithName(pipeline.SignalLogs, "right")
	logsLeftID := pipeline.NewIDWithName(pipeline.SignalLogs, "left")

	profilesInID := pipeline.NewIDWithName(xpipeline.SignalProfiles, "in")
	profilesRightID := pipeline.NewIDWithName(xpipeline.SignalProfiles, "right")
	profilesLeftID := pipeline.NewIDWithName(xpipeline.SignalProfiles, "left")

	ctx := context.Background()
	testTel := componenttest.NewTelemetry()
	set := Settings{
		Telemetry: testTel.NewTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverBuilder: builders.NewReceiver(
			map[component.ID]component.Config{
				rcvrID: testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
			},
			map[component.Type]receiver.Factory{
				testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
			},
		),
		ProcessorBuilder: builders.NewProcessor(
			map[component.ID]component.Config{
				procID: testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
			},
			map[component.Type]processor.Factory{
				testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
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
				routerID: testcomponents.ExampleRouterConfig{
					Traces: &testcomponents.LeftRightConfig{
						Right: tracesRightID,
						Left:  tracesLeftID,
					},
					Metrics: &testcomponents.LeftRightConfig{
						Right: metricsRightID,
						Left:  metricsLeftID,
					},
					Logs: &testcomponents.LeftRightConfig{
						Right: logsRightID,
						Left:  logsLeftID,
					},
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
		PipelineConfigs: pipelines.Config{
			tracesInID: {
				Receivers:  []component.ID{rcvrID},
				Processors: []component.ID{procID},
				Exporters:  []component.ID{routerID},
			},
			tracesRightID: {
				Receivers: []component.ID{routerID},
				Exporters: []component.ID{expRightID},
			},
			tracesLeftID: {
				Receivers: []component.ID{routerID},
				Exporters: []component.ID{expLeftID},
			},
			metricsInID: {
				Receivers:  []component.ID{rcvrID},
				Processors: []component.ID{procID},
				Exporters:  []component.ID{routerID},
			},
			metricsRightID: {
				Receivers: []component.ID{routerID},
				Exporters: []component.ID{expRightID},
			},
			metricsLeftID: {
				Receivers: []component.ID{routerID},
				Exporters: []component.ID{expLeftID},
			},
			logsInID: {
				Receivers:  []component.ID{rcvrID},
				Processors: []component.ID{procID},
				Exporters:  []component.ID{routerID},
			},
			logsRightID: {
				Receivers: []component.ID{routerID},
				Exporters: []component.ID{expRightID},
			},
			logsLeftID: {
				Receivers: []component.ID{routerID},
				Exporters: []component.ID{expLeftID},
			},
			profilesInID: {
				Receivers:  []component.ID{rcvrID},
				Processors: []component.ID{procID},
				Exporters:  []component.ID{routerID},
			},
			profilesRightID: {
				Receivers: []component.ID{routerID},
				Exporters: []component.ID{expRightID},
			},
			profilesLeftID: {
				Receivers: []component.ID{routerID},
				Exporters: []component.ID{expLeftID},
			},
		},
	}

	pg, err := Build(ctx, set)
	require.NoError(t, err)

	allReceivers := pg.getReceivers()

	assert.Len(t, pg.pipelines, len(set.PipelineConfigs))

	// First 3 right, next 2 left
	tracesReceiver := allReceivers[pipeline.SignalTraces][rcvrID].(*testcomponents.ExampleReceiver)
	require.NoError(t, tracesReceiver.ConsumeTraces(ctx, testdata.GenerateTraces(3)))
	require.NoError(t, tracesReceiver.ConsumeTraces(ctx, testdata.GenerateTraces(2)))

	// First 5 right, next 4 left
	metricsReceiver := allReceivers[pipeline.SignalMetrics][rcvrID].(*testcomponents.ExampleReceiver)
	require.NoError(t, metricsReceiver.ConsumeMetrics(ctx, testdata.GenerateMetrics(5)))
	require.NoError(t, metricsReceiver.ConsumeMetrics(ctx, testdata.GenerateMetrics(4)))

	// First 7 right, next 6 left
	logsReceiver := allReceivers[pipeline.SignalLogs][rcvrID].(*testcomponents.ExampleReceiver)
	require.NoError(t, logsReceiver.ConsumeLogs(ctx, testdata.GenerateLogs(7)))
	require.NoError(t, logsReceiver.ConsumeLogs(ctx, testdata.GenerateLogs(6)))

	// First 9 right, next 8 left
	profilesReceiver := allReceivers[xpipeline.SignalProfiles][rcvrID].(*testcomponents.ExampleReceiver)
	require.NoError(t, profilesReceiver.ConsumeProfiles(ctx, testdata.GenerateProfiles(9)))
	require.NoError(t, profilesReceiver.ConsumeProfiles(ctx, testdata.GenerateProfiles(8)))

	// TODO fix metric name prefix delimiter
	expectedScopeMetrics := simpleScopeMetrics{
		// Traces
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "receiver"),
			attribute.String("otelcol.component.id", "examplereceiver/foo"),
			attribute.String("otelcol.signal", "traces"),
		): simpleMetricMap{
			"otelcol.receiver.produced.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 5,
			},
			"otelcol.receiver.produced.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 866,
			},
		},
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "processor"),
			attribute.String("otelcol.component.id", "exampleprocessor/bar"),
			attribute.String("otelcol.pipeline.id", "traces/in"),
			attribute.String("otelcol.signal", "traces"),
		): simpleMetricMap{
			"otelcol.processor.consumed.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 5,
			},
			"otelcol.processor.produced.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 5,
			},
			"otelcol.processor.consumed.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 866,
			},
			"otelcol.processor.produced.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 866,
			},
		},
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "connector"),
			attribute.String("otelcol.component.id", "examplerouter/baz"),
			attribute.String("otelcol.signal", "traces"),
			attribute.String("otelcol.signal.output", "traces"),
		): simpleMetricMap{
			"otelcol.connector.consumed.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 5,
			},
			"otelcol.connector.produced.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
					attribute.String("otelcol.pipeline.id", "traces/right"),
				): 3,
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
					attribute.String("otelcol.pipeline.id", "traces/left"),
				): 2,
			},
			"otelcol.connector.consumed.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 866,
			},
			"otelcol.connector.produced.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
					attribute.String("otelcol.pipeline.id", "traces/right"),
				): 528,
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
					attribute.String("otelcol.pipeline.id", "traces/left"),
				): 338,
			},
		},
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "exporter"),
			attribute.String("otelcol.component.id", "exampleexporter/right"),
			attribute.String("otelcol.signal", "traces"),
		): simpleMetricMap{
			"otelcol.exporter.consumed.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 3,
			},
			"otelcol.exporter.consumed.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 528,
			},
		},
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "exporter"),
			attribute.String("otelcol.component.id", "exampleexporter/left"),
			attribute.String("otelcol.signal", "traces"),
		): simpleMetricMap{
			"otelcol.exporter.consumed.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 2,
			},
			"otelcol.exporter.consumed.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 338,
			},
		},

		// Metrics
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "receiver"),
			attribute.String("otelcol.component.id", "examplereceiver/foo"),
			attribute.String("otelcol.signal", "metrics"),
		): simpleMetricMap{
			"otelcol.receiver.produced.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 18, // GenerateMetrics(9) produces 18 data points
			},
			"otelcol.receiver.produced.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 1741, // GenerateMetrics(9) produces 18 data points
			},
		},
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "processor"),
			attribute.String("otelcol.component.id", "exampleprocessor/bar"),
			attribute.String("otelcol.pipeline.id", "metrics/in"),
			attribute.String("otelcol.signal", "metrics"),
		): simpleMetricMap{
			"otelcol.processor.consumed.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 18, // GenerateMetrics(9) produces 18 data points
			},
			"otelcol.processor.produced.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 18, // GenerateMetrics(9) produces 18 data points
			},
			"otelcol.processor.consumed.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 1741, // GenerateMetrics(9) produces 18 data points
			},
			"otelcol.processor.produced.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 1741, // GenerateMetrics(9) produces 18 data points
			},
		},
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "connector"),
			attribute.String("otelcol.component.id", "examplerouter/baz"),
			attribute.String("otelcol.signal", "metrics"),
			attribute.String("otelcol.signal.output", "metrics"),
		): simpleMetricMap{
			"otelcol.connector.consumed.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 18, // GenerateMetrics(9) produces 18 data points
			},
			"otelcol.connector.produced.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
					attribute.String("otelcol.pipeline.id", "metrics/right"),
				): 10, // GenerateMetrics(5) produces 10 data points
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
					attribute.String("otelcol.pipeline.id", "metrics/left"),
				): 8, // GenerateMetrics(4) produces 8 data points
			},
			"otelcol.connector.consumed.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 1741, // GenerateMetrics(9) produces 18 data points
			},
			"otelcol.connector.produced.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
					attribute.String("otelcol.pipeline.id", "metrics/right"),
				): 1035, // GenerateMetrics(5) produces 10 data points
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
					attribute.String("otelcol.pipeline.id", "metrics/left"),
				): 706, // GenerateMetrics(4) produces 8 data points
			},
		},
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "exporter"),
			attribute.String("otelcol.component.id", "exampleexporter/right"),
			attribute.String("otelcol.signal", "metrics"),
		): simpleMetricMap{
			"otelcol.exporter.consumed.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 10, // GenerateMetrics(5) produces 10 data points
			},
			"otelcol.exporter.consumed.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 1035, // GenerateMetrics(9) produces 18 data points
			},
		},
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "exporter"),
			attribute.String("otelcol.component.id", "exampleexporter/left"),
			attribute.String("otelcol.signal", "metrics"),
		): simpleMetricMap{
			"otelcol.exporter.consumed.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 8, // GenerateMetrics(4) produces 8 data points
			},
			"otelcol.exporter.consumed.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 706, // GenerateMetrics(9) produces 18 data points
			},
		},

		// Logs
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "receiver"),
			attribute.String("otelcol.component.id", "examplereceiver/foo"),
			attribute.String("otelcol.signal", "logs"),
		): simpleMetricMap{
			"otelcol.receiver.produced.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 13,
			},
			"otelcol.receiver.produced.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 1363,
			},
		},
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "processor"),
			attribute.String("otelcol.component.id", "exampleprocessor/bar"),
			attribute.String("otelcol.pipeline.id", "logs/in"),
			attribute.String("otelcol.signal", "logs"),
		): simpleMetricMap{
			"otelcol.processor.consumed.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 13,
			},
			"otelcol.processor.produced.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 13,
			},
			"otelcol.processor.consumed.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 1363,
			},
			"otelcol.processor.produced.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 1363,
			},
		},
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "connector"),
			attribute.String("otelcol.component.id", "examplerouter/baz"),
			attribute.String("otelcol.signal", "logs"),
			attribute.String("otelcol.signal.output", "logs"),
		): simpleMetricMap{
			"otelcol.connector.consumed.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 13,
			},
			"otelcol.connector.produced.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
					attribute.String("otelcol.pipeline.id", "logs/right"),
				): 7,
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
					attribute.String("otelcol.pipeline.id", "logs/left"),
				): 6,
			},
			"otelcol.connector.consumed.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 1363,
			},
			"otelcol.connector.produced.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
					attribute.String("otelcol.pipeline.id", "logs/right"),
				): 737,
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
					attribute.String("otelcol.pipeline.id", "logs/left"),
				): 626,
			},
		},
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "exporter"),
			attribute.String("otelcol.component.id", "exampleexporter/right"),
			attribute.String("otelcol.signal", "logs"),
		): simpleMetricMap{
			"otelcol.exporter.consumed.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 7,
			},
			"otelcol.exporter.consumed.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 737,
			},
		},
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "exporter"),
			attribute.String("otelcol.component.id", "exampleexporter/left"),
			attribute.String("otelcol.signal", "logs"),
		): simpleMetricMap{
			"otelcol.exporter.consumed.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 6,
			},
			"otelcol.exporter.consumed.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 626,
			},
		},

		// Profiles
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "receiver"),
			attribute.String("otelcol.component.id", "examplereceiver/foo"),
			attribute.String("otelcol.signal", "profiles"),
		): simpleMetricMap{
			"otelcol.receiver.produced.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 17,
			},
			"otelcol.receiver.produced.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 1035,
			},
		},
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "processor"),
			attribute.String("otelcol.component.id", "exampleprocessor/bar"),
			attribute.String("otelcol.pipeline.id", "profiles/in"),
			attribute.String("otelcol.signal", "profiles"),
		): simpleMetricMap{
			"otelcol.processor.consumed.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 17,
			},
			"otelcol.processor.produced.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 17,
			},
			"otelcol.processor.consumed.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 1035,
			},
			"otelcol.processor.produced.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 1035,
			},
		},
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "connector"),
			attribute.String("otelcol.component.id", "examplerouter/baz"),
			attribute.String("otelcol.signal", "profiles"),
			attribute.String("otelcol.signal.output", "profiles"),
		): simpleMetricMap{
			"otelcol.connector.consumed.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 17,
			},
			"otelcol.connector.produced.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
					attribute.String("otelcol.pipeline.id", "profiles/right"),
				): 9,
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
					attribute.String("otelcol.pipeline.id", "profiles/left"),
				): 8,
			},
			"otelcol.connector.consumed.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 1035,
			},
			"otelcol.connector.produced.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
					attribute.String("otelcol.pipeline.id", "profiles/right"),
				): 542,
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
					attribute.String("otelcol.pipeline.id", "profiles/left"),
				): 493,
			},
		},
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "exporter"),
			attribute.String("otelcol.component.id", "exampleexporter/right"),
			attribute.String("otelcol.signal", "profiles"),
		): simpleMetricMap{
			"otelcol.exporter.consumed.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 9,
			},
			"otelcol.exporter.consumed.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 542,
			},
		},
		attribute.NewSet(
			attribute.String("otelcol.component.kind", "exporter"),
			attribute.String("otelcol.component.id", "exampleexporter/left"),
			attribute.String("otelcol.signal", "profiles"),
		): simpleMetricMap{
			"otelcol.exporter.consumed.items": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 8,
			},
			"otelcol.exporter.consumed.size": simpleMetric{
				attribute.NewSet(
					attribute.String(obsconsumer.ComponentOutcome, "success"),
				): 493,
			},
		},
	}

	// Verify telemetry was properly emitted by components
	var rm metricdata.ResourceMetrics
	require.NoError(t, testTel.Reader.Collect(ctx, &rm))

	require.NotNil(t, rm)
	require.NotNil(t, rm.ScopeMetrics)

	assert.Len(t, rm.ScopeMetrics, len(expectedScopeMetrics))

	for _, actualScopeMetrics := range rm.ScopeMetrics {
		expectedScopeMetrics, ok := expectedScopeMetrics[actualScopeMetrics.Scope.Attributes]
		require.True(t, ok)

		for _, actualMetric := range actualScopeMetrics.Metrics {
			expectedMetric, ok := expectedScopeMetrics[actualMetric.Name]
			require.True(t, ok)

			for _, actualPoint := range actualMetric.Data.(metricdata.Sum[int64]).DataPoints {
				expectedPoint, ok := expectedMetric[actualPoint.Attributes]
				require.True(t, ok)

				assert.Equal(t, int64(expectedPoint), actualPoint.Value)
			}
		}
	}

	assert.NoError(t, testTel.Shutdown(ctx))
}

// map[dataPointAttrs]value
type simpleMetric map[attribute.Set]int

// map[metricName]simpleMetric
type simpleMetricMap map[string]simpleMetric

// map[scopeAttrs]simpleMetricMap
type simpleScopeMetrics map[attribute.Set]simpleMetricMap
