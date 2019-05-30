// Copyright 2019, OpenCensus Authors
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

package configv2

import (
	"os"
	"path"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/census-instrumentation/opencensus-service/internal/configmodels"
	"github.com/census-instrumentation/opencensus-service/internal/factories"
)

// ExampleReceiver: for testing purposes we are defining an example config and factory
// for "examplereceiver" receiver type.
type ExampleReceiver struct {
	configmodels.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraSetting                  string                   `mapstructure:"extra"`
}

type ExampleReceiverFactory struct {
}

// Type gets the type of the Receiver config created by this factory.
func (f *ExampleReceiverFactory) Type() string {
	return "examplereceiver"
}

// CreateDefaultConfig creates the default configuration for the Receiver.
func (f *ExampleReceiverFactory) CreateDefaultConfig() configmodels.Receiver {
	return &ExampleReceiver{
		ReceiverSettings: configmodels.ReceiverSettings{
			Endpoint: "localhost:1000",
			Enabled:  false,
		},
		ExtraSetting: "some string",
	}
}

// MultiProtoReceiver: for testing purposes we are defining an example multi protocol
// config and factory for "multireceiver" receiver type.
type MultiProtoReceiver struct {
	Protocols map[string]MultiProtoReceiverOneCfg `mapstructure:"protocols"`
}

var _ configmodels.Receiver = (*MultiProtoReceiver)(nil)

type MultiProtoReceiverOneCfg struct {
	Enabled      bool   `mapstructure:"enabled"`
	Endpoint     string `mapstructure:"endpoint"`
	ExtraSetting string `mapstructure:"extra"`
}

type MultiProtoReceiverFactory struct {
}

// Type gets the type of the Receiver config created by this factory.
func (f *MultiProtoReceiverFactory) Type() string {
	return "multireceiver"
}

// CreateDefaultConfig creates the default configuration for the Receiver.
func (f *MultiProtoReceiverFactory) CreateDefaultConfig() configmodels.Receiver {
	return &MultiProtoReceiver{
		Protocols: map[string]MultiProtoReceiverOneCfg{
			"http": {
				Enabled:      false,
				Endpoint:     "example.com:8888",
				ExtraSetting: "extra string 1",
			},
			"tcp": {
				Enabled:      false,
				Endpoint:     "omnition.com:9999",
				ExtraSetting: "extra string 2",
			},
		},
	}
}

// ExampleExporter: for testing purposes we are defining an example config and factory
// for "exampleexporter" exporter type.
type ExampleExporter struct {
	configmodels.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraSetting                  string                   `mapstructure:"extra"`
}

type ExampleExporterFactory struct {
}

// Type gets the type of the Exporter config created by this factory.
func (f *ExampleExporterFactory) Type() string {
	return "exampleexporter"
}

// CreateDefaultConfig creates the default configuration for the Exporter.
func (f *ExampleExporterFactory) CreateDefaultConfig() configmodels.Exporter {
	return &ExampleExporter{
		ExporterSettings: configmodels.ExporterSettings{
			Enabled: false,
		},
		ExtraSetting: "some export string",
	}
}

// ExampleProcessor: for testing purposes we are defining an example config and factory
// for "exampleprocessor" processor type.
type ExampleProcessor struct {
	configmodels.ProcessorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraSetting                   string                   `mapstructure:"extra"`
}

type ExampleProcessorFactory struct {
}

// Type gets the type of the Processor config created by this factory.
func (f *ExampleProcessorFactory) Type() string {
	return "exampleprocessor"
}

// CreateDefaultConfig creates the default configuration for the Processor.
func (f *ExampleProcessorFactory) CreateDefaultConfig() configmodels.Processor {
	return &ExampleProcessor{
		ProcessorSettings: configmodels.ProcessorSettings{
			Enabled: false,
		},
		ExtraSetting: "some export string",
	}
}

// Register all factories
var _ = factories.RegisterReceiverFactory(&ExampleReceiverFactory{})
var _ = factories.RegisterReceiverFactory(&MultiProtoReceiverFactory{})
var _ = factories.RegisterExporterFactory(&ExampleExporterFactory{})
var _ = factories.RegisterProcessorFactory(&ExampleProcessorFactory{})

func loadConfigFile(t *testing.T, fileName string) (*configmodels.ConfigV2, error) {
	// Open the file for reading.
	file, err := os.Open(fileName)
	if err != nil {
		t.Error(err)
		return nil, err
	}

	// Read yaml config from file
	v := viper.New()
	v.SetConfigType("yaml")
	err = v.ReadConfig(file)
	if err != nil {
		t.Errorf("unable to read yaml, %v", err)
		return nil, err
	}

	// Load the config from viper
	return Load(v)
}

