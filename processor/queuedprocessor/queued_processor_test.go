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

package queuedprocessor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/component/componenttest"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumererror"
	"github.com/open-telemetry/opentelemetry-collector/internal/collector/telemetry"
	"github.com/open-telemetry/opentelemetry-collector/processor"
)

func TestQueuedProcessor_noEnqueueOnPermanentError(t *testing.T) {
	ctx := context.Background()
	td := consumerdata.TraceData{
		Spans: make([]*tracepb.Span, 7),
	}

	c := &waitGroupTraceConsumer{
		consumeTraceDataError: consumererror.Permanent(errors.New("bad data")),
	}

	cfg := generateDefaultConfig()
	cfg.NumWorkers = 1
	cfg.QueueSize = 2
	cfg.RetryOnFailure = true
	cfg.BackoffDelay = time.Hour
	qp := newQueuedSpanProcessor(zap.NewNop(), c, cfg)

	require.NoError(t, qp.Start(context.Background(), componenttest.NewNopHost()))
	c.Add(1)
	require.Nil(t, qp.ConsumeTraceData(ctx, td))
	c.Wait()
	<-time.After(50 * time.Millisecond)

	require.Zero(t, qp.queue.Size())

	c.consumeTraceDataError = errors.New("transient error")
	c.Add(1)
	// This is asynchronous so it should just enqueue, no errors expected.
	require.Nil(t, qp.ConsumeTraceData(ctx, td))
	c.Wait()
	<-time.After(50 * time.Millisecond)

	require.Equal(t, 1, qp.queue.Size())
}

type waitGroupTraceConsumer struct {
	sync.WaitGroup
	consumeTraceDataError error
}

var _ consumer.TraceConsumerOld = (*waitGroupTraceConsumer)(nil)

func (c *waitGroupTraceConsumer) ConsumeTraceData(_ context.Context, _ consumerdata.TraceData) error {
	defer c.Done()
	return c.consumeTraceDataError
}

func (c *waitGroupTraceConsumer) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: false}
}

func findViewNamed(views []*view.View, name string) (*view.View, error) {
	for _, v := range views {
		if v.Name == name {
			return v, nil
		}
	}
	return nil, fmt.Errorf("view %s not found", name)
}

func TestQueueProcessorHappyPath(t *testing.T) {
	views := processor.MetricViews(telemetry.Detailed)
	view.Register(views...)
	defer view.Unregister(views...)

	mockProc := newMockConcurrentSpanProcessor()
	qp := newQueuedSpanProcessor(zap.NewNop(), mockProc, generateDefaultConfig())
	require.NoError(t, qp.Start(context.Background(), componenttest.NewNopHost()))
	goFn := func(td consumerdata.TraceData) {
		qp.ConsumeTraceData(context.Background(), td)
	}

	spans := []*tracepb.Span{{}}
	wantBatches := 10
	wantSpans := 0
	for i := 0; i < wantBatches; i++ {
		td := consumerdata.TraceData{
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

	require.Equal(t, wantBatches, int(mockProc.batchCount), "Incorrect batches count")
	require.Equal(t, wantSpans, int(mockProc.spanCount), "Incorrect batches spans")

	droppedView, err := findViewNamed(views, processor.StatDroppedSpanCount.Name())
	require.NoError(t, err)

	data, err := view.RetrieveData(droppedView.Name)
	require.NoError(t, err)
	require.Len(t, data, 1)
	assert.Equal(t, 0.0, data[0].Data.(*view.SumData).Value)

	data, err = view.RetrieveData(processor.StatTraceBatchesDroppedCount.Name())
	require.NoError(t, err)
	assert.Equal(t, 0.0, data[0].Data.(*view.SumData).Value)
}

type mockConcurrentSpanProcessor struct {
	waitGroup  *sync.WaitGroup
	batchCount int32
	spanCount  int32
}

var _ consumer.TraceConsumerOld = (*mockConcurrentSpanProcessor)(nil)

func (p *mockConcurrentSpanProcessor) ConsumeTraceData(_ context.Context, td consumerdata.TraceData) error {
	atomic.AddInt32(&p.batchCount, 1)
	atomic.AddInt32(&p.spanCount, int32(len(td.Spans)))
	p.waitGroup.Done()
	return nil
}

func (p *mockConcurrentSpanProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: false}
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
