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

// Package config implements loading of configuration from Viper configuration.
// The implementation relies on registered factories that allow creating
// default configuration for each type of receiver/exporter/processor.
package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
)

// These are errors that can be returned by Load(). Note that error codes are not part
// of Load()'s public API, they are for internal unit testing only.
type configErrorCode int

const (
	_ configErrorCode = iota // skip 0, start errors codes from 1.
	errInvalidTypeAndNameKey
	errUnknownExtensionType
	errUnknownReceiverType
	errUnknownExporterType
	errUnknownProcessorType
	errInvalidPipelineType
	errDuplicateExtensionName
	errDuplicateReceiverName
	errDuplicateExporterName
	errDuplicateProcessorName
	errDuplicatePipelineName
	errMissingPipelines
	errPipelineMustHaveReceiver
	errPipelineMustHaveExporter
	errExtensionNotExists
	errPipelineReceiverNotExists
	errPipelineProcessorNotExists
	errPipelineExporterNotExists
	errMissingReceivers
	errMissingExporters
	errUnmarshalErrorOnTopLevelSection
	errUnmarshalErrorOnExtension
	errUnmarshalErrorOnService
	errUnmarshalErrorOnReceiver
	errUnmarshalErrorOnProcessor
	errUnmarshalErrorOnExporter
	errUnmarshalErrorOnPipeline
)

type configError struct {
	msg  string          // human readable error message.
	code configErrorCode // internal error code.
}

func (e *configError) Error() string {
	return e.msg
}

// YAML top-level configuration keys
const (
	// extensionsKeyName is the configuration key name for extensions section.
	extensionsKeyName = "extensions"

	// serviceKeyName is the configuration key name for service section.
	serviceKeyName = "service"

	// receiversKeyName is the configuration key name for receivers section.
	receiversKeyName = "receivers"

	// exportersKeyName is the configuration key name for exporters section.
	exportersKeyName = "exporters"

	// processorsKeyName is the configuration key name for processors section.
	processorsKeyName = "processors"

	// pipelinesKeyName is the configuration key name for pipelines section.
	pipelinesKeyName = "pipelines"
)

// typeAndNameSeparator is the separator that is used between type and name in type/name composite keys.
const typeAndNameSeparator = "/"

// Factories struct holds in a single type all component factories that
// can be handled by the Config.
type Factories struct {
	// Receivers maps receiver type names in the config to the respective factory.
	Receivers map[configmodels.Type]component.ReceiverFactoryBase

	// Processors maps processor type names in the config to the respective factory.
	Processors map[configmodels.Type]component.ProcessorFactoryBase

	// Exporters maps exporter type names in the config to the respective factory.
	Exporters map[configmodels.Type]component.ExporterFactoryBase

	// Extensions maps extension type names in the config to the respective factory.
	Extensions map[configmodels.Type]component.ExtensionFactory
}

// Creates a new Viper instance with a different key-delimitor "::" instead of the
// default ".". This way configs can have keys that contain ".".
func NewViper() *viper.Viper {
	return viper.NewWithOptions(viper.KeyDelimiter("::"))
}

// Load loads a Config from Viper.
// After loading the config, need to check if it is valid by calling `ValidateConfig`.
func Load(
	v *viper.Viper,
	factories Factories,
) (*configmodels.Config, error) {

	var config configmodels.Config

	// Load the config.

	// Struct to validate top level sections.
	var topLevelSections struct {
		Extensions map[string]interface{} `mapstructure:"extensions"`
		Service    map[string]interface{} `mapstructure:"service"`
		Receivers  map[string]interface{} `mapstructure:"receivers"`
		Processors map[string]interface{} `mapstructure:"processors"`
		Exporters  map[string]interface{} `mapstructure:"exporters"`
	}

	if err := v.UnmarshalExact(&topLevelSections); err != nil {
		return nil, &configError{
			code: errUnmarshalErrorOnTopLevelSection,
			msg:  fmt.Sprintf("error reading top level sections: %s", err.Error()),
		}
	}

	// Start with the service extensions.

	extensions, err := loadExtensions(v, factories.Extensions)
	if err != nil {
		return nil, err
	}
	config.Extensions = extensions

	// Load data components (receivers, exporters, and processors).

	receivers, err := loadReceivers(v, factories.Receivers)
	if err != nil {
		return nil, err
	}
	config.Receivers = receivers

	exporters, err := loadExporters(v, factories.Exporters)
	if err != nil {
		return nil, err
	}
	config.Exporters = exporters

	processors, err := loadProcessors(v, factories.Processors)
	if err != nil {
		return nil, err
	}
	config.Processors = processors

	// Load the service and its data pipelines.
	service, err := loadService(v)
	if err != nil {
		return nil, err
	}
	config.Service = service

	return &config, nil
}

