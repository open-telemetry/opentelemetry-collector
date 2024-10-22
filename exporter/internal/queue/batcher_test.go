// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/internal"
)

func testExportFunc(ctx context.Context, req internal.Request) error {
	return req.Export(ctx)
}

func TestBatcher_Merge(t *testing.T) {
	cfg := exporterbatcher.NewDefaultConfig()
	cfg.MinSizeItems = 10
	cfg.FlushTimeout = 100 * time.Millisecond

	tests := []struct {
		name       string
		batcherCfg exporterbatcher.Config
	}{
		{
			name:       "split_disabled",
			batcherCfg: cfg,
		},
		{
			name: "split_high_limit",
			batcherCfg: func() exporterbatcher.Config {
				c := cfg
				c.MaxSizeItems = 1000
				return c
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewBoundedMemoryQueue[internal.Request](
				MemoryQueueSettings[internal.Request]{
					Sizer:    &RequestSizer[internal.Request]{},
					Capacity: 10,
				})
			maxWorkers := 1
			ba := NewBatcher(cfg, q, maxWorkers, testExportFunc)

			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, ba.Shutdown(context.Background()))
			})

			sink := newFakeRequestSink()

			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 8, sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 3, sink: sink}))

			// the first two requests should be merged into one and sent by reaching the minimum items size
			assert.Eventually(t, func() bool {
				return sink.requestsCount.Load() == 1 && sink.itemsCount.Load() == 11
			}, 50*time.Millisecond, 10*time.Millisecond)

			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 3, sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 1, sink: sink}))

			// the third and fifth requests should be sent by reaching the timeout
			// the fourth request should be ignored because of the merge error.
			time.Sleep(50 * time.Millisecond)

			// should be ignored because of the merge error.
			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 3, sink: sink,
				mergeErr: errors.New("merge error")}))

			assert.Equal(t, int64(1), sink.requestsCount.Load())
			assert.Eventually(t, func() bool {
				return sink.requestsCount.Load() == 2 && sink.itemsCount.Load() == 15
			}, 100*time.Millisecond, 10*time.Millisecond)
		})
	}
}

func TestBatcher_BatchExportError(t *testing.T) {
	cfg := exporterbatcher.NewDefaultConfig()
	cfg.MinSizeItems = 10
	cfg.FlushTimeout = 10 * time.Second // so batches are never flushed because of timeout for this test.
	tests := []struct {
		name             string
		batcherCfg       exporterbatcher.Config
		expectedRequests int64
		expectedItems    int64
	}{
		{
			name:       "merge_only",
			batcherCfg: cfg,
		},
		{
			name: "merge_without_split_triggered",
			batcherCfg: func() exporterbatcher.Config {
				c := cfg
				c.MaxSizeItems = 200
				return c
			}(),
		},
		{
			name: "merge_with_split_triggered",
			batcherCfg: func() exporterbatcher.Config {
				c := cfg
				c.MaxSizeItems = 20
				return c
			}(),
			expectedRequests: 1,
			expectedItems:    20,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewBoundedMemoryQueue[internal.Request](
				MemoryQueueSettings[internal.Request]{
					Sizer:    &RequestSizer[internal.Request]{},
					Capacity: 10,
				})

			maxWorkers := 1
			ba := NewBatcher(tt.batcherCfg, q, maxWorkers, testExportFunc)

			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, ba.Shutdown(context.Background()))
			})

			sink := newFakeRequestSink()

			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 4, sink: sink}))
			require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 4, sink: sink}))

			// the first two requests should be blocked because they don't hit the size threshold.
			time.Sleep(50 * time.Millisecond)
			assert.Equal(t, int64(0), sink.requestsCount.Load())

			// the third request should trigger the export and cause an error.
			errReq := &fakeRequest{items: 20, exportErr: errors.New("transient error"), sink: sink}
			require.NoError(t, q.Offer(context.Background(), errReq))

			// the batch should be dropped since the queue doesn't have requeuing enabled.
			assert.Eventually(t, func() bool {
				return sink.requestsCount.Load() == tt.expectedRequests &&
					sink.itemsCount.Load() == tt.expectedItems &&
					q.Size() == 0
			}, 100*time.Millisecond, 10*time.Millisecond)
		})
	}
}

