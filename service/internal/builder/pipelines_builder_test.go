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

package builder

import (
	"context"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/internal/testcomponents"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/model/pdata"
)

func TestBuildPipelines(t *testing.T) {
	tests := []struct {
		name          string
		pipelineName  string
		exporterNames []config.ComponentID
	}{
		{
			name:          "one-exporter",
			pipelineName:  "traces",
			exporterNames: []config.ComponentID{config.NewID("exampleexporter")},
		},
		{
			name:          "multi-exporter",
			pipelineName:  "traces/2",
			exporterNames: []config.ComponentID{config.NewID("exampleexporter"), config.NewIDWithName("exampleexporter", "2")},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testPipeline(t, test.pipelineName, test.exporterNames)
		})
	}
}

func createExampleConfig(dataType string) *config.Config {
	exampleReceiverFactory := testcomponents.ExampleReceiverFactory
	exampleProcessorFactory := testcomponents.ExampleProcessorFactory
	exampleExporterFactory := testcomponents.ExampleExporterFactory

	cfg := &config.Config{
		Receivers: map[config.ComponentID]config.Receiver{
			config.NewID(exampleReceiverFactory.Type()): exampleReceiverFactory.CreateDefaultConfig(),
		},
		Processors: map[config.ComponentID]config.Processor{
			config.NewID(exampleProcessorFactory.Type()): exampleProcessorFactory.CreateDefaultConfig(),
		},
		Exporters: map[config.ComponentID]config.Exporter{
			config.NewID(exampleExporterFactory.Type()): exampleExporterFactory.CreateDefaultConfig(),
		},
		Service: config.Service{
			Pipelines: map[string]*config.Pipeline{
				dataType: {
					Name:       dataType,
					InputType:  config.DataType(dataType),
					Receivers:  []config.ComponentID{config.NewID(exampleReceiverFactory.Type())},
					Processors: []config.ComponentID{config.NewID(exampleProcessorFactory.Type())},
					Exporters:  []config.ComponentID{config.NewID(exampleExporterFactory.Type())},
				},
			},
		},
	}
	return cfg
}

