// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/internal/storagetest"
)

type fakeEncoding struct {
	mr request.Request
}

func (f fakeEncoding) Marshal(request.Request) ([]byte, error) {
	return []byte("mockRequest"), nil
}

func (f fakeEncoding) Unmarshal([]byte) (request.Request, error) {
	return f.mr, nil
}

func newFakeEncoding(mr request.Request) exporterqueue.Encoding[request.Request] {
	return &fakeEncoding{mr: mr}
}

var defaultQueueSettings = exporterqueue.Settings[request.Request]{
	Signal:           defaultSignal,
	ExporterSettings: defaultSettings,
}

func TestQueueBatcherStopWhileWaiting(t *testing.T) {
	sink := requesttest.NewSink()
	qCfg := exporterqueue.NewDefaultConfig()
	qCfg.NumConsumers = 1
	be, err := NewQueueSender(
		defaultQueueSettings, qCfg, exporterbatcher.Config{}, "", newSender(sink.Export))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	sink.SetExportErr(errors.New("transient error"))
	require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4}))
	// Enqueue another request to ensure when calling shutdown we drain the queue.
	require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 3, Delay: 100 * time.Millisecond}))
	require.LessOrEqual(t, int64(1), be.queue.Size())

	require.NoError(t, be.Shutdown(context.Background()))
	assert.EqualValues(t, 1, sink.RequestsCount())
	assert.EqualValues(t, 3, sink.ItemsCount())
	require.Zero(t, be.queue.Size())
}

func TestQueueBatcherDoNotPreserveCancellation(t *testing.T) {
	sink := requesttest.NewSink()
	qCfg := exporterqueue.NewDefaultConfig()
	qCfg.NumConsumers = 1
	be, err := NewQueueSender(defaultQueueSettings, qCfg, exporterbatcher.Config{}, "", newSender(sink.Export))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	ctx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()

	require.NoError(t, be.Send(ctx, &requesttest.FakeRequest{Items: 4}))
	require.NoError(t, be.Shutdown(context.Background()))

	assert.EqualValues(t, 1, sink.RequestsCount())
	assert.EqualValues(t, 4, sink.ItemsCount())
	require.Zero(t, be.queue.Size())
}

func TestQueueBatcherHappyPath(t *testing.T) {
	qCfg := exporterqueue.Config{
		Enabled:      true,
		QueueSize:    10,
		NumConsumers: 1,
	}
	sink := requesttest.NewSink()
	be, err := NewQueueSender(defaultQueueSettings, qCfg, exporterbatcher.Config{}, "", newSender(sink.Export))
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: i}))
	}

	// expect queue to be full
	require.Error(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 2}))

	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 10 && sink.ItemsCount() == 45
	}, 1*time.Second, 10*time.Millisecond)
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestQueueFailedRequestDropped(t *testing.T) {
	qSet := defaultQueueSettings
	logger, observed := observer.New(zap.ErrorLevel)
	qSet.ExporterSettings.Logger = zap.New(logger)
	be, err := NewQueueSender(
		qSet, exporterqueue.NewDefaultConfig(), exporterbatcher.Config{}, "", newSender(func(context.Context, request.Request) error { return errors.New("some error") }))

	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 2}))
	require.NoError(t, be.Shutdown(context.Background()))
	assert.Len(t, observed.All(), 1)
	assert.Equal(t, "Exporting failed. Dropping data.", observed.All()[0].Message)
}

