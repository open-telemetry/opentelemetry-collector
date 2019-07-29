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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/config"
	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
	"github.com/open-telemetry/opentelemetry-service/exporter/opencensusexporter"
)

func TestExportersBuilder_Build(t *testing.T) {
	_, _, exporterFactories, err := config.ExampleComponents()
	assert.Nil(t, err)

	oceFactory := &opencensusexporter.Factory{}
	exporterFactories[oceFactory.Type()] = oceFactory
	cfg := &configmodels.Config{
		Exporters: map[string]configmodels.Exporter{
			"opencensus": &opencensusexporter.Config{
				ExporterSettings: configmodels.ExporterSettings{
					NameVal: "opencensus",
					TypeVal: "opencensus",
				},
				Endpoint: "0.0.0.0:12345",
			},
		},

		Pipelines: map[string]*configmodels.Pipeline{
			"trace": {
				Name:      "trace",
				InputType: configmodels.TracesDataType,
				Exporters: []string{"opencensus"},
			},
		},
	}

	exporters, err := NewExportersBuilder(zap.NewNop(), cfg, exporterFactories).Build()

	assert.NoError(t, err)
	require.NotNil(t, exporters)

	e1 := exporters[cfg.Exporters["opencensus"]]

	// Ensure exporter has its fields correctly populated.
	require.NotNil(t, e1)
	assert.NotNil(t, e1.tc)
	assert.Nil(t, e1.mc)
	assert.NotNil(t, e1.stop)

	// Ensure it can be stopped.
	err = e1.Stop()
	if err != nil {
		// Since the endpoint of opencensus exporter doesn't actually exist, e1 may
		// already stop because it cannot connect.
		assert.Equal(t, err.Error(), "rpc error: code = Canceled desc = grpc: the client connection is closing")
	}

	// Now change only pipeline data type to "metrics" and make sure exporter builder
	// now fails (because opencensus exporter does not currently support metrics).
	cfg.Pipelines["trace"].InputType = configmodels.MetricsDataType
	_, err = NewExportersBuilder(zap.NewNop(), cfg, exporterFactories).Build()
	assert.NotNil(t, err)

	// Remove the pipeline so that the exporter is not attached to any pipeline.
	// This should result in creating an exporter that has none of consumption
	// functions set.
	delete(cfg.Pipelines, "trace")
	exporters, err = NewExportersBuilder(zap.NewNop(), cfg, exporterFactories).Build()
	assert.NotNil(t, exporters)
	assert.Nil(t, err)

	e1 = exporters[cfg.Exporters["opencensus"]]

	// Ensure exporter has its fields correctly populated.
	require.NotNil(t, e1)
	assert.Nil(t, e1.tc)
	assert.Nil(t, e1.mc)
	assert.Nil(t, e1.stop)

	// TODO: once we have an exporter that supports metrics data type test it too.
}

func TestExportersBuilder_StopAll(t *testing.T) {
	exporters := make(Exporters)
	expCfg := &configmodels.ExporterSettings{}
	stopCalled := false
	exporters[expCfg] = &builtExporter{
		stop: func() error {
			stopCalled = true
			return nil
		},
	}

	exporters.StopAll()

	assert.Equal(t, stopCalled, true)
}

func Test_combineStopFunc(t *testing.T) {
	f := combineStopFunc(nil, nil)
	assert.Nil(t, f)

	err1 := errors.New("err1")
	f1called := false
	f1 := func() error {
		f1called = true
		return err1
	}

	err2 := errors.New("err2")
	f2called := false
	f2 := func() error {
		f2called = true
		return err2
	}

	f = combineStopFunc(f1, nil)
	assert.Equal(t, f(), err1)
	assert.Equal(t, f1called, true)
	assert.Equal(t, f2called, false)

	f1called = false
	f2called = false
	f = combineStopFunc(nil, f2)
	assert.Equal(t, f(), err2)
	assert.Equal(t, f1called, false)
	assert.Equal(t, f2called, true)

	// When 2 functions are combined ensure both functions are called.
	f1called = false
	f2called = false
	f = combineStopFunc(f1, f2)
	assert.NotNil(t, f())
	assert.Equal(t, f1called, true)
	assert.Equal(t, f2called, true)

	// If first function does not return error the result should be equal to what
	// second function returns.
	err1 = nil
	f = combineStopFunc(f1, f2)
	assert.Equal(t, f(), err2)

	// If second function does not return error the result should be equal to what
	// first function returns.
	err1 = errors.New("err1")
	err2 = nil
	f = combineStopFunc(f1, f2)
	assert.Equal(t, f(), err1)

	// If none of the functions returns an error then combined should also not return
	// an error.
	err1 = nil
	err2 = nil
	f = combineStopFunc(f1, f2)
	assert.Equal(t, f(), nil)
}
