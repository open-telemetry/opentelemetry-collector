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

package config

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
)

func TestDecodeConfig(t *testing.T) {
	factories, err := ExampleComponents()
	assert.Nil(t, err)

	// Load the config
	config, err := LoadConfigFile(t, path.Join(".", "testdata", "valid-config.yaml"), factories)
	if err != nil {
		t.Fatalf("unable to load config, %v", err)
	}

	// Verify extensions.
	assert.Equal(t, 3, len(config.Extensions))
	assert.False(t, config.Extensions["exampleextension/disabled"].IsEnabled())
	assert.Equal(t, "some string", config.Extensions["exampleextension/1"].(*ExampleExtension).ExtraSetting)

	// Verify service.
	assert.Equal(t, 2, len(config.Service.Extensions))
	assert.Equal(t, "exampleextension/0", config.Service.Extensions[0])
	assert.Equal(t, "exampleextension/1", config.Service.Extensions[1])

	// Verify receivers
	assert.Equal(t, 2, len(config.Receivers), "Incorrect receivers count")

	assert.Equal(t,
		&ExampleReceiver{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal:  "examplereceiver",
				NameVal:  "examplereceiver",
				Endpoint: "localhost:1000",
			},
			ExtraSetting: "some string",
		},
		config.Receivers["examplereceiver"],
		"Did not load receiver config correctly")

	assert.Equal(t,
		&ExampleReceiver{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal:  "examplereceiver",
				NameVal:  "examplereceiver/myreceiver",
				Endpoint: "127.0.0.1:12345",
			},
			ExtraSetting: "some string",
		},
		config.Receivers["examplereceiver/myreceiver"],
		"Did not load receiver config correctly")

	// Verify exporters
	assert.Equal(t, 2, len(config.Exporters), "Incorrect exporters count")

	assert.Equal(t,
		&ExampleExporter{
			ExporterSettings: configmodels.ExporterSettings{
				NameVal: "exampleexporter",
				TypeVal: "exampleexporter",
			},
			ExtraSetting: "some export string",
		},
		config.Exporters["exampleexporter"],
		"Did not load exporter config correctly")

	assert.Equal(t,
		&ExampleExporter{
			ExporterSettings: configmodels.ExporterSettings{
				NameVal: "exampleexporter/myexporter",
				TypeVal: "exampleexporter",
			},
			ExtraSetting: "some export string 2",
		},
		config.Exporters["exampleexporter/myexporter"],
		"Did not load exporter config correctly")

	// Verify Processors
	assert.Equal(t, 1, len(config.Processors), "Incorrect processors count")

	assert.Equal(t,
		&ExampleProcessor{
			ProcessorSettings: configmodels.ProcessorSettings{
				TypeVal: "exampleprocessor",
				NameVal: "exampleprocessor",
			},
			ExtraSetting: "some export string",
		},
		config.Processors["exampleprocessor"],
		"Did not load processor config correctly")

	// Verify Pipelines
	assert.Equal(t, 1, len(config.Pipelines), "Incorrect pipelines count")

	assert.Equal(t,
		&configmodels.Pipeline{
			Name:       "traces",
			InputType:  configmodels.TracesDataType,
			Receivers:  []string{"examplereceiver"},
			Processors: []string{"exampleprocessor"},
			Exporters:  []string{"exampleexporter"},
		},
		config.Pipelines["traces"],
		"Did not load pipeline config correctly")
}

