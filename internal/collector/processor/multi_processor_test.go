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
	"testing"

	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
)

func TestMultiSpanProcessorMultiplexing(t *testing.T) {
	processors := make([]SpanProcessor, 3)
	for i := range processors {
		processors[i] = &mockSpanProcessor{}
	}

	tt := NewMultiSpanProcessor(processors...)
	batch := &agenttracepb.ExportTraceServiceRequest{
		Spans: make([]*tracepb.Span, 7),
	}

	var wantSpansCount = 0
	for i := 0; i < 2; i++ {
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
}

func TestMultiSpanProcessorSomeNotOk(t *testing.T) {
	processors := make([]SpanProcessor, 3)
	for i := range processors {
		processors[i] = &mockSpanProcessor{}
	}

	// Make one processor return false for some spans
	m := processors[1].(*mockSpanProcessor)
	wantFailures := uint64(2)
	m.Failures = wantFailures

	tt := NewMultiSpanProcessor(processors...)
	spans := make([]*tracepb.Span, wantFailures+3)
	for i := range spans {
		spans[i] = &tracepb.Span{}
	}
	batch := &agenttracepb.ExportTraceServiceRequest{
		Spans: spans,
	}

	var wantSpansCount = 0
	for i := 0; i < 2; i++ {
		failures, _ := tt.ProcessSpans(batch, "test")
		batchSize := len(batch.Spans)
		wantSpansCount += batchSize
		if wantFailures != failures {
			t.Errorf("Wanted %d failures but got %d", wantFailures, failures)
		}
	}

	for _, p := range processors {
		m := p.(*mockSpanProcessor)
		if m.TotalSpans != wantSpansCount {
			t.Errorf("Wanted %d for every processor but got %d", wantSpansCount, m.TotalSpans)
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

	tt := NewMultiSpanProcessor(processors...)
	batch := &agenttracepb.ExportTraceServiceRequest{
		Spans: make([]*tracepb.Span, 5),
	}

	var wantSpansCount = 0
	for i := 0; i < 2; i++ {
		failures, err := tt.ProcessSpans(batch, "test")
		if err == nil {
			t.Errorf("Wanted error got nil")
			return
		}
		batchSize := len(batch.Spans)
		wantSpansCount += batchSize
		if failures != uint64(batchSize) {
			t.Errorf("Wanted all spans to fail, got a different value.")
		}
	}

	for _, p := range processors {
		m := p.(*mockSpanProcessor)
		if m.TotalSpans != wantSpansCount {
			t.Errorf("Wanted %d for every processor but got %d", wantSpansCount, m.TotalSpans)
			return
		}
	}
}

type mockSpanProcessor struct {
	Failures   uint64
	TotalSpans int
	MustFail   bool
}

var _ SpanProcessor = &mockSpanProcessor{}

func (p *mockSpanProcessor) ProcessSpans(batch *agenttracepb.ExportTraceServiceRequest, spanFormat string) (uint64, error) {
	batchSize := len(batch.Spans)
	p.TotalSpans += batchSize
	if p.MustFail {
		return uint64(batchSize), fmt.Errorf("this processor must fail")
	}

	return p.Failures, nil
}
