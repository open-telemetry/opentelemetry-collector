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

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/internal"
	"go.opentelemetry.io/collector/exporter/internal/requesttest"
)

func TestBatchSender_Merge(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping flaky test on Windows, see https://github.com/open-telemetry/opentelemetry-collector/issues/10758")
	}
	cfg := exporterbatcher.NewDefaultConfig()
	cfg.MinSizeItems = 10
	cfg.FlushTimeout = 100 * time.Millisecond

	tests := []struct {
		name          string
		batcherOption Option
	}{
		{
			name:          "split_disabled",
			batcherOption: WithBatcher(cfg),
		},
		{
			name: "split_high_limit",
			batcherOption: func() Option {
				c := cfg
				c.MaxSizeItems = 1000
				return WithBatcher(c)
			}(),
		},
	}

	runTest := func(testName string, enableQueueBatcher bool, tt struct {
		name          string
		batcherOption Option
	},
	) {
		t.Run(testName, func(t *testing.T) {
			resetFeatureGate := setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)
			be := queueBatchExporter(t, tt.batcherOption)

			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, be.Shutdown(context.Background()))
				resetFeatureGate()
			})

			sink := requesttest.NewSink()

			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 8, Sink: sink}))
			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 3, Sink: sink}))

			// the first two requests should be merged into one and sent by reaching the minimum items size
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 1 && sink.ItemsCount() == 11
			}, 50*time.Millisecond, 10*time.Millisecond)

			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 3, Sink: sink}))
			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 1, Sink: sink}))

			// the third and fifth requests should be sent by reaching the timeout
			// the fourth request should be ignored because of the merge error.
			time.Sleep(50 * time.Millisecond)

			// should be ignored because of the merge error.
			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{
				Items: 3, Sink: sink,
				MergeErr: errors.New("merge error"),
			}))

			assert.Equal(t, int64(1), sink.RequestsCount())
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 2 && sink.ItemsCount() == 15
			}, 100*time.Millisecond, 10*time.Millisecond)
		})
	}
	for _, tt := range tests {
		runTest(tt.name+"_enable_queue_batcher", true, tt)
		runTest(tt.name+"_disable_queue_batcher", false, tt)
	}
}

func TestBatchSender_BatchExportError(t *testing.T) {
	cfg := exporterbatcher.NewDefaultConfig()
	cfg.MinSizeItems = 10
	tests := []struct {
		name             string
		batcherOption    Option
		expectedRequests int64
		expectedItems    int64
	}{
		{
			name:          "merge_only",
			batcherOption: WithBatcher(cfg),
		},
		{
			name: "merge_without_split_triggered",
			batcherOption: func() Option {
				c := cfg
				c.MaxSizeItems = 200
				return WithBatcher(c)
			}(),
		},
		{
			name: "merge_with_split_triggered",
			batcherOption: func() Option {
				c := cfg
				c.MaxSizeItems = 20
				return WithBatcher(c)
			}(),
			expectedRequests: 1,
			expectedItems:    20,
		},
	}
	runTest := func(testName string, enableQueueBatcher bool, tt struct {
		name             string
		batcherOption    Option
		expectedRequests int64
		expectedItems    int64
	},
	) {
		t.Run(testName, func(t *testing.T) {
			resetFeatureGate := setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)
			be := queueBatchExporter(t, tt.batcherOption)

			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, be.Shutdown(context.Background()))
				resetFeatureGate()
			})

			sink := requesttest.NewSink()

			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4, Sink: sink}))
			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4, Sink: sink}))

			// the first two requests should be blocked by the batchSender.
			time.Sleep(50 * time.Millisecond)
			assert.Equal(t, int64(0), sink.RequestsCount())

			// the third request should trigger the export and cause an error.
			errReq := &requesttest.FakeRequest{Items: 20, ExportErr: errors.New("transient error"), Sink: sink}
			require.NoError(t, be.Send(context.Background(), errReq))

			// the batch should be dropped since the queue doesn't have requeuing enabled.
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == tt.expectedRequests &&
					sink.ItemsCount() == tt.expectedItems &&
					be.QueueSender.(*QueueSender).queue.Size() == 0
			}, 100*time.Millisecond, 10*time.Millisecond)
		})
	}
	for _, tt := range tests {
		runTest(tt.name+"_enable_queue_batcher", true, tt)
		runTest(tt.name+"_disable_queue_batcher", false, tt)
	}
}

