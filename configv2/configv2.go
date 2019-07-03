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

// Package configv2 implements loading of configuration V2 from Viper configuration.
// The implementation relies on registered factories that allow creating
// default configuration for each type of receiver/exporter/processor.
package configv2

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"

	"github.com/open-telemetry/opentelemetry-service/configv2/configmodels"
	"github.com/open-telemetry/opentelemetry-service/exporter"
	"github.com/open-telemetry/opentelemetry-service/processor"
	"github.com/open-telemetry/opentelemetry-service/receiver"
)

// These are errors that can be returned by Load(). Note that error codes are not part
// of Load()'s public API, they are for internal unit testing only.
type configErrorCode int

const (
	_ configErrorCode = iota // skip 0, start errors codes from 1.
	errInvalidTypeAndNameKey
	errUnknownReceiverType
	errUnknownExporterType
	errUnknownProcessorType
	errInvalidPipelineType
	errDuplicateReceiverName
	errDuplicateExporterName
	errDuplicateProcessorName
	errDuplicatePipelineName
	errMissingPipelines
	errPipelineMustHaveReceiver
	errPipelineMustHaveExporter
	errPipelineMustHaveProcessors
	errPipelineReceiverNotExists
	errPipelineProcessorNotExists
	errPipelineExporterNotExists
	errMetricPipelineCannotHaveProcessors
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

// Load loads a ConfigV2 from Viper.
func Load(v *viper.Viper) (*configmodels.ConfigV2, error) {

	var config configmodels.ConfigV2

	// Load the config.

	receivers, err := loadReceivers(v)
	if err != nil {
		return nil, err
	}
	config.Receivers = receivers

	exporters, err := loadExporters(v)
	if err != nil {
		return nil, err
	}
	config.Exporters = exporters

	processors, err := loadProcessors(v)
	if err != nil {
		return nil, err
	}
	config.Processors = processors

	pipelines, err := loadPipelines(v)
	if err != nil {
		return nil, err
	}
	config.Pipelines = pipelines

	// Config is loaded. Now validate it.

	if err := validateConfig(&config); err != nil {
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

func loadReceivers(v *viper.Viper) (configmodels.Receivers, error) {
	// Get the list of all "receivers" sub vipers from config source.
	subViper := v.Sub(receiversKeyName)

	// Get the map of "receivers" sub-keys.
	keyMap := v.GetStringMap(receiversKeyName)

	// Prepare resulting map
	receivers := make(configmodels.Receivers, 0)

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
		factory := receiver.GetFactory(typeStr)
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

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		customUnmarshaler := factory.CustomUnmarshaler()
		if customUnmarshaler != nil {
			// This configuration requires a custom unmarshaler, use it.
			err = customUnmarshaler(subViper, key, receiverCfg)
		} else {
			// Standard viper unmarshaler is fine.
			err = subViper.UnmarshalKey(key, receiverCfg)
		}

		if err != nil {
			return nil, &configError{
				code: errUnmarshalError,
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

func loadExporters(v *viper.Viper) (configmodels.Exporters, error) {
	// Get the list of all "exporters" sub vipers from config source.
	subViper := v.Sub(exportersKeyName)

	// Get the map of "exporters" sub-keys.
	keyMap := v.GetStringMap(exportersKeyName)

	// Prepare resulting map
	exporters := make(configmodels.Exporters, 0)

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
		factory := exporter.GetFactory(typeStr)
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

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		if err := subViper.UnmarshalKey(key, exporterCfg); err != nil {
			return nil, &configError{
				code: errUnmarshalError,
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

func loadProcessors(v *viper.Viper) (configmodels.Processors, error) {
	// Get the list of all "processors" sub vipers from config source.
	subViper := v.Sub(processorsKeyName)

	// Get the map of "processors" sub-keys.
	keyMap := v.GetStringMap(processorsKeyName)

	// Prepare resulting map.
	processors := make(configmodels.Processors, 0)

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
		factory := processor.GetFactory(typeStr)
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

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		if err := subViper.UnmarshalKey(key, processorCfg); err != nil {
			return nil, &configError{
				code: errUnmarshalError,
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
	pipelines := make(configmodels.Pipelines, 0)

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
		if err := subViper.UnmarshalKey(key, &pipelineCfg); err != nil {
			return nil, &configError{
				code: errUnmarshalError,
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

func validateConfig(cfg *configmodels.ConfigV2) error {
	// This function performs basic validation of configuration. There may be more subtle
	// invalid cases that we currently don't check for but which we may want to add in
	// the future (e.g. disallowing receiving and exporting on the same endpoint).

	if err := validatePipelines(cfg); err != nil {
		return err
	}

	return nil
}

func validatePipelines(cfg *configmodels.ConfigV2) error {
	// Must have at least one pipeline.
	if len(cfg.Pipelines) < 1 {
		return &configError{code: errMissingPipelines, msg: "must have at least one pipeline"}
	}

	// Validate pipelines.
	for _, pipeline := range cfg.Pipelines {
		if err := validatePipeline(cfg, pipeline); err != nil {
			return err
		}
	}
	return nil
}

func validatePipeline(cfg *configmodels.ConfigV2, pipeline *configmodels.Pipeline) error {
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

func validatePipelineReceivers(cfg *configmodels.ConfigV2, pipeline *configmodels.Pipeline) error {
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

func validatePipelineExporters(cfg *configmodels.ConfigV2, pipeline *configmodels.Pipeline) error {
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

func validatePipelineProcessors(cfg *configmodels.ConfigV2, pipeline *configmodels.Pipeline) error {
	if pipeline.InputType == configmodels.TracesDataType {
		// Traces pipeline must have at least one processor.
		if len(pipeline.Processors) == 0 {
			return &configError{
				code: errPipelineMustHaveProcessors,
				msg:  fmt.Sprintf("pipeline %q must have at least one processor", pipeline.Name),
			}
		}
	} else if pipeline.InputType == configmodels.MetricsDataType {
		// Metrics pipeline cannot have processors.
		if len(pipeline.Processors) > 0 {
			return &configError{
				code: errMetricPipelineCannotHaveProcessors,
				msg:  fmt.Sprintf("metrics pipeline %q cannot have processors", pipeline.Name),
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

	return nil
}
