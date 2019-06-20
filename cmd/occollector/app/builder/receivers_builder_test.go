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

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"

	"github.com/open-telemetry/opentelemetry-service/data"
	"github.com/open-telemetry/opentelemetry-service/internal/configmodels"
	"github.com/open-telemetry/opentelemetry-service/internal/configv2"
)

type testCase struct {
	name                      string
	receiverName              string
	exporterNames             []string
	spanDuplicationByExporter map[string]int
	hasTraces                 bool
	hasMetrics                bool
}

func TestReceiversBuilder_Build(t *testing.T) {
	tests := []testCase{
		{
			name:          "one-exporter",
			receiverName:  "examplereceiver",
			exporterNames: []string{"exampleexporter"},
			hasTraces:     true,
			hasMetrics:    true,
		},
		{
			name:          "multi-exporter",
			receiverName:  "examplereceiver/2",
			exporterNames: []string{"exampleexporter", "exampleexporter/2"},
			hasTraces:     true,
		},
		{
			name:          "multi-metrics-receiver",
			receiverName:  "examplereceiver/3",
			exporterNames: []string{"exampleexporter", "exampleexporter/2"},
			hasTraces:     false,
			hasMetrics:    true,
		},
		{
			name:          "multi-receiver-multi-exporter",
			receiverName:  "examplereceiver/multi",
			exporterNames: []string{"exampleexporter", "exampleexporter/2"},

			// Check pipelines_builder.yaml to understand this case.
			// We have 2 pipelines, one exporting to one exporter, the other
			// exporting to both exporters, so we expect a duplication on
			// one of the exporters, but not on the other.
			spanDuplicationByExporter: map[string]int{
				"exampleexporter": 2, "exampleexporter/2": 1,
			},
			hasTraces: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testReceivers(t, test)
		})
	}
}

func testReceivers(
	t *testing.T,
	test testCase,
) {
	// Load the config
	config, err := configv2.LoadConfigFile(t, "testdata/pipelines_builder.yaml")
	require.Nil(t, err)

	// Build the pipeline
	allExporters, err := NewExportersBuilder(zap.NewNop(), config).Build()
	pipelineProcessors, err := NewPipelinesBuilder(zap.NewNop(), config, allExporters).Build()
	receivers, err := NewReceiversBuilder(zap.NewNop(), config, pipelineProcessors).Build()

	assert.NoError(t, err)
	require.NotNil(t, receivers)

	receiver := receivers[config.Receivers[test.receiverName]]

	// Ensure receiver has its fields correctly populated.
	require.NotNil(t, receiver)

	if test.hasTraces {
		assert.NotNil(t, receiver.trace)
	} else {
		assert.Nil(t, receiver.trace)
	}

	if test.hasMetrics {
		assert.NotNil(t, receiver.metrics)
	} else {
		assert.Nil(t, receiver.metrics)
	}

	// Compose the list of created exporters.
	var exporters []*builtExporter
	for _, name := range test.exporterNames {
		// Ensure exporter is created.
		exp := allExporters[config.Exporters[name]]
		require.NotNil(t, exp)
		exporters = append(exporters, exp)
	}

	// Send TraceData via receiver and verify that all exporters of the pipeline receive it.

	// First check that there are no traces in the exporters yet.
	for _, exporter := range exporters {
		consumer := exporter.tc.(*configv2.ExampleExporterConsumer)
		require.Equal(t, len(consumer.Traces), 0)
		require.Equal(t, len(consumer.Metrics), 0)
	}

	// Send one trace.
	name := tracepb.TruncatableString{Value: "testspanname"}
	traceData := data.TraceData{
		SourceFormat: "test-source-format",
		Spans: []*tracepb.Span{
			{Name: &name},
		},
	}
	if test.hasTraces {
		traceProducer := receiver.trace.(*configv2.ExampleReceiverProducer)
		traceProducer.TraceConsumer.ConsumeTraceData(context.Background(), traceData)
	}

	metricsData := data.MetricsData{
		Metrics: []*metricspb.Metric{
			{MetricDescriptor: &metricspb.MetricDescriptor{Name: "testmetric"}},
		},
	}
	if test.hasMetrics {
		metricsProducer := receiver.metrics.(*configv2.ExampleReceiverProducer)
		metricsProducer.MetricsConsumer.ConsumeMetricsData(context.Background(), metricsData)
	}

	// Now verify received data.
	for _, name := range test.exporterNames {
		// Check that the data is received by exporter.
		exporter := allExporters[config.Exporters[name]]

		// Validate traces.
		if test.hasTraces {
			var spanDuplicationCount int
			if test.spanDuplicationByExporter != nil {
				spanDuplicationCount = test.spanDuplicationByExporter[name]
			} else {
				spanDuplicationCount = 1
			}

			traceConsumer := exporter.tc.(*configv2.ExampleExporterConsumer)
			require.Equal(t, spanDuplicationCount, len(traceConsumer.Traces))

			for i := 0; i < spanDuplicationCount; i++ {
				assert.Equal(t, traceData, traceConsumer.Traces[i])

				// Check that the span was processed by "attributes" processor and an
				// attribute was added.
				assert.Equal(t, int64(12345),
					traceConsumer.Traces[i].Spans[0].Attributes.AttributeMap["attr1"].GetIntValue())
			}
		}

		// Validate metrics.
		if test.hasMetrics {
			metricsConsumer := exporter.mc.(*configv2.ExampleExporterConsumer)
			require.Equal(t, 1, len(metricsConsumer.Metrics))
			assert.Equal(t, metricsData, metricsConsumer.Metrics[0])
		}
	}
}

