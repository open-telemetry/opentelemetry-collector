// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/hosttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sendertest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/storagetest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
)

func newFakeRequestSettings() Settings[request.Request] {
	return Settings[request.Request]{
		Signal:    pipeline.SignalMetrics,
		ID:        component.NewID(exportertest.NopType),
		Telemetry: componenttest.NewNopTelemetrySettings(),
		Encoding:  newFakeEncoding(&requesttest.FakeRequest{}),
		Sizers: map[request.SizerType]request.Sizer[request.Request]{
			request.SizerTypeRequests: request.RequestsSizer[request.Request]{},
			request.SizerTypeItems:    request.NewItemsSizer(),
		},
	}
}

type fakeEncoding struct {
	mr request.Request
}

func (f fakeEncoding) Marshal(request.Request) ([]byte, error) {
	return []byte("mockRequest"), nil
}

func (f fakeEncoding) Unmarshal([]byte) (request.Request, error) {
	return f.mr, nil
}

func newFakeEncoding(mr request.Request) Encoding[request.Request] {
	return &fakeEncoding{mr: mr}
}

func TestQueueBatchStopWhileWaiting(t *testing.T) {
	sink := requesttest.NewSink()
	cfg := newTestConfig()
	cfg.NumConsumers = 1
	cfg.Batch = nil
	qb, err := NewQueueBatch(newFakeRequestSettings(), cfg, sink.Export)
	require.NoError(t, err)
	require.NoError(t, qb.Start(context.Background(), componenttest.NewNopHost()))
	sink.SetExportErr(errors.New("transient error"))
	require.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 4}))
	// Enqueue another request to ensure when calling shutdown we drain the queue.
	require.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 3, Delay: 100 * time.Millisecond}))
	require.LessOrEqual(t, int64(1), qb.queue.Size())

	require.NoError(t, qb.Shutdown(context.Background()))
	assert.EqualValues(t, 1, sink.RequestsCount())
	assert.EqualValues(t, 3, sink.ItemsCount())
	require.Zero(t, qb.queue.Size())
}

func TestQueueBatchDoNotPreserveCancellation(t *testing.T) {
	sink := requesttest.NewSink()
	cfg := newTestConfig()
	cfg.NumConsumers = 1
	qb, err := NewQueueBatch(newFakeRequestSettings(), cfg, sink.Export)
	require.NoError(t, err)
	require.NoError(t, qb.Start(context.Background(), componenttest.NewNopHost()))

	ctx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()

	require.NoError(t, qb.Send(ctx, &requesttest.FakeRequest{Items: 4}))
	require.NoError(t, qb.Shutdown(context.Background()))

	assert.EqualValues(t, 1, sink.RequestsCount())
	assert.EqualValues(t, 4, sink.ItemsCount())
	require.Zero(t, qb.queue.Size())
}

func TestQueueBatchHappyPath(t *testing.T) {
	cfg := newTestConfig()
	cfg.BlockOnOverflow = false
	cfg.QueueSize = 56
	sink := requesttest.NewSink()
	qb, err := NewQueueBatch(newFakeRequestSettings(), cfg, sink.Export)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		require.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: i + 1}))
	}

	// expect queue to be full
	require.Error(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 2}))

	require.NoError(t, qb.Start(context.Background(), componenttest.NewNopHost()))
	assert.Eventually(t, func() bool {
		// Because batching is used, cannot guarantee that will be 1 batch or multiple because of the flush interval.
		// Check only for total items count.
		return sink.ItemsCount() == 55
	}, 1*time.Second, 10*time.Millisecond)
	require.NoError(t, qb.Shutdown(context.Background()))
}

func TestQueueBatchPersistenceEnabled(t *testing.T) {
	cfg := newTestConfig()
	storageID := component.MustNewIDWithName("file_storage", "storage")
	cfg.StorageID = &storageID
	qb, err := NewQueueBatch(newFakeRequestSettings(), cfg, sendertest.NewNopSenderFunc[request.Request]())
	require.NoError(t, err)

	host := hosttest.NewHost(map[component.ID]component.Component{
		storageID: storagetest.NewMockStorageExtension(nil),
	})

	// we start correctly with a file storage extension
	require.NoError(t, qb.Start(context.Background(), host))
	require.NoError(t, qb.Shutdown(context.Background()))
}

