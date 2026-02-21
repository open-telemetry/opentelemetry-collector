// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
)

type testContextKey string

const timestampKey testContextKey = "timestamp"

func TestPartitionBatcher_NoSplit_MinThresholdZero_TimeoutDisabled(t *testing.T) {
	tests := []struct {
		name       string
		sizerType  request.SizerType
		sizer      request.Sizer
		maxWorkers int
	}{
		{
			name:       "items/one_worker",
			sizerType:  request.SizerTypeItems,
			sizer:      request.NewItemsSizer(),
			maxWorkers: 1,
		},
		{
			name:       "items/three_workers",
			sizerType:  request.SizerTypeItems,
			sizer:      request.NewItemsSizer(),
			maxWorkers: 3,
		},
		{
			name:       "bytes/one_worker",
			sizerType:  request.SizerTypeBytes,
			sizer:      request.NewBytesSizer(),
			maxWorkers: 1,
		},
		{
			name:       "bytes/three_workers",
			sizerType:  request.SizerTypeBytes,
			sizer:      request.NewBytesSizer(),
			maxWorkers: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := BatchConfig{
				FlushTimeout: 0,
				Sizer:        tt.sizerType,
				MinSize:      0,
			}

			sink := requesttest.NewSink()
			ba := newPartitionBatcher(cfg, tt.sizer, nil, newWorkerPool(tt.maxWorkers), sink.Export, zap.NewNop())
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, ba.Shutdown(context.Background()))
			})

			done := newFakeDone()
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8, Bytes: 8}, done)
			sink.SetExportErr(errors.New("transient error"))
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8, Bytes: 8}, done)
			<-time.After(10 * time.Millisecond)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 17, Bytes: 17}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 13, Bytes: 13}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 35, Bytes: 35}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 2, Bytes: 2}, done)
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 5 && (sink.ItemsCount() == 75 || sink.BytesCount() == 75)
			}, 1*time.Second, 10*time.Millisecond)
			// Check that done callback is called for the right number of times.
			assert.EqualValues(t, 1, done.errors.Load())
			assert.EqualValues(t, 5, done.success.Load())
		})
	}
}

func TestPartitionBatcher_NoSplit_TimeoutDisabled(t *testing.T) {
	tests := []struct {
		name       string
		sizerType  request.SizerType
		sizer      request.Sizer
		maxWorkers int
	}{
		{
			name:       "items/one_worker",
			sizerType:  request.SizerTypeItems,
			sizer:      request.NewItemsSizer(),
			maxWorkers: 1,
		},
		{
			name:       "items/three_workers",
			sizerType:  request.SizerTypeItems,
			sizer:      request.NewItemsSizer(),
			maxWorkers: 3,
		},
		{
			name:       "bytes/one_worker",
			sizerType:  request.SizerTypeBytes,
			sizer:      request.NewBytesSizer(),
			maxWorkers: 1,
		},
		{
			name:       "bytes/three_workers",
			sizerType:  request.SizerTypeBytes,
			sizer:      request.NewBytesSizer(),
			maxWorkers: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := BatchConfig{
				FlushTimeout: 0,
				Sizer:        tt.sizerType,
				MinSize:      10,
			}

			sink := requesttest.NewSink()
			ba := newPartitionBatcher(cfg, tt.sizer, nil, newWorkerPool(tt.maxWorkers), sink.Export, zap.NewNop())
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))

			done := newFakeDone()
			// These two requests will be dropped because of export error.
			sink.SetExportErr(errors.New("transient error"))
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8, Bytes: 8}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8, Bytes: 8}, done)
			<-time.After(10 * time.Millisecond)

			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 7, Bytes: 7}, done)
			// This requests will be dropped because of merge error.
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8, Bytes: 8, MergeErr: errors.New("transient error")}, done)

			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 13, Bytes: 13}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 35, Bytes: 35}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 2, Bytes: 2}, done)

			// Only the requests with 7+13 and 35 will be flushed.
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 2 && (sink.ItemsCount() == 55 || sink.BytesCount() == 55)
			}, 1*time.Second, 10*time.Millisecond)

			require.NoError(t, ba.Shutdown(context.Background()))

			// After shutdown the pending "current batch" is also flushed.
			assert.Equal(t, 3, sink.RequestsCount())
			assert.True(t, sink.ItemsCount() == 57 || sink.BytesCount() == 57)

			// Check that done callback is called for the right number of times.
			assert.EqualValues(t, 3, done.errors.Load())
			assert.EqualValues(t, 4, done.success.Load())
		})
	}
}

