// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gonum.org/v1/gonum/graph"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/pipelines"
)

type testExporterHelperBatcher struct {
	component.Component
	enabled bool
}

func (b testExporterHelperBatcher) ExporterHelperBatchingEnabled() bool {
	return b.enabled
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
						componentID: enabledExporterID,
						Component:   testExporterHelperBatcher{enabled: true},
					},
					2: &exporterNode{
						componentID: disabledExporterID,
						Component:   testExporterHelperBatcher{enabled: false},
					},
				},
			},
			metricsID: {
				exporters: map[int64]graph.Node{
					1: &exporterNode{
						componentID: enabledExporterID,
						Component:   testExporterHelperBatcher{enabled: true},
					},
				},
			},
			logsID: {
				exporters: map[int64]graph.Node{
					1: &exporterNode{
						componentID: enabledExporterID,
						Component:   testExporterHelperBatcher{enabled: true},
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
