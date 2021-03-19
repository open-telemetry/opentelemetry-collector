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
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/testcomponents"
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestBuildPipelines(t *testing.T) {
	tests := []struct {
		name          string
		pipelineName  string
		exporterNames []string
	}{
		{
			name:          "one-exporter",
			pipelineName:  "traces",
			exporterNames: []string{"exampleexporter"},
		},
		{
			name:          "multi-exporter",
			pipelineName:  "traces/2",
			exporterNames: []string{"exampleexporter", "exampleexporter/2"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testPipeline(t, test.pipelineName, test.exporterNames)
		})
	}
}

func createExampleConfig(dataType string) *configmodels.Config {
	exampleReceiverFactory := testcomponents.ExampleReceiverFactory
	exampleProcessorFactory := testcomponents.ExampleProcessorFactory
	exampleExporterFactory := testcomponents.ExampleExporterFactory

	cfg := &configmodels.Config{
		Receivers: map[string]configmodels.Receiver{
			string(exampleReceiverFactory.Type()): exampleReceiverFactory.CreateDefaultConfig(),
		},
		Processors: map[string]configmodels.Processor{
			string(exampleProcessorFactory.Type()): exampleProcessorFactory.CreateDefaultConfig(),
		},
		Exporters: map[string]configmodels.Exporter{
			string(exampleExporterFactory.Type()): exampleExporterFactory.CreateDefaultConfig(),
		},
		Service: configmodels.Service{
			Pipelines: map[string]*configmodels.Pipeline{
				dataType: {
					Name:       dataType,
					InputType:  configmodels.DataType(dataType),
					Receivers:  []string{string(exampleReceiverFactory.Type())},
					Processors: []string{string(exampleProcessorFactory.Type())},
					Exporters:  []string{string(exampleExporterFactory.Type())},
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
			allExporters, err := BuildExporters(zap.NewNop(), component.DefaultApplicationStartInfo(), cfg, factories.Exporters)
			if test.shouldFail {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.EqualValues(t, 1, len(allExporters))
			pipelineProcessors, err := BuildPipelines(zap.NewNop(), component.DefaultApplicationStartInfo(), cfg, allExporters, factories.Processors)

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
			exporterNames := []string{"exampleexporter"}
			var exporters []*builtExporter
			for _, name := range exporterNames {
				// Ensure exporter is created.
				exp := allExporters[cfg.Exporters[name]]
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
			processor.firstLC.(consumer.LogsConsumer).ConsumeLogs(context.Background(), log)

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

func testPipeline(t *testing.T, pipelineName string, exporterNames []string) {
	factories, err := testcomponents.ExampleComponents()
	assert.NoError(t, err)
	cfg, err := configtest.LoadConfigFile(t, "testdata/pipelines_builder.yaml", factories)
	// Load the config
	require.Nil(t, err)

	// BuildProcessors the pipeline
	allExporters, err := BuildExporters(zap.NewNop(), component.DefaultApplicationStartInfo(), cfg, factories.Exporters)
	assert.NoError(t, err)
	pipelineProcessors, err := BuildPipelines(zap.NewNop(), component.DefaultApplicationStartInfo(), cfg, allExporters, factories.Processors)

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
	for _, name := range exporterNames {
		// Ensure exporter is created.
		exp := allExporters[cfg.Exporters[name]]
		require.NotNil(t, exp)
		exporters = append(exporters, exp)
	}

	// Send TraceData via processor and verify that all exporters of the pipeline receive it.

	// First check that there are no traces in the exporters yet.
	var exporterConsumers []*testcomponents.ExampleExporterConsumer
	for _, exporter := range exporters {
		expConsumer := exporter.getTraceExporter().(*testcomponents.ExampleExporterConsumer)
		exporterConsumers = append(exporterConsumers, expConsumer)
		require.Equal(t, len(expConsumer.Traces), 0)
	}

	td := testdata.GenerateTraceDataOneSpan()
	processor.firstTC.(consumer.TracesConsumer).ConsumeTraces(context.Background(), td)

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

			cfg, err := configtest.LoadConfigFile(t, path.Join("testdata", test.configFile), factories)
			require.Nil(t, err)

			allExporters, err := BuildExporters(zap.NewNop(), component.DefaultApplicationStartInfo(), cfg, factories.Exporters)
			assert.NoError(t, err)

			pipelineProcessors, err := BuildPipelines(zap.NewNop(), component.DefaultApplicationStartInfo(), cfg, allExporters, factories.Processors)
			assert.Error(t, err)
			assert.Zero(t, len(pipelineProcessors))
		})
	}
}
