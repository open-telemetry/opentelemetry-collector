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

package exporterhelper

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
)

func TestQueuedRetry_DropOnPermanentError(t *testing.T) {
	qCfg := DefaultQueueSettings()
	rCfg := DefaultRetrySettings()
	be := newBaseExporter(defaultExporterCfg, zap.NewNop(), WithRetry(rCfg), WithQueue(qCfg))
	ocs := newObservabilityConsumerSender(be.qrSender.consumerSender)
	be.qrSender.consumerSender = ocs
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	mockR := newMockRequest(context.Background(), 2, consumererror.Permanent(errors.New("bad data")))
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		droppedItems, err := be.sender.send(mockR)
		require.NoError(t, err)
		assert.Equal(t, 0, droppedItems)
	})
	ocs.awaitAsyncProcessing()
	// In the newMockConcurrentExporter we count requests and items even for failed requests
	mockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 0)
	ocs.checkDroppedItemsCount(t, 2)
}

func TestQueuedRetry_DropOnNoRetry(t *testing.T) {
	qCfg := DefaultQueueSettings()
	rCfg := DefaultRetrySettings()
	rCfg.Enabled = false
	be := newBaseExporter(defaultExporterCfg, zap.NewNop(), WithRetry(rCfg), WithQueue(qCfg))
	ocs := newObservabilityConsumerSender(be.qrSender.consumerSender)
	be.qrSender.consumerSender = ocs
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	mockR := newMockRequest(context.Background(), 2, errors.New("transient error"))
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		droppedItems, err := be.sender.send(mockR)
		require.NoError(t, err)
		assert.Equal(t, 0, droppedItems)
	})
	ocs.awaitAsyncProcessing()
	// In the newMockConcurrentExporter we count requests and items even for failed requests
	mockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 0)
	ocs.checkDroppedItemsCount(t, 2)
}

func TestQueuedRetry_PartialError(t *testing.T) {
	qCfg := DefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := DefaultRetrySettings()
	rCfg.InitialInterval = 0
	be := newBaseExporter(defaultExporterCfg, zap.NewNop(), WithRetry(rCfg), WithQueue(qCfg))
	ocs := newObservabilityConsumerSender(be.qrSender.consumerSender)
	be.qrSender.consumerSender = ocs
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	partialErr := consumererror.PartialTracesError(errors.New("some error"), testdata.GenerateTraceDataOneSpan())
	mockR := newMockRequest(context.Background(), 2, partialErr)
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		droppedItems, err := be.sender.send(mockR)
		require.NoError(t, err)
		assert.Equal(t, 0, droppedItems)
	})
	ocs.awaitAsyncProcessing()

	// In the newMockConcurrentExporter we count requests and items even for failed requests
	mockR.checkNumRequests(t, 2)
	ocs.checkSendItemsCount(t, 2)
	ocs.checkDroppedItemsCount(t, 0)
}

func TestQueuedRetry_StopWhileWaiting(t *testing.T) {
	qCfg := DefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := DefaultRetrySettings()
	be := newBaseExporter(defaultExporterCfg, zap.NewNop(), WithRetry(rCfg), WithQueue(qCfg))
	ocs := newObservabilityConsumerSender(be.qrSender.consumerSender)
	be.qrSender.consumerSender = ocs
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	firstMockR := newMockRequest(context.Background(), 2, errors.New("transient error"))
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		droppedItems, err := be.sender.send(firstMockR)
		require.NoError(t, err)
		assert.Equal(t, 0, droppedItems)
	})

	// Enqueue another request to ensure when calling shutdown we drain the queue.
	secondMockR := newMockRequest(context.Background(), 3, nil)
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		droppedItems, err := be.sender.send(secondMockR)
		require.NoError(t, err)
		assert.Equal(t, 0, droppedItems)
	})

	assert.NoError(t, be.Shutdown(context.Background()))

	// TODO: Ensure that queue is drained, and uncomment the next 3 lines.
	//  https://github.com/jaegertracing/jaeger/pull/2349
	firstMockR.checkNumRequests(t, 1)
	// secondMockR.checkNumRequests(t, 1)
	// ocs.checkSendItemsCount(t, 3)
	ocs.checkDroppedItemsCount(t, 2)
	// require.Zero(t, be.qrSender.queue.Size())
}

func TestQueuedRetry_DoNotPreserveCancellation(t *testing.T) {
	qCfg := DefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := DefaultRetrySettings()
	be := newBaseExporter(defaultExporterCfg, zap.NewNop(), WithRetry(rCfg), WithQueue(qCfg))
	ocs := newObservabilityConsumerSender(be.qrSender.consumerSender)
	be.qrSender.consumerSender = ocs
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	ctx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()
	mockR := newMockRequest(ctx, 2, nil)
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		droppedItems, err := be.sender.send(mockR)
		require.NoError(t, err)
		assert.Equal(t, 0, droppedItems)
	})
	ocs.awaitAsyncProcessing()

	mockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 2)
	ocs.checkDroppedItemsCount(t, 0)
	require.Zero(t, be.qrSender.queue.Size())
}

