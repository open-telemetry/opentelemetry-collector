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

package configunmarshaler

import (
	"context"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmapprovider"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/testcomponents"
)

func TestDecodeConfig(t *testing.T) {
	factories, err := testcomponents.ExampleComponents()
	assert.NoError(t, err)

	// Unmarshal the config
	cfg, err := loadConfigFile(t, path.Join(".", "testdata", "valid-config.yaml"), factories)
	require.NoError(t, err, "Unable to load config")

	// Verify extensions.
	assert.Equal(t, 3, len(cfg.Extensions))
	assert.Equal(t, "some string", cfg.Extensions[config.NewComponentIDWithName("exampleextension", "1")].(*testcomponents.ExampleExtensionCfg).ExtraSetting)

	// Verify receivers
	assert.Equal(t, 2, len(cfg.Receivers), "Incorrect receivers count")

	assert.Equal(t,
		&testcomponents.ExampleReceiver{
			ReceiverSettings: config.NewReceiverSettings(config.NewComponentID("examplereceiver")),
			TCPAddr: confignet.TCPAddr{
				Endpoint: "localhost:1000",
			},
			ExtraSetting: "some string",
		},
		cfg.Receivers[config.NewComponentID("examplereceiver")],
		"Did not load receiver config correctly")

	assert.Equal(t,
		&testcomponents.ExampleReceiver{
			ReceiverSettings: config.NewReceiverSettings(config.NewComponentIDWithName("examplereceiver", "myreceiver")),
			TCPAddr: confignet.TCPAddr{
				Endpoint: "localhost:12345",
			},
			ExtraSetting: "some string",
		},
		cfg.Receivers[config.NewComponentIDWithName("examplereceiver", "myreceiver")],
		"Did not load receiver config correctly")

	// Verify exporters
	assert.Equal(t, 2, len(cfg.Exporters), "Incorrect exporters count")

	assert.Equal(t,
		&testcomponents.ExampleExporter{
			ExporterSettings: config.NewExporterSettings(config.NewComponentID("exampleexporter")),
			ExtraSetting:     "some export string",
		},
		cfg.Exporters[config.NewComponentID("exampleexporter")],
		"Did not load exporter config correctly")

	assert.Equal(t,
		&testcomponents.ExampleExporter{
			ExporterSettings: config.NewExporterSettings(config.NewComponentIDWithName("exampleexporter", "myexporter")),
			ExtraSetting:     "some export string 2",
		},
		cfg.Exporters[config.NewComponentIDWithName("exampleexporter", "myexporter")],
		"Did not load exporter config correctly")

	// Verify Processors
	assert.Equal(t, 1, len(cfg.Processors), "Incorrect processors count")

	assert.Equal(t,
		&testcomponents.ExampleProcessorCfg{
			ProcessorSettings: config.NewProcessorSettings(config.NewComponentID("exampleprocessor")),
			ExtraSetting:      "some export string",
		},
		cfg.Processors[config.NewComponentID("exampleprocessor")],
		"Did not load processor config correctly")

	// Verify Service Telemetry
	assert.Equal(t,
		config.ServiceTelemetry{
			Logs: config.ServiceTelemetryLogs{
				Level:             zapcore.DebugLevel,
				Development:       true,
				Encoding:          "console",
				DisableCaller:     true,
				DisableStacktrace: true,
				OutputPaths:       []string{"stderr", "./output-logs"},
				ErrorOutputPaths:  []string{"stderr", "./error-output-logs"},
				InitialFields:     map[string]interface{}{"field_key": "filed_value"},
			},
			Metrics: config.ServiceTelemetryMetrics{
				Level:   configtelemetry.LevelNormal,
				Address: ":8081",
			},
		}, cfg.Service.Telemetry)

	// Verify Service Extensions
	assert.Equal(t, 2, len(cfg.Service.Extensions))
	assert.Equal(t, config.NewComponentIDWithName("exampleextension", "0"), cfg.Service.Extensions[0])
	assert.Equal(t, config.NewComponentIDWithName("exampleextension", "1"), cfg.Service.Extensions[1])

	// Verify Service Pipelines
	assert.Equal(t, 1, len(cfg.Service.Pipelines), "Incorrect pipelines count")

	assert.Equal(t,
		&config.Pipeline{
			Receivers:  []config.ComponentID{config.NewComponentID("examplereceiver")},
			Processors: []config.ComponentID{config.NewComponentID("exampleprocessor")},
			Exporters:  []config.ComponentID{config.NewComponentID("exampleexporter")},
		},
		cfg.Service.Pipelines[config.NewComponentID("traces")],
		"Did not load pipeline config correctly")
}

