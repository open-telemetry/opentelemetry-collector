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

package attributesprocessor

import (
	"errors"
	"fmt"
	"strings"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/spf13/cast"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configerror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/processor"
)

const (
	// typeStr is the value of "type" key in configuration.
	typeStr = "attributes"
)

// Factory is the factory for Attributes processor.
type Factory struct {
}

// Type gets the type of the config created by this factory.
func (f *Factory) Type() string {
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
	logger *zap.Logger,
	nextConsumer consumer.TraceConsumer,
	cfg configmodels.Processor,
) (processor.TraceProcessor, error) {

	oCfg := cfg.(*Config)
	actions, err := buildAttributesConfiguration(*oCfg)
	if err != nil {
		return nil, err
	}
	include, err := buildMatchProperties(oCfg.Include)
	if err != nil {
		return nil, err
	}
	exclude, err := buildMatchProperties(oCfg.Exclude)
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
	logger *zap.Logger,
	nextConsumer consumer.MetricsConsumer,
	cfg configmodels.Processor,
) (processor.MetricsProcessor, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}

// attributeValue is used to convert the raw `value` from ActionKeyValue to the supported trace attribute values.
func attributeValue(value interface{}) (*tracepb.AttributeValue, error) {
	attrib := &tracepb.AttributeValue{}
	switch val := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		attrib.Value = &tracepb.AttributeValue_IntValue{IntValue: cast.ToInt64(val)}
	case float32, float64:
		attrib.Value = &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(val)}
	case string:
		attrib.Value = &tracepb.AttributeValue_StringValue{
			StringValue: &tracepb.TruncatableString{Value: val},
		}
	case bool:
		attrib.Value = &tracepb.AttributeValue_BoolValue{BoolValue: val}
	default:
		return nil, fmt.Errorf("error unsupported value type \"%T\"", value)
	}
	return attrib, nil
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
				val, err := attributeValue(a.Value)
				if err != nil {
					return nil, err
				}
				action.AttributeValue = val
			} else {
				action.FromAttribute = a.FromAttribute
			}

		case DELETE:
			// Do nothing since `key` is the only required field for `delete` action.

		default:
			return nil, fmt.Errorf("error creating \"attributes\" processor due to unsupported action %q at the %d-th actions of processor %q", a.Action, i, config.Name())
		}

		attributeActions = append(attributeActions, action)
	}
	return attributeActions, nil
}

func buildMatchProperties(config *MatchProperties) (*matchingProperties, error) {
	if config == nil {
		return nil, nil
	}

	if len(config.Services) == 0 && len(config.Attributes) == 0 {
		return nil, errors.New("error creating \"attributes\" processor. At least one field \"services\" or \"attributes\" must be specified")
	}
	properties := &matchingProperties{}
	for _, serviceName := range config.Services {
		if serviceName == "" {
			return nil, errors.New("error creating \"attributes\" processor. Can't have empty string for service name in list of services")
		}
	}
	svcNames := make(map[string]bool, len(config.Services))
	for _, name := range config.Services {
		svcNames[name] = true
	}
	// Copy the list of service names after each one has been validated. If the list is empty, that is okay because the match properties too will be empty.
	properties.Services = svcNames

	var rawAttributes []matchAttribute
	for _, attribute := range config.Attributes {

		if attribute.Key == "" {
			return nil, errors.New("error creating \"attributes\" processor. Can't have empty string for service name in list of attributes")
		}

		entry := matchAttribute{
			Key: attribute.Key,
		}
		if attribute.Value != nil {
			val, err := attributeValue(attribute.Value)
			if err != nil {
				return nil, err
			}
			entry.AttributeValue = val
		}

		rawAttributes = append(rawAttributes, entry)

	}
	properties.Attributes = rawAttributes
	return properties, nil
}
