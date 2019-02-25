// Copyright 2018, OpenCensus Authors
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

package processor

import (
	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/internal"

	"github.com/spf13/cast"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
)

// MultiProcessorOption represents options that can be applied to a MultiSpanProcessor.
type MultiProcessorOption func(*multiSpanProcessor)
type preProcessFn func(data.TraceData, string)

// MultiSpanProcessor enables processing on multiple processors.
// For each incoming span batch, it calls ProcessSpans method on each span
// processor one-by-one. It aggregates success/failures/errors from all of
// them and reports the result upstream.
type multiSpanProcessor struct {
	processors    []SpanProcessor
	preProcessFns []preProcessFn
}

// NewMultiSpanProcessor creates a multiSpanProcessor from the variadic
// list of passed SpanProcessors and options.
func NewMultiSpanProcessor(procs []SpanProcessor, options ...MultiProcessorOption) SpanProcessor {
	multiSpanProc := &multiSpanProcessor{
		processors: procs,
	}
	for _, opt := range options {
		opt(multiSpanProc)
	}
	return multiSpanProc
}

// WithPreProcessFn returns a MultiProcessorOption that applies some preProcessFn to all ExportTraceServiceRequests.
func WithPreProcessFn(preProcFn preProcessFn) MultiProcessorOption {
	return func(msp *multiSpanProcessor) {
		msp.preProcessFns = append(msp.preProcessFns, preProcFn)
	}
}

// WithAddAttributes returns a MultiProcessorOption that adds the provided attributes to all spans
// in each ExportTraceServiceRequest.
func WithAddAttributes(attributes map[string]interface{}, overwrite bool) MultiProcessorOption {
	return WithPreProcessFn(
		func(td data.TraceData, spanFormat string) {
			if len(attributes) == 0 {
				return
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
					span.Attributes.AttributeMap = make(map[string]*tracepb.AttributeValue, len(attributes))
				}
				// Copy all attributes that need to be added into the span's attribute map.
				// If a key already exists, we will only overwrite is the overwrite flag
				// is set to true.
				for key, value := range attributes {
					if _, exists := span.Attributes.AttributeMap[key]; overwrite || !exists {
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
							// just skip values that we cannot parse.
						}
						if attrib.Value != nil {
							span.Attributes.AttributeMap[key] = attrib
						}
					}
				}
			}
		},
	)
}

// ProcessSpans implements the SpanProcessor interface
func (msp *multiSpanProcessor) ProcessSpans(td data.TraceData, spanFormat string) error {
	for _, preProcessFn := range msp.preProcessFns {
		preProcessFn(td, spanFormat)
	}
	var errors []error
	for _, sp := range msp.processors {
		err := sp.ProcessSpans(td, spanFormat)
		if err != nil {
			errors = append(errors, err)
		}
	}
	return internal.CombineErrors(errors)
}
