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
	"context"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/component/componenterror"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal/processor/span"
	"github.com/open-telemetry/opentelemetry-collector/processor"
)

type attributesProcessor struct {
	nextConsumer consumer.TraceConsumerOld
	config       attributesConfig
}

// This structure is very similar to the config for attributes processor
// with the value in the converted attribute format instead of the
// raw format from the configuration.
type attributesConfig struct {
	actions []attributeAction
	include span.Matcher
	exclude span.Matcher
}

type attributeAction struct {
	Key           string
	FromAttribute string
	// TODO https://github.com/open-telemetry/opentelemetry-collector/issues/296
	// Do benchmark testing between having action be of type string vs integer.
	// The reason is attributes processor will most likely be commonly used
	// and could impact performance.
	Action         Action
	AttributeValue *tracepb.AttributeValue
}

// newTraceProcessor returns a processor that modifies attributes of a span.
// To construct the attributes processors, the use of the factory methods are required
// in order to validate the inputs.
func newTraceProcessor(nextConsumer consumer.TraceConsumerOld, config attributesConfig) (component.TraceProcessorOld, error) {
	if nextConsumer == nil {
		return nil, componenterror.ErrNilNextConsumer
	}
	ap := &attributesProcessor{
		nextConsumer: nextConsumer,
		config:       config,
	}
	return ap, nil
}

func (a *attributesProcessor) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	serviceName := processor.ServiceNameForNode(td.Node)
	for _, span := range td.Spans {
		if span == nil {
			// Do not create empty spans just to add attributes
			continue
		}

		if a.skipSpan(span, serviceName) {
			continue
		}

		if span.Attributes == nil {
			span.Attributes = &tracepb.Span_Attributes{}
		}
		// Create a new map if one does not exist and size it to the number of actions.
		// This is the largest size of the new map could be if every action is an insert.
		if span.Attributes.AttributeMap == nil {
			span.Attributes.AttributeMap = make(map[string]*tracepb.AttributeValue, len(a.config.actions))
		}

		for _, action := range a.config.actions {

			// TODO https://github.com/open-telemetry/opentelemetry-collector/issues/296
			// Do benchmark testing between having action be of type string vs integer.
			// The reason is attributes processor will most likely be commonly used
			// and could impact performance.
			switch action.Action {
			case DELETE:
				delete(span.Attributes.AttributeMap, action.Key)
			case INSERT:
				insertAttribute(action, span.Attributes.AttributeMap)
			case UPDATE:
				updateAttribute(action, span.Attributes.AttributeMap)
			case UPSERT:
				// There is no need to check if the target key exists in the attribute map
				// because the value is to be set regardless.
				setAttribute(action, span.Attributes.AttributeMap)
			case HASH:
				hashAttribute(action, span.Attributes.AttributeMap)
			}
		}
	}
	return a.nextConsumer.ConsumeTraceData(ctx, td)
}

func (a *attributesProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

// Start is invoked during service startup.
func (a *attributesProcessor) Start(ctx context.Context, host component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (a *attributesProcessor) Shutdown(context.Context) error {
	return nil
}

func insertAttribute(action attributeAction, attributesMap map[string]*tracepb.AttributeValue) {
	// Insert is only performed when the target key does not already exist
	// in the attribute map.
	if _, exists := attributesMap[action.Key]; exists {
		return
	}

	setAttribute(action, attributesMap)
}

func updateAttribute(action attributeAction, attributesMap map[string]*tracepb.AttributeValue) {
	// Update is only performed when the target key already exists in
	// the attribute map.
	if _, exists := attributesMap[action.Key]; !exists {
		return
	}

	setAttribute(action, attributesMap)
}

func setAttribute(action attributeAction, attributesMap map[string]*tracepb.AttributeValue) {
	// Set the key with a value from the configuration.
	if action.AttributeValue != nil {
		attributesMap[action.Key] = action.AttributeValue
	} else if value, fromAttributeExists := attributesMap[action.FromAttribute]; fromAttributeExists {
		// Set the key with a value from another attribute, if it exists.
		attributesMap[action.Key] = value
	}
}

func hashAttribute(action attributeAction, attributesMap map[string]*tracepb.AttributeValue) {
	if value, exists := attributesMap[action.Key]; exists {
		attributesMap[action.Key] = SHA1AttributeHahser(value)
	}
}

// skipSpan determines if a span should be processed.
// True is returned when a span should be skipped.
// False is returned when a span should not be skipped.
// The logic determining if a span should be processed is set
// in the attribute configuration with the include and exclude settings.
// Include properties are checked before exclude settings are checked.
func (a *attributesProcessor) skipSpan(span *tracepb.Span, serviceName string) bool {
	if a.config.include != nil {
		// A false returned in this case means the span should not be processed.
		if include := a.config.include.MatchSpan(span, serviceName); !include {
			return true
		}
	}

	if a.config.exclude != nil {
		// A true returned in this case means the span should not be processed.
		if exclude := a.config.exclude.MatchSpan(span, serviceName); exclude {
			return true
		}
	}

	return false
}
