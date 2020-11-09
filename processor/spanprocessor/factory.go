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

package spanprocessor

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

const (
	// typeStr is the value of "type" Span processor in the configuration.
	typeStr = "span"
)

var processorCapabilities = component.ProcessorCapabilities{MutatesConsumedData: true}

// errMissingRequiredField is returned when a required field in the config
// is not specified.
// TODO https://github.com/open-telemetry/opentelemetry-collector/issues/215
//	Move this to the error package that allows for span name and field to be specified.
var errMissingRequiredField = errors.New("error creating \"span\" processor: either \"from_attributes\" or \"to_attributes\" must be specified in \"name:\"")

// NewFactory returns a new factory for the Span processor.
func NewFactory() component.ProcessorFactory {
	return processorhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		processorhelper.WithTraces(createTraceProcessor))
}

func createDefaultConfig() configmodels.Processor {
	return &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
	}
}

func createTraceProcessor(
	_ context.Context,
	_ component.ProcessorCreateParams,
	cfg configmodels.Processor,
	nextConsumer consumer.TracesConsumer,
) (component.TracesProcessor, error) {

	// 'from_attributes' or 'to_attributes' under 'name' has to be set for the span
	// processor to be valid. If not set and not enforced, the processor would do no work.
	oCfg := cfg.(*Config)
	if len(oCfg.Rename.FromAttributes) == 0 &&
		(oCfg.Rename.ToAttributes == nil || len(oCfg.Rename.ToAttributes.Rules) == 0) {
		return nil, errMissingRequiredField
	}

	sp, err := newSpanProcessor(*oCfg)
	if err != nil {
		return nil, err
	}
	return processorhelper.NewTraceProcessor(
		cfg,
		nextConsumer,
		sp,
		processorhelper.WithCapabilities(processorCapabilities))
}
