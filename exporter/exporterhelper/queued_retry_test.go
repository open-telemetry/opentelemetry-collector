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

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/internal/data/testdata"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
)

func TestQueuedRetry_DropOnPermanentError(t *testing.T) {
	mockP := newMockConcurrentExporter()
	mockP.updateError(consumererror.Permanent(errors.New("bad data")))

	qCfg := CreateDefaultQueueSettings()
	rCfg := CreateDefaultRetrySettings()
	rCfg.Disabled = false
	rCfg.InitialInterval = 30 * time.Second
	be := newBaseExporter(defaultExporterCfg, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		mockP.stop()
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	mockP.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		droppedItems, err := be.sender.send(newMockRequest(context.Background(), 2, mockP))
		require.NoError(t, err)
		assert.Equal(t, 0, droppedItems)
	})
	mockP.awaitAsyncProcessing()
	<-time.After(200 * time.Millisecond)
	require.Zero(t, be.qrSender.queue.Size())
}

func TestQueuedRetry_DropOnNoRetry(t *testing.T) {
	mockP := newMockConcurrentExporter()
	mockP.updateError(errors.New("transient error"))

	qCfg := CreateDefaultQueueSettings()
	rCfg := CreateDefaultRetrySettings()
	rCfg.Disabled = true
	be := newBaseExporter(defaultExporterCfg, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		mockP.stop()
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	mockP.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		droppedItems, err := be.sender.send(newMockRequest(context.Background(), 2, mockP))
		require.NoError(t, err)
		assert.Equal(t, 0, droppedItems)
	})
	mockP.awaitAsyncProcessing()
	<-time.After(200 * time.Millisecond)
	require.Zero(t, be.qrSender.queue.Size())
}

func TestQueuedRetry_PartialError(t *testing.T) {
	partialErr := consumererror.PartialTracesError(errors.New("some error"), testdata.GenerateTraceDataOneSpan())
	mockP := newMockConcurrentExporter()
	mockP.updateError(partialErr)

	qCfg := CreateDefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := CreateDefaultRetrySettings()
	rCfg.Disabled = false
	rCfg.InitialInterval = 30 * time.Second
	be := newBaseExporter(defaultExporterCfg, WithRetry(rCfg), WithQueue(qCfg))
	ocs := &observabilityConsumerSender{nextSender: be.qrSender.consumerSender}
	be.qrSender.consumerSender = ocs
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		mockP.stop()
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	mockP.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		droppedItems, err := be.sender.send(newMockRequest(context.Background(), 2, mockP))
		require.NoError(t, err)
		assert.Equal(t, 0, droppedItems)
	})
	mockP.awaitAsyncProcessing()
	// There is a small race condition in this test, but expect to execute this in less than 1 second.
	mockP.updateError(nil)
	mockP.waitGroup.Add(1)
	mockP.awaitAsyncProcessing()

	// In the newMockConcurrentExporter we count requests and items even for failed requests
	mockP.checkNumRequests(t, 2)
	ocs.checkSendItemsCount(t, 2)
	ocs.checkDroppedItemsCount(t, 0)
	require.Zero(t, be.qrSender.queue.Size())
}

func TestQueuedRetry_StopWhileWaiting(t *testing.T) {
	mockP := newMockConcurrentExporter()
	mockP.updateError(errors.New("transient error"))

	qCfg := CreateDefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := CreateDefaultRetrySettings()
	rCfg.Disabled = false
	rCfg.InitialInterval = 30 * time.Minute
	be := newBaseExporter(defaultExporterCfg, WithRetry(rCfg), WithQueue(qCfg))
	ocs := &observabilityConsumerSender{nextSender: be.qrSender.consumerSender}
	be.qrSender.consumerSender = ocs
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	mockP.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		droppedItems, err := be.sender.send(newMockRequest(context.Background(), 2, mockP))
		require.NoError(t, err)
		assert.Equal(t, 0, droppedItems)
	})
	mockP.awaitAsyncProcessing()

	// Enqueue another request to ensure when calling shutdown we drain the queue.
	mockP.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		droppedItems, err := be.sender.send(newMockRequest(context.Background(), 3, mockP))
		require.NoError(t, err)
		assert.Equal(t, 0, droppedItems)
	})

	mockP.stop()
	assert.NoError(t, be.Shutdown(context.Background()))

	mockP.checkNumRequests(t, 1)
	// TODO: Ensure that queue is drained, and uncomment the next 3 lines.
	//  https://github.com/jaegertracing/jaeger/pull/2349
	// ocs.checkSendItemsCount(t, 3)
	ocs.checkDroppedItemsCount(t, 2)
	// require.Zero(t, be.qrSender.queue.Size())
}

