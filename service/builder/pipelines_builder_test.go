// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	idata "go.opentelemetry.io/collector/internal/data"
	"go.opentelemetry.io/collector/processor/attributesprocessor"
	"go.opentelemetry.io/collector/translator/internaldata"
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

func createExampleFactories() config.Factories {
	exampleReceiverFactory := &config.ExampleReceiverFactory{}
	exampleProcessorFactory := &config.ExampleProcessorFactory{}
	exampleExporterFactory := &config.ExampleExporterFactory{}

	factories := config.Factories{
		Receivers: map[configmodels.Type]component.ReceiverFactoryBase{
			exampleReceiverFactory.Type(): exampleReceiverFactory,
		},
		Processors: map[configmodels.Type]component.ProcessorFactoryBase{
			exampleProcessorFactory.Type(): exampleProcessorFactory,
		},
		Exporters: map[configmodels.Type]component.ExporterFactoryBase{
			exampleExporterFactory.Type(): exampleExporterFactory,
		},
	}

	return factories
}

func createExampleConfig(dataType string) *configmodels.Config {

	exampleReceiverFactory := &config.ExampleReceiverFactory{}
	exampleProcessorFactory := &config.ExampleProcessorFactory{}
	exampleExporterFactory := &config.ExampleExporterFactory{}

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
			allExporters, err := NewExportersBuilder(zap.NewNop(), cfg, factories.Exporters).Build()
			if test.shouldFail {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.EqualValues(t, 1, len(allExporters))
			pipelineProcessors, err := NewPipelinesBuilder(zap.NewNop(), cfg, allExporters, factories.Processors).Build()

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
			var exporterConsumers []*config.ExampleExporterConsumer
			for _, exporter := range exporters {
				consumer := exporter.le.(*config.ExampleExporterConsumer)
				exporterConsumers = append(exporterConsumers, consumer)
				require.Equal(t, len(consumer.Logs), 0)
			}

			// Send one custom data.
			log := idata.Logs{}
			processor.firstLC.(consumer.LogConsumer).ConsumeLogs(context.Background(), log)

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

func assertEqualTraceData(t *testing.T, expected consumerdata.TraceData, actual consumerdata.TraceData) {
	assert.True(t, proto.Equal(expected.Resource, actual.Resource))
	assert.True(t, proto.Equal(expected.Node, actual.Node))

	for i := range expected.Spans {
		assert.True(t, proto.Equal(expected.Spans[i], actual.Spans[i]))
	}

	// TODO: Source format is not very well supported in the new data, fix this.
	// assert.EqualValues(t, expected.SourceFormat, actual.SourceFormat)
}

func generateTestTraceData() consumerdata.TraceData {
	return consumerdata.TraceData{
		SourceFormat: "test-source-format",
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{
				Name: "servicename",
			},
		},
		Resource: &resourcepb.Resource{
			Type: "resourcetype",
		},
		Spans: []*tracepb.Span{
			{
				Name: &tracepb.TruncatableString{Value: "testspanname"},
				StartTime: &timestamp.Timestamp{
					Seconds: 123456789,
					Nanos:   456,
				},
				EndTime: &timestamp.Timestamp{
					Seconds: 123456789,
					Nanos:   456,
				},
			},
		},
	}
}

func generateTestTraceDataWithAttributes() consumerdata.TraceData {
	return consumerdata.TraceData{
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{
				Name: "servicename",
			},
		},
		Resource: &resourcepb.Resource{
			Type: "resourcetype",
		},
		SourceFormat: "test-source-format",
		Spans: []*tracepb.Span{
			{
				Name: &tracepb.TruncatableString{Value: "testspanname"},
				StartTime: &timestamp.Timestamp{
					Seconds: 123456789,
					Nanos:   456,
				},
				EndTime: &timestamp.Timestamp{
					Seconds: 123456789,
					Nanos:   456,
				},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"attr1": {
							Value: &tracepb.AttributeValue_IntValue{IntValue: 12345},
						},
					},
				},
			},
		},
	}
}

func assertEqualMetricsData(t *testing.T, expected consumerdata.MetricsData, actual consumerdata.MetricsData) {
	assert.True(t, proto.Equal(expected.Resource, actual.Resource))
	assert.True(t, proto.Equal(expected.Node, actual.Node))

	for i := range expected.Metrics {
		assert.True(t, proto.Equal(expected.Metrics[i], actual.Metrics[i]))
	}
}

