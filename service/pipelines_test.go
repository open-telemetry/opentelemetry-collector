// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/service/internal/testcomponents"
)

func TestBuildPipelines(t *testing.T) {
	tests := []struct {
		name             string
		receiverIDs      []component.ID
		processorIDs     []component.ID
		exporterIDs      []component.ID
		pipelineConfigs  map[component.ID]*PipelineConfig
		expectedRequests int
	}{
		{
			name: "pipelines_simple",
			pipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("traces"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
				component.NewID("metrics"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
				component.NewID("logs"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
			},
			expectedRequests: 1,
		},
		{
			name: "pipelines_simple_multi_proc",
			pipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("traces"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor"), component.NewIDWithName("exampleprocessor", "1")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
				component.NewID("metrics"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor"), component.NewIDWithName("exampleprocessor", "1")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
				component.NewID("logs"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor"), component.NewIDWithName("exampleprocessor", "1")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
			},
			expectedRequests: 1,
		},
		{
			name: "pipelines_simple_no_proc",
			pipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("traces"): {
					Receivers: []component.ID{component.NewID("examplereceiver")},
					Exporters: []component.ID{component.NewID("exampleexporter")},
				},
				component.NewID("metrics"): {
					Receivers: []component.ID{component.NewID("examplereceiver")},
					Exporters: []component.ID{component.NewID("exampleexporter")},
				},
				component.NewID("logs"): {
					Receivers: []component.ID{component.NewID("examplereceiver")},
					Exporters: []component.ID{component.NewID("exampleexporter")},
				},
			},
			expectedRequests: 1,
		},
		{
			name: "pipelines_multi",
			pipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("traces"): {
					Receivers:  []component.ID{component.NewID("examplereceiver"), component.NewIDWithName("examplereceiver", "1")},
					Processors: []component.ID{component.NewID("exampleprocessor"), component.NewIDWithName("exampleprocessor", "1")},
					Exporters:  []component.ID{component.NewID("exampleexporter"), component.NewIDWithName("exampleexporter", "1")},
				},
				component.NewID("metrics"): {
					Receivers:  []component.ID{component.NewID("examplereceiver"), component.NewIDWithName("examplereceiver", "1")},
					Processors: []component.ID{component.NewID("exampleprocessor"), component.NewIDWithName("exampleprocessor", "1")},
					Exporters:  []component.ID{component.NewID("exampleexporter"), component.NewIDWithName("exampleexporter", "1")},
				},
				component.NewID("logs"): {
					Receivers:  []component.ID{component.NewID("examplereceiver"), component.NewIDWithName("examplereceiver", "1")},
					Processors: []component.ID{component.NewID("exampleprocessor"), component.NewIDWithName("exampleprocessor", "1")},
					Exporters:  []component.ID{component.NewID("exampleexporter"), component.NewIDWithName("exampleexporter", "1")},
				},
			},
			expectedRequests: 2,
		},
		{
			name: "pipelines_multi_no_proc",
			pipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("traces"): {
					Receivers: []component.ID{component.NewID("examplereceiver"), component.NewIDWithName("examplereceiver", "1")},
					Exporters: []component.ID{component.NewID("exampleexporter"), component.NewIDWithName("exampleexporter", "1")},
				},
				component.NewID("metrics"): {
					Receivers: []component.ID{component.NewID("examplereceiver"), component.NewIDWithName("examplereceiver", "1")},
					Exporters: []component.ID{component.NewID("exampleexporter"), component.NewIDWithName("exampleexporter", "1")},
				},
				component.NewID("logs"): {
					Receivers: []component.ID{component.NewID("examplereceiver"), component.NewIDWithName("examplereceiver", "1")},
					Exporters: []component.ID{component.NewID("exampleexporter"), component.NewIDWithName("exampleexporter", "1")},
				},
			},
			expectedRequests: 2,
		},
		{
			name:        "pipelines_exporter_multi_pipeline",
			receiverIDs: []component.ID{component.NewID("examplereceiver")},
			exporterIDs: []component.ID{component.NewID("exampleexporter")},
			pipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("traces"): {
					Receivers: []component.ID{component.NewID("examplereceiver")},
					Exporters: []component.ID{component.NewID("exampleexporter")},
				},
				component.NewIDWithName("traces", "1"): {
					Receivers: []component.ID{component.NewID("examplereceiver")},
					Exporters: []component.ID{component.NewID("exampleexporter")},
				},
				component.NewID("metrics"): {
					Receivers: []component.ID{component.NewID("examplereceiver")},
					Exporters: []component.ID{component.NewID("exampleexporter")},
				},
				component.NewIDWithName("metrics", "1"): {
					Receivers: []component.ID{component.NewID("examplereceiver")},
					Exporters: []component.ID{component.NewID("exampleexporter")},
				},
				component.NewID("logs"): {
					Receivers: []component.ID{component.NewID("examplereceiver")},
					Exporters: []component.ID{component.NewID("exampleexporter")},
				},
				component.NewIDWithName("logs", "1"): {
					Receivers: []component.ID{component.NewID("examplereceiver")},
					Exporters: []component.ID{component.NewID("exampleexporter")},
				},
			},
			expectedRequests: 4,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Build the pipeline
			pips, err := buildPipelines(context.Background(), pipelinesSettings{
				Telemetry: componenttest.NewNopTelemetrySettings(),
				BuildInfo: component.NewDefaultBuildInfo(),
				Receivers: receiver.NewBuilder(
					map[component.ID]component.Config{
						component.NewID("examplereceiver"):              testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
						component.NewIDWithName("examplereceiver", "1"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
					},
					map[component.Type]receiver.Factory{
						testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
					}),
				Processors: processor.NewBuilder(
					map[component.ID]component.Config{
						component.NewID("exampleprocessor"):              testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
						component.NewIDWithName("exampleprocessor", "1"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
					},
					map[component.Type]processor.Factory{
						testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
					}),
				Exporters: exporter.NewBuilder(
					map[component.ID]component.Config{
						component.NewID("exampleexporter"):              testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
						component.NewIDWithName("exampleexporter", "1"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
					},
					map[component.Type]exporter.Factory{
						testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
					}),
				PipelineConfigs: test.pipelineConfigs,
			})
			assert.NoError(t, err)
			assert.IsType(t, &builtPipelines{}, pips)

			pipelines := pips.(*builtPipelines)
			assert.NoError(t, pipelines.StartAll(context.Background(), componenttest.NewNopHost()))

			for dt, pipeline := range test.pipelineConfigs {
				// Verify exporters created, started and empty.
				for _, expID := range pipeline.Exporters {
					exp := pipelines.GetExporters()[dt.Type()][expID].(*testcomponents.ExampleExporter)
					assert.True(t, exp.Started())
					switch dt.Type() {
					case component.DataTypeTraces:
						assert.Len(t, exp.RecallTraces(), 0)
					case component.DataTypeMetrics:
						assert.Len(t, exp.RecallMetrics(), 0)
					case component.DataTypeLogs:
						assert.Len(t, exp.RecallLogs(), 0)
					}
				}

				// Verify processors created in the given order and started.
				for i, procID := range pipeline.Processors {
					processor := pipelines.pipelines[dt].processors[i]
					assert.Equal(t, procID, processor.id)
					assert.True(t, processor.comp.(*testcomponents.ExampleProcessor).Started())
				}

				// Verify receivers created, started and send data to confirm pipelines correctly connected.
				for _, recvID := range pipeline.Receivers {
					receiver := pipelines.allReceivers[dt.Type()][recvID].(*testcomponents.ExampleReceiver)
					assert.True(t, receiver.Started())
				}
			}

			// Send data to confirm pipelines correctly connected.
			for dt, pipeline := range test.pipelineConfigs {
				for _, recvID := range pipeline.Receivers {
					receiver := pipelines.allReceivers[dt.Type()][recvID].(*testcomponents.ExampleReceiver)
					switch dt.Type() {
					case component.DataTypeTraces:
						assert.NoError(t, receiver.ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
					case component.DataTypeMetrics:
						assert.NoError(t, receiver.ConsumeMetrics(context.Background(), testdata.GenerateMetrics(1)))
					case component.DataTypeLogs:
						assert.NoError(t, receiver.ConsumeLogs(context.Background(), testdata.GenerateLogs(1)))
					}
				}
			}

			assert.NoError(t, pipelines.ShutdownAll(context.Background()))

			for dt, pipeline := range test.pipelineConfigs {
				// Verify receivers shutdown.
				for _, recvID := range pipeline.Receivers {
					receiver := pipelines.allReceivers[dt.Type()][recvID].(*testcomponents.ExampleReceiver)
					assert.True(t, receiver.Stopped())
				}

				// Verify processors shutdown.
				for i := range pipeline.Processors {
					processor := pipelines.pipelines[dt].processors[i]
					assert.True(t, processor.comp.(*testcomponents.ExampleProcessor).Stopped())
				}

				// Now verify that exporters received data, and are shutdown.
				for _, expID := range pipeline.Exporters {
					exp := pipelines.GetExporters()[dt.Type()][expID].(*testcomponents.ExampleExporter)
					switch dt.Type() {
					case component.DataTypeTraces:
						require.Len(t, exp.RecallTraces(), test.expectedRequests)
						assert.EqualValues(t, testdata.GenerateTraces(1), exp.RecallTraces()[0])
					case component.DataTypeMetrics:
						require.Len(t, exp.RecallMetrics(), test.expectedRequests)
						assert.EqualValues(t, testdata.GenerateMetrics(1), exp.RecallMetrics()[0])
					case component.DataTypeLogs:
						require.Len(t, exp.RecallLogs(), test.expectedRequests)
						assert.EqualValues(t, testdata.GenerateLogs(1), exp.RecallLogs()[0])
					}
					assert.True(t, exp.Stopped())
				}
			}
		})
	}
}

func TestBuildErrors(t *testing.T) {
	nopReceiverFactory := receivertest.NewNopFactory()
	nopProcessorFactory := processortest.NewNopFactory()
	nopExporterFactory := exportertest.NewNopFactory()
	badReceiverFactory := newBadReceiverFactory()
	badProcessorFactory := newBadProcessorFactory()
	badExporterFactory := newBadExporterFactory()

	tests := []struct {
		name             string
		ReceiverConfigs  map[component.ID]component.Config
		ProcessorConfigs map[component.ID]component.Config
		ExporterConfigs  map[component.ID]component.Config
		PipelineConfigs  map[component.ID]*PipelineConfig
		expected         string
	}{
		{
			name: "not_supported_exporter_logs",
			ReceiverConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			ExporterConfigs: map[component.ID]component.Config{
				component.NewID("bf"): badExporterFactory.CreateDefaultConfig(),
			},
			PipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("logs"): {
					Receivers: []component.ID{component.NewID("nop")},
					Exporters: []component.ID{component.NewID("bf")},
				},
			},
			expected: "failed to create \"bf\" exporter, in pipeline \"logs\": telemetry type is not supported",
		},
		{
			name: "not_supported_exporter_metrics",
			ReceiverConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			ExporterConfigs: map[component.ID]component.Config{
				component.NewID("bf"): badExporterFactory.CreateDefaultConfig(),
			},
			PipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("metrics"): {
					Receivers: []component.ID{component.NewID("nop")},
					Exporters: []component.ID{component.NewID("bf")},
				},
			},
			expected: "failed to create \"bf\" exporter, in pipeline \"metrics\": telemetry type is not supported",
		},
		{
			name: "not_supported_exporter_traces",
			ReceiverConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			ExporterConfigs: map[component.ID]component.Config{
				component.NewID("bf"): badExporterFactory.CreateDefaultConfig(),
			},
			PipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("traces"): {
					Receivers: []component.ID{component.NewID("nop")},
					Exporters: []component.ID{component.NewID("bf")},
				},
			},
			expected: "failed to create \"bf\" exporter, in pipeline \"traces\": telemetry type is not supported",
		},
		{
			name: "not_supported_processor_logs",
			ReceiverConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			ProcessorConfigs: map[component.ID]component.Config{
				component.NewID("bf"): badProcessorFactory.CreateDefaultConfig(),
			},
			ExporterConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			PipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("logs"): {
					Receivers:  []component.ID{component.NewID("nop")},
					Processors: []component.ID{component.NewID("bf")},
					Exporters:  []component.ID{component.NewID("nop")},
				},
			},
			expected: "failed to create \"bf\" processor, in pipeline \"logs\": telemetry type is not supported",
		},
		{
			name: "not_supported_processor_metrics",
			ReceiverConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			ProcessorConfigs: map[component.ID]component.Config{
				component.NewID("bf"): badProcessorFactory.CreateDefaultConfig(),
			},
			ExporterConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			PipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("metrics"): {
					Receivers:  []component.ID{component.NewID("nop")},
					Processors: []component.ID{component.NewID("bf")},
					Exporters:  []component.ID{component.NewID("nop")},
				},
			},
			expected: "failed to create \"bf\" processor, in pipeline \"metrics\": telemetry type is not supported",
		},
		{
			name: "not_supported_processor_traces",
			ReceiverConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			ProcessorConfigs: map[component.ID]component.Config{
				component.NewID("bf"): badProcessorFactory.CreateDefaultConfig(),
			},
			ExporterConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			PipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("traces"): {
					Receivers:  []component.ID{component.NewID("nop")},
					Processors: []component.ID{component.NewID("bf")},
					Exporters:  []component.ID{component.NewID("nop")},
				},
			},
			expected: "failed to create \"bf\" processor, in pipeline \"traces\": telemetry type is not supported",
		},
		{
			name: "not_supported_receiver_logs",
			ReceiverConfigs: map[component.ID]component.Config{
				component.NewID("bf"): badReceiverFactory.CreateDefaultConfig(),
			},
			ExporterConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			PipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("logs"): {
					Receivers: []component.ID{component.NewID("bf")},
					Exporters: []component.ID{component.NewID("nop")},
				},
			},
			expected: "failed to create \"bf\" receiver, in pipeline \"logs\": telemetry type is not supported",
		},
		{
			name: "not_supported_receiver_metrics",
			ReceiverConfigs: map[component.ID]component.Config{
				component.NewID("bf"): badReceiverFactory.CreateDefaultConfig(),
			},
			ExporterConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			PipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("metrics"): {
					Receivers: []component.ID{component.NewID("bf")},
					Exporters: []component.ID{component.NewID("nop")},
				},
			},
			expected: "failed to create \"bf\" receiver, in pipeline \"metrics\": telemetry type is not supported",
		},
		{
			name: "not_supported_receiver_traces",
			ReceiverConfigs: map[component.ID]component.Config{
				component.NewID("bf"): badReceiverFactory.CreateDefaultConfig(),
			},
			ExporterConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			PipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("traces"): {
					Receivers: []component.ID{component.NewID("bf")},
					Exporters: []component.ID{component.NewID("nop")},
				},
			},
			expected: "failed to create \"bf\" receiver, in pipeline \"traces\": telemetry type is not supported",
		},
		{
			name: "unknown_exporter_config",
			ReceiverConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			ExporterConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			PipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("traces"): {
					Receivers: []component.ID{component.NewID("nop")},
					Exporters: []component.ID{component.NewID("nop"), component.NewIDWithName("nop", "1")},
				},
			},
			expected: "failed to create \"nop/1\" exporter, in pipeline \"traces\": exporter \"nop/1\" is not configured",
		},
		{
			name: "unknown_exporter_factory",
			ReceiverConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			ExporterConfigs: map[component.ID]component.Config{
				component.NewID("unknown"): nopExporterFactory.CreateDefaultConfig(),
			},
			PipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("traces"): {
					Receivers: []component.ID{component.NewID("nop")},
					Exporters: []component.ID{component.NewID("unknown")},
				},
			},
			expected: "failed to create \"unknown\" exporter, in pipeline \"traces\": exporter factory not available for: \"unknown\"",
		},
		{
			name: "unknown_processor_config",
			ReceiverConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			ProcessorConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
			},
			ExporterConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			PipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("metrics"): {
					Receivers:  []component.ID{component.NewID("nop")},
					Processors: []component.ID{component.NewID("nop"), component.NewIDWithName("nop", "1")},
					Exporters:  []component.ID{component.NewID("nop")},
				},
			},
			expected: "failed to create \"nop/1\" processor, in pipeline \"metrics\": processor \"nop/1\" is not configured",
		},
		{
			name: "unknown_processor_factory",
			ReceiverConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			ProcessorConfigs: map[component.ID]component.Config{
				component.NewID("unknown"): nopProcessorFactory.CreateDefaultConfig(),
			},
			ExporterConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			PipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("metrics"): {
					Receivers:  []component.ID{component.NewID("nop")},
					Processors: []component.ID{component.NewID("unknown")},
					Exporters:  []component.ID{component.NewID("nop")},
				},
			},
			expected: "failed to create \"unknown\" processor, in pipeline \"metrics\": processor factory not available for: \"unknown\"",
		},
		{
			name: "unknown_receiver_config",
			ReceiverConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
			},
			ExporterConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			PipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("logs"): {
					Receivers: []component.ID{component.NewID("nop"), component.NewIDWithName("nop", "1")},
					Exporters: []component.ID{component.NewID("nop")},
				},
			},
			expected: "failed to create \"nop/1\" receiver, in pipeline \"logs\": receiver \"nop/1\" is not configured",
		},
		{
			name: "unknown_receiver_factory",
			ReceiverConfigs: map[component.ID]component.Config{
				component.NewID("unknown"): nopReceiverFactory.CreateDefaultConfig(),
			},
			ExporterConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
			},
			PipelineConfigs: map[component.ID]*PipelineConfig{
				component.NewID("logs"): {
					Receivers: []component.ID{component.NewID("unknown")},
					Exporters: []component.ID{component.NewID("nop")},
				},
			},
			expected: "failed to create \"unknown\" receiver, in pipeline \"logs\": receiver factory not available for: \"unknown\"",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			set := pipelinesSettings{
				Telemetry: componenttest.NewNopTelemetrySettings(),
				BuildInfo: component.NewDefaultBuildInfo(),
				Receivers: receiver.NewBuilder(
					test.ReceiverConfigs,
					map[component.Type]receiver.Factory{
						nopReceiverFactory.Type(): nopReceiverFactory,
						badReceiverFactory.Type(): badReceiverFactory,
					}),
				Processors: processor.NewBuilder(
					test.ProcessorConfigs,
					map[component.Type]processor.Factory{
						nopProcessorFactory.Type(): nopProcessorFactory,
						badProcessorFactory.Type(): badProcessorFactory,
					}),
				Exporters: exporter.NewBuilder(
					test.ExporterConfigs,
					map[component.Type]exporter.Factory{
						nopExporterFactory.Type(): nopExporterFactory,
						badExporterFactory.Type(): badExporterFactory,
					}),
				PipelineConfigs: test.PipelineConfigs,
			}

			_, err := buildPipelines(context.Background(), set)
			assert.EqualError(t, err, test.expected)
		})
	}
}

