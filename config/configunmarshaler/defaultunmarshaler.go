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

package configunmarshaler // import "go.opentelemetry.io/collector/config/configunmarshaler"

import (
	"fmt"
	"os"
	"reflect"

	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

// These are errors that can be returned by Unmarshal(). Note that error codes are not part
// of Unmarshal()'s public API, they are for internal unit testing only.
type configErrorCode int

const (
	// Skip 0, start errors codes from 1.
	_ configErrorCode = iota

	errUnknownType
	errUnmarshalTopLevelStructureError
)

type configError struct {
	// Human readable error message.
	msg string

	// Internal error code.
	code configErrorCode
}

func (e *configError) Error() string {
	return e.msg
}

// YAML top-level configuration keys.
const (
	// extensionsKeyName is the configuration key name for extensions section.
	extensionsKeyName = "extensions"

	// receiversKeyName is the configuration key name for receivers section.
	receiversKeyName = "receivers"

	// exportersKeyName is the configuration key name for exporters section.
	exportersKeyName = "exporters"

	// processorsKeyName is the configuration key name for processors section.
	processorsKeyName = "processors"

	// pipelinesKeyName is the configuration key name for pipelines section.
	pipelinesKeyName = "pipelines"
)

type configSettings struct {
	Receivers  map[config.ComponentID]map[string]interface{} `mapstructure:"receivers"`
	Processors map[config.ComponentID]map[string]interface{} `mapstructure:"processors"`
	Exporters  map[config.ComponentID]map[string]interface{} `mapstructure:"exporters"`
	Extensions map[config.ComponentID]map[string]interface{} `mapstructure:"extensions"`
	Service    serviceSettings                               `mapstructure:"service"`
}

type serviceSettings struct {
	Telemetry  config.ServiceTelemetry                 `mapstructure:"telemetry"`
	Extensions []config.ComponentID                    `mapstructure:"extensions"`
	Pipelines  map[config.ComponentID]*config.Pipeline `mapstructure:"pipelines"`
}

type defaultUnmarshaler struct{}

// NewDefault returns a default ConfigUnmarshaler that unmarshalls every configuration
// using the custom unmarshaler if present or default to strict
func NewDefault() ConfigUnmarshaler {
	return &defaultUnmarshaler{}
}

// Unmarshal the Config from a config.Map.
// After the config is unmarshaled, `Validate()` must be called to validate.
func (*defaultUnmarshaler) Unmarshal(v *config.Map, factories component.Factories) (*config.Config, error) {
	var cfg config.Config

	// Unmarshal the config.

	// Struct to validate top level sections.
	rawCfg := configSettings{
		// Setup default telemetry values as in service/logger.go.
		// TODO: Add a component.ServiceFactory to allow this to be defined by the Service.
		Service: serviceSettings{
			Telemetry: config.ServiceTelemetry{
				Logs: config.ServiceTelemetryLogs{
					Level:       zapcore.InfoLevel,
					Development: false,
					Encoding:    "console",
				},
			},
		},
	}
	if err := v.UnmarshalExact(&rawCfg); err != nil {
		return nil, &configError{
			code: errUnmarshalTopLevelStructureError,
			msg:  fmt.Sprintf("error reading top level configuration sections: %s", err.Error()),
		}
	}

	// Start with the service extensions.

	extensions, err := unmarshalExtensions(rawCfg.Extensions, factories.Extensions)
	if err != nil {
		return nil, err
	}
	cfg.Extensions = extensions

	// Unmarshal data components (receivers, exporters, and processors).

	receivers, err := unmarshalReceivers(rawCfg.Receivers, factories.Receivers)
	if err != nil {
		return nil, err
	}
	cfg.Receivers = receivers

	exporters, err := unmarshalExporters(rawCfg.Exporters, factories.Exporters)
	if err != nil {
		return nil, err
	}
	cfg.Exporters = exporters

	processors, err := unmarshalProcessors(rawCfg.Processors, factories.Processors)
	if err != nil {
		return nil, err
	}
	cfg.Processors = processors

	// Unmarshal the service and its data pipelines.
	service, err := unmarshalService(rawCfg.Service)
	if err != nil {
		return nil, err
	}
	cfg.Service = service

	return &cfg, nil
}

func errorUnknownType(component string, id config.ComponentID) error {
	return &configError{
		code: errUnknownType,
		msg:  fmt.Sprintf("unknown %s type %q for %v", component, id.Type(), id),
	}
}

func errorUnmarshalError(component string, id config.ComponentID, err error) error {
	return &configError{
		code: errUnmarshalTopLevelStructureError,
		msg:  fmt.Sprintf("error reading %s configuration for %v: %v", component, id, err),
	}
}

func unmarshalExtensions(exts map[config.ComponentID]map[string]interface{}, factories map[config.Type]component.ExtensionFactory) (config.Extensions, error) {
	// Prepare resulting map.
	extensions := make(config.Extensions)

	// Iterate over extensions and create a config for each.
	for id, value := range exts {
		componentConfig := config.NewMapFromStringMap(value)

		// Find extension factory based on "type" that we read from config source.
		factory := factories[id.Type()]
		if factory == nil {
			return nil, errorUnknownType(extensionsKeyName, id)
		}

		// Create the default config for this extension.
		extensionCfg := factory.CreateDefaultConfig()
		extensionCfg.SetIDName(id.Name())
		expandEnvLoadedConfig(extensionCfg)

		// Now that the default config struct is created we can Unmarshal into it,
		// and it will apply user-defined config on top of the default.
		if err := unmarshal(componentConfig, extensionCfg); err != nil {
			return nil, errorUnmarshalError(extensionsKeyName, id, err)
		}

		extensions[id] = extensionCfg
	}

	return extensions, nil
}

func unmarshalService(rawService serviceSettings) (config.Service, error) {
	var ret config.Service

	ret.Telemetry = rawService.Telemetry
	ret.Extensions = rawService.Extensions

	// Process the pipelines first so in case of error on them it can be properly
	// reported.
	pipelines, err := unmarshalPipelines(rawService.Pipelines)
	ret.Pipelines = pipelines

	return ret, err
}

// LoadReceiver loads a receiver config from componentConfig using the provided factories.
func LoadReceiver(componentConfig *config.Map, id config.ComponentID, factory component.ReceiverFactory) (config.Receiver, error) {
	// Create the default config for this receiver.
	receiverCfg := factory.CreateDefaultConfig()
	receiverCfg.SetIDName(id.Name())
	expandEnvLoadedConfig(receiverCfg)

	// Now that the default config struct is created we can Unmarshal into it,
	// and it will apply user-defined config on top of the default.
	if err := unmarshal(componentConfig, receiverCfg); err != nil {
		return nil, errorUnmarshalError(receiversKeyName, id, err)
	}

	return receiverCfg, nil
}

func unmarshalReceivers(recvs map[config.ComponentID]map[string]interface{}, factories map[config.Type]component.ReceiverFactory) (config.Receivers, error) {
	// Prepare resulting map.
	receivers := make(config.Receivers)

	// Iterate over input map and create a config for each.
	for id, value := range recvs {
		componentConfig := config.NewMapFromStringMap(value)

		// Find receiver factory based on "type" that we read from config source.
		factory := factories[id.Type()]
		if factory == nil {
			return nil, errorUnknownType(receiversKeyName, id)
		}

		receiverCfg, err := LoadReceiver(componentConfig, id, factory)

		if err != nil {
			// LoadReceiver already wraps the error.
			return nil, err
		}

		receivers[id] = receiverCfg
	}

	return receivers, nil
}

func unmarshalExporters(exps map[config.ComponentID]map[string]interface{}, factories map[config.Type]component.ExporterFactory) (config.Exporters, error) {
	// Prepare resulting map.
	exporters := make(config.Exporters)

	// Iterate over Exporters and create a config for each.
	for id, value := range exps {
		componentConfig := config.NewMapFromStringMap(value)

		// Find exporter factory based on "type" that we read from config source.
		factory := factories[id.Type()]
		if factory == nil {
			return nil, errorUnknownType(exportersKeyName, id)
		}

		// Create the default config for this exporter.
		exporterCfg := factory.CreateDefaultConfig()
		exporterCfg.SetIDName(id.Name())
		expandEnvLoadedConfig(exporterCfg)

		// Now that the default config struct is created we can Unmarshal into it,
		// and it will apply user-defined config on top of the default.
		if err := unmarshal(componentConfig, exporterCfg); err != nil {
			return nil, errorUnmarshalError(exportersKeyName, id, err)
		}

		exporters[id] = exporterCfg
	}

	return exporters, nil
}

func unmarshalProcessors(procs map[config.ComponentID]map[string]interface{}, factories map[config.Type]component.ProcessorFactory) (config.Processors, error) {
	// Prepare resulting map.
	processors := make(config.Processors)

	// Iterate over processors and create a config for each.
	for id, value := range procs {
		componentConfig := config.NewMapFromStringMap(value)

		// Find processor factory based on "type" that we read from config source.
		factory := factories[id.Type()]
		if factory == nil {
			return nil, errorUnknownType(processorsKeyName, id)
		}

		// Create the default config for this processor.
		processorCfg := factory.CreateDefaultConfig()
		processorCfg.SetIDName(id.Name())
		expandEnvLoadedConfig(processorCfg)

		// Now that the default config struct is created we can Unmarshal into it,
		// and it will apply user-defined config on top of the default.
		if err := unmarshal(componentConfig, processorCfg); err != nil {
			return nil, errorUnmarshalError(processorsKeyName, id, err)
		}

		processors[id] = processorCfg
	}

	return processors, nil
}

func unmarshalPipelines(rawPipelines map[config.ComponentID]*config.Pipeline) (config.Pipelines, error) {
	pipelines := make(map[string]*config.Pipeline)
	// Iterate over input map and create a config for each.
	for id, pipeline := range rawPipelines {
		pipeline.Name = id.String()
		pipeline.InputType = config.DataType(id.Type())
		switch pipeline.InputType {
		case config.TracesDataType:
		case config.MetricsDataType:
		case config.LogsDataType:
		default:
			return nil, errorUnknownType(pipelinesKeyName, id)
		}

		pipelines[pipeline.Name] = pipeline
	}

	return pipelines, nil
}

// expandEnvLoadedConfig is a utility function that goes recursively through a config object
// and tries to expand environment variables in its string fields.
func expandEnvLoadedConfig(s interface{}) {
	expandEnvLoadedConfigPointer(s)
}

func expandEnvLoadedConfigPointer(s interface{}) {
	// Check that the value given is indeed a pointer, otherwise safely stop the search here
	value := reflect.ValueOf(s)
	if value.Kind() != reflect.Ptr {
		return
	}
	// Run expandLoadedConfigValue on the value behind the pointer.
	expandEnvLoadedConfigValue(value.Elem())
}

func expandEnvLoadedConfigValue(value reflect.Value) {
	// The value given is a string, we expand it (if allowed).
	if value.Kind() == reflect.String && value.CanSet() {
		value.SetString(expandEnv(value.String()))
	}
	// The value given is a struct, we go through its fields.
	if value.Kind() == reflect.Struct {
		for i := 0; i < value.NumField(); i++ {
			// Returns the content of the field.
			field := value.Field(i)

			// Only try to modify a field if it can be modified (eg. skip unexported private fields).
			if field.CanSet() {
				switch field.Kind() {
				case reflect.String:
					// The current field is a string, expand env variables in the string.
					field.SetString(expandEnv(field.String()))
				case reflect.Ptr:
					// The current field is a pointer, run the expansion function on the pointer.
					expandEnvLoadedConfigPointer(field.Interface())
				case reflect.Struct:
					// The current field is a nested struct, go through the nested struct
					expandEnvLoadedConfigValue(field)
				}
			}
		}
	}
}

func expandEnv(s string) string {
	return os.Expand(s, func(str string) string {
		// This allows escaping environment variable substitution via $$, e.g.
		// - $FOO will be substituted with env var FOO
		// - $$FOO will be replaced with $FOO
		// - $$$FOO will be replaced with $ + substituted env var FOO
		if str == "$" {
			return "$"
		}
		return os.Getenv(str)
	})
}

func unmarshal(componentSection *config.Map, intoCfg interface{}) error {
	if cu, ok := intoCfg.(config.Unmarshallable); ok {
		return cu.Unmarshal(componentSection)
	}

	return componentSection.UnmarshalExact(intoCfg)
}