func testPipeline(t *testing.T, pipelineName string, exporterNames []string) {
	factories, err := config.ExampleComponents()
	assert.NoError(t, err)
	attrFactory := attributesprocessor.NewFactory()
	factories.Processors[attrFactory.Type()] = attrFactory
	cfg, err := config.LoadConfigFile(t, "testdata/pipelines_builder.yaml", factories)
	// Load the config
	require.Nil(t, err)

	// BuildProcessors the pipeline
	allExporters, err := NewExportersBuilder(zap.NewNop(), cfg, factories.Exporters).Build()
	assert.NoError(t, err)
	pipelineProcessors, err := NewPipelinesBuilder(zap.NewNop(), cfg, allExporters, factories.Processors).Build()

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
	var exporterConsumers []*config.ExampleExporterConsumer
	for _, exporter := range exporters {
		consumer := exporter.te.(*config.ExampleExporterConsumer)
		exporterConsumers = append(exporterConsumers, consumer)
		require.Equal(t, len(consumer.Traces), 0)
	}

	processor.firstTC.(consumer.TraceConsumer).ConsumeTraces(context.Background(), internaldata.OCToTraceData(generateTestTraceData()))

	// Now verify received data.
	for _, consumer := range exporterConsumers {
		// Check that the trace is received by exporter.
		require.Equal(t, 1, len(consumer.Traces))

		// Verify that span is successfully delivered.
		assertEqualTraceData(t, generateTestTraceDataWithAttributes(), consumer.Traces[0])
	}

	err = pipelineProcessors.ShutdownProcessors(context.Background())
	assert.NoError(t, err)
}

func TestPipelinesBuilder_Error(t *testing.T) {
	factories, err := config.ExampleComponents()
	assert.NoError(t, err)
	attrFactory := attributesprocessor.NewFactory()
	factories.Processors[attrFactory.Type()] = attrFactory
	cfg, err := config.LoadConfigFile(t, "testdata/pipelines_builder.yaml", factories)
	require.Nil(t, err)

	// Corrupt the pipeline, change data type to metrics. We have to forcedly do it here
	// since there is no way to have such config loaded by LoadConfigFile, it would not
	// pass validation. We are doing this to test failure mode of PipelinesBuilder.
	pipeline := cfg.Service.Pipelines["traces"]
	pipeline.InputType = configmodels.MetricsDataType

	exporters, err := NewExportersBuilder(zap.NewNop(), cfg, factories.Exporters).Build()
	assert.NoError(t, err)

	// This should fail because "attributes" processor defined in the config does
	// not support metrics data type.
	_, err = NewPipelinesBuilder(zap.NewNop(), cfg, exporters, factories.Processors).Build()

	assert.NotNil(t, err)
}

func TestProcessorsBuilder_ErrorOnNilProcessor(t *testing.T) {
	factories, err := config.ExampleComponents()
	assert.NoError(t, err)

	bf := &badProcessorFactory{}
	factories.Processors[bf.Type()] = bf

	cfg, err := config.LoadConfigFile(t, "testdata/bad_processor_factory.yaml", factories)
	require.Nil(t, err)

	allExporters, err := NewExportersBuilder(zap.NewNop(), cfg, factories.Exporters).Build()
	assert.NoError(t, err)

	// First test only trace receivers by removing the metrics pipeline.
	metricsPipeline := cfg.Service.Pipelines["metrics"]
	delete(cfg.Service.Pipelines, "metrics")
	require.Equal(t, 1, len(cfg.Service.Pipelines))

	pipelineProcessors, err := NewPipelinesBuilder(zap.NewNop(), cfg, allExporters, factories.Processors).Build()
	assert.Error(t, err)
	assert.Zero(t, len(pipelineProcessors))

	// Now test the metric pipeline.
	delete(cfg.Service.Pipelines, "traces")
	cfg.Service.Pipelines["metrics"] = metricsPipeline
	require.Equal(t, 1, len(cfg.Service.Pipelines))

	pipelineProcessors, err = NewPipelinesBuilder(zap.NewNop(), cfg, allExporters, factories.Processors).Build()
	assert.Error(t, err)
	assert.Zero(t, len(pipelineProcessors))
}

// badProcessorFactory is a factory that returns no error but returns a nil object.
type badProcessorFactory struct{}

func (b *badProcessorFactory) Type() configmodels.Type {
	return "bf"
}

func (b *badProcessorFactory) CreateDefaultConfig() configmodels.Processor {
	return &configmodels.ProcessorSettings{}
}

func (b *badProcessorFactory) CreateTraceProcessor(
	_ *zap.Logger,
	_ consumer.TraceConsumerOld,
	_ configmodels.Processor,
) (component.TraceProcessorOld, error) {
	return nil, nil
}

func (b *badProcessorFactory) CreateMetricsProcessor(
	_ *zap.Logger,
	_ consumer.MetricsConsumerOld,
	_ configmodels.Processor,
) (component.MetricsProcessorOld, error) {
	return nil, nil
}
