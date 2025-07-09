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
				MinSize:      0,
			}

			sink := requesttest.NewSink()
			ba := newPartitionBatcher(cfg, tt.sizerType, tt.sizer, newWorkerPool(tt.maxWorkers), sink.Export)
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
				MinSize:      10,
			}

			sink := requesttest.NewSink()
			ba := newPartitionBatcher(cfg, tt.sizerType, tt.sizer, newWorkerPool(tt.maxWorkers), sink.Export)
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
				MinSize:      100,
			}

			sink := requesttest.NewSink()
			ba := newPartitionBatcher(cfg, tt.sizerType, tt.sizer, newWorkerPool(tt.maxWorkers), sink.Export)
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
				MinSize:      100,
				MaxSize:      100,
			}

			sink := requesttest.NewSink()
			ba := newPartitionBatcher(cfg, tt.sizerType, tt.sizer, newWorkerPool(tt.maxWorkers), sink.Export)
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
		MinSize:      10,
	}

	sink := requesttest.NewSink()
	ba := newPartitionBatcher(cfg, request.SizerTypeItems, request.NewItemsSizer(), newWorkerPool(2), sink.Export)
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
		MinSize:      5,
		MaxSize:      7,
	}

	sink := requesttest.NewSink()
	ba := newPartitionBatcher(cfg, request.SizerTypeItems, request.NewItemsSizer(), newWorkerPool(2), sink.Export)
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
