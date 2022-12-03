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
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/service/internal/testcomponents"
)

func TestBuildPipelines(t *testing.T) {
	tests := []struct {
		name             string
		receiverIDs      []component.ID
		processorIDs     []component.ID
		exporterIDs      []component.ID
		pipelineConfigs  map[component.ID]*ConfigServicePipeline
		expectedRequests int
	}{
		{
			name: "pipelines_simple",
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
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
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
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
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
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
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
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
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
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
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
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
			pipelines, err := buildPipelines(context.Background(), Settings{
				Telemetry: componenttest.NewNopTelemetrySettings(),
				BuildInfo: component.NewDefaultBuildInfo(),
				ReceiverFactories: map[component.Type]component.ReceiverFactory{
					testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
				},
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("examplereceiver"):              testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
					component.NewIDWithName("examplereceiver", "1"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
				},
				ProcessorFactories: map[component.Type]component.ProcessorFactory{
					testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
				},
				ProcessorConfigs: map[component.ID]component.Config{
					component.NewID("exampleprocessor"):              testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
					component.NewIDWithName("exampleprocessor", "1"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
				},
				ExporterFactories: map[component.Type]component.ExporterFactory{
					testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("exampleexporter"):              testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
					component.NewIDWithName("exampleexporter", "1"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: test.pipelineConfigs,
			})
			assert.NoError(t, err)

			assert.NoError(t, pipelines.StartAll(context.Background(), componenttest.NewNopHost()))

			for dt, pipeline := range test.pipelineConfigs {
				// Verify exporters created, started and empty.
				for _, expID := range pipeline.Exporters {
					exporter := pipelines.GetExporters()[dt.Type()][expID].(*testcomponents.ExampleExporter)
					assert.True(t, exporter.Started)
					switch dt.Type() {
					case component.DataTypeTraces:
						assert.Len(t, exporter.Traces, 0)
					case component.DataTypeMetrics:
						assert.Len(t, exporter.Metrics, 0)
					case component.DataTypeLogs:
						assert.Len(t, exporter.Logs, 0)
					}
				}

				// Verify processors created in the given order and started.
				for i, procID := range pipeline.Processors {
					processor := pipelines.pipelines[dt].processors[i]
					assert.Equal(t, procID, processor.id)
					assert.True(t, processor.comp.(*testcomponents.ExampleProcessor).Started)
				}

				// Verify receivers created, started and send data to confirm pipelines correctly connected.
				for _, recvID := range pipeline.Receivers {
					receiver := pipelines.allReceivers[dt.Type()][recvID].(*testcomponents.ExampleReceiver)
					assert.True(t, receiver.Started)
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
					assert.True(t, receiver.Stopped)
				}

				// Verify processors shutdown.
				for i := range pipeline.Processors {
					processor := pipelines.pipelines[dt].processors[i]
					assert.True(t, processor.comp.(*testcomponents.ExampleProcessor).Stopped)
				}

				// Now verify that exporters received data, and are shutdown.
				for _, expID := range pipeline.Exporters {
					exporter := pipelines.GetExporters()[dt.Type()][expID].(*testcomponents.ExampleExporter)
					switch dt.Type() {
					case component.DataTypeTraces:
						require.Len(t, exporter.Traces, test.expectedRequests)
						assert.EqualValues(t, testdata.GenerateTraces(1), exporter.Traces[0])
					case component.DataTypeMetrics:
						require.Len(t, exporter.Metrics, test.expectedRequests)
						assert.EqualValues(t, testdata.GenerateMetrics(1), exporter.Metrics[0])
					case component.DataTypeLogs:
						require.Len(t, exporter.Logs, test.expectedRequests)
						assert.EqualValues(t, testdata.GenerateLogs(1), exporter.Logs[0])
					}
					assert.True(t, exporter.Stopped)
				}
			}
		})
	}
}

func TestBuildErrors(t *testing.T) {
	nopReceiverFactory := componenttest.NewNopReceiverFactory()
	nopProcessorFactory := componenttest.NewNopProcessorFactory()
	nopExporterFactory := componenttest.NewNopExporterFactory()
	badReceiverFactory := newBadReceiverFactory()
	badProcessorFactory := newBadProcessorFactory()
	badExporterFactory := newBadExporterFactory()

	tests := []struct {
		name     string
		settings Settings
	}{
		{
			name: "not_supported_exporter_logs",
			settings: Settings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("bf"): badExporterFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("logs"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("bf")},
					},
				},
			},
		},
		{
			name: "not_supported_exporter_metrics",
			settings: Settings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("bf"): badExporterFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("metrics"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("bf")},
					},
				},
			},
		},
		{
			name: "not_supported_exporter_traces",
			settings: Settings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("bf"): badExporterFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("traces"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("bf")},
					},
				},
			},
		},
		{
			name: "not_supported_processor_logs",
			settings: Settings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ProcessorConfigs: map[component.ID]component.Config{
					component.NewID("bf"): badProcessorFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("logs"): {
						Receivers:  []component.ID{component.NewID("nop")},
						Processors: []component.ID{component.NewID("bf")},
						Exporters:  []component.ID{component.NewID("nop")},
					},
				},
			},
		},
		{
			name: "not_supported_processor_metrics",
			settings: Settings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ProcessorConfigs: map[component.ID]component.Config{
					component.NewID("bf"): badProcessorFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("metrics"): {
						Receivers:  []component.ID{component.NewID("nop")},
						Processors: []component.ID{component.NewID("bf")},
						Exporters:  []component.ID{component.NewID("nop")},
					},
				},
			},
		},
		{
			name: "not_supported_processor_traces",
			settings: Settings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ProcessorConfigs: map[component.ID]component.Config{
					component.NewID("bf"): badProcessorFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("traces"): {
						Receivers:  []component.ID{component.NewID("nop")},
						Processors: []component.ID{component.NewID("bf")},
						Exporters:  []component.ID{component.NewID("nop")},
					},
				},
			},
		},
		{
			name: "not_supported_receiver_logs",
			settings: Settings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("bf"): badReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("logs"): {
						Receivers: []component.ID{component.NewID("bf")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
		},
		{
			name: "not_supported_receiver_metrics",
			settings: Settings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("bf"): badReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("metrics"): {
						Receivers: []component.ID{component.NewID("bf")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
		},
		{
			name: "not_supported_receiver_traces",
			settings: Settings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("bf"): badReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("traces"): {
						Receivers: []component.ID{component.NewID("bf")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
		},
		{
			name: "unknown_exporter_config",
			settings: Settings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("traces"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("nop"), component.NewID("nop/1")},
					},
				},
			},
		},
		{
			name: "unknown_exporter_factory",
			settings: Settings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("unknown"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("traces"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("unknown")},
					},
				},
			},
		},
		{
			name: "unknown_processor_config",
			settings: Settings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ProcessorConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("metrics"): {
						Receivers:  []component.ID{component.NewID("nop")},
						Processors: []component.ID{component.NewID("nop"), component.NewID("nop/1")},
						Exporters:  []component.ID{component.NewID("nop")},
					},
				},
			},
		},
		{
			name: "unknown_processor_factory",
			settings: Settings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ProcessorConfigs: map[component.ID]component.Config{
					component.NewID("unknown"): nopProcessorFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("metrics"): {
						Receivers:  []component.ID{component.NewID("nop")},
						Processors: []component.ID{component.NewID("unknown")},
						Exporters:  []component.ID{component.NewID("nop")},
					},
				},
			},
		},
		{
			name: "unknown_receiver_config",
			settings: Settings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("logs"): {
						Receivers: []component.ID{component.NewID("nop"), component.NewID("nop/1")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
		},
		{
			name: "unknown_receiver_factory",
			settings: Settings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("unknown"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("logs"): {
						Receivers: []component.ID{component.NewID("unknown")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			set := test.settings
			set.BuildInfo = component.NewDefaultBuildInfo()
			set.Telemetry = componenttest.NewNopTelemetrySettings()
			set.ReceiverFactories = map[component.Type]component.ReceiverFactory{
				nopReceiverFactory.Type(): nopReceiverFactory,
				badReceiverFactory.Type(): badReceiverFactory,
			}
			set.ProcessorFactories = map[component.Type]component.ProcessorFactory{
				nopProcessorFactory.Type(): nopProcessorFactory,
				badProcessorFactory.Type(): badProcessorFactory,
			}
			set.ExporterFactories = map[component.Type]component.ExporterFactory{
				nopExporterFactory.Type(): nopExporterFactory,
				badExporterFactory.Type(): badExporterFactory,
			}

			_, err := buildPipelines(context.Background(), set)
			assert.Error(t, err)
		})
	}
}

func TestFailToStartAndShutdown(t *testing.T) {
	errReceiverFactory := newErrReceiverFactory()
	errProcessorFactory := newErrProcessorFactory()
	errExporterFactory := newErrExporterFactory()
	nopReceiverFactory := componenttest.NewNopReceiverFactory()
	nopProcessorFactory := componenttest.NewNopProcessorFactory()
	nopExporterFactory := componenttest.NewNopExporterFactory()

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverFactories: map[component.Type]component.ReceiverFactory{
			nopReceiverFactory.Type(): nopReceiverFactory,
			errReceiverFactory.Type(): errReceiverFactory,
		},
		ReceiverConfigs: map[component.ID]component.Config{
			component.NewID(nopReceiverFactory.Type()): nopReceiverFactory.CreateDefaultConfig(),
			component.NewID(errReceiverFactory.Type()): errReceiverFactory.CreateDefaultConfig(),
		},
		ProcessorFactories: map[component.Type]component.ProcessorFactory{
			nopProcessorFactory.Type(): nopProcessorFactory,
			errProcessorFactory.Type(): errProcessorFactory,
		},
		ProcessorConfigs: map[component.ID]component.Config{
			component.NewID(nopProcessorFactory.Type()): nopProcessorFactory.CreateDefaultConfig(),
			component.NewID(errProcessorFactory.Type()): errProcessorFactory.CreateDefaultConfig(),
		},
		ExporterFactories: map[component.Type]component.ExporterFactory{
			nopExporterFactory.Type(): nopExporterFactory,
			errExporterFactory.Type(): errExporterFactory,
		},
		ExporterConfigs: map[component.ID]component.Config{
			component.NewID(nopExporterFactory.Type()): nopExporterFactory.CreateDefaultConfig(),
			component.NewID(errExporterFactory.Type()): errExporterFactory.CreateDefaultConfig(),
		},
	}

	for _, dt := range []component.DataType{component.DataTypeTraces, component.DataTypeMetrics, component.DataTypeLogs} {
		t.Run(string(dt)+"/receiver", func(t *testing.T) {
			set.PipelineConfigs = map[component.ID]*ConfigServicePipeline{
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
			set.PipelineConfigs = map[component.ID]*ConfigServicePipeline{
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
			set.PipelineConfigs = map[component.ID]*ConfigServicePipeline{
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

func newBadReceiverFactory() component.ReceiverFactory {
	return component.NewReceiverFactory("bf", func() component.Config {
		return &struct {
			config.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
		}{
			ReceiverSettings: config.NewReceiverSettings(component.NewID("bf")),
		}
	})
}

func newBadProcessorFactory() component.ProcessorFactory {
	return component.NewProcessorFactory("bf", func() component.Config {
		return &struct {
			config.ProcessorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
		}{
			ProcessorSettings: config.NewProcessorSettings(component.NewID("bf")),
		}
	})
}

func newBadExporterFactory() component.ExporterFactory {
	return component.NewExporterFactory("bf", func() component.Config {
		return &struct {
			config.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
		}{
			ExporterSettings: config.NewExporterSettings(component.NewID("bf")),
		}
	})
}

func newErrReceiverFactory() component.ReceiverFactory {
	return component.NewReceiverFactory("err", func() component.Config {
		return &struct {
			config.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
		}{
			ReceiverSettings: config.NewReceiverSettings(component.NewID("bf")),
		}
	},
		component.WithTracesReceiver(func(context.Context, component.ReceiverCreateSettings, component.Config, consumer.Traces) (component.TracesReceiver, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		component.WithLogsReceiver(func(context.Context, component.ReceiverCreateSettings, component.Config, consumer.Logs) (component.LogsReceiver, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		component.WithMetricsReceiver(func(context.Context, component.ReceiverCreateSettings, component.Config, consumer.Metrics) (component.MetricsReceiver, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func newErrProcessorFactory() component.ProcessorFactory {
	return component.NewProcessorFactory("err", func() component.Config {
		return &struct {
			config.ProcessorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
		}{
			ProcessorSettings: config.NewProcessorSettings(component.NewID("bf")),
		}
	},
		component.WithTracesProcessor(func(context.Context, component.ProcessorCreateSettings, component.Config, consumer.Traces) (component.TracesProcessor, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		component.WithLogsProcessor(func(context.Context, component.ProcessorCreateSettings, component.Config, consumer.Logs) (component.LogsProcessor, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		component.WithMetricsProcessor(func(context.Context, component.ProcessorCreateSettings, component.Config, consumer.Metrics) (component.MetricsProcessor, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func newErrExporterFactory() component.ExporterFactory {
	return component.NewExporterFactory("err", func() component.Config {
		return &struct {
			config.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
		}{
			ExporterSettings: config.NewExporterSettings(component.NewID("bf")),
		}
	},
		component.WithTracesExporter(func(context.Context, component.ExporterCreateSettings, component.Config) (component.TracesExporter, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		component.WithLogsExporter(func(context.Context, component.ExporterCreateSettings, component.Config) (component.LogsExporter, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		component.WithMetricsExporter(func(context.Context, component.ExporterCreateSettings, component.Config) (component.MetricsExporter, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
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