func TestBatchSender_MergeOrSplit(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			resetFeatureGate := setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)

			cfg := exporterbatcher.NewDefaultConfig()
			cfg.MinSizeItems = 5
			cfg.MaxSizeItems = 10
			cfg.FlushTimeout = 100 * time.Millisecond
			be := queueBatchExporter(t, WithBatcher(cfg))

			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, be.Shutdown(context.Background()))
				resetFeatureGate()
			})

			sink := requesttest.NewSink()

			// should be sent right away by reaching the minimum items size.
			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 8, Sink: sink}))
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 1 && sink.ItemsCount() == 8
			}, 50*time.Millisecond, 10*time.Millisecond)

			// big request should be broken down into two requests, both are sent right away.
			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 17, Sink: sink}))
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 3 && sink.ItemsCount() == 25
			}, 50*time.Millisecond, 10*time.Millisecond)

			// request that cannot be split should be dropped.
			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{
				Items: 11, Sink: sink,
				MergeErr: errors.New("split error"),
			}))

			// big request should be broken down into two requests, both are sent right away.
			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 13, Sink: sink}))

			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 5 && sink.ItemsCount() == 38
			}, 50*time.Millisecond, 10*time.Millisecond)
		})
	}

	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

func TestBatchSender_Shutdown(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)()
			batchCfg := exporterbatcher.NewDefaultConfig()
			batchCfg.MinSizeItems = 10
			be := queueBatchExporter(t, WithBatcher(batchCfg))

			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

			sink := requesttest.NewSink()
			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 3, Sink: sink}))

			// To make the request reached the batchSender before shutdown.
			time.Sleep(50 * time.Millisecond)

			require.NoError(t, be.Shutdown(context.Background()))

			// shutdown should force sending the batch
			assert.Equal(t, int64(1), sink.RequestsCount())
			assert.Equal(t, int64(3), sink.ItemsCount())
		})
	}

	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

func TestBatchSender_Disabled(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)()
			cfg := exporterbatcher.NewDefaultConfig()
			cfg.Enabled = false
			cfg.MaxSizeItems = 5
			be, err := NewBaseExporter(defaultSettings, defaultSignal, newNoopObsrepSender,
				WithBatcher(cfg))
			require.NotNil(t, be)
			require.NoError(t, err)

			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, be.Shutdown(context.Background()))
			})

			sink := requesttest.NewSink()
			// should be sent right away without splitting because batching is disabled.
			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 8, Sink: sink}))
			assert.Equal(t, int64(1), sink.RequestsCount())
			assert.Equal(t, int64(8), sink.ItemsCount())
		})
	}
	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

// func TestBatchSender_InvalidMergeSplitFunc(t *testing.T) {
// 	invalidMergeSplitFunc := func(_ context.Context, _ exporterbatcher.MaxSizeConfig, _ internal.Request, req2 internal.Request) ([]internal.Request,
// 		error) {
// 		// reply with invalid 0 length slice if req2 is more than 20 items
// 		if req2.(*requesttest.FakeRequest).items > 20 {
// 			return []internal.Request{}, nil
// 		}
// 		// otherwise reply with a single request.
// 		return []internal.Request{req2}, nil
// 	}
// 	cfg := exporterbatcher.NewDefaultConfig()
// 	cfg.FlushTimeout = 50 * time.Millisecond
// 	cfg.MaxSizeItems = 20
// 	be := queueBatchExporter(t, WithBatcher(cfg), WithBatchFuncs(fakeBatchMergeFunc, invalidMergeSplitFunc))

// 	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
// 	t.Cleanup(func() {
// 		require.NoError(t, be.Shutdown(context.Background()))
// 	})

