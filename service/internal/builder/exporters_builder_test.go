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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/exporter/opencensusexporter"
	"go.opentelemetry.io/collector/internal/testcomponents"
)

func TestBuildExporters(t *testing.T) {
	factories, err := testcomponents.ExampleComponents()
	assert.NoError(t, err)

	oceFactory := opencensusexporter.NewFactory()
	factories.Exporters[oceFactory.Type()] = oceFactory
	factories.Exporters[exampleRldExporterFactory.Type()] = exampleRldExporterFactory
	factories.Exporters[exampleExporterFactory.Type()] = exampleExporterFactory
	cfg := &config.Config{
		Exporters: map[config.ComponentID]config.Exporter{
			config.NewID("opencensus"): &opencensusexporter.Config{
				ExporterSettings: config.NewExporterSettings(config.NewID("opencensus")),
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: "0.0.0.0:12345",
				},
				NumWorkers: 2,
			},
			config.NewID(expType):    createDefaultConfig(),
			config.NewID(rldExpType): createRldDefaultConfig(),
		},

		Service: config.Service{
			Pipelines: map[string]*config.Pipeline{
				"trace": {
					Name:      "trace",
					InputType: config.TracesDataType,
					Exporters: []config.ComponentID{config.NewID("opencensus"), createDefaultConfig().ID(), createRldDefaultConfig().ID()},
				},
			},
		},
	}

	exporters, err := BuildExporters(zap.NewNop(), component.DefaultBuildInfo(), cfg, factories.Exporters)

	assert.NoError(t, err)
	require.NotNil(t, exporters)

	e1 := exporters[config.NewID("opencensus")]

	// Ensure exporter has its fields correctly populated.
	require.NotNil(t, e1)
	assert.NotNil(t, e1.getTracesExporter())
	assert.Nil(t, e1.getMetricExporter())
	assert.Nil(t, e1.getLogExporter())

	e2 := exporters[config.NewID(expType)]
	require.NotNil(t, e2)
	assert.NotNil(t, e2.getTracesExporter())
	assert.Nil(t, e2.getMetricExporter())
	assert.Nil(t, e2.getLogExporter())

	e3 := exporters[config.NewID(rldExpType)]
	require.NotNil(t, e3)
	assert.NotNil(t, e3.getTracesExporter())
	assert.Nil(t, e3.getMetricExporter())
	assert.Nil(t, e3.getLogExporter())

	// Ensure it can be started.
	assert.NoError(t, exporters.StartAll(context.Background(), componenttest.NewNopHost()))

	// Ensure it can be stopped.
	if err = e1.Shutdown(context.Background()); err != nil {
		// TODO Find a better way to handle this case
		// Since the endpoint of opencensus exporter doesn't actually exist, e1 may
		// already stop because it cannot connect.
		// The test should stop running if this isn't the error cause.
		require.Equal(t, err.Error(), "rpc error: code = Canceled desc = grpc: the client connection is closing")
	}

	// Remove the pipeline so that the exporter is not attached to any pipeline.
	// This should result in creating an exporter that has none of consumption
	// functions set.
	delete(cfg.Service.Pipelines, "trace")
	exporters, err = BuildExporters(zap.NewNop(), component.DefaultBuildInfo(), cfg, factories.Exporters)
	assert.NotNil(t, exporters)
	assert.NoError(t, err)

	e1 = exporters[config.NewID("opencensus")]

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

	exporters, err := BuildExporters(zap.NewNop(), component.DefaultBuildInfo(), cfg, factories.Exporters)

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
	exporters, err = BuildExporters(zap.NewNop(), component.DefaultBuildInfo(), cfg, factories.Exporters)
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