func TestBatcher_MergeOrSplit(t *testing.T) {
	cfg := exporterbatcher.NewDefaultConfig()
	cfg.MinSizeItems = 5
	cfg.MaxSizeItems = 10
	cfg.FlushTimeout = 100 * time.Millisecond

	q := NewBoundedMemoryQueue[internal.Request](
		MemoryQueueSettings[internal.Request]{
			Sizer:    &RequestSizer[internal.Request]{},
			Capacity: 10,
		})
	maxWorkers := 1
	ba := NewBatcher(cfg, q, maxWorkers, testExportFunc)

	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	sink := newFakeRequestSink()

	// should be sent right away by reaching the minimum items size.
	require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 8, sink: sink}))
	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 1 && sink.itemsCount.Load() == 8
	}, 50*time.Millisecond, 10*time.Millisecond)

	// big request should be broken down into two requests, both are sent right away.
	require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 17, sink: sink}))
	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 3 && sink.itemsCount.Load() == 25
	}, 50*time.Millisecond, 10*time.Millisecond)

	// request that cannot be split should be dropped.
	require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 11, sink: sink,
		mergeErr: errors.New("split error")}))

	// big request should be broken down into two requests, both are sent right away.
	require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 13, sink: sink}))

	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 5 && sink.itemsCount.Load() == 38
	}, 50*time.Millisecond, 10*time.Millisecond)
}

func TestBatcher_Shutdown(t *testing.T) {
	batchCfg := exporterbatcher.NewDefaultConfig()
	batchCfg.MinSizeItems = 10
	q := NewBoundedMemoryQueue[internal.Request](
		MemoryQueueSettings[internal.Request]{
			Sizer:    &RequestSizer[internal.Request]{},
			Capacity: 10,
		})
	maxWorkers := 1
	ba := NewBatcher(batchCfg, q, maxWorkers, testExportFunc)

	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))

	sink := newFakeRequestSink()
	require.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 3, sink: sink}))

	// To make the request reached the batchSender before shutdown.
	time.Sleep(50 * time.Millisecond)

	require.NoError(t, ba.Shutdown(context.Background()))

	// shutdown should force sending the batch
	assert.Equal(t, int64(1), sink.requestsCount.Load())
	assert.Equal(t, int64(3), sink.itemsCount.Load())
}

func TestBatcher_UnstartedShutdown(t *testing.T) {
	batchCfg := exporterbatcher.NewDefaultConfig()
	batchCfg.MinSizeItems = 10
	q := NewBoundedMemoryQueue[internal.Request](
		MemoryQueueSettings[internal.Request]{
			Sizer:    &RequestSizer[internal.Request]{},
			Capacity: 10,
		})
	maxWorkers := 1
	ba := NewBatcher(batchCfg, q, maxWorkers, testExportFunc)

	err := ba.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestBatchSenderTimerFlush(t *testing.T) {
	bCfg := exporterbatcher.NewDefaultConfig()
	bCfg.MinSizeItems = 8
	bCfg.FlushTimeout = 100 * time.Millisecond
	q := NewBoundedMemoryQueue[internal.Request](
		MemoryQueueSettings[internal.Request]{
			Sizer:    &RequestSizer[internal.Request]{},
			Capacity: 10,
		})
	maxWorkers := 1
	ba := NewBatcher(bCfg, q, maxWorkers, testExportFunc)

	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	sink := newFakeRequestSink()
	time.Sleep(50 * time.Millisecond)

	// Send 2 concurrent requests that should be merged in one batch and sent immediately
	go func() {
		assert.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 4, sink: sink}))
	}()
	go func() {
		assert.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 4, sink: sink}))
	}()
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.LessOrEqual(c, int64(1), sink.requestsCount.Load())
		assert.EqualValues(c, 8, sink.itemsCount.Load())
	}, 30*time.Millisecond, 5*time.Millisecond)

	// Send another request that should be flushed after 100ms instead of 50ms since last flush
	go func() {
		assert.NoError(t, q.Offer(context.Background(), &fakeRequest{items: 4, sink: sink}))
	}()

	// Confirm that it is not flushed in 50ms
	time.Sleep(60 * time.Millisecond)
	assert.LessOrEqual(t, int64(1), sink.requestsCount.Load())
	assert.EqualValues(t, 8, sink.itemsCount.Load())

	// Confirm that it is flushed after 100ms (using 60+50=110 here to be safe)
	time.Sleep(50 * time.Millisecond)
	assert.LessOrEqual(t, int64(2), sink.requestsCount.Load())
	assert.EqualValues(t, 12, sink.itemsCount.Load())
	require.NoError(t, ba.Shutdown(context.Background()))
}