// 	sink := requesttest.NewSink()
// 	// first request should be ignored due to invalid merge/split function.
// 	require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 30, Sink: sink}))
// 	// second request should be sent after reaching the timeout.
// 	require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 15, Sink: sink}))
// 	assert.Eventually(t, func() bool {
// 		return sink.RequestsCount() == 1 && sink.ItemsCount() == 15
// 	}, 100*time.Millisecond, 10*time.Millisecond)
// }

func TestBatchSender_PostShutdown(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)()
			be, err := NewBaseExporter(defaultSettings, defaultSignal, newNoopObsrepSender,
				WithBatcher(exporterbatcher.NewDefaultConfig()))
			require.NotNil(t, be)
			require.NoError(t, err)
			assert.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			assert.NoError(t, be.Shutdown(context.Background()))

			// Closed batch sender should act as a pass-through to not block queue draining.
			sink := requesttest.NewSink()
			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 8, Sink: sink}))
			assert.Equal(t, int64(1), sink.RequestsCount())
			assert.Equal(t, int64(8), sink.ItemsCount())
		})
	}
	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

func TestBatchSender_ConcurrencyLimitReached(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping flaky test on Windows, see https://github.com/open-telemetry/opentelemetry-collector/issues/10810")
	}
	tests := []struct {
		name             string
		batcherCfg       exporterbatcher.Config
		expectedRequests int64
		expectedItems    int64
	}{
		{
			name: "merge_only",
			batcherCfg: func() exporterbatcher.Config {
				cfg := exporterbatcher.NewDefaultConfig()
				cfg.FlushTimeout = 20 * time.Millisecond
				return cfg
			}(),
			expectedRequests: 6,
			expectedItems:    51,
		},
		{
			name: "merge_without_split_triggered",
			batcherCfg: func() exporterbatcher.Config {
				cfg := exporterbatcher.NewDefaultConfig()
				cfg.FlushTimeout = 20 * time.Millisecond
				cfg.MaxSizeItems = 200
				return cfg
			}(),
			expectedRequests: 6,
			expectedItems:    51,
		},
		{
			name: "merge_with_split_triggered",
			batcherCfg: func() exporterbatcher.Config {
				cfg := exporterbatcher.NewDefaultConfig()
				cfg.FlushTimeout = 50 * time.Millisecond
				cfg.MaxSizeItems = 10
				return cfg
			}(),
			expectedRequests: 8,
			expectedItems:    51,
		},
	}

	// Why do we not expect the same behavior when usePullingBasedExporterQueueBatcher is true?
	// This test checks that when concurrency limit of batch_sender is reached, the batch_sender will flush immediately.
	// To avoid blocking, the concurrency limit is set to the number of concurrent goroutines that are in charge of
	// reading from the queue and adding to batch. With the new model, we are pulling instead of pushing so we don't
	// block the reading goroutine anymore.
	defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, false)()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qCfg := exporterqueue.NewDefaultConfig()
			qCfg.NumConsumers = 2
			be, err := NewBaseExporter(defaultSettings, defaultSignal, newNoopObsrepSender,
				WithBatcher(tt.batcherCfg),
				WithRequestQueue(qCfg, exporterqueue.NewMemoryQueueFactory[internal.Request]()))
			require.NotNil(t, be)
			require.NoError(t, err)
			assert.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				assert.NoError(t, be.Shutdown(context.Background()))
			})

			sink := requesttest.NewSink()
			// the 1st and 2nd request should be flushed in the same batched request by max concurrency limit.
			assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 2, Sink: sink}))
			assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 2, Sink: sink}))

			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 1 && sink.ItemsCount() == 4
			}, 100*time.Millisecond, 10*time.Millisecond)

			// the 3rd request should be flushed by itself due to flush interval
			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 2, Sink: sink}))
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 2 && sink.ItemsCount() == 6
			}, 100*time.Millisecond, 10*time.Millisecond)

			// the 4th and 5th request should be flushed in the same batched request by max concurrency limit.
			assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 2, Sink: sink}))
			assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 2, Sink: sink}))
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 3 && sink.ItemsCount() == 10
			}, 100*time.Millisecond, 10*time.Millisecond)

			// do it a few more times to ensure it produces the correct batch size regardless of goroutine scheduling.
			assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 5, Sink: sink}))
			assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 6, Sink: sink}))
			if tt.batcherCfg.MaxSizeItems == 10 {
				// in case of MaxSizeItems=10, wait for the leftover request to send
				assert.Eventually(t, func() bool {
					return sink.RequestsCount() == 5 && sink.ItemsCount() == 21
				}, 50*time.Millisecond, 10*time.Millisecond)
			}

			assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4, Sink: sink}))
			assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 6, Sink: sink}))
			assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 20, Sink: sink}))
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == tt.expectedRequests && sink.ItemsCount() == tt.expectedItems
			}, 100*time.Millisecond, 10*time.Millisecond)
		})
	}
}

