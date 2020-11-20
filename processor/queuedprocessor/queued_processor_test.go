// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
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
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
	"go.opentelemetry.io/collector/processor"
)

func TestTraceQueueProcessor_NoEnqueueOnPermanentError(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	td := testdata.GenerateTraceDataOneSpan()

	mockP := newMockConcurrentSpanProcessor()
	mockP.updateError(consumererror.Permanent(errors.New("bad data")))

	cfg := createDefaultConfig().(*Config)
	cfg.RetryOnFailure = true
	cfg.BackoffDelay = time.Hour
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}

	qp := newQueuedTracesProcessor(creationParams, mockP, cfg)
	require.NoError(t, qp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		mockP.stop()
		assert.NoError(t, qp.Shutdown(context.Background()))
	})

	mockP.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, qp.ConsumeTraces(context.Background(), td))
	})
	mockP.awaitAsyncProcessing()
	<-time.After(200 * time.Millisecond)
	require.Zero(t, qp.queue.Size())
	obsreporttest.CheckProcessorTracesViews(t, cfg.Name(), 1, 0, 1)
}

func TestTraceQueueProcessor_EnqueueOnNoRetry(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	td := testdata.GenerateTraceDataOneSpan()

	mockP := newMockConcurrentSpanProcessor()
	mockP.updateError(errors.New("transient error"))

	cfg := createDefaultConfig().(*Config)
	cfg.RetryOnFailure = false
	cfg.BackoffDelay = 0
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}

	qp := newQueuedTracesProcessor(creationParams, mockP, cfg)
	require.NoError(t, qp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		mockP.stop()
		assert.NoError(t, qp.Shutdown(context.Background()))
	})

	mockP.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, qp.ConsumeTraces(context.Background(), td))
	})
	mockP.awaitAsyncProcessing()
	<-time.After(200 * time.Millisecond)
	require.Zero(t, qp.queue.Size())
	obsreporttest.CheckProcessorTracesViews(t, cfg.Name(), 1, 0, 1)
}

func TestTraceQueueProcessor_PartialError(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	partialErr := consumererror.PartialTracesError(errors.New("some error"), testdata.GenerateTraceDataOneSpan())
	td := testdata.GenerateTraceDataTwoSpansSameResource()

	mockP := newMockConcurrentSpanProcessor()
	mockP.updateError(partialErr)

	cfg := createDefaultConfig().(*Config)
	cfg.NumWorkers = 1
	cfg.RetryOnFailure = true
	cfg.BackoffDelay = time.Second

	qp := newQueuedTracesProcessor(component.ProcessorCreateParams{Logger: zap.NewNop()}, mockP, cfg)
	require.NoError(t, qp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		mockP.stop()
		assert.NoError(t, qp.Shutdown(context.Background()))
	})

	mockP.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, qp.ConsumeTraces(context.Background(), td))
	})
	mockP.awaitAsyncProcessing()
	// There is a small race condition in this test, but expect to execute this in less than 1 second.
	mockP.updateError(nil)
	mockP.waitGroup.Add(1)
	mockP.awaitAsyncProcessing()

	mockP.checkNumBatches(t, 2)
	mockP.checkNumSpans(t, 2+1)

	obsreporttest.CheckProcessorTracesViews(t, cfg.Name(), 2, 0, 0)
}

func TestTraceQueueProcessor_EnqueueOnError(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	td := testdata.GenerateTraceDataOneSpan()

	mockP := newMockConcurrentSpanProcessor()
	mockP.updateError(errors.New("transient error"))

	cfg := createDefaultConfig().(*Config)
	cfg.NumWorkers = 1
	cfg.QueueSize = 1
	cfg.RetryOnFailure = true
	cfg.BackoffDelay = time.Hour
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}

	qp := newQueuedTracesProcessor(creationParams, mockP, cfg)
	require.NoError(t, qp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		mockP.stop()
		assert.NoError(t, qp.Shutdown(context.Background()))
	})

	mockP.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, qp.ConsumeTraces(context.Background(), td))
	})
	mockP.awaitAsyncProcessing()
	<-time.After(200 * time.Millisecond)
	require.Equal(t, 1, qp.queue.Size())

	mockP.run(func() {
		// The queue is full, cannot enqueue other item
		require.Error(t, qp.ConsumeTraces(context.Background(), td))
	})
	obsreporttest.CheckProcessorTracesViews(t, cfg.Name(), 1, 1, 0)
}

