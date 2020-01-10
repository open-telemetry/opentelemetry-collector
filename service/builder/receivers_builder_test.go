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

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/processor/attributesprocessor"
	"github.com/open-telemetry/opentelemetry-collector/processor/processortest"
	"github.com/open-telemetry/opentelemetry-collector/receiver"
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
	factories, err := config.ExampleComponents()
	assert.Nil(t, err)

	attrFactory := &attributesprocessor.Factory{}
	factories.Processors[attrFactory.Type()] = attrFactory
	cfg, err := config.LoadConfigFile(t, "testdata/pipelines_builder.yaml", factories)
	require.Nil(t, err)

	// Build the pipeline
	allExporters, err := NewExportersBuilder(zap.NewNop(), cfg, factories.Exporters).Build()
	assert.NoError(t, err)
	pipelineProcessors, err := NewPipelinesBuilder(zap.NewNop(), cfg, allExporters, factories.Processors).Build()
	assert.NoError(t, err)
	receivers, err := NewReceiversBuilder(zap.NewNop(), cfg, pipelineProcessors, factories.Receivers).Build()

	assert.NoError(t, err)
	require.NotNil(t, receivers)

	receiver := receivers[cfg.Receivers[test.receiverName]]

	// Ensure receiver has its fields correctly populated.
	require.NotNil(t, receiver)

	assert.NotNil(t, receiver.receiver)

	// Compose the list of created exporters.
	var exporters []*builtExporter
	for _, name := range test.exporterNames {
		// Ensure exporter is created.
		exp := allExporters[cfg.Exporters[name]]
		require.NotNil(t, exp)
		exporters = append(exporters, exp)
	}

	// Send TraceData via receiver and verify that all exporters of the pipeline receive it.

	// First check that there are no traces in the exporters yet.
	for _, exporter := range exporters {
		consumer := exporter.te.(*config.ExampleExporterConsumer)
		require.Equal(t, len(consumer.Traces), 0)
		require.Equal(t, len(consumer.Metrics), 0)
	}

	// Send one trace.
	name := tracepb.TruncatableString{Value: "testspanname"}
	traceData := consumerdata.TraceData{
		SourceFormat: "test-source-format",
		Spans: []*tracepb.Span{
			{Name: &name},
		},
	}
	if test.hasTraces {
		traceProducer := receiver.receiver.(*config.ExampleReceiverProducer)
		traceProducer.TraceConsumer.ConsumeTraceData(context.Background(), traceData)
	}

	metricsData := consumerdata.MetricsData{
		Metrics: []*metricspb.Metric{
			{MetricDescriptor: &metricspb.MetricDescriptor{Name: "testmetric"}},
		},
	}
	if test.hasMetrics {
		metricsProducer := receiver.receiver.(*config.ExampleReceiverProducer)
		metricsProducer.MetricsConsumer.ConsumeMetricsData(context.Background(), metricsData)
	}

	// Now verify received data.
	for _, name := range test.exporterNames {
		// Check that the data is received by exporter.
		exporter := allExporters[cfg.Exporters[name]]

		// Validate traces.
		if test.hasTraces {
			var spanDuplicationCount int
			if test.spanDuplicationByExporter != nil {
				spanDuplicationCount = test.spanDuplicationByExporter[name]
			} else {
				spanDuplicationCount = 1
			}

			traceConsumer := exporter.te.(*config.ExampleExporterConsumer)
			require.Equal(t, spanDuplicationCount, len(traceConsumer.Traces))

			for i := 0; i < spanDuplicationCount; i++ {
				assertEqualTraceData(t, traceData, traceConsumer.Traces[i])

				// Check that the span was processed by "attributes" processor and an
				// attribute was added.
				assert.Equal(t, int64(12345),
					traceConsumer.Traces[i].Spans[0].Attributes.AttributeMap["attr1"].GetIntValue())
			}
		}

		// Validate metrics.
		if test.hasMetrics {
			metricsConsumer := exporter.me.(*config.ExampleExporterConsumer)
			require.Equal(t, 1, len(metricsConsumer.Metrics))
			assertEqualMetricsData(t, metricsData, metricsConsumer.Metrics[0])
		}
	}
}

