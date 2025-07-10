// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"errors"
	"fmt"
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

type customPartialErrorRequest struct {
	*requesttest.FakeRequest
	failureCount int
	maxSize      int
}

func (r *customPartialErrorRequest) MergeSplit(_ context.Context, _ int, _ request.SizerType, _ request.Request) ([]request.Request, error) {
	return nil, fmt.Errorf("partial success: failed to split request: size is greater than max size. size: %d, max_size: %d. Failed: %d",
		r.ItemsCount(),
		r.maxSize,
		r.ItemsCount(),
	)
}

func (r *customPartialErrorRequest) ItemsCount() int {
	return r.FakeRequest.ItemsCount()
}

func (r *customPartialErrorRequest) OnError(err error) request.Request {
	return r.FakeRequest.OnError(err)
}

func TestShardBatcher_PartialSuccessError(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 0,
		MinSize:      0,
		MaxSize:      10,
	}

	core, observed := observer.New(zap.WarnLevel)
	logger := zap.New(core)
	sink := requesttest.NewSink()
	ba := newPartitionBatcher(cfg, request.NewItemsSizer(), newWorkerPool(1), sink.Export, logger)
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	done := newFakeDone()

	req := &customPartialErrorRequest{
		maxSize: 10,
		FakeRequest: &requesttest.FakeRequest{
			Items: 100,
			Bytes: 100,
		},
		failureCount: 42,
	}
	ba.Consume(context.Background(), req, done)

	assert.Eventually(t, func() bool {
		logs := observed.All()
		if len(logs) == 0 {
			return false
		}
		log := logs[0]
		return log.Level == zap.WarnLevel &&
			log.Message == "partial success: failed to split request: size is greater than max size. size: 100, max_size: 10. Failed: 100"
	}, time.Second, 10*time.Millisecond)

	// Verify that done callback was called with the error
	assert.Eventually(t, func() bool {
		return done.errors.Load() == 1
	}, time.Second, 10*time.Millisecond)
}

func TestShardBatcher_PartialSuccessError_WithLogs(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 0,
		MinSize:      10,
	}

	core, logs := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	sink := requesttest.NewSink()
	ba := newPartitionBatcher(cfg, request.NewItemsSizer(), newWorkerPool(1), sink.Export, logger)
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	done := newFakeDone()
	req := &customPartialErrorRequest{
		FakeRequest: &requesttest.FakeRequest{
			Items:    8,
			MergeErr: errors.New("partial success: failed to split request: size is greater than max size. size: 3, max_size: 10. Failed: 3"),
		},
		failureCount: 42,
	}

	ba.Consume(context.Background(), req, done)

	assert.Eventually(t, func() bool {
		return done.errors.Load() == 1
	}, 1*time.Second, 10*time.Millisecond)

	allLogs := logs.All()
	require.Len(t, allLogs, 1)
	assert.Contains(t, allLogs[0].Message, "partial success: failed to split request:")
}

func TestShardBatcher_PartialSuccessError_WithFailureCountAndReason(t *testing.T) {
	core, observedLogs := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	cfg := BatchConfig{
		FlushTimeout: 0,
		MinSize:      5,
		MaxSize:      10,
	}

	sink := requesttest.NewSink()
	ba := newPartitionBatcher(cfg, request.NewItemsSizer(), newWorkerPool(1), sink.Export, logger)
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	// First add a request to create a current batch
	done1 := newFakeDone()
	req1 := &requesttest.FakeRequest{
		Items: 3,
	}
	ba.Consume(context.Background(), req1, done1)

	// Now add a request that will trigger the partial success error
	done2 := newFakeDone()
	req2 := &requesttest.FakeRequest{
		Items:    3,
		MergeErr: errors.New("partial success: failed to split request: size is greater than max size. size: 3, max_size: 10. Failed: 3"),
	}
	ba.Consume(context.Background(), req2, done2)

	assert.Eventually(t, func() bool {
		return observedLogs.Len() > 0
	}, time.Second, 10*time.Millisecond)

	allLogs := observedLogs.All()
	require.Len(t, allLogs, 1)
	assert.Contains(t, allLogs[0].Message, "failed to split request")
	assert.EqualValues(t, 1, done2.errors.Load())
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