func TestQueueBatchPersistenceEnabledStorageError(t *testing.T) {
	storageError := errors.New("could not get storage client")
	cfg := newTestConfig()
	storageID := component.MustNewIDWithName("file_storage", "storage")
	cfg.StorageID = &storageID
	qb, err := NewQueueBatch(newFakeRequestSettings(), cfg, sendertest.NewNopSenderFunc[request.Request]())
	require.NoError(t, err)

	host := hosttest.NewHost(map[component.ID]component.Component{
		storageID: storagetest.NewMockStorageExtension(storageError),
	})

	// we fail to start if we get an error creating the storage client
	require.Error(t, qb.Start(context.Background(), host), "could not get storage client")
}

func TestQueueBatchPersistentEnabled_NoDataLossOnShutdown(t *testing.T) {
	cfg := newTestConfig()
	cfg.NumConsumers = 1
	storageID := component.MustNewIDWithName("file_storage", "storage")
	cfg.StorageID = &storageID

	mockReq := &requesttest.FakeRequest{Items: 2}
	qSet := newFakeRequestSettings()
	qSet.Encoding = newFakeEncoding(mockReq)
	done := make(chan struct{})
	qb, err := NewQueueBatch(qSet, cfg, func(context.Context, request.Request) error {
		<-done
		return experr.NewShutdownErr(errors.New("could not export data"))
	})
	require.NoError(t, err)

	host := hosttest.NewHost(map[component.ID]component.Component{
		storageID: storagetest.NewMockStorageExtension(nil),
	})

	require.NoError(t, qb.Start(context.Background(), host))

	// Invoke queuedRetrySender so the producer will put the item for consumer to poll
	require.NoError(t, qb.Send(context.Background(), mockReq))

	// first wait for the item to be consumed from the queue
	assert.Eventually(t, func() bool {
		return qb.queue.Size() == 0
	}, 1*time.Second, 10*time.Millisecond)

	// shuts down the exporter, unsent data should be preserved as in-flight data in the persistent queue.
	close(done)
	require.NoError(t, qb.Shutdown(context.Background()))

	// start the exporter again replacing the preserved mockRequest in the unmarshaler with a new one that doesn't fail.
	sink := requesttest.NewSink()
	replacedReq := &requesttest.FakeRequest{Items: 7}
	qSet.Encoding = newFakeEncoding(replacedReq)
	qb, err = NewQueueBatch(qSet, cfg, sink.Export)
	require.NoError(t, err)
	require.NoError(t, qb.Start(context.Background(), host))

	assert.Eventually(t, func() bool {
		return sink.ItemsCount() == 7 && sink.RequestsCount() == 1
	}, 1*time.Second, 10*time.Millisecond)
	require.NoError(t, qb.Shutdown(context.Background()))
}

func TestQueueBatchNoStartShutdown(t *testing.T) {
	qs, err := NewQueueBatch(newFakeRequestSettings(), newTestConfig(), sendertest.NewNopSenderFunc[request.Request]())
	require.NoError(t, err)
	assert.NoError(t, qs.Shutdown(context.Background()))
}

func TestQueueBatch_Merge(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping flaky test on Windows, see https://github.com/open-telemetry/opentelemetry-collector/issues/10758")
	}

	tests := []struct {
		name     string
		batchCfg *BatchConfig
	}{
		{
			name: "split_disabled",
			batchCfg: &BatchConfig{
				FlushTimeout: 100 * time.Millisecond,
				MinSize:      10,
			},
		},
		{
			name: "split_high_limit",
			batchCfg: &BatchConfig{
				FlushTimeout: 100 * time.Millisecond,
				MinSize:      10,
				MaxSize:      1000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := requesttest.NewSink()
			cfg := newTestConfig()
			cfg.Batch = tt.batchCfg
			qb, err := NewQueueBatch(newFakeRequestSettings(), cfg, sink.Export)
			require.NoError(t, err)
			require.NoError(t, qb.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, qb.Shutdown(context.Background()))
			})

			require.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 8}))
			require.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 3}))

			// the first two requests should be merged into one and sent by reaching the minimum items size
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 1 && sink.ItemsCount() == 11
			}, 50*time.Millisecond, 10*time.Millisecond)

			require.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 3}))
			require.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 1}))

			// the third and fifth requests should be sent by reaching the timeout
			// the fourth request should be ignored because of the merge error.
			time.Sleep(50 * time.Millisecond)

			// should be ignored because of the merge error.
			require.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{
				Items:    3,
				MergeErr: errors.New("merge error"),
			}))

			assert.Equal(t, 1, sink.RequestsCount())
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 2 && sink.ItemsCount() == 15
			}, 1*time.Second, 10*time.Millisecond)
		})
	}
}