func TestDecodeConfigWithEnv(t *testing.T) {
	const extensionExtra = "some extension string"
	const extensionExtraMapValue = "some extension map value"
	const extensionExtraListElement = "some extension list element"
	os.Setenv("EXTENSIONS_EXAMPLEEXTENSION_EXTRA", extensionExtra)
	os.Setenv("EXTENSIONS_EXAMPLEEXTENSION_EXTRA_MAP_EXT_VALUE", extensionExtraMapValue)
	os.Setenv("EXTENSIONS_EXAMPLEEXTENSION_EXTRA_LIST_ELEMENT_1", extensionExtraListElement)

	const receiverExtra = "some receiver string"
	const receiverExtraMapValue = "some receiver map value"
	const receiverExtraListElement = "some receiver list element"
	os.Setenv("RECEIVERS_EXAMPLERECEIVER_EXTRA", receiverExtra)
	os.Setenv("RECEIVERS_EXAMPLERECEIVER_EXTRA_MAP_RECV_VALUE", receiverExtraMapValue)
	os.Setenv("RECEIVERS_EXAMPLERECEIVER_EXTRA_LIST_ELEMENT_1", receiverExtraListElement)

	const processorExtra = "some processor string"
	const processorExtraMapValue = "some processor map value"
	const processorExtraListElement = "some processor list element"
	os.Setenv("PROCESSORS_EXAMPLEPROCESSOR_EXTRA", processorExtra)
	os.Setenv("PROCESSORS_EXAMPLEPROCESSOR_EXTRA_MAP_PROC_VALUE", processorExtraMapValue)
	os.Setenv("PROCESSORS_EXAMPLEPROCESSOR_EXTRA_LIST_ELEMENT_1", processorExtraListElement)

	const exporterExtra = "some exporter string"
	const exporterExtraMapValue = "some exporter map value"
	const exporterExtraListElement = "some exporter list element"
	os.Setenv("EXPORTERS_EXAMPLEEXPORTER_EXTRA", exporterExtra)
	os.Setenv("EXPORTERS_EXAMPLEEXPORTER_EXTRA_MAP_EXP_VALUE", exporterExtraMapValue)
	os.Setenv("EXPORTERS_EXAMPLEEXPORTER_EXTRA_LIST_ELEMENT_1", exporterExtraListElement)

	defer func() {
		os.Unsetenv("EXTENSIONS_EXAMPLEEXTENSION_EXTRA")
		os.Unsetenv("EXTENSIONS_EXAMPLEEXTENSION_EXTRA_MAP_EXT_VALUE")
		os.Unsetenv("EXTENSIONS_EXAMPLEEXTENSION_EXTRA_LIST_ELEMENT_1")

		os.Unsetenv("RECEIVERS_EXAMPLERECEIVER_EXTRA")
		os.Unsetenv("RECEIVERS_EXAMPLERECEIVER_EXTRA_MAP_RECV_VALUE")
		os.Unsetenv("RECEIVERS_EXAMPLERECEIVER_EXTRA_LIST_ELEMENT_1")

		os.Unsetenv("PROCESSORS_EXAMPLEPROCESSOR_EXTRA")
		os.Unsetenv("PROCESSORS_EXAMPLEPROCESSOR_EXTRA_MAP_PROC_VALUE")
		os.Unsetenv("PROCESSORS_EXAMPLEPROCESSOR_EXTRA_LIST_ELEMENT_1")

		os.Unsetenv("EXPORTERS_EXAMPLEEXPORTER_EXTRA")
		os.Unsetenv("EXPORTERS_EXAMPLEEXPORTER_EXTRA_MAP_EXP_VALUE")
		os.Unsetenv("EXPORTERS_EXAMPLEEXPORTER_EXTRA_LIST_ELEMENT_1")
	}()

	t.Log(expandStringValues([]string{"${EXTENSIONS_EXAMPLEEXTENSION_EXTRA}"}))

	factories, err := ExampleComponents()
	assert.Nil(t, err)

	// Load the config
	config, err := LoadConfigFile(t, path.Join(".", "testdata", "valid-config-with-env.yaml"), factories)
	if err != nil {
		t.Fatalf("unable to load config, %v", err)
	}

	// Verify extensions.
	assert.Equal(t, 1, len(config.Extensions))
	assert.True(t, config.Extensions["exampleextension"].IsEnabled())
	assert.Equal(t,
		&ExampleExtension{
			ExtensionSettings: configmodels.ExtensionSettings{
				TypeVal: "exampleextension",
				NameVal: "exampleextension",
			},
			ExtraSetting:     extensionExtra,
			ExtraMapSetting:  map[string]string{"ext": extensionExtraMapValue},
			ExtraListSetting: []string{extensionExtraListElement},
		},
		config.Extensions["exampleextension"],
		"Did not load extension config correctly")

	// Verify service.
	assert.Equal(t, 1, len(config.Service.Extensions))
	assert.Equal(t, "exampleextension", config.Service.Extensions[0])

	// Verify receivers
	assert.Equal(t, 1, len(config.Receivers), "Incorrect receivers count")
	assert.True(t, config.Receivers["examplereceiver"].IsEnabled())

	assert.Equal(t,
		&ExampleReceiver{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal:  "examplereceiver",
				NameVal:  "examplereceiver",
				Endpoint: "127.0.0.1:1234",
			},
			ExtraSetting:     receiverExtra,
			ExtraMapSetting:  map[string]string{"recv": receiverExtraMapValue},
			ExtraListSetting: []string{receiverExtraListElement},
		},
		config.Receivers["examplereceiver"],
		"Did not load receiver config correctly")

	// Verify exporters
	assert.Equal(t, 1, len(config.Exporters), "Incorrect exporters count")
	assert.True(t, config.Exporters["exampleexporter"].IsEnabled())

	assert.Equal(t,
		&ExampleExporter{
			ExporterSettings: configmodels.ExporterSettings{
				NameVal: "exampleexporter",
				TypeVal: "exampleexporter",
			},
			ExtraSetting:     exporterExtra,
			ExtraMapSetting:  map[string]string{"exp": exporterExtraMapValue},
			ExtraListSetting: []string{exporterExtraListElement},
		},
		config.Exporters["exampleexporter"],
		"Did not load exporter config correctly")

	// Verify Processors
	assert.Equal(t, 1, len(config.Processors), "Incorrect processors count")
	assert.True(t, config.Processors["exampleprocessor"].IsEnabled())

	assert.Equal(t,
		&ExampleProcessor{
			ProcessorSettings: configmodels.ProcessorSettings{
				TypeVal: "exampleprocessor",
				NameVal: "exampleprocessor",
			},
			ExtraSetting:     processorExtra,
			ExtraMapSetting:  map[string]string{"proc": processorExtraMapValue},
			ExtraListSetting: []string{processorExtraListElement},
		},
		config.Processors["exampleprocessor"],
		"Did not load processor config correctly")

	// Verify Pipelines
	assert.Equal(t, 1, len(config.Pipelines), "Incorrect pipelines count")

	assert.Equal(t,
		&configmodels.Pipeline{
			Name:       "traces",
			InputType:  configmodels.TracesDataType,
			Receivers:  []string{"examplereceiver"},
			Processors: []string{"exampleprocessor"},
			Exporters:  []string{"exampleexporter"},
		},
		config.Pipelines["traces"],
		"Did not load pipeline config correctly")
}

