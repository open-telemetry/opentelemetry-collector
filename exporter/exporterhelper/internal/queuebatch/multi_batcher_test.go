// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
)

func TestMultiBatcher_NoTimeout(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 0,
		Sizer:        request.SizerTypeItems,
		MinSize:      10,
	}
	sink := requesttest.NewSink()

	type partitionKey struct{}

	ba, err := newMultiBatcher(cfg,
		request.NewItemsSizer(),
		newWorkerPool(1),
		NewPartitioner(func(ctx context.Context, _ request.Request) string {
			return ctx.Value(partitionKey{}).(string)
		}),
		nil,
		sink.Export,
		zap.NewNop(),
	)

	require.NoError(t, err)
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	done := newFakeDone()
	assert.Equal(t, int64(0), ba.getActivePartitionsCount())
	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p1"), &requesttest.FakeRequest{Items: 8}, done)
	assert.Equal(t, int64(1), ba.getActivePartitionsCount())
	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p2"), &requesttest.FakeRequest{Items: 6}, done)
	assert.Equal(t, int64(2), ba.getActivePartitionsCount())

	// Neither batch should be flushed since they haven't reached min threshold.
	assert.Equal(t, 0, sink.RequestsCount())
	assert.Equal(t, 0, sink.ItemsCount())

	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p1"), &requesttest.FakeRequest{Items: 8}, done)

	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 1 && sink.ItemsCount() == 16
	}, 500*time.Millisecond, 10*time.Millisecond)

	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p2"), &requesttest.FakeRequest{Items: 6}, done)

	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 2 && sink.ItemsCount() == 28
	}, 500*time.Millisecond, 10*time.Millisecond)

	// Check that done callback is called for the right amount of times.
	assert.EqualValues(t, 0, done.errors.Load())
	assert.EqualValues(t, 4, done.success.Load())

	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
}

func TestMultiBatcher_Timeout(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 100 * time.Millisecond,
		Sizer:        request.SizerTypeItems,
		MinSize:      100,
	}
	sink := requesttest.NewSink()

	type partitionKey struct{}

	ba, err := newMultiBatcher(cfg,
		request.NewItemsSizer(),
		newWorkerPool(1),
		NewPartitioner(func(ctx context.Context, _ request.Request) string {
			return ctx.Value(partitionKey{}).(string)
		}),
		nil,
		sink.Export,
		zap.NewNop(),
	)

	require.NoError(t, err)
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	done := newFakeDone()
	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p1"), &requesttest.FakeRequest{Items: 8}, done)
	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p2"), &requesttest.FakeRequest{Items: 6}, done)

	// Neither batch should be flushed since they haven't reached min threshold.
	assert.Equal(t, 0, sink.RequestsCount())
	assert.Equal(t, 0, sink.ItemsCount())

	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p1"), &requesttest.FakeRequest{Items: 8}, done)
	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p2"), &requesttest.FakeRequest{Items: 6}, done)

	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 2 && sink.ItemsCount() == 28
	}, 1*time.Second, 10*time.Millisecond)
	// Check that done callback is called for the right amount of times.
	assert.EqualValues(t, 0, done.errors.Load())
	assert.EqualValues(t, 4, done.success.Load())

	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
}

func TestMultiBatcher_RefCountPreventsRemoval(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 50 * time.Millisecond,
		Sizer:        request.SizerTypeItems,
		MinSize:      10,
	}
	sink := requesttest.NewSink()

	type partitionKey struct{}

	ba, err := newMultiBatcher(cfg,
		request.NewItemsSizer(),
		newWorkerPool(1),
		NewPartitioner(func(ctx context.Context, _ request.Request) string {
			return ctx.Value(partitionKey{}).(string)
		}),
		nil,
		sink.Export,
		zap.NewNop(),
	)

	require.NoError(t, err)
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	// Create a partition by consuming data
	done := newFakeDone()
	ctx := context.WithValue(context.Background(), partitionKey{}, "p1")
	ba.Consume(ctx, &requesttest.FakeRequest{Items: 15}, done) // Exceeds MinSize, will flush immediately

	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 1
	}, 500*time.Millisecond, 10*time.Millisecond)

	// Partition should still exist after flush
	assert.Equal(t, int64(1), ba.getActivePartitionsCount())

	// After waiting for flush timeout, partition should still exist
	// (because idle timeout is 30s, not the flush timeout)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int64(1), ba.getActivePartitionsCount())
}

func TestPartitionBatcher_IsRemovable(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 50 * time.Millisecond,
		Sizer:        request.SizerTypeItems,
		MinSize:      10,
	}
	sink := requesttest.NewSink()

	pb := newPartitionBatcher(cfg, request.NewItemsSizer(), nil, newWorkerPool(1), sink.Export, zap.NewNop(), nil)

	// Initially removable (no data, no refs)
	assert.True(t, pb.isRemovable())

	// Increment ref count - should not be removable
	pb.incRef()
	assert.False(t, pb.isRemovable())

	// Decrement ref count - should be removable again
	pb.decRef()
	assert.True(t, pb.isRemovable())
}