func TestQueueBatch_BatchExportError(t *testing.T) {
	tests := []struct {
		name             string
		batchCfg         *BatchConfig
		expectedRequests int
		expectedItems    int
	}{
		{
			name: "merge_only",
			batchCfg: &BatchConfig{
				FlushTimeout: 200 * time.Millisecond,
				MinSize:      10,
			},
		},
		{
			name: "merge_without_split_triggered",
			batchCfg: &BatchConfig{
				FlushTimeout: 200 * time.Millisecond,
				MinSize:      10,
				MaxSize:      200,
			},
		},
		{
			name: "merge_with_split_triggered",
			batchCfg: &BatchConfig{
				FlushTimeout: 200 * time.Millisecond,
				MinSize:      10,
				MaxSize:      20,
			},
			expectedRequests: 1,
			expectedItems:    8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := requesttest.NewSink()
			cfg := newTestConfig()
			cfg.Batch = tt.batchCfg
			qb, err := NewQueueBatch(newFakeRequestSettings(), cfg, sink.Export)
			require.NoError(t, err)
			require.NoError(t, qb.Start(context.Background(), componenttest.NewNopHost()))

			require.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 4}))
			require.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 4}))

			// the first two requests should be blocked by the batchSender.
			time.Sleep(50 * time.Millisecond)
			assert.Equal(t, 0, sink.RequestsCount())

			// the third request should trigger the export and cause an error.
			sink.SetExportErr(errors.New("transient error"))
			errReq := &requesttest.FakeRequest{Items: 20}
			require.NoError(t, qb.Send(context.Background(), errReq))

			// the batch should be dropped since the queue doesn't have re-queuing enabled.
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == tt.expectedRequests &&
					sink.ItemsCount() == tt.expectedItems &&
					qb.queue.Size() == 0
			}, 1*time.Second, 10*time.Millisecond)

			require.NoError(t, qb.Shutdown(context.Background()))
		})
	}
}

func TestQueueBatch_MergeOrSplit(t *testing.T) {
	sink := requesttest.NewSink()
	cfg := newTestConfig()
	cfg.Batch = &BatchConfig{
		FlushTimeout: 100 * time.Millisecond,
		MinSize:      5,
		MaxSize:      10,
	}
	qb, err := NewQueueBatch(newFakeRequestSettings(), cfg, sink.Export)
	require.NoError(t, err)
	require.NoError(t, qb.Start(context.Background(), componenttest.NewNopHost()))

	// should be sent right away by reaching the minimum items size.
	require.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 8}))
	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 1 && sink.ItemsCount() == 8
	}, 1*time.Second, 10*time.Millisecond)

	// big request should be broken down into two requests, both are sent right away.
	require.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 17}))
	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 3 && sink.ItemsCount() == 25
	}, 1*time.Second, 10*time.Millisecond)

	// request that cannot be split should be dropped.
	require.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{
		Items:    11,
		MergeErr: errors.New("split error"),
	}))

	// big request should be broken down into two requests, both are sent right away.
	require.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 13}))
	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 5 && sink.ItemsCount() == 38
	}, 1*time.Second, 10*time.Millisecond)
	require.NoError(t, qb.Shutdown(context.Background()))
}