// DecodeTypeAndName decodes a key in type[/name] format into type and fullName.
// fullName is the key normalized such that type and name components have spaces trimmed.
// The "type" part must be present, the forward slash and "name" are optional. typeStr
// will be non-empty if err is nil.
func DecodeTypeAndName(key string) (typeStr configmodels.Type, fullName string, err error) {
	items := strings.SplitN(key, typeAndNameSeparator, 2)

	if len(items) >= 1 {
		typeStr = configmodels.Type(strings.TrimSpace(items[0]))
	}

	if len(items) == 0 || typeStr == "" {
		err = errors.New("type/name key must have the type part")
		return
	}

	var nameSuffix string
	if len(items) > 1 {
		// "name" part is present.
		nameSuffix = strings.TrimSpace(items[1])
		if nameSuffix == "" {
			err = errors.New("name part must be specified after " + typeAndNameSeparator + " in type/name key")
			return
		}
	} else {
		nameSuffix = ""
	}

	// Create normalized fullName.
	if nameSuffix == "" {
		fullName = string(typeStr)
	} else {
		fullName = string(typeStr) + typeAndNameSeparator + nameSuffix
	}

	err = nil
	return
}

func loadExtensions(v *viper.Viper, factories map[configmodels.Type]component.ExtensionFactory) (configmodels.Extensions, error) {
	// Get the list of all "extensions" sub vipers from config source.
	extensionsConfig := ViperSub(v, extensionsKeyName)
	expandEnvConfig(extensionsConfig)

	// Get the map of "extensions" sub-keys.
	keyMap := v.GetStringMap(extensionsKeyName)

	// Prepare resulting map.
	extensions := make(configmodels.Extensions)

	// Iterate over extensions and create a config for each.
	for key := range keyMap {
		// Decode the key into type and fullName components.
		typeStr, fullName, err := DecodeTypeAndName(key)
		if err != nil {
			return nil, &configError{
				code: errInvalidTypeAndNameKey,
				msg:  fmt.Sprintf("invalid key %q: %s", key, err.Error()),
			}
		}

		// Find extension factory based on "type" that we read from config source.
		factory := factories[typeStr]
		if factory == nil {
			return nil, &configError{
				code: errUnknownExtensionType,
				msg:  fmt.Sprintf("unknown extension type %q", typeStr),
			}
		}

		// Create the default config for this extension
		extensionCfg := factory.CreateDefaultConfig()
		extensionCfg.SetType(typeStr)
		extensionCfg.SetName(fullName)

		// Unmarshal only the subconfig for this exporter.
		componentConfig := ViperSub(extensionsConfig, key)

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		if err := componentConfig.UnmarshalExact(extensionCfg); err != nil {
			return nil, &configError{
				code: errUnmarshalErrorOnExtension,
				msg:  fmt.Sprintf("error reading settings for extension type %q: %v", typeStr, err),
			}
		}

		if extensions[fullName] != nil {
			return nil, &configError{
				code: errDuplicateExtensionName,
				msg:  fmt.Sprintf("duplicate extension name %q", fullName),
			}
		}

		extensions[fullName] = extensionCfg
	}

	return extensions, nil
}

func loadService(v *viper.Viper) (configmodels.Service, error) {
	var service configmodels.Service
	serviceSub := ViperSub(v, serviceKeyName)
	expandEnvConfig(serviceSub)

	// Process the pipelines first so in case of error on them it can be properly
	// reported.
	pipelines, err := loadPipelines(serviceSub)
	if err != nil {
		return service, err
	}

	// Do an exact match to find any unused section on config.
	if err := serviceSub.UnmarshalExact(&service); err != nil {
		return service, &configError{
			code: errUnmarshalErrorOnService,
			msg:  fmt.Sprintf("error reading settings for %q: %v", serviceKeyName, err),
		}
	}

	// Unmarshal cannot properly build Pipelines field, set it to the value
	// previously loaded.
	service.Pipelines = pipelines

	return service, nil
}