func TestDecodeConfig_Invalid(t *testing.T) {

	var testCases = []struct {
		name            string          // test case name (also file name containing config yaml)
		expected        configErrorCode // expected error (if nil any error is acceptable)
		expectedMessage string          // string that the error must contain
	}{
		{name: "invalid-extension-type", expected: errUnmarshalTopLevelStructure},
		{name: "invalid-receiver-type", expected: errUnmarshalTopLevelStructure},
		{name: "invalid-processor-type", expected: errUnmarshalTopLevelStructure},
		{name: "invalid-exporter-type", expected: errUnmarshalTopLevelStructure},
		{name: "invalid-pipeline-type", expected: errUnmarshalService},

		{name: "invalid-extension-name-after-slash", expected: errUnmarshalTopLevelStructure},
		{name: "invalid-receiver-name-after-slash", expected: errUnmarshalTopLevelStructure},
		{name: "invalid-processor-name-after-slash", expected: errUnmarshalTopLevelStructure},
		{name: "invalid-exporter-name-after-slash", expected: errUnmarshalTopLevelStructure},
		{name: "invalid-pipeline-name-after-slash", expected: errUnmarshalService},

		{name: "unknown-extension-type", expected: errUnmarshalExtension, expectedMessage: "extensions"},
		{name: "unknown-receiver-type", expected: errUnmarshalReceiver, expectedMessage: "receivers"},
		{name: "unknown-processor-type", expected: errUnmarshalProcessor, expectedMessage: "processors"},
		{name: "unknown-exporter-type", expected: errUnmarshalExporter, expectedMessage: "exporters"},
		{name: "unknown-pipeline-type", expected: errUnmarshalService, expectedMessage: "pipelines"},

		{name: "duplicate-extension", expected: errUnmarshalTopLevelStructure, expectedMessage: "duplicate name"},
		{name: "duplicate-receiver", expected: errUnmarshalTopLevelStructure, expectedMessage: "duplicate name"},
		{name: "duplicate-processor", expected: errUnmarshalTopLevelStructure, expectedMessage: "duplicate name"},
		{name: "duplicate-exporter", expected: errUnmarshalTopLevelStructure, expectedMessage: "duplicate name"},
		{name: "duplicate-pipeline", expected: errUnmarshalService, expectedMessage: "duplicate name"},

		{name: "invalid-top-level-section", expected: errUnmarshalTopLevelStructure, expectedMessage: "top level"},
		{name: "invalid-extension-section", expected: errUnmarshalExtension, expectedMessage: "extensions"},
		{name: "invalid-receiver-section", expected: errUnmarshalReceiver, expectedMessage: "receivers"},
		{name: "invalid-processor-section", expected: errUnmarshalProcessor, expectedMessage: "processors"},
		{name: "invalid-exporter-section", expected: errUnmarshalExporter, expectedMessage: "exporters"},
		{name: "invalid-service-section", expected: errUnmarshalService},
		{name: "invalid-service-extensions-section", expected: errUnmarshalService},
		{name: "invalid-pipeline-section", expected: errUnmarshalService, expectedMessage: "pipelines"},
		{name: "invalid-sequence-value", expected: errUnmarshalService, expectedMessage: "pipelines"},

		{name: "invalid-extension-sub-config", expected: errUnmarshalTopLevelStructure},
		{name: "invalid-receiver-sub-config", expected: errUnmarshalTopLevelStructure},
		{name: "invalid-processor-sub-config", expected: errUnmarshalTopLevelStructure},
		{name: "invalid-exporter-sub-config", expected: errUnmarshalTopLevelStructure},
		{name: "invalid-pipeline-sub-config", expected: errUnmarshalService},

		{name: "invalid-logs-level", expected: errUnmarshalService},
		{name: "invalid-metrics-level", expected: errUnmarshalService},
	}

	factories, err := testcomponents.ExampleComponents()
	assert.NoError(t, err)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			_, err := loadConfigFile(t, path.Join(".", "testdata", test.name+".yaml"), factories)
			require.Error(t, err)
			if test.expected != 0 {
				cfgErr, ok := err.(*configError)
				if !ok {
					t.Errorf("expected config error code %v but got a different error '%v'", test.expected, err)
				} else {
					assert.Equal(t, test.expected, cfgErr.code, err)
					if test.expectedMessage != "" {
						assert.Contains(t, cfgErr.Error(), test.expectedMessage)
					}
					assert.NotEmpty(t, cfgErr.Error(), "returned config error %v with empty error message", cfgErr.code)
				}
			}
		})
	}
}

func TestLoadEmpty(t *testing.T) {
	factories, err := testcomponents.ExampleComponents()
	assert.NoError(t, err)

	_, err = loadConfigFile(t, path.Join(".", "testdata", "empty-config.yaml"), factories)
	assert.NoError(t, err)
}

func TestLoadEmptyAllSections(t *testing.T) {
	factories, err := testcomponents.ExampleComponents()
	assert.NoError(t, err)

	_, err = loadConfigFile(t, path.Join(".", "testdata", "empty-all-sections.yaml"), factories)
	assert.NoError(t, err)
}

func loadConfigFile(t *testing.T, fileName string, factories component.Factories) (*config.Config, error) {
	v, err := configmapprovider.NewFile(fileName).Retrieve(context.Background(), nil)
	require.NoError(t, err)
	cm, err := v.Get(context.Background())
	require.NoError(t, err)

	// Unmarshal the config from the config.Map using the given factories.
	return NewDefault().Unmarshal(cm, factories)
}

func TestDefaultLoggerConfig(t *testing.T) {
	factories, err := testcomponents.ExampleComponents()
	assert.NoError(t, err)

	cfg, err := loadConfigFile(t, path.Join(".", "testdata", "empty-all-sections.yaml"), factories)
	assert.NoError(t, err)

	zapProdCfg := zap.NewProductionConfig()
	assert.Equal(t,
		config.ServiceTelemetryLogs{
			Level:             zapProdCfg.Level.Level(),
			Development:       zapProdCfg.Development,
			Encoding:          "console",
			DisableCaller:     zapProdCfg.DisableCaller,
			DisableStacktrace: zapProdCfg.DisableStacktrace,
			OutputPaths:       zapProdCfg.OutputPaths,
			ErrorOutputPaths:  zapProdCfg.ErrorOutputPaths,
			InitialFields:     zapProdCfg.InitialFields,
		}, cfg.Service.Telemetry.Logs)
}
