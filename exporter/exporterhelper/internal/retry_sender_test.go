// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

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
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/internal"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/pipeline"
)

func mockRequestUnmarshaler(mr internal.Request) exporterqueue.Unmarshaler[internal.Request] {
	return func([]byte) (internal.Request, error) {
		return mr, nil
	}
}

func mockRequestMarshaler(internal.Request) ([]byte, error) {
	return []byte("mockRequest"), nil
}

func TestQueuedRetry_DropOnPermanentError(t *testing.T) {
	qCfg := NewDefaultQueueConfig()
	rCfg := configretry.NewDefaultBackOffConfig()
	mockR := newMockRequest(2, consumererror.NewPermanent(errors.New("bad data")))
	be, err := NewBaseExporter(defaultSettings, defaultSignal, newObservabilityConsumerSender,
		WithMarshaler(mockRequestMarshaler), WithUnmarshaler(mockRequestUnmarshaler(mockR)), WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.ObsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.Send(context.Background(), mockR))
	})
	ocs.awaitAsyncProcessing()
	// In the newMockConcurrentExporter we count requests and items even for failed requests
	mockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 0)
	ocs.checkDroppedItemsCount(t, 2)
}

func TestQueuedRetry_DropOnNoRetry(t *testing.T) {
	qCfg := NewDefaultQueueConfig()
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.Enabled = false
	be, err := NewBaseExporter(defaultSettings, defaultSignal, newObservabilityConsumerSender, WithMarshaler(mockRequestMarshaler),
		WithUnmarshaler(mockRequestUnmarshaler(newMockRequest(2, errors.New("transient error")))),
		WithQueue(qCfg), WithRetry(rCfg))
	require.NoError(t, err)
	ocs := be.ObsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	mockR := newMockRequest(2, errors.New("transient error"))
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.Send(context.Background(), mockR))
	})
	ocs.awaitAsyncProcessing()
	// In the newMockConcurrentExporter we count requests and items even for failed requests
	mockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 0)
	ocs.checkDroppedItemsCount(t, 2)
}

func TestQueuedRetry_OnError(t *testing.T) {
	qCfg := NewDefaultQueueConfig()
	qCfg.NumConsumers = 1
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = 0
	be, err := NewBaseExporter(defaultSettings, defaultSignal, newObservabilityConsumerSender,
		WithMarshaler(mockRequestMarshaler), WithUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
		WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	traceErr := consumererror.NewTraces(errors.New("some error"), testdata.GenerateTraces(1))
	mockR := newMockRequest(2, traceErr)
	ocs := be.ObsrepSender.(*observabilityConsumerSender)
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.Send(context.Background(), mockR))
	})
	ocs.awaitAsyncProcessing()

	// In the newMockConcurrentExporter we count requests and items even for failed requests
	mockR.checkNumRequests(t, 2)
	ocs.checkSendItemsCount(t, 2)
	ocs.checkDroppedItemsCount(t, 0)
}

func TestQueuedRetry_MaxElapsedTime(t *testing.T) {
	qCfg := NewDefaultQueueConfig()
	qCfg.NumConsumers = 1
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = time.Millisecond
	rCfg.MaxElapsedTime = 100 * time.Millisecond
	be, err := NewBaseExporter(defaultSettings, defaultSignal, newObservabilityConsumerSender,
		WithMarshaler(mockRequestMarshaler), WithUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
		WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.ObsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	ocs.run(func() {
		// Add an item that will always fail.
		require.NoError(t, be.Send(context.Background(), newErrorRequest()))
	})

	mockR := newMockRequest(2, nil)
	start := time.Now()
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.Send(context.Background(), mockR))
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
	require.Zero(t, be.QueueSender.(*QueueSender).queue.Size())
}

type wrappedError struct {
	error
}

func (e wrappedError) Unwrap() error {
	return e.error
}