func TestDecodeConfig(t *testing.T) {

	// Load the config
	config, err := loadConfigFile(t, path.Join(".", "testdata", "valid-config.yaml"))
	if err != nil {
		t.Fatalf("unable to load config, %v", err)
	}

	// Verify receivers
	assert.Equal(t, len(config.Receivers), 2, "Incorrect receivers count")

	assert.Equal(t, config.Receivers["examplereceiver"],
		&ExampleReceiver{
			ReceiverSettings: configmodels.ReceiverSettings{
				Endpoint: "localhost:1000",
				Enabled:  false,
			},
			ExtraSetting: "some string",
		}, "Did not load receiver config correctly")

	assert.Equal(t, config.Receivers["examplereceiver/myreceiver"],
		&ExampleReceiver{
			ReceiverSettings: configmodels.ReceiverSettings{
				Endpoint: "127.0.0.1:12345",
				Enabled:  true,
			},
			ExtraSetting: "some string",
		}, "Did not load receiver config correctly")

	// Verify exporters
	assert.Equal(t, len(config.Exporters), 2, "Incorrect exporters count")

	assert.Equal(t, config.Exporters["exampleexporter"],
		&ExampleExporter{
			ExporterSettings: configmodels.ExporterSettings{
				Enabled: false,
			},
			ExtraSetting: "some export string",
		}, "Did not load exporter config correctly")

	assert.Equal(t, config.Exporters["exampleexporter/myexporter"],
		&ExampleExporter{
			ExporterSettings: configmodels.ExporterSettings{
				Enabled: true,
			},
			ExtraSetting: "some export string 2",
		}, "Did not load exporter config correctly")

	// Verify Processors
	assert.Equal(t, len(config.Processors), 1, "Incorrect processors count")

	assert.Equal(t, config.Processors["exampleprocessor"],
		&ExampleProcessor{
			ProcessorSettings: configmodels.ProcessorSettings{
				Enabled: false,
			},
			ExtraSetting: "some export string",
		}, "Did not load processor config correctly")

	// Verify Pipelines
	assert.Equal(t, len(config.Pipelines), 1, "Incorrect pipelines count")

	assert.Equal(t, config.Pipelines["traces"],
		&configmodels.Pipeline{
			Name:       "traces",
			InputType:  configmodels.TracesDataType,
			Receivers:  []string{"examplereceiver"},
			Processors: []string{"exampleprocessor"},
			Exporters:  []string{"exampleexporter"},
		}, "Did not load pipeline config correctly")
}

func TestDecodeConfig_MultiProto(t *testing.T) {

	// Load the config
	config, err := loadConfigFile(t, path.Join(".", "testdata", "multiproto-config.yaml"))
	if err != nil {
		t.Fatalf("unable to load config, %v", err)
	}

	// Verify receivers
	assert.Equal(t, len(config.Receivers), 2, "Incorrect receivers count")

	assert.Equal(t, config.Receivers["multireceiver"],
		&MultiProtoReceiver{
			Protocols: map[string]MultiProtoReceiverOneCfg{
				"http": {
					Enabled:      false,
					Endpoint:     "example.com:8888",
					ExtraSetting: "extra string 1",
				},
				"tcp": {
					Enabled:      false,
					Endpoint:     "omnition.com:9999",
					ExtraSetting: "extra string 2",
				},
			},
		}, "Did not load receiver config correctly")

	assert.Equal(t, config.Receivers["multireceiver/myreceiver"],
		&MultiProtoReceiver{
			Protocols: map[string]MultiProtoReceiverOneCfg{
				"http": {
					Enabled:      true,
					Endpoint:     "127.0.0.1:12345",
					ExtraSetting: "some string 1",
				},
				"tcp": {
					Enabled:      true,
					Endpoint:     "0.0.0.0:4567",
					ExtraSetting: "some string 2",
				},
			},
		}, "Did not load receiver config correctly")
}

func TestDecodeConfig_Invalid(t *testing.T) {

	var testCases = []struct {
		name     string          // test case name (also file name containing config yaml)
		expected configErrorCode // expected error (if nil any error is acceptable)
	}{
		{name: "empty-config"},
		{name: "missing-all-sections"},
		{name: "missing-receivers"},
		{name: "missing-exporters"},
		{name: "missing-processors"},
		{name: "invalid-receiver-name"},
		{name: "invalid-receiver-reference", expected: errPipelineReceiverNotExists},
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
		{name: "metric-pipeline-cannot-have-processors", expected: errMetricPipelineCannotHaveProcessors},
		{name: "unknown-receiver-type", expected: errUnknownReceiverType},
		{name: "unknown-exporter-type", expected: errUnknownExporterType},
		{name: "unknown-processor-type", expected: errUnknownProcessorType},
		{name: "invalid-bool-value", expected: errUnmarshalError},
		{name: "invalid-sequence-value", expected: errUnmarshalError},
		{name: "invalid-enabled-bool-value", expected: errUnmarshalError},
		{name: "invalid-enabled-bool-value2", expected: errUnmarshalError},
		{name: "invalid-pipeline-type", expected: errInvalidPipelineType},
		{name: "invalid-pipeline-type-and-name", expected: errInvalidTypeAndNameKey},
		{name: "duplicate-receiver", expected: errDuplicateReceiverName},
		{name: "duplicate-exporter", expected: errDuplicateExporterName},
		{name: "duplicate-processor", expected: errDuplicateProcessorName},
		{name: "duplicate-pipeline", expected: errDuplicatePipelineName},
	}

	for _, test := range testCases {
		_, err := loadConfigFile(t, path.Join(".", "testdata", test.name+".yaml"))
		if err == nil {
			t.Errorf("expected error but succedded on invalid config case: %s", test.name)
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
