// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
)

func TestBatchSender_Merge(t *testing.T) {
	cfg := exporterbatcher.NewDefaultConfig()
	cfg.MinSizeItems = 10
	cfg.FlushTimeout = 100 * time.Millisecond

	tests := []struct {
		name          string
		batcherOption Option
	}{
		{
			name:          "split_disabled",
			batcherOption: WithBatcher(cfg, WithRequestBatchFuncs(fakeBatchMergeFunc, fakeBatchMergeSplitFunc)),
		},
		{
			name: "split_high_limit",
			batcherOption: func() Option {
				c := cfg
				c.MaxSizeItems = 1000
				return WithBatcher(c, WithRequestBatchFuncs(fakeBatchMergeFunc, fakeBatchMergeSplitFunc))
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			be := queueBatchExporter(t, tt.batcherOption)

			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, be.Shutdown(context.Background()))
			})

			sink := newFakeRequestSink()

			require.NoError(t, be.send(context.Background(), &fakeRequest{items: 8, sink: sink}))
			require.NoError(t, be.send(context.Background(), &fakeRequest{items: 3, sink: sink}))

			// the first two requests should be merged into one and sent by reaching the minimum items size
			assert.Eventually(t, func() bool {
				return sink.requestsCount.Load() == 1 && sink.itemsCount.Load() == 11
			}, 50*time.Millisecond, 10*time.Millisecond)

			require.NoError(t, be.send(context.Background(), &fakeRequest{items: 3, sink: sink}))
			require.NoError(t, be.send(context.Background(), &fakeRequest{items: 1, sink: sink}))

			// the third and fifth requests should be sent by reaching the timeout
			// the fourth request should be ignored because of the merge error.
			time.Sleep(50 * time.Millisecond)

			// should be ignored because of the merge error.
			require.NoError(t, be.send(context.Background(), &fakeRequest{items: 3, sink: sink,
				mergeErr: errors.New("merge error")}))

			assert.Equal(t, uint64(1), sink.requestsCount.Load())
			assert.Eventually(t, func() bool {
				return sink.requestsCount.Load() == 2 && sink.itemsCount.Load() == 15
			}, 100*time.Millisecond, 10*time.Millisecond)
		})
	}
}

func TestBatchSender_BatchExportError(t *testing.T) {
	cfg := exporterbatcher.NewDefaultConfig()
	cfg.MinSizeItems = 10
	tests := []struct {
		name             string
		batcherOption    Option
		expectedRequests uint64
		expectedItems    uint64
	}{
		{
			name:          "merge_only",
			batcherOption: WithBatcher(cfg, WithRequestBatchFuncs(fakeBatchMergeFunc, fakeBatchMergeSplitFunc)),
		},
		{
			name: "merge_without_split_triggered",
			batcherOption: func() Option {
				c := cfg
				c.MaxSizeItems = 200
				return WithBatcher(c, WithRequestBatchFuncs(fakeBatchMergeFunc, fakeBatchMergeSplitFunc))
			}(),
		},
		{
			name: "merge_with_split_triggered",
			batcherOption: func() Option {
				c := cfg
				c.MaxSizeItems = 20
				return WithBatcher(c, WithRequestBatchFuncs(fakeBatchMergeFunc, fakeBatchMergeSplitFunc))
			}(),
			expectedRequests: 1,
			expectedItems:    20,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			be := queueBatchExporter(t, tt.batcherOption)

			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, be.Shutdown(context.Background()))
			})

			sink := newFakeRequestSink()

			require.NoError(t, be.send(context.Background(), &fakeRequest{items: 4, sink: sink}))
			require.NoError(t, be.send(context.Background(), &fakeRequest{items: 4, sink: sink}))

			// the first two requests should be blocked by the batchSender.
			time.Sleep(50 * time.Millisecond)
			assert.Equal(t, uint64(0), sink.requestsCount.Load())

			// the third request should trigger the export and cause an error.
			errReq := &fakeRequest{items: 20, exportErr: errors.New("transient error"), sink: sink}
			require.NoError(t, be.send(context.Background(), errReq))

			// the batch should be dropped since the queue doesn't have requeuing enabled.
			assert.Eventually(t, func() bool {
				return sink.requestsCount.Load() == tt.expectedRequests &&
					sink.itemsCount.Load() == tt.expectedItems &&
					be.batchSender.(*batchSender).activeRequests.Load() == uint64(0) &&
					be.queueSender.(*queueSender).queue.Size() == 0
			}, 100*time.Millisecond, 10*time.Millisecond)
		})
	}
}