func TestQueuedRetry_MaxElapsedTime(t *testing.T) {
	qCfg := DefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := DefaultRetrySettings()
	rCfg.InitialInterval = time.Millisecond
	rCfg.MaxElapsedTime = 100 * time.Millisecond
	be := newBaseExporter(defaultExporterCfg, zap.NewNop(), WithRetry(rCfg), WithQueue(qCfg))
	ocs := newObservabilityConsumerSender(be.qrSender.consumerSender)
	be.qrSender.consumerSender = ocs
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	ocs.run(func() {
		// Add an item that will always fail.
		droppedItems, err := be.sender.send(newErrorRequest(context.Background()))
		require.NoError(t, err)
		assert.Equal(t, 0, droppedItems)
	})

	mockR := newMockRequest(context.Background(), 2, nil)
	start := time.Now()
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		droppedItems, err := be.sender.send(mockR)
		require.NoError(t, err)
		assert.Equal(t, 0, droppedItems)
	})
	ocs.awaitAsyncProcessing()

	// We should ensure that we wait for more than 50ms but less than 150ms (50% less and 50% more than max elapsed).
	waitingTime := time.Since(start)
	assert.Less(t, 50*time.Millisecond, waitingTime)
	assert.Less(t, waitingTime, 150*time.Millisecond)

	// In the newMockConcurrentExporter we count requests and items even for failed requests.
	mockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 2)
	ocs.checkDroppedItemsCount(t, 7)
	require.Zero(t, be.qrSender.queue.Size())
}

func TestQueuedRetry_ThrottleError(t *testing.T) {
	qCfg := DefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := DefaultRetrySettings()
	rCfg.InitialInterval = 10 * time.Millisecond
	be := newBaseExporter(defaultExporterCfg, zap.NewNop(), WithRetry(rCfg), WithQueue(qCfg))
	ocs := newObservabilityConsumerSender(be.qrSender.consumerSender)
	be.qrSender.consumerSender = ocs
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	mockR := newMockRequest(context.Background(), 2, NewThrottleRetry(errors.New("throttle error"), 100*time.Millisecond))
	start := time.Now()
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		droppedItems, err := be.sender.send(mockR)
		require.NoError(t, err)
		assert.Equal(t, 0, droppedItems)
	})
	ocs.awaitAsyncProcessing()

	// The initial backoff is 10ms, but because of the throttle this should wait at least 100ms.
	assert.True(t, 100*time.Millisecond < time.Since(start))

	mockR.checkNumRequests(t, 2)
	ocs.checkSendItemsCount(t, 2)
	ocs.checkDroppedItemsCount(t, 0)
	require.Zero(t, be.qrSender.queue.Size())
}

func TestQueuedRetry_RetryOnError(t *testing.T) {
	qCfg := DefaultQueueSettings()
	qCfg.NumConsumers = 1
	qCfg.QueueSize = 1
	rCfg := DefaultRetrySettings()
	rCfg.InitialInterval = 0
	be := newBaseExporter(defaultExporterCfg, zap.NewNop(), WithRetry(rCfg), WithQueue(qCfg))
	ocs := newObservabilityConsumerSender(be.qrSender.consumerSender)
	be.qrSender.consumerSender = ocs
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	mockR := newMockRequest(context.Background(), 2, errors.New("transient error"))
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		droppedItems, err := be.sender.send(mockR)
		require.NoError(t, err)
		assert.Equal(t, 0, droppedItems)
	})
	ocs.awaitAsyncProcessing()

	// In the newMockConcurrentExporter we count requests and items even for failed requests
	mockR.checkNumRequests(t, 2)
	ocs.checkSendItemsCount(t, 2)
	ocs.checkDroppedItemsCount(t, 0)
	require.Zero(t, be.qrSender.queue.Size())
}

func TestQueuedRetry_DropOnFull(t *testing.T) {
	qCfg := DefaultQueueSettings()
	qCfg.QueueSize = 0
	rCfg := DefaultRetrySettings()
	be := newBaseExporter(defaultExporterCfg, zap.NewNop(), WithRetry(rCfg), WithQueue(qCfg))
	ocs := newObservabilityConsumerSender(be.qrSender.consumerSender)
	be.qrSender.consumerSender = ocs
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})
	droppedItems, err := be.sender.send(newMockRequest(context.Background(), 2, errors.New("transient error")))
	require.Error(t, err)
	assert.Equal(t, 2, droppedItems)
}