func TestBuildExporters_StartAll(t *testing.T) {
	exporters := make(Exporters)
	expCfg := &config.ExporterSettings{}
	traceExporter := &testcomponents.ExampleExporterConsumer{}
	metricExporter := &testcomponents.ExampleExporterConsumer{}
	logsExporter := &testcomponents.ExampleExporterConsumer{}
	exporters[expCfg.ID()] = &builtExporter{
		logger: zap.NewNop(),
		expByDataType: map[config.DataType]component.Exporter{
			config.TracesDataType:  &exporterWrapper{config.TracesDataType, nil, traceExporter, nil, traceExporter, expCfg.ID(), zap.NewNop(), component.DefaultBuildInfo(), testcomponents.ExampleExporterFactory},
			config.MetricsDataType: &exporterWrapper{config.MetricsDataType, metricExporter, nil, nil, metricExporter, expCfg.ID(), zap.NewNop(), component.DefaultBuildInfo(), testcomponents.ExampleExporterFactory},
			config.LogsDataType:    &exporterWrapper{config.LogsDataType, nil, nil, logsExporter, logsExporter, expCfg.ID(), zap.NewNop(), component.DefaultBuildInfo(), testcomponents.ExampleExporterFactory},
		},
	}
	assert.False(t, traceExporter.ExporterStarted)
	assert.False(t, metricExporter.ExporterStarted)
	assert.False(t, logsExporter.ExporterStarted)

	assert.NoError(t, exporters.StartAll(context.Background(), componenttest.NewNopHost()))

	assert.True(t, traceExporter.ExporterStarted)
	assert.True(t, metricExporter.ExporterStarted)
	assert.True(t, logsExporter.ExporterStarted)
}

func TestBuildExporters_Reload(t *testing.T) {
	exporters := make(Exporters)

	factories, err := testcomponents.ExampleComponents()
	assert.NoError(t, err)
	factories.Exporters[exampleRldExporterFactory.Type()] = exampleRldExporterFactory
	factories.Exporters[exampleExporterFactory.Type()] = exampleExporterFactory

	expCfg := config.NewExporterSettings(config.NewID(expType))
	traceExporter := &exampleExporter{}
	metricExporter := &exampleExporter{}
	logExporter := &exampleExporter{}
	exporters[expCfg.ID()] = &builtExporter{
		logger: zap.NewNop(),
		expByDataType: map[config.DataType]component.Exporter{
			config.TracesDataType:  &exporterWrapper{config.TracesDataType, nil, traceExporter, nil, traceExporter, expCfg.ID(), zap.NewNop(), component.DefaultBuildInfo(), exampleExporterFactory},
			config.MetricsDataType: &exporterWrapper{config.MetricsDataType, metricExporter, nil, nil, metricExporter, expCfg.ID(), zap.NewNop(), component.DefaultBuildInfo(), exampleExporterFactory},
			config.LogsDataType:    &exporterWrapper{config.LogsDataType, nil, nil, logExporter, logExporter, expCfg.ID(), zap.NewNop(), component.DefaultBuildInfo(), exampleExporterFactory},
		},
	}

	assert.False(t, traceExporter.started)
	assert.False(t, metricExporter.started)
	assert.False(t, logExporter.started)

	rldExpCfg := config.NewExporterSettings(config.NewID(rldExpType))
	rldTraceExporter := &reloadableExporter{}
	rldMetricExporter := &reloadableExporter{}
	rldLogExporter := &reloadableExporter{}
	exporters[rldExpCfg.ID()] = &builtExporter{
		logger: zap.NewNop(),
		expByDataType: map[config.DataType]component.Exporter{
			config.TracesDataType:  &exporterWrapper{config.TracesDataType, nil, rldTraceExporter, nil, rldTraceExporter, rldExpCfg.ID(), zap.NewNop(), component.DefaultBuildInfo(), exampleRldExporterFactory},
			config.MetricsDataType: &exporterWrapper{config.MetricsDataType, rldMetricExporter, nil, nil, rldMetricExporter, rldExpCfg.ID(), zap.NewNop(), component.DefaultBuildInfo(), exampleRldExporterFactory},
			config.LogsDataType:    &exporterWrapper{config.LogsDataType, nil, nil, rldLogExporter, rldLogExporter, rldExpCfg.ID(), zap.NewNop(), component.DefaultBuildInfo(), exampleRldExporterFactory},
		},
	}

	assert.False(t, rldTraceExporter.started)
	assert.False(t, rldMetricExporter.started)
	assert.False(t, rldLogExporter.started)

	assert.NoError(t, exporters.StartAll(context.Background(), componenttest.NewNopHost()))
	assert.True(t, traceExporter.started)
	assert.True(t, metricExporter.started)
	assert.True(t, logExporter.started)
	assert.True(t, rldTraceExporter.started)
	assert.True(t, rldMetricExporter.started)
	assert.True(t, rldLogExporter.started)

	cfg := &config.Config{
		Exporters: map[config.ComponentID]config.Exporter{
			config.NewID(expType):    &expCfg,
			config.NewID(rldExpType): &rldExpCfg,
		},
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = exporters.ReloadExporters(context.Background(), zap.NewNop(), component.DefaultBuildInfo(), cfg, factories.Exporters, componenttest.NewNopHost())
		assert.NoError(t, err)
	}()
	wg.Wait()
	assert.False(t, exporters[expCfg.ID()].getLogExporter().(*exporterWrapper).lc == logExporter)
	assert.False(t, exporters[expCfg.ID()].getMetricExporter().(*exporterWrapper).mc == metricExporter)
	assert.False(t, exporters[expCfg.ID()].getTracesExporter().(*exporterWrapper).tc == traceExporter)
	assert.True(t, exporters[rldExpCfg.ID()].getLogExporter().(*exporterWrapper).lc == rldLogExporter)
	assert.True(t, exporters[rldExpCfg.ID()].getMetricExporter().(*exporterWrapper).mc == rldMetricExporter)
	assert.True(t, exporters[rldExpCfg.ID()].getTracesExporter().(*exporterWrapper).tc == rldTraceExporter)
	assert.Equal(t, exporters[rldExpCfg.ID()].getLogExporter().(*exporterWrapper).lc.(*reloadableExporter).reloadCount, 1)
	assert.Equal(t, exporters[rldExpCfg.ID()].getMetricExporter().(*exporterWrapper).mc.(*reloadableExporter).reloadCount, 1)
	assert.Equal(t, exporters[rldExpCfg.ID()].getTracesExporter().(*exporterWrapper).tc.(*reloadableExporter).reloadCount, 1)
}