func TestQueueBatcherPersistenceEnabled(t *testing.T) {
	qSet := exporterqueue.Settings[request.Request]{
		Signal:           defaultSignal,
		ExporterSettings: defaultSettings,
		Encoding:         newFakeEncoding(&requesttest.FakeRequest{}),
	}
	qCfg := exporterqueue.NewDefaultConfig()
	storageID := component.MustNewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID
	be, err := NewQueueSender(qSet, qCfg, exporterbatcher.Config{}, "", newNoopExportSender())
	require.NoError(t, err)

	extensions := map[component.ID]component.Component{
		storageID: storagetest.NewMockStorageExtension(nil),
	}
	host := &MockHost{Ext: extensions}

	// we start correctly with a file storage extension
	require.NoError(t, be.Start(context.Background(), host))
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestQueueBatcherPersistenceEnabledStorageError(t *testing.T) {
	storageError := errors.New("could not get storage client")

	qSet := exporterqueue.Settings[request.Request]{
		Signal:           defaultSignal,
		ExporterSettings: defaultSettings,
		Encoding:         newFakeEncoding(&requesttest.FakeRequest{}),
	}
	qCfg := exporterqueue.NewDefaultConfig()
	storageID := component.MustNewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID
	be, err := NewQueueSender(qSet, qCfg, exporterbatcher.Config{}, "", newNoopExportSender())
	require.NoError(t, err)

	extensions := map[component.ID]component.Component{
		storageID: storagetest.NewMockStorageExtension(storageError),
	}
	host := &MockHost{Ext: extensions}

	// we fail to start if we get an error creating the storage client
	require.Error(t, be.Start(context.Background(), host), "could not get storage client")
}

func TestQueueBatcherPersistentEnabled_NoDataLossOnShutdown(t *testing.T) {
	qCfg := exporterqueue.NewDefaultConfig()
	qCfg.NumConsumers = 1
	storageID := component.MustNewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID

	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = time.Millisecond
	rCfg.MaxElapsedTime = 0 // retry infinitely so shutdown can be triggered
	rs := newRetrySender(rCfg, defaultSettings, newSender(func(context.Context, request.Request) error { return errors.New("transient error") }))
	require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))

	mockReq := &requesttest.FakeRequest{Items: 2}
	qSet := exporterqueue.Settings[request.Request]{
		Signal:           defaultSignal,
		ExporterSettings: defaultSettings,
		Encoding:         newFakeEncoding(mockReq),
	}
	be, err := NewQueueSender(qSet, qCfg, exporterbatcher.Config{}, "", rs)
	require.NoError(t, err)

	extensions := map[component.ID]component.Component{
		storageID: storagetest.NewMockStorageExtension(nil),
	}
	host := &MockHost{Ext: extensions}

	require.NoError(t, be.Start(context.Background(), host))

	// Invoke queuedRetrySender so the producer will put the item for consumer to poll
	require.NoError(t, be.Send(context.Background(), mockReq))

	// first wait for the item to be consumed from the queue
	assert.Eventually(t, func() bool {
		return be.queue.Size() == 0
	}, 1*time.Second, 10*time.Millisecond)

	// shuts down the exporter, unsent data should be preserved as in-flight data in the persistent queue.
	require.NoError(t, rs.Shutdown(context.Background()))
	require.NoError(t, be.Shutdown(context.Background()))

	// start the exporter again replacing the preserved mockRequest in the unmarshaler with a new one that doesn't fail.
	sink := requesttest.NewSink()
	replacedReq := &requesttest.FakeRequest{Items: 7}
	qSet.Encoding = newFakeEncoding(replacedReq)
	be, err = NewQueueSender(qSet, qCfg, exporterbatcher.Config{}, "", newSender(sink.Export))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), host))

	assert.Eventually(t, func() bool {
		return sink.ItemsCount() == 7 && sink.RequestsCount() == 1
	}, 1*time.Second, 10*time.Millisecond)
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestQueueBatcherNoStartShutdown(t *testing.T) {
	qs, err := NewQueueSender(defaultQueueSettings, exporterqueue.NewDefaultConfig(), exporterbatcher.NewDefaultConfig(), "", newNoopExportSender())
	require.NoError(t, err)
	assert.NoError(t, qs.Shutdown(context.Background()))
}