func TestQueueBatch_Shutdown(t *testing.T) {
	sink := requesttest.NewSink()
	qb, err := NewQueueBatch(newFakeRequestSettings(), newTestConfig(), sink.Export)
	require.NoError(t, err)
	require.NoError(t, qb.Start(context.Background(), componenttest.NewNopHost()))

	require.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 3}))

	// To make the request reached the batchSender before shutdown.
	time.Sleep(50 * time.Millisecond)

	require.NoError(t, qb.Shutdown(context.Background()))

	// shutdown should force sending the batch
	assert.Equal(t, 1, sink.RequestsCount())
	assert.Equal(t, 3, sink.ItemsCount())
}

func TestQueueBatch_BatchBlocking(t *testing.T) {
	sink := requesttest.NewSink()
	cfg := newTestConfig()
	cfg.WaitForResult = true
	cfg.Batch.MinSize = 3
	qb, err := NewQueueBatch(newFakeRequestSettings(), cfg, sink.Export)
	require.NoError(t, err)
	require.NoError(t, qb.Start(context.Background(), componenttest.NewNopHost()))

	// send 6 blocking requests
	wg := sync.WaitGroup{}
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 1, Delay: 10 * time.Millisecond}))
		}()
	}
	wg.Wait()

	// should be sent in two batches since the batch size is 3
	assert.Equal(t, 2, sink.RequestsCount())
	assert.Equal(t, 6, sink.ItemsCount())

	require.NoError(t, qb.Shutdown(context.Background()))
}

// Validate that the batch is cancelled once the first request in the request is cancelled
func TestQueueBatch_BatchCancelled(t *testing.T) {
	sink := requesttest.NewSink()
	cfg := newTestConfig()
	cfg.WaitForResult = true
	cfg.Batch.MinSize = 2
	qb, err := NewQueueBatch(newFakeRequestSettings(), cfg, sink.Export)
	require.NoError(t, err)
	require.NoError(t, qb.Start(context.Background(), componenttest.NewNopHost()))

	// send 2 blocking requests
	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.ErrorIs(t, qb.Send(ctx, &requesttest.FakeRequest{Items: 1, Delay: 100 * time.Millisecond}), context.Canceled)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond) // ensure this call is the second
		assert.ErrorIs(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 1, Delay: 100 * time.Millisecond}), context.Canceled)
	}()
	cancel() // canceling the first request should cancel the whole batch
	wg.Wait()

	// nothing should be delivered
	assert.Equal(t, 0, sink.RequestsCount())
	assert.Equal(t, 0, sink.ItemsCount())

	require.NoError(t, qb.Shutdown(context.Background()))
}

func TestQueueBatch_DrainActiveRequests(t *testing.T) {
	sink := requesttest.NewSink()
	cfg := newTestConfig()
	cfg.WaitForResult = true
	cfg.Batch.MinSize = 2
	qb, err := NewQueueBatch(newFakeRequestSettings(), cfg, sink.Export)
	require.NoError(t, err)
	require.NoError(t, qb.Start(context.Background(), componenttest.NewNopHost()))

	// send 3 blocking requests with a timeout
	go func() {
		assert.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 1, Delay: 40 * time.Millisecond}))
	}()
	go func() {
		assert.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 1, Delay: 40 * time.Millisecond}))
	}()
	go func() {
		assert.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 1, Delay: 40 * time.Millisecond}))
	}()

	// give time for the first two requests to be batched
	time.Sleep(20 * time.Millisecond)

	// Shutdown should force the active batch to be dispatched and wait for all batches to be delivered.
	// It should take 120 milliseconds to complete.
	require.NoError(t, qb.Shutdown(context.Background()))

	assert.Equal(t, 2, sink.RequestsCount())
	assert.Equal(t, 3, sink.ItemsCount())
}

