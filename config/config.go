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
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/exporter"
	"github.com/open-telemetry/opentelemetry-collector/extension"
	"github.com/open-telemetry/opentelemetry-collector/processor"
	"github.com/open-telemetry/opentelemetry-collector/receiver"
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
	errPipelineMustHaveProcessors
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
	Receivers map[string]receiver.Factory

	// Processors maps processor type names in the config to the respective factory.
	Processors map[string]processor.Factory

	// Exporters maps exporter type names in the config to the respective factory.
	Exporters map[string]exporter.Factory

	// Extensions maps extension type names in the config to the respective factory.
	Extensions map[string]extension.Factory
}

// Load loads a Config from Viper.
func Load(
	v *viper.Viper,
	factories Factories,
	logger *zap.Logger,
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

	// Config is loaded. Now validate it.

	if err := validateConfig(&config, logger); err != nil {
		return nil, err
	}

	return &config, nil
}

// decodeTypeAndName decodes a key in type[/name] format into type and fullName.
// fullName is the key normalized such that type and name components have spaces trimmed.
// The "type" part must be present, the forward slash and "name" are optional.
func decodeTypeAndName(key string) (typeStr, fullName string, err error) {
	items := strings.SplitN(key, typeAndNameSeparator, 2)

	if len(items) >= 1 {
		typeStr = strings.TrimSpace(items[0])
	}

	if len(items) < 1 || typeStr == "" {
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
		fullName = typeStr
	} else {
		fullName = typeStr + typeAndNameSeparator + nameSuffix
	}

	err = nil
	return
}

