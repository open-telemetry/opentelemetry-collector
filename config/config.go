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
	errInvalidSubConfig
	errInvalidTypeAndNameKey
	errUnknownType
	errDuplicateName
	errMissingPipelines
	errPipelineMustHaveReceiver
	errPipelineMustHaveExporter
	errExtensionNotExists
	errPipelineReceiverNotExists
	errPipelineProcessorNotExists
	errPipelineExporterNotExists
	errMissingReceivers
	errMissingExporters
	errUnmarshalError
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

// deprecatedUnmarshaler is the old/deprecated way to provide custom unmarshaler.
type deprecatedUnmarshaler interface {
	// CustomUnmarshaler returns a custom unmarshaler for the configuration or nil if
	// there is no need for custom unmarshaling. This is typically used if viper.UnmarshalExact()
	// is not sufficient to unmarshal correctly.
	CustomUnmarshaler() component.CustomUnmarshaler
}

// typeAndNameSeparator is the separator that is used between type and name in type/name composite keys.
const typeAndNameSeparator = "/"

// Creates a new Viper instance with a different key-delimitor "::" instead of the
// default ".". This way configs can have keys that contain ".".
func NewViper() *viper.Viper {
	return viper.NewWithOptions(viper.KeyDelimiter("::"))
}

// Load loads a Config from Viper.
// After loading the config, need to check if it is valid by calling `ValidateConfig`.
func Load(
	v *viper.Viper,
	factories component.Factories,
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
			code: errUnmarshalError,
			msg:  fmt.Sprintf("error reading top level configuration sections: %s", err.Error()),
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

func errorInvalidTypeAndNameKey(component, key string, err error) error {
	return &configError{
		code: errInvalidTypeAndNameKey,
		msg:  fmt.Sprintf("invalid %s type and name key %q: %v", component, key, err),
	}
}

func errorUnknownType(component string, typeStr configmodels.Type, fullName string) error {
	return &configError{
		code: errUnknownType,
		msg:  fmt.Sprintf("unknown %s type %q for %s", component, typeStr, fullName),
	}
}

func errorUnmarshalError(component string, fullName string, err error) error {
	return &configError{
		code: errUnmarshalError,
		msg:  fmt.Sprintf("error reading %s configuration for %s: %v", component, fullName, err),
	}
}

func errorDuplicateName(component string, fullName string) error {
	return &configError{
		code: errDuplicateName,
		msg:  fmt.Sprintf("duplicate %s name %s", component, fullName),
	}
}

func loadExtensions(v *viper.Viper, factories map[configmodels.Type]component.ExtensionFactory) (configmodels.Extensions, error) {
	// Get the list of all "receivers" sub vipers from config source.
	extensionsConfig, err := ViperSubExact(v, extensionsKeyName)
	if err != nil {
		return nil, err
	}
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
			return nil, errorInvalidTypeAndNameKey(extensionsKeyName, key, err)
		}

		// Find extension factory based on "type" that we read from config source.
		factory := factories[typeStr]
		if factory == nil {
			return nil, errorUnknownType(extensionsKeyName, typeStr, fullName)
		}

		// Create the default config for this extension
		extensionCfg := factory.CreateDefaultConfig()
		extensionCfg.SetName(fullName)

		// Unmarshal only the subconfig for this exporter.
		componentConfig, err := ViperSubExact(extensionsConfig, key)
		if err != nil {
			return nil, err
		}

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		unm := unmarshaler(factory)
		if err := unm(componentConfig, extensionCfg); err != nil {
			return nil, errorUnmarshalError(extensionsKeyName, fullName, err)
		}

		if extensions[fullName] != nil {
			return nil, errorDuplicateName(extensionsKeyName, fullName)
		}

		extensions[fullName] = extensionCfg
	}

	return extensions, nil
}

