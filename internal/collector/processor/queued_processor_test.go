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
	"sync"
	"sync/atomic"
	"testing"

	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
)

func TestQueueProcessorHappyPath(t *testing.T) {
	mockProc := newMockConcurrentSpanProcessor()
	qp := NewQueuedSpanProcessor(mockProc)
	goFn := func(b *agenttracepb.ExportTraceServiceRequest) {
		qp.ProcessSpans(b, "test")
	}

	spans := []*tracepb.Span{{}}
	wantBatches := 10
	wantSpans := 0
	for i := 0; i < wantBatches; i++ {
		batch := &agenttracepb.ExportTraceServiceRequest{
			Spans: spans,
		}
		wantSpans += len(spans)
		spans = append(spans, &tracepb.Span{})
		fn := func() { goFn(batch) }
		mockProc.runConcurrently(fn)
	}

	// Wait until all batches received
	mockProc.awaitAsyncProcessing()

	if wantBatches != int(mockProc.batchCount) {
		t.Fatalf("Wanted %d batches, got %d", wantBatches, mockProc.batchCount)
	}
	if wantSpans != int(mockProc.spanCount) {
		t.Fatalf("Wanted %d spans, got %d", wantSpans, mockProc.spanCount)
	}
}

type mockConcurrentSpanProcessor struct {
	waitGroup  *sync.WaitGroup
	batchCount int32
	spanCount  int32
}

var _ SpanProcessor = (*mockConcurrentSpanProcessor)(nil)

func (p *mockConcurrentSpanProcessor) ProcessSpans(batch *agenttracepb.ExportTraceServiceRequest, spanFormat string) (uint64, error) {
	atomic.AddInt32(&p.batchCount, 1)
	atomic.AddInt32(&p.spanCount, int32(len(batch.Spans)))
	p.waitGroup.Done()
	return 0, nil
}

func newMockConcurrentSpanProcessor() *mockConcurrentSpanProcessor {
	return &mockConcurrentSpanProcessor{waitGroup: new(sync.WaitGroup)}
}
func (p *mockConcurrentSpanProcessor) runConcurrently(fn func()) {
	p.waitGroup.Add(1)
	go fn()
}

func (p *mockConcurrentSpanProcessor) awaitAsyncProcessing() {
	p.waitGroup.Wait()
}
