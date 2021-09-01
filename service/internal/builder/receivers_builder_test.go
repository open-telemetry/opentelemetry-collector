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

type testCase struct {
	name                      string
	receiverID                config.ComponentID
	exporterIDs               []config.ComponentID
	spanDuplicationByExporter map[config.ComponentID]int
	hasTraces                 bool
	hasMetrics                bool
}

func TestBuildReceivers(t *testing.T) {
	tests := []testCase{
		{
			name:        "one-exporter",
			receiverID:  config.NewID("examplereceiver"),
			exporterIDs: []config.ComponentID{config.NewID("exampleexporter")},
			hasTraces:   true,
			hasMetrics:  true,
		},
		{
			name:        "multi-exporter",
			receiverID:  config.NewIDWithName("examplereceiver", "2"),
			exporterIDs: []config.ComponentID{config.NewID("exampleexporter"), config.NewIDWithName("exampleexporter", "2")},
			hasTraces:   true,
		},
		{
			name:        "multi-metrics-receiver",
			receiverID:  config.NewIDWithName("examplereceiver", "3"),
			exporterIDs: []config.ComponentID{config.NewID("exampleexporter"), config.NewIDWithName("exampleexporter", "2")},
			hasTraces:   false,
			hasMetrics:  true,
		},
		{
			name:        "multi-receiver-multi-exporter",
			receiverID:  config.NewIDWithName("examplereceiver", "multi"),
			exporterIDs: []config.ComponentID{config.NewID("exampleexporter"), config.NewIDWithName("exampleexporter", "2")},

			// Check pipelines_builder.yaml to understand this case.
			// We have 2 pipelines, one exporting to one exporter, the other
			// exporting to both exporters, so we expect a duplication on
			// one of the exporters, but not on the other.
			spanDuplicationByExporter: map[config.ComponentID]int{
				config.NewID("exampleexporter"): 2, config.NewIDWithName("exampleexporter", "2"): 1,
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

func testReceivers(t *testing.T, test testCase) {
	factories, err := testcomponents.ExampleComponents()
	assert.NoError(t, err)

	cfg, err := configtest.LoadConfigAndValidate("testdata/pipelines_builder.yaml", factories)
	require.NoError(t, err)

	// Build the pipeline
	allExporters, err := BuildExporters(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, factories.Exporters)
	assert.NoError(t, err)
	pipelineProcessors, err := BuildPipelines(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, allExporters, factories.Processors)
	assert.NoError(t, err)
	receivers, err := BuildReceivers(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, pipelineProcessors, factories.Receivers)

	assert.NoError(t, err)
	require.NotNil(t, receivers)

	receiver := receivers[test.receiverID]

	// Ensure receiver has its fields correctly populated.
	require.NotNil(t, receiver)

	assert.NotNil(t, receiver.receiver)

	// Compose the list of created exporters.
	var exporters []*builtExporter
	for _, expID := range test.exporterIDs {
		// Ensure exporter is created.
		exp := allExporters[expID]
		require.NotNil(t, exp)
		exporters = append(exporters, exp)
	}

	// Send TraceData via receiver and verify that all exporters of the pipeline receive it.

	// First check that there are no traces in the exporters yet.
	for _, exporter := range exporters {
		consumer := exporter.getTracesExporter().(*testcomponents.ExampleExporterConsumer)
		require.Equal(t, len(consumer.Traces), 0)
		require.Equal(t, len(consumer.Metrics), 0)
	}

	td := testdata.GenerateTracesOneSpan()
	if test.hasTraces {
		traceProducer := receiver.receiver.(*testcomponents.ExampleReceiverProducer)
		assert.NoError(t, traceProducer.ConsumeTraces(context.Background(), td))
	}

	md := testdata.GenerateMetricsOneMetric()
	if test.hasMetrics {
		metricsProducer := receiver.receiver.(*testcomponents.ExampleReceiverProducer)
		assert.NoError(t, metricsProducer.ConsumeMetrics(context.Background(), md))
	}

	// Now verify received data.
	for _, expID := range test.exporterIDs {
		// Check that the data is received by exporter.
		exporter := allExporters[expID]

		// Validate traces.
		if test.hasTraces {
			var spanDuplicationCount int
			if test.spanDuplicationByExporter != nil {
				spanDuplicationCount = test.spanDuplicationByExporter[expID]
			} else {
				spanDuplicationCount = 1
			}

			traceConsumer := exporter.getTracesExporter().(*testcomponents.ExampleExporterConsumer)
			require.Equal(t, spanDuplicationCount, len(traceConsumer.Traces))

			for i := 0; i < spanDuplicationCount; i++ {
				assert.EqualValues(t, td, traceConsumer.Traces[i])
			}
		}

		// Validate metrics.
		if test.hasMetrics {
			metricsConsumer := exporter.getMetricExporter().(*testcomponents.ExampleExporterConsumer)
			require.Equal(t, 1, len(metricsConsumer.Metrics))
			assert.EqualValues(t, md, metricsConsumer.Metrics[0])
		}
	}
}

func TestBuildReceivers_BuildCustom(t *testing.T) {
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

			// Build the pipeline
			allExporters, err := BuildExporters(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, factories.Exporters)
			if test.shouldFail {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			pipelineProcessors, err := BuildPipelines(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, allExporters, factories.Processors)
			assert.NoError(t, err)
			receivers, err := BuildReceivers(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, pipelineProcessors, factories.Receivers)

			assert.NoError(t, err)
			require.NotNil(t, receivers)

			receiver := receivers[config.NewID("examplereceiver")]

			// Ensure receiver has its fields correctly populated.
			require.NotNil(t, receiver)

			assert.NotNil(t, receiver.receiver)

			// Compose the list of created exporters.
			exporterIDs := []config.ComponentID{config.NewID("exampleexporter")}
			var exporters []*builtExporter
			for _, expID := range exporterIDs {
				// Ensure exporter is created.
				exp := allExporters[expID]
				require.NotNil(t, exp)
				exporters = append(exporters, exp)
			}

			// Send Data via receiver and verify that all exporters of the pipeline receive it.

			// First check that there are no traces in the exporters yet.
			for _, exporter := range exporters {
				consumer := exporter.getLogExporter().(*testcomponents.ExampleExporterConsumer)
				require.Equal(t, len(consumer.Logs), 0)
			}

			// Send one data.
			log := pdata.Logs{}
			producer := receiver.receiver.(*testcomponents.ExampleReceiverProducer)
			require.NoError(t, producer.ConsumeLogs(context.Background(), log))

			// Now verify received data.
			for _, expID := range exporterIDs {
				// Check that the data is received by exporter.
				exporter := allExporters[expID]

				// Validate exported data.
				consumer := exporter.getLogExporter().(*testcomponents.ExampleExporterConsumer)
				require.Equal(t, 1, len(consumer.Logs))
				assert.EqualValues(t, log, consumer.Logs[0])
			}
		})
	}
}

func TestBuildReceivers_StartAll(t *testing.T) {
	receivers := make(Receivers)
	receiver := &testcomponents.ExampleReceiverProducer{}

	receivers[config.NewID("example")] = &builtReceiver{
		logger:   zap.NewNop(),
		receiver: receiver,
	}

	assert.False(t, receiver.Started)
	assert.NoError(t, receivers.StartAll(context.Background(), componenttest.NewNopHost()))
	assert.True(t, receiver.Started)

	assert.False(t, receiver.Stopped)
	assert.NoError(t, receivers.ShutdownAll(context.Background()))
	assert.True(t, receiver.Stopped)
}

func TestBuildReceivers_Unused(t *testing.T) {
	factories, err := testcomponents.ExampleComponents()
	assert.NoError(t, err)

	cfg, err := configtest.LoadConfigAndValidate("testdata/unused_receiver.yaml", factories)
	assert.NoError(t, err)

	// Build the pipeline
	allExporters, err := BuildExporters(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, factories.Exporters)
	assert.NoError(t, err)
	pipelineProcessors, err := BuildPipelines(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, allExporters, factories.Processors)
	assert.NoError(t, err)
	receivers, err := BuildReceivers(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, pipelineProcessors, factories.Receivers)
	assert.NoError(t, err)
	assert.NotNil(t, receivers)

	assert.NoError(t, receivers.StartAll(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, receivers.ShutdownAll(context.Background()))
}

func TestBuildReceivers_NotSupportedDataType(t *testing.T) {
	factories := createTestFactories()

	tests := []struct {
		configFile string
	}{
		{
			configFile: "not_supported_receiver_logs.yaml",
		},
		{
			configFile: "not_supported_receiver_metrics.yaml",
		},
		{
			configFile: "not_supported_receiver_traces.yaml",
		},
	}

	for _, test := range tests {
		t.Run(test.configFile, func(t *testing.T) {

			cfg, err := configtest.LoadConfigAndValidate(path.Join("testdata", test.configFile), factories)
			assert.NoError(t, err)
			require.NotNil(t, cfg)

			allExporters, err := BuildExporters(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, factories.Exporters)
			assert.NoError(t, err)

			pipelineProcessors, err := BuildPipelines(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, allExporters, factories.Processors)
			assert.NoError(t, err)

			receivers, err := BuildReceivers(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, pipelineProcessors, factories.Receivers)
			assert.Error(t, err)
			assert.Zero(t, len(receivers))
		})
	}
}