func loadService(v *viper.Viper) (configmodels.Service, error) {
	var service configmodels.Service
	serviceSub, err := ViperSubExact(v, serviceKeyName)
	if err != nil {
		return service, err
	}
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
			code: errUnmarshalError,
			msg:  fmt.Sprintf("error reading service configuration: %v", err),
		}
	}

	// Unmarshal cannot properly build Pipelines field, set it to the value
	// previously loaded.
	service.Pipelines = pipelines

	return service, nil
}

// LoadReceiver loads a receiver config from componentConfig using the provided factories.
func LoadReceiver(componentConfig *viper.Viper, typeStr configmodels.Type, fullName string, factory component.ReceiverFactory) (configmodels.Receiver, error) {
	// Create the default config for this receiver.
	receiverCfg := factory.CreateDefaultConfig()
	receiverCfg.SetName(fullName)

	// Now that the default config struct is created we can Unmarshal into it
	// and it will apply user-defined config on top of the default.
	unm := unmarshaler(factory)
	if err := unm(componentConfig, receiverCfg); err != nil {
		return nil, errorUnmarshalError(receiversKeyName, fullName, err)
	}

	return receiverCfg, nil
}

func loadReceivers(v *viper.Viper, factories map[configmodels.Type]component.ReceiverFactory) (configmodels.Receivers, error) {
	// Get the list of all "receivers" sub vipers from config source.
	receiversConfig, err := ViperSubExact(v, receiversKeyName)
	if err != nil {
		return nil, err
	}
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
			return nil, errorInvalidTypeAndNameKey(receiversKeyName, key, err)
		}

		// Find receiver factory based on "type" that we read from config source
		factory := factories[typeStr]
		if factory == nil {
			return nil, errorUnknownType(receiversKeyName, typeStr, fullName)
		}

		// Unmarshal only the subconfig for this exporter.
		componentConfig, err := ViperSubExact(receiversConfig, key)
		if err != nil {
			return nil, err
		}

		receiverCfg, err := LoadReceiver(componentConfig, typeStr, fullName, factory)

		if err != nil {
			// LoadReceiver already wraps the error.
			return nil, err
		}

		if receivers[receiverCfg.Name()] != nil {
			return nil, errorDuplicateName(receiversKeyName, fullName)
		}
		receivers[receiverCfg.Name()] = receiverCfg
	}

	return receivers, nil
}

func loadExporters(v *viper.Viper, factories map[configmodels.Type]component.ExporterFactory) (configmodels.Exporters, error) {
	// Get the list of all "exporters" sub vipers from config source.
	exportersConfig, err := ViperSubExact(v, exportersKeyName)
	if err != nil {
		return nil, err
	}
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
			return nil, errorInvalidTypeAndNameKey(exportersKeyName, key, err)
		}

		// Find exporter factory based on "type" that we read from config source
		factory := factories[typeStr]
		if factory == nil {
			return nil, errorUnknownType(exportersKeyName, typeStr, fullName)
		}

		// Create the default config for this exporter
		exporterCfg := factory.CreateDefaultConfig()
		exporterCfg.SetName(fullName)

		// Unmarshal only the subconfig for this exporter.
		componentConfig, err := ViperSubExact(exportersConfig, key)
		if err != nil {
			return nil, err
		}

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		unm := unmarshaler(factory)
		if err := unm(componentConfig, exporterCfg); err != nil {
			return nil, errorUnmarshalError(exportersKeyName, fullName, err)
		}

		if exporters[fullName] != nil {
			return nil, errorDuplicateName(exportersKeyName, fullName)
		}

		exporters[fullName] = exporterCfg
	}

	return exporters, nil
}

