// Copyright 2019, OpenTelemetry Authors
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/processor"
	"github.com/open-telemetry/opentelemetry-collector/processor/attributesprocessor"
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

func assertEqualTraceData(t *testing.T, expected consumerdata.TraceData, actual consumerdata.TraceData) {
	assert.True(t, proto.Equal(expected.Resource, actual.Resource))
	assert.True(t, proto.Equal(expected.Node, actual.Node))

	for i := range expected.Spans {
		assert.True(t, proto.Equal(expected.Spans[i], actual.Spans[i]))
	}

	assert.EqualValues(t, expected.SourceFormat, actual.SourceFormat)
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
	assert.Nil(t, err)
	attrFactory := &attributesprocessor.Factory{}
	factories.Processors[attrFactory.Type()] = attrFactory
	cfg, err := config.LoadConfigFile(t, "testdata/pipelines_builder.yaml", factories)
	// Load the config
	require.Nil(t, err)

	// Build the pipeline
	allExporters, err := NewExportersBuilder(zap.NewNop(), cfg, factories.Exporters).Build()
	assert.NoError(t, err)
	pipelineProcessors, err := NewPipelinesBuilder(zap.NewNop(), cfg, allExporters, factories.Processors).Build()

	assert.NoError(t, err)
	require.NotNil(t, pipelineProcessors)

	mh := component.NewMockHost()
	err = pipelineProcessors.StartProcessors(zap.NewNop(), mh)
	assert.NoError(t, err)

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

	// Send one trace.
	name := tracepb.TruncatableString{Value: "testspanname"}
	traceData := consumerdata.TraceData{
		SourceFormat: "test-source-format",
		Spans: []*tracepb.Span{
			{Name: &name},
		},
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{
				Name: "servicename",
			},
		},
		Resource: &resourcepb.Resource{
			Type: "resourcetype",
		},
	}
	processor.firstTC.ConsumeTraceData(context.Background(), traceData)

	// Now verify received data.
	for _, consumer := range exporterConsumers {
		// Check that the trace is received by exporter.
		require.Equal(t, 1, len(consumer.Traces))

		// Verify that span is successfully delivered.
		assertEqualTraceData(t, traceData, consumer.Traces[0])

		// Check that the span was processed by "attributes" processor and an
		// attribute was added.
		assert.Equal(t, int64(12345),
			consumer.Traces[0].Spans[0].Attributes.AttributeMap["attr1"].GetIntValue())
	}

	err = pipelineProcessors.ShutdownProcessors(zap.NewNop())
	assert.NoError(t, err)
}

func TestPipelinesBuilder_Error(t *testing.T) {
	factories, err := config.ExampleComponents()
	assert.Nil(t, err)
	attrFactory := &attributesprocessor.Factory{}
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
	assert.Nil(t, err)

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

var _ processor.Factory = (*badProcessorFactory)(nil)

func (b *badProcessorFactory) Type() string {
	return "bf"
}

func (b *badProcessorFactory) CreateDefaultConfig() configmodels.Processor {
	return &configmodels.ProcessorSettings{}
}

func (b *badProcessorFactory) CreateTraceProcessor(
	logger *zap.Logger,
	nextConsumer consumer.TraceConsumer,
	cfg configmodels.Processor,
) (processor.TraceProcessor, error) {
	return nil, nil
}

func (b *badProcessorFactory) CreateMetricsProcessor(
	logger *zap.Logger,
	consumer consumer.MetricsConsumer,
	cfg configmodels.Processor,
) (processor.MetricsProcessor, error) {
	return nil, nil
}
