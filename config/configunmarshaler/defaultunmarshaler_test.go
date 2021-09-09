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
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configparser"
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
	assert.Equal(t, "some string", cfg.Extensions[config.NewIDWithName("exampleextension", "1")].(*testcomponents.ExampleExtensionCfg).ExtraSetting)

	// Verify service.
	assert.Equal(t, 2, len(cfg.Service.Extensions))
	assert.Equal(t, config.NewIDWithName("exampleextension", "0"), cfg.Service.Extensions[0])
	assert.Equal(t, config.NewIDWithName("exampleextension", "1"), cfg.Service.Extensions[1])

	// Verify receivers
	assert.Equal(t, 2, len(cfg.Receivers), "Incorrect receivers count")

	assert.Equal(t,
		&testcomponents.ExampleReceiver{
			ReceiverSettings: config.NewReceiverSettings(config.NewID("examplereceiver")),
			TCPAddr: confignet.TCPAddr{
				Endpoint: "localhost:1000",
			},
			ExtraSetting: "some string",
		},
		cfg.Receivers[config.NewID("examplereceiver")],
		"Did not load receiver config correctly")

	assert.Equal(t,
		&testcomponents.ExampleReceiver{
			ReceiverSettings: config.NewReceiverSettings(config.NewIDWithName("examplereceiver", "myreceiver")),
			TCPAddr: confignet.TCPAddr{
				Endpoint: "localhost:12345",
			},
			ExtraSetting: "some string",
		},
		cfg.Receivers[config.NewIDWithName("examplereceiver", "myreceiver")],
		"Did not load receiver config correctly")

	// Verify exporters
	assert.Equal(t, 2, len(cfg.Exporters), "Incorrect exporters count")

	assert.Equal(t,
		&testcomponents.ExampleExporter{
			ExporterSettings: config.NewExporterSettings(config.NewID("exampleexporter")),
			ExtraSetting:     "some export string",
		},
		cfg.Exporters[config.NewID("exampleexporter")],
		"Did not load exporter config correctly")

	assert.Equal(t,
		&testcomponents.ExampleExporter{
			ExporterSettings: config.NewExporterSettings(config.NewIDWithName("exampleexporter", "myexporter")),
			ExtraSetting:     "some export string 2",
		},
		cfg.Exporters[config.NewIDWithName("exampleexporter", "myexporter")],
		"Did not load exporter config correctly")

	// Verify Processors
	assert.Equal(t, 1, len(cfg.Processors), "Incorrect processors count")

	assert.Equal(t,
		&testcomponents.ExampleProcessorCfg{
			ProcessorSettings: config.NewProcessorSettings(config.NewID("exampleprocessor")),
			ExtraSetting:      "some export string",
		},
		cfg.Processors[config.NewID("exampleprocessor")],
		"Did not load processor config correctly")

	// Verify Pipelines
	assert.Equal(t, 1, len(cfg.Service.Pipelines), "Incorrect pipelines count")

	assert.Equal(t,
		&config.Pipeline{
			Name:       "traces",
			InputType:  config.TracesDataType,
			Receivers:  []config.ComponentID{config.NewID("examplereceiver")},
			Processors: []config.ComponentID{config.NewID("exampleprocessor")},
			Exporters:  []config.ComponentID{config.NewID("exampleexporter")},
		},
		cfg.Service.Pipelines["traces"],
		"Did not load pipeline config correctly")
}