func TestDecodeConfig_MultiProto(t *testing.T) {
	factories, err := ExampleComponents()
	assert.Nil(t, err)

	// Load the config
	config, err := LoadConfigFile(t, path.Join(".", "testdata", "multiproto-config.yaml"), factories)
	if err != nil {
		t.Fatalf("unable to load config, %v", err)
	}

	// Verify receivers
	assert.Equal(t, 2, len(config.Receivers), "Incorrect receivers count")

	assert.Equal(t,
		&MultiProtoReceiver{
			TypeVal: "multireceiver",
			NameVal: "multireceiver",
			Protocols: map[string]MultiProtoReceiverOneCfg{
				"http": {
					Endpoint:     "example.com:8888",
					ExtraSetting: "extra string 1",
				},
				"tcp": {
					Endpoint:     "omnition.com:9999",
					ExtraSetting: "extra string 2",
				},
			},
		},
		config.Receivers["multireceiver"],
		"Did not load receiver config correctly")

	assert.Equal(t,
		&MultiProtoReceiver{
			TypeVal: "multireceiver",
			NameVal: "multireceiver/myreceiver",
			Protocols: map[string]MultiProtoReceiverOneCfg{
				"http": {
					Endpoint:     "127.0.0.1:12345",
					ExtraSetting: "some string 1",
				},
				"tcp": {
					Endpoint:     "0.0.0.0:4567",
					ExtraSetting: "some string 2",
				},
			},
		},
		config.Receivers["multireceiver/myreceiver"],
		"Did not load receiver config correctly")
}

