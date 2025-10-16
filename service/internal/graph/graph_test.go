// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/pdata/xpdata/pref"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/status"
	"go.opentelemetry.io/collector/service/internal/testcomponents"
	"go.opentelemetry.io/collector/service/pipelines"
)

func TestConnectorPipelinesGraph(t *testing.T) {
	t.Run("with_internal_telemetry", func(t *testing.T) {
		setObsConsumerGateForTest(t, true)
		testConnectorPipelinesGraph(t)
	})
	t.Run("without_internal_telemetry", func(t *testing.T) {
		setObsConsumerGateForTest(t, false)
		testConnectorPipelinesGraph(t)
	})
}

func testConnectorPipelinesGraph(t *testing.T) {
	tests := []struct {
		name                string
		pipelineConfigs     pipelines.Config
		expectedPerExporter int // requires symmetry in Pipelines
	}{
		{
			name: "pipelines_simple.yaml",
			pipelineConfigs: pipelines.Config{
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
				pipeline.NewID(pipeline.SignalLogs): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(xpipeline.SignalProfiles): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_simple_mutate.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewID(pipeline.SignalTraces): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(pipeline.SignalMetrics): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(pipeline.SignalLogs): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(xpipeline.SignalProfiles): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_simple_multi_proc.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewID(pipeline.SignalTraces): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor"), component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(pipeline.SignalMetrics): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor"), component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(pipeline.SignalLogs): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor"), component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(xpipeline.SignalProfiles): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor"), component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_simple_no_proc.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewID(pipeline.SignalTraces): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(pipeline.SignalMetrics): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(pipeline.SignalLogs): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(xpipeline.SignalProfiles): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_multi.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewID(pipeline.SignalTraces): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver"), component.MustNewIDWithName("examplereceiver", "1")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate"), component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter"), component.MustNewIDWithName("exampleexporter", "1")},
				},
				pipeline.NewID(pipeline.SignalMetrics): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver"), component.MustNewIDWithName("examplereceiver", "1")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate"), component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter"), component.MustNewIDWithName("exampleexporter", "1")},
				},
				pipeline.NewID(pipeline.SignalLogs): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver"), component.MustNewIDWithName("examplereceiver", "1")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate"), component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter"), component.MustNewIDWithName("exampleexporter", "1")},
				},
				pipeline.NewID(xpipeline.SignalProfiles): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver"), component.MustNewIDWithName("examplereceiver", "1")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate"), component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter"), component.MustNewIDWithName("exampleexporter", "1")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_multi_no_proc.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewID(pipeline.SignalTraces): {
					Receivers: []component.ID{component.MustNewID("examplereceiver"), component.MustNewIDWithName("examplereceiver", "1")},
					Exporters: []component.ID{component.MustNewID("exampleexporter"), component.MustNewIDWithName("exampleexporter", "1")},
				},
				pipeline.NewID(pipeline.SignalMetrics): {
					Receivers: []component.ID{component.MustNewID("examplereceiver"), component.MustNewIDWithName("examplereceiver", "1")},
					Exporters: []component.ID{component.MustNewID("exampleexporter"), component.MustNewIDWithName("exampleexporter", "1")},
				},
				pipeline.NewID(pipeline.SignalLogs): {
					Receivers: []component.ID{component.MustNewID("examplereceiver"), component.MustNewIDWithName("examplereceiver", "1")},
					Exporters: []component.ID{component.MustNewID("exampleexporter"), component.MustNewIDWithName("exampleexporter", "1")},
				},
				pipeline.NewID(xpipeline.SignalProfiles): {
					Receivers: []component.ID{component.MustNewID("examplereceiver"), component.MustNewIDWithName("examplereceiver", "1")},
					Exporters: []component.ID{component.MustNewID("exampleexporter"), component.MustNewIDWithName("exampleexporter", "1")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "multi_pipeline_receivers_and_exporters.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewID(pipeline.SignalTraces): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "1"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(pipeline.SignalMetrics): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "1"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(pipeline.SignalLogs): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "1"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(xpipeline.SignalProfiles): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "1"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_conn_simple_traces.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalTraces, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_simple_metrics.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalMetrics, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_simple_logs.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalLogs, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_simple_profiles.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_fork_merge_traces.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalTraces, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "type0"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "type1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_conn_fork_merge_metrics.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalMetrics, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "type0"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "type1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_conn_fork_merge_logs.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalLogs, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "type0"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "type1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_conn_fork_merge_profiles.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "type0"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "type1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "merge")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_conn_translate_from_traces.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewID(pipeline.SignalTraces): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleconnector")},
				},
				pipeline.NewID(pipeline.SignalMetrics): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(pipeline.SignalLogs): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(xpipeline.SignalProfiles): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_translate_from_metrics.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewID(pipeline.SignalMetrics): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleconnector")},
				},
				pipeline.NewID(pipeline.SignalTraces): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(pipeline.SignalLogs): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(xpipeline.SignalProfiles): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_translate_from_logs.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewID(pipeline.SignalLogs): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleconnector")},
				},
				pipeline.NewID(pipeline.SignalTraces): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(pipeline.SignalMetrics): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(xpipeline.SignalProfiles): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_translate_from_profiles.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewID(pipeline.SignalTraces): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleconnector")},
				},
				pipeline.NewID(pipeline.SignalMetrics): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(pipeline.SignalLogs): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(xpipeline.SignalProfiles): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_matrix_immutable.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalTraces, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleconnector")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleconnector")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleconnector")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleconnector")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "out"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "out"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "out"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "out"): {
					Receivers:  []component.ID{component.MustNewID("exampleconnector")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 4,
		},
		{
			name: "pipelines_conn_matrix_mutable.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalTraces, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewIDWithName("exampleprocessor", "mutate")}, // mutate propagates upstream to connector
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 4,
		},
		{
			name: "pipelines_conn_lanes.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalTraces, "in"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("mockforward")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "out"): {
					Receivers: []component.ID{component.MustNewID("mockforward")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "in"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("mockforward")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "out"): {
					Receivers: []component.ID{component.MustNewID("mockforward")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "in"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("mockforward")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "out"): {
					Receivers: []component.ID{component.MustNewID("mockforward")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "in"): {
					Receivers: []component.ID{component.MustNewID("examplereceiver")},
					Exporters: []component.ID{component.MustNewID("mockforward")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "out"): {
					Receivers: []component.ID{component.MustNewID("mockforward")},
					Exporters: []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_mutate_traces.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalTraces, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "out0"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "middle"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "mutate")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "out1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_conn_mutate_metrics.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalMetrics, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "out0"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "middle"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "mutate")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "out1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_conn_mutate_logs.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalLogs, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "out0"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "middle"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "mutate")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "out1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_conn_mutate_profiles.yaml",
			pipelineConfigs: pipelines.Config{
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "in"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "out0"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "middle"): {
					Receivers:  []component.ID{component.MustNewIDWithName("exampleconnector", "inherit_mutate")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewIDWithName("exampleconnector", "mutate")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "out1"): {
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

			assert.Len(t, pg.pipelines, len(tt.pipelineConfigs))

			require.NoError(t, pg.StartAll(context.Background(), &Host{Reporter: status.NewReporter(func(*componentstatus.InstanceID, *componentstatus.Event) {}, func(error) {})}))

			mutatingPipelines := make(map[pipeline.ID]bool, len(tt.pipelineConfigs))

			// Check each pipeline individually, ensuring that all components are started
			// and that they have observed no signals yet.
			for pipelineID, pipelineCfg := range tt.pipelineConfigs {
				pl, ok := pg.pipelines[pipelineID]
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
				assert.Equal(t, expectMutatesData, pl.capabilitiesNode.getConsumer().Capabilities().MutatesData)
				mutatingPipelines[pipelineID] = expectMutatesData

				expectedReceivers, expectedExporters := expectedInstances(tt.pipelineConfigs, pipelineID)
				require.Len(t, pl.receivers, expectedReceivers)
				require.Len(t, pl.processors, len(pipelineCfg.Processors))
				require.Len(t, pl.exporters, expectedExporters)

				for _, n := range pl.exporters {
					switch c := n.(type) {
					case *exporterNode:
						e := c.Component.(*testcomponents.ExampleExporter)
						require.True(t, e.Started())
						require.Empty(t, e.Traces)
						require.Empty(t, e.Metrics)
						require.Empty(t, e.Logs)
						require.Empty(t, e.Profiles)
					case *connectorNode:
						require.True(t, c.Component.(*testcomponents.ExampleConnector).Started())
					default:
						require.Fail(t, fmt.Sprintf("unexpected type %T", c))
					}
				}

				for _, n := range pl.processors {
					require.True(t, n.(*processorNode).Component.(*testcomponents.ExampleProcessor).Started())
				}

				for _, n := range pl.receivers {
					switch c := n.(type) {
					case *receiverNode:
						require.True(t, c.Component.(*testcomponents.ExampleReceiver).Started())
					case *connectorNode:
						require.True(t, c.Component.(*testcomponents.ExampleConnector).Started())
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
			for _, c := range allReceivers[xpipeline.SignalProfiles] {
				profilesReceiver := c.(*testcomponents.ExampleReceiver)
				require.NoError(t, profilesReceiver.ConsumeProfiles(context.Background(), testdata.GenerateProfiles(1)))
			}

			// Shut down the entire component graph
			require.NoError(t, pg.ShutdownAll(context.Background(), status.NewNopStatusReporter()))

			// Check each pipeline individually, ensuring that all components are stopped.
			for pipelineID := range tt.pipelineConfigs {
				pl, ok := pg.pipelines[pipelineID]
				require.True(t, ok, "expected to find pipeline: %s", pipelineID.String())

				for _, n := range pl.receivers {
					switch c := n.(type) {
					case *receiverNode:
						require.True(t, c.Component.(*testcomponents.ExampleReceiver).Stopped())
					case *connectorNode:
						require.True(t, c.Component.(*testcomponents.ExampleConnector).Stopped())
					default:
						require.Fail(t, fmt.Sprintf("unexpected type %T", c))
					}
				}

				for _, n := range pl.processors {
					require.True(t, n.(*processorNode).Component.(*testcomponents.ExampleProcessor).Stopped())
				}

				for _, n := range pl.exporters {
					switch c := n.(type) {
					case *exporterNode:
						require.True(t, c.Component.(*testcomponents.ExampleExporter).Stopped())
					case *connectorNode:
						require.True(t, c.Component.(*testcomponents.ExampleConnector).Stopped())
					default:
						require.Fail(t, fmt.Sprintf("unexpected type %T", c))
					}
				}
			}

			// Get the list of Exporters directly from the overall component graph. Like Receivers,
			// exclude Connectors and validate each exporter once regardless of sharing between Pipelines.
			allExporters := pg.GetExporters()
			for _, e := range allExporters[pipeline.SignalTraces] {
				tracesExporter := e.(consumer.Traces).(*testcomponents.ExampleExporter)
				assert.Len(t, tracesExporter.Traces, tt.expectedPerExporter)
				expected := testdata.GenerateTraces(1)
				for i := 0; i < tt.expectedPerExporter; i++ {
					assert.True(t, pref.EqualTraces(expected, tracesExporter.Traces[i]))
				}
			}
			for _, e := range allExporters[pipeline.SignalMetrics] {
				metricsExporter := e.(consumer.Metrics).(*testcomponents.ExampleExporter)
				assert.Len(t, metricsExporter.Metrics, tt.expectedPerExporter)
				expected := testdata.GenerateMetrics(1)
				for i := 0; i < tt.expectedPerExporter; i++ {
					assert.True(t, pref.EqualMetrics(expected, metricsExporter.Metrics[i]))
				}
			}
			for _, e := range allExporters[pipeline.SignalLogs] {
				logsExporter := e.(consumer.Logs).(*testcomponents.ExampleExporter)
				assert.Len(t, logsExporter.Logs, tt.expectedPerExporter)
				expected := testdata.GenerateLogs(1)
				for i := 0; i < tt.expectedPerExporter; i++ {
					assert.True(t, pref.EqualLogs(expected, logsExporter.Logs[i]))
				}
			}
			for _, e := range allExporters[xpipeline.SignalProfiles] {
				profilesExporter := e.(xconsumer.Profiles).(*testcomponents.ExampleExporter)
				assert.Len(t, profilesExporter.Profiles, tt.expectedPerExporter)
				expected := testdata.GenerateProfiles(1)
				for i := 0; i < tt.expectedPerExporter; i++ {
					assert.True(t, pref.EqualProfiles(expected, profilesExporter.Profiles[i]))
				}
			}
		})
	}
}

func TestInstances(t *testing.T) {
	t.Run("with_internal_telemetry", func(t *testing.T) {
		setObsConsumerGateForTest(t, true)
		testInstances(t)
	})
	t.Run("without_internal_telemetry", func(t *testing.T) {
		setObsConsumerGateForTest(t, false)
		testInstances(t)
	})
}

func testInstances(t *testing.T) {
	tests := []struct {
		name            string
		pipelineConfigs pipelines.Config
		expectInstances map[component.ID]int
	}{
		{
			name: "one_pipeline_each_signal",
			pipelineConfigs: pipelines.Config{
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
				pipeline.NewID(pipeline.SignalLogs): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewID(xpipeline.SignalProfiles): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectInstances: map[component.ID]int{
				component.MustNewID("examplereceiver"):  4, // one per signal
				component.MustNewID("exampleprocessor"): 4, // one per pipeline
				component.MustNewID("exampleexporter"):  4, // one per signal
			},
		},
		{
			name: "shared_by_signals",
			pipelineConfigs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalTraces, "1"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "2"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "1"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "2"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "1"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "2"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "1"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "2"): {
					Receivers:  []component.ID{component.MustNewID("examplereceiver")},
					Processors: []component.ID{component.MustNewID("exampleprocessor")},
					Exporters:  []component.ID{component.MustNewID("exampleexporter")},
				},
			},
			expectInstances: map[component.ID]int{
				component.MustNewID("examplereceiver"):  4, // one per signal
				component.MustNewID("exampleprocessor"): 8, // one per pipeline
				component.MustNewID("exampleexporter"):  4, // one per signal
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := Settings{
				Telemetry: componenttest.NewNopTelemetrySettings(),
				BuildInfo: component.NewDefaultBuildInfo(),
				ReceiverBuilder: builders.NewReceiver(
					map[component.ID]component.Config{
						component.MustNewID("examplereceiver"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
					},
					map[component.Type]receiver.Factory{
						testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
					},
				),
				ProcessorBuilder: builders.NewProcessor(
					map[component.ID]component.Config{
						component.MustNewID("exampleprocessor"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
					},
					map[component.Type]processor.Factory{
						testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
					},
				),
				ExporterBuilder: builders.NewExporter(
					map[component.ID]component.Config{
						component.MustNewID("exampleexporter"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
					},
					map[component.Type]exporter.Factory{
						testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
					},
				),
				ConnectorBuilder: builders.NewConnector(map[component.ID]component.Config{}, map[component.Type]connector.Factory{}),
				PipelineConfigs:  tt.pipelineConfigs,
			}

			pg, err := Build(context.Background(), set)
			require.NoError(t, err)

			require.Len(t, pg.pipelines, len(set.PipelineConfigs))

			// For each component id, build a map of the instances of that component.
			// Use graph.Node.ID() as the key to determine uniqueness of instances.
			componentInstances := map[component.ID]map[int64]struct{}{}
			for _, pipeline := range pg.pipelines {
				for _, n := range pipeline.receivers {
					r := n.(*receiverNode)
					if _, ok := componentInstances[r.componentID]; !ok {
						componentInstances[r.componentID] = map[int64]struct{}{}
					}
					componentInstances[r.componentID][n.ID()] = struct{}{}
				}
				for _, n := range pipeline.processors {
					p := n.(*processorNode)
					if _, ok := componentInstances[p.componentID]; !ok {
						componentInstances[p.componentID] = map[int64]struct{}{}
					}
					componentInstances[p.componentID][n.ID()] = struct{}{}
				}
				for _, n := range pipeline.exporters {
					e := n.(*exporterNode)
					if _, ok := componentInstances[e.componentID]; !ok {
						componentInstances[e.componentID] = map[int64]struct{}{}
					}
					componentInstances[e.componentID][n.ID()] = struct{}{}
				}
			}

			var totalExpected int
			for id, instances := range componentInstances {
				totalExpected += tt.expectInstances[id]
				require.Len(t, instances, tt.expectInstances[id], id.String())
			}
			totalExpected += len(tt.pipelineConfigs) * 2 // one fanout & one capabilities node per pipeline
			require.Equal(t, totalExpected, pg.componentGraph.Nodes().Len())
		})
	}
}

func TestConnectorRouter(t *testing.T) {
	t.Run("with_internal_telemetry", func(t *testing.T) {
		setObsConsumerGateForTest(t, true)
		testConnectorRouter(t)
	})
	t.Run("without_internal_telemetry", func(t *testing.T) {
		setObsConsumerGateForTest(t, false)
		testConnectorRouter(t)
	})
}

func testConnectorRouter(t *testing.T) {
	rcvrID := component.MustNewID("examplereceiver")
	routeTracesID := component.MustNewIDWithName("examplerouter", "traces")
	routeMetricsID := component.MustNewIDWithName("examplerouter", "metrics")
	routeLogsID := component.MustNewIDWithName("examplerouter", "logs")
	routeProfilesID := component.MustNewIDWithName("examplerouter", "profiles")
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
		PipelineConfigs: pipelines.Config{
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

	assert.Len(t, pg.pipelines, len(set.PipelineConfigs))

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
	profilesReceiver := allReceivers[xpipeline.SignalProfiles][rcvrID].(*testcomponents.ExampleReceiver)
	profilesRight := allExporters[xpipeline.SignalProfiles][expRightID].(*testcomponents.ExampleExporter)
	profilesLeft := allExporters[xpipeline.SignalProfiles][expLeftID].(*testcomponents.ExampleExporter)

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
	t.Run("with_internal_telemetry", func(t *testing.T) {
		setObsConsumerGateForTest(t, true)
		testGraphBuildErrors(t)
	})
	t.Run("without_internal_telemetry", func(t *testing.T) {
		setObsConsumerGateForTest(t, false)
		testGraphBuildErrors(t)
	})
}

func testGraphBuildErrors(t *testing.T) {
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
		pipelineCfgs  pipelines.Config
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(pipeline.SignalLogs): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(pipeline.SignalMetrics): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(pipeline.SignalTraces): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(xpipeline.SignalProfiles): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(pipeline.SignalLogs): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(pipeline.SignalMetrics): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(pipeline.SignalTraces): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(xpipeline.SignalProfiles): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(pipeline.SignalLogs): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(pipeline.SignalMetrics): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(pipeline.SignalTraces): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(xpipeline.SignalProfiles): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalTraces, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in [traces/in] pipeline but not used in any supported receiver pipeline",
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalTraces, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in [traces/in] pipeline but not used in any supported receiver pipeline",
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalTraces, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in [traces/in] pipeline but not used in any supported receiver pipeline",
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalTraces, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in [traces/in] pipeline but not used in any supported receiver pipeline",
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalMetrics, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in [metrics/in] pipeline but not used in any supported receiver pipeline",
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalMetrics, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in [metrics/in] pipeline but not used in any supported receiver pipeline",
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalMetrics, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in [metrics/in] pipeline but not used in any supported receiver pipeline",
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalMetrics, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in [metrics/in] pipeline but not used in any supported receiver pipeline",
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalLogs, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in [logs/in] pipeline but not used in any supported receiver pipeline",
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalLogs, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in [logs/in] pipeline but not used in any supported receiver pipeline",
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalLogs, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in [logs/in] pipeline but not used in any supported receiver pipeline",
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalLogs, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in [logs/in] pipeline but not used in any supported receiver pipeline",
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in [profiles/in] pipeline but not used in any supported receiver pipeline",
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in [profiles/in] pipeline but not used in any supported receiver pipeline",
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in [profiles/in] pipeline but not used in any supported receiver pipeline",
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("bf")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "out"): {
					Receivers: []component.ID{component.MustNewID("bf")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: "connector \"bf\" used as exporter in [profiles/in] pipeline but not used in any supported receiver pipeline",
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalMetrics, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewIDWithName("nop", "conn")},
				},
			},
			expected: `connector "nop/conn" used as exporter in [metrics/in] pipeline but not used in any supported receiver pipeline`,
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalTraces, "out"): {
					Receivers: []component.ID{component.MustNewIDWithName("nop", "conn")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: `connector "nop/conn" used as receiver in [traces/out] pipeline but not used in any supported exporter pipeline`,
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalTraces, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("mockforward")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "out"): {
					Receivers: []component.ID{component.MustNewID("mockforward")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("mockforward")},
				},
			},
			expected: `connector "mockforward" used as exporter in [metrics/in] pipeline but not used in any supported receiver pipeline`,
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalMetrics, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("mockforward")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "out"): {
					Receivers: []component.ID{component.MustNewID("mockforward")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "out"): {
					Receivers: []component.ID{component.MustNewID("mockforward")},
					Exporters: []component.ID{component.MustNewID("nop")},
				},
			},
			expected: `connector "mockforward" used as receiver in [traces/out] pipeline but not used in any supported exporter pipeline`,
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(pipeline.SignalTraces): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(pipeline.SignalMetrics): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(pipeline.SignalLogs): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(xpipeline.SignalProfiles): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalTraces, "in"): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn1")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "2"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn1")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn2"), component.MustNewIDWithName("nop", "conn")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "out"): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalMetrics, "in"): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn1")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "2"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn1")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn2"), component.MustNewIDWithName("nop", "conn")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "out"): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalLogs, "in"): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn1")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "2"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn1")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn2"), component.MustNewIDWithName("nop", "conn")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "out"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn2")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				},
			},
			expected: `cycle detected: ` +
				`connector "nop/conn" (logs to logs) -> ` +
				`processor "nop" in pipeline "logs/1" -> ` +
				`connector "nop/conn1" (logs to logs) -> ` +
				`processor "nop" in pipeline "logs/2" -> ` +
				`connector "nop/conn" (logs to logs)`,
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "in"): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn1")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "2"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "conn1")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "conn2"), component.MustNewIDWithName("nop", "conn")},
				},
				pipeline.NewIDWithName(xpipeline.SignalProfiles, "out"): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalTraces, "in"): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "fork")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "copy1"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "fork")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "count")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "copy2"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "fork")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "forkagain")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "copy2a"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "forkagain")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "count")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "copy2b"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "forkagain")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewIDWithName("nop", "rawlog")},
				},
				pipeline.NewIDWithName(pipeline.SignalMetrics, "count"): {
					Receivers:  []component.ID{component.MustNewIDWithName("nop", "count")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "raw"): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(pipeline.SignalTraces): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(pipeline.SignalTraces): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(pipeline.SignalMetrics): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(pipeline.SignalMetrics): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(pipeline.SignalLogs): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewID(pipeline.SignalLogs): {
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
			pipelineCfgs: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalTraces, "in"): {
					Receivers: []component.ID{component.MustNewID("nop")},
					Exporters: []component.ID{component.MustNewID("unknown")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "out"): {
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