func TestFailToStartAndShutdown(t *testing.T) {
	errReceiverFactory := newErrReceiverFactory()
	errProcessorFactory := newErrProcessorFactory()
	errExporterFactory := newErrExporterFactory()
	nopReceiverFactory := receivertest.NewNopFactory()
	nopProcessorFactory := processortest.NewNopFactory()
	nopExporterFactory := exportertest.NewNopFactory()

	set := pipelinesSettings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		Receivers: receiver.NewBuilder(
			map[component.ID]component.Config{
				component.NewID(nopReceiverFactory.Type()): nopReceiverFactory.CreateDefaultConfig(),
				component.NewID(errReceiverFactory.Type()): errReceiverFactory.CreateDefaultConfig(),
			},
			map[component.Type]receiver.Factory{
				nopReceiverFactory.Type(): nopReceiverFactory,
				errReceiverFactory.Type(): errReceiverFactory,
			}),
		Processors: processor.NewBuilder(
			map[component.ID]component.Config{
				component.NewID(nopProcessorFactory.Type()): nopProcessorFactory.CreateDefaultConfig(),
				component.NewID(errProcessorFactory.Type()): errProcessorFactory.CreateDefaultConfig(),
			},
			map[component.Type]processor.Factory{
				nopProcessorFactory.Type(): nopProcessorFactory,
				errProcessorFactory.Type(): errProcessorFactory,
			}),
		Exporters: exporter.NewBuilder(
			map[component.ID]component.Config{
				component.NewID(nopExporterFactory.Type()): nopExporterFactory.CreateDefaultConfig(),
				component.NewID(errExporterFactory.Type()): errExporterFactory.CreateDefaultConfig(),
			},
			map[component.Type]exporter.Factory{
				nopExporterFactory.Type(): nopExporterFactory,
				errExporterFactory.Type(): errExporterFactory,
			}),
	}

	for _, dt := range []component.DataType{component.DataTypeTraces, component.DataTypeMetrics, component.DataTypeLogs} {
		t.Run(string(dt)+"/receiver", func(t *testing.T) {
			set.PipelineConfigs = map[component.ID]*PipelineConfig{
				component.NewID(dt): {
					Receivers:  []component.ID{component.NewID("nop"), component.NewID("err")},
					Processors: []component.ID{component.NewID("nop")},
					Exporters:  []component.ID{component.NewID("nop")},
				},
			}
			pipelines, err := buildPipelines(context.Background(), set)
			assert.NoError(t, err)
			assert.Error(t, pipelines.StartAll(context.Background(), componenttest.NewNopHost()))
			assert.Error(t, pipelines.ShutdownAll(context.Background()))
		})

		t.Run(string(dt)+"/processor", func(t *testing.T) {
			set.PipelineConfigs = map[component.ID]*PipelineConfig{
				component.NewID(dt): {
					Receivers:  []component.ID{component.NewID("nop")},
					Processors: []component.ID{component.NewID("nop"), component.NewID("err")},
					Exporters:  []component.ID{component.NewID("nop")},
				},
			}
			pipelines, err := buildPipelines(context.Background(), set)
			assert.NoError(t, err)
			assert.Error(t, pipelines.StartAll(context.Background(), componenttest.NewNopHost()))
			assert.Error(t, pipelines.ShutdownAll(context.Background()))
		})

		t.Run(string(dt)+"/exporter", func(t *testing.T) {
			set.PipelineConfigs = map[component.ID]*PipelineConfig{
				component.NewID(dt): {
					Receivers:  []component.ID{component.NewID("nop")},
					Processors: []component.ID{component.NewID("nop")},
					Exporters:  []component.ID{component.NewID("nop"), component.NewID("err")},
				},
			}
			pipelines, err := buildPipelines(context.Background(), set)
			assert.NoError(t, err)
			assert.Error(t, pipelines.StartAll(context.Background(), componenttest.NewNopHost()))
			assert.Error(t, pipelines.ShutdownAll(context.Background()))
		})
	}
}