func TestSimpleConfig(t *testing.T) {
	var testCases = []struct {
		name string // test case name (also file name containing config yaml)
	}{
		{name: "simple-config-with-no-env"},
		{name: "simple-config-with-partial-env"},
		{name: "simple-config-with-all-env"},
	}

	const extensionExtra = "some extension string"
	const extensionExtraMapValue = "some extension map value"
	const extensionExtraListElement = "some extension list value"
	assert.NoError(t, os.Setenv("EXTENSIONS_EXAMPLEEXTENSION_EXTRA", extensionExtra))
	assert.NoError(t, os.Setenv("EXTENSIONS_EXAMPLEEXTENSION_EXTRA_MAP_EXT_VALUE_1", extensionExtraMapValue+"_1"))
	assert.NoError(t, os.Setenv("EXTENSIONS_EXAMPLEEXTENSION_EXTRA_MAP_EXT_VALUE_2", extensionExtraMapValue+"_2"))
	assert.NoError(t, os.Setenv("EXTENSIONS_EXAMPLEEXTENSION_EXTRA_LIST_VALUE_1", extensionExtraListElement+"_1"))
	assert.NoError(t, os.Setenv("EXTENSIONS_EXAMPLEEXTENSION_EXTRA_LIST_VALUE_2", extensionExtraListElement+"_2"))

	const receiverExtra = "some receiver string"
	const receiverExtraMapValue = "some receiver map value"
	const receiverExtraListElement = "some receiver list value"
	assert.NoError(t, os.Setenv("RECEIVERS_EXAMPLERECEIVER_EXTRA", receiverExtra))
	assert.NoError(t, os.Setenv("RECEIVERS_EXAMPLERECEIVER_EXTRA_MAP_RECV_VALUE_1", receiverExtraMapValue+"_1"))
	assert.NoError(t, os.Setenv("RECEIVERS_EXAMPLERECEIVER_EXTRA_MAP_RECV_VALUE_2", receiverExtraMapValue+"_2"))
	assert.NoError(t, os.Setenv("RECEIVERS_EXAMPLERECEIVER_EXTRA_LIST_VALUE_1", receiverExtraListElement+"_1"))
	assert.NoError(t, os.Setenv("RECEIVERS_EXAMPLERECEIVER_EXTRA_LIST_VALUE_2", receiverExtraListElement+"_2"))

	const processorExtra = "some processor string"
	const processorExtraMapValue = "some processor map value"
	const processorExtraListElement = "some processor list value"
	assert.NoError(t, os.Setenv("PROCESSORS_EXAMPLEPROCESSOR_EXTRA", processorExtra))
	assert.NoError(t, os.Setenv("PROCESSORS_EXAMPLEPROCESSOR_EXTRA_MAP_PROC_VALUE_1", processorExtraMapValue+"_1"))
	assert.NoError(t, os.Setenv("PROCESSORS_EXAMPLEPROCESSOR_EXTRA_MAP_PROC_VALUE_2", processorExtraMapValue+"_2"))
	assert.NoError(t, os.Setenv("PROCESSORS_EXAMPLEPROCESSOR_EXTRA_LIST_VALUE_1", processorExtraListElement+"_1"))
	assert.NoError(t, os.Setenv("PROCESSORS_EXAMPLEPROCESSOR_EXTRA_LIST_VALUE_2", processorExtraListElement+"_2"))

	const exporterExtra = "some exporter string"
	const exporterExtraMapValue = "some exporter map value"
	const exporterExtraListElement = "some exporter list value"
	assert.NoError(t, os.Setenv("EXPORTERS_EXAMPLEEXPORTER_EXTRA_INT", "65"))
	assert.NoError(t, os.Setenv("EXPORTERS_EXAMPLEEXPORTER_EXTRA", exporterExtra))
	assert.NoError(t, os.Setenv("EXPORTERS_EXAMPLEEXPORTER_EXTRA_MAP_EXP_VALUE_1", exporterExtraMapValue+"_1"))
	assert.NoError(t, os.Setenv("EXPORTERS_EXAMPLEEXPORTER_EXTRA_MAP_EXP_VALUE_2", exporterExtraMapValue+"_2"))
	assert.NoError(t, os.Setenv("EXPORTERS_EXAMPLEEXPORTER_EXTRA_LIST_VALUE_1", exporterExtraListElement+"_1"))
	assert.NoError(t, os.Setenv("EXPORTERS_EXAMPLEEXPORTER_EXTRA_LIST_VALUE_2", exporterExtraListElement+"_2"))

	defer func() {
		assert.NoError(t, os.Unsetenv("EXTENSIONS_EXAMPLEEXTENSION_EXTRA"))
		assert.NoError(t, os.Unsetenv("EXTENSIONS_EXAMPLEEXTENSION_EXTRA_MAP_EXT_VALUE"))
		assert.NoError(t, os.Unsetenv("EXTENSIONS_EXAMPLEEXTENSION_EXTRA_LIST_VALUE_1"))

		assert.NoError(t, os.Unsetenv("RECEIVERS_EXAMPLERECEIVER_EXTRA"))
		assert.NoError(t, os.Unsetenv("RECEIVERS_EXAMPLERECEIVER_EXTRA_MAP_RECV_VALUE"))
		assert.NoError(t, os.Unsetenv("RECEIVERS_EXAMPLERECEIVER_EXTRA_LIST_VALUE_1"))

		assert.NoError(t, os.Unsetenv("PROCESSORS_EXAMPLEPROCESSOR_EXTRA"))
		assert.NoError(t, os.Unsetenv("PROCESSORS_EXAMPLEPROCESSOR_EXTRA_MAP_PROC_VALUE"))
		assert.NoError(t, os.Unsetenv("PROCESSORS_EXAMPLEPROCESSOR_EXTRA_LIST_VALUE_1"))

		assert.NoError(t, os.Unsetenv("EXPORTERS_EXAMPLEEXPORTER_EXTRA_INT"))
		assert.NoError(t, os.Unsetenv("EXPORTERS_EXAMPLEEXPORTER_EXTRA"))
		assert.NoError(t, os.Unsetenv("EXPORTERS_EXAMPLEEXPORTER_EXTRA_MAP_EXP_VALUE"))
		assert.NoError(t, os.Unsetenv("EXPORTERS_EXAMPLEEXPORTER_EXTRA_LIST_VALUE_1"))
	}()

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			factories, err := testcomponents.ExampleComponents()
			assert.NoError(t, err)

			// Unmarshal the config
			cfg, err := loadConfigFile(t, path.Join(".", "testdata", test.name+".yaml"), factories)
			require.NoError(t, err, "Unable to load config")

			// Verify extensions.
			assert.Equalf(t, 1, len(cfg.Extensions), "TEST[%s]", test.name)
			assert.Equalf(t,
				&testcomponents.ExampleExtensionCfg{
					ExtensionSettings: config.NewExtensionSettings(config.NewID("exampleextension")),
					ExtraSetting:      extensionExtra,
					ExtraMapSetting:   map[string]string{"ext-1": extensionExtraMapValue + "_1", "ext-2": extensionExtraMapValue + "_2"},
					ExtraListSetting:  []string{extensionExtraListElement + "_1", extensionExtraListElement + "_2"},
				},
				cfg.Extensions[config.NewID("exampleextension")],
				"TEST[%s] Did not load extension config correctly", test.name)

			// Verify service.
			assert.Equalf(t, 1, len(cfg.Service.Extensions), "TEST[%s]", test.name)
			assert.Equalf(t, config.NewID("exampleextension"), cfg.Service.Extensions[0], "TEST[%s]", test.name)

			// Verify receivers
			assert.Equalf(t, 1, len(cfg.Receivers), "TEST[%s]", test.name)

			assert.Equalf(t,
				&testcomponents.ExampleReceiver{
					ReceiverSettings: config.NewReceiverSettings(config.NewID("examplereceiver")),
					TCPAddr: confignet.TCPAddr{
						Endpoint: "localhost:1234",
					},
					ExtraSetting:     receiverExtra,
					ExtraMapSetting:  map[string]string{"recv.1": receiverExtraMapValue + "_1", "recv.2": receiverExtraMapValue + "_2"},
					ExtraListSetting: []string{receiverExtraListElement + "_1", receiverExtraListElement + "_2"},
				},
				cfg.Receivers[config.NewID("examplereceiver")],
				"TEST[%s] Did not load receiver config correctly", test.name)

			// Verify exporters
			assert.Equalf(t, 1, len(cfg.Exporters), "TEST[%s]", test.name)

			assert.Equalf(t,
				&testcomponents.ExampleExporter{
					ExporterSettings: config.NewExporterSettings(config.NewID("exampleexporter")),
					ExtraInt:         65,
					ExtraSetting:     exporterExtra,
					ExtraMapSetting:  map[string]string{"exp_1": exporterExtraMapValue + "_1", "exp_2": exporterExtraMapValue + "_2"},
					ExtraListSetting: []string{exporterExtraListElement + "_1", exporterExtraListElement + "_2"},
				},
				cfg.Exporters[config.NewID("exampleexporter")],
				"TEST[%s] Did not load exporter config correctly", test.name)

			// Verify Processors
			assert.Equalf(t, 1, len(cfg.Processors), "TEST[%s]", test.name)

			assert.Equalf(t,
				&testcomponents.ExampleProcessorCfg{
					ProcessorSettings: config.NewProcessorSettings(config.NewID("exampleprocessor")),
					ExtraSetting:      processorExtra,
					ExtraMapSetting:   map[string]string{"proc_1": processorExtraMapValue + "_1", "proc_2": processorExtraMapValue + "_2"},
					ExtraListSetting:  []string{processorExtraListElement + "_1", processorExtraListElement + "_2"},
				},
				cfg.Processors[config.NewID("exampleprocessor")],
				"TEST[%s] Did not load processor config correctly", test.name)

			// Verify Pipelines
			assert.Equalf(t, 1, len(cfg.Service.Pipelines), "TEST[%s]", test.name)

			assert.Equalf(t,
				&config.Pipeline{
					Name:       "traces",
					InputType:  config.TracesDataType,
					Receivers:  []config.ComponentID{config.NewID("examplereceiver")},
					Processors: []config.ComponentID{config.NewID("exampleprocessor")},
					Exporters:  []config.ComponentID{config.NewID("exampleexporter")},
				},
				cfg.Service.Pipelines["traces"],
				"TEST[%s] Did not load pipeline config correctly", test.name)
		})
	}
}

