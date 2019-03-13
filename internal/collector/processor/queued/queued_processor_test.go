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

package queued

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/internal/collector/processor"
)

func TestQueueProcessorHappyPath(t *testing.T) {
	mockProc := newMockConcurrentSpanProcessor()
	qp := NewQueuedSpanProcessor(mockProc)
	goFn := func(td data.TraceData) {
		qp.ProcessSpans(context.Background(), td)
	}

	spans := []*tracepb.Span{{}}
	wantBatches := 10
	wantSpans := 0
	for i := 0; i < wantBatches; i++ {
		td := data.TraceData{
			Spans:        spans,
			SourceFormat: "oc_trace",
		}
		wantSpans += len(spans)
		spans = append(spans, &tracepb.Span{})
		fn := func() { goFn(td) }
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

var _ processor.SpanProcessor = (*mockConcurrentSpanProcessor)(nil)

func (p *mockConcurrentSpanProcessor) ProcessSpans(ctx context.Context, td data.TraceData) error {
	atomic.AddInt32(&p.batchCount, 1)
	atomic.AddInt32(&p.spanCount, int32(len(td.Spans)))
	p.waitGroup.Done()
	return nil
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
