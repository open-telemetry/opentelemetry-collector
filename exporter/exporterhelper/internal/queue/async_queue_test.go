// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestAsyncMemoryQueue(t *testing.T) {
	consumed := &atomic.Int64{}

	set := newSettings(request.SizerTypeItems, 100)
	set.NumConsumers = 1
	ac, err := newAsyncQueue(newMemoryQueue[intRequest](set),
		set, func(_ context.Context, _ intRequest, done Done) {
			consumed.Add(1)
			done.OnDone(nil)
		})
	require.NoError(t, err)
	require.NoError(t, ac.Start(context.Background(), componenttest.NewNopHost()))
	for range 10 {
		require.NoError(t, ac.Offer(context.Background(), 10))
	}
	require.NoError(t, ac.Shutdown(context.Background()))
	assert.EqualValues(t, 10, consumed.Load())
}

func TestAsyncMemoryQueueBlocking(t *testing.T) {
	consumed := &atomic.Int64{}
	set := newSettings(request.SizerTypeItems, 100)
	set.BlockOnOverflow = true
	set.NumConsumers = 4
	ac, err := newAsyncQueue(newMemoryQueue[intRequest](set),
		set, func(_ context.Context, _ intRequest, done Done) {
			consumed.Add(1)
			done.OnDone(nil)
		})
	require.NoError(t, err)
	require.NoError(t, ac.Start(context.Background(), componenttest.NewNopHost()))
	wg := &sync.WaitGroup{}
	for range 10 {
		wg.Go(func() {
			for range 100_000 {
				assert.NoError(t, ac.Offer(context.Background(), 10))
			}
		})
	}
	wg.Wait()
	require.NoError(t, ac.Shutdown(context.Background()))
	assert.EqualValues(t, 1_000_000, consumed.Load())
}

func TestAsyncMemoryWaitForResultQueueBlocking(t *testing.T) {
	consumed := &atomic.Int64{}
	set := newSettings(request.SizerTypeItems, 100)
	set.BlockOnOverflow = true
	set.WaitForResult = true
	set.NumConsumers = 4
	ac, err := newAsyncQueue(newMemoryQueue[intRequest](set),
		set, func(_ context.Context, _ intRequest, done Done) {
			consumed.Add(1)
			done.OnDone(nil)
		})
	require.NoError(t, err)
	require.NoError(t, ac.Start(context.Background(), componenttest.NewNopHost()))
	wg := &sync.WaitGroup{}
	for range 10 {
		wg.Go(func() {
			for range 100_000 {
				assert.NoError(t, ac.Offer(context.Background(), 10))
			}
		})
	}
	wg.Wait()
	require.NoError(t, ac.Shutdown(context.Background()))
	assert.EqualValues(t, 1_000_000, consumed.Load())
}

func TestAsyncMemoryQueueBlockingCancelled(t *testing.T) {
	stop := make(chan struct{})
	set := newSettings(request.SizerTypeItems, 10)
	set.BlockOnOverflow = true
	set.NumConsumers = 1
	ac, err := newAsyncQueue(newMemoryQueue[intRequest](set),
		set, func(_ context.Context, _ intRequest, done Done) {
			<-stop
			done.OnDone(nil)
		})
	require.NoError(t, err)
	require.NoError(t, ac.Start(context.Background(), componenttest.NewNopHost()))

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Go(func() {
		for range 10 {
			require.NoError(t, ac.Offer(ctx, 1))
		}
		assert.ErrorIs(t, ac.Offer(ctx, 3), context.Canceled)
	})
	// Sleep some time so that the go-routine blocks, it doesn't have to but even if it
	// does things will work.
	<-time.After(1 * time.Second)
	cancel()
	wg.Wait()

	close(stop)
	require.NoError(t, ac.Shutdown(context.Background()))
}

func BenchmarkAsyncMemoryQueue(b *testing.B) {
	b.Skip("Consistently fails with 'sending queue is full'")
	consumed := &atomic.Int64{}
	set := newSettings(request.SizerTypeItems, int64(10*b.N))
	set.NumConsumers = 1
	ac, err := newAsyncQueue(newMemoryQueue[intRequest](set), set, func(_ context.Context, _ intRequest, done Done) {
		consumed.Add(1)
		done.OnDone(nil)
	})
	require.NoError(b, err)
	require.NoError(b, ac.Start(context.Background(), componenttest.NewNopHost()))

	b.ReportAllocs()
	for b.Loop() {
		require.NoError(b, ac.Offer(context.Background(), 10))
	}
	require.NoError(b, ac.Shutdown(context.Background()))
	assert.EqualValues(b, b.N, consumed.Load())
}

func TestQueueRecordsBatchSendAge(t *testing.T) {
	telemetry := componenttest.NewTelemetry()
	t.Cleanup(func() {
		require.NoError(t, telemetry.Shutdown(context.Background()))
	})

	set := newSettings(request.SizerTypeItems, 100)
	set.Telemetry = telemetry.NewTelemetrySettings()
	set.NumConsumers = 1

	consumed := make(chan struct{}, 1)
	base := newMemoryQueue[intRequest](set)
	require.NoError(t, base.Offer(context.Background(), intRequest(3)))
	time.Sleep(25 * time.Millisecond)

	q, err := newAsyncQueue(base, set, func(_ context.Context, _ intRequest, done Done) {
		consumed <- struct{}{}
		done.OnDone(nil)
	})
	require.NoError(t, err)
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, q.Shutdown(context.Background()))
	})

	select {
	case <-consumed:
	case <-time.After(2 * time.Second):
		t.Fatal("expected queued request to be consumed")
	}

	metric, err := telemetry.GetMetric("otelcol_exporter_queue_batch_send_age")
	require.NoError(t, err)
	histogram, ok := metric.Data.(metricdata.Histogram[int64])
	require.True(t, ok)
	require.Len(t, histogram.DataPoints, 1)
	require.Equal(t, uint64(1), histogram.DataPoints[0].Count)
	require.Greater(t, histogram.DataPoints[0].Sum, int64(0))
	require.True(t, histogram.DataPoints[0].Attributes.HasValue(attribute.Key(exporterKey)))
	require.True(t, histogram.DataPoints[0].Attributes.HasValue(attribute.Key(dataTypeKey)))
}