func TestReceiversBuilder_DataTypeError(t *testing.T) {
	factories, err := config.ExampleComponents()
	assert.Nil(t, err)

	attrFactory := &attributesprocessor.Factory{}
	factories.Processors[attrFactory.Type()] = attrFactory
	cfg, err := config.LoadConfigFile(t, "testdata/pipelines_builder.yaml", factories)
	assert.Nil(t, err)

	// Make examplereceiver to "unsupport" trace data type.
	receiver := cfg.Receivers["examplereceiver"]
	receiver.(*config.ExampleReceiver).FailTraceCreation = true

	// Build the pipeline
	allExporters, err := NewExportersBuilder(zap.NewNop(), cfg, factories.Exporters).Build()
	assert.NoError(t, err)
	pipelineProcessors, err := NewPipelinesBuilder(zap.NewNop(), cfg, allExporters, factories.Processors).Build()
	assert.NoError(t, err)
	receivers, err := NewReceiversBuilder(zap.NewNop(), cfg, pipelineProcessors, factories.Receivers).Build()

	// This should fail because "examplereceiver" is attached to "traces" pipeline
	// which is a configuration error.
	assert.NotNil(t, err)
	assert.Nil(t, receivers)
}

func TestReceiversBuilder_StartAll(t *testing.T) {
	receivers := make(Receivers)
	rcvCfg := &configmodels.ReceiverSettings{}

	receiver := &config.ExampleReceiverProducer{}

	receivers[rcvCfg] = &builtReceiver{
		receiver: receiver,
	}

	assert.Equal(t, false, receiver.Started)

	mh := component.NewMockHost()
	err := receivers.StartAll(zap.NewNop(), mh)
	assert.Nil(t, err)

	assert.Equal(t, true, receiver.Started)
}

func TestReceiversBuilder_StopAll(t *testing.T) {
	receivers := make(Receivers)
	rcvCfg := &configmodels.ReceiverSettings{}

	receiver := &config.ExampleReceiverProducer{}

	receivers[rcvCfg] = &builtReceiver{
		receiver: receiver,
	}

	assert.Equal(t, false, receiver.Stopped)

	receivers.StopAll()

	assert.Equal(t, true, receiver.Stopped)
}

func TestReceiversBuilder_ErrorOnNilReceiver(t *testing.T) {
	factories, err := config.ExampleComponents()
	assert.Nil(t, err)

	npf := &processortest.NopProcessorFactory{}
	factories.Processors[npf.Type()] = npf

	bf := &badReceiverFactory{}
	factories.Receivers[bf.Type()] = bf

	cfg, err := config.LoadConfigFile(t, "testdata/bad_receiver_factory.yaml", factories)
	require.Nil(t, err)

	// Build the pipeline
	allExporters, err := NewExportersBuilder(zap.NewNop(), cfg, factories.Exporters).Build()
	assert.NoError(t, err)
	pipelineProcessors, err := NewPipelinesBuilder(zap.NewNop(), cfg, allExporters, factories.Processors).Build()
	assert.NoError(t, err)

	// First test only trace receivers by removing the metrics pipeline.
	metricsPipeline := cfg.Service.Pipelines["metrics"]
	delete(cfg.Service.Pipelines, "metrics")
	require.Equal(t, 1, len(cfg.Service.Pipelines))

	receivers, err := NewReceiversBuilder(zap.NewNop(), cfg, pipelineProcessors, factories.Receivers).Build()
	assert.Error(t, err)
	assert.Zero(t, len(receivers))

	// Now test the metric pipeline.
	delete(cfg.Service.Pipelines, "traces")
	cfg.Service.Pipelines["metrics"] = metricsPipeline
	require.Equal(t, 1, len(cfg.Service.Pipelines))

	receivers, err = NewReceiversBuilder(zap.NewNop(), cfg, pipelineProcessors, factories.Receivers).Build()
	assert.Error(t, err)
	assert.Zero(t, len(receivers))
}

// badReceiverFactory is a factory that returns no error but returns a nil object.
type badReceiverFactory struct{}

var _ receiver.Factory = (*badReceiverFactory)(nil)

func (b *badReceiverFactory) Type() string {
	return "bf"
}

func (b *badReceiverFactory) CreateDefaultConfig() configmodels.Receiver {
	return &configmodels.ReceiverSettings{}
}

func (b *badReceiverFactory) CustomUnmarshaler() receiver.CustomUnmarshaler {
	return nil
}

func (b *badReceiverFactory) CreateTraceReceiver(
	ctx context.Context,
	logger *zap.Logger,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumer,
) (receiver.TraceReceiver, error) {
	return nil, nil
}

func (b *badReceiverFactory) CreateMetricsReceiver(
	logger *zap.Logger,
	cfg configmodels.Receiver,
	consumer consumer.MetricsConsumer,
) (receiver.MetricsReceiver, error) {
	return nil, nil
}