func TestMetricsQueueProcessor_NoEnqueueOnPermanentError(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	md := testdata.GenerateMetricsTwoMetrics()

	mockP := newMockConcurrentSpanProcessor()
	mockP.updateError(consumererror.Permanent(errors.New("bad data")))

	cfg := createDefaultConfig().(*Config)
	cfg.RetryOnFailure = true
	cfg.BackoffDelay = time.Hour
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}

	qp := newQueuedMetricsProcessor(creationParams, mockP, cfg)
	require.NoError(t, qp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		mockP.stop()
		assert.NoError(t, qp.Shutdown(context.Background()))
	})

	mockP.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, qp.ConsumeMetrics(context.Background(), md))
	})
	mockP.awaitAsyncProcessing()
	<-time.After(200 * time.Millisecond)
	require.Zero(t, qp.queue.Size())
	obsreporttest.CheckProcessorMetricsViews(t, cfg.Name(), 4, 0, 4)
}

func TestMetricsQueueProcessor_NoEnqueueOnNoRetry(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	md := testdata.GenerateMetricsTwoMetrics()

	mockP := newMockConcurrentSpanProcessor()
	mockP.updateError(errors.New("transient error"))

	cfg := createDefaultConfig().(*Config)
	cfg.RetryOnFailure = false
	cfg.BackoffDelay = 0
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}

	qp := newQueuedMetricsProcessor(creationParams, mockP, cfg)
	require.NoError(t, qp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		mockP.stop()
		assert.NoError(t, qp.Shutdown(context.Background()))
	})

	mockP.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, qp.ConsumeMetrics(context.Background(), md))
	})
	mockP.awaitAsyncProcessing()
	<-time.After(200 * time.Millisecond)
	require.Zero(t, qp.queue.Size())
	obsreporttest.CheckProcessorMetricsViews(t, cfg.Name(), 4, 0, 4)
}

func TestMetricsQueueProcessor_EnqueueOnError(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	md := testdata.GenerateMetricsTwoMetrics()

	mockP := newMockConcurrentSpanProcessor()
	mockP.updateError(errors.New("transient error"))

	cfg := createDefaultConfig().(*Config)
	cfg.NumWorkers = 1
	cfg.QueueSize = 1
	cfg.RetryOnFailure = true
	cfg.BackoffDelay = time.Hour
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}

	qp := newQueuedMetricsProcessor(creationParams, mockP, cfg)
	require.NoError(t, qp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		mockP.stop()
		assert.NoError(t, qp.Shutdown(context.Background()))
	})

	mockP.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, qp.ConsumeMetrics(context.Background(), md))
	})
	mockP.awaitAsyncProcessing()
	<-time.After(200 * time.Millisecond)
	require.Equal(t, 1, qp.queue.Size())

	mockP.run(func() {
		// The queue is full, cannot enqueue other item
		require.Error(t, qp.ConsumeMetrics(context.Background(), md))
	})
	obsreporttest.CheckProcessorMetricsViews(t, cfg.Name(), 4, 4, 0)
}

func TestTraceQueueProcessorHappyPath(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	views := processor.MetricViews()
	assert.NoError(t, view.Register(views...))
	defer view.Unregister(views...)

	mockP := newMockConcurrentSpanProcessor()
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	cfg := createDefaultConfig().(*Config)
	qp := newQueuedTracesProcessor(creationParams, mockP, cfg)
	require.NoError(t, qp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		mockP.stop()
		assert.NoError(t, qp.Shutdown(context.Background()))
	})

	wantBatches := 10
	wantSpans := 20
	for i := 0; i < wantBatches; i++ {
		td := testdata.GenerateTraceDataTwoSpansSameResource()
		mockP.run(func() {
			require.NoError(t, qp.ConsumeTraces(context.Background(), td))
		})
	}

	// Wait until all batches received
	mockP.awaitAsyncProcessing()

	mockP.checkNumBatches(t, wantBatches)
	mockP.checkNumSpans(t, wantSpans)

	droppedView, err := findViewNamed(views, "processor/"+processor.StatDroppedSpanCount.Name())
	require.NoError(t, err)

	data, err := view.RetrieveData(droppedView.Name)
	require.NoError(t, err)
	require.Len(t, data, 1)
	assert.Equal(t, 0.0, data[0].Data.(*view.SumData).Value)

	data, err = view.RetrieveData("processor/" + processor.StatTraceBatchesDroppedCount.Name())
	require.NoError(t, err)
	assert.Equal(t, 0.0, data[0].Data.(*view.SumData).Value)
	obsreporttest.CheckProcessorTracesViews(t, cfg.Name(), int64(wantSpans), 0, 0)
}