func TestBatchSender_MergeOrSplit(t *testing.T) {
	cfg := exporterbatcher.NewDefaultConfig()
	cfg.MinSizeItems = 5
	cfg.MaxSizeItems = 10
	cfg.FlushTimeout = 100 * time.Millisecond
	be := queueBatchExporter(t, WithBatcher(cfg, WithRequestBatchFuncs(fakeBatchMergeFunc, fakeBatchMergeSplitFunc)))

	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, be.Shutdown(context.Background()))
	})

	sink := newFakeRequestSink()

	// should be sent right away by reaching the minimum items size.
	require.NoError(t, be.send(context.Background(), &fakeRequest{items: 8, sink: sink}))
	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 1 && sink.itemsCount.Load() == 8
	}, 50*time.Millisecond, 10*time.Millisecond)

	// big request should be broken down into two requests, both are sent right away.
	require.NoError(t, be.send(context.Background(), &fakeRequest{items: 17, sink: sink}))

	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 3 && sink.itemsCount.Load() == 25
	}, 50*time.Millisecond, 10*time.Millisecond)

	// request that cannot be split should be dropped.
	require.NoError(t, be.send(context.Background(), &fakeRequest{items: 11, sink: sink,
		mergeErr: errors.New("split error")}))

	// big request should be broken down into two requests, both are sent right away.
	require.NoError(t, be.send(context.Background(), &fakeRequest{items: 13, sink: sink}))

	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 5 && sink.itemsCount.Load() == 38
	}, 50*time.Millisecond, 10*time.Millisecond)
}

func TestBatchSender_Shutdown(t *testing.T) {
	batchCfg := exporterbatcher.NewDefaultConfig()
	batchCfg.MinSizeItems = 10
	be := queueBatchExporter(t, WithBatcher(batchCfg, WithRequestBatchFuncs(fakeBatchMergeFunc, fakeBatchMergeSplitFunc)))

	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	sink := newFakeRequestSink()
	require.NoError(t, be.send(context.Background(), &fakeRequest{items: 3, sink: sink}))

	// To make the request reached the batchSender before shutdown.
	time.Sleep(50 * time.Millisecond)

	require.NoError(t, be.Shutdown(context.Background()))

	// shutdown should force sending the batch
	assert.Equal(t, uint64(1), sink.requestsCount.Load())
	assert.Equal(t, uint64(3), sink.itemsCount.Load())
}

func TestBatchSender_Disabled(t *testing.T) {
	cfg := exporterbatcher.NewDefaultConfig()
	cfg.Enabled = false
	cfg.MaxSizeItems = 5
	be, err := newBaseExporter(defaultSettings, defaultDataType, newNoopObsrepSender,
		WithBatcher(cfg, WithRequestBatchFuncs(fakeBatchMergeFunc, fakeBatchMergeSplitFunc)))
	require.NotNil(t, be)
	require.NoError(t, err)

	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, be.Shutdown(context.Background()))
	})

	sink := newFakeRequestSink()
	// should be sent right away without splitting because batching is disabled.
	require.NoError(t, be.send(context.Background(), &fakeRequest{items: 8, sink: sink}))
	assert.Equal(t, uint64(1), sink.requestsCount.Load())
	assert.Equal(t, uint64(8), sink.itemsCount.Load())
}

func TestBatchSender_InvalidMergeSplitFunc(t *testing.T) {
	invalidMergeSplitFunc := func(_ context.Context, _ exporterbatcher.MaxSizeConfig, _ Request, req2 Request) ([]Request,
		error) {
		// reply with invalid 0 length slice if req2 is more than 20 items
		if req2.(*fakeRequest).items > 20 {
			return []Request{}, nil
		}
		// otherwise reply with a single request.
		return []Request{req2}, nil
	}
	cfg := exporterbatcher.NewDefaultConfig()
	cfg.FlushTimeout = 50 * time.Millisecond
	cfg.MaxSizeItems = 20
	be := queueBatchExporter(t, WithBatcher(cfg, WithRequestBatchFuncs(fakeBatchMergeFunc, invalidMergeSplitFunc)))

	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, be.Shutdown(context.Background()))
	})

	sink := newFakeRequestSink()
	// first request should be ignored due to invalid merge/split function.
	require.NoError(t, be.send(context.Background(), &fakeRequest{items: 30, sink: sink}))
	// second request should be sent after reaching the timeout.
	require.NoError(t, be.send(context.Background(), &fakeRequest{items: 15, sink: sink}))
	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 1 && sink.itemsCount.Load() == 15
	}, 100*time.Millisecond, 10*time.Millisecond)
}