func TestReceiversBuilder_DataTypeError(t *testing.T) {
	config, err := configv2.LoadConfigFile(t, "testdata/pipelines_builder.yaml")
	require.Nil(t, err)

	// Make examplereceiver to "unsupport" trace data type.
	receiver := config.Receivers["examplereceiver"]
	receiver.(*configv2.ExampleReceiver).FailTraceCreation = true

	// Build the pipeline
	allExporters, err := NewExportersBuilder(zap.NewNop(), config).Build()
	pipelineProcessors, err := NewPipelinesBuilder(zap.NewNop(), config, allExporters).Build()
	receivers, err := NewReceiversBuilder(zap.NewNop(), config, pipelineProcessors).Build()

	// This should fail because "examplereceiver" is attached to "traces" pipeline
	// which is a configuration error.
	assert.NotNil(t, err)
	assert.Nil(t, receivers)
}

func TestReceiversBuilder_StartAll(t *testing.T) {
	receivers := make(Receivers)
	rcvCfg := &configmodels.ReceiverSettings{}

	receiver := &configv2.ExampleReceiverProducer{}

	receivers[rcvCfg] = &builtReceiver{
		trace:   receiver,
		metrics: receiver,
	}

	assert.Equal(t, false, receiver.TraceStarted)
	assert.Equal(t, false, receiver.MetricsStarted)

	ch := make(chan error)
	err := receivers.StartAll(zap.NewNop(), ch)
	assert.Nil(t, err)

	assert.Equal(t, true, receiver.TraceStarted)
	assert.Equal(t, true, receiver.MetricsStarted)
}

func TestReceiversBuilder_StopAll(t *testing.T) {
	receivers := make(Receivers)
	rcvCfg := &configmodels.ReceiverSettings{}

	receiver := &configv2.ExampleReceiverProducer{}

	receivers[rcvCfg] = &builtReceiver{
		trace:   receiver,
		metrics: receiver,
	}

	assert.Equal(t, false, receiver.TraceStopped)
	assert.Equal(t, false, receiver.MetricsStopped)

	receivers.StopAll()

	assert.Equal(t, true, receiver.TraceStopped)
	assert.Equal(t, true, receiver.MetricsStopped)
}
