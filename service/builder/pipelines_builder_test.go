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
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

func TestPipelinesBuilder_Build(t *testing.T) {
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

func createExampleFactories() component.Factories {
	exampleReceiverFactory := &componenttest.ExampleReceiverFactory{}
	exampleProcessorFactory := &componenttest.ExampleProcessorFactory{}
	exampleExporterFactory := &componenttest.ExampleExporterFactory{}

	factories := component.Factories{
		Receivers: map[configmodels.Type]component.ReceiverFactory{
			exampleReceiverFactory.Type(): exampleReceiverFactory,
		},
		Processors: map[configmodels.Type]component.ProcessorFactory{
			exampleProcessorFactory.Type(): exampleProcessorFactory,
		},
		Exporters: map[configmodels.Type]component.ExporterFactory{
			exampleExporterFactory.Type(): exampleExporterFactory,
		},
	}

	return factories
}

func createExampleConfig(dataType string) *configmodels.Config {

	exampleReceiverFactory := &componenttest.ExampleReceiverFactory{}
	exampleProcessorFactory := &componenttest.ExampleProcessorFactory{}
	exampleExporterFactory := &componenttest.ExampleExporterFactory{}

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

func TestPipelinesBuilder_BuildVarious(t *testing.T) {

	factories := createExampleFactories()

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
			allExporters, err := NewExportersBuilder(zap.NewNop(), componenttest.TestApplicationStartInfo(), cfg, factories.Exporters).Build()
			if test.shouldFail {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.EqualValues(t, 1, len(allExporters))
			pipelineProcessors, err := NewPipelinesBuilder(zap.NewNop(), componenttest.TestApplicationStartInfo(), cfg, allExporters, factories.Processors).Build()

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
			var exporterConsumers []*componenttest.ExampleExporterConsumer
			for _, exporter := range exporters {
				consumer := exporter.getLogExporter().(*componenttest.ExampleExporterConsumer)
				exporterConsumers = append(exporterConsumers, consumer)
				require.Equal(t, len(consumer.Logs), 0)
			}

			// Send one custom data.
			log := pdata.Logs{}
			processor.firstLC.(consumer.LogsConsumer).ConsumeLogs(context.Background(), log)

			// Now verify received data.
			for _, consumer := range exporterConsumers {
				// Check that the trace is received by exporter.
				require.Equal(t, 1, len(consumer.Logs))

				// Verify that span is successfully delivered.
				assert.EqualValues(t, log, consumer.Logs[0])
			}

			err = pipelineProcessors.ShutdownProcessors(context.Background())
			assert.NoError(t, err)
		})
	}
}

func testPipeline(t *testing.T, pipelineName string, exporterNames []string) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)
	cfg, err := configtest.LoadConfigFile(t, "testdata/pipelines_builder.yaml", factories)
	// Load the config
	require.Nil(t, err)

	// BuildProcessors the pipeline
	allExporters, err := NewExportersBuilder(zap.NewNop(), componenttest.TestApplicationStartInfo(), cfg, factories.Exporters).Build()
	assert.NoError(t, err)
	pipelineProcessors, err := NewPipelinesBuilder(zap.NewNop(), componenttest.TestApplicationStartInfo(), cfg, allExporters, factories.Processors).Build()

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
	var exporterConsumers []*componenttest.ExampleExporterConsumer
	for _, exporter := range exporters {
		consumer := exporter.getTraceExporter().(*componenttest.ExampleExporterConsumer)
		exporterConsumers = append(exporterConsumers, consumer)
		require.Equal(t, len(consumer.Traces), 0)
	}

	td := testdata.GenerateTraceDataOneSpan()
	processor.firstTC.(consumer.TracesConsumer).ConsumeTraces(context.Background(), td)

	// Now verify received data.
	for _, consumer := range exporterConsumers {
		// Check that the trace is received by exporter.
		require.Equal(t, 1, len(consumer.Traces))

		// Verify that span is successfully delivered.
		assert.EqualValues(t, td, consumer.Traces[0])
	}

	err = pipelineProcessors.ShutdownProcessors(context.Background())
	assert.NoError(t, err)
}

func TestProcessorsBuilder_ErrorOnUnsupportedProcessor(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	bf := newBadProcessorFactory()
	factories.Processors[bf.Type()] = bf

	cfg, err := configtest.LoadConfigFile(t, "testdata/bad_processor_factory.yaml", factories)
	require.Nil(t, err)

	allExporters, err := NewExportersBuilder(zap.NewNop(), componenttest.TestApplicationStartInfo(), cfg, factories.Exporters).Build()
	assert.NoError(t, err)

	// First test only trace receivers by removing the metrics pipeline.
	metricsPipeline := cfg.Service.Pipelines["metrics"]
	logsPipeline := cfg.Service.Pipelines["logs"]
	delete(cfg.Service.Pipelines, "metrics")
	delete(cfg.Service.Pipelines, "logs")
	require.Equal(t, 1, len(cfg.Service.Pipelines))

	pipelineProcessors, err := NewPipelinesBuilder(zap.NewNop(), componenttest.TestApplicationStartInfo(), cfg, allExporters, factories.Processors).Build()
	assert.Error(t, err)
	assert.Zero(t, len(pipelineProcessors))

	// Now test the metric pipeline.
	delete(cfg.Service.Pipelines, "traces")
	cfg.Service.Pipelines["metrics"] = metricsPipeline
	require.Equal(t, 1, len(cfg.Service.Pipelines))

	pipelineProcessors, err = NewPipelinesBuilder(zap.NewNop(), componenttest.TestApplicationStartInfo(), cfg, allExporters, factories.Processors).Build()
	assert.Error(t, err)
	assert.Zero(t, len(pipelineProcessors))

	// Now test the logs pipeline.
	delete(cfg.Service.Pipelines, "metrics")
	cfg.Service.Pipelines["logs"] = logsPipeline
	require.Equal(t, 1, len(cfg.Service.Pipelines))

	pipelineProcessors, err = NewPipelinesBuilder(zap.NewNop(), componenttest.TestApplicationStartInfo(), cfg, allExporters, factories.Processors).Build()
	assert.Error(t, err)
	assert.Zero(t, len(pipelineProcessors))
}

func newBadProcessorFactory() component.ProcessorFactory {
	return processorhelper.NewFactory("bf", func() configmodels.Processor {
		return &configmodels.ProcessorSettings{
			TypeVal: "bf",
			NameVal: "bf",
		}
	})
}
