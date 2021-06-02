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

package configloader

import (
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/spf13/cast"
	"github.com/spf13/viper"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configparser"
)

// These are errors that can be returned by Load(). Note that error codes are not part
// of Load()'s public API, they are for internal unit testing only.
type configErrorCode int

const (
	// Skip 0, start errors codes from 1.
	_ configErrorCode = iota

	errInvalidTypeAndNameKey
	errUnknownType
	errDuplicateName
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
	Receivers  map[string]map[string]interface{} `mapstructure:"receivers"`
	Processors map[string]map[string]interface{} `mapstructure:"processors"`
	Exporters  map[string]map[string]interface{} `mapstructure:"exporters"`
	Extensions map[string]map[string]interface{} `mapstructure:"extensions"`
	Service    serviceSettings                   `mapstructure:"service"`
}

type serviceSettings struct {
	Extensions []string                    `mapstructure:"extensions"`
	Pipelines  map[string]pipelineSettings `mapstructure:"pipelines"`
}

type pipelineSettings struct {
	Receivers  []string `mapstructure:"receivers"`
	Processors []string `mapstructure:"processors"`
	Exporters  []string `mapstructure:"exporters"`
}

// Load loads a Config from Parser.
// After loading the config, `Validate()` must be called to validate.
func Load(v *configparser.Parser, factories component.Factories) (*config.Config, error) {
	var cfg config.Config

	// Load the config.

	// Struct to validate top level sections.
	var rawCfg configSettings
	if err := v.UnmarshalExact(&rawCfg); err != nil {
		return nil, &configError{
			code: errUnmarshalTopLevelStructureError,
			msg:  fmt.Sprintf("error reading top level configuration sections: %s", err.Error()),
		}
	}

	// In the following section use v.GetStringMap(xyzKeyName) instead of rawCfg.Xyz, because
	// UnmarshalExact will not unmarshal entries in the map[string]interface{} with nil values.
	// GetStringMap does the correct thing.

	// Start with the service extensions.

	extensions, err := loadExtensions(cast.ToStringMap(v.Get(extensionsKeyName)), factories.Extensions)
	if err != nil {
		return nil, err
	}
	cfg.Extensions = extensions

	// Load data components (receivers, exporters, and processors).

	receivers, err := loadReceivers(cast.ToStringMap(v.Get(receiversKeyName)), factories.Receivers)
	if err != nil {
		return nil, err
	}
	cfg.Receivers = receivers

	exporters, err := loadExporters(cast.ToStringMap(v.Get(exportersKeyName)), factories.Exporters)
	if err != nil {
		return nil, err
	}
	cfg.Exporters = exporters

	processors, err := loadProcessors(cast.ToStringMap(v.Get(processorsKeyName)), factories.Processors)
	if err != nil {
		return nil, err
	}
	cfg.Processors = processors

	// Load the service and its data pipelines.
	service, err := loadService(rawCfg.Service)
	if err != nil {
		return nil, err
	}
	cfg.Service = service

	return &cfg, nil
}