func TestQueueBatcher_Merge(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping flaky test on Windows, see https://github.com/open-telemetry/opentelemetry-collector/issues/10758")
	}

	tests := []struct {
		name     string
		batchCfg exporterbatcher.Config
	}{
		{
			name: "split_disabled",
			batchCfg: func() exporterbatcher.Config {
				qCfg := exporterbatcher.NewDefaultConfig()
				qCfg.MinSize = 10
				qCfg.FlushTimeout = 100 * time.Millisecond
				return qCfg
			}(),
		},
		{
			name: "split_high_limit",
			batchCfg: func() exporterbatcher.Config {
				qCfg := exporterbatcher.NewDefaultConfig()
				qCfg.MinSize = 10
				qCfg.FlushTimeout = 100 * time.Millisecond
				qCfg.MaxSize = 1000
				return qCfg
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := requesttest.NewSink()
			be, err := NewQueueSender(defaultQueueSettings, exporterqueue.NewDefaultConfig(), tt.batchCfg, "", newSender(sink.Export))
			require.NoError(t, err)
			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, be.Shutdown(context.Background()))
			})

			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 8}))
			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 3}))

			// the first two requests should be merged into one and sent by reaching the minimum items size
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 1 && sink.ItemsCount() == 11
			}, 50*time.Millisecond, 10*time.Millisecond)

			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 3}))
			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 1}))

			// the third and fifth requests should be sent by reaching the timeout
			// the fourth request should be ignored because of the merge error.
			time.Sleep(50 * time.Millisecond)

			// should be ignored because of the merge error.
			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{
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

func TestQueueBatcher_BatchExportError(t *testing.T) {
	tests := []struct {
		name             string
		batchCfg         exporterbatcher.Config
		expectedRequests int
		expectedItems    int
	}{
		{
			name: "merge_only",
			batchCfg: func() exporterbatcher.Config {
				cfg := exporterbatcher.NewDefaultConfig()
				cfg.MinSize = 10
				return cfg
			}(),
		},
		{
			name: "merge_without_split_triggered",
			batchCfg: func() exporterbatcher.Config {
				cfg := exporterbatcher.NewDefaultConfig()
				cfg.MinSize = 10
				cfg.MaxSize = 200
				return cfg
			}(),
		},
		{
			name: "merge_with_split_triggered",
			batchCfg: func() exporterbatcher.Config {
				cfg := exporterbatcher.NewDefaultConfig()
				cfg.MinSize = 10
				cfg.MaxSize = 20
				return cfg
			}(),
			expectedRequests: 1,
			expectedItems:    8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := requesttest.NewSink()
			qSet := exporterqueue.Settings[request.Request]{
				Signal:           defaultSignal,
				ExporterSettings: defaultSettings,
			}
			be, err := NewQueueSender(qSet, exporterqueue.NewDefaultConfig(), tt.batchCfg, "", newSender(sink.Export))
			require.NoError(t, err)
			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4}))
			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4}))

			// the first two requests should be blocked by the batchSender.
			time.Sleep(50 * time.Millisecond)
			assert.Equal(t, 0, sink.RequestsCount())

			// the third request should trigger the export and cause an error.
			sink.SetExportErr(errors.New("transient error"))
			errReq := &requesttest.FakeRequest{Items: 20}
			require.NoError(t, be.Send(context.Background(), errReq))

			// the batch should be dropped since the queue doesn't have re-queuing enabled.
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == tt.expectedRequests &&
					sink.ItemsCount() == tt.expectedItems &&
					be.queue.Size() == 0
			}, 1*time.Second, 10*time.Millisecond)

			require.NoError(t, be.Shutdown(context.Background()))
		})
	}
}