func TestEscapedEnvVars(t *testing.T) {
	const receiverExtraMapValue = "some receiver map value"
	assert.NoError(t, os.Setenv("RECEIVERS_EXAMPLERECEIVER_EXTRA_MAP_RECV_VALUE_2", receiverExtraMapValue))
	defer func() {
		assert.NoError(t, os.Unsetenv("RECEIVERS_EXAMPLERECEIVER_EXTRA_MAP_RECV_VALUE_2"))
	}()

	factories, err := testcomponents.ExampleComponents()
	assert.NoError(t, err)

	// Unmarshal the config
	cfg, err := loadConfigFile(t, path.Join(".", "testdata", "simple-config-with-escaped-env.yaml"), factories)
	require.NoError(t, err, "Unable to load config")

	// Verify extensions.
	assert.Equal(t, 1, len(cfg.Extensions))
	assert.Equal(t,
		&testcomponents.ExampleExtensionCfg{
			ExtensionSettings: config.NewExtensionSettings(config.NewID("exampleextension")),
			ExtraSetting:      "${EXTENSIONS_EXAMPLEEXTENSION_EXTRA}",
			ExtraMapSetting:   map[string]string{"ext-1": "${EXTENSIONS_EXAMPLEEXTENSION_EXTRA_MAP_EXT_VALUE_1}", "ext-2": "${EXTENSIONS_EXAMPLEEXTENSION_EXTRA_MAP_EXT_VALUE_2}"},
			ExtraListSetting:  []string{"${EXTENSIONS_EXAMPLEEXTENSION_EXTRA_LIST_VALUE_1}", "${EXTENSIONS_EXAMPLEEXTENSION_EXTRA_LIST_VALUE_2}"},
		},
		cfg.Extensions[config.NewID("exampleextension")],
		"Did not load extension config correctly")

	// Verify service.
	assert.Equal(t, 1, len(cfg.Service.Extensions))
	assert.Equal(t, config.NewID("exampleextension"), cfg.Service.Extensions[0])

	// Verify receivers
	assert.Equal(t, 1, len(cfg.Receivers))

	assert.Equal(t,
		&testcomponents.ExampleReceiver{
			ReceiverSettings: config.NewReceiverSettings(config.NewID("examplereceiver")),
			TCPAddr: confignet.TCPAddr{
				Endpoint: "localhost:1234",
			},
			ExtraSetting: "$RECEIVERS_EXAMPLERECEIVER_EXTRA",
			ExtraMapSetting: map[string]string{
				// $$ -> escaped $
				"recv.1": "$RECEIVERS_EXAMPLERECEIVER_EXTRA_MAP_RECV_VALUE_1",
				// $$$ -> escaped $ + substituted env var
				"recv.2": "$" + receiverExtraMapValue,
				// $$$$ -> two escaped $
				"recv.3": "$$RECEIVERS_EXAMPLERECEIVER_EXTRA_MAP_RECV_VALUE_3",
				// escaped $ in the middle
				"recv.4": "some${RECEIVERS_EXAMPLERECEIVER_EXTRA_MAP_RECV_VALUE_4}text",
				// $$$$ -> two escaped $
				"recv.5": "${ONE}${TWO}",
				// trailing escaped $
				"recv.6": "text$",
				// escaped $ alone
				"recv.7": "$",
			},
			ExtraListSetting: []string{"$RECEIVERS_EXAMPLERECEIVER_EXTRA_LIST_VALUE_1", "$RECEIVERS_EXAMPLERECEIVER_EXTRA_LIST_VALUE_2"},
		},
		cfg.Receivers[config.NewID("examplereceiver")],
		"Did not load receiver config correctly")

	// Verify exporters
	assert.Equal(t, 1, len(cfg.Exporters))

	assert.Equal(t,
		&testcomponents.ExampleExporter{
			ExporterSettings: config.NewExporterSettings(config.NewID("exampleexporter")),
			ExtraSetting:     "${EXPORTERS_EXAMPLEEXPORTER_EXTRA}",
			ExtraMapSetting:  map[string]string{"exp_1": "${EXPORTERS_EXAMPLEEXPORTER_EXTRA_MAP_EXP_VALUE_1}", "exp_2": "${EXPORTERS_EXAMPLEEXPORTER_EXTRA_MAP_EXP_VALUE_2}"},
			ExtraListSetting: []string{"${EXPORTERS_EXAMPLEEXPORTER_EXTRA_LIST_VALUE_1}", "${EXPORTERS_EXAMPLEEXPORTER_EXTRA_LIST_VALUE_2}"},
		},
		cfg.Exporters[config.NewID("exampleexporter")],
		"Did not load exporter config correctly")

	// Verify Processors
	assert.Equal(t, 1, len(cfg.Processors))

	assert.Equal(t,
		&testcomponents.ExampleProcessorCfg{
			ProcessorSettings: config.NewProcessorSettings(config.NewID("exampleprocessor")),
			ExtraSetting:      "$PROCESSORS_EXAMPLEPROCESSOR_EXTRA",
			ExtraMapSetting:   map[string]string{"proc_1": "$PROCESSORS_EXAMPLEPROCESSOR_EXTRA_MAP_PROC_VALUE_1", "proc_2": "$PROCESSORS_EXAMPLEPROCESSOR_EXTRA_MAP_PROC_VALUE_2"},
			ExtraListSetting:  []string{"$PROCESSORS_EXAMPLEPROCESSOR_EXTRA_LIST_VALUE_1", "$PROCESSORS_EXAMPLEPROCESSOR_EXTRA_LIST_VALUE_2"},
		},
		cfg.Processors[config.NewID("exampleprocessor")],
		"Did not load processor config correctly")

	// Verify Pipelines
	assert.Equal(t, 1, len(cfg.Service.Pipelines))

	assert.Equal(t,
		&config.Pipeline{
			Name:       "traces",
			InputType:  config.TracesDataType,
			Receivers:  []config.ComponentID{config.NewID("examplereceiver")},
			Processors: []config.ComponentID{config.NewID("exampleprocessor")},
			Exporters:  []config.ComponentID{config.NewID("exampleexporter")},
		},
		cfg.Service.Pipelines["traces"],
		"Did not load pipeline config correctly")
}

