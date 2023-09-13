// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

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
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricproducer"
	"go.opencensus.io/tag"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
)

func mockRequestUnmarshaler(mr *mockRequest) internal.RequestUnmarshaler {
	return func(bytes []byte) (internal.Request, error) {
		return mr, nil
	}
}

func mockRequestMarshaler(_ internal.Request) ([]byte, error) {
	return nil, nil
}

func TestQueuedRetry_DropOnPermanentError(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	rCfg := NewDefaultRetrySettings()
	mockR := newMockRequest(context.Background(), 2, consumererror.NewPermanent(errors.New("bad data")))
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.send(mockR))
	})
	ocs.awaitAsyncProcessing()
	// In the newMockConcurrentExporter we count requests and items even for failed requests
	mockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 0)
	ocs.checkDroppedItemsCount(t, 2)
}

func TestQueuedRetry_DropOnNoRetry(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	rCfg := NewDefaultRetrySettings()
	rCfg.Enabled = false
	be, err := newBaseExporter(defaultSettings, "", false, mockRequestMarshaler,
		mockRequestUnmarshaler(newMockRequest(context.Background(), 2, errors.New("transient error"))),
		newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	mockR := newMockRequest(context.Background(), 2, errors.New("transient error"))
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.send(mockR))
	})
	ocs.awaitAsyncProcessing()
	// In the newMockConcurrentExporter we count requests and items even for failed requests
	mockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 0)
	ocs.checkDroppedItemsCount(t, 2)
}

func TestQueuedRetry_OnError(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := NewDefaultRetrySettings()
	rCfg.InitialInterval = 0
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	traceErr := consumererror.NewTraces(errors.New("some error"), testdata.GenerateTraces(1))
	mockR := newMockRequest(context.Background(), 2, traceErr)
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.send(mockR))
	})
	ocs.awaitAsyncProcessing()

	// In the newMockConcurrentExporter we count requests and items even for failed requests
	mockR.checkNumRequests(t, 2)
	ocs.checkSendItemsCount(t, 2)
	ocs.checkDroppedItemsCount(t, 0)
}

func TestQueuedRetry_StopWhileWaiting(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := NewDefaultRetrySettings()
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	firstMockR := newErrorRequest(context.Background())
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.send(firstMockR))
	})

	// Enqueue another request to ensure when calling shutdown we drain the queue.
	secondMockR := newMockRequest(context.Background(), 3, nil)
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.send(secondMockR))
	})

	require.LessOrEqual(t, 1, be.queueSender.(*queueSender).queue.Size())

	assert.NoError(t, be.Shutdown(context.Background()))

	secondMockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 3)
	ocs.checkDroppedItemsCount(t, 7)
	require.Zero(t, be.queueSender.(*queueSender).queue.Size())
}

func TestQueuedRetry_DoNotPreserveCancellation(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := NewDefaultRetrySettings()
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	ctx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()
	mockR := newMockRequest(ctx, 2, nil)
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.send(mockR))
	})
	ocs.awaitAsyncProcessing()

	mockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 2)
	ocs.checkDroppedItemsCount(t, 0)
	require.Zero(t, be.queueSender.(*queueSender).queue.Size())
}

func TestQueuedRetry_MaxElapsedTime(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := NewDefaultRetrySettings()
	rCfg.InitialInterval = time.Millisecond
	rCfg.MaxElapsedTime = 100 * time.Millisecond
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	ocs.run(func() {
		// Add an item that will always fail.
		require.NoError(t, be.send(newErrorRequest(context.Background())))
	})

	mockR := newMockRequest(context.Background(), 2, nil)
	start := time.Now()
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.send(mockR))
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
	require.Zero(t, be.queueSender.(*queueSender).queue.Size())
}

type wrappedError struct {
	error
}

func (e wrappedError) Unwrap() error {
	return e.error
}

func TestQueuedRetry_ThrottleError(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := NewDefaultRetrySettings()
	rCfg.InitialInterval = 10 * time.Millisecond
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	retry := NewThrottleRetry(errors.New("throttle error"), 100*time.Millisecond)
	mockR := newMockRequest(context.Background(), 2, wrappedError{retry})
	start := time.Now()
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.send(mockR))
	})
	ocs.awaitAsyncProcessing()

	// The initial backoff is 10ms, but because of the throttle this should wait at least 100ms.
	assert.True(t, 100*time.Millisecond < time.Since(start))

	mockR.checkNumRequests(t, 2)
	ocs.checkSendItemsCount(t, 2)
	ocs.checkDroppedItemsCount(t, 0)
	require.Zero(t, be.queueSender.(*queueSender).queue.Size())
}

