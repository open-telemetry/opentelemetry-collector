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
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/opencensusexporter"
)

func TestExportersBuilder_Build(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	oceFactory := opencensusexporter.NewFactory()
	factories.Exporters[oceFactory.Type()] = oceFactory
	cfg := &configmodels.Config{
		Exporters: map[string]configmodels.Exporter{
			"opencensus": &opencensusexporter.Config{
				ExporterSettings: configmodels.ExporterSettings{
					NameVal: "opencensus",
					TypeVal: "opencensus",
				},
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: "0.0.0.0:12345",
				},
				NumWorkers: 2,
			},
		},

		Service: configmodels.Service{
			Pipelines: map[string]*configmodels.Pipeline{
				"trace": {
					Name:      "trace",
					InputType: configmodels.TracesDataType,
					Exporters: []string{"opencensus"},
				},
			},
		},
	}

	exporters, err := BuildExporters(zap.NewNop(), component.DefaultApplicationStartInfo(), cfg, factories.Exporters)

	assert.NoError(t, err)
	require.NotNil(t, exporters)

	e1 := exporters[cfg.Exporters["opencensus"]]

	// Ensure exporter has its fields correctly populated.
	require.NotNil(t, e1)
	assert.NotNil(t, e1.getTraceExporter())
	assert.Nil(t, e1.getMetricExporter())
	assert.Nil(t, e1.getLogExporter())

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
	exporters, err = BuildExporters(zap.NewNop(), component.DefaultApplicationStartInfo(), cfg, factories.Exporters)
	assert.NotNil(t, exporters)
	assert.NoError(t, err)

	e1 = exporters[cfg.Exporters["opencensus"]]

	// Ensure exporter has its fields correctly populated, ie Trace Exporter and
	// Metrics Exporter are nil.
	require.NotNil(t, e1)
	assert.Nil(t, e1.getTraceExporter())
	assert.Nil(t, e1.getMetricExporter())
	assert.Nil(t, e1.getLogExporter())

	// TODO: once we have an exporter that supports metrics data type test it too.
}

func TestExportersBuilder_BuildLogs(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.Nil(t, err)

	cfg := &configmodels.Config{
		Exporters: map[string]configmodels.Exporter{
			"exampleexporter": &componenttest.ExampleExporter{
				ExporterSettings: configmodels.ExporterSettings{
					NameVal: "exampleexporter",
					TypeVal: "exampleexporter",
				},
			},
		},

		Service: configmodels.Service{
			Pipelines: map[string]*configmodels.Pipeline{
				"logs": {
					Name:      "logs",
					InputType: "logs",
					Exporters: []string{"exampleexporter"},
				},
			},
		},
	}

	exporters, err := BuildExporters(zap.NewNop(), component.DefaultApplicationStartInfo(), cfg, factories.Exporters)

	assert.NoError(t, err)
	require.NotNil(t, exporters)

	e1 := exporters[cfg.Exporters["exampleexporter"]]

	// Ensure exporter has its fields correctly populated.
	require.NotNil(t, e1)
	assert.NotNil(t, e1.getLogExporter())
	assert.Nil(t, e1.getTraceExporter())
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
	exporters, err = BuildExporters(zap.NewNop(), component.DefaultApplicationStartInfo(), cfg, factories.Exporters)
	assert.NotNil(t, exporters)
	assert.Nil(t, err)

	e1 = exporters[cfg.Exporters["exampleexporter"]]

	// Ensure exporter has its fields correctly populated, ie Trace Exporter and
	// Metrics Exporter are nil.
	require.NotNil(t, e1)
	assert.Nil(t, e1.getTraceExporter())
	assert.Nil(t, e1.getMetricExporter())
	assert.Nil(t, e1.getLogExporter())
}

func TestExportersBuilder_StartAll(t *testing.T) {
	exporters := make(Exporters)
	expCfg := &configmodels.ExporterSettings{}
	traceExporter := &componenttest.ExampleExporterConsumer{}
	metricExporter := &componenttest.ExampleExporterConsumer{}
	logsExporter := &componenttest.ExampleExporterConsumer{}
	exporters[expCfg] = &builtExporter{
		logger: zap.NewNop(),
		expByDataType: map[configmodels.DataType]component.Exporter{
			configmodels.TracesDataType:  traceExporter,
			configmodels.MetricsDataType: metricExporter,
			configmodels.LogsDataType:    logsExporter,
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

func TestExportersBuilder_StopAll(t *testing.T) {
	exporters := make(Exporters)
	expCfg := &configmodels.ExporterSettings{}
	traceExporter := &componenttest.ExampleExporterConsumer{}
	metricExporter := &componenttest.ExampleExporterConsumer{}
	logsExporter := &componenttest.ExampleExporterConsumer{}
	exporters[expCfg] = &builtExporter{
		logger: zap.NewNop(),
		expByDataType: map[configmodels.DataType]component.Exporter{
			configmodels.TracesDataType:  traceExporter,
			configmodels.MetricsDataType: metricExporter,
			configmodels.LogsDataType:    logsExporter,
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

func TestExportersBuilder_ErrorOnNilExporter(t *testing.T) {
	bf := newBadExporterFactory()
	fm := map[configmodels.Type]component.ExporterFactory{
		bf.Type(): bf,
	}

	pipelines := []*configmodels.Pipeline{
		{
			Name:      "trace",
			InputType: configmodels.TracesDataType,
			Exporters: []string{string(bf.Type())},
		},
		{
			Name:      "metrics",
			InputType: configmodels.MetricsDataType,
			Exporters: []string{string(bf.Type())},
		},
		{
			Name:      "logs",
			InputType: configmodels.LogsDataType,
			Exporters: []string{string(bf.Type())},
		},
	}

	for _, pipeline := range pipelines {
		t.Run(pipeline.Name, func(t *testing.T) {

			cfg := &configmodels.Config{
				Exporters: map[string]configmodels.Exporter{
					string(bf.Type()): &configmodels.ExporterSettings{
						TypeVal: bf.Type(),
					},
				},

				Service: configmodels.Service{
					Pipelines: map[string]*configmodels.Pipeline{
						pipeline.Name: pipeline,
					},
				},
			}

			exporters, err := BuildExporters(zap.NewNop(), component.DefaultApplicationStartInfo(), cfg, fm)
			assert.Error(t, err)
			assert.Zero(t, len(exporters))
		})
	}
}

func newBadExporterFactory() component.ExporterFactory {
	return exporterhelper.NewFactory("bf", func() configmodels.Exporter {
		return &configmodels.ExporterSettings{}
	})
}
