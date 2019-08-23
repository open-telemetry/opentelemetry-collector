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

type attributesAction struct {
	Key            string
	FromAttribute  string
	Action         string
	AttributeValue *tracepb.AttributeValue
}
type attributesProcessor struct {
	nextConsumer consumer.TraceConsumer
	// Should convert the attributes to internal format for value stuff
	actions []attributesAction
}

// NewTraceProcessor returns a processor that modifies attributes of a span.
func NewTraceProcessor(nextConsumer consumer.TraceConsumer, actions []attributesAction) (processor.TraceProcessor, error) {
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

		for _, action := range a.actions {

			// Delete is the simplest action.
			if action.Action == DELETE {
				// delete() will do nothing if the key does not exist.
				delete(span.Attributes.AttributeMap, action.Key)
				continue
			}

			// Exists is used to determine if the action insert or update should occur.
			_, exists := span.Attributes.AttributeMap[action.Key]
			// Default value of FromAttribute is "" and is a valid key into an attribute map.
			_, fromAttributeExists := span.Attributes.AttributeMap[action.FromAttribute]

			// No action should occur:
			// - if the key exists and the action is insert
			// or
			// - if the key does not exist and the action is update.
			// or
			// - the from attribute does not exists and it is the source of the value.
			if (exists && action.Action == INSERT) ||
				(!exists && action.Action == UPDATE) ||
				(action.AttributeValue == nil && !fromAttributeExists) {
				continue
			}

			// At this point, the attribute will be set because:
			// - the action is upsert
			// - the key does not exist and the action is insert
			// - the key does exist and the action is update.
			if action.AttributeValue != nil {
				span.Attributes.AttributeMap[action.Key] = action.AttributeValue
			} else {
				span.Attributes.AttributeMap[action.Key] = span.Attributes.AttributeMap[action.FromAttribute]
			}

		}
	}
	return a.nextConsumer.ConsumeTraceData(ctx, td)
}