func TestQueuedRetry_RetryOnError(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	qCfg.QueueSize = 1
	rCfg := NewDefaultRetrySettings()
	rCfg.InitialInterval = 0
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	mockR := newMockRequest(context.Background(), 2, errors.New("transient error"))
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.send(mockR))
	})
	ocs.awaitAsyncProcessing()

	// In the newMockConcurrentExporter we count requests and items even for failed requests
	mockR.checkNumRequests(t, 2)
	ocs.checkSendItemsCount(t, 2)
	ocs.checkDroppedItemsCount(t, 0)
	require.Zero(t, be.queueSender.(*queueSender).queue.Size())
}

func TestQueuedRetry_DropOnFull(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.QueueSize = 0
	rCfg := NewDefaultRetrySettings()
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})
	ocs.run(func() {
		require.Error(t, be.send(newMockRequest(context.Background(), 2, errors.New("transient error"))))
	})
}

func TestQueuedRetryHappyPath(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(defaultID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	qCfg := NewDefaultQueueSettings()
	rCfg := NewDefaultRetrySettings()
	set := tt.ToExporterCreateSettings()
	be, err := newBaseExporter(set, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.obsrepSender.(*observabilityConsumerSender)
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
			require.NoError(t, be.send(req))
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

func TestQueuedRetry_QueueMetricsReported(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 0 // to make every request go straight to the queue
	rCfg := NewDefaultRetrySettings()
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	checkValueForGlobalManager(t, defaultExporterTags, int64(defaultQueueSize), "exporter/queue_capacity")
	for i := 0; i < 7; i++ {
		require.NoError(t, be.send(newErrorRequest(context.Background())))
	}
	checkValueForGlobalManager(t, defaultExporterTags, int64(7), "exporter/queue_size")

	assert.NoError(t, be.Shutdown(context.Background()))
	checkValueForGlobalManager(t, defaultExporterTags, int64(0), "exporter/queue_size")
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

func TestQueueSettings_Validate(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	assert.NoError(t, qCfg.Validate())

	qCfg.QueueSize = 0
	assert.EqualError(t, qCfg.Validate(), "queue size must be positive")

	// Confirm Validate doesn't return error with invalid config when feature is disabled
	qCfg.Enabled = false
	assert.NoError(t, qCfg.Validate())
}

// if requeueing is enabled, we eventually retry even if we failed at first
func TestQueuedRetry_RequeuingEnabled(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := NewDefaultRetrySettings()
	rCfg.MaxElapsedTime = time.Nanosecond // we don't want to retry at all, but requeue instead
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	be.queueSender.(*queueSender).requeuingEnabled = true
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	traceErr := consumererror.NewTraces(errors.New("some error"), testdata.GenerateTraces(1))
	mockR := newMockRequest(context.Background(), 1, traceErr)
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.send(mockR))
		ocs.waitGroup.Add(1) // necessary because we'll call send() again after requeueing
	})
	ocs.awaitAsyncProcessing()

	// In the newMockConcurrentExporter we count requests and items even for failed requests
	mockR.checkNumRequests(t, 2)
	ocs.checkSendItemsCount(t, 1)
	ocs.checkDroppedItemsCount(t, 1) // not actually dropped, but ocs counts each failed send here
}

// if requeueing is enabled, but the queue is full, we get an error
func TestQueuedRetry_RequeuingEnabledQueueFull(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 0
	qCfg.QueueSize = 0
	rCfg := NewDefaultRetrySettings()
	rCfg.MaxElapsedTime = time.Nanosecond // we don't want to retry at all, but requeue instead
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	be.queueSender.(*queueSender).requeuingEnabled = true
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	traceErr := consumererror.NewTraces(errors.New("some error"), testdata.GenerateTraces(1))
	mockR := newMockRequest(context.Background(), 1, traceErr)

	require.Error(t, be.retrySender.send(mockR), "sending_queue is full")
	mockR.checkNumRequests(t, 1)
}

func TestQueueRetryWithDisabledQueue(t *testing.T) {
	qs := NewDefaultQueueSettings()
	qs.Enabled = false
	be, err := newBaseExporter(exportertest.NewNopCreateSettings(), component.DataTypeLogs, false, nil, nil, newObservabilityConsumerSender, WithQueue(qs))
	require.Nil(t, be.queueSender.(*queueSender).queue)
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	mockR := newMockRequest(context.Background(), 2, errors.New("some error"))
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.Error(t, be.send(mockR))
	})
	ocs.awaitAsyncProcessing()
	mockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 0)
	ocs.checkDroppedItemsCount(t, 2)
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestQueueRetryWithNoQueue(t *testing.T) {
	rCfg := NewDefaultRetrySettings()
	rCfg.MaxElapsedTime = time.Nanosecond // fail fast
	be, err := newBaseExporter(exportertest.NewNopCreateSettings(), component.DataTypeLogs, false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	mockR := newMockRequest(context.Background(), 2, errors.New("some error"))
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.Error(t, be.send(mockR))
	})
	ocs.awaitAsyncProcessing()
	mockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 0)
	ocs.checkDroppedItemsCount(t, 2)
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestQueuedRetryPersistenceEnabled(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(defaultID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	qCfg := NewDefaultQueueSettings()
	storageID := component.NewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID // enable persistence
	rCfg := NewDefaultRetrySettings()
	set := tt.ToExporterCreateSettings()
	be, err := newBaseExporter(set, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	var extensions = map[component.ID]component.Component{
		storageID: internal.NewMockStorageExtension(nil),
	}
	host := &mockHost{ext: extensions}

	// we start correctly with a file storage extension
	require.NoError(t, be.Start(context.Background(), host))
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestQueuedRetryPersistenceEnabledStorageError(t *testing.T) {
	storageError := errors.New("could not get storage client")
	tt, err := obsreporttest.SetupTelemetry(defaultID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	qCfg := NewDefaultQueueSettings()
	storageID := component.NewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID // enable persistence
	rCfg := NewDefaultRetrySettings()
	set := tt.ToExporterCreateSettings()
	be, err := newBaseExporter(set, "", false, mockRequestMarshaler, mockRequestUnmarshaler(&mockRequest{}), newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	var extensions = map[component.ID]component.Component{
		storageID: internal.NewMockStorageExtension(storageError),
	}
	host := &mockHost{ext: extensions}

	// we fail to start if we get an error creating the storage client
	require.Error(t, be.Start(context.Background(), host), "could not get storage client")
}

func TestQueuedRetryPersistentEnabled_shutdown_dataIsRequeued(t *testing.T) {

	produceCounter := &atomic.Uint32{}

	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := NewDefaultRetrySettings()
	rCfg.InitialInterval = time.Millisecond
	rCfg.MaxElapsedTime = 0 // retry infinitely so shutdown can be triggered

	req := newMockRequest(context.Background(), 3, errors.New("some error"))

	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newNoopObsrepSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	require.NoError(t, be.Start(context.Background(), &mockHost{}))

	// wraps original queue so we can count operations
	be.queueSender.(*queueSender).queue = &producerConsumerQueueWithCounter{
		ProducerConsumerQueue: be.queueSender.(*queueSender).queue,
		produceCounter:        produceCounter,
	}
	be.queueSender.(*queueSender).requeuingEnabled = true

	// replace nextSender inside retrySender to always return error so it doesn't exit send loop
	be.retrySender.setNextSender(&errorRequestSender{
		errToReturn: errors.New("some error"),
	})

	// Invoke queuedRetrySender so the producer will put the item for consumer to poll
	require.NoError(t, be.send(req))

	// first wait for the item to be produced to the queue initially
	assert.Eventually(t, func() bool {
		return produceCounter.Load() == uint32(1)
	}, time.Second, 1*time.Millisecond)

	// shuts down and ensure the item is produced in the queue again
	require.NoError(t, be.Shutdown(context.Background()))
	assert.Eventually(t, func() bool {
		return produceCounter.Load() == uint32(2)
	}, time.Second, 1*time.Millisecond)
}

func TestQueueRetryOptionsWithRequestExporter(t *testing.T) {
	bs, err := newBaseExporter(exportertest.NewNopCreateSettings(), "", true, nil, nil, newNoopObsrepSender,
		WithRetry(NewDefaultRetrySettings()))
	assert.Nil(t, err)
	assert.True(t, bs.requestExporter)
	assert.Panics(t, func() {
		_, _ = newBaseExporter(exportertest.NewNopCreateSettings(), "", true, nil, nil, newNoopObsrepSender,
			WithRetry(NewDefaultRetrySettings()), WithQueue(NewDefaultQueueSettings()))
	})
}

type mockErrorRequest struct {
	baseRequest
}

func (mer *mockErrorRequest) Export(_ context.Context) error {
	return errors.New("transient error")
}

func (mer *mockErrorRequest) OnError(error) internal.Request {
	return mer
}

func (mer *mockErrorRequest) Count() int {
	return 7
}

func newErrorRequest(ctx context.Context) internal.Request {
	return &mockErrorRequest{
		baseRequest: baseRequest{ctx: ctx},
	}
}

type mockRequest struct {
	baseRequest
	cnt          int
	mu           sync.Mutex
	consumeError error
	requestCount *atomic.Int64
}

func (m *mockRequest) Export(ctx context.Context) error {
	m.requestCount.Add(int64(1))
	m.mu.Lock()
	defer m.mu.Unlock()
	err := m.consumeError
	m.consumeError = nil
	if err != nil {
		return err
	}
	// Respond like gRPC/HTTP, if context is cancelled, return error
	return ctx.Err()
}

func (m *mockRequest) OnError(error) internal.Request {
	return &mockRequest{
		baseRequest:  m.baseRequest,
		cnt:          1,
		consumeError: nil,
		requestCount: m.requestCount,
	}
}

func (m *mockRequest) checkNumRequests(t *testing.T, want int) {
	assert.Eventually(t, func() bool {
		return int64(want) == m.requestCount.Load()
	}, time.Second, 1*time.Millisecond)
}

func (m *mockRequest) Count() int {
	return m.cnt
}

func newMockRequest(ctx context.Context, cnt int, consumeError error) *mockRequest {
	return &mockRequest{
		baseRequest:  baseRequest{ctx: ctx},
		cnt:          cnt,
		consumeError: consumeError,
		requestCount: &atomic.Int64{},
	}
}

type observabilityConsumerSender struct {
	baseRequestSender
	waitGroup         *sync.WaitGroup
	sentItemsCount    *atomic.Int64
	droppedItemsCount *atomic.Int64
}

func newObservabilityConsumerSender(_ *obsExporter) requestSender {
	return &observabilityConsumerSender{
		waitGroup:         new(sync.WaitGroup),
		droppedItemsCount: &atomic.Int64{},
		sentItemsCount:    &atomic.Int64{},
	}
}

func (ocs *observabilityConsumerSender) send(req internal.Request) error {
	err := ocs.nextSender.send(req)
	if err != nil {
		ocs.droppedItemsCount.Add(int64(req.Count()))
	} else {
		ocs.sentItemsCount.Add(int64(req.Count()))
	}
	ocs.waitGroup.Done()
	return err
}

func (ocs *observabilityConsumerSender) run(fn func()) {
	ocs.waitGroup.Add(1)
	fn()
}

func (ocs *observabilityConsumerSender) awaitAsyncProcessing() {
	ocs.waitGroup.Wait()
}

func (ocs *observabilityConsumerSender) checkSendItemsCount(t *testing.T, want int) {
	assert.EqualValues(t, want, ocs.sentItemsCount.Load())
}

func (ocs *observabilityConsumerSender) checkDroppedItemsCount(t *testing.T, want int) {
	assert.EqualValues(t, want, ocs.droppedItemsCount.Load())
}

// checkValueForGlobalManager checks that the given metrics with wantTags is reported by one of the
// metric producers
func checkValueForGlobalManager(t *testing.T, wantTags []tag.Tag, value int64, vName string) {
	producers := metricproducer.GlobalManager().GetAll()
	for _, producer := range producers {
		if checkValueForProducer(t, producer, wantTags, value, vName) {
			return
		}
	}
	require.Fail(t, fmt.Sprintf("could not find metric %v with tags %s reported", vName, wantTags))
}

// checkValueForProducer checks that the given metrics with wantTags is reported by the metric producer
func checkValueForProducer(t *testing.T, producer metricproducer.Producer, wantTags []tag.Tag, value int64, vName string) bool {
	for _, metric := range producer.Read() {
		if metric.Descriptor.Name == vName && len(metric.TimeSeries) > 0 {
			for _, ts := range metric.TimeSeries {
				if tagsMatchLabelKeys(wantTags, metric.Descriptor.LabelKeys, ts.LabelValues) {
					require.Equal(t, value, ts.Points[len(ts.Points)-1].Value.(int64))
					return true
				}
			}
		}
	}
	return false
}

// tagsMatchLabelKeys returns true if provided tags match keys and values
func tagsMatchLabelKeys(tags []tag.Tag, keys []metricdata.LabelKey, labels []metricdata.LabelValue) bool {
	if len(tags) != len(keys) {
		return false
	}
	for i := 0; i < len(tags); i++ {
		var labelVal string
		if labels[i].Present {
			labelVal = labels[i].Value
		}
		if tags[i].Key.Name() != keys[i].Key || tags[i].Value != labelVal {
			return false
		}
	}
	return true
}

type mockHost struct {
	component.Host
	ext map[component.ID]component.Component
}

func (nh *mockHost) GetExtensions() map[component.ID]component.Component {
	return nh.ext
}

type producerConsumerQueueWithCounter struct {
	internal.ProducerConsumerQueue
	produceCounter *atomic.Uint32
}

func (pcq *producerConsumerQueueWithCounter) Produce(item internal.Request) bool {
	pcq.produceCounter.Add(1)
	return pcq.ProducerConsumerQueue.Produce(item)
}

type errorRequestSender struct {
	baseRequestSender
	errToReturn error
}

func (rs *errorRequestSender) send(_ internal.Request) error {
	return rs.errToReturn
}