func TestQueuedRetry_PreserveCancellation(t *testing.T) {
	mockP := newMockConcurrentExporter()
	mockP.updateError(errors.New("transient error"))

	qCfg := CreateDefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := CreateDefaultRetrySettings()
	rCfg.Disabled = false
	rCfg.InitialInterval = 30 * time.Second
	be := newBaseExporter(defaultExporterCfg, WithRetry(rCfg), WithQueue(qCfg))
	ocs := &observabilityConsumerSender{nextSender: be.qrSender.consumerSender}
	be.qrSender.consumerSender = ocs
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	ctx, cancelFunc := context.WithCancel(context.Background())
	start := time.Now()
	mockP.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		droppedItems, err := be.sender.send(newMockRequest(ctx, 2, mockP))
		require.NoError(t, err)
		assert.Equal(t, 0, droppedItems)
	})
	mockP.awaitAsyncProcessing()

	cancelFunc()

	mockP.checkNumRequests(t, 1)
	require.Zero(t, be.qrSender.queue.Size())

	// Stop should succeed and not retry.
	mockP.stop()
	assert.NoError(t, be.Shutdown(context.Background()))

	// We should ensure that we actually did not wait for the initial backoff (30 sec).
	assert.True(t, 5*time.Second > time.Since(start))

	// In the newMockConcurrentExporter we count requests and items even for failed requests.
	mockP.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 0)
	ocs.checkDroppedItemsCount(t, 2)
	require.Zero(t, be.qrSender.queue.Size())
}

func TestQueuedRetry_MaxElapsedTime(t *testing.T) {
	mockP := newMockConcurrentExporter()

	qCfg := CreateDefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := CreateDefaultRetrySettings()
	rCfg.Disabled = false
	rCfg.InitialInterval = 100 * time.Millisecond
	rCfg.MaxElapsedTime = 1 * time.Second
	be := newBaseExporter(defaultExporterCfg, WithRetry(rCfg), WithQueue(qCfg))
	ocs := &observabilityConsumerSender{nextSender: be.qrSender.consumerSender}
	be.qrSender.consumerSender = ocs
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		mockP.stop()
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	// Add an item that will always fail.
	droppedItems, err := be.sender.send(newErrorRequest(context.Background()))
	require.NoError(t, err)
	assert.Equal(t, 0, droppedItems)

	mockP.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		droppedItems, err := be.sender.send(newMockRequest(context.Background(), 2, mockP))
		require.NoError(t, err)
		assert.Equal(t, 0, droppedItems)
	})
	mockP.awaitAsyncProcessing()

	// In the newMockConcurrentExporter we count requests and items even for failed requests.
	mockP.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 2)
	ocs.checkDroppedItemsCount(t, 7)
	require.Zero(t, be.qrSender.queue.Size())
}

func TestQueuedRetry_ThrottleError(t *testing.T) {
	mockP := newMockConcurrentExporter()
	mockP.updateError(NewThrottleRetry(errors.New("throttle error"), 1*time.Second))

	qCfg := CreateDefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := CreateDefaultRetrySettings()
	rCfg.Disabled = false
	rCfg.InitialInterval = 100 * time.Millisecond
	be := newBaseExporter(defaultExporterCfg, WithRetry(rCfg), WithQueue(qCfg))
	ocs := &observabilityConsumerSender{nextSender: be.qrSender.consumerSender}
	be.qrSender.consumerSender = ocs
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		mockP.stop()
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	start := time.Now()
	mockP.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		droppedItems, err := be.sender.send(newMockRequest(context.Background(), 2, mockP))
		require.NoError(t, err)
		assert.Equal(t, 0, droppedItems)
	})
	mockP.awaitAsyncProcessing()
	// There is a small race condition in this test, but expect to execute this in less than 2 second.
	mockP.updateError(nil)
	mockP.waitGroup.Add(1)
	mockP.awaitAsyncProcessing()

	// The initial backoff is 100ms, but because of the throttle this should wait at least 1 seconds.
	assert.True(t, 1*time.Second < time.Since(start))

	mockP.checkNumRequests(t, 2)
	ocs.checkSendItemsCount(t, 2)
	ocs.checkDroppedItemsCount(t, 0)
	require.Zero(t, be.qrSender.queue.Size())
}

func TestQueuedRetry_RetryOnError(t *testing.T) {
	mockP := newMockConcurrentExporter()
	mockP.updateError(errors.New("transient error"))

	qCfg := CreateDefaultQueueSettings()
	qCfg.NumConsumers = 1
	qCfg.QueueSize = 1
	rCfg := CreateDefaultRetrySettings()
	rCfg.Disabled = false
	rCfg.InitialInterval = 2 * time.Second
	be := newBaseExporter(defaultExporterCfg, WithRetry(rCfg), WithQueue(qCfg))
	ocs := &observabilityConsumerSender{nextSender: be.qrSender.consumerSender}
	be.qrSender.consumerSender = ocs
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		mockP.stop()
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	mockP.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		droppedItems, err := be.sender.send(newMockRequest(context.Background(), 2, mockP))
		require.NoError(t, err)
		assert.Equal(t, 0, droppedItems)
	})
	mockP.awaitAsyncProcessing()
	ocs.checkSendItemsCount(t, 0)
	ocs.checkDroppedItemsCount(t, 0)

	// There is a small race condition in this test, but expect to execute this in less than 2 second.
	mockP.updateError(nil)
	mockP.waitGroup.Add(1)
	mockP.awaitAsyncProcessing()

	// In the newMockConcurrentExporter we count requests and items even for failed requests
	mockP.checkNumRequests(t, 2)
	ocs.checkSendItemsCount(t, 2)
	ocs.checkDroppedItemsCount(t, 0)
	require.Zero(t, be.qrSender.queue.Size())
}