func TestDecodeConfig_Invalid(t *testing.T) {

	var testCases = []struct {
		name     string          // test case name (also file name containing config yaml)
		expected configErrorCode // expected error (if nil any error is acceptable)
	}{
		{name: "empty-config"},
		{name: "missing-all-sections"},
		{name: "missing-enabled-exporters", expected: errMissingExporters},
		{name: "missing-enabled-receivers", expected: errMissingReceivers},
		{name: "missing-exporters", expected: errMissingExporters},
		{name: "missing-receivers", expected: errMissingReceivers},
		{name: "missing-processors"},
		{name: "invalid-extension-name", expected: errExtensionNotExists},
		{name: "invalid-receiver-name"},
		{name: "invalid-receiver-reference", expected: errPipelineReceiverNotExists},
		{name: "missing-extension-type", expected: errInvalidTypeAndNameKey},
		{name: "missing-receiver-type", expected: errInvalidTypeAndNameKey},
		{name: "missing-exporter-name-after-slash", expected: errInvalidTypeAndNameKey},
		{name: "missing-processor-type", expected: errInvalidTypeAndNameKey},
		{name: "missing-pipelines", expected: errMissingPipelines},
		{name: "pipeline-must-have-exporter", expected: errPipelineMustHaveExporter},
		{name: "pipeline-must-have-exporter2", expected: errPipelineMustHaveExporter},
		{name: "pipeline-must-have-receiver", expected: errPipelineMustHaveReceiver},
		{name: "pipeline-exporter-not-exists", expected: errPipelineExporterNotExists},
		{name: "pipeline-processor-not-exists", expected: errPipelineProcessorNotExists},
		{name: "pipeline-must-have-processors", expected: errPipelineMustHaveProcessors},
		{name: "unknown-extension-type", expected: errUnknownExtensionType},
		{name: "unknown-receiver-type", expected: errUnknownReceiverType},
		{name: "unknown-exporter-type", expected: errUnknownExporterType},
		{name: "unknown-processor-type", expected: errUnknownProcessorType},
		{name: "invalid-extension-disabled-value", expected: errUnmarshalError},
		{name: "invalid-service-extensions-value", expected: errUnmarshalError},
		{name: "invalid-bool-value", expected: errUnmarshalError},
		{name: "invalid-sequence-value", expected: errUnmarshalError},
		{name: "invalid-disabled-bool-value", expected: errUnmarshalError},
		{name: "invalid-disabled-bool-value2", expected: errUnmarshalError},
		{name: "invalid-pipeline-type", expected: errInvalidPipelineType},
		{name: "invalid-pipeline-type-and-name", expected: errInvalidTypeAndNameKey},
		{name: "duplicate-extension", expected: errDuplicateExtensionName},
		{name: "duplicate-receiver", expected: errDuplicateReceiverName},
		{name: "duplicate-exporter", expected: errDuplicateExporterName},
		{name: "duplicate-processor", expected: errDuplicateProcessorName},
		{name: "duplicate-pipeline", expected: errDuplicatePipelineName},
	}

	factories, err := ExampleComponents()
	assert.Nil(t, err)

	for _, test := range testCases {
		_, err := LoadConfigFile(t, path.Join(".", "testdata", test.name+".yaml"), factories)
		if err == nil {
			t.Errorf("expected error but succeeded on invalid config case: %s", test.name)
		} else if test.expected != 0 {
			cfgErr, ok := err.(*configError)
			if !ok {
				t.Errorf("expected config error code %v but got a different error '%v' on invalid config case: %s",
					test.expected, err, test.name)
			} else {
				if cfgErr.code != test.expected {
					t.Errorf("expected config error code %v but got error code %v on invalid config case: %s",
						test.expected, cfgErr.code, test.name)
				}

				if cfgErr.Error() == "" {
					t.Errorf("returned config error %v with empty error message", cfgErr.code)
				}
			}
		}
	}
}