func TestPartitionBatcher_IsRemovable_WithData(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 0, // Disable timeout
		Sizer:        request.SizerTypeItems,
		MinSize:      100, // High threshold so data stays in batch
	}
	sink := requesttest.NewSink()

	pb := newPartitionBatcher(cfg, request.NewItemsSizer(), nil, newWorkerPool(1), sink.Export, zap.NewNop(), nil)
	require.NoError(t, pb.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, pb.Shutdown(context.Background()))
	})

	// Initially removable
	assert.True(t, pb.isRemovable())

	// Add data that won't be flushed (below MinSize)
	done := newFakeDone()
	pb.Consume(context.Background(), &requesttest.FakeRequest{Items: 5}, done)

	// Now has data - should not be removable
	assert.False(t, pb.isRemovable())
}

func TestPartitionBatcher_RefCount(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 0,
		Sizer:        request.SizerTypeItems,
		MinSize:      10,
	}
	sink := requesttest.NewSink()

	pb := newPartitionBatcher(cfg, request.NewItemsSizer(), nil, newWorkerPool(1), sink.Export, zap.NewNop(), nil)

	// Initial ref count should be 0
	assert.Equal(t, int64(0), pb.refCount.Load())

	// Increment multiple times
	pb.incRef()
	assert.Equal(t, int64(1), pb.refCount.Load())
	pb.incRef()
	assert.Equal(t, int64(2), pb.refCount.Load())

	// Decrement
	pb.decRef()
	assert.Equal(t, int64(1), pb.refCount.Load())
	pb.decRef()
	assert.Equal(t, int64(0), pb.refCount.Load())
}

func TestPartitionBatcher_LastDataTimeUpdated(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 0,
		Sizer:        request.SizerTypeItems,
		MinSize:      0,
	}
	sink := requesttest.NewSink()

	pb := newPartitionBatcher(cfg, request.NewItemsSizer(), nil, newWorkerPool(1), sink.Export, zap.NewNop(), nil)
	require.NoError(t, pb.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, pb.Shutdown(context.Background()))
	})

	initialTime := pb.lastDataTime

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Consume data - should update lastDataTime
	done := newFakeDone()
	pb.Consume(context.Background(), &requesttest.FakeRequest{Items: 5}, done)

	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 1
	}, 500*time.Millisecond, 10*time.Millisecond)

	// lastDataTime should have been updated
	assert.True(t, pb.lastDataTime.After(initialTime))
}

func TestMultiBatcher_OnEmptyCallback(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 0,
		Sizer:        request.SizerTypeItems,
		MinSize:      10,
	}
	sink := requesttest.NewSink()

	var onEmptyCalled bool
	var onEmptyResult bool

	// Create a partition batcher with a custom onEmpty callback for testing
	pb := newPartitionBatcher(cfg, request.NewItemsSizer(), nil, newWorkerPool(1), sink.Export, zap.NewNop(), nil)

	// Set the callback after creation to capture pb in the closure
	pb.onEmpty = func() bool {
		onEmptyCalled = true
		// Simulate the check that multiBatcher does
		if pb.isRemovable() {
			onEmptyResult = true
			return true
		}
		onEmptyResult = false
		return false
	}

	// With refCount > 0, onEmpty should return false
	pb.incRef()

	// Manually call onEmpty to test
	result := pb.onEmpty()

	assert.True(t, onEmptyCalled)
	assert.False(t, result)
	assert.False(t, onEmptyResult)

	// With refCount == 0, onEmpty should return true
	onEmptyCalled = false
	pb.decRef()

	result = pb.onEmpty()

	assert.True(t, onEmptyCalled)
	assert.True(t, result)
	assert.True(t, onEmptyResult)
}

func TestMultiBatcher_ConcurrentConsumeRefCount(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 50 * time.Millisecond,
		Sizer:        request.SizerTypeItems,
		MinSize:      100, // High threshold so nothing flushes automatically
	}
	sink := requesttest.NewSink()

	type partitionKey struct{}

	ba, err := newMultiBatcher(cfg,
		request.NewItemsSizer(),
		newWorkerPool(10),
		NewPartitioner(func(ctx context.Context, _ request.Request) string {
			return ctx.Value(partitionKey{}).(string)
		}),
		nil,
		sink.Export,
		zap.NewNop(),
	)

	require.NoError(t, err)
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	// Concurrent consumes to the same partition
	done := newFakeDone()
	ctx := context.WithValue(context.Background(), partitionKey{}, "p1")

	const numGoroutines = 50
	var wg sync.WaitGroup

	for range numGoroutines {
		wg.Go(func() {
			ba.Consume(ctx, &requesttest.FakeRequest{Items: 1}, done)
		})
	}

	wg.Wait()

	// Partition should exist
	assert.Equal(t, int64(1), ba.getActivePartitionsCount())

	// All consumes should have completed successfully
	// (refCount should be back to 0 after all Consume calls complete)
	ba.lock.Lock()
	pb, _ := ba.partitions.Get("p1")
	ba.lock.Unlock()

	assert.Equal(t, int64(0), pb.refCount.Load())
}