func TestPartitionBatcher_NoSplit_WithTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows, see https://github.com/open-telemetry/opentelemetry-collector/issues/11869")
	}

	tests := []struct {
		name       string
		sizerType  request.SizerType
		sizer      request.Sizer
		maxWorkers int
	}{
		{
			name:       "items/one_worker",
			sizerType:  request.SizerTypeItems,
			sizer:      request.NewItemsSizer(),
			maxWorkers: 1,
		},
		{
			name:       "items/three_workers",
			sizerType:  request.SizerTypeItems,
			sizer:      request.NewItemsSizer(),
			maxWorkers: 3,
		},
		{
			name:       "bytes/one_worker",
			sizerType:  request.SizerTypeBytes,
			sizer:      request.NewBytesSizer(),
			maxWorkers: 1,
		},
		{
			name:       "bytes/three_workers",
			sizerType:  request.SizerTypeBytes,
			sizer:      request.NewBytesSizer(),
			maxWorkers: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := BatchConfig{
				FlushTimeout: 50 * time.Millisecond,
				Sizer:        tt.sizerType,
				MinSize:      100,
			}

			sink := requesttest.NewSink()
			ba := newPartitionBatcher(cfg, tt.sizer, nil, newWorkerPool(tt.maxWorkers), sink.Export, zap.NewNop())
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, ba.Shutdown(context.Background()))
			})

			done := newFakeDone()
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8, Bytes: 8}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 17, Bytes: 17}, done)
			// This requests will be dropped because of merge error.
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8, Bytes: 8, MergeErr: errors.New("transient error")}, done)

			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 13, Bytes: 13}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 35, Bytes: 35}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 2, Bytes: 2}, done)
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 1 && (sink.ItemsCount() == 75 || sink.BytesCount() == 75)
			}, 1*time.Second, 10*time.Millisecond)

			// Check that done callback is called for the right number of times.
			assert.EqualValues(t, 1, done.errors.Load())
			assert.EqualValues(t, 5, done.success.Load())
		})
	}
}

func TestPartitionBatcher_Split_TimeoutDisabled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows, see https://github.com/open-telemetry/opentelemetry-collector/issues/11847")
	}

	tests := []struct {
		name       string
		sizerType  request.SizerType
		sizer      request.Sizer
		maxWorkers int
	}{
		{
			name:       "items/one_worker",
			sizerType:  request.SizerTypeItems,
			sizer:      request.NewItemsSizer(),
			maxWorkers: 1,
		},
		{
			name:       "items/three_workers",
			sizerType:  request.SizerTypeItems,
			sizer:      request.NewItemsSizer(),
			maxWorkers: 3,
		},
		{
			name:       "bytes/one_worker",
			sizerType:  request.SizerTypeBytes,
			sizer:      request.NewBytesSizer(),
			maxWorkers: 1,
		},
		{
			name:       "bytes/three_workers",
			sizerType:  request.SizerTypeBytes,
			sizer:      request.NewBytesSizer(),
			maxWorkers: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := BatchConfig{
				FlushTimeout: 0,
				Sizer:        tt.sizerType,
				MinSize:      100,
				MaxSize:      100,
			}

			sink := requesttest.NewSink()
			ba := newPartitionBatcher(cfg, tt.sizer, nil, newWorkerPool(tt.maxWorkers), sink.Export, zap.NewNop())
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))

			done := newFakeDone()
			// This requests will be dropped because of merge error.
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8, Bytes: 8, MergeErr: errors.New("transient error")}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8, Bytes: 8}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 17, Bytes: 17}, done)
			// This requests will be dropped because of merge error.
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8, Bytes: 8, MergeErr: errors.New("transient error")}, done)

			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 13, Bytes: 13}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 35, Bytes: 35}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 2, Bytes: 2}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 30, Bytes: 30}, done)
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 1 && (sink.ItemsCount() == 100 || sink.BytesCount() == 100)
			}, 1*time.Second, 10*time.Millisecond)

			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 900, Bytes: 900}, done)
			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 10 && (sink.ItemsCount() == 1000 || sink.BytesCount() == 1000)
			}, 1*time.Second, 10*time.Millisecond)

			// At this point the 7th not failing request is still pending.
			assert.EqualValues(t, 6, done.success.Load())

			require.NoError(t, ba.Shutdown(context.Background()))

			// After shutdown the pending "current batch" is also flushed.
			assert.Equal(t, 11, sink.RequestsCount())
			assert.True(t, sink.ItemsCount() == 1005 || sink.BytesCount() == 1005)

			// Check that done callback is called for the right number of times.
			assert.EqualValues(t, 2, done.errors.Load())
			assert.EqualValues(t, 7, done.success.Load())
		})
	}
}