func TestBuildPipelines_BuildVarious(t *testing.T) {

	factories := createTestFactories()

	tests := []struct {
		dataType   string
		shouldFail bool
	}{
		{
			dataType:   "logs",
			shouldFail: false,
		},
		{
			dataType:   "nosuchdatatype",
			shouldFail: true,
		},
	}

	for _, test := range tests {
		t.Run(test.dataType, func(t *testing.T) {
			dataType := test.dataType

			cfg := createExampleConfig(dataType)

			// BuildProcessors the pipeline
			allExporters, err := BuildExporters(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, factories.Exporters)
			if test.shouldFail {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.EqualValues(t, 1, len(allExporters))
			pipelineProcessors, err := BuildPipelines(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, allExporters, factories.Processors)

			assert.NoError(t, err)
			require.NotNil(t, pipelineProcessors)

			err = pipelineProcessors.StartProcessors(context.Background(), componenttest.NewNopHost())
			assert.NoError(t, err)

			pipelineName := dataType
			processor := pipelineProcessors[cfg.Service.Pipelines[pipelineName]]

			// Ensure pipeline has its fields correctly populated.
			require.NotNil(t, processor)
			assert.Nil(t, processor.firstTC)
			assert.Nil(t, processor.firstMC)
			assert.NotNil(t, processor.firstLC)

			// Compose the list of created exporters.
			exporterIDs := []config.ComponentID{config.NewID("exampleexporter")}
			var exporters []*builtExporter
			for _, expID := range exporterIDs {
				// Ensure exporter is created.
				exp := allExporters[expID]
				require.NotNil(t, exp)
				exporters = append(exporters, exp)
			}

			// Send Logs via processor and verify that all exporters of the pipeline receive it.

			// First check that there are no logs in the exporters yet.
			var exporterConsumers []*testcomponents.ExampleExporterConsumer
			for _, exporter := range exporters {
				expConsumer := exporter.getLogExporter().(*testcomponents.ExampleExporterConsumer)
				exporterConsumers = append(exporterConsumers, expConsumer)
				require.Equal(t, len(expConsumer.Logs), 0)
			}

			// Send one custom data.
			log := pdata.Logs{}
			require.NoError(t, processor.firstLC.ConsumeLogs(context.Background(), log))

			// Now verify received data.
			for _, expConsumer := range exporterConsumers {
				// Check that the trace is received by exporter.
				require.Equal(t, 1, len(expConsumer.Logs))

				// Verify that span is successfully delivered.
				assert.EqualValues(t, log, expConsumer.Logs[0])
			}

			err = pipelineProcessors.ShutdownProcessors(context.Background())
			assert.NoError(t, err)
		})
	}
}

func testPipeline(t *testing.T, pipelineName string, exporterIDs []config.ComponentID) {
	factories, err := testcomponents.ExampleComponents()
	assert.NoError(t, err)
	cfg, err := configtest.LoadConfigAndValidate("testdata/pipelines_builder.yaml", factories)
	// Unmarshal the config
	require.Nil(t, err)

	// BuildProcessors the pipeline
	allExporters, err := BuildExporters(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, factories.Exporters)
	assert.NoError(t, err)
	pipelineProcessors, err := BuildPipelines(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, allExporters, factories.Processors)

	assert.NoError(t, err)
	require.NotNil(t, pipelineProcessors)

	assert.NoError(t, pipelineProcessors.StartProcessors(context.Background(), componenttest.NewNopHost()))

	processor := pipelineProcessors[cfg.Service.Pipelines[pipelineName]]

	// Ensure pipeline has its fields correctly populated.
	require.NotNil(t, processor)
	assert.NotNil(t, processor.firstTC)
	assert.Nil(t, processor.firstMC)

	// Compose the list of created exporters.
	var exporters []*builtExporter
	for _, expID := range exporterIDs {
		// Ensure exporter is created.
		exp := allExporters[expID]
		require.NotNil(t, exp)
		exporters = append(exporters, exp)
	}

	// Send TraceData via processor and verify that all exporters of the pipeline receive it.

	// First check that there are no traces in the exporters yet.
	var exporterConsumers []*testcomponents.ExampleExporterConsumer
	for _, exporter := range exporters {
		expConsumer := exporter.getTracesExporter().(*testcomponents.ExampleExporterConsumer)
		exporterConsumers = append(exporterConsumers, expConsumer)
		require.Equal(t, len(expConsumer.Traces), 0)
	}

	td := testdata.GenerateTracesOneSpan()
	require.NoError(t, processor.firstTC.ConsumeTraces(context.Background(), td))

	// Now verify received data.
	for _, expConsumer := range exporterConsumers {
		// Check that the trace is received by exporter.
		require.Equal(t, 1, len(expConsumer.Traces))

		// Verify that span is successfully delivered.
		assert.EqualValues(t, td, expConsumer.Traces[0])
	}

	err = pipelineProcessors.ShutdownProcessors(context.Background())
	assert.NoError(t, err)
}

func TestBuildPipelines_NotSupportedDataType(t *testing.T) {
	factories := createTestFactories()

	tests := []struct {
		configFile string
	}{
		{
			configFile: "not_supported_processor_logs.yaml",
		},
		{
			configFile: "not_supported_processor_metrics.yaml",
		},
		{
			configFile: "not_supported_processor_traces.yaml",
		},
	}

	for _, test := range tests {
		t.Run(test.configFile, func(t *testing.T) {

			cfg, err := configtest.LoadConfigAndValidate(path.Join("testdata", test.configFile), factories)
			require.Nil(t, err)

			allExporters, err := BuildExporters(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, factories.Exporters)
			assert.NoError(t, err)

			pipelineProcessors, err := BuildPipelines(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, allExporters, factories.Processors)
			assert.Error(t, err)
			assert.Zero(t, len(pipelineProcessors))
		})
	}
}