func TestDecodeConfig_Invalid(t *testing.T) {

	var testCases = []struct {
		name            string          // test case name (also file name containing config yaml)
		expected        configErrorCode // expected error (if nil any error is acceptable)
		expectedMessage string          // string that the error must contain
	}{
		{name: "invalid-extension-type", expected: errInvalidTypeAndNameKey},
		{name: "invalid-receiver-type", expected: errInvalidTypeAndNameKey},
		{name: "invalid-exporter-type", expected: errInvalidTypeAndNameKey},
		{name: "invalid-processor-type", expected: errInvalidTypeAndNameKey},
		{name: "invalid-pipeline-type", expected: errInvalidTypeAndNameKey},

		{name: "invalid-extension-name-after-slash", expected: errInvalidTypeAndNameKey},
		{name: "invalid-receiver-name-after-slash", expected: errInvalidTypeAndNameKey},
		{name: "invalid-exporter-name-after-slash", expected: errInvalidTypeAndNameKey},
		{name: "invalid-processor-name-after-slash", expected: errInvalidTypeAndNameKey},
		{name: "invalid-pipeline-name-after-slash", expected: errInvalidTypeAndNameKey},

		{name: "unknown-extension-type", expected: errUnknownType, expectedMessage: "extensions"},
		{name: "unknown-receiver-type", expected: errUnknownType, expectedMessage: "receivers"},
		{name: "unknown-exporter-type", expected: errUnknownType, expectedMessage: "exporters"},
		{name: "unknown-processor-type", expected: errUnknownType, expectedMessage: "processors"},
		{name: "unknown-pipeline-type", expected: errUnknownType, expectedMessage: "pipelines"},

		{name: "duplicate-extension", expected: errDuplicateName, expectedMessage: "extensions"},
		{name: "duplicate-receiver", expected: errDuplicateName, expectedMessage: "receivers"},
		{name: "duplicate-exporter", expected: errDuplicateName, expectedMessage: "exporters"},
		{name: "duplicate-processor", expected: errDuplicateName, expectedMessage: "processors"},
		{name: "duplicate-pipeline", expected: errDuplicateName, expectedMessage: "pipelines"},

		{name: "invalid-top-level-section", expected: errUnmarshalTopLevelStructureError, expectedMessage: "top level"},
		{name: "invalid-extension-section", expected: errUnmarshalTopLevelStructureError, expectedMessage: "extensions"},
		{name: "invalid-receiver-section", expected: errUnmarshalTopLevelStructureError, expectedMessage: "receivers"},
		{name: "invalid-processor-section", expected: errUnmarshalTopLevelStructureError, expectedMessage: "processors"},
		{name: "invalid-exporter-section", expected: errUnmarshalTopLevelStructureError, expectedMessage: "exporters"},
		{name: "invalid-service-section", expected: errUnmarshalTopLevelStructureError, expectedMessage: "service"},
		{name: "invalid-service-extensions-section", expected: errUnmarshalTopLevelStructureError, expectedMessage: "service"},
		{name: "invalid-pipeline-section", expected: errUnmarshalTopLevelStructureError, expectedMessage: "pipelines"},
		{name: "invalid-sequence-value", expected: errUnmarshalTopLevelStructureError, expectedMessage: "pipelines"},

		{name: "invalid-extension-sub-config", expected: errUnmarshalTopLevelStructureError},
		{name: "invalid-exporter-sub-config", expected: errUnmarshalTopLevelStructureError},
		{name: "invalid-processor-sub-config", expected: errUnmarshalTopLevelStructureError},
		{name: "invalid-receiver-sub-config", expected: errUnmarshalTopLevelStructureError},
		{name: "invalid-pipeline-sub-config", expected: errUnmarshalTopLevelStructureError},
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
	v, err := configparser.NewParserFromFile(fileName)
	require.NoError(t, err)

	// Unmarshal the config from the configparser.ConfigMap using the given factories.
	return NewDefault().Unmarshal(v, factories)
}

type nestedConfig struct {
	NestedStringValue string
	NestedIntValue    int
}

type testConfig struct {
	config.ExporterSettings

	NestedConfigPtr   *nestedConfig
	NestedConfigValue nestedConfig
	StringValue       string
	StringPtrValue    *string
	IntValue          int
}

func TestExpandEnvLoadedConfig(t *testing.T) {
	assert.NoError(t, os.Setenv("NESTED_VALUE", "replaced_nested_value"))
	assert.NoError(t, os.Setenv("VALUE", "replaced_value"))
	assert.NoError(t, os.Setenv("PTR_VALUE", "replaced_ptr_value"))

	defer func() {
		assert.NoError(t, os.Unsetenv("NESTED_VALUE"))
		assert.NoError(t, os.Unsetenv("VALUE"))
		assert.NoError(t, os.Unsetenv("PTR_VALUE"))
	}()

	testString := "$PTR_VALUE"

	cfg := &testConfig{
		ExporterSettings: config.NewExporterSettings(config.NewID("test")),
		NestedConfigPtr: &nestedConfig{
			NestedStringValue: "$NESTED_VALUE",
			NestedIntValue:    1,
		},
		NestedConfigValue: nestedConfig{
			NestedStringValue: "$NESTED_VALUE",
			NestedIntValue:    2,
		},
		StringValue:    "$VALUE",
		StringPtrValue: &testString,
		IntValue:       3,
	}

	expandEnvLoadedConfig(cfg)

	replacedTestString := "replaced_ptr_value"

	assert.Equal(t, &testConfig{
		ExporterSettings: config.NewExporterSettings(config.NewID("test")),
		NestedConfigPtr: &nestedConfig{
			NestedStringValue: "replaced_nested_value",
			NestedIntValue:    1,
		},
		NestedConfigValue: nestedConfig{
			NestedStringValue: "replaced_nested_value",
			NestedIntValue:    2,
		},
		StringValue:    "replaced_value",
		StringPtrValue: &replacedTestString,
		IntValue:       3,
	}, cfg)
}

func TestExpandEnvLoadedConfigEscapedEnv(t *testing.T) {
	assert.NoError(t, os.Setenv("NESTED_VALUE", "replaced_nested_value"))
	assert.NoError(t, os.Setenv("ESCAPED_VALUE", "replaced_escaped_value"))
	assert.NoError(t, os.Setenv("ESCAPED_PTR_VALUE", "replaced_escaped_pointer_value"))

	defer func() {
		assert.NoError(t, os.Unsetenv("NESTED_VALUE"))
		assert.NoError(t, os.Unsetenv("ESCAPED_VALUE"))
		assert.NoError(t, os.Unsetenv("ESCAPED_PTR_VALUE"))
	}()

	testString := "$$ESCAPED_PTR_VALUE"

	cfg := &testConfig{
		ExporterSettings: config.NewExporterSettings(config.NewID("test")),
		NestedConfigPtr: &nestedConfig{
			NestedStringValue: "$NESTED_VALUE",
			NestedIntValue:    1,
		},
		NestedConfigValue: nestedConfig{
			NestedStringValue: "$NESTED_VALUE",
			NestedIntValue:    2,
		},
		StringValue:    "$$ESCAPED_VALUE",
		StringPtrValue: &testString,
		IntValue:       3,
	}

	expandEnvLoadedConfig(cfg)

	replacedTestString := "$ESCAPED_PTR_VALUE"

	assert.Equal(t, &testConfig{
		ExporterSettings: config.NewExporterSettings(config.NewID("test")),
		NestedConfigPtr: &nestedConfig{
			NestedStringValue: "replaced_nested_value",
			NestedIntValue:    1,
		},
		NestedConfigValue: nestedConfig{
			NestedStringValue: "replaced_nested_value",
			NestedIntValue:    2,
		},
		StringValue:    "$ESCAPED_VALUE",
		StringPtrValue: &replacedTestString,
		IntValue:       3,
	}, cfg)
}

func TestExpandEnvLoadedConfigMissingEnv(t *testing.T) {
	assert.NoError(t, os.Setenv("NESTED_VALUE", "replaced_nested_value"))

	defer func() {
		assert.NoError(t, os.Unsetenv("NESTED_VALUE"))
	}()

	testString := "$PTR_VALUE"

	cfg := &testConfig{
		ExporterSettings: config.NewExporterSettings(config.NewID("test")),
		NestedConfigPtr: &nestedConfig{
			NestedStringValue: "$NESTED_VALUE",
			NestedIntValue:    1,
		},
		NestedConfigValue: nestedConfig{
			NestedStringValue: "$NESTED_VALUE",
			NestedIntValue:    2,
		},
		StringValue:    "$VALUE",
		StringPtrValue: &testString,
		IntValue:       3,
	}

	expandEnvLoadedConfig(cfg)

	replacedTestString := ""

	assert.Equal(t, &testConfig{
		ExporterSettings: config.NewExporterSettings(config.NewID("test")),
		NestedConfigPtr: &nestedConfig{
			NestedStringValue: "replaced_nested_value",
			NestedIntValue:    1,
		},
		NestedConfigValue: nestedConfig{
			NestedStringValue: "replaced_nested_value",
			NestedIntValue:    2,
		},
		StringValue:    "",
		StringPtrValue: &replacedTestString,
		IntValue:       3,
	}, cfg)
}

func TestExpandEnvLoadedConfigNil(t *testing.T) {
	var cfg *testConfig

	// This should safely do nothing
	expandEnvLoadedConfig(cfg)

	assert.Equal(t, (*testConfig)(nil), cfg)
}

func TestExpandEnvLoadedConfigNoPointer(t *testing.T) {
	assert.NoError(t, os.Setenv("VALUE", "replaced_value"))

	cfg := testConfig{
		StringValue: "$VALUE",
	}

	// This should do nothing as cfg is not a pointer
	expandEnvLoadedConfig(cfg)

	assert.Equal(t, testConfig{StringValue: "$VALUE"}, cfg)
}

type testUnexportedConfig struct {
	config.ExporterSettings

	unexportedStringValue string
	ExportedStringValue   string
}

func TestExpandEnvLoadedConfigUnexportedField(t *testing.T) {
	assert.NoError(t, os.Setenv("VALUE", "replaced_value"))

	defer func() {
		assert.NoError(t, os.Unsetenv("VALUE"))
	}()

	cfg := &testUnexportedConfig{
		unexportedStringValue: "$VALUE",
		ExportedStringValue:   "$VALUE",
	}

	expandEnvLoadedConfig(cfg)

	assert.Equal(t, &testUnexportedConfig{
		unexportedStringValue: "$VALUE",
		ExportedStringValue:   "replaced_value",
	}, cfg)
}
