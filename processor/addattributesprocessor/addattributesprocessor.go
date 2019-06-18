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

package addattributesprocessor

import (
	"context"
	"errors"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/spf13/cast"

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/data"
	"github.com/open-telemetry/opentelemetry-service/processor"
)

var (
	errUnsupportedType = errors.New("Unsupported type")
)

type addattributesprocessor struct {
	attributeMap map[string]*tracepb.AttributeValue
	overwrite    bool
	nextConsumer consumer.TraceConsumer
}

// Option represents options that can be applied to a NopExporter.
type Option func(*addattributesprocessor) error

// WithOverwrite returns an Option to configure the capability to overwrite an already existing key.
func WithOverwrite(overwrite bool) Option {
	return func(aap *addattributesprocessor) error {
		aap.overwrite = overwrite
		return nil
	}
}

// WithAttributes returns an Option to configure the attributes to be added to all spans.
func WithAttributes(attributes map[string]interface{}) Option {
	return func(aap *addattributesprocessor) error {
		attributeMap := make(map[string]*tracepb.AttributeValue, len(attributes))
		// Copy all attributes that need to be added into the span's attribute map.
		for key, value := range attributes {
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
				return errUnsupportedType
			}
			attributeMap[key] = attrib
		}
		aap.attributeMap = attributeMap
		return nil
	}
}

var _ processor.TraceProcessor = (*addattributesprocessor)(nil)

// NewTraceProcessor returns a processor.TraceProcessor that adds the WithAttributeMap(attributes) to all spans
// passed to it. If a key already exists, we will only overwrite is the WithOverwrite(true) is set.
func NewTraceProcessor(nextConsumer consumer.TraceConsumer, options ...Option) (processor.TraceProcessor, error) {
	aap := &addattributesprocessor{nextConsumer: nextConsumer}
	for _, opt := range options {
		if err := opt(aap); err != nil {
			return nil, err
		}
	}
	return aap, nil
}

func (aap *addattributesprocessor) ConsumeTraceData(ctx context.Context, td data.TraceData) error {
	if len(aap.attributeMap) == 0 {
		return aap.nextConsumer.ConsumeTraceData(ctx, td)
	}
	for _, span := range td.Spans {
		if span == nil {
			// We will not create nil spans with just attributes on them
			continue
		}
		if span.Attributes == nil {
			span.Attributes = &tracepb.Span_Attributes{}
		}
		// Create a new map if one does not exist. Could re-use passed in map, but
		// feels too unsafe.
		if span.Attributes.AttributeMap == nil {
			span.Attributes.AttributeMap = make(map[string]*tracepb.AttributeValue, len(aap.attributeMap))
		}
		// Copy all attributes that need to be added into the span's attribute map.
		for key, value := range aap.attributeMap {
			if _, exists := span.Attributes.AttributeMap[key]; aap.overwrite || !exists {
				span.Attributes.AttributeMap[key] = value
			}
		}
	}
	return aap.nextConsumer.ConsumeTraceData(ctx, td)
}