func TestQueueBatchWithTimeout(t *testing.T) {
	sink := requesttest.NewSink()
	cfg := newTestConfig()
	cfg.WaitForResult = true
	cfg.Batch.MinSize = 10
	qb, err := NewQueueBatch(newFakeRequestSettings(), cfg, sink.Export)
	require.NoError(t, err)
	require.NoError(t, qb.Start(context.Background(), componenttest.NewNopHost()))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	// Send 3 concurrent requests that should be merged in one batch
	wg := sync.WaitGroup{}
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			assert.NoError(t, qb.Send(ctx, &requesttest.FakeRequest{Items: 4}))
			wg.Done()
		}()
	}
	wg.Wait()
	assert.EqualValues(t, 1, sink.RequestsCount())
	assert.EqualValues(t, 12, sink.ItemsCount())

	// 3 requests with a 90ms cumulative delay must be cancelled by the timeout sender
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			assert.Error(t, qb.Send(ctx, &requesttest.FakeRequest{Items: 4, Delay: 30 * time.Millisecond}))
			wg.Done()
		}()
	}
	wg.Wait()

	require.NoError(t, qb.Shutdown(context.Background()))

	// The sink should not change
	assert.EqualValues(t, 1, sink.RequestsCount())
	assert.EqualValues(t, 12, sink.ItemsCount())
}

func TestQueueBatchTimerResetNoConflict(t *testing.T) {
	sink := requesttest.NewSink()
	cfg := newTestConfig()
	cfg.WaitForResult = true
	cfg.Batch.MinSize = 8
	cfg.Batch.FlushTimeout = 100 * time.Millisecond
	qb, err := NewQueueBatch(newFakeRequestSettings(), cfg, sink.Export)
	require.NoError(t, err)
	require.NoError(t, qb.Start(context.Background(), componenttest.NewNopHost()))

	// Send 2 concurrent requests that should be merged in one batch in the same interval as the flush timer
	go func() {
		assert.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 4}))
	}()
	time.Sleep(30 * time.Millisecond)
	go func() {
		assert.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 4}))
	}()

	// The batch should be sent either with the flush interval or by reaching the minimum items size with no conflict
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.LessOrEqual(c, 1, sink.RequestsCount())
		assert.Equal(c, 8, sink.ItemsCount())
	}, 1*time.Second, 10*time.Millisecond)

	require.NoError(t, qb.Shutdown(context.Background()))
}

func TestQueueBatchTimerFlush(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping flaky test on Windows, see https://github.com/open-telemetry/opentelemetry-collector/issues/10802")
	}
	sink := requesttest.NewSink()
	cfg := newTestConfig()
	cfg.WaitForResult = true
	cfg.Batch.MinSize = 8
	cfg.Batch.FlushTimeout = 100 * time.Millisecond
	qb, err := NewQueueBatch(newFakeRequestSettings(), cfg, sink.Export)
	require.NoError(t, err)
	require.NoError(t, qb.Start(context.Background(), componenttest.NewNopHost()))
	time.Sleep(50 * time.Millisecond)

	// Send 2 concurrent requests that should be merged in one batch and sent immediately
	go func() {
		assert.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 4}))
	}()
	go func() {
		assert.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 4}))
	}()
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.LessOrEqual(c, 1, sink.RequestsCount())
		assert.Equal(c, 8, sink.ItemsCount())
	}, 30*time.Millisecond, 5*time.Millisecond)

	// Send another request that should be flushed after 100ms instead of 50ms since last flush
	go func() {
		assert.NoError(t, qb.Send(context.Background(), &requesttest.FakeRequest{Items: 4}))
	}()

	// Confirm that it is not flushed in 50ms
	time.Sleep(60 * time.Millisecond)
	assert.LessOrEqual(t, 1, sink.RequestsCount())
	assert.Equal(t, 8, sink.ItemsCount())

	// Confirm that it is flushed after 100ms (using 60+50=110 here to be safe)
	time.Sleep(50 * time.Millisecond)
	assert.LessOrEqual(t, 2, sink.RequestsCount())
	assert.Equal(t, 12, sink.ItemsCount())
	require.NoError(t, qb.Shutdown(context.Background()))
}

func newTestConfig() Config {
	return Config{
		Enabled:         true,
		Sizer:           request.SizerTypeItems,
		NumConsumers:    runtime.NumCPU(),
		QueueSize:       100_000,
		BlockOnOverflow: true,
		Batch: &BatchConfig{
			FlushTimeout: 200 * time.Millisecond,
			MinSize:      2048,
		},
	}
}
