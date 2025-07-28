// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"errors"
	"runtime"
	"sync"
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

func TestPartitionBatcher_NoSplit_MinThresholdZero_TimeoutDisabled(t *testing.T) {
	tests := []struct {
		name       string
		sizerType  request.SizerType
		sizer      request.Sizer[request.Request]
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
			sizer:      requesttest.NewBytesSizer(),
			maxWorkers: 1,
		},
		{
			name:       "bytes/three_workers",
			sizerType:  request.SizerTypeBytes,
			sizer:      requesttest.NewBytesSizer(),
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
			ba := newPartitionBatcher(cfg, tt.sizer, newWorkerPool(tt.maxWorkers), sink.Export, zap.NewNop())
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
		sizer      request.Sizer[request.Request]
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
			sizer:      requesttest.NewBytesSizer(),
			maxWorkers: 1,
		},
		{
			name:       "bytes/three_workers",
			sizerType:  request.SizerTypeBytes,
			sizer:      requesttest.NewBytesSizer(),
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
			ba := newPartitionBatcher(cfg, tt.sizer, newWorkerPool(tt.maxWorkers), sink.Export, zap.NewNop())
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
		sizer      request.Sizer[request.Request]
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
			sizer:      requesttest.NewBytesSizer(),
			maxWorkers: 1,
		},
		{
			name:       "bytes/three_workers",
			sizerType:  request.SizerTypeBytes,
			sizer:      requesttest.NewBytesSizer(),
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
			ba := newPartitionBatcher(cfg, tt.sizer, newWorkerPool(tt.maxWorkers), sink.Export, zap.NewNop())
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
		sizer      request.Sizer[request.Request]
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
			sizer:      requesttest.NewBytesSizer(),
			maxWorkers: 1,
		},
		{
			name:       "bytes/three_workers",
			sizerType:  request.SizerTypeBytes,
			sizer:      requesttest.NewBytesSizer(),
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
			ba := newPartitionBatcher(cfg, tt.sizer, newWorkerPool(tt.maxWorkers), sink.Export, zap.NewNop())
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
	ba := newPartitionBatcher(cfg, request.NewItemsSizer(), newWorkerPool(2), sink.Export, zap.NewNop())
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
	ba := newPartitionBatcher(cfg, request.NewItemsSizer(), newWorkerPool(2), sink.Export, zap.NewNop())
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
	ba := newPartitionBatcher(cfg, request.NewItemsSizer(), newWorkerPool(1), sink.Export, logger)
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
	ba := newPartitionBatcher(cfg, request.NewItemsSizer(), newWorkerPool(1), sink.Export, logger)
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
	ba := newPartitionBatcher(cfg, request.NewItemsSizer(), newWorkerPool(1), sink.Export, zap.NewNop())
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

// TestPartitionWaitForResultRefs creates a scenario where the refCount bug occurs
// This test specifically reproduces the deadlock by targeting the first code path
// where currentBatch == nil and the buggy refCount logic exists
func TestPartitionWaitForResultRefs(t *testing.T) {
	// Test the refCount bug deterministically - check if done is called prematurely
	cfg := BatchConfig{
		MinSize:      50,
		MaxSize:      40,
		FlushTimeout: time.Hour, // Disable timeout
		Sizer:        request.SizerTypeItems,
	}

	// Track export calls
	var exportCalls []int
	var exportMutex sync.Mutex

	exportFunc := func(ctx context.Context, req request.Request) error {
		exportMutex.Lock()
		size := req.ItemsCount()
		exportCalls = append(exportCalls, size)
		t.Logf("Export called with size: %d, total exports: %d", size, len(exportCalls))
		exportMutex.Unlock()
		return nil
	}

	batcher := newPartitionBatcher(cfg, request.NewItemsSizer(), newWorkerPool(1), exportFunc, zap.NewNop())
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))
	defer batcher.Shutdown(context.Background())

	// Create request that should trigger the bug:
	// The request will be split into 3 parts where last becomes currentBatch
	req := &requesttest.FakeRequest{
		Items:    100,
		MergeErr: errors.New("test merge error"),
		MergeErrResult: []request.Request{
			&requesttest.FakeRequest{Items: 40}, // Part 1: Export immediately (= MaxSize)
			&requesttest.FakeRequest{Items: 40}, // Part 2: Export immediately (= MaxSize)
			&requesttest.FakeRequest{Items: 20}, // Part 3: < MinSize, becomes currentBatch
		},
	}

	// Track done calls with precise timing
	doneCallCount := int64(0)
	var doneErrors []error
	var doneMutex sync.Mutex
	doneCalled := make(chan struct{}, 1)

	done := &deterministicDone{
		onDoneFunc: func(err error) {
			doneMutex.Lock()
			defer doneMutex.Unlock()

			// Capture export state at the moment done is called
			exportMutex.Lock()
			exportsAtDone := len(exportCalls)
			exportsAtDoneList := make([]int, len(exportCalls))
			copy(exportsAtDoneList, exportCalls)
			exportMutex.Unlock()

			count := atomic.AddInt64(&doneCallCount, 1)
			doneErrors = append(doneErrors, err)

			t.Logf("done.OnDone call #%d with err=%v, exports at this moment: %v (%d total)",
				count, err, exportsAtDoneList, exportsAtDone)

			// Check for the bug: done called with only 2 exports (currentBatch still pending)
			if exportsAtDone == 2 && len(exportsAtDoneList) == 2 {
				if exportsAtDoneList[0] == 40 && exportsAtDoneList[1] == 40 {
					t.Log("BUG DETECTED: done.OnDone called prematurely!")
					t.Log("Only immediate exports completed, currentBatch (20 items) still pending")
				}
			}

			select {
			case doneCalled <- struct{}{}:
			default:
			}
		},
	}

	t.Log("Executing Consume operation...")
	batcher.Consume(context.Background(), req, done)

	// Wait briefly to see if done gets called immediately after the immediate exports
	select {
	case <-doneCalled:
		t.Log("done.OnDone was called (checking if prematurely)")

		// Give a moment for any remaining async operations
		time.Sleep(10 * time.Millisecond)

		exportMutex.Lock()
		finalExports := make([]int, len(exportCalls))
		copy(finalExports, exportCalls)
		exportMutex.Unlock()

		t.Logf("Final export state: %v", finalExports)

		// If done was called but we still have only 2 exports, that's the bug
		if len(finalExports) == 2 {
			t.Log("CONFIRMED: done.OnDone called before currentBatch was processed")
			return
		}

	case <-time.After(50 * time.Millisecond):
		t.Log("done.OnDone not called immediately - currentBatch likely still pending")

		// Trigger currentBatch flush
		t.Log("Triggering currentBatch flush...")
		triggerReq := &requesttest.FakeRequest{Items: 35} // Enough to trigger flush
		triggerDone := &deterministicDone{
			onDoneFunc: func(err error) {
				t.Logf("trigger done.OnDone called with err=%v", err)
			},
		}
		batcher.Consume(context.Background(), triggerReq, triggerDone)

		// Wait for original done to be called
		select {
		case <-doneCalled:
			t.Log("Original done.OnDone called after trigger")
		case <-time.After(100 * time.Millisecond):
			t.Log("Original done.OnDone still not called after trigger - may indicate hanging")
		}
	}

	// Final summary
	exportMutex.Lock()
	finalExports := make([]int, len(exportCalls))
	copy(finalExports, exportCalls)
	exportMutex.Unlock()

	doneMutex.Lock()
	finalDoneCount := atomic.LoadInt64(&doneCallCount)
	doneMutex.Unlock()

	t.Logf("Test complete: exports=%v, done calls=%d", finalExports, finalDoneCount)
}