// LoadReceiver loads a receiver config from componentConfig using the provided factories.
func LoadReceiver(componentConfig *viper.Viper, typeStr configmodels.Type, fullName string, factory component.ReceiverFactoryBase) (configmodels.Receiver, error) {
	// Create the default config for this receiver.
	receiverCfg := factory.CreateDefaultConfig()
	receiverCfg.SetType(typeStr)
	receiverCfg.SetName(fullName)

	// Now that the default config struct is created we can Unmarshal into it
	// and it will apply user-defined config on top of the default.
	customUnmarshaler := factory.CustomUnmarshaler()
	var err error
	if customUnmarshaler != nil {
		// This configuration requires a custom unmarshaler, use it.
		err = customUnmarshaler(componentConfig, receiverCfg)
	} else {
		err = componentConfig.UnmarshalExact(receiverCfg)
	}

	if err != nil {
		return nil, &configError{
			code: errUnmarshalErrorOnReceiver,
			msg:  fmt.Sprintf("error reading settings for receiver type %q: %v", typeStr, err),
		}
	}

	return receiverCfg, nil
}

func loadReceivers(v *viper.Viper, factories map[configmodels.Type]component.ReceiverFactoryBase) (configmodels.Receivers, error) {
	// Get the list of all "receivers" sub vipers from config source.
	receiversConfig := ViperSub(v, receiversKeyName)
	expandEnvConfig(receiversConfig)

	// Get the map of "receivers" sub-keys.
	keyMap := v.GetStringMap(receiversKeyName)

	// Prepare resulting map
	receivers := make(configmodels.Receivers)

	// Iterate over input map and create a config for each.
	for key := range keyMap {
		// Decode the key into type and fullName components.
		typeStr, fullName, err := DecodeTypeAndName(key)
		if err != nil {
			return nil, &configError{
				code: errInvalidTypeAndNameKey,
				msg:  fmt.Sprintf("invalid key %q: %s", key, err.Error()),
			}
		}

		// Find receiver factory based on "type" that we read from config source
		factory := factories[typeStr]
		if factory == nil {
			return nil, &configError{
				code: errUnknownReceiverType,
				msg:  fmt.Sprintf("unknown receiver type %q", typeStr),
			}
		}

		receiverCfg, err := LoadReceiver(ViperSub(receiversConfig, key), typeStr, fullName, factory)

		if err != nil {
			// LoadReceiver already wraps the error.
			return nil, err
		}

		if receivers[receiverCfg.Name()] != nil {
			return nil, &configError{
				code: errDuplicateReceiverName,
				msg:  fmt.Sprintf("duplicate receiver name %q", receiverCfg.Name()),
			}
		}
		receivers[receiverCfg.Name()] = receiverCfg
	}

	return receivers, nil
}

func loadExporters(v *viper.Viper, factories map[configmodels.Type]component.ExporterFactoryBase) (configmodels.Exporters, error) {
	// Get the list of all "exporters" sub vipers from config source.
	exportersConfig := ViperSub(v, exportersKeyName)
	expandEnvConfig(exportersConfig)

	// Get the map of "exporters" sub-keys.
	keyMap := v.GetStringMap(exportersKeyName)

	// Prepare resulting map
	exporters := make(configmodels.Exporters)

	// Iterate over exporters and create a config for each.
	for key := range keyMap {
		// Decode the key into type and fullName components.
		typeStr, fullName, err := DecodeTypeAndName(key)
		if err != nil {
			return nil, &configError{
				code: errInvalidTypeAndNameKey,
				msg:  fmt.Sprintf("invalid key %q: %s", key, err.Error()),
			}
		}

		// Find exporter factory based on "type" that we read from config source
		factory := factories[typeStr]
		if factory == nil {
			return nil, &configError{
				code: errUnknownExporterType,
				msg:  fmt.Sprintf("unknown exporter type %q", typeStr),
			}
		}

		// Create the default config for this exporter
		exporterCfg := factory.CreateDefaultConfig()
		exporterCfg.SetType(typeStr)
		exporterCfg.SetName(fullName)

		// Unmarshal only the subconfig for this exporter.
		componentConfig := ViperSub(exportersConfig, key)

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		if err := componentConfig.UnmarshalExact(exporterCfg); err != nil {
			return nil, &configError{
				code: errUnmarshalErrorOnExporter,
				msg:  fmt.Sprintf("error reading settings for exporter type %q: %v", typeStr, err),
			}
		}

		if exporters[fullName] != nil {
			return nil, &configError{
				code: errDuplicateExporterName,
				msg:  fmt.Sprintf("duplicate exporter name %q", fullName),
			}
		}

		exporters[fullName] = exporterCfg
	}

	return exporters, nil
}