func TestQueueBatcher_MergeOrSplit(t *testing.T) {
	sink := requesttest.NewSink()
	batchCfg := exporterbatcher.NewDefaultConfig()
	batchCfg.MinSize = 5
	batchCfg.MaxSize = 10
	batchCfg.FlushTimeout = 100 * time.Millisecond
	be, err := NewQueueSender(defaultQueueSettings, exporterqueue.NewDefaultConfig(), batchCfg, "", newSender(sink.Export))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	// should be sent right away by reaching the minimum items size.
	require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 8}))
	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 1 && sink.ItemsCount() == 8
	}, 1*time.Second, 10*time.Millisecond)

	// big request should be broken down into two requests, both are sent right away.
	require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 17}))
	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 3 && sink.ItemsCount() == 25
	}, 1*time.Second, 10*time.Millisecond)

	// request that cannot be split should be dropped.
	require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{
		Items:    11,
		MergeErr: errors.New("split error"),
	}))

	// big request should be broken down into two requests, both are sent right away.
	require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 13}))
	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 5 && sink.ItemsCount() == 38
	}, 1*time.Second, 10*time.Millisecond)
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestQueueBatcher_Shutdown(t *testing.T) {
	sink := requesttest.NewSink()
	batchCfg := exporterbatcher.NewDefaultConfig()
	batchCfg.MinSize = 10
	be, err := NewQueueSender(defaultQueueSettings, exporterqueue.NewDefaultConfig(), batchCfg, "", newSender(sink.Export))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 3}))

	// To make the request reached the batchSender before shutdown.
	time.Sleep(50 * time.Millisecond)

	require.NoError(t, be.Shutdown(context.Background()))

	// shutdown should force sending the batch
	assert.Equal(t, 1, sink.RequestsCount())
	assert.Equal(t, 3, sink.ItemsCount())
}

func TestQueueBatcher_BatchBlocking(t *testing.T) {
	sink := requesttest.NewSink()
	bCfg := exporterbatcher.NewDefaultConfig()
	bCfg.MinSize = 3
	be, err := NewQueueSender(defaultQueueSettings, exporterqueue.Config{}, bCfg, "", newSender(sink.Export))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	// send 6 blocking requests
	wg := sync.WaitGroup{}
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 1, Delay: 10 * time.Millisecond}))
		}()
	}
	wg.Wait()

	// should be sent in two batches since the batch size is 3
	assert.Equal(t, 2, sink.RequestsCount())
	assert.Equal(t, 6, sink.ItemsCount())

	require.NoError(t, be.Shutdown(context.Background()))
}

// Validate that the batch is cancelled once the first request in the request is cancelled
func TestQueueBatcher_BatchCancelled(t *testing.T) {
	sink := requesttest.NewSink()
	bCfg := exporterbatcher.NewDefaultConfig()
	bCfg.MinSize = 2
	be, err := NewQueueSender(defaultQueueSettings, exporterqueue.Config{}, bCfg, "", newSender(sink.Export))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	// send 2 blocking requests
	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.ErrorIs(t, be.Send(ctx, &requesttest.FakeRequest{Items: 1, Delay: 100 * time.Millisecond}), context.Canceled)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond) // ensure this call is the second
		assert.ErrorIs(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 1, Delay: 100 * time.Millisecond}), context.Canceled)
	}()
	cancel() // canceling the first request should cancel the whole batch
	wg.Wait()

	// nothing should be delivered
	assert.Equal(t, 0, sink.RequestsCount())
	assert.Equal(t, 0, sink.ItemsCount())

	require.NoError(t, be.Shutdown(context.Background()))
}

func TestQueueBatcher_DrainActiveRequests(t *testing.T) {
	sink := requesttest.NewSink()
	bCfg := exporterbatcher.NewDefaultConfig()
	bCfg.MinSize = 2

	be, err := NewQueueSender(defaultQueueSettings, exporterqueue.Config{}, bCfg, "", newSender(sink.Export))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	// send 3 blocking requests with a timeout
	go func() {
		assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 1, Delay: 40 * time.Millisecond}))
	}()
	go func() {
		assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 1, Delay: 40 * time.Millisecond}))
	}()
	go func() {
		assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 1, Delay: 40 * time.Millisecond}))
	}()

	// give time for the first two requests to be batched
	time.Sleep(20 * time.Millisecond)

	// Shutdown should force the active batch to be dispatched and wait for all batches to be delivered.
	// It should take 120 milliseconds to complete.
	require.NoError(t, be.Shutdown(context.Background()))

	assert.Equal(t, 2, sink.RequestsCount())
	assert.Equal(t, 3, sink.ItemsCount())
}

