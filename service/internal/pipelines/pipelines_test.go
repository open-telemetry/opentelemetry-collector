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

package pipelines

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/internal/testcomponents"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/service/servicetest"
)

func TestBuild(t *testing.T) {
	tests := []struct {
		name             string
		receiverIDs      []config.ComponentID
		processorIDs     []config.ComponentID
		exporterIDs      []config.ComponentID
		expectedRequests int
	}{
		{
			name:             "pipelines_simple.yaml",
			receiverIDs:      []config.ComponentID{config.NewComponentID("examplereceiver")},
			processorIDs:     []config.ComponentID{config.NewComponentID("exampleprocessor")},
			exporterIDs:      []config.ComponentID{config.NewComponentID("exampleexporter")},
			expectedRequests: 1,
		},
		{
			name:             "pipelines_simple_multi_proc.yaml",
			receiverIDs:      []config.ComponentID{config.NewComponentID("examplereceiver")},
			processorIDs:     []config.ComponentID{config.NewComponentID("exampleprocessor"), config.NewComponentID("exampleprocessor")},
			exporterIDs:      []config.ComponentID{config.NewComponentID("exampleexporter")},
			expectedRequests: 1,
		},
		{
			name:             "pipelines_simple_no_proc.yaml",
			receiverIDs:      []config.ComponentID{config.NewComponentID("examplereceiver")},
			exporterIDs:      []config.ComponentID{config.NewComponentID("exampleexporter")},
			expectedRequests: 1,
		},
		{
			name:             "pipelines_multi.yaml",
			receiverIDs:      []config.ComponentID{config.NewComponentID("examplereceiver"), config.NewComponentIDWithName("examplereceiver", "1")},
			processorIDs:     []config.ComponentID{config.NewComponentID("exampleprocessor"), config.NewComponentIDWithName("exampleprocessor", "1")},
			exporterIDs:      []config.ComponentID{config.NewComponentID("exampleexporter"), config.NewComponentIDWithName("exampleexporter", "1")},
			expectedRequests: 2,
		},
		{
			name:             "pipelines_multi_no_proc.yaml",
			receiverIDs:      []config.ComponentID{config.NewComponentID("examplereceiver"), config.NewComponentIDWithName("examplereceiver", "1")},
			exporterIDs:      []config.ComponentID{config.NewComponentID("exampleexporter"), config.NewComponentIDWithName("exampleexporter", "1")},
			expectedRequests: 2,
		},
		{
			name:             "pipelines_exporter_multi_pipeline.yaml",
			receiverIDs:      []config.ComponentID{config.NewComponentID("examplereceiver")},
			exporterIDs:      []config.ComponentID{config.NewComponentID("exampleexporter")},
			expectedRequests: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			factories, err := testcomponents.ExampleComponents()
			assert.NoError(t, err)

			cfg, err := servicetest.LoadConfigAndValidate(filepath.Join("testdata", test.name), factories)
			require.NoError(t, err)

			// Build the pipeline
			pipelines, err := Build(context.Background(), componenttest.NewNopTelemetrySettings(), component.NewDefaultBuildInfo(), cfg, factories)
			assert.NoError(t, err)

			assert.NoError(t, pipelines.StartAll(context.Background(), componenttest.NewNopHost()))

			// Verify exporters created, started and empty.
			for _, expID := range test.exporterIDs {
				traceExporter := pipelines.GetExporters()[config.TracesDataType][expID].(*testcomponents.ExampleExporter)
				assert.True(t, traceExporter.Started)
				assert.Equal(t, len(traceExporter.Traces), 0)

				// Validate metrics.
				metricsExporter := pipelines.GetExporters()[config.MetricsDataType][expID].(*testcomponents.ExampleExporter)
				assert.True(t, metricsExporter.Started)
				assert.Zero(t, len(metricsExporter.Traces))

				// Validate logs.
				logsExporter := pipelines.GetExporters()[config.LogsDataType][expID].(*testcomponents.ExampleExporter)
				assert.True(t, logsExporter.Started)
				assert.Zero(t, len(logsExporter.Traces))
			}

			// Verify processors created in the given order and started.
			for i := range test.processorIDs {
				traceProcessor := pipelines.pipelines[config.NewComponentID(config.TracesDataType)].Processors[i]
				assert.True(t, traceProcessor.(*testcomponents.ExampleProcessor).Started)

				// Validate metrics.
				metricsProcessor := pipelines.pipelines[config.NewComponentID(config.MetricsDataType)].Processors[i]
				assert.True(t, metricsProcessor.(*testcomponents.ExampleProcessor).Started)

				// Validate logs.
				logsProcessor := pipelines.pipelines[config.NewComponentID(config.LogsDataType)].Processors[i]
				assert.True(t, logsProcessor.(*testcomponents.ExampleProcessor).Started)
			}

			// Verify receivers created, started and send data to confirm pipelines correctly connected.
			for _, recvID := range test.receiverIDs {
				traceReceiver := pipelines.receivers[recvID].Receiver.(*testcomponents.ExampleReceiver)
				assert.True(t, traceReceiver.Started)
				// Send traces.
				assert.NoError(t, traceReceiver.ConsumeTraces(context.Background(), testdata.GenerateTracesOneSpan()))

				metricsReceiver := pipelines.receivers[recvID].Receiver.(*testcomponents.ExampleReceiver)
				assert.True(t, metricsReceiver.Started)
				// Send metrics.
				assert.NoError(t, metricsReceiver.ConsumeMetrics(context.Background(), testdata.GenerateMetricsOneMetric()))

				logsReceiver := pipelines.receivers[recvID].Receiver.(*testcomponents.ExampleReceiver)
				assert.True(t, logsReceiver.Started)
				// Send logs.
				assert.NoError(t, logsReceiver.ConsumeLogs(context.Background(), testdata.GenerateLogsOneLogRecord()))
			}

			assert.NoError(t, pipelines.ShutdownAll(context.Background()))

			// Verify receivers shutdown.
			for _, recvID := range test.receiverIDs {
				traceReceiver := pipelines.receivers[recvID].Receiver.(*testcomponents.ExampleReceiver)
				assert.True(t, traceReceiver.Stopped)

				metricsReceiver := pipelines.receivers[recvID].Receiver.(*testcomponents.ExampleReceiver)
				assert.True(t, metricsReceiver.Stopped)

				logsReceiver := pipelines.receivers[recvID].Receiver.(*testcomponents.ExampleReceiver)
				assert.True(t, logsReceiver.Stopped)
			}

			// Verify processors shutdown.
			for i := range test.processorIDs {
				traceProcessor := pipelines.pipelines[config.NewComponentID(config.TracesDataType)].Processors[i]
				assert.True(t, traceProcessor.(*testcomponents.ExampleProcessor).Stopped)

				// Validate metrics.
				metricsProcessor := pipelines.pipelines[config.NewComponentID(config.MetricsDataType)].Processors[i]
				assert.True(t, metricsProcessor.(*testcomponents.ExampleProcessor).Stopped)

				// Validate logs.
				logsProcessor := pipelines.pipelines[config.NewComponentID(config.LogsDataType)].Processors[i]
				assert.True(t, logsProcessor.(*testcomponents.ExampleProcessor).Stopped)
			}

			// Now verify that exporters received data, and are shutdown.
			for _, expID := range test.exporterIDs {
				// Validate traces.
				traceExporter := pipelines.GetExporters()[config.TracesDataType][expID].(*testcomponents.ExampleExporter)
				require.Len(t, traceExporter.Traces, test.expectedRequests)
				assert.EqualValues(t, testdata.GenerateTracesOneSpan(), traceExporter.Traces[0])
				assert.True(t, traceExporter.Stopped)

				// Validate metrics.
				metricsExporter := pipelines.GetExporters()[config.MetricsDataType][expID].(*testcomponents.ExampleExporter)
				require.Len(t, metricsExporter.Metrics, test.expectedRequests)
				assert.EqualValues(t, testdata.GenerateMetricsOneMetric(), metricsExporter.Metrics[0])
				assert.True(t, metricsExporter.Stopped)

				// Validate logs.
				logsExporter := pipelines.GetExporters()[config.LogsDataType][expID].(*testcomponents.ExampleExporter)
				require.Len(t, logsExporter.Logs, test.expectedRequests)
				assert.EqualValues(t, testdata.GenerateLogsOneLogRecord(), logsExporter.Logs[0])
				assert.True(t, logsExporter.Stopped)
			}
		})
	}
}