func loadProcessors(v *viper.Viper, factories map[configmodels.Type]component.ProcessorFactoryBase) (configmodels.Processors, error) {
	// Get the list of all "processors" sub vipers from config source.
	processorsConfig := ViperSub(v, processorsKeyName)
	expandEnvConfig(processorsConfig)

	// Get the map of "processors" sub-keys.
	keyMap := v.GetStringMap(processorsKeyName)

	// Prepare resulting map.
	processors := make(configmodels.Processors)

	// Iterate over processors and create a config for each.
	for key := range keyMap {
		// Decode the key into type and fullName components.
		typeStr, fullName, err := DecodeTypeAndName(key)
		if err != nil {
			return nil, &configError{
				code: errInvalidTypeAndNameKey,
				msg:  fmt.Sprintf("invalid key %q: %s", key, err.Error()),
			}
		}

		// Find processor factory based on "type" that we read from config source.
		factory := factories[typeStr]
		if factory == nil {
			return nil, &configError{
				code: errUnknownProcessorType,
				msg:  fmt.Sprintf("unknown processor type %q", typeStr),
			}
		}

		// Create the default config for this processors
		processorCfg := factory.CreateDefaultConfig()
		processorCfg.SetType(typeStr)
		processorCfg.SetName(fullName)

		// Unmarshal only the subconfig for this processor.
		componentConfig := ViperSub(processorsConfig, key)

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		if err := componentConfig.UnmarshalExact(processorCfg); err != nil {
			return nil, &configError{
				code: errUnmarshalErrorOnProcessor,
				msg:  fmt.Sprintf("error reading settings for processor type %q: %v", typeStr, err),
			}
		}

		if processors[fullName] != nil {
			return nil, &configError{
				code: errDuplicateProcessorName,
				msg:  fmt.Sprintf("duplicate processor name %q", fullName),
			}
		}

		processors[fullName] = processorCfg
	}

	return processors, nil
}

func loadPipelines(v *viper.Viper) (configmodels.Pipelines, error) {
	// Get the list of all "pipelines" sub vipers from config source.
	pipelinesConfig := ViperSub(v, pipelinesKeyName)

	// Get the map of "pipelines" sub-keys.
	keyMap := v.GetStringMap(pipelinesKeyName)

	// Prepare resulting map.
	pipelines := make(configmodels.Pipelines)

	// Iterate over input map and create a config for each.
	for key := range keyMap {
		// Decode the key into type and name components.
		typeStr, name, err := DecodeTypeAndName(key)
		if err != nil {
			return nil, &configError{
				code: errInvalidTypeAndNameKey,
				msg:  fmt.Sprintf("invalid key %q: %s", key, err.Error()),
			}
		}

		// Create the config for this pipeline.
		var pipelineCfg configmodels.Pipeline

		// Set the type.
		pipelineCfg.InputType = configmodels.DataType(typeStr)
		switch pipelineCfg.InputType {
		case configmodels.TracesDataType:
		case configmodels.MetricsDataType:
		case configmodels.LogsDataType:
		default:
			return nil, &configError{
				code: errInvalidPipelineType,
				msg:  fmt.Sprintf("invalid pipeline type %q (must be metrics or traces)", typeStr),
			}
		}

		pipelineConfig := ViperSub(pipelinesConfig, key)

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		if err := pipelineConfig.UnmarshalExact(&pipelineCfg); err != nil {
			return nil, &configError{
				code: errUnmarshalErrorOnPipeline,
				msg:  fmt.Sprintf("error reading settings for pipeline type %q: %v", typeStr, err),
			}
		}

		pipelineCfg.Name = name

		if pipelines[name] != nil {
			return nil, &configError{
				code: errDuplicatePipelineName,
				msg:  fmt.Sprintf("duplicate pipeline name %q", name),
			}
		}

		pipelines[name] = &pipelineCfg
	}

	return pipelines, nil
}

// ValidateConfig validates config.
func ValidateConfig(cfg *configmodels.Config, logger *zap.Logger) error {
	// This function performs basic validation of configuration. There may be more subtle
	// invalid cases that we currently don't check for but which we may want to add in
	// the future (e.g. disallowing receiving and exporting on the same endpoint).

	if err := validateReceivers(cfg); err != nil {
		return err
	}

	if err := validateExporters(cfg); err != nil {
		return err
	}

	if err := validateService(cfg); err != nil {
		return err
	}

	return nil
}

func validateService(cfg *configmodels.Config) error {
	if err := validatePipelines(cfg); err != nil {
		return err
	}

	return validateServiceExtensions(cfg)
}

func validateServiceExtensions(cfg *configmodels.Config) error {
	if len(cfg.Service.Extensions) == 0 {
		return nil
	}

	// Validate extensions.
	for _, ref := range cfg.Service.Extensions {
		// Check that the name referenced in the service extensions exists in the top-level extensions
		if cfg.Extensions[ref] == nil {
			return &configError{
				code: errExtensionNotExists,
				msg:  fmt.Sprintf("service references extension %q which does not exists", ref),
			}
		}
	}

	return nil
}