func errorInvalidTypeAndNameKey(component, key string, err error) error {
	return &configError{
		code: errInvalidTypeAndNameKey,
		msg:  fmt.Sprintf("invalid %s type and name key %q: %v", component, key, err),
	}
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

func errorDuplicateName(component string, id config.ComponentID) error {
	return &configError{
		code: errDuplicateName,
		msg:  fmt.Sprintf("duplicate %s name %v", component, id),
	}
}

func loadExtensions(exts map[string]interface{}, factories map[config.Type]component.ExtensionFactory) (config.Extensions, error) {
	// Prepare resulting map.
	extensions := make(config.Extensions)

	// Iterate over extensions and create a config for each.
	for key, value := range exts {
		componentConfig := configparser.NewParserFromStringMap(cast.ToStringMap(value))
		expandEnvConfig(componentConfig)

		// Decode the key into type and fullName components.
		id, err := config.NewIDFromString(key)
		if err != nil {
			return nil, errorInvalidTypeAndNameKey(extensionsKeyName, key, err)
		}

		// Find extension factory based on "type" that we read from config source.
		factory := factories[id.Type()]
		if factory == nil {
			return nil, errorUnknownType(extensionsKeyName, id)
		}

		// Create the default config for this extension.
		extensionCfg := factory.CreateDefaultConfig()
		extensionCfg.SetIDName(id.Name())
		expandEnvLoadedConfig(extensionCfg)

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		unm := unmarshaler(factory)
		if err := unm(componentConfig, extensionCfg); err != nil {
			return nil, errorUnmarshalError(extensionsKeyName, id, err)
		}

		if extensions[id] != nil {
			return nil, errorDuplicateName(extensionsKeyName, id)
		}

		extensions[id] = extensionCfg
	}

	return extensions, nil
}

func loadService(rawService serviceSettings) (config.Service, error) {
	var ret config.Service
	ret.Extensions = make([]config.ComponentID, 0, len(rawService.Extensions))
	for _, extIDStr := range rawService.Extensions {
		id, err := config.NewIDFromString(extIDStr)
		if err != nil {
			return ret, err
		}
		ret.Extensions = append(ret.Extensions, id)
	}

	// Process the pipelines first so in case of error on them it can be properly
	// reported.
	pipelines, err := loadPipelines(rawService.Pipelines)
	ret.Pipelines = pipelines

	return ret, err
}

// LoadReceiver loads a receiver config from componentConfig using the provided factories.
func LoadReceiver(componentConfig *configparser.Parser, id config.ComponentID, factory component.ReceiverFactory) (config.Receiver, error) {
	// Create the default config for this receiver.
	receiverCfg := factory.CreateDefaultConfig()
	receiverCfg.SetIDName(id.Name())
	expandEnvLoadedConfig(receiverCfg)

	// Now that the default config struct is created we can Unmarshal into it
	// and it will apply user-defined config on top of the default.
	unm := unmarshaler(factory)
	if err := unm(componentConfig, receiverCfg); err != nil {
		return nil, errorUnmarshalError(receiversKeyName, id, err)
	}

	return receiverCfg, nil
}

func loadReceivers(recvs map[string]interface{}, factories map[config.Type]component.ReceiverFactory) (config.Receivers, error) {
	// Prepare resulting map.
	receivers := make(config.Receivers)

	// Iterate over input map and create a config for each.
	for key, value := range recvs {
		componentConfig := configparser.NewParserFromStringMap(cast.ToStringMap(value))
		expandEnvConfig(componentConfig)

		// Decode the key into type and fullName components.
		id, err := config.NewIDFromString(key)
		if err != nil {
			return nil, errorInvalidTypeAndNameKey(receiversKeyName, key, err)
		}

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

		if receivers[id] != nil {
			return nil, errorDuplicateName(receiversKeyName, id)
		}
		receivers[id] = receiverCfg
	}

	return receivers, nil
}

func loadExporters(exps map[string]interface{}, factories map[config.Type]component.ExporterFactory) (config.Exporters, error) {
	// Prepare resulting map.
	exporters := make(config.Exporters)

	// Iterate over Exporters and create a config for each.
	for key, value := range exps {
		componentConfig := configparser.NewParserFromStringMap(cast.ToStringMap(value))
		expandEnvConfig(componentConfig)

		// Decode the key into type and fullName components.
		id, err := config.NewIDFromString(key)
		if err != nil {
			return nil, errorInvalidTypeAndNameKey(exportersKeyName, key, err)
		}

		// Find exporter factory based on "type" that we read from config source.
		factory := factories[id.Type()]
		if factory == nil {
			return nil, errorUnknownType(exportersKeyName, id)
		}

		// Create the default config for this exporter.
		exporterCfg := factory.CreateDefaultConfig()
		exporterCfg.SetIDName(id.Name())
		expandEnvLoadedConfig(exporterCfg)

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		unm := unmarshaler(factory)
		if err := unm(componentConfig, exporterCfg); err != nil {
			return nil, errorUnmarshalError(exportersKeyName, id, err)
		}

		if exporters[id] != nil {
			return nil, errorDuplicateName(exportersKeyName, id)
		}

		exporters[id] = exporterCfg
	}

	return exporters, nil
}

func loadProcessors(procs map[string]interface{}, factories map[config.Type]component.ProcessorFactory) (config.Processors, error) {
	// Prepare resulting map.
	processors := make(config.Processors)

	// Iterate over processors and create a config for each.
	for key, value := range procs {
		componentConfig := configparser.NewParserFromStringMap(cast.ToStringMap(value))
		expandEnvConfig(componentConfig)

		// Decode the key into type and fullName components.
		id, err := config.NewIDFromString(key)
		if err != nil {
			return nil, errorInvalidTypeAndNameKey(processorsKeyName, key, err)
		}

		// Find processor factory based on "type" that we read from config source.
		factory := factories[id.Type()]
		if factory == nil {
			return nil, errorUnknownType(processorsKeyName, id)
		}

		// Create the default config for this processor.
		processorCfg := factory.CreateDefaultConfig()
		processorCfg.SetIDName(id.Name())
		expandEnvLoadedConfig(processorCfg)

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		unm := unmarshaler(factory)
		if err := unm(componentConfig, processorCfg); err != nil {
			return nil, errorUnmarshalError(processorsKeyName, id, err)
		}

		if processors[id] != nil {
			return nil, errorDuplicateName(processorsKeyName, id)
		}

		processors[id] = processorCfg
	}

	return processors, nil
}

func loadPipelines(pipelinesConfig map[string]pipelineSettings) (config.Pipelines, error) {
	// Prepare resulting map.
	pipelines := make(config.Pipelines)

	// Iterate over input map and create a config for each.
	for key, rawPipeline := range pipelinesConfig {
		// Decode the key into type and name components.
		id, err := config.NewIDFromString(key)
		if err != nil {
			return nil, errorInvalidTypeAndNameKey(pipelinesKeyName, key, err)
		}
		fullName := id.String()

		// Create the config for this pipeline.
		var pipelineCfg config.Pipeline

		// Set the type.
		pipelineCfg.InputType = config.DataType(id.Type())
		switch pipelineCfg.InputType {
		case config.TracesDataType:
		case config.MetricsDataType:
		case config.LogsDataType:
		default:
			return nil, errorUnknownType(pipelinesKeyName, id)
		}

		pipelineCfg.Name = fullName
		if pipelineCfg.Receivers, err = parseIDNames(id, receiversKeyName, rawPipeline.Receivers); err != nil {
			return nil, err
		}
		if pipelineCfg.Processors, err = parseIDNames(id, processorsKeyName, rawPipeline.Processors); err != nil {
			return nil, err
		}
		if pipelineCfg.Exporters, err = parseIDNames(id, exportersKeyName, rawPipeline.Exporters); err != nil {
			return nil, err
		}

		if pipelines[fullName] != nil {
			return nil, errorDuplicateName(pipelinesKeyName, id)
		}

		pipelines[fullName] = &pipelineCfg
	}

	return pipelines, nil
}

func parseIDNames(pipelineID config.ComponentID, componentType string, names []string) ([]config.ComponentID, error) {
	var ret []config.ComponentID
	for _, idProcStr := range names {
		idRecv, err := config.NewIDFromString(idProcStr)
		if err != nil {
			return nil, fmt.Errorf("pipelines: config for %v contains invalid %s name %s : %w", pipelineID, componentType, idProcStr, err)
		}
		ret = append(ret, idRecv)
	}
	return ret, nil
}

// expandEnvConfig creates a new viper config with expanded values for all the values (simple, list or map value).
// It does not expand the keys.
func expandEnvConfig(v *configparser.Parser) {
	for _, k := range v.AllKeys() {
		v.Set(k, expandStringValues(v.Get(k)))
	}
}

func expandStringValues(value interface{}) interface{} {
	switch v := value.(type) {
	default:
		return v
	case string:
		return expandEnv(v)
	case []interface{}:
		nslice := make([]interface{}, 0, len(v))
		for _, vint := range v {
			nslice = append(nslice, expandStringValues(vint))
		}
		return nslice
	case map[interface{}]interface{}:
		nmap := make(map[interface{}]interface{}, len(v))
		for k, vint := range v {
			nmap[k] = expandStringValues(vint)
		}
		return nmap
	}
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

// deprecatedUnmarshaler interface is a deprecated optional interface that if implemented by a Factory,
// the configuration loading system will use to unmarshal the config.
// Implement config.CustomUnmarshable on the configuration struct instead.
type deprecatedUnmarshaler interface {
	// Unmarshal is a function that un-marshals a viper data into a config struct in a custom way.
	// componentViperSection *viper.Viper
	//   The config for this specific component. May be nil or empty if no config available.
	// intoCfg interface{}
	//   An empty interface wrapping a pointer to the config struct to unmarshal into.
	Unmarshal(componentViperSection *viper.Viper, intoCfg interface{}) error
}

func unmarshal(componentSection *configparser.Parser, intoCfg interface{}) error {
	if cu, ok := intoCfg.(config.CustomUnmarshable); ok {
		return cu.Unmarshal(componentSection)
	}

	return componentSection.UnmarshalExact(intoCfg)
}

// unmarshaler returns an unmarshaling function. It should be removed when deprecatedUnmarshaler is removed.
func unmarshaler(factory component.Factory) func(componentViperSection *configparser.Parser, intoCfg interface{}) error {
	if _, ok := factory.(deprecatedUnmarshaler); ok {
		return func(componentParser *configparser.Parser, intoCfg interface{}) error {
			return errors.New("deprecated way to specify custom unmarshaler no longer supported")
		}
	}
	return unmarshal
}