func loadExtensions(v *viper.Viper, factories map[string]extension.Factory) (configmodels.Extensions, error) {
	// Get the list of all "extensions" sub vipers from config source.
	subViper := v.Sub(extensionsKeyName)

	// Get the map of "extensions" sub-keys.
	keyMap := v.GetStringMap(extensionsKeyName)

	// Prepare resulting map.
	extensions := make(configmodels.Extensions)

	// Iterate over extensions and create a config for each.
	for key := range keyMap {
		// Decode the key into type and fullName components.
		typeStr, fullName, err := decodeTypeAndName(key)
		if err != nil || typeStr == "" {
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
		sv := getConfigSection(subViper, key)

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		if err := sv.UnmarshalExact(extensionCfg); err != nil {
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
	serviceSub := getConfigSection(v, serviceKeyName)

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

func loadReceivers(v *viper.Viper, factories map[string]receiver.Factory) (configmodels.Receivers, error) {
	// Get the list of all "receivers" sub vipers from config source.
	subViper := v.Sub(receiversKeyName)

	// Get the map of "receivers" sub-keys.
	keyMap := v.GetStringMap(receiversKeyName)

	// Currently there is no default receiver enabled. The configuration must specify at least one receiver to enable
	// functionality.
	if len(keyMap) == 0 {
		return nil, &configError{
			code: errMissingReceivers,
			msg:  "no receivers specified in config",
		}
	}

	// Prepare resulting map
	receivers := make(configmodels.Receivers)

	// Iterate over input map and create a config for each.
	for key := range keyMap {
		// Decode the key into type and fullName components.
		typeStr, fullName, err := decodeTypeAndName(key)
		if err != nil || typeStr == "" {
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

		// Create the default config for this receiver.
		receiverCfg := factory.CreateDefaultConfig()
		receiverCfg.SetType(typeStr)
		receiverCfg.SetName(fullName)

		// Unmarshal only the subconfig for this exporter.
		sv := getConfigSection(subViper, key)

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		customUnmarshaler := factory.CustomUnmarshaler()
		if customUnmarshaler != nil {
			// This configuration requires a custom unmarshaler, use it.
			err = customUnmarshaler(subViper, key, sv, receiverCfg)
		} else {
			err = sv.UnmarshalExact(receiverCfg)
		}

		if err != nil {
			return nil, &configError{
				code: errUnmarshalErrorOnReceiver,
				msg:  fmt.Sprintf("error reading settings for receiver type %q: %v", typeStr, err),
			}
		}

		if receivers[fullName] != nil {
			return nil, &configError{
				code: errDuplicateReceiverName,
				msg:  fmt.Sprintf("duplicate receiver name %q", fullName),
			}
		}
		receivers[fullName] = receiverCfg
	}

	return receivers, nil
}

func loadExporters(v *viper.Viper, factories map[string]exporter.Factory) (configmodels.Exporters, error) {
	// Get the list of all "exporters" sub vipers from config source.
	subViper := v.Sub(exportersKeyName)

	// Get the map of "exporters" sub-keys.
	keyMap := v.GetStringMap(exportersKeyName)

	// There is no default exporter. The configuration must specify at least one exporter to enable functionality.
	if len(keyMap) == 0 {
		return nil, &configError{
			code: errMissingExporters,
			msg:  "no exporters specified in config",
		}
	}

	// Prepare resulting map
	exporters := make(configmodels.Exporters)

	// Iterate over exporters and create a config for each.
	for key := range keyMap {
		// Decode the key into type and fullName components.
		typeStr, fullName, err := decodeTypeAndName(key)
		if err != nil || typeStr == "" {
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
		sv := getConfigSection(subViper, key)

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		if err := sv.UnmarshalExact(exporterCfg); err != nil {
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

func loadProcessors(v *viper.Viper, factories map[string]processor.Factory) (configmodels.Processors, error) {
	// Get the list of all "processors" sub vipers from config source.
	subViper := v.Sub(processorsKeyName)

	// Get the map of "processors" sub-keys.
	keyMap := v.GetStringMap(processorsKeyName)

	// Prepare resulting map.
	processors := make(configmodels.Processors)

	// Iterate over processors and create a config for each.
	for key := range keyMap {
		// Decode the key into type and fullName components.
		typeStr, fullName, err := decodeTypeAndName(key)
		if err != nil || typeStr == "" {
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

		// Unmarshal only the subconfig for this exporter.
		sv := getConfigSection(subViper, key)

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		if err := sv.UnmarshalExact(processorCfg); err != nil {
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
	subViper := v.Sub(pipelinesKeyName)

	// Get the map of "pipelines" sub-keys.
	keyMap := v.GetStringMap(pipelinesKeyName)

	// Prepare resulting map.
	pipelines := make(configmodels.Pipelines)

	// Iterate over input map and create a config for each.
	for key := range keyMap {
		// Decode the key into type and name components.
		typeStr, name, err := decodeTypeAndName(key)
		if err != nil || typeStr == "" {
			return nil, &configError{
				code: errInvalidTypeAndNameKey,
				msg:  fmt.Sprintf("invalid key %q: %s", key, err.Error()),
			}
		}

		// Create the config for this pipeline.
		var pipelineCfg configmodels.Pipeline

		// Set the type.
		switch typeStr {
		case configmodels.TracesDataTypeStr:
			pipelineCfg.InputType = configmodels.TracesDataType
		case configmodels.MetricsDataTypeStr:
			pipelineCfg.InputType = configmodels.MetricsDataType
		default:
			return nil, &configError{
				code: errInvalidPipelineType,
				msg:  fmt.Sprintf("invalid pipeline type %q (must be metrics or traces)", typeStr),
			}
		}

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		if err := subViper.UnmarshalKey(key, &pipelineCfg, errorOnUnused); err != nil {
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

func validateConfig(cfg *configmodels.Config, logger *zap.Logger) error {
	// This function performs basic validation of configuration. There may be more subtle
	// invalid cases that we currently don't check for but which we may want to add in
	// the future (e.g. disallowing receiving and exporting on the same endpoint).

	if err := validateService(cfg, logger); err != nil {
		return err
	}

	if err := validatePipelines(cfg, logger); err != nil {
		return err
	}

	if err := validateReceivers(cfg); err != nil {
		return err
	}
	if err := validateExporters(cfg); err != nil {
		return err
	}
	validateProcessors(cfg)

	return nil
}

func validateService(cfg *configmodels.Config, logger *zap.Logger) error {
	// Currently only to validate extensions.
	return validateServiceExtensions(cfg, &cfg.Service, logger)
}

func validateServiceExtensions(
	cfg *configmodels.Config,
	service *configmodels.Service,
	logger *zap.Logger,
) error {
	if len(cfg.Service.Extensions) < 1 {
		return nil
	}

	// Validate extensions.
	for _, ref := range service.Extensions {
		// Check that the name referenced in the service extensions exists in the top-level extensions
		if cfg.Extensions[ref] == nil {
			return &configError{
				code: errExtensionNotExists,
				msg:  fmt.Sprintf("service references extension %q which does not exists", ref),
			}
		}
	}

	// Remove disabled extensions.
	extensions := service.Extensions[:0]
	for _, ref := range service.Extensions {
		ext := cfg.Extensions[ref]
		if ext.IsEnabled() {
			// The extension is enabled. Keep it in the pipeline.
			extensions = append(extensions, ref)
		} else {
			logger.Info("service references a disabled extension. Ignoring the extension.",
				zap.String("extension", ref))
		}
	}

	service.Extensions = extensions

	return nil
}

func validatePipelines(cfg *configmodels.Config, logger *zap.Logger) error {
	// Must have at least one pipeline.
	if len(cfg.Service.Pipelines) < 1 {
		return &configError{code: errMissingPipelines, msg: "must have at least one pipeline"}
	}

	// Validate pipelines.
	for _, pipeline := range cfg.Service.Pipelines {
		if err := validatePipeline(cfg, pipeline, logger); err != nil {
			return err
		}
	}
	return nil
}

func validatePipeline(
	cfg *configmodels.Config,
	pipeline *configmodels.Pipeline,
	logger *zap.Logger,
) error {
	if err := validatePipelineReceivers(cfg, pipeline, logger); err != nil {
		return err
	}

	if err := validatePipelineExporters(cfg, pipeline, logger); err != nil {
		return err
	}

	if err := validatePipelineProcessors(cfg, pipeline, logger); err != nil {
		return err
	}

	return nil
}

func validatePipelineReceivers(
	cfg *configmodels.Config,
	pipeline *configmodels.Pipeline,
	logger *zap.Logger,
) error {
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

	// Remove disabled receivers.
	rs := pipeline.Receivers[:0]
	for _, ref := range pipeline.Receivers {
		rcv := cfg.Receivers[ref]
		if rcv.IsEnabled() {
			// The receiver is enabled. Keep it in the pipeline.
			rs = append(rs, ref)
		} else {
			logger.Info("pipeline references a disabled receiver. Ignoring the receiver.",
				zap.String("pipeline", pipeline.Name),
				zap.String("receiver", ref))
		}
	}

	pipeline.Receivers = rs

	return nil
}

func validatePipelineExporters(
	cfg *configmodels.Config,
	pipeline *configmodels.Pipeline,
	logger *zap.Logger,
) error {
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

	// Remove disabled exporters.
	rs := pipeline.Exporters[:0]
	for _, ref := range pipeline.Exporters {
		exp := cfg.Exporters[ref]
		if exp.IsEnabled() {
			// The exporter is enabled. Keep it in the pipeline.
			rs = append(rs, ref)
		} else {
			logger.Info("pipeline references a disabled exporter. Ignoring the exporter.",
				zap.String("pipeline", pipeline.Name),
				zap.String("exporter", ref))
		}
	}
	pipeline.Exporters = rs

	return nil
}

func validatePipelineProcessors(
	cfg *configmodels.Config,
	pipeline *configmodels.Pipeline,
	logger *zap.Logger,
) error {
	if pipeline.InputType == configmodels.TracesDataType {
		// Traces pipeline must have at least one processor.
		if len(pipeline.Processors) == 0 {
			return &configError{
				code: errPipelineMustHaveProcessors,
				msg:  fmt.Sprintf("pipeline %q must have at least one processor", pipeline.Name),
			}
		}
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

	// Remove disabled processors.
	rs := pipeline.Processors[:0]
	for _, ref := range pipeline.Processors {
		proc := cfg.Processors[ref]
		if proc.IsEnabled() {
			// The processor is enabled. Keep it in the pipeline.
			rs = append(rs, ref)
		} else {
			logger.Info("pipeline references a disabled processor. Ignoring the processor.",
				zap.String("pipeline", pipeline.Name),
				zap.String("processor", ref))
		}
	}
	pipeline.Processors = rs

	return nil
}

func validateReceivers(cfg *configmodels.Config) error {
	// Remove disabled receivers.
	for name, rcv := range cfg.Receivers {
		if !rcv.IsEnabled() {
			delete(cfg.Receivers, name)
		}
	}

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
	// Remove disabled exporters.
	for name, rcv := range cfg.Exporters {
		if !rcv.IsEnabled() {
			delete(cfg.Exporters, name)
		}
	}

	// There must be at least one enabled exporter to be considered a valid configuration.
	if len(cfg.Exporters) == 0 {
		return &configError{
			code: errMissingExporters,
			msg:  "no enabled exporters specified in config",
		}
	}
	return nil
}

func validateProcessors(cfg *configmodels.Config) {
	// Remove disabled processors.
	for name, rcv := range cfg.Processors {
		if !rcv.IsEnabled() {
			delete(cfg.Processors, name)
		}
	}
}

// getConfigSection returns a sub-config from the viper config that has the corresponding given key.
// It also expands all the string values.
func getConfigSection(v *viper.Viper, key string) *viper.Viper {
	// Unmarshal only the subconfig for this processor.
	sv := v.Sub(key)
	if sv == nil {
		// When the config for this key is empty Sub returns nil. In order to avoid nil checks
		// just return an empty config.
		return viper.New()
	}
	// Before unmarshaling first expand all environment variables.
	return expandEnvConfig(sv)
}

// expandEnvConfig creates a new viper config with expanded values for all the values (simple, list or map value).
// It does not expand the keys.
// Need to copy everything because of a bug in Viper: Set a value "map[string]interface{}" where a key has a ".",
// then AllSettings will return the previous value not the newly set one.
func expandEnvConfig(v *viper.Viper) *viper.Viper {
	newCfg := make(map[string]interface{})
	for k, val := range v.AllSettings() {
		newCfg[k] = expandStringValues(val)
	}
	newVip := viper.New()
	newVip.MergeConfigMap(newCfg)
	return newVip
}

func expandStringValues(value interface{}) interface{} {
	switch v := value.(type) {
	default:
		return v
	case string:
		return os.ExpandEnv(v)
	case []interface{}:
		// Viper treats all the slices as []interface{} (at least in what the otelcol tests).
		nslice := make([]interface{}, 0, len(v))
		for _, vint := range v {
			nslice = append(nslice, expandStringValues(vint))
		}
		return nslice
	case map[string]interface{}:
		nmap := make(map[string]interface{}, len(v))
		// Viper treats all the maps as [string]interface{} (at least in what the otelcol tests).
		for k, vint := range v {
			nmap[k] = expandStringValues(vint)
		}
		return nmap
	}
}

// errorOnUnused sets the decoder configuration to error in case of unused sections
// are present in the configuration.
func errorOnUnused(decoderCfg *mapstructure.DecoderConfig) {
	// If ErrorUnused is true, then it is an error for there to exist
	// keys in the original map that were unused in the decoding process
	// (extra keys).
	decoderCfg.ErrorUnused = true
}
