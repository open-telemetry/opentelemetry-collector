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

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-service/oterr"
	"github.com/open-telemetry/opentelemetry-service/processor"
)

type attributesProcessor struct {
	nextConsumer consumer.TraceConsumer
	// This structure is very similar to the config for attributes processor
	// with the value in the converted attribute format instead of the
	// raw format from the configuration.
	actions []attributeAction
}

type attributeAction struct {
	Key            string
	FromAttribute  string
	Action         Action
	AttributeValue *tracepb.AttributeValue
}

// newTraceProcessor returns a processor that modifies attributes of a span.
// To construct the attributes processors, the use of the factory methods are required
// in order to validate the inputs.
func newTraceProcessor(nextConsumer consumer.TraceConsumer, actions []attributeAction) (processor.TraceProcessor, error) {
	if nextConsumer == nil {
		return nil, oterr.ErrNilNextConsumer
	}
	ap := &attributesProcessor{
		nextConsumer: nextConsumer,
		actions:      actions,
	}
	return ap, nil
}

func (a *attributesProcessor) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	for _, span := range td.Spans {
		if span == nil {
			// Do not create empty spans just to add attributes
			continue
		}
		if span.Attributes == nil {
			span.Attributes = &tracepb.Span_Attributes{}
		}
		// Create a new map if one does not exist and size it to the number of actions.
		// This is the largest size of the new map could be if every action is an insert.
		if span.Attributes.AttributeMap == nil {
			span.Attributes.AttributeMap = make(map[string]*tracepb.AttributeValue, len(a.actions))
		}

		for _, action := range a.actions {

			// TODO https://github.com/open-telemetry/opentelemetry-service/issues/296
			// Do benchmark testing between having action be of type string vs integer.
			// The reason is attributes processor will most likely be commonly used
			// and could impact performance.
			switch action.Action {
			case DELETE:
				delete(span.Attributes.AttributeMap, action.Key)
			case INSERT, UPSERT, UPDATE:
				modifyAttribute(action, span.Attributes.AttributeMap)
			}
		}
	}
	return a.nextConsumer.ConsumeTraceData(ctx, td)
}

func modifyAttribute(action attributeAction, attributesMap map[string]*tracepb.AttributeValue) {
	// Exists is used to determine if the action insert or update should occur.
	// This value doesn't matter if the action is upsert.
	_, exists := attributesMap[action.Key]

	// No action should occur:
	// - if the key exists and the action is insert
	// or
	// - if the key does not exist and the action is update.
	// If the action is UPSERT, this value does not matter.
	if (exists && action.Action == INSERT) ||
		(!exists && action.Action == UPDATE) {
		return
	}

	// Set the key with a value from the configuration.
	if action.AttributeValue != nil {
		attributesMap[action.Key] = action.AttributeValue
	} else {
		// Set the key with a value from another attribute, if it exists.
		value, fromAttributeExists := attributesMap[action.FromAttribute]
		if !fromAttributeExists {
			return
		}
		attributesMap[action.Key] = value
	}
}