func TestPartitionBatcher_Shutdown(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 100 * time.Second,
		Sizer:        request.SizerTypeItems,
		MinSize:      10,
	}

	sink := requesttest.NewSink()
	ba := newPartitionBatcher(cfg, request.NewItemsSizer(), nil, newWorkerPool(2), sink.Export, zap.NewNop())
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))

	done := newFakeDone()
	ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 1, Bytes: 1}, done)
	ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 2, Bytes: 2}, done)

	assert.Equal(t, 0, sink.RequestsCount())
	assert.Equal(t, 0, sink.ItemsCount())

	require.NoError(t, ba.Shutdown(context.Background()))

	assert.Equal(t, 1, sink.RequestsCount())
	assert.Equal(t, 3, sink.ItemsCount())

	// Check that done callback is called for the right number of times.
	assert.EqualValues(t, 0, done.errors.Load())
	assert.EqualValues(t, 2, done.success.Load())
}

func TestPartitionBatcher_MergeError(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 200 * time.Second,
		Sizer:        request.SizerTypeItems,
		MinSize:      5,
		MaxSize:      7,
	}

	sink := requesttest.NewSink()
	ba := newPartitionBatcher(cfg, request.NewItemsSizer(), nil, newWorkerPool(2), sink.Export, zap.NewNop())
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	done := newFakeDone()
	ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 9, Bytes: 9}, done)
	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 1 && sink.ItemsCount() == 7
	}, 1*time.Second, 10*time.Millisecond)

	sink.SetExportErr(errors.New("transient error"))
	ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 4, Bytes: 4}, done)
	assert.Eventually(t, func() bool {
		return done.errors.Load() == 2
	}, 1*time.Second, 10*time.Millisecond)

	// Check that done callback is called for the right number of times.
	assert.EqualValues(t, 2, done.errors.Load())
	assert.EqualValues(t, 0, done.success.Load())
}

func TestPartitionBatcher_PartialSuccessError(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 0,
		Sizer:        request.SizerTypeBytes,
		MinSize:      10,
		MaxSize:      15,
	}

	core, observed := observer.New(zap.WarnLevel)
	logger := zap.New(core)
	sink := requesttest.NewSink()
	ba := newPartitionBatcher(cfg, request.NewItemsSizer(), nil, newWorkerPool(1), sink.Export, logger)
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))

	done := newFakeDone()
	req := &requesttest.FakeRequest{
		Items:          100,
		Bytes:          100,
		MergeErr:       errors.New("split error"),
		MergeErrResult: []request.Request{&requesttest.FakeRequest{Items: 10, Bytes: 15}},
	}
	ba.Consume(context.Background(), req, done)

	assert.Eventually(t, func() bool {
		logs := observed.All()
		if len(logs) == 0 {
			return false
		}
		log := logs[0]
		return log.Level == zap.WarnLevel &&
			log.Message == "Failed to split request."
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, ba.Shutdown(context.Background()))

	// Verify that done callback was called with the returned batch and error for the split.
	assert.Equal(t, int64(1), done.errors.Load())
	assert.Equal(t, 1, sink.RequestsCount())
	assert.Equal(t, 10, sink.ItemsCount())
	assert.Equal(t, 15, sink.BytesCount())
}

