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

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"

	"github.com/open-telemetry/opentelemetry-service/config"
	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-service/processor/addattributesprocessor"
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

func testPipeline(t *testing.T, pipelineName string, exporterNames []string) {
	receiverFactories, processorsFactories, exporterFactories, err := config.ExampleComponents()
	assert.Nil(t, err)
	attrFactory := &addattributesprocessor.Factory{}
	processorsFactories[attrFactory.Type()] = attrFactory
	cfg, err := config.LoadConfigFile(
		t, "testdata/pipelines_builder.yaml", receiverFactories, processorsFactories, exporterFactories,
	)
	// Load the config
	require.Nil(t, err)

	// Build the pipeline
	allExporters, err := NewExportersBuilder(zap.NewNop(), cfg, exporterFactories).Build()
	assert.NoError(t, err)
	pipelineProcessors, err := NewPipelinesBuilder(zap.NewNop(), cfg, allExporters, processorsFactories).Build()

	assert.NoError(t, err)
	require.NotNil(t, pipelineProcessors)

	processor := pipelineProcessors[cfg.Pipelines[pipelineName]]

	// Ensure pipeline has its fields correctly populated.
	require.NotNil(t, processor)
	assert.NotNil(t, processor.tc)
	assert.Nil(t, processor.mc)

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
	}
	processor.tc.ConsumeTraceData(context.Background(), traceData)

	// Now verify received data.
	for _, consumer := range exporterConsumers {
		// Check that the trace is received by exporter.
		require.Equal(t, 1, len(consumer.Traces))
		assert.Equal(t, traceData, consumer.Traces[0])

		// Check that the span was processed by "attributes" processor and an
		// attribute was added.
		assert.Equal(t, int64(12345),
			consumer.Traces[0].Spans[0].Attributes.AttributeMap["attr1"].GetIntValue())
	}
}

func TestPipelinesBuilder_Error(t *testing.T) {
	receiverFactories, processorsFactories, exporterFactories, err := config.ExampleComponents()
	assert.Nil(t, err)
	attrFactory := &addattributesprocessor.Factory{}
	processorsFactories[attrFactory.Type()] = attrFactory
	cfg, err := config.LoadConfigFile(
		t, "testdata/pipelines_builder.yaml", receiverFactories, processorsFactories, exporterFactories,
	)
	require.Nil(t, err)

	// Corrupt the pipeline, change data type to metrics. We have to forcedly do it here
	// since there is no way to have such config loaded by LoadConfigFile, it would not
	// pass validation. We are doing this to test failure mode of PipelinesBuilder.
	pipeline := cfg.Pipelines["traces"]
	pipeline.InputType = configmodels.MetricsDataType

	exporters, err := NewExportersBuilder(zap.NewNop(), cfg, exporterFactories).Build()
	assert.NoError(t, err)

	// This should fail because "attributes" processor defined in the config does
	// not support metrics data type.
	_, err = NewPipelinesBuilder(zap.NewNop(), cfg, exporters, processorsFactories).Build()

	assert.NotNil(t, err)
}