func TestQueuedRetry_ThrottleError(t *testing.T) {
	qCfg := NewDefaultQueueConfig()
	qCfg.NumConsumers = 1
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = 10 * time.Millisecond
	be, err := NewBaseExporter(defaultSettings, defaultSignal, newObservabilityConsumerSender,
		WithMarshaler(mockRequestMarshaler), WithUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
		WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.ObsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	retry := NewThrottleRetry(errors.New("throttle error"), 100*time.Millisecond)
	mockR := newMockRequest(2, wrappedError{retry})
	start := time.Now()
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.Send(context.Background(), mockR))
	})
	ocs.awaitAsyncProcessing()

	// The initial backoff is 10ms, but because of the throttle this should wait at least 100ms.
	assert.Less(t, 100*time.Millisecond, time.Since(start))

	mockR.checkNumRequests(t, 2)
	ocs.checkSendItemsCount(t, 2)
	ocs.checkDroppedItemsCount(t, 0)
	require.Zero(t, be.QueueSender.(*QueueSender).queue.Size())
}

func TestQueuedRetry_RetryOnError(t *testing.T) {
	qCfg := NewDefaultQueueConfig()
	qCfg.NumConsumers = 1
	qCfg.QueueSize = 1
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = 0
	be, err := NewBaseExporter(defaultSettings, defaultSignal, newObservabilityConsumerSender,
		WithMarshaler(mockRequestMarshaler), WithUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
		WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.ObsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	mockR := newMockRequest(2, errors.New("transient error"))
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.Send(context.Background(), mockR))
	})
	ocs.awaitAsyncProcessing()

	// In the newMockConcurrentExporter we count requests and items even for failed requests
	mockR.checkNumRequests(t, 2)
	ocs.checkSendItemsCount(t, 2)
	ocs.checkDroppedItemsCount(t, 0)
	require.Zero(t, be.QueueSender.(*QueueSender).queue.Size())
}

func TestQueueRetryWithNoQueue(t *testing.T) {
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.MaxElapsedTime = time.Nanosecond // fail fast
	be, err := NewBaseExporter(exportertest.NewNopSettings(), pipeline.SignalLogs, newObservabilityConsumerSender, WithRetry(rCfg))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	ocs := be.ObsrepSender.(*observabilityConsumerSender)
	mockR := newMockRequest(2, errors.New("some error"))
	ocs.run(func() {
		require.Error(t, be.Send(context.Background(), mockR))
	})
	ocs.awaitAsyncProcessing()
	mockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 0)
	ocs.checkDroppedItemsCount(t, 2)
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestQueueRetryWithDisabledRetires(t *testing.T) {
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.Enabled = false
	set := exportertest.NewNopSettings()
	logger, observed := observer.New(zap.ErrorLevel)
	set.Logger = zap.New(logger)
	be, err := NewBaseExporter(set, pipeline.SignalLogs, newObservabilityConsumerSender, WithRetry(rCfg))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	ocs := be.ObsrepSender.(*observabilityConsumerSender)
	mockR := newMockRequest(2, errors.New("some error"))
	ocs.run(func() {
		require.Error(t, be.Send(context.Background(), mockR))
	})
	assert.Len(t, observed.All(), 1)
	assert.Equal(t, "Exporting failed. Rejecting data. "+
		"Try enabling retry_on_failure config option to retry on retryable errors.", observed.All()[0].Message)
	ocs.awaitAsyncProcessing()
	mockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 0)
	ocs.checkDroppedItemsCount(t, 2)
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestRetryWithContextTimeout(t *testing.T) {
	const testTimeout = 10 * time.Second

	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.Enabled = true

	// First attempt after 100ms is attempted
	rCfg.InitialInterval = 100 * time.Millisecond
	rCfg.RandomizationFactor = 0
	// Second attempt is at twice the testTimeout
	rCfg.Multiplier = float64(2 * testTimeout / rCfg.InitialInterval)
	qCfg := exporterqueue.NewDefaultConfig()
	qCfg.Enabled = false
	set := exportertest.NewNopSettings()
	logger, observed := observer.New(zap.InfoLevel)
	set.Logger = zap.New(logger)
	be, err := NewBaseExporter(
		set,
		pipeline.SignalLogs,
		newObservabilityConsumerSender,
		WithRetry(rCfg),
		WithRequestQueue(qCfg, exporterqueue.NewMemoryQueueFactory[internal.Request]()),
	)
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	ocs := be.ObsrepSender.(*observabilityConsumerSender)
	mockR := newErrorRequest()

	start := time.Now()
	ocs.run(func() {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		err := be.Send(ctx, mockR)
		require.Error(t, err)
		require.Equal(t, "request will be cancelled before next retry: transient error", err.Error())
	})
	assert.Len(t, observed.All(), 2)
	assert.Equal(t, "Exporting failed. Will retry the request after interval.", observed.All()[0].Message)
	assert.Equal(t, "Exporting failed. Rejecting data. "+
		"Try enabling sending_queue to survive temporary failures.", observed.All()[1].Message)
	ocs.awaitAsyncProcessing()
	ocs.checkDroppedItemsCount(t, 7)
	require.Equal(t, 2, mockR.(*mockErrorRequest).getNumRequests())
	require.NoError(t, be.Shutdown(context.Background()))

	// There should be no delay, because the initial interval is
	// longer than the context timeout.  Merely checking that no
	// delays on the order of either the context timeout or the
	// retry interval were introduced, i.e., fail fast.
	elapsed := time.Since(start)
	require.Less(t, elapsed, testTimeout/2)
}

