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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config"
	"github.com/open-telemetry/opentelemetry-collector/config/configgrpc"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/exporter/opencensusexporter"
	"github.com/open-telemetry/opentelemetry-collector/receiver/receivertest"
)

func TestExportersBuilder_Build(t *testing.T) {
	factories, err := config.ExampleComponents()
	assert.Nil(t, err)

	oceFactory := &opencensusexporter.Factory{}
	factories.Exporters[oceFactory.Type()] = oceFactory
	cfg := &configmodels.Config{
		Exporters: map[string]configmodels.Exporter{
			"opencensus": &opencensusexporter.Config{
				ExporterSettings: configmodels.ExporterSettings{
					NameVal: "opencensus",
					TypeVal: "opencensus",
				},
				GRPCSettings: configgrpc.GRPCSettings{
					Endpoint: "0.0.0.0:12345",
				},
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

	exporters, err := NewExportersBuilder(zap.NewNop(), cfg, factories.Exporters).Build()

	assert.NoError(t, err)
	require.NotNil(t, exporters)

	e1 := exporters[cfg.Exporters["opencensus"]]

	// Ensure exporter has its fields correctly populated.
	require.NotNil(t, e1)
	assert.NotNil(t, e1.te)
	assert.Nil(t, e1.me)

	// Ensure it can be started.
	mh := receivertest.NewMockHost()
	err = exporters.StartAll(zap.NewNop(), mh)
	assert.NoError(t, err)

	// Ensure it can be stopped.
	if err = e1.Shutdown(); err != nil {
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
	exporters, err = NewExportersBuilder(zap.NewNop(), cfg, factories.Exporters).Build()
	assert.NotNil(t, exporters)
	assert.Nil(t, err)

	e1 = exporters[cfg.Exporters["opencensus"]]

	// Ensure exporter has its fields correctly populated, ie Trace Exporter and
	// Metrics Exporter are nil.
	require.NotNil(t, e1)
	assert.Nil(t, e1.te)
	assert.Nil(t, e1.me)

	// TODO: once we have an exporter that supports metrics data type test it too.
}

func TestExportersBuilder_StartAll(t *testing.T) {
	exporters := make(Exporters)
	expCfg := &configmodels.ExporterSettings{}
	traceExporter := &config.ExampleExporterConsumer{}
	metricExporter := &config.ExampleExporterConsumer{}
	exporters[expCfg] = &builtExporter{
		te: traceExporter,
		me: metricExporter,
	}
	assert.False(t, traceExporter.ExporterStarted)
	assert.False(t, metricExporter.ExporterStarted)

	mh := receivertest.NewMockHost()
	err := exporters.StartAll(zap.NewNop(), mh)
	assert.NoError(t, err)

	assert.True(t, traceExporter.ExporterStarted)
	assert.True(t, metricExporter.ExporterStarted)
}

func TestExportersBuilder_StopAll(t *testing.T) {
	exporters := make(Exporters)
	expCfg := &configmodels.ExporterSettings{}
	traceExporter := &config.ExampleExporterConsumer{}
	metricExporter := &config.ExampleExporterConsumer{}
	exporters[expCfg] = &builtExporter{
		te: traceExporter,
		me: metricExporter,
	}
	assert.False(t, traceExporter.ExporterShutdown)
	assert.False(t, metricExporter.ExporterShutdown)
	exporters.ShutdownAll()

	assert.True(t, traceExporter.ExporterShutdown)
	assert.True(t, metricExporter.ExporterShutdown)
}
