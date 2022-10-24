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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/service/telemetry"
)

func TestLoadEmpty(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	_, err = Unmarshal(confmap.New(), factories)
	assert.NoError(t, err)
}

func TestLoadEmptyAllSections(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	conf := confmap.NewFromStringMap(map[string]interface{}{
		"receivers":  nil,
		"processors": nil,
		"exporters":  nil,
		"extensions": nil,
		"service":    nil,
	})
	cfg, err := Unmarshal(conf, factories)
	assert.NoError(t, err)

	zapProdCfg := zap.NewProductionConfig()
	assert.Equal(t, telemetry.LogsConfig{
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

func TestUnmarshal(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	// Unmarshal the config
	cfg, err := loadConfigFile(t, filepath.Join("testdata", "valid-config.yaml"), factories)
	require.NoError(t, err, "Unable to load config")

	assert.Equal(t,
		&config.Config{
			Receivers: map[config.ComponentID]config.Receiver{
				config.NewComponentID("nop"): factories.Receivers["nop"].CreateDefaultConfig(),
			},
			Exporters: map[config.ComponentID]config.Exporter{
				config.NewComponentID("nop"): factories.Exporters["nop"].CreateDefaultConfig(),
			},
			Processors: map[config.ComponentID]config.Processor{
				config.NewComponentID("nop"): factories.Processors["nop"].CreateDefaultConfig(),
			},
			Extensions: map[config.ComponentID]config.Extension{
				config.NewComponentID("nop"): factories.Extensions["nop"].CreateDefaultConfig(),
			},
			Service: config.Service{
				Extensions: []config.ComponentID{config.NewComponentID("nop")},
				Pipelines: map[config.ComponentID]*config.Pipeline{
					config.NewComponentID("traces"): {
						Receivers:  []config.ComponentID{config.NewComponentID("nop")},
						Processors: []config.ComponentID{config.NewComponentID("nop")},
						Exporters:  []config.ComponentID{config.NewComponentID("nop")},
					},
				},
				Telemetry: telemetry.Config{
					Logs: telemetry.LogsConfig{
						Level:             zapcore.DebugLevel,
						Development:       true,
						Encoding:          "console",
						DisableCaller:     true,
						DisableStacktrace: true,
						OutputPaths:       []string{"stderr", "./output-logs"},
						ErrorOutputPaths:  []string{"stderr", "./error-output-logs"},
						InitialFields:     map[string]interface{}{"field_key": "filed_value"},
					},
					Metrics: telemetry.MetricsConfig{
						Level:   configtelemetry.LevelNormal,
						Address: ":8081",
					},
				},
			},
		}, cfg)
}

func TestUnmarshalError(t *testing.T) {
	var testCases = []struct {
		name            string          // test case name (also file name containing config yaml)
		expected        configErrorCode // expected error (if nil any error is acceptable)
		expectedMessage string          // string that the error must contain
	}{
		{name: "duplicate-pipeline", expected: errUnmarshalTopLevelStructure, expectedMessage: "duplicate name"},
		{name: "invalid-logs-level", expected: errUnmarshalTopLevelStructure},
		{name: "invalid-metrics-level", expected: errUnmarshalTopLevelStructure},
		{name: "invalid-pipeline-name-after-slash", expected: errUnmarshalTopLevelStructure},
		{name: "invalid-pipeline-section", expected: errUnmarshalTopLevelStructure, expectedMessage: "pipelines"},
		{name: "invalid-pipeline-sub-config", expected: errUnmarshalTopLevelStructure},
		{name: "invalid-pipeline-type", expected: errUnmarshalTopLevelStructure},
		{name: "invalid-sequence-value", expected: errUnmarshalTopLevelStructure, expectedMessage: "pipelines"},
		{name: "invalid-service-extensions-section", expected: errUnmarshalTopLevelStructure},
		{name: "invalid-service-section", expected: errUnmarshalTopLevelStructure},
		{name: "invalid-top-level-section", expected: errUnmarshalTopLevelStructure, expectedMessage: "top level"},
	}

	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			_, err = loadConfigFile(t, filepath.Join("testdata", test.name+".yaml"), factories)
			require.Error(t, err)
			if test.expected != 0 {
				var cfgErr configError
				require.ErrorAs(t, err, &cfgErr)
				assert.Equal(t, test.expected, cfgErr.code, err)
				if test.expectedMessage != "" {
					assert.Contains(t, cfgErr.Error(), test.expectedMessage)
				}
				assert.NotEmpty(t, cfgErr.Error(), "returned config error %v with empty error message", cfgErr.code)
			}
		})
	}
}

func loadConfigFile(t *testing.T, fileName string, factories component.Factories) (*config.Config, error) {
	cm, err := confmaptest.LoadConf(fileName)
	require.NoError(t, err)

	// Unmarshal the config from the confmap.Conf using the given factories.
	return Unmarshal(cm, factories)
}