type mockErrorRequest struct {
	mu       sync.Mutex
	requests int
}

func (mer *mockErrorRequest) Export(context.Context) error {
	mer.mu.Lock()
	defer mer.mu.Unlock()
	mer.requests++
	return errors.New("transient error")
}

func (mer *mockErrorRequest) getNumRequests() int {
	mer.mu.Lock()
	defer mer.mu.Unlock()
	return mer.requests
}

func (mer *mockErrorRequest) OnError(error) internal.Request {
	return mer
}

func (mer *mockErrorRequest) ItemsCount() int {
	return 7
}

func (mer *mockErrorRequest) Merge(context.Context, internal.Request) (internal.Request, error) {
	return nil, nil
}

func (mer *mockErrorRequest) MergeSplit(context.Context, exporterbatcher.MaxSizeConfig, internal.Request) ([]internal.Request, error) {
	return nil, nil
}

func newErrorRequest() internal.Request {
	return &mockErrorRequest{}
}

type mockRequest struct {
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

func (m *mockRequest) ItemsCount() int {
	return m.cnt
}

func (m *mockRequest) Merge(context.Context, internal.Request) (internal.Request, error) {
	return nil, nil
}

func (m *mockRequest) MergeSplit(context.Context, exporterbatcher.MaxSizeConfig, internal.Request) ([]internal.Request, error) {
	return nil, nil
}

func newMockRequest(cnt int, consumeError error) *mockRequest {
	return &mockRequest{
		cnt:          cnt,
		consumeError: consumeError,
		requestCount: &atomic.Int64{},
	}
}

type observabilityConsumerSender struct {
	BaseRequestSender
	waitGroup         *sync.WaitGroup
	sentItemsCount    *atomic.Int64
	droppedItemsCount *atomic.Int64
}

func newObservabilityConsumerSender(*ObsReport) RequestSender {
	return &observabilityConsumerSender{
		waitGroup:         new(sync.WaitGroup),
		droppedItemsCount: &atomic.Int64{},
		sentItemsCount:    &atomic.Int64{},
	}
}

func (ocs *observabilityConsumerSender) Send(ctx context.Context, req internal.Request) error {
	err := ocs.NextSender.Send(ctx, req)
	if err != nil {
		ocs.droppedItemsCount.Add(int64(req.ItemsCount()))
	} else {
		ocs.sentItemsCount.Add(int64(req.ItemsCount()))
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
