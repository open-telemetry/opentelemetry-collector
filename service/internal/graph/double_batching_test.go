// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gonum.org/v1/gonum/graph"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/pipelines"
)

type wrappedMetricsExporter struct {
	exporter.Metrics
}

type wrappedMetricsExporterWithBatching struct {
	exporter.Metrics
	exporterHelperBatcher
}

func TestExporterNodeReadsBatchingStatusCapability(t *testing.T) {
	for _, tt := range []struct {
		name            string
		queueEnabled    bool
		batchEnabled    bool
		wrapped         bool
		exposesBatching bool
		wantBatching    bool
	}{
		{name: "no_queue"},
		{name: "queue_without_batching", queueEnabled: true},
		{name: "direct", queueEnabled: true, batchEnabled: true, wantBatching: true},
		{name: "opaque_wrapper", queueEnabled: true, batchEnabled: true, wrapped: true},
		{name: "wrapped_disabled", queueEnabled: true, wrapped: true},
		{name: "wrapper_exposes_capability", queueEnabled: true, batchEnabled: true, exposesBatching: true, wantBatching: true},
		{name: "wrapper_exposes_disabled_capability", queueEnabled: true, exposesBatching: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			exporterType := component.MustNewType("wrapped")
			exporterID := component.NewID(exporterType)
			factory := exporter.NewFactory(
				exporterType,
				func() component.Config { return struct{}{} },
				exporter.WithMetrics(
					func(ctx context.Context, set exporter.Settings, cfg component.Config) (exporter.Metrics, error) {
						qCfg := exporterhelper.NewDefaultQueueConfig()
						qCfg.Batch = configoptional.None[exporterhelper.BatchConfig]()
						if tt.batchEnabled {
							qCfg.Batch = configoptional.Some(exporterhelper.NewDefaultBatchConfig())
						}
						queue := configoptional.None[exporterhelper.QueueBatchConfig]()
						if tt.queueEnabled {
							queue = configoptional.Some(qCfg)
						}
						push := func(context.Context, pmetric.Metrics) error { return nil }
						metrics, err := exporterhelper.NewMetrics(ctx, set, cfg, push,
							exporterhelper.WithQueue(queue))
						if err != nil {
							return nil, err
						}
						if tt.exposesBatching {
							helper, ok := metrics.(exporterHelperBatcher)
							if !ok {
								return nil, fmt.Errorf("exporterhelper exporter %T does not expose batching status", metrics)
							}
							return &wrappedMetricsExporterWithBatching{Metrics: metrics, exporterHelperBatcher: helper}, nil
						}
						if tt.wrapped {
							return &wrappedMetricsExporter{Metrics: metrics}, nil
						}
						return metrics, nil
					},
					component.StabilityLevelAlpha,
				),
			)
			node := newExporterNode(pipeline.SignalMetrics, exporterID)
			core, logs := observer.New(zapcore.WarnLevel)
			telemetry := componenttest.NewNopTelemetrySettings()
			telemetry.Logger = zap.New(core)

			err := node.buildComponent(context.Background(), telemetry, component.NewDefaultBuildInfo(),
				builders.NewExporter(
					map[component.ID]component.Config{exporterID: factory.CreateDefaultConfig()},
					map[component.Type]exporter.Factory{exporterType: factory},
				))
			require.NoError(t, err)
			require.Equal(t, tt.wantBatching, node.exporterHelperBatchingEnabled)

			pipelineID := pipeline.NewID(pipeline.SignalMetrics)
			g := &Graph{
				pipelines: map[pipeline.ID]*pipelineNodes{
					pipelineID: {exporters: map[int64]graph.Node{node.ID(): node}},
				},
				telemetry: telemetry,
			}
			g.warnIfDoubleBatching(pipelines.Config{
				pipelineID: {Processors: []component.ID{component.MustNewID("batch")}},
			})
			entries := logs.FilterMessageSnippet("double batching").All()
			if tt.wantBatching {
				require.Len(t, entries, 1)
				require.Equal(t, pipelineID.String(), entries[0].ContextMap()["pipeline"])
				require.Equal(t, exporterID.String(), entries[0].ContextMap()["exporter"])
			} else {
				require.Empty(t, entries)
			}
		})
	}
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
