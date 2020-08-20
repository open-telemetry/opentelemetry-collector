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

package configtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/confignet"
)

func TestLoadConfigFile(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	cfg, err := LoadConfigFile(t, "testdata/config.yaml", factories)
	require.NoError(t, err, "Unable to load config")

	// Verify extensions.
	assert.Equal(t, 3, len(cfg.Extensions))
	assert.Equal(t, "some string", cfg.Extensions["exampleextension/1"].(*componenttest.ExampleExtensionCfg).ExtraSetting)

	// Verify service.
	assert.Equal(t, 2, len(cfg.Service.Extensions))
	assert.Equal(t, "exampleextension/0", cfg.Service.Extensions[0])
	assert.Equal(t, "exampleextension/1", cfg.Service.Extensions[1])

	// Verify receivers
	assert.Equal(t, 2, len(cfg.Receivers), "Incorrect receivers count")

	assert.Equal(t,
		&componenttest.ExampleReceiver{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal: "examplereceiver",
				NameVal: "examplereceiver",
			},
			TCPAddr: confignet.TCPAddr{
				Endpoint: "localhost:1000",
			},
			ExtraSetting: "some string",
		},
		cfg.Receivers["examplereceiver"],
		"Did not load receiver config correctly")

	assert.Equal(t,
		&componenttest.ExampleReceiver{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal: "examplereceiver",
				NameVal: "examplereceiver/myreceiver",
			},
			TCPAddr: confignet.TCPAddr{
				Endpoint: "localhost:12345",
			},
			ExtraSetting: "some string",
		},
		cfg.Receivers["examplereceiver/myreceiver"],
		"Did not load receiver config correctly")

	// Verify exporters
	assert.Equal(t, 2, len(cfg.Exporters), "Incorrect exporters count")

	assert.Equal(t,
		&componenttest.ExampleExporter{
			ExporterSettings: configmodels.ExporterSettings{
				NameVal: "exampleexporter",
				TypeVal: "exampleexporter",
			},
			ExtraSetting: "some export string",
		},
		cfg.Exporters["exampleexporter"],
		"Did not load exporter config correctly")

	assert.Equal(t,
		&componenttest.ExampleExporter{
			ExporterSettings: configmodels.ExporterSettings{
				NameVal: "exampleexporter/myexporter",
				TypeVal: "exampleexporter",
			},
			ExtraSetting: "some export string 2",
		},
		cfg.Exporters["exampleexporter/myexporter"],
		"Did not load exporter config correctly")

	// Verify Processors
	assert.Equal(t, 1, len(cfg.Processors), "Incorrect processors count")

	assert.Equal(t,
		&componenttest.ExampleProcessorCfg{
			ProcessorSettings: configmodels.ProcessorSettings{
				TypeVal: "exampleprocessor",
				NameVal: "exampleprocessor",
			},
			ExtraSetting: "some export string",
		},
		cfg.Processors["exampleprocessor"],
		"Did not load processor config correctly")

	// Verify Pipelines
	assert.Equal(t, 1, len(cfg.Service.Pipelines), "Incorrect pipelines count")

	assert.Equal(t,
		&configmodels.Pipeline{
			Name:       "traces",
			InputType:  configmodels.TracesDataType,
			Receivers:  []string{"examplereceiver"},
			Processors: []string{"exampleprocessor"},
			Exporters:  []string{"exampleexporter"},
		},
		cfg.Service.Pipelines["traces"],
		"Did not load pipeline config correctly")
}
