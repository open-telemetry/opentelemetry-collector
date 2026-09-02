// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadatatest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
)

type partitionKey struct{}

func TestMultiBatcher_NoTimeout(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 0,
		Sizer:        request.SizerTypeItems,
		MinSize:      10,
	}
	sink := requesttest.NewSink()

	ba, err := newMultiBatcher(cfg,
		request.NewItemsSizer(),
		newWorkerPool(1),
		batcherSettings[request.Request]{
			partitioner: NewPartitioner(func(ctx context.Context, _ request.Request) string {
				return ctx.Value(partitionKey{}).(string)
			}),
			next:      sink.Export,
			telemetry: componenttest.NewNopTelemetrySettings(),
			logger:    zap.NewNop(),
		},
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

	ba, err := newMultiBatcher(cfg,
		request.NewItemsSizer(),
		newWorkerPool(1),
		batcherSettings[request.Request]{
			partitioner: NewPartitioner(func(ctx context.Context, _ request.Request) string {
				return ctx.Value(partitionKey{}).(string)
			}),
			next:      sink.Export,
			telemetry: componenttest.NewNopTelemetrySettings(),
			logger:    zap.NewNop(),
		},
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

func TestMultiBatcher_PartitionRemovedAfterIdleTimeout(t *testing.T) {
	// Use a short FlushTimeout so the idle threshold (partitionIdleCycles*FlushTimeout) is reached quickly.
	cfg := BatchConfig{
		FlushTimeout: 10 * time.Millisecond,
		Sizer:        request.SizerTypeItems,
		MinSize:      100, // High min size to prevent immediate flush
	}
	sink := requesttest.NewSink()

	ba, err := newMultiBatcher(cfg,
		request.NewItemsSizer(),
		newWorkerPool(1),
		batcherSettings[request.Request]{
			partitioner: NewPartitioner(func(ctx context.Context, _ request.Request) string {
				return ctx.Value(partitionKey{}).(string)
			}),
			next:      sink.Export,
			telemetry: componenttest.NewNopTelemetrySettings(),
			logger:    zap.NewNop(),
		},
	)

	require.NoError(t, err)
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	done := newFakeDone()

	// Create a partition
	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p1"), &requesttest.FakeRequest{Items: 5}, done)
	assert.Equal(t, int64(1), ba.getActivePartitionsCount())

	// Wait for the batch to flush via timeout
	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 1
	}, 500*time.Millisecond, 10*time.Millisecond)

	// Wait for idle timeout (partitionIdleCycles * FlushTimeout = 10 * 10ms = 100ms)
	// After this, the partition should be removed from the LRU cache.
	assert.Eventually(t, func() bool {
		return ba.getActivePartitionsCount() == 0
	}, 500*time.Millisecond, 10*time.Millisecond)
}

func TestMultiBatcher_DefaultCacheSize(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 100 * time.Millisecond,
		Sizer:        request.SizerTypeItems,
		MinSize:      10,
	}
	sink := requesttest.NewSink()

	ba, err := newMultiBatcher(cfg,
		request.NewItemsSizer(),
		newWorkerPool(1),
		batcherSettings[request.Request]{
			partitioner: NewPartitioner(func(ctx context.Context, _ request.Request) string {
				return ctx.Value(partitionKey{}).(string)
			}),
			next:      sink.Export,
			telemetry: componenttest.NewNopTelemetrySettings(),
			logger:    zap.NewNop(),
		},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	assert.Equal(t, DefaultPartitionCacheSize, ba.cacheSize)
}

func TestMultiBatcher_CacheSizeEviction(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 0,
		Sizer:        request.SizerTypeItems,
		MinSize:      100,
		CacheSize:    2,
	}
	sink := requesttest.NewSink()

	// Evicting a partition flushes it through the worker pool, which in turn needs
	// a worker to send the pending batch, so more than one worker is required.
	ba, err := newMultiBatcher(cfg,
		request.NewItemsSizer(),
		newWorkerPool(2),
		batcherSettings[request.Request]{
			partitioner: NewPartitioner(func(ctx context.Context, _ request.Request) string {
				return ctx.Value(partitionKey{}).(string)
			}),
			next:      sink.Export,
			telemetry: componenttest.NewNopTelemetrySettings(),
			logger:    zap.NewNop(),
		},
	)
	require.NoError(t, err)
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	assert.Equal(t, 2, ba.cacheSize)

	done := newFakeDone()
	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p1"), &requesttest.FakeRequest{Items: 5}, done)
	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p2"), &requesttest.FakeRequest{Items: 5}, done)
	assert.Equal(t, int64(2), ba.getActivePartitionsCount())

	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p3"), &requesttest.FakeRequest{Items: 5}, done)
	assert.Equal(t, int64(2), ba.getActivePartitionsCount())
	assert.False(t, ba.partitions.Contains("p1"))
	assert.True(t, ba.partitions.Contains("p2"))
	assert.True(t, ba.partitions.Contains("p3"))
}

func TestMultiBatcher_PartitionCacheMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	exporterID := component.NewID(exportertest.NopType)
	cfg := BatchConfig{
		FlushTimeout: 0,
		Sizer:        request.SizerTypeItems,
		MinSize:      100,
		CacheSize:    5,
	}
	sink := requesttest.NewSink()

	ba, err := newMultiBatcher(cfg,
		request.NewItemsSizer(),
		newWorkerPool(1),
		batcherSettings[request.Request]{
			partitioner: NewPartitioner(func(ctx context.Context, _ request.Request) string {
				return ctx.Value(partitionKey{}).(string)
			}),
			next:      sink.Export,
			id:        exporterID,
			signal:    pipeline.SignalLogs,
			telemetry: tt.NewTelemetrySettings(),
			logger:    zap.NewNop(),
		},
	)
	require.NoError(t, err)
	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	done := newFakeDone()
	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p1"), &requesttest.FakeRequest{Items: 5}, done)
	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p2"), &requesttest.FakeRequest{Items: 5}, done)

	attrs := attribute.NewSet(
		attribute.String(exporterKey, exporterID.String()),
		attribute.String(dataTypeKey, pipeline.SignalLogs.String()),
	)
	metadatatest.AssertEqualExporterQueueBatchPartitionCacheSize(t, tt,
		[]metricdata.DataPoint[int64]{
			{Attributes: attrs, Value: 2},
		}, metricdatatest.IgnoreTimestamp())
	metadatatest.AssertEqualExporterQueueBatchPartitionCacheCapacity(t, tt,
		[]metricdata.DataPoint[int64]{
			{Attributes: attrs, Value: 5},
		}, metricdatatest.IgnoreTimestamp())
}