func TestBatchSender_PostShutdown(t *testing.T) {
	be, err := newBaseExporter(defaultSettings, defaultDataType, newNoopObsrepSender,
		WithBatcher(exporterbatcher.NewDefaultConfig(), WithRequestBatchFuncs(fakeBatchMergeFunc,
			fakeBatchMergeSplitFunc)))
	require.NotNil(t, be)
	require.NoError(t, err)
	assert.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, be.Shutdown(context.Background()))

	// Closed batch sender should act as a pass-through to not block queue draining.
	sink := newFakeRequestSink()
	require.NoError(t, be.send(context.Background(), &fakeRequest{items: 8, sink: sink}))
	assert.Equal(t, uint64(1), sink.requestsCount.Load())
	assert.Equal(t, uint64(8), sink.itemsCount.Load())
}

func TestBatchSender_ConcurrencyLimitReached(t *testing.T) {
	qCfg := exporterqueue.NewDefaultConfig()
	qCfg.NumConsumers = 2
	be, err := newBaseExporter(defaultSettings, defaultDataType, newNoopObsrepSender,
		WithBatcher(exporterbatcher.NewDefaultConfig(), WithRequestBatchFuncs(fakeBatchMergeFunc, fakeBatchMergeSplitFunc)),
		WithRequestQueue(qCfg, exporterqueue.NewMemoryQueueFactory[Request]()))
	require.NotNil(t, be)
	require.NoError(t, err)
	assert.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	sink := newFakeRequestSink()
	assert.NoError(t, be.send(context.Background(), &fakeRequest{items: 8, sink: sink}))

	// the second request should be sent by reaching max concurrency limit.
	assert.NoError(t, be.send(context.Background(), &fakeRequest{items: 8, sink: sink}))

	assert.Eventually(t, func() bool {
		return sink.requestsCount.Load() == 1 && sink.itemsCount.Load() == 16
	}, 100*time.Millisecond, 10*time.Millisecond)
}

func TestBatchSender_BatchBlocking(t *testing.T) {
	bCfg := exporterbatcher.NewDefaultConfig()
	bCfg.MinSizeItems = 3
	be, err := newBaseExporter(defaultSettings, defaultDataType, newNoopObsrepSender,
		WithBatcher(bCfg, WithRequestBatchFuncs(fakeBatchMergeFunc, fakeBatchMergeSplitFunc)))
	require.NotNil(t, be)
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	sink := newFakeRequestSink()

	// send 6 blocking requests
	wg := sync.WaitGroup{}
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			assert.NoError(t, be.send(context.Background(), &fakeRequest{items: 1, sink: sink, delay: 10 * time.Millisecond}))
			wg.Done()
		}()
	}
	wg.Wait()

	// should be sent in two batches since the batch size is 3
	assert.Equal(t, uint64(2), sink.requestsCount.Load())
	assert.Equal(t, uint64(6), sink.itemsCount.Load())

	require.NoError(t, be.Shutdown(context.Background()))
}

// Validate that the batch is cancelled once the first request in the request is cancelled
func TestBatchSender_BatchCancelled(t *testing.T) {
	bCfg := exporterbatcher.NewDefaultConfig()
	bCfg.MinSizeItems = 2
	be, err := newBaseExporter(defaultSettings, defaultDataType, newNoopObsrepSender,
		WithBatcher(bCfg, WithRequestBatchFuncs(fakeBatchMergeFunc, fakeBatchMergeSplitFunc)))
	require.NotNil(t, be)
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	sink := newFakeRequestSink()

	// send 2 blocking requests
	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go func() {
		assert.ErrorIs(t, be.send(ctx, &fakeRequest{items: 1, sink: sink, delay: 100 * time.Millisecond}), context.Canceled)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		time.Sleep(20 * time.Millisecond) // ensure this call is the second
		assert.ErrorIs(t, be.send(context.Background(), &fakeRequest{items: 1, sink: sink, delay: 100 * time.Millisecond}), context.Canceled)
		wg.Done()
	}()
	cancel() // canceling the first request should cancel the whole batch
	wg.Wait()

	// nothing should be delivered
	assert.Equal(t, uint64(0), sink.requestsCount.Load())
	assert.Equal(t, uint64(0), sink.itemsCount.Load())

	require.NoError(t, be.Shutdown(context.Background()))
}