func TestBatchSender_BatchBlocking(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)()
			bCfg := exporterbatcher.NewDefaultConfig()
			bCfg.MinSizeItems = 3
			be, err := NewBaseExporter(defaultSettings, defaultSignal, newNoopObsrepSender,
				WithBatcher(bCfg))
			require.NotNil(t, be)
			require.NoError(t, err)
			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

			sink := requesttest.NewSink()

			// send 6 blocking requests
			wg := sync.WaitGroup{}
			for i := 0; i < 6; i++ {
				wg.Add(1)
				go func() {
					assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 1, Sink: sink, Delay: 10 * time.Millisecond}))
					wg.Done()
				}()
			}
			wg.Wait()

			// should be sent in two batches since the batch size is 3
			assert.Equal(t, int64(2), sink.RequestsCount())
			assert.Equal(t, int64(6), sink.ItemsCount())

			require.NoError(t, be.Shutdown(context.Background()))
		})
	}
	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

// Validate that the batch is cancelled once the first request in the request is cancelled
func TestBatchSender_BatchCancelled(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)()
			bCfg := exporterbatcher.NewDefaultConfig()
			bCfg.MinSizeItems = 2
			be, err := NewBaseExporter(defaultSettings, defaultSignal, newNoopObsrepSender,
				WithBatcher(bCfg))
			require.NotNil(t, be)
			require.NoError(t, err)
			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

			sink := requesttest.NewSink()

			// send 2 blocking requests
			wg := sync.WaitGroup{}
			ctx, cancel := context.WithCancel(context.Background())
			wg.Add(1)
			go func() {
				assert.ErrorIs(t, be.Send(ctx, &requesttest.FakeRequest{Items: 1, Sink: sink, Delay: 100 * time.Millisecond}), context.Canceled)
				wg.Done()
			}()
			wg.Add(1)
			go func() {
				time.Sleep(20 * time.Millisecond) // ensure this call is the second
				assert.ErrorIs(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 1, Sink: sink, Delay: 100 * time.Millisecond}), context.Canceled)
				wg.Done()
			}()
			cancel() // canceling the first request should cancel the whole batch
			wg.Wait()

			// nothing should be delivered
			assert.Equal(t, int64(0), sink.RequestsCount())
			assert.Equal(t, int64(0), sink.ItemsCount())

			require.NoError(t, be.Shutdown(context.Background()))
		})
	}
	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

func TestBatchSender_DrainActiveRequests(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)()
			bCfg := exporterbatcher.NewDefaultConfig()
			bCfg.MinSizeItems = 2
			be, err := NewBaseExporter(defaultSettings, defaultSignal, newNoopObsrepSender,
				WithBatcher(bCfg))
			require.NotNil(t, be)
			require.NoError(t, err)
			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

			sink := requesttest.NewSink()

			// send 3 blocking requests with a timeout
			go func() {
				assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 1, Sink: sink, Delay: 40 * time.Millisecond}))
			}()
			go func() {
				assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 1, Sink: sink, Delay: 40 * time.Millisecond}))
			}()
			go func() {
				assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 1, Sink: sink, Delay: 40 * time.Millisecond}))
			}()

			// give time for the first two requests to be batched
			time.Sleep(20 * time.Millisecond)

			// Shutdown should force the active batch to be dispatched and wait for all batches to be delivered.
			// It should take 120 milliseconds to complete.
			require.NoError(t, be.Shutdown(context.Background()))

			assert.Equal(t, int64(2), sink.RequestsCount())
			assert.Equal(t, int64(3), sink.ItemsCount())
		})
	}
	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