func TestSPartitionBatcher_PartialSuccessError_AfterOkRequest(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 0,
		Sizer:        request.SizerTypeBytes,
		MinSize:      10,
		MaxSize:      15,
	}

	core, observed := observer.New(zap.WarnLevel)
	logger := zap.New(core)
	sink := requesttest.NewSink()
	ba := newPartitionBatcher(cfg, request.NewItemsSizer(), nil, newWorkerPool(1), sink.Export, logger)
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))

	done := newFakeDone()
	ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 5, Bytes: 5}, done)
	req := &requesttest.FakeRequest{
		Items:          100,
		Bytes:          100,
		MergeErr:       errors.New("split error"),
		MergeErrResult: []request.Request{&requesttest.FakeRequest{Items: 10, Bytes: 15}},
	}
	ba.Consume(context.Background(), req, done)

	assert.Eventually(t, func() bool {
		logs := observed.All()
		if len(logs) == 0 {
			return false
		}
		log := logs[0]
		return log.Level == zap.WarnLevel &&
			log.Message == "Failed to split request."
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, ba.Shutdown(context.Background()))

	// Verify that done callback was called with the success for the returned batch and error for the split.
	assert.Equal(t, int64(1), done.errors.Load())
	assert.Equal(t, int64(1), done.success.Load())
	assert.Equal(t, 1, sink.RequestsCount())
	assert.Equal(t, 10, sink.ItemsCount())
	assert.Equal(t, 15, sink.BytesCount())
}

type fakeDone struct {
	errors  *atomic.Int64
	success *atomic.Int64
}

func newFakeDone() fakeDone {
	return fakeDone{
		errors:  &atomic.Int64{},
		success: &atomic.Int64{},
	}
}

func (fd fakeDone) OnDone(err error) {
	if err != nil {
		fd.errors.Add(1)
	} else {
		fd.success.Add(1)
	}
}

func TestShardBatcher_EmptyRequestList(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 0,
		MinSize:      0,
	}

	sink := requesttest.NewSink()
	ba := newPartitionBatcher(cfg, request.NewItemsSizer(), nil, newWorkerPool(1), sink.Export, zap.NewNop())
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	done := newFakeDone()
	req := &requesttest.FakeRequest{
		Items:    1,
		MergeErr: errors.New("force empty list"),
	}
	ba.Consume(context.Background(), req, done)

	assert.Eventually(t, func() bool {
		return done.errors.Load() == 1
	}, time.Second, 10*time.Millisecond)
	assert.Equal(t, int64(0), done.success.Load())
	assert.Equal(t, 0, sink.RequestsCount())
}