func TestBuildExportersNotSupportedDataType(t *testing.T) {
	nopReceiverFactory := componenttest.NewNopReceiverFactory()
	nopProcessorFactory := componenttest.NewNopProcessorFactory()
	nopExporterFactory := componenttest.NewNopExporterFactory()
	badReceiverFactory := newBadReceiverFactory()
	badProcessorFactory := newBadProcessorFactory()
	badExporterFactory := newBadExporterFactory()

	tests := []struct {
		configFile string
	}{
		{configFile: "not_supported_exporter_logs.yaml"},
		{configFile: "not_supported_exporter_metrics.yaml"},
		{configFile: "not_supported_exporter_traces.yaml"},
		{configFile: "not_supported_processor_logs.yaml"},
		{configFile: "not_supported_processor_metrics.yaml"},
		{configFile: "not_supported_processor_traces.yaml"},
		{configFile: "not_supported_receiver_traces.yaml"},
		{configFile: "not_supported_receiver_metrics.yaml"},
		{configFile: "not_supported_receiver_traces.yaml"},
		{configFile: "unknown_exporter_config.yaml"},
		{configFile: "unknown_exporter_factory.yaml"},
		{configFile: "unknown_processor_config.yaml"},
		{configFile: "unknown_processor_factory.yaml"},
		{configFile: "unknown_receiver_config.yaml"},
		{configFile: "unknown_receiver_factory.yaml"},
	}

	for _, test := range tests {
		t.Run(test.configFile, func(t *testing.T) {
			if test.configFile == "unknown_receiver_config.yaml" {
				t.Skip("This is a small issue with current implementation which will be fixed in #5512." +
					"In real binary this will not be hit, since the validation of the config will catch this issue in advance.")
			}
			factories := component.Factories{
				Receivers: map[config.Type]component.ReceiverFactory{
					nopReceiverFactory.Type(): nopReceiverFactory,
					"unknown":                 nopReceiverFactory,
					badReceiverFactory.Type(): badReceiverFactory,
				},
				Processors: map[config.Type]component.ProcessorFactory{
					nopProcessorFactory.Type(): nopProcessorFactory,
					"unknown":                  nopProcessorFactory,
					badProcessorFactory.Type(): badProcessorFactory,
				},
				Exporters: map[config.Type]component.ExporterFactory{
					nopExporterFactory.Type(): nopExporterFactory,
					"unknown":                 nopExporterFactory,
					badExporterFactory.Type(): badExporterFactory,
				},
			}

			// Need the unknown factories to do unmarshalling.
			cfg, err := servicetest.LoadConfig(filepath.Join("testdata", test.configFile), factories)
			require.NoError(t, err)

			// Remove the unknown factories, so they are NOT available during building.
			delete(factories.Exporters, "unknown")
			delete(factories.Processors, "unknown")
			delete(factories.Receivers, "unknown")

			_, err = Build(context.Background(), componenttest.NewNopTelemetrySettings(), component.NewDefaultBuildInfo(), cfg, factories)
			assert.Error(t, err)
		})
	}
}

func newBadReceiverFactory() component.ReceiverFactory {
	return component.NewReceiverFactory("bf", func() config.Receiver {
		return &struct {
			config.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
		}{
			ReceiverSettings: config.NewReceiverSettings(config.NewComponentID("bf")),
		}
	})
}

func newBadProcessorFactory() component.ProcessorFactory {
	return component.NewProcessorFactory("bf", func() config.Processor {
		return &struct {
			config.ProcessorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
		}{
			ProcessorSettings: config.NewProcessorSettings(config.NewComponentID("bf")),
		}
	})
}

func newBadExporterFactory() component.ExporterFactory {
	return component.NewExporterFactory("bf", func() config.Exporter {
		return &struct {
			config.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
		}{
			ExporterSettings: config.NewExporterSettings(config.NewComponentID("bf")),
		}
	})
}