func TestMetricsQueueProcessorHappyPath(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	mockP := newMockConcurrentSpanProcessor()
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	cfg := createDefaultConfig().(*Config)
	qp := newQueuedMetricsProcessor(creationParams, mockP, cfg)
	require.NoError(t, qp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, qp.Shutdown(context.Background()))
	})

	wantBatches := 10
	wantMetricPoints := 2 * 20
	for i := 0; i < wantBatches; i++ {
		md := testdata.GenerateMetricsTwoMetrics()
		mockP.run(func() {
			require.NoError(t, qp.ConsumeMetrics(context.Background(), md))
		})
	}

	// Wait until all batches received
	mockP.awaitAsyncProcessing()

	mockP.checkNumBatches(t, wantBatches)
	mockP.checkNumPoints(t, wantMetricPoints)
	obsreporttest.CheckProcessorMetricsViews(t, cfg.Name(), int64(wantMetricPoints), 0, 0)
}

type mockConcurrentSpanProcessor struct {
	waitGroup         *sync.WaitGroup
	mu                sync.Mutex
	consumeError      error
	batchCount        int64
	spanCount         int64
	metricPointsCount int64
	stopped           int32
}

var _ consumer.TracesConsumer = (*mockConcurrentSpanProcessor)(nil)
var _ consumer.MetricsConsumer = (*mockConcurrentSpanProcessor)(nil)

func newMockConcurrentSpanProcessor() *mockConcurrentSpanProcessor {
	return &mockConcurrentSpanProcessor{waitGroup: new(sync.WaitGroup)}
}

func (p *mockConcurrentSpanProcessor) ConsumeTraces(_ context.Context, td pdata.Traces) error {
	if atomic.LoadInt32(&p.stopped) == 1 {
		return nil
	}
	atomic.AddInt64(&p.batchCount, 1)
	atomic.AddInt64(&p.spanCount, int64(td.SpanCount()))
	p.mu.Lock()
	defer p.mu.Unlock()
	defer p.waitGroup.Done()
	return p.consumeError
}

func (p *mockConcurrentSpanProcessor) ConsumeMetrics(_ context.Context, md pdata.Metrics) error {
	if atomic.LoadInt32(&p.stopped) == 1 {
		return nil
	}
	atomic.AddInt64(&p.batchCount, 1)
	_, mpc := md.MetricAndDataPointCount()
	atomic.AddInt64(&p.metricPointsCount, int64(mpc))
	p.mu.Lock()
	defer p.mu.Unlock()
	defer p.waitGroup.Done()
	return p.consumeError
}

func (p *mockConcurrentSpanProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: false}
}

func (p *mockConcurrentSpanProcessor) checkNumBatches(t *testing.T, want int) {
	assert.EqualValues(t, want, atomic.LoadInt64(&p.batchCount))
}

func (p *mockConcurrentSpanProcessor) checkNumSpans(t *testing.T, want int) {
	assert.EqualValues(t, want, atomic.LoadInt64(&p.spanCount))
}

func (p *mockConcurrentSpanProcessor) checkNumPoints(t *testing.T, want int) {
	assert.EqualValues(t, want, atomic.LoadInt64(&p.metricPointsCount))
}

func (p *mockConcurrentSpanProcessor) updateError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.consumeError = err
}

func (p *mockConcurrentSpanProcessor) run(fn func()) {
	p.waitGroup.Add(1)
	fn()
}

func (p *mockConcurrentSpanProcessor) awaitAsyncProcessing() {
	p.waitGroup.Wait()
}

func (p *mockConcurrentSpanProcessor) stop() {
	atomic.StoreInt32(&p.stopped, 1)
}

func findViewNamed(views []*view.View, name string) (*view.View, error) {
	for _, v := range views {
		if v.Name == name {
			return v, nil
		}
	}
	return nil, fmt.Errorf("view %s not found", name)
}