func TestQueueBatcherUnstartedShutdown(t *testing.T) {
	be, err := NewQueueSender(defaultQueueSettings, exporterqueue.NewDefaultConfig(), exporterbatcher.NewDefaultConfig(), "", newNoopExportSender())
	require.NoError(t, err)
	err = be.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestQueueBatcherWithTimeout(t *testing.T) {
	sink := requesttest.NewSink()
	bCfg := exporterbatcher.NewDefaultConfig()
	bCfg.MinSize = 10

	be, err := NewQueueSender(defaultQueueSettings, exporterqueue.Config{}, bCfg, "", newSender(sink.Export))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	// Send 3 concurrent requests that should be merged in one batch
	wg := sync.WaitGroup{}
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			assert.NoError(t, be.Send(ctx, &requesttest.FakeRequest{Items: 4}))
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
			assert.Error(t, be.Send(ctx, &requesttest.FakeRequest{Items: 4, Delay: 30 * time.Millisecond}))
			wg.Done()
		}()
	}
	wg.Wait()

	require.NoError(t, be.Shutdown(context.Background()))

	// The sink should not change
	assert.EqualValues(t, 1, sink.RequestsCount())
	assert.EqualValues(t, 12, sink.ItemsCount())
}

func TestQueueBatcherTimerResetNoConflict(t *testing.T) {
	sink := requesttest.NewSink()
	bCfg := exporterbatcher.NewDefaultConfig()
	bCfg.MinSize = 8
	bCfg.FlushTimeout = 100 * time.Millisecond
	be, err := NewQueueSender(defaultQueueSettings, exporterqueue.Config{}, bCfg, "", newSender(sink.Export))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	// Send 2 concurrent requests that should be merged in one batch in the same interval as the flush timer
	go func() {
		assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4}))
	}()
	time.Sleep(30 * time.Millisecond)
	go func() {
		assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4}))
	}()

	// The batch should be sent either with the flush interval or by reaching the minimum items size with no conflict
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.LessOrEqual(c, 1, sink.RequestsCount())
		assert.Equal(c, 8, sink.ItemsCount())
	}, 1*time.Second, 10*time.Millisecond)

	require.NoError(t, be.Shutdown(context.Background()))
}

func TestQueueBatcherTimerFlush(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping flaky test on Windows, see https://github.com/open-telemetry/opentelemetry-collector/issues/10802")
	}
	sink := requesttest.NewSink()
	bCfg := exporterbatcher.NewDefaultConfig()
	bCfg.MinSize = 8
	bCfg.FlushTimeout = 100 * time.Millisecond
	be, err := NewQueueSender(defaultQueueSettings, exporterqueue.Config{}, bCfg, "", newSender(sink.Export))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	time.Sleep(50 * time.Millisecond)

	// Send 2 concurrent requests that should be merged in one batch and sent immediately
	go func() {
		assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4}))
	}()
	go func() {
		assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4}))
	}()
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.LessOrEqual(c, 1, sink.RequestsCount())
		assert.Equal(c, 8, sink.ItemsCount())
	}, 30*time.Millisecond, 5*time.Millisecond)

	// Send another request that should be flushed after 100ms instead of 50ms since last flush
	go func() {
		assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4}))
	}()

	// Confirm that it is not flushed in 50ms
	time.Sleep(60 * time.Millisecond)
	assert.LessOrEqual(t, 1, sink.RequestsCount())
	assert.Equal(t, 8, sink.ItemsCount())

	// Confirm that it is flushed after 100ms (using 60+50=110 here to be safe)
	time.Sleep(50 * time.Millisecond)
	assert.LessOrEqual(t, 2, sink.RequestsCount())
	assert.Equal(t, 12, sink.ItemsCount())
	require.NoError(t, be.Shutdown(context.Background()))
}