func TestBuildExporters_StopAll(t *testing.T) {
	exporters := make(Exporters)
	expCfg := &config.ExporterSettings{}
	traceExporter := &testcomponents.ExampleExporterConsumer{}
	metricExporter := &testcomponents.ExampleExporterConsumer{}
	logsExporter := &testcomponents.ExampleExporterConsumer{}
	exporters[expCfg.ID()] = &builtExporter{
		logger: zap.NewNop(),
		expByDataType: map[config.DataType]component.Exporter{
			config.TracesDataType:  &exporterWrapper{config.TracesDataType, nil, traceExporter, nil, traceExporter, expCfg.ID(), zap.NewNop(), component.DefaultBuildInfo(), testcomponents.ExampleExporterFactory},
			config.MetricsDataType: &exporterWrapper{config.MetricsDataType, metricExporter, nil, nil, metricExporter, expCfg.ID(), zap.NewNop(), component.DefaultBuildInfo(), testcomponents.ExampleExporterFactory},
			config.LogsDataType:    &exporterWrapper{config.LogsDataType, nil, nil, logsExporter, logsExporter, expCfg.ID(), zap.NewNop(), component.DefaultBuildInfo(), testcomponents.ExampleExporterFactory},
		},
	}
	assert.False(t, traceExporter.ExporterShutdown)
	assert.False(t, metricExporter.ExporterShutdown)
	assert.False(t, logsExporter.ExporterShutdown)
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

			cfg, err := configtest.LoadConfigFile(t, path.Join("testdata", test.configFile), factories)
			require.Nil(t, err)

			exporters, err := BuildExporters(zap.NewNop(), component.DefaultBuildInfo(), cfg, factories.Exporters)
			assert.Error(t, err)
			assert.Zero(t, len(exporters))
		})
	}
}