func TestBatchSender_UnstartedShutdown(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)()
			be, err := NewBaseExporter(defaultSettings, defaultSignal, newNoopObsrepSender,
				WithBatcher(exporterbatcher.NewDefaultConfig()))
			require.NoError(t, err)

			err = be.Shutdown(context.Background())
			require.NoError(t, err)
		})
	}
	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

// TestBatchSender_ShutdownDeadlock tests that the exporter does not deadlock when shutting down while a batch is being
// merged.
// func TestBatchSender_ShutdownDeadlock(t *testing.T) {
// 	blockMerge := make(chan struct{})
// 	waitMerge := make(chan struct{}, 10)

// 	// blockedBatchMergeFunc blocks until the blockMerge channel is closed
// 	blockedBatchMergeFunc := func(_ context.Context, r1 internal.Request, r2 internal.Request) (internal.Request, error) {
// 		waitMerge <- struct{}{}
// 		<-blockMerge
// 		r1.(*requesttest.FakeRequest).items += r2.(*requesttest.FakeRequest).items
// 		return r1, nil
// 	}

// 	bCfg := exporterbatcher.NewDefaultConfig()
// 	bCfg.FlushTimeout = 10 * time.Minute // high timeout to avoid the timeout to trigger
// 	be, err := NewBaseExporter(defaultSettings, defaultSignal, newNoopObsrepSender,
// 		WithBatcher(bCfg))
// 	require.NoError(t, err)
// 	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

// 	sink := requesttest.NewSink()

// 	// Send 2 concurrent requests
// 	go func() { assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4, Sink: sink})) }()
// 	go func() { assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4, Sink: sink})) }()

// 	// Wait for the requests to enter the merge function
// 	<-waitMerge

// 	// Initiate the exporter shutdown, unblock the batch merge function to catch possible deadlocks,
// 	// then wait for the exporter to finish.
// 	startShutdown := make(chan struct{})
// 	doneShutdown := make(chan struct{})
// 	go func() {
// 		close(startShutdown)
// 		assert.NoError(t, be.Shutdown(context.Background()))
// 		close(doneShutdown)
// 	}()
// 	<-startShutdown
// 	close(blockMerge)
// 	<-doneShutdown

// 	assert.EqualValues(t, 1, sink.RequestsCount())
// 	assert.EqualValues(t, 8, sink.ItemsCount())
// }

func TestBatchSenderWithTimeout(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)()
			bCfg := exporterbatcher.NewDefaultConfig()
			bCfg.MinSizeItems = 10
			tCfg := NewDefaultTimeoutConfig()
			tCfg.Timeout = 50 * time.Millisecond
			be, err := NewBaseExporter(defaultSettings, defaultSignal, newNoopObsrepSender,
				WithBatcher(bCfg),
				WithTimeout(tCfg))
			require.NoError(t, err)
			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

			sink := requesttest.NewSink()

			// Send 3 concurrent requests that should be merged in one batch
			wg := sync.WaitGroup{}
			for i := 0; i < 3; i++ {
				wg.Add(1)
				go func() {
					assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4, Sink: sink}))
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
					assert.Error(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4, Sink: sink, Delay: 30 * time.Millisecond}))
					wg.Done()
				}()
			}
			wg.Wait()

			require.NoError(t, be.Shutdown(context.Background()))

			// The sink should not change
			assert.EqualValues(t, 1, sink.RequestsCount())
			assert.EqualValues(t, 12, sink.ItemsCount())
		})
	}
	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