func TestQueuedRetryHappyPath(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	qCfg := DefaultQueueSettings()
	rCfg := DefaultRetrySettings()
	be := newBaseExporter(defaultExporterCfg, zap.NewNop(), WithRetry(rCfg), WithQueue(qCfg))
	ocs := newObservabilityConsumerSender(be.qrSender.consumerSender)
	be.qrSender.consumerSender = ocs
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	wantRequests := 10
	reqs := make([]*mockRequest, 0, 10)
	for i := 0; i < wantRequests; i++ {
		ocs.run(func() {
			req := newMockRequest(context.Background(), 2, nil)
			reqs = append(reqs, req)
			droppedItems, err := be.sender.send(req)
			require.NoError(t, err)
			assert.Equal(t, 0, droppedItems)
		})
	}

	// Wait until all batches received
	ocs.awaitAsyncProcessing()

	require.Len(t, reqs, wantRequests)
	for _, req := range reqs {
		req.checkNumRequests(t, 1)
	}

	ocs.checkSendItemsCount(t, 2*wantRequests)
	ocs.checkDroppedItemsCount(t, 0)
}

func TestNoCancellationContext(t *testing.T) {
	deadline := time.Now().Add(1 * time.Second)
	ctx, cancelFunc := context.WithDeadline(context.Background(), deadline)
	cancelFunc()
	require.Error(t, ctx.Err())
	d, ok := ctx.Deadline()
	require.True(t, ok)
	require.Equal(t, deadline, d)

	nctx := noCancellationContext{Context: ctx}
	assert.NoError(t, nctx.Err())
	d, ok = nctx.Deadline()
	assert.False(t, ok)
	assert.True(t, d.IsZero())
}

type mockErrorRequest struct {
	baseRequest
}

func (mer *mockErrorRequest) export(_ context.Context) (int, error) {
	return 0, errors.New("transient error")
}

func (mer *mockErrorRequest) onPartialError(consumererror.PartialError) request {
	return mer
}

func (mer *mockErrorRequest) count() int {
	return 7
}

func newErrorRequest(ctx context.Context) request {
	return &mockErrorRequest{
		baseRequest: baseRequest{ctx: ctx},
	}
}

type mockRequest struct {
	baseRequest
	cnt          int
	mu           sync.Mutex
	consumeError error
	requestCount *int64
}

func (m *mockRequest) export(ctx context.Context) (int, error) {
	atomic.AddInt64(m.requestCount, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	err := m.consumeError
	m.consumeError = nil
	if err != nil {
		return m.cnt, err
	}
	// Respond like gRPC/HTTP, if context is cancelled, return error
	return 0, ctx.Err()
}

func (m *mockRequest) onPartialError(consumererror.PartialError) request {
	return &mockRequest{
		baseRequest:  m.baseRequest,
		cnt:          1,
		consumeError: nil,
		requestCount: m.requestCount,
	}
}

func (m *mockRequest) checkNumRequests(t *testing.T, want int) {
	assert.Eventually(t, func() bool {
		return int64(want) == atomic.LoadInt64(m.requestCount)
	}, time.Second, 1*time.Millisecond)
}

func (m *mockRequest) count() int {
	return m.cnt
}

func newMockRequest(ctx context.Context, cnt int, consumeError error) *mockRequest {
	return &mockRequest{
		baseRequest:  baseRequest{ctx: ctx},
		cnt:          cnt,
		consumeError: consumeError,
		requestCount: new(int64),
	}
}

type observabilityConsumerSender struct {
	waitGroup         *sync.WaitGroup
	sentItemsCount    int64
	droppedItemsCount int64
	nextSender        requestSender
}

func newObservabilityConsumerSender(nextSender requestSender) *observabilityConsumerSender {
	return &observabilityConsumerSender{waitGroup: new(sync.WaitGroup), nextSender: nextSender}
}

func (ocs *observabilityConsumerSender) send(req request) (int, error) {
	dic, err := ocs.nextSender.send(req)
	atomic.AddInt64(&ocs.sentItemsCount, int64(req.count()-dic))
	atomic.AddInt64(&ocs.droppedItemsCount, int64(dic))
	ocs.waitGroup.Done()
	return dic, err
}

func (ocs *observabilityConsumerSender) run(fn func()) {
	ocs.waitGroup.Add(1)
	fn()
}

func (ocs *observabilityConsumerSender) awaitAsyncProcessing() {
	ocs.waitGroup.Wait()
}

func (ocs *observabilityConsumerSender) checkSendItemsCount(t *testing.T, want int) {
	assert.EqualValues(t, want, atomic.LoadInt64(&ocs.sentItemsCount))
}

func (ocs *observabilityConsumerSender) checkDroppedItemsCount(t *testing.T, want int) {
	assert.EqualValues(t, want, atomic.LoadInt64(&ocs.droppedItemsCount))
}
