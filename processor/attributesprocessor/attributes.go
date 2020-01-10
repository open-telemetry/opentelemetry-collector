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
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
	"github.com/open-telemetry/opentelemetry-collector/processor"
)

type attributesProcessor struct {
	nextConsumer consumer.TraceConsumer
	config       attributesConfig
}

// This structure is very similar to the config for attributes processor
// with the value in the converted attribute format instead of the
// raw format from the configuration.
type attributesConfig struct {
	actions []attributeAction
	include *matchingProperties
	exclude *matchingProperties
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

// matchingProperties stores the MatchProperties in a format to simplify
// property checking:
// 1. the list of service names is stored in a map for quick lookup.
// 2. the attribute values are stored in the internal format.
type matchingProperties struct {
	Services   map[string]bool
	Attributes []matchAttribute
}

type matchAttribute struct {
	Key            string
	AttributeValue *tracepb.AttributeValue
}

// newTraceProcessor returns a processor that modifies attributes of a span.
// To construct the attributes processors, the use of the factory methods are required
// in order to validate the inputs.
func newTraceProcessor(nextConsumer consumer.TraceConsumer, config attributesConfig) (processor.TraceProcessor, error) {
	if nextConsumer == nil {
		return nil, oterr.ErrNilNextConsumer
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
			}
		}
	}
	return a.nextConsumer.ConsumeTraceData(ctx, td)
}

func (a *attributesProcessor) GetCapabilities() processor.Capabilities {
	return processor.Capabilities{MutatesConsumedData: true}
}

// Start is invoked during service startup.
func (a *attributesProcessor) Start(host component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (a *attributesProcessor) Shutdown() error {
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

// skipSpan determines if a span should be processed.
// True is returned when a span should be skipped.
// False is returned when a span should not be skipped.
// The logic determining if a span should be processed is set
// in the attribute configuration with the include and exclude settings.
// Include properties are checked before exclude settings are checked.
func (a *attributesProcessor) skipSpan(span *tracepb.Span, serviceName string) bool {
	// By default all spans are processed when no include and exclude properties are set.
	if a.config.include == nil && a.config.exclude == nil {
		return false
	}

	if a.config.include != nil {
		// A false returned in this case means the span should not be processed.
		if include := matchSpanToProperties(*a.config.include, span, serviceName); !include {
			return true
		}
	}

	if a.config.exclude != nil {
		// A true returned in this case means the span should not be processed.
		if exclude := matchSpanToProperties(*a.config.exclude, span, serviceName); exclude {
			return true
		}
	}

	return false
}

// matchProperties matches a span and service to a set of properties.
// There are two sets of properties to match against.
// The service name is checked first, if specified. The attributes are checked
// afterwards, if specified.
// At least one of services or attributes must be specified. It is supported
// to have both specified, but both `services` and `attributes` must evaluate
// to true for a match to occur.
func matchSpanToProperties(mp matchingProperties, span *tracepb.Span, serviceName string) bool {

	if len(mp.Services) != 0 {
		if serviceFound := mp.Services[serviceName]; !serviceFound {
			return false
		}
	}

	// If there are no attributes to match against, the span matches.
	if len(mp.Attributes) == 0 {
		return true
	}

	// At this point, it is expected of the span to have attributes because of
	// len (mp.Attributes) != 0. For spans with no attributes, it does not match.
	if span.Attributes == nil || len(span.Attributes.AttributeMap) == 0 {
		return false
	}

	// Check that all expected properties are set.
	for _, property := range mp.Attributes {
		val, exist := span.Attributes.AttributeMap[property.Key]
		if !exist {
			return false
		}

		// This is for the case of checking that the key existed.
		if property.AttributeValue == nil {
			continue
		}

		var isMatch bool
		switch attribValue := val.Value.(type) {
		case *tracepb.AttributeValue_StringValue:
			if sv, ok := property.AttributeValue.GetValue().(*tracepb.AttributeValue_StringValue); ok {
				isMatch = attribValue.StringValue.GetValue() == sv.StringValue.GetValue()
			}
		case *tracepb.AttributeValue_IntValue:
			if iv, ok := property.AttributeValue.GetValue().(*tracepb.AttributeValue_IntValue); ok {
				isMatch = attribValue.IntValue == iv.IntValue
			}
		case *tracepb.AttributeValue_BoolValue:
			if bv, ok := property.AttributeValue.GetValue().(*tracepb.AttributeValue_BoolValue); ok {
				isMatch = attribValue.BoolValue == bv.BoolValue
			}
		case *tracepb.AttributeValue_DoubleValue:
			if dv, ok := property.AttributeValue.GetValue().(*tracepb.AttributeValue_DoubleValue); ok {
				isMatch = attribValue.DoubleValue == dv.DoubleValue
			}
		}
		if !isMatch {
			return false
		}

	}

	// All properties have been satisfied so the span does match.
	return true
}