// func TestBatchSenderTimerResetNoConflict(t *testing.T) {
// 	delayBatchMergeFunc := func(_ context.Context, r1 internal.Request, r2 internal.Request) (internal.Request, error) {
// 		time.Sleep(30 * time.Millisecond)
// 		if r1 == nil {
// 			return r2, nil
// 		}
// 		fr1 := r1.(*requesttest.FakeRequest)
// 		fr2 := r2.(*requesttest.FakeRequest)
// 		if fr2.mergeErr != nil {
// 			return nil, fr2.mergeErr
// 		}
// 		return &requesttest.FakeRequest{
// 			Items:     fr1.items + fr2.items,
// 			Sink:      fr1.sink,
// 			ExportErr: fr2.exportErr,
// 			Delay:     fr1.delay + fr2.delay,
// 		}, nil
// 	}
// 	bCfg := exporterbatcher.NewDefaultConfig()
// 	bCfg.MinSizeItems = 8
// 	bCfg.FlushTimeout = 50 * time.Millisecond
// 	be, err := NewBaseExporter(defaultSettings, defaultSignal, newNoopObsrepSender,
// 		WithBatcher(bCfg))
// 	require.NoError(t, err)
// 	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
// 	sink := requesttest.NewSink()

// 	// Send 2 concurrent requests that should be merged in one batch in the same interval as the flush timer
// 	go func() {
// 		assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4, Sink: sink}))
// 	}()
// 	time.Sleep(30 * time.Millisecond)
// 	go func() {
// 		assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4, Sink: sink}))
// 	}()

// 	// The batch should be sent either with the flush interval or by reaching the minimum items size with no conflict
// 	assert.EventuallyWithT(t, func(c *assert.CollectT) {
// 		assert.LessOrEqual(c, int64(1), sink.RequestsCount())
// 		assert.EqualValues(c, 8, sink.ItemsCount())
// 	}, 200*time.Millisecond, 10*time.Millisecond)

// 	require.NoError(t, be.Shutdown(context.Background()))
// }

func TestBatchSenderTimerFlush(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)()
			if runtime.GOOS == "windows" {
				t.Skip("skipping flaky test on Windows, see https://github.com/open-telemetry/opentelemetry-collector/issues/10802")
			}
			bCfg := exporterbatcher.NewDefaultConfig()
			bCfg.MinSizeItems = 8
			bCfg.FlushTimeout = 100 * time.Millisecond
			be, err := NewBaseExporter(defaultSettings, defaultSignal, newNoopObsrepSender,
				WithBatcher(bCfg))
			require.NoError(t, err)
			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			sink := requesttest.NewSink()
			time.Sleep(50 * time.Millisecond)

			// Send 2 concurrent requests that should be merged in one batch and sent immediately
			go func() {
				assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4, Sink: sink}))
			}()
			go func() {
				assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4, Sink: sink}))
			}()
			assert.EventuallyWithT(t, func(c *assert.CollectT) {
				assert.LessOrEqual(c, int64(1), sink.RequestsCount())
				assert.EqualValues(c, 8, sink.ItemsCount())
			}, 30*time.Millisecond, 5*time.Millisecond)

			// Send another request that should be flushed after 100ms instead of 50ms since last flush
			go func() {
				assert.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4, Sink: sink}))
			}()

			// Confirm that it is not flushed in 50ms
			time.Sleep(60 * time.Millisecond)
			assert.LessOrEqual(t, int64(1), sink.RequestsCount())
			assert.EqualValues(t, 8, sink.ItemsCount())

			// Confirm that it is flushed after 100ms (using 60+50=110 here to be safe)
			time.Sleep(50 * time.Millisecond)
			assert.LessOrEqual(t, int64(2), sink.RequestsCount())
			assert.EqualValues(t, 12, sink.ItemsCount())
			require.NoError(t, be.Shutdown(context.Background()))
		})
	}
	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

func queueBatchExporter(t *testing.T, opts ...Option) *BaseExporter {
	opts = append(opts, WithRequestQueue(exporterqueue.NewDefaultConfig(), exporterqueue.NewMemoryQueueFactory[internal.Request]()))
	be, err := NewBaseExporter(defaultSettings, defaultSignal, newNoopObsrepSender, opts...)
	require.NotNil(t, be)
	require.NoError(t, err)
	return be
}