func newBadReceiverFactory() receiver.Factory {
	return receiver.NewFactory("bf", func() component.Config {
		return &struct{}{}
	})
}

func newBadProcessorFactory() processor.Factory {
	return processor.NewFactory("bf", func() component.Config {
		return &struct{}{}
	})
}

func newBadExporterFactory() exporter.Factory {
	return exporter.NewFactory("bf", func() component.Config {
		return &struct{}{}
	})
}

func newBadConnectorFactory() connector.Factory {
	return connector.NewFactory("bf", func() component.Config {
		return &struct{}{}
	})
}

func newErrReceiverFactory() receiver.Factory {
	return receiver.NewFactory("err",
		func() component.Config { return &struct{}{} },
		receiver.WithTraces(func(context.Context, receiver.CreateSettings, component.Config, consumer.Traces) (receiver.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		receiver.WithLogs(func(context.Context, receiver.CreateSettings, component.Config, consumer.Logs) (receiver.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		receiver.WithMetrics(func(context.Context, receiver.CreateSettings, component.Config, consumer.Metrics) (receiver.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func newErrProcessorFactory() processor.Factory {
	return processor.NewFactory("err",
		func() component.Config { return &struct{}{} },
		processor.WithTraces(func(context.Context, processor.CreateSettings, component.Config, consumer.Traces) (processor.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		processor.WithLogs(func(context.Context, processor.CreateSettings, component.Config, consumer.Logs) (processor.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		processor.WithMetrics(func(context.Context, processor.CreateSettings, component.Config, consumer.Metrics) (processor.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func newErrExporterFactory() exporter.Factory {
	return exporter.NewFactory("err",
		func() component.Config { return &struct{}{} },
		exporter.WithTraces(func(context.Context, exporter.CreateSettings, component.Config) (exporter.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		exporter.WithLogs(func(context.Context, exporter.CreateSettings, component.Config) (exporter.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		exporter.WithMetrics(func(context.Context, exporter.CreateSettings, component.Config) (exporter.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func newErrConnectorFactory() connector.Factory {
	return connector.NewFactory("err", func() component.Config {
		return &struct{}{}
	},
		connector.WithTracesToTraces(func(context.Context, connector.CreateSettings, component.Config, consumer.Traces) (connector.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connector.WithTracesToMetrics(func(context.Context, connector.CreateSettings, component.Config, consumer.Metrics) (connector.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connector.WithTracesToLogs(func(context.Context, connector.CreateSettings, component.Config, consumer.Logs) (connector.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),

		connector.WithMetricsToTraces(func(context.Context, connector.CreateSettings, component.Config, consumer.Traces) (connector.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connector.WithMetricsToMetrics(func(context.Context, connector.CreateSettings, component.Config, consumer.Metrics) (connector.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connector.WithMetricsToLogs(func(context.Context, connector.CreateSettings, component.Config, consumer.Logs) (connector.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),

		connector.WithLogsToTraces(func(context.Context, connector.CreateSettings, component.Config, consumer.Traces) (connector.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connector.WithLogsToMetrics(func(context.Context, connector.CreateSettings, component.Config, consumer.Metrics) (connector.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connector.WithLogsToLogs(func(context.Context, connector.CreateSettings, component.Config, consumer.Logs) (connector.Logs, error) {
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
