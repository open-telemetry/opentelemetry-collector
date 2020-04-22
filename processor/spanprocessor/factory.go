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

package spanprocessor

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configerror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
)

const (
	// typeStr is the value of "type" Span processor in the configuration.
	typeStr = "span"
)

// errMissingRequiredField is returned when a required field in the config
// is not specified.
// TODO https://github.com/open-telemetry/opentelemetry-collector/issues/215
//	Move this to the error package that allows for span name and field to be specified.
var errMissingRequiredField = errors.New("error creating \"span\" processor: either \"from_attributes\" or \"to_attributes\" must be specified in \"name:\"")

// Factory is the factory for the Span processor.
type Factory struct {
}

var _ component.ProcessorFactory = (*Factory)(nil)

// Type gets the type of the config created by this factory.
func (f *Factory) Type() configmodels.Type {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for processor.
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
	cfg configmodels.Processor) (component.TraceProcessor, error) {

	// 'from_attributes' or 'to_attributes' under 'name' has to be set for the span
	// processor to be valid. If not set and not enforced, the processor would do no work.
	oCfg := cfg.(*Config)
	if len(oCfg.Rename.FromAttributes) == 0 &&
		(oCfg.Rename.ToAttributes == nil || len(oCfg.Rename.ToAttributes.Rules) == 0) {
		return nil, errMissingRequiredField
	}

	return newSpanProcessor(nextConsumer, *oCfg)
}

// CreateMetricsProcessor creates a metric processor based on this config.
func (f *Factory) CreateMetricsProcessor(
	_ context.Context,
	_ component.ProcessorCreateParams,
	_ consumer.MetricsConsumer,
	_ configmodels.Processor) (component.MetricsProcessor, error) {
	// Span Processor does not support Metrics.
	return nil, configerror.ErrDataTypeIsNotSupported
}
