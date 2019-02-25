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
	"fmt"
	"sync/atomic"
	"testing"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/data"
)

func TestMultiSpanProcessorMultiplexing(t *testing.T) {
	processors := make([]SpanProcessor, 3)
	for i := range processors {
		processors[i] = &mockSpanProcessor{}
	}

	tt := NewMultiSpanProcessor(processors)
	td := data.TraceData{
		Spans: make([]*tracepb.Span, 7),
	}

	var wantSpansCount = 0
	for i := 0; i < 2; i++ {
		wantSpansCount += len(td.Spans)
		tt.ProcessSpans(td, "test")
	}

	for _, p := range processors {
		m := p.(*mockSpanProcessor)
		if m.TotalSpans != wantSpansCount {
			t.Errorf("Wanted %d spans for every processor but got %d", wantSpansCount, m.TotalSpans)
			return
		}
	}
}

func TestMultiSpanProcessorWhenOneErrors(t *testing.T) {
	processors := make([]SpanProcessor, 3)
	for i := range processors {
		processors[i] = &mockSpanProcessor{}
	}

	// Make one processor return error
	m := processors[1].(*mockSpanProcessor)
	m.MustFail = true

	tt := NewMultiSpanProcessor(processors)
	td := data.TraceData{
		Spans: make([]*tracepb.Span, 5),
	}

	var wantSpansCount = 0
	for i := 0; i < 2; i++ {
		err := tt.ProcessSpans(td, "test")
		if err == nil {
			t.Errorf("Wanted error got nil")
			return
		}
		batchSize := len(td.Spans)
		wantSpansCount += batchSize
	}

	for _, p := range processors {
		m := p.(*mockSpanProcessor)
		if m.TotalSpans != wantSpansCount {
			t.Errorf("Wanted %d for every processor but got %d", wantSpansCount, m.TotalSpans)
			return
		}
	}
}

func TestMultiSpanProcessorWithPreProcessFn(t *testing.T) {
	processors := make([]SpanProcessor, 3)
	for i := range processors {
		processors[i] = &mockSpanProcessor{}
	}

	calledFnCount := int32(0)
	testPreProcessFn := func(data.TraceData, string) {
		atomic.AddInt32(&calledFnCount, 1)
	}

	tt := NewMultiSpanProcessor(processors, WithPreProcessFn(testPreProcessFn))
	batch := data.TraceData{
		Spans: make([]*tracepb.Span, 7),
	}

	var wantSpansCount = 0
	batchCount := 2
	for i := 0; i < batchCount; i++ {
		wantSpansCount += len(batch.Spans)
		tt.ProcessSpans(batch, "test")
	}

	for _, p := range processors {
		m := p.(*mockSpanProcessor)
		if m.TotalSpans != wantSpansCount {
			t.Errorf("Wanted %d spans for every processor but got %d", wantSpansCount, m.TotalSpans)
			return
		}
	}
	// We should call the preprocess function exactly once for each batch
	if int(calledFnCount) != batchCount {
		t.Errorf("Wanted to call preProcessFn %d times, but called %d", wantSpansCount, calledFnCount)
		return
	}
}

func TestMultiSpanProcessorWithAddAttributesOverwrite(t *testing.T) {
	multiSpanProcessorWithAddAttributesTestHelper(t, true)
}

func TestMultiSpanProcessorWithAddAttributesNoOverwrite(t *testing.T) {
	multiSpanProcessorWithAddAttributesTestHelper(t, false)
}

func multiSpanProcessorWithAddAttributesTestHelper(t *testing.T, overwrite bool) {
	processors := make([]SpanProcessor, 3)
	for i := range processors {
		processors[i] = &mockSpanProcessor{}
	}

	tt := NewMultiSpanProcessor(
		processors,
		WithAddAttributes(map[string]interface{}{
			"some_int":   1234,
			"some_str":   "some_string",
			"some_bool":  true,
			"some_float": 3.14159,
		}, overwrite),
	)

	td := data.TraceData{}
	for i := 0; i < 7; i++ {
		td.Spans = append(td.Spans, &tracepb.Span{
			Attributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					"some_int": {
						Value: &tracepb.AttributeValue_IntValue{IntValue: int64(4567)},
					},
				},
			},
		})
	}

	spans := make([]*tracepb.Span, 0, len(td.Spans)*2)
	for i := 0; i < 2; i++ {
		tt.ProcessSpans(td, "test")
		spans = append(spans, td.Spans...)
	}

	expectedSomeIntValue := int64(4567)
	if overwrite {
		expectedSomeIntValue = int64(1234)
	}

	// This should have modified the spans themselves
	for _, span := range spans {
		if val, ok := span.Attributes.AttributeMap["some_int"]; !ok || val.GetIntValue() != expectedSomeIntValue {
			t.Errorf("Missing or invalid int value")
			return
		}
		if val, ok := span.Attributes.AttributeMap["some_str"]; !ok || val.GetStringValue().Value != "some_string" {
			t.Errorf("Missing or invalid string value")
			return
		}
		if val, ok := span.Attributes.AttributeMap["some_bool"]; !ok || val.GetBoolValue() != true {
			t.Errorf("Missing or invalid bool value")
			return
		}
		if val, ok := span.Attributes.AttributeMap["some_float"]; !ok || val.GetDoubleValue() != float64(3.14159) {
			t.Errorf("Missing or invalid float value")
			return
		}
	}
}

type mockSpanProcessor struct {
	TotalSpans int
	MustFail   bool
}

var _ SpanProcessor = &mockSpanProcessor{}

func (p *mockSpanProcessor) ProcessSpans(td data.TraceData, spanFormat string) error {
	batchSize := len(td.Spans)
	p.TotalSpans += batchSize
	if p.MustFail {
		return fmt.Errorf("this processor must fail")
	}

	return nil
}
