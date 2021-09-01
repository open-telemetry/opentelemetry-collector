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
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/internal/testcomponents"
)

func TestBuildExporters(t *testing.T) {
	factories, err := testcomponents.ExampleComponents()
	assert.NoError(t, err)

	otlpFactory := otlpexporter.NewFactory()
	factories.Exporters[otlpFactory.Type()] = otlpFactory
	cfg := &config.Config{
		Exporters: map[config.ComponentID]config.Exporter{
			config.NewID("otlp"): &otlpexporter.Config{
				ExporterSettings: config.NewExporterSettings(config.NewID("otlp")),
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: "0.0.0.0:12345",
				},
			},
		},

		Service: config.Service{
			Pipelines: map[string]*config.Pipeline{
				"trace": {
					Name:      "trace",
					InputType: config.TracesDataType,
					Exporters: []config.ComponentID{config.NewID("otlp")},
				},
			},
		},
	}

	exporters, err := BuildExporters(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, factories.Exporters)

	assert.NoError(t, err)
	require.NotNil(t, exporters)

	e1 := exporters[config.NewID("otlp")]

	// Ensure exporter has its fields correctly populated.
	require.NotNil(t, e1)
	assert.NotNil(t, e1.getTracesExporter())
	assert.Nil(t, e1.getMetricExporter())
	assert.Nil(t, e1.getLogExporter())

	// Ensure it can be started.
	assert.NoError(t, exporters.StartAll(context.Background(), componenttest.NewNopHost()))

	// Ensure it can be stopped.
	if err = e1.Shutdown(context.Background()); err != nil {
		// TODO Find a better way to handle this case
		// Since the endpoint of otlp exporter doesn't actually exist, e1 may
		// already stop because it cannot connect.
		// The test should stop running if this isn't the error cause.
		require.EqualError(t, err, "rpc error: code = Canceled desc = grpc: the client connection is closing")
	}

	// Remove the pipeline so that the exporter is not attached to any pipeline.
	// This should result in creating an exporter that has none of consumption
	// functions set.
	delete(cfg.Service.Pipelines, "trace")
	exporters, err = BuildExporters(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, factories.Exporters)
	assert.NotNil(t, exporters)
	assert.NoError(t, err)

	e1 = exporters[config.NewID("otlp")]

	// Ensure exporter has its fields correctly populated, ie Trace Exporter and
	// Metrics Exporter are nil.
	require.NotNil(t, e1)
	assert.Nil(t, e1.getTracesExporter())
	assert.Nil(t, e1.getMetricExporter())
	assert.Nil(t, e1.getLogExporter())

	// TODO: once we have an exporter that supports metrics data type test it too.
}

func TestBuildExporters_BuildLogs(t *testing.T) {
	factories, err := testcomponents.ExampleComponents()
	assert.Nil(t, err)

	cfg := &config.Config{
		Exporters: map[config.ComponentID]config.Exporter{
			config.NewID("exampleexporter"): &testcomponents.ExampleExporter{
				ExporterSettings: config.NewExporterSettings(config.NewID("exampleexporter")),
			},
		},

		Service: config.Service{
			Pipelines: map[string]*config.Pipeline{
				"logs": {
					Name:      "logs",
					InputType: "logs",
					Exporters: []config.ComponentID{config.NewID("exampleexporter")},
				},
			},
		},
	}

	exporters, err := BuildExporters(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, factories.Exporters)

	assert.NoError(t, err)
	require.NotNil(t, exporters)

	e1 := exporters[config.NewID("exampleexporter")]

	// Ensure exporter has its fields correctly populated.
	require.NotNil(t, e1)
	assert.NotNil(t, e1.getLogExporter())
	assert.Nil(t, e1.getTracesExporter())
	assert.Nil(t, e1.getMetricExporter())

	// Ensure it can be started.
	err = exporters.StartAll(context.Background(), componenttest.NewNopHost())
	assert.NoError(t, err)

	// Ensure it can be stopped.
	err = e1.Shutdown(context.Background())
	assert.NoError(t, err)

	// Remove the pipeline so that the exporter is not attached to any pipeline.
	// This should result in creating an exporter that has none of consumption
	// functions set.
	delete(cfg.Service.Pipelines, "logs")
	exporters, err = BuildExporters(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, factories.Exporters)
	assert.NotNil(t, exporters)
	assert.Nil(t, err)

	e1 = exporters[config.NewID("exampleexporter")]

	// Ensure exporter has its fields correctly populated, ie Trace Exporter and
	// Metrics Exporter are nil.
	require.NotNil(t, e1)
	assert.Nil(t, e1.getTracesExporter())
	assert.Nil(t, e1.getMetricExporter())
	assert.Nil(t, e1.getLogExporter())
}

func TestBuildExporters_StartStopAll(t *testing.T) {
	exporters := make(Exporters)
	traceExporter := &testcomponents.ExampleExporterConsumer{}
	metricExporter := &testcomponents.ExampleExporterConsumer{}
	logsExporter := &testcomponents.ExampleExporterConsumer{}
	exporters[config.NewID("example")] = &builtExporter{
		logger: zap.NewNop(),
		expByDataType: map[config.DataType]component.Exporter{
			config.TracesDataType:  traceExporter,
			config.MetricsDataType: metricExporter,
			config.LogsDataType:    logsExporter,
		},
	}
	assert.False(t, traceExporter.ExporterStarted)
	assert.False(t, metricExporter.ExporterStarted)
	assert.False(t, logsExporter.ExporterStarted)

	assert.NoError(t, exporters.StartAll(context.Background(), componenttest.NewNopHost()))
	assert.True(t, traceExporter.ExporterStarted)
	assert.True(t, metricExporter.ExporterStarted)
	assert.True(t, logsExporter.ExporterStarted)

	assert.NoError(t, exporters.ShutdownAll(context.Background()))
	assert.True(t, traceExporter.ExporterShutdown)
	assert.True(t, metricExporter.ExporterShutdown)
	assert.True(t, logsExporter.ExporterShutdown)
}

func TestBuildExporters_NotSupportedDataType(t *testing.T) {
	factories := createTestFactories()

	tests := []struct {
		configFile string
	}{
		{
			configFile: "not_supported_exporter_logs.yaml",
		},
		{
			configFile: "not_supported_exporter_metrics.yaml",
		},
		{
			configFile: "not_supported_exporter_traces.yaml",
		},
	}

	for _, test := range tests {
		t.Run(test.configFile, func(t *testing.T) {

			cfg, err := configtest.LoadConfigAndValidate(path.Join("testdata", test.configFile), factories)
			require.Nil(t, err)

			exporters, err := BuildExporters(zap.NewNop(), trace.NewNoopTracerProvider(), component.DefaultBuildInfo(), cfg, factories.Exporters)
			assert.Error(t, err)
			assert.Zero(t, len(exporters))
		})
	}
}
