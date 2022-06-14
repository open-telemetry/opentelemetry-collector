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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/internal/testcomponents"
	"go.opentelemetry.io/collector/service/servicetest"
)

func TestBuildExporters(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	cfg := &config.Config{
		Exporters: map[config.ComponentID]config.Exporter{
			config.NewComponentID("nop"): factories.Exporters["nop"].CreateDefaultConfig(),
		},

		Service: config.Service{
			Pipelines: map[config.ComponentID]*config.Pipeline{
				config.NewComponentID("traces"): {
					Exporters: []config.ComponentID{config.NewComponentID("nop")},
				},
				config.NewComponentID("metrics"): {
					Exporters: []config.ComponentID{config.NewComponentID("nop")},
				},
				config.NewComponentID("logs"): {
					Exporters: []config.ComponentID{config.NewComponentID("nop")},
				},
			},
		},
	}

	exporters, err := BuildExporters(context.Background(), componenttest.NewNopTelemetrySettings(), component.NewDefaultBuildInfo(), cfg, factories.Exporters)
	assert.NoError(t, err)

	exps := exporters.ToMapByDataType()
	require.Len(t, exps, 3)
	assert.NotNil(t, exps[config.TracesDataType][config.NewComponentID("nop")])
	assert.NotNil(t, exps[config.MetricsDataType][config.NewComponentID("nop")])
	assert.NotNil(t, exps[config.LogsDataType][config.NewComponentID("nop")])

	// Ensure it can be started.
	assert.NoError(t, exporters.StartAll(context.Background(), componenttest.NewNopHost()))

	// Ensure it can be stopped.
	assert.NoError(t, exporters.ShutdownAll(context.Background()))

	// Remove the pipeline so that the exporter is not attached to any pipeline.
	// This should result in creating an exporter that has none of consumption
	// functions set.
	cfg.Service.Pipelines = map[config.ComponentID]*config.Pipeline{}
	exporters, err = BuildExporters(context.Background(), componenttest.NewNopTelemetrySettings(), component.NewDefaultBuildInfo(), cfg, factories.Exporters)
	assert.NoError(t, err)

	exps = exporters.ToMapByDataType()
	require.Len(t, exps, 3)
	assert.Len(t, exps[config.TracesDataType], 0)
	assert.Len(t, exps[config.MetricsDataType], 0)
	assert.Len(t, exps[config.LogsDataType], 0)
}

func TestBuildExportersStartStopAll(t *testing.T) {
	traceExporter := &testcomponents.ExampleExporter{}
	metricExporter := &testcomponents.ExampleExporter{}
	logsExporter := &testcomponents.ExampleExporter{}
	exps := &BuiltExporters{
		settings: componenttest.NewNopTelemetrySettings(),
		exporters: map[config.DataType]map[config.ComponentID]component.Exporter{
			config.TracesDataType: {
				config.NewComponentID("example"): traceExporter,
			},
			config.MetricsDataType: {
				config.NewComponentID("example"): metricExporter,
			},
			config.LogsDataType: {
				config.NewComponentID("example"): logsExporter,
			},
		},
	}
	assert.False(t, traceExporter.Started)
	assert.False(t, metricExporter.Started)
	assert.False(t, logsExporter.Started)

	assert.NoError(t, exps.StartAll(context.Background(), componenttest.NewNopHost()))
	assert.True(t, traceExporter.Started)
	assert.True(t, metricExporter.Started)
	assert.True(t, logsExporter.Started)

	assert.NoError(t, exps.ShutdownAll(context.Background()))
	assert.True(t, traceExporter.Stopped)
	assert.True(t, metricExporter.Stopped)
	assert.True(t, logsExporter.Stopped)
}

func TestBuildExportersNotSupportedDataType(t *testing.T) {
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

			cfg, err := servicetest.LoadConfigAndValidate(filepath.Join("testdata", test.configFile), factories)
			require.Nil(t, err)

			_, err = BuildExporters(context.Background(), componenttest.NewNopTelemetrySettings(), component.NewDefaultBuildInfo(), cfg, factories.Exporters)
			assert.Error(t, err)
		})
	}
}
