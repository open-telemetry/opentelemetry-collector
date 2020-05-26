// Copyright The OpenTelemetry Authors
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

package attributesprocessor

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configerror"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/internal/processor/filterhelper"
	"go.opentelemetry.io/collector/internal/processor/filterspan"
)

const (
	// typeStr is the value of "type" key in configuration.
	typeStr = "attributes"
)

// Factory is the factory for Attributes processor.
type Factory struct {
}

// Type gets the type of the config created by this factory.
func (f *Factory) Type() configmodels.Type {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for the processor.
// Note: This isn't a valid configuration because the processor would do no work.
func (f *Factory) CreateDefaultConfig() configmodels.Processor {
	return &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
	}
}

// CreateTraceProcessor creates a trace processor based on this config.
func (f *Factory) CreateTraceProcessor(
	_ context.Context,
	_ component.ProcessorCreateParams,
	nextConsumer consumer.TraceConsumer,
	cfg configmodels.Processor,
) (component.TraceProcessor, error) {

	oCfg := cfg.(*Config)
	actions, err := buildAttributesConfiguration(*oCfg)
	if err != nil {
		return nil, err
	}
	include, err := filterspan.NewMatcher(oCfg.Include)
	if err != nil {
		return nil, err
	}
	exclude, err := filterspan.NewMatcher(oCfg.Exclude)
	if err != nil {
		return nil, err
	}
	config := attributesConfig{
		actions: actions,
		include: include,
		exclude: exclude,
	}
	return newTraceProcessor(nextConsumer, config)
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (f *Factory) CreateMetricsProcessor(
	_ context.Context,
	_ component.ProcessorCreateParams,
	_ consumer.MetricsConsumer,
	_ configmodels.Processor,
) (component.MetricsProcessor, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}

// buildAttributesConfiguration validates the input configuration has all of the required fields for the processor
// and returns a list of valid actions to configure the processor.
// An error is returned if there are any invalid inputs.
func buildAttributesConfiguration(config Config) ([]attributeAction, error) {
	if len(config.Actions) == 0 {
		return nil, fmt.Errorf("error creating \"attributes\" processor due to missing required field \"actions\" of processor %q", config.Name())
	}

	var attributeActions []attributeAction
	for i, a := range config.Actions {
		// `key` is a required field
		if a.Key == "" {
			return nil, fmt.Errorf("error creating \"attributes\" processor due to missing required field \"key\" at the %d-th actions of processor %q", i, config.Name())
		}

		// Convert `action` to lowercase for comparison.
		a.Action = Action(strings.ToLower(string(a.Action)))
		action := attributeAction{
			Key:    a.Key,
			Action: a.Action,
		}

		switch a.Action {
		case INSERT, UPDATE, UPSERT:
			if a.Value == nil && a.FromAttribute == "" {
				return nil, fmt.Errorf("error creating \"attributes\" processor. Either field \"value\" or \"from_attribute\" setting must be specified for %d-th action of processor %q", i, config.Name())
			}
			if a.Value != nil && a.FromAttribute != "" {
				return nil, fmt.Errorf("error creating \"attributes\" processor due to both fields \"value\" and \"from_attribute\" being set at the %d-th actions of processor %q", i, config.Name())
			}
			// Convert the raw value from the configuration to the internal trace representation of the value.
			if a.Value != nil {
				val, err := filterhelper.NewAttributeValueRaw(a.Value)
				if err != nil {
					return nil, err
				}
				action.AttributeValue = &val
			} else {
				action.FromAttribute = a.FromAttribute
			}
		case HASH, DELETE:
			if a.Value != nil || a.FromAttribute != "" {
				return nil, fmt.Errorf("error creating \"attributes\" processor. Action \"%s\" does not use \"value\" or \"from_attribute\" field. These must not be specified for %d-th action of processor %q", a.Action, i, config.Name())
			}

		default:
			return nil, fmt.Errorf("error creating \"attributes\" processor due to unsupported action %q at the %d-th actions of processor %q", a.Action, i, config.Name())
		}

		attributeActions = append(attributeActions, action)
	}
	return attributeActions, nil
}