func loadProcessors(v *viper.Viper, factories map[configmodels.Type]component.ProcessorFactory) (configmodels.Processors, error) {
	// Get the list of all "processors" sub vipers from config source.
	processorsConfig, err := ViperSubExact(v, processorsKeyName)
	if err != nil {
		return nil, err
	}
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
			return nil, errorInvalidTypeAndNameKey(processorsKeyName, key, err)
		}

		// Find processor factory based on "type" that we read from config source.
		factory := factories[typeStr]
		if factory == nil {
			return nil, errorUnknownType(processorsKeyName, typeStr, fullName)
		}

		// Create the default config for this processors
		processorCfg := factory.CreateDefaultConfig()
		processorCfg.SetName(fullName)

		// Unmarshal only the subconfig for this processor.
		componentConfig, vsErr := ViperSubExact(processorsConfig, key)
		if vsErr != nil {
			return nil, vsErr
		}

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		unm := unmarshaler(factory)
		if err := unm(componentConfig, processorCfg); err != nil {
			return nil, errorUnmarshalError(processorsKeyName, fullName, err)
		}

		if processors[fullName] != nil {
			return nil, errorDuplicateName(processorsKeyName, fullName)
		}

		processors[fullName] = processorCfg
	}

	return processors, nil
}

func loadPipelines(v *viper.Viper) (configmodels.Pipelines, error) {
	// Get the list of all "pipelines" sub vipers from config source.
	pipelinesConfig, err := ViperSubExact(v, pipelinesKeyName)
	if err != nil {
		return nil, err
	}

	// Get the map of "pipelines" sub-keys.
	keyMap := v.GetStringMap(pipelinesKeyName)

	// Prepare resulting map.
	pipelines := make(configmodels.Pipelines)

	// Iterate over input map and create a config for each.
	for key := range keyMap {
		// Decode the key into type and name components.
		typeStr, fullName, err := DecodeTypeAndName(key)
		if err != nil {
			return nil, errorInvalidTypeAndNameKey(pipelinesKeyName, key, err)
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
			return nil, errorUnknownType(pipelinesKeyName, typeStr, fullName)
		}

		pipelineConfig, err := ViperSubExact(pipelinesConfig, key)
		if err != nil {
			return nil, err
		}

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		if err := pipelineConfig.UnmarshalExact(&pipelineCfg); err != nil {
			return nil, errorUnmarshalError(pipelinesKeyName, fullName, err)
		}

		pipelineCfg.Name = fullName

		if pipelines[fullName] != nil {
			return nil, errorDuplicateName(pipelinesKeyName, fullName)
		}

		pipelines[fullName] = &pipelineCfg
	}

	return pipelines, nil
}

// ValidateConfig validates config.
func ValidateConfig(cfg *configmodels.Config, _ *zap.Logger) error {
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
				msg:  fmt.Sprintf("service references extension %q which does not exist", ref),
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
				msg:  fmt.Sprintf("pipeline %q references receiver %q which does not exist", pipeline.Name, ref),
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
				msg:  fmt.Sprintf("pipeline %q references exporter %q which does not exist", pipeline.Name, ref),
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
				msg:  fmt.Sprintf("pipeline %q references processor %s which does not exist", pipeline.Name, ref),
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

func unmarshaler(factory component.Factory) component.CustomUnmarshaler {
	if fu, ok := factory.(component.ConfigUnmarshaler); ok {
		return fu.Unmarshal
	}

	if du, ok := factory.(deprecatedUnmarshaler); ok {
		cu := du.CustomUnmarshaler()
		if cu != nil {
			return cu
		}
	}

	return defaultUnmarshaler
}

func defaultUnmarshaler(componentViperSection *viper.Viper, intoCfg interface{}) error {
	return componentViperSection.UnmarshalExact(intoCfg)
}

// Copied from the Viper but changed to use the same delimiter
// and return error if the sub is not a map.
// See https://github.com/spf13/viper/issues/871
func ViperSubExact(v *viper.Viper, key string) (*viper.Viper, error) {
	data := v.Get(key)
	if data == nil {
		return NewViper(), nil
	}

	if reflect.TypeOf(data).Kind() == reflect.Map {
		subv := NewViper()
		// Cannot return error because the subv is empty.
		_ = subv.MergeConfigMap(cast.ToStringMap(data))
		return subv, nil
	}
	return nil, &configError{
		code: errInvalidSubConfig,
		msg:  fmt.Sprintf("unexpected sub-config value kind for key:%s value:%v kind:%v)", key, data, reflect.TypeOf(data).Kind()),
	}
}
