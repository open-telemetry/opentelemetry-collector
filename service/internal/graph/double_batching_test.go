// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gonum.org/v1/gonum/graph"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/pipelines"
)

type wrappedMetricsExporter struct {
	consumer.Metrics
}

func (*wrappedMetricsExporter) Start(context.Context, component.Host) error {
	return nil
}

func (*wrappedMetricsExporter) Shutdown(context.Context) error {
	return nil
}

func TestExporterNodeRecordsBatchingStatusThroughWrapper(t *testing.T) {
	exporterType := component.MustNewType("wrapped")
	exporterID := component.NewID(exporterType)
	factory := exporter.NewFactory(
		exporterType,
		func() component.Config { return struct{}{} },
		exporter.WithMetrics(
			func(_ context.Context, set exporter.Settings, _ component.Config) (exporter.Metrics, error) {
				exporter.ReportBatchingStatus(set, true)
				metrics, err := consumer.NewMetrics(func(context.Context, pmetric.Metrics) error { return nil })
				if err != nil {
					return nil, err
				}
				return &wrappedMetricsExporter{Metrics: metrics}, nil
			},
			component.StabilityLevelAlpha,
		),
	)
	node := newExporterNode(pipeline.SignalMetrics, exporterID)

	err := node.buildComponent(
		context.Background(),
		componenttest.NewNopTelemetrySettings(),
		component.NewDefaultBuildInfo(),
		builders.NewExporter(
			map[component.ID]component.Config{exporterID: factory.CreateDefaultConfig()},
			map[component.Type]exporter.Factory{exporterType: factory},
		),
	)
	require.NoError(t, err)
	require.True(t, node.exporterHelperBatchingEnabled)
}

func TestWarnIfDoubleBatching(t *testing.T) {
	t.Parallel()

	batchID := component.MustNewID("batch")
	otherProcessorID := component.MustNewID("other")
	enabledExporterID := component.MustNewIDWithName("otlp", "enabled")
	disabledExporterID := component.MustNewIDWithName("otlp", "disabled")
	tracesID := pipeline.NewID(pipeline.SignalTraces)
	metricsID := pipeline.NewID(pipeline.SignalMetrics)
	logsID := pipeline.NewID(pipeline.SignalLogs)

	core, logs := observer.New(zapcore.WarnLevel)
	g := &Graph{
		pipelines: map[pipeline.ID]*pipelineNodes{
			tracesID: {
				exporters: map[int64]graph.Node{
					1: &exporterNode{
						componentID:                   enabledExporterID,
						exporterHelperBatchingEnabled: true,
					},
					2: &exporterNode{
						componentID: disabledExporterID,
					},
				},
			},
			metricsID: {
				exporters: map[int64]graph.Node{
					1: &exporterNode{
						componentID:                   enabledExporterID,
						exporterHelperBatchingEnabled: true,
					},
				},
			},
			logsID: {
				exporters: map[int64]graph.Node{
					1: &exporterNode{
						componentID:                   enabledExporterID,
						exporterHelperBatchingEnabled: true,
					},
				},
			},
		},
		telemetry: component.TelemetrySettings{Logger: zap.New(core)},
	}

	g.warnIfDoubleBatching(pipelines.Config{
		tracesID: {
			Processors: []component.ID{batchID},
			Exporters:  []component.ID{enabledExporterID, disabledExporterID},
		},
		metricsID: {
			Processors: []component.ID{otherProcessorID},
			Exporters:  []component.ID{enabledExporterID},
		},
		logsID: {
			Processors: []component.ID{component.MustNewIDWithName("batch", "custom")},
			Exporters:  []component.ID{enabledExporterID},
		},
	})

	entries := logs.All()
	require.Len(t, entries, 2)
	pipelineNames := make([]any, 0, len(entries))
	for _, entry := range entries {
		pipelineNames = append(pipelineNames, entry.ContextMap()["pipeline"])
		require.Equal(t, "otlp/enabled", entry.ContextMap()["exporter"])
	}
	require.ElementsMatch(t, []any{"traces", "logs"}, pipelineNames)
}

func TestWarnIfDoubleBatchingWithConnectorExporter(t *testing.T) {
	t.Parallel()

	tracesID := pipeline.NewID(pipeline.SignalTraces)
	core, logs := observer.New(zapcore.WarnLevel)
	g := &Graph{
		pipelines: map[pipeline.ID]*pipelineNodes{
			tracesID: {
				exporters: map[int64]graph.Node{
					1: &connectorNode{},
				},
			},
		},
		telemetry: component.TelemetrySettings{Logger: zap.New(core)},
	}

	require.NotPanics(t, func() {
		g.warnIfDoubleBatching(pipelines.Config{
			tracesID: {
				Processors: []component.ID{component.MustNewID("batch")},
				Exporters:  []component.ID{component.MustNewID("forward")},
			},
		})
	})
	require.Empty(t, logs.All())
}
