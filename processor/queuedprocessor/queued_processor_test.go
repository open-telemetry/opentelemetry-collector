// Copyright The OpenTelemetry Authors
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/collector/telemetry"
	"go.opentelemetry.io/collector/internal/data/testdata"
	"go.opentelemetry.io/collector/processor"
)

func TestQueuedProcessor_noEnqueueOnPermanentError(t *testing.T) {
	ctx := context.Background()
	td := testdata.GenerateTraceDataOneSpan()

	c := &waitGroupTraceConsumer{
		consumeTraceDataError: consumererror.Permanent(errors.New("bad data")),
	}

	cfg := generateDefaultConfig()
	cfg.NumWorkers = 1
	cfg.QueueSize = 2
	cfg.RetryOnFailure = true
	cfg.BackoffDelay = time.Hour
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	qp := newQueuedSpanProcessor(creationParams, c, cfg)

	require.NoError(t, qp.Start(context.Background(), componenttest.NewNopHost()))
	c.Add(1)
	require.Nil(t, qp.ConsumeTraces(ctx, td))
	c.Wait()
	<-time.After(50 * time.Millisecond)

	require.Zero(t, qp.queue.Size())

	c.consumeTraceDataError = errors.New("transient error")
	c.Add(1)
	// This is asynchronous so it should just enqueue, no errors expected.
	require.Nil(t, qp.ConsumeTraces(ctx, td))
	c.Wait()
	<-time.After(50 * time.Millisecond)

	require.Equal(t, 1, qp.queue.Size())
}

type waitGroupTraceConsumer struct {
	sync.WaitGroup
	consumeTraceDataError error
}

var _ consumer.TraceConsumer = (*waitGroupTraceConsumer)(nil)

func (c *waitGroupTraceConsumer) ConsumeTraces(_ context.Context, _ pdata.Traces) error {
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
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	qp := newQueuedSpanProcessor(creationParams, mockProc, generateDefaultConfig())
	require.NoError(t, qp.Start(context.Background(), componenttest.NewNopHost()))
	goFn := func(td pdata.Traces) {
		qp.ConsumeTraces(context.Background(), td)
	}

	wantBatches := 10
	wantSpans := 20
	for i := 0; i < wantBatches; i++ {
		td := testdata.GenerateTraceDataTwoSpansSameResource()
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

var _ consumer.TraceConsumer = (*mockConcurrentSpanProcessor)(nil)

func (p *mockConcurrentSpanProcessor) ConsumeTraces(_ context.Context, td pdata.Traces) error {
	atomic.AddInt32(&p.batchCount, 1)
	atomic.AddInt32(&p.spanCount, int32(td.SpanCount()))
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