func TestBatchSender_DrainActiveRequests(t *testing.T) {
	bCfg := exporterbatcher.NewDefaultConfig()
	bCfg.MinSizeItems = 2
	be, err := newBaseExporter(defaultSettings, defaultDataType, newNoopObsrepSender,
		WithBatcher(bCfg, WithRequestBatchFuncs(fakeBatchMergeFunc, fakeBatchMergeSplitFunc)))
	require.NotNil(t, be)
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	sink := newFakeRequestSink()

	// send 3 blocking requests with a timeout
	go func() {
		assert.NoError(t, be.send(context.Background(), &fakeRequest{items: 1, sink: sink, delay: 40 * time.Millisecond}))
	}()
	go func() {
		assert.NoError(t, be.send(context.Background(), &fakeRequest{items: 1, sink: sink, delay: 40 * time.Millisecond}))
	}()
	go func() {
		assert.NoError(t, be.send(context.Background(), &fakeRequest{items: 1, sink: sink, delay: 40 * time.Millisecond}))
	}()

	// give time for the first two requests to be batched
	time.Sleep(20 * time.Millisecond)

	// Shutdown should force the active batch to be dispatched and wait for all batches to be delivered.
	// It should take 120 milliseconds to complete.
	require.NoError(t, be.Shutdown(context.Background()))

	assert.Equal(t, uint64(2), sink.requestsCount.Load())
	assert.Equal(t, uint64(3), sink.itemsCount.Load())
}

func TestBatchSender_WithBatcherOption(t *testing.T) {
	tests := []struct {
		name        string
		opts        []Option
		expectedErr bool
	}{
		{
			name:        "no_funcs_set",
			opts:        []Option{WithBatcher(exporterbatcher.NewDefaultConfig())},
			expectedErr: true,
		},
		{
			name:        "funcs_set_internally",
			opts:        []Option{withBatchFuncs(fakeBatchMergeFunc, fakeBatchMergeSplitFunc), WithBatcher(exporterbatcher.NewDefaultConfig())},
			expectedErr: false,
		},
		{
			name: "funcs_set_twice",
			opts: []Option{
				withBatchFuncs(fakeBatchMergeFunc, fakeBatchMergeSplitFunc),
				WithBatcher(exporterbatcher.NewDefaultConfig(), WithRequestBatchFuncs(fakeBatchMergeFunc,
					fakeBatchMergeSplitFunc)),
			},
			expectedErr: true,
		},
		{
			name:        "nil_funcs",
			opts:        []Option{WithBatcher(exporterbatcher.NewDefaultConfig(), WithRequestBatchFuncs(nil, nil))},
			expectedErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			be, err := newBaseExporter(defaultSettings, defaultDataType, newNoopObsrepSender, tt.opts...)
			if tt.expectedErr {
				assert.Nil(t, be)
				assert.Error(t, err)
			} else {
				assert.NotNil(t, be)
				assert.NoError(t, err)
			}
		})
	}
}

// TestBatchSender_ShutdownDeadlock tests that the exporter does not deadlock when shutting down while a batch is being
// merged.
func TestBatchSender_ShutdownDeadlock(t *testing.T) {
	blockMerge := make(chan struct{})
	waitMerge := make(chan struct{}, 10)

	// blockedBatchMergeFunc blocks until the blockMerge channel is closed
	blockedBatchMergeFunc := func(_ context.Context, r1 Request, r2 Request) (Request, error) {
		waitMerge <- struct{}{}
		<-blockMerge
		r1.(*fakeRequest).items += r2.(*fakeRequest).items
		return r1, nil
	}

	bCfg := exporterbatcher.NewDefaultConfig()
	bCfg.FlushTimeout = 10 * time.Minute // high timeout to avoid the timeout to trigger
	be, err := newBaseExporter(defaultSettings, defaultDataType, newNoopObsrepSender,
		WithBatcher(bCfg, WithRequestBatchFuncs(blockedBatchMergeFunc, fakeBatchMergeSplitFunc)))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	sink := newFakeRequestSink()

	// Send 2 concurrent requests
	go func() { require.NoError(t, be.send(context.Background(), &fakeRequest{items: 4, sink: sink})) }()
	go func() { require.NoError(t, be.send(context.Background(), &fakeRequest{items: 4, sink: sink})) }()

	// Wait for the requests to enter the merge function
	<-waitMerge

	// Initiate the exporter shutdown, unblock the batch merge function to catch possible deadlocks,
	// then wait for the exporter to finish.
	startShutdown := make(chan struct{})
	doneShutdown := make(chan struct{})
	go func() {
		close(startShutdown)
		require.Nil(t, be.Shutdown(context.Background()))
		close(doneShutdown)
	}()
	<-startShutdown
	close(blockMerge)
	<-doneShutdown

	assert.EqualValues(t, 1, sink.requestsCount.Load())
	assert.EqualValues(t, 8, sink.itemsCount.Load())
}

func queueBatchExporter(t *testing.T, batchOption Option) *baseExporter {
	be, err := newBaseExporter(defaultSettings, defaultDataType, newNoopObsrepSender, batchOption,
		WithRequestQueue(exporterqueue.NewDefaultConfig(), exporterqueue.NewMemoryQueueFactory[Request]()))
	require.NotNil(t, be)
	require.NoError(t, err)
	return be
}