// deterministicDone for testing without timing dependencies
type deterministicDone struct {
	onDoneFunc func(error)
}

func (dd *deterministicDone) OnDone(err error) {
	if dd.onDoneFunc != nil {
		dd.onDoneFunc(err)
	}
}

// blockingTestDone simulates the blocking behavior of WaitForResult=true
// This is crucial for reproducing the deadlock
type blockingTestDone struct {
	name     string
	t        *testing.T
	resultCh chan error
	called   bool
}

func newBlockingTestDone(name string, t *testing.T) *blockingTestDone {
	return &blockingTestDone{
		name:     name,
		t:        t,
		resultCh: make(chan error, 1),
	}
}

func (btd *blockingTestDone) OnDone(err error) {
	if btd.called {
		btd.t.Errorf("OnDone called multiple times for %s", btd.name)
		return
	}
	btd.called = true
	btd.t.Logf("OnDone called for %s with err=%v", btd.name, err)

	// Send result to channel (simulating what blockingDone does)
	select {
	case btd.resultCh <- err:
	default:
		btd.t.Errorf("Failed to send result for %s - channel full", btd.name)
	}
}

type testDone struct {
	t      *testing.T
	name   string
	called bool
}

func (td *testDone) OnDone(err error) {
	if td.called {
		td.t.Errorf("OnDone called multiple times for %s", td.name)
	}
	td.called = true
	td.t.Logf("OnDone called for %s with err=%v", td.name, err)
}