func TestQueuedRetry_DropOnFull(t *testing.T) {
	mockP := newMockConcurrentExporter()
	mockP.updateError(errors.New("transient error"))

	qCfg := CreateDefaultQueueSettings()
	qCfg.QueueSize = 0
	rCfg := CreateDefaultRetrySettings()
	be := newBaseExporter(defaultExporterCfg, WithRetry(rCfg), WithQueue(qCfg))
	ocs := &observabilityConsumerSender{nextSender: be.qrSender.consumerSender}
	be.qrSender.consumerSender = ocs
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		mockP.stop()
		assert.NoError(t, be.Shutdown(context.Background()))
	})
	droppedItems, err := be.sender.send(newMockRequest(context.Background(), 2, mockP))
	require.Error(t, err)
	assert.Equal(t, 2, droppedItems)
}

func TestQueuedRetryHappyPath(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	mockP := newMockConcurrentExporter()
	qCfg := CreateDefaultQueueSettings()
	rCfg := CreateDefaultRetrySettings()
	be := newBaseExporter(defaultExporterCfg, WithRetry(rCfg), WithQueue(qCfg))
	ocs := &observabilityConsumerSender{nextSender: be.qrSender.consumerSender}
	be.qrSender.consumerSender = ocs
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		mockP.stop()
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	wantRequests := 10
	for i := 0; i < wantRequests; i++ {
		mockP.run(func() {
			droppedItems, err := be.sender.send(newMockRequest(context.Background(), 2, mockP))
			require.NoError(t, err)
			assert.Equal(t, 0, droppedItems)
		})
	}

	// Wait until all batches received
	mockP.awaitAsyncProcessing()

	mockP.checkNumRequests(t, wantRequests)
	ocs.checkSendItemsCount(t, 2*wantRequests)
	ocs.checkDroppedItemsCount(t, 0)
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
	cnt int
	mce *mockConcurrentExporter
}

func (m *mockRequest) export(_ context.Context) (int, error) {
	err := m.mce.export()
	if err != nil {
		return m.cnt, err
	}
	return 0, nil
}

func (m *mockRequest) onPartialError(consumererror.PartialError) request {
	return &mockRequest{
		baseRequest: m.baseRequest,
		cnt:         1,
		mce:         m.mce,
	}
}

func (m *mockRequest) count() int {
	return m.cnt
}

func newMockRequest(ctx context.Context, cnt int, mce *mockConcurrentExporter) request {
	return &mockRequest{
		baseRequest: baseRequest{ctx: ctx},
		cnt:         cnt,
		mce:         mce,
	}
}

type mockConcurrentExporter struct {
	waitGroup    *sync.WaitGroup
	mu           sync.Mutex
	consumeError error
	requestCount int64
	stopped      int32
}

func newMockConcurrentExporter() *mockConcurrentExporter {
	return &mockConcurrentExporter{waitGroup: new(sync.WaitGroup)}
}

func (p *mockConcurrentExporter) export() error {
	if atomic.LoadInt32(&p.stopped) == 1 {
		return nil
	}
	atomic.AddInt64(&p.requestCount, 1)
	p.mu.Lock()
	defer p.mu.Unlock()
	defer p.waitGroup.Done()
	return p.consumeError
}

func (p *mockConcurrentExporter) updateError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.consumeError = err
}

func (p *mockConcurrentExporter) checkNumRequests(t *testing.T, want int) {
	assert.EqualValues(t, want, atomic.LoadInt64(&p.requestCount))
}

func (p *mockConcurrentExporter) run(fn func()) {
	p.waitGroup.Add(1)
	fn()
}

func (p *mockConcurrentExporter) awaitAsyncProcessing() {
	p.waitGroup.Wait()
}

func (p *mockConcurrentExporter) stop() {
	atomic.StoreInt32(&p.stopped, 1)
}

type observabilityConsumerSender struct {
	sentItemsCount    int64
	droppedItemsCount int64
	nextSender        requestSender
}

func (ocs *observabilityConsumerSender) send(req request) (int, error) {
	dic, err := ocs.nextSender.send(req)
	atomic.AddInt64(&ocs.sentItemsCount, int64(req.count()-dic))
	atomic.AddInt64(&ocs.droppedItemsCount, int64(dic))
	return dic, err
}

func (ocs *observabilityConsumerSender) checkSendItemsCount(t *testing.T, want int) {
	assert.EqualValues(t, want, atomic.LoadInt64(&ocs.sentItemsCount))
}

func (ocs *observabilityConsumerSender) checkDroppedItemsCount(t *testing.T, want int) {
	assert.EqualValues(t, want, atomic.LoadInt64(&ocs.droppedItemsCount))
}