// TestPartitionBatcher_ShutdownRespectsContextTimeout verifies that Shutdown respects
// the context deadline and doesn't hang indefinitely when workers are blocked.
func TestPartitionBatcher_ShutdownRespectsContextTimeout(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 0,
		Sizer:        request.SizerTypeItems,
		MinSize:      0, // Flush immediately
	}

	// Create a worker pool with 1 worker
	wp := newWorkerPool(1)

	// Create a blocking export function that holds the worker
	blockCh := make(chan struct{})
	blockingExport := func(_ context.Context, _ request.Request) error {
		<-blockCh // Block until channel is closed
		return nil
	}

	core, observed := observer.New(zap.WarnLevel)
	logger := zap.New(core)
	ba := newPartitionBatcher(cfg, request.NewItemsSizer(), nil, wp, blockingExport, logger)
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))

	// Send a request that will block the only worker
	done1 := newFakeDone()
	ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 1, Bytes: 1}, done1)

	// Wait a bit for the worker to be blocked
	time.Sleep(50 * time.Millisecond)

	// Now queue another item that will need to wait for a worker during shutdown
	done2 := newFakeDone()
	go ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 1, Bytes: 1}, done2)
	time.Sleep(50 * time.Millisecond)

	// Create a context with a short timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Shutdown should return within the timeout, not hang
	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- ba.Shutdown(ctx)
	}()

	select {
	case <-shutdownDone:
		// Shutdown completed (may have timed out but didn't hang)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Shutdown hung despite context timeout - bug not fixed")
	}

	// Verify that a warning was logged about data loss
	assert.Eventually(t, func() bool {
		logs := observed.All()
		for _, log := range logs {
			if log.Level == zap.WarnLevel && log.Message == "Failed to flush batch during shutdown, data may be lost" {
				return true
			}
		}
		return false
	}, 500*time.Millisecond, 10*time.Millisecond, "Expected warning log about failed flush during shutdown")

	// Clean up: unblock the worker so test can exit cleanly
	close(blockCh)
}

// TestPartitionBatcher_WorkerPoolExecuteWithContext tests the context-aware execute function.
func TestPartitionBatcher_WorkerPoolExecuteWithContext(t *testing.T) {
	t.Run("succeeds_when_worker_available", func(t *testing.T) {
		wp := newWorkerPool(1)
		executed := make(chan struct{})

		err := wp.executeWithContext(context.Background(), func() {
			close(executed)
		})

		require.NoError(t, err)
		select {
		case <-executed:
			// Success
		case <-time.After(time.Second):
			t.Fatal("function was not executed")
		}
	})

	t.Run("returns_error_when_context_cancelled", func(t *testing.T) {
		wp := newWorkerPool(1)

		// Block the only worker
		blockCh := make(chan struct{})
		wp.execute(func() {
			<-blockCh
		})

		// Try to execute with a cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		executed := false
		err := wp.executeWithContext(ctx, func() {
			executed = true
		})

		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		assert.False(t, executed, "function should not have been executed")

		// Clean up
		close(blockCh)
	})

	t.Run("returns_error_when_context_times_out", func(t *testing.T) {
		wp := newWorkerPool(1)

		// Block the only worker
		blockCh := make(chan struct{})
		wp.execute(func() {
			<-blockCh
		})

		// Try to execute with a short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		executed := false
		err := wp.executeWithContext(ctx, func() {
			executed = true
		})

		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
		assert.False(t, executed, "function should not have been executed")

		// Clean up
		close(blockCh)
	})
}

func TestPartitionBatcher_ContextMerging(t *testing.T) {
	tests := []struct {
		name         string
		mergeCtxFunc func(ctx1, ctx2 context.Context) context.Context
	}{
		{
			name: "merge_context_with_timestamp",
			mergeCtxFunc: func(ctx1, _ context.Context) context.Context {
				return context.WithValue(ctx1, timestampKey, 1234)
			},
		},
		{
			name: "merge_context_returns_background",
			mergeCtxFunc: func(_, _ context.Context) context.Context {
				return context.Background()
			},
		},
		{
			name:         "nil_merge_context",
			mergeCtxFunc: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := BatchConfig{
				FlushTimeout: 0,
				Sizer:        request.SizerTypeItems,
				MinSize:      10,
			}
			sink := requesttest.NewSink()
			ba := newPartitionBatcher(cfg, request.NewItemsSizer(), tt.mergeCtxFunc, newWorkerPool(1), sink.Export, zap.NewNop())
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))

			done := newFakeDone()
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8, Bytes: 8}, done)
			ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 8, Bytes: 8}, done)
			<-time.After(10 * time.Millisecond)
			assert.Equal(t, 1, sink.RequestsCount())
			assert.EqualValues(t, 2, done.success.Load())
		})
	}
}
