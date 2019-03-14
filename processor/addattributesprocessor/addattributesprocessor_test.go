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

package addattributesprocessor

import (
	"context"
	"errors"
	"testing"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/exporter/exportertest"
)

func TestAddAttributesProcessorInvalidValue(t *testing.T) {
	_, err := NewTraceProcessor(
		exportertest.NewNopTraceExporter(),
		WithAttributes(map[string]interface{}{
			"some_int": func() error { return nil },
		}))
	if err == nil {
		t.Fatalf("Unexpected error when creating with invalid attribute value: want not-nil got nil")
	}
}

func TestAddAttributesProcessorWithEmptyMap(t *testing.T) {
	want := error(nil)
	tt, err := NewTraceProcessor(exportertest.NewNopTraceExporter())
	if err != nil {
		t.Fatalf("Unexpected error when creating: want nil got %v", err)
	}
	if got := tt.ConsumeTraceData(context.Background(), data.TraceData{}); got != want {
		t.Fatalf("ConsumeTraceData return error: want %v got %v", want, got)
	}
}

func TestAddAttributesProcessorWithEmptyMapReturnsError(t *testing.T) {
	want := errors.New("my_error")
	tt, err := NewTraceProcessor(exportertest.NewNopTraceExporter(exportertest.WithReturnError(want)))
	if err != nil {
		t.Fatalf("Unexpected error when creating: want nil got %v", err)
	}
	if got := tt.ConsumeTraceData(context.Background(), data.TraceData{}); got != want {
		t.Fatalf("ConsumeTraceData return error: want %v got %v", want, got)
	}
}

func TestAddAttributesProcessorNoAttributesAndNilSpan(t *testing.T) {
	sinkExporter := &exportertest.SinkTraceExporter{}

	tt, err := NewTraceProcessor(
		sinkExporter,
		WithAttributes(map[string]interface{}{
			"some_int":   1234,
			"some_str":   "some_string",
			"some_bool":  true,
			"some_float": 3.14159,
		}),
		WithOverwrite(true))
	if err != nil {
		t.Fatalf("Unexpected error when creating: want nil got %v", err)
	}

	td := data.TraceData{}
	td.Spans = append(td.Spans, nil, &tracepb.Span{
		Name: &tracepb.TruncatableString{Value: "foo"},
	})

	if err := tt.ConsumeTraceData(context.Background(), td); err != nil {
		t.Fatalf("ConsumeTraceData return error: want nil got %v", err)
	}

	gotTraces := sinkExporter.AllTraces()
	if len(gotTraces) != 1 {
		t.Fatalf("Unexpected number of received traces")
	}
	if len(gotTraces[0].Spans) != 2 {
		t.Errorf("Unexpected number of received spans")
	}
	if gotTraces[0].Spans[0] != nil {
		t.Errorf("First received span should be nil")
	}
	span := gotTraces[0].Spans[1]
	if val, ok := span.Attributes.AttributeMap["some_int"]; !ok || val.GetIntValue() != 1234 {
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

func TestAddAttributesProcessorOverwrite(t *testing.T) {
	addAttributesProcessorTestHelper(t, true)
}

func TestAddAttributesProcessorNoOverwrite(t *testing.T) {
	addAttributesProcessorTestHelper(t, false)
}

func addAttributesProcessorTestHelper(t *testing.T, overwrite bool) {
	sinkExporter := &exportertest.SinkTraceExporter{}

	tt, err := NewTraceProcessor(
		sinkExporter,
		WithAttributes(map[string]interface{}{
			"some_int":   1234,
			"some_str":   "some_string",
			"some_bool":  true,
			"some_float": 3.14159,
		}),
		WithOverwrite(overwrite))
	if err != nil {
		t.Fatalf("Unexpected error when creating: want nil got %v", err)
	}

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

	for i := 0; i < 2; i++ {
		if err := tt.ConsumeTraceData(context.Background(), td); err != nil {
			t.Fatalf("ConsumeTraceData return error: want nil got %v", err)
		}
	}

	expectedSomeIntValue := int64(4567)
	if overwrite {
		expectedSomeIntValue = int64(1234)
	}

	// This should have modified the spans themselves
	for _, td := range sinkExporter.AllTraces() {
		for _, span := range td.Spans {
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
}
