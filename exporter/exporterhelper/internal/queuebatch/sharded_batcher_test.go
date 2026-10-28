// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
)

func TestShardedBatcherRequiresPositiveShardCount(t *testing.T) {
	sink := requesttest.NewSink()
	sb, err := newShardedBatcher(BatchConfig{}, request.NewItemsSizer(), nil, newWorkerPool(1), sink.Export, zap.NewNop(), 0)

	require.Error(t, err)
	assert.Nil(t, sb)
}

func TestNewBatcherUsesOneShardForUnpartitionedBatching(t *testing.T) {
	sink := requesttest.NewSink()
	batcher, err := NewBatcher(configoptional.Some(BatchConfig{
		FlushTimeout: 50 * time.Millisecond,
		Sizer:        request.SizerTypeItems,
		MinSize:      10,
	}), batcherSettings[request.Request]{
		next:       sink.Export,
		maxWorkers: 1,
		logger:     zap.NewNop(),
	})

	require.NoError(t, err)
	sb, ok := batcher.(*shardedBatcher)
	require.True(t, ok)
	assert.Len(t, sb.shards, 1)
}

func TestShardedBatcherOneShardFlushesOnMinSize(t *testing.T) {
	sink := requesttest.NewSink()
	sb, err := newShardedBatcher(BatchConfig{
		FlushTimeout: 0,
		Sizer:        request.SizerTypeItems,
		MinSize:      10,
	}, request.NewItemsSizer(), nil, newWorkerPool(1), sink.Export, zap.NewNop(), 1)
	require.NoError(t, err)
	require.NoError(t, sb.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, sb.Shutdown(context.Background()))
	})

	done := newFakeDone()
	sb.Consume(context.Background(), &requesttest.FakeRequest{Items: 4}, done)
	sb.Consume(context.Background(), &requesttest.FakeRequest{Items: 6}, done)

	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 1 && sink.ItemsCount() == 10 &&
			done.success.Load() == 2 && done.errors.Load() == 0
	}, time.Second, 10*time.Millisecond)
}

func TestShardedBatcherMultipleShardsFlushOnMinSize(t *testing.T) {
	sink := requesttest.NewSink()
	sb, err := newShardedBatcher(BatchConfig{
		FlushTimeout: 0,
		Sizer:        request.SizerTypeItems,
		MinSize:      10,
	}, request.NewItemsSizer(), nil, newWorkerPool(2), sink.Export, zap.NewNop(), 2)
	require.NoError(t, err)
	require.NoError(t, sb.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, sb.Shutdown(context.Background()))
	})

	done := newFakeDone()
	sb.Consume(context.Background(), &requesttest.FakeRequest{Items: 4}, done)
	sb.Consume(context.Background(), &requesttest.FakeRequest{Items: 4}, done)
	sb.Consume(context.Background(), &requesttest.FakeRequest{Items: 6}, done)
	sb.Consume(context.Background(), &requesttest.FakeRequest{Items: 6}, done)

	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 2 && sink.ItemsCount() == 20 &&
			done.success.Load() == 4 && done.errors.Load() == 0
	}, time.Second, 10*time.Millisecond)
}

func TestShardedBatcherOneShardFlushesOnTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows, see https://github.com/open-telemetry/opentelemetry-collector/issues/11869")
	}

	sink := requesttest.NewSink()
	sb, err := newShardedBatcher(BatchConfig{
		FlushTimeout: 50 * time.Millisecond,
		Sizer:        request.SizerTypeItems,
		MinSize:      10,
	}, request.NewItemsSizer(), nil, newWorkerPool(1), sink.Export, zap.NewNop(), 1)
	require.NoError(t, err)
	require.NoError(t, sb.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, sb.Shutdown(context.Background()))
	})

	done := newFakeDone()
	sb.Consume(context.Background(), &requesttest.FakeRequest{Items: 4}, done)

	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 1 && sink.ItemsCount() == 4 &&
			done.success.Load() == 1 && done.errors.Load() == 0
	}, time.Second, 10*time.Millisecond)
}

func TestShardedBatcherOneShardDrainsOnShutdown(t *testing.T) {
	sink := requesttest.NewSink()
	sb, err := newShardedBatcher(BatchConfig{
		FlushTimeout: 0,
		Sizer:        request.SizerTypeItems,
		MinSize:      10,
	}, request.NewItemsSizer(), nil, newWorkerPool(1), sink.Export, zap.NewNop(), 1)
	require.NoError(t, err)
	require.NoError(t, sb.Start(context.Background(), componenttest.NewNopHost()))

	done := newFakeDone()
	sb.Consume(context.Background(), &requesttest.FakeRequest{Items: 4}, done)
	sb.Consume(context.Background(), &requesttest.FakeRequest{Items: 3}, done)

	require.NoError(t, sb.Shutdown(context.Background()))
	assert.Equal(t, 1, sink.RequestsCount())
	assert.Equal(t, 7, sink.ItemsCount())
	assert.EqualValues(t, 2, done.success.Load())
	assert.EqualValues(t, 0, done.errors.Load())
}

func TestShardedBatcherMultipleShardsDrainOnShutdown(t *testing.T) {
	sink := requesttest.NewSink()
	sb, err := newShardedBatcher(BatchConfig{
		FlushTimeout: 0,
		Sizer:        request.SizerTypeItems,
		MinSize:      10,
	}, request.NewItemsSizer(), nil, newWorkerPool(2), sink.Export, zap.NewNop(), 2)
	require.NoError(t, err)
	require.NoError(t, sb.Start(context.Background(), componenttest.NewNopHost()))

	done := newFakeDone()
	sb.Consume(context.Background(), &requesttest.FakeRequest{Items: 4}, done)
	sb.Consume(context.Background(), &requesttest.FakeRequest{Items: 3}, done)

	require.NoError(t, sb.Shutdown(context.Background()))
	assert.Equal(t, 2, sink.RequestsCount())
	assert.Equal(t, 7, sink.ItemsCount())
	assert.EqualValues(t, 2, done.success.Load())
	assert.EqualValues(t, 0, done.errors.Load())
}