func validatePipelines(cfg *configmodels.Config) error {
	// Must have at least one pipeline.
	if len(cfg.Service.Pipelines) == 0 {
		return &configError{code: errMissingPipelines, msg: "must have at least one pipeline"}
	}

	// Validate pipelines.
	for _, pipeline := range cfg.Service.Pipelines {
		if err := validatePipeline(cfg, pipeline); err != nil {
			return err
		}
	}
	return nil
}

func validatePipeline(cfg *configmodels.Config, pipeline *configmodels.Pipeline) error {
	if err := validatePipelineReceivers(cfg, pipeline); err != nil {
		return err
	}

	if err := validatePipelineExporters(cfg, pipeline); err != nil {
		return err
	}

	if err := validatePipelineProcessors(cfg, pipeline); err != nil {
		return err
	}

	return nil
}

func validatePipelineReceivers(cfg *configmodels.Config, pipeline *configmodels.Pipeline) error {
	if len(pipeline.Receivers) == 0 {
		return &configError{
			code: errPipelineMustHaveReceiver,
			msg:  fmt.Sprintf("pipeline %q must have at least one receiver", pipeline.Name),
		}
	}

	// Validate pipeline receiver name references.
	for _, ref := range pipeline.Receivers {
		// Check that the name referenced in the pipeline's Receivers exists in the top-level Receivers
		if cfg.Receivers[ref] == nil {
			return &configError{
				code: errPipelineReceiverNotExists,
				msg:  fmt.Sprintf("pipeline %q references receiver %q which does not exists", pipeline.Name, ref),
			}
		}
	}

	return nil
}

func validatePipelineExporters(cfg *configmodels.Config, pipeline *configmodels.Pipeline) error {
	if len(pipeline.Exporters) == 0 {
		return &configError{
			code: errPipelineMustHaveExporter,
			msg:  fmt.Sprintf("pipeline %q must have at least one exporter", pipeline.Name),
		}
	}

	// Validate pipeline exporter name references.
	for _, ref := range pipeline.Exporters {
		// Check that the name referenced in the pipeline's Exporters exists in the top-level Exporters
		if cfg.Exporters[ref] == nil {
			return &configError{
				code: errPipelineExporterNotExists,
				msg:  fmt.Sprintf("pipeline %q references exporter %q which does not exists", pipeline.Name, ref),
			}
		}
	}

	return nil
}

func validatePipelineProcessors(cfg *configmodels.Config, pipeline *configmodels.Pipeline) error {
	if len(pipeline.Processors) == 0 {
		return nil
	}

	// Validate pipeline processor name references
	for _, ref := range pipeline.Processors {
		// Check that the name referenced in the pipeline's processors exists in the top-level processors.
		if cfg.Processors[ref] == nil {
			return &configError{
				code: errPipelineProcessorNotExists,
				msg:  fmt.Sprintf("pipeline %q references processor %s which does not exists", pipeline.Name, ref),
			}
		}
	}

	return nil
}

func validateReceivers(cfg *configmodels.Config) error {
	// Currently there is no default receiver enabled. The configuration must specify at least one enabled receiver to
	// be valid.
	if len(cfg.Receivers) == 0 {
		return &configError{
			code: errMissingReceivers,
			msg:  "no enabled receivers specified in config",
		}
	}
	return nil
}

func validateExporters(cfg *configmodels.Config) error {
	// There must be at least one enabled exporter to be considered a valid configuration.
	if len(cfg.Exporters) == 0 {
		return &configError{
			code: errMissingExporters,
			msg:  "no enabled exporters specified in config",
		}
	}
	return nil
}

// expandEnvConfig creates a new viper config with expanded values for all the values (simple, list or map value).
// It does not expand the keys.
func expandEnvConfig(v *viper.Viper) {
	for _, k := range v.AllKeys() {
		v.Set(k, expandStringValues(v.Get(k)))
	}
}

func expandStringValues(value interface{}) interface{} {
	switch v := value.(type) {
	default:
		return v
	case string:
		return os.ExpandEnv(v)
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

// Copied from the Viper but changed to use the same delimiter.
// See https://github.com/spf13/viper/issues/871
func ViperSub(v *viper.Viper, key string) *viper.Viper {
	subv := NewViper()
	data := v.Get(key)
	if data == nil {
		return subv
	}

	if reflect.TypeOf(data).Kind() == reflect.Map {
		subv.MergeConfigMap(cast.ToStringMap(data))
		return subv
	}
	return subv
}