func TestShardedBatcherOnQueueFullFlushesEveryShard(t *testing.T) {
	sink := requesttest.NewSink()
	sb, err := newShardedBatcher(BatchConfig{
		FlushTimeout: 0,
		Sizer:        request.SizerTypeItems,
		MinSize:      10,
	}, request.NewItemsSizer(), nil, newWorkerPool(2), sink.Export, zap.NewNop(), 2)
	require.NoError(t, err)
	require.NoError(t, sb.Start(context.Background(), componenttest.NewNopHost()))

	done := newFakeDone()
	sb.Consume(context.Background(), &requesttest.FakeRequest{Items: 4}, done)
	sb.Consume(context.Background(), &requesttest.FakeRequest{Items: 3}, done)
	assert.Equal(t, 0, sink.RequestsCount(), "both shards hold partial batches below min_size")

	onFull := sb.onQueueFull()
	require.NotNil(t, onFull)
	onFull()
	onFull()
	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 2 && sink.ItemsCount() == 7
	}, time.Second, 10*time.Millisecond)
	assert.EqualValues(t, 2, done.success.Load())

	require.NoError(t, sb.Shutdown(context.Background()))
	assert.Equal(t, 2, sink.RequestsCount())
}

func TestShardedBatcherOnQueueFullIsNilWithOneShard(t *testing.T) {
	sink := requesttest.NewSink()
	sb, err := newShardedBatcher(BatchConfig{
		FlushTimeout: 0,
		Sizer:        request.SizerTypeItems,
		MinSize:      10,
	}, request.NewItemsSizer(), nil, newWorkerPool(1), sink.Export, zap.NewNop(), 1)
	require.NoError(t, err)
	assert.Nil(t, sb.onQueueFull(), "a single shard keeps waiting for min_size or the timer")
	require.NoError(t, sb.Start(context.Background(), componenttest.NewNopHost()))

	done := newFakeDone()
	sb.Consume(context.Background(), &requesttest.FakeRequest{Items: 4}, done)
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, sink.RequestsCount())

	require.NoError(t, sb.Shutdown(context.Background()))
	assert.Equal(t, 1, sink.RequestsCount())
	assert.Equal(t, 4, sink.ItemsCount())
}

func TestQueueOnFullHook(t *testing.T) {
	sink := requesttest.NewSink()
	cfg := BatchConfig{FlushTimeout: 0, Sizer: request.SizerTypeItems, MinSize: 10}

	t.Run("disabled batcher", func(t *testing.T) {
		assert.Nil(t, queueOnFullHook(newDisabledBatcher(sink.Export)))
	})
	t.Run("one shard", func(t *testing.T) {
		sb, err := newShardedBatcher(cfg, request.NewItemsSizer(), nil, newWorkerPool(1), sink.Export, zap.NewNop(), 1)
		require.NoError(t, err)
		assert.Nil(t, queueOnFullHook(sb))
	})
	t.Run("multiple shards", func(t *testing.T) {
		sb, err := newShardedBatcher(cfg, request.NewItemsSizer(), nil, newWorkerPool(2), sink.Export, zap.NewNop(), 2)
		require.NoError(t, err)
		require.NoError(t, sb.Start(context.Background(), componenttest.NewNopHost()))
		t.Cleanup(func() {
			require.NoError(t, sb.Shutdown(context.Background()))
		})

		done := newFakeDone()
		sb.Consume(context.Background(), &requesttest.FakeRequest{Items: 4}, done)
		sb.Consume(context.Background(), &requesttest.FakeRequest{Items: 3}, done)
		assert.Equal(t, 0, sink.RequestsCount())

		onFull := queueOnFullHook(sb)
		require.NotNil(t, onFull)
		onFull()
		assert.Eventually(t, func() bool {
			return sink.RequestsCount() == 2 && sink.ItemsCount() == 7
		}, time.Second, 10*time.Millisecond)
	})
}

func TestShardedBatcherReleasesQueueCapacityWhenQueueIsFull(t *testing.T) {
	sink := requesttest.NewSink()
	sb, err := newShardedBatcher(BatchConfig{
		FlushTimeout: 0,
		Sizer:        request.SizerTypeItems,
		MinSize:      100,
	}, request.NewItemsSizer(), nil, newWorkerPool(2), sink.Export, zap.NewNop(), 2)
	require.NoError(t, err)

	set := newFakeRequestSettings()
	q, err := queue.NewQueue(queue.Settings[request.Request]{
		OnFull:          sb.onQueueFull(),
		SizerType:       request.SizerTypeItems,
		Capacity:        4,
		NumConsumers:    2,
		BlockOnOverflow: true,
		Signal:          set.Signal,
		Encoding:        set.Encoding,
		ID:              set.ID,
		Telemetry:       set.Telemetry,
	}, sb.Consume)
	require.NoError(t, err)
	require.NoError(t, sb.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))

	for range 8 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		require.NoError(t, q.Offer(ctx, &requesttest.FakeRequest{Items: 1}))
		cancel()
	}

	require.NoError(t, q.Shutdown(context.Background()))
	require.NoError(t, sb.Shutdown(context.Background()))
	assert.Equal(t, 8, sink.ItemsCount())
	require.Zero(t, q.Size())
}
