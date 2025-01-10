// Copyright The OpenTelemetry Authors
// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
// SPDX-License-Identifier: Apache-2.0

package exporterqueue

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
)

// In this test we run a queue with capacity 1 and a single consumer.
// We want to test the overflow behavior, so we block the consumer
// by holding a startLock before submitting items to the queue.
func TestBoundedQueue(t *testing.T) {
	q := newBoundedMemoryQueue[string](memoryQueueSettings[string]{sizer: &requestSizer[string]{}, capacity: 1})

	require.NoError(t, q.Offer(context.Background(), "a"))

	numConsumed := 0
	assert.True(t, consume(q, func(_ context.Context, item string) error {
		assert.Equal(t, "a", item)
		numConsumed++
		return nil
	}))
	assert.Equal(t, 1, numConsumed)
	assert.Equal(t, 0, q.Size())

	// produce two more items. The first one should be accepted, but not consumed.
	require.NoError(t, q.Offer(context.Background(), "b"))
	assert.Equal(t, 1, q.Size())

	// the second should be rejected since the queue is full
	require.ErrorIs(t, q.Offer(context.Background(), "c"), ErrQueueIsFull)
	assert.Equal(t, 1, q.Size())

	assert.True(t, consume(q, func(_ context.Context, item string) error {
		assert.Equal(t, "b", item)
		numConsumed++
		return nil
	}))
	assert.Equal(t, 2, numConsumed)

	for _, toAddItem := range []string{"d", "e", "f"} {
		require.NoError(t, q.Offer(context.Background(), toAddItem))
		assert.True(t, consume(q, func(_ context.Context, item string) error {
			assert.Equal(t, toAddItem, item)
			numConsumed++
			return nil
		}))
	}
	assert.Equal(t, 5, numConsumed)
	require.NoError(t, q.Shutdown(context.Background()))
	assert.False(t, consume(q, func(_ context.Context, item string) error {
		panic(item)
	}))
}

// In this test we run a queue with many items and a slow consumer.
// When the queue is stopped, the remaining items should be processed.
// Due to the way q.Stop() waits for all consumers to finish, the
// same lock strategy use above will not work, as calling Unlock
// only after Stop will mean the consumers are still locked while
// trying to perform the final consumptions.
func TestShutdownWhileNotEmpty(t *testing.T) {
	q := newBoundedMemoryQueue[string](memoryQueueSettings[string]{sizer: &requestSizer[string]{}, capacity: 1000})

	assert.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
	for i := 0; i < 10; i++ {
		require.NoError(t, q.Offer(context.Background(), strconv.FormatInt(int64(i), 10)))
	}
	assert.NoError(t, q.Shutdown(context.Background()))

	assert.Equal(t, 10, q.Size())
	numConsumed := 0
	for i := 0; i < 10; i++ {
		assert.True(t, consume(q, func(_ context.Context, item string) error {
			assert.Equal(t, strconv.FormatInt(int64(i), 10), item)
			numConsumed++
			return nil
		}))
	}
	assert.Equal(t, 10, numConsumed)
	assert.Equal(t, 0, q.Size())

	assert.False(t, consume(q, func(_ context.Context, item string) error {
		panic(item)
	}))
}

func Benchmark_QueueUsage_1000_requests(b *testing.B) {
	benchmarkQueueUsage(b, &requestSizer[ptrace.Traces]{}, 1000)
}

func Benchmark_QueueUsage_100000_requests(b *testing.B) {
	benchmarkQueueUsage(b, &requestSizer[ptrace.Traces]{}, 100000)
}

func Benchmark_QueueUsage_10000_items(b *testing.B) {
	// each request has 10 items: 1000 requests = 10000 items
	benchmarkQueueUsage(b, &itemsSizer{}, 1000)
}

func Benchmark_QueueUsage_1M_items(b *testing.B) {
	// each request has 10 items: 100000 requests = 1M items
	benchmarkQueueUsage(b, &itemsSizer{}, 100000)
}

func TestQueueUsage(t *testing.T) {
	t.Run("requests_based", func(t *testing.T) {
		queueUsage(t, &requestSizer[ptrace.Traces]{}, 10)
	})
	t.Run("items_based", func(t *testing.T) {
		queueUsage(t, &itemsSizer{}, 10)
	})
}

func benchmarkQueueUsage(b *testing.B, sizer sizer[ptrace.Traces], requestsCount int) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		queueUsage(b, sizer, requestsCount)
	}
}

func queueUsage(tb testing.TB, sizer sizer[ptrace.Traces], requestsCount int) {
	q := newBoundedMemoryQueue[ptrace.Traces](memoryQueueSettings[ptrace.Traces]{sizer: sizer, capacity: int64(10 * requestsCount)})
	consumed := &atomic.Int64{}
	require.NoError(tb, q.Start(context.Background(), componenttest.NewNopHost()))
	ac := newAsyncConsumer(q, 1, func(context.Context, ptrace.Traces) error {
		consumed.Add(1)
		return nil
	})
	td := testdata.GenerateTraces(10)
	for j := 0; j < requestsCount; j++ {
		require.NoError(tb, q.Offer(context.Background(), td))
	}
	assert.NoError(tb, q.Shutdown(context.Background()))
	assert.NoError(tb, ac.Shutdown(context.Background()))
	assert.Equal(tb, int64(requestsCount), consumed.Load())
}

func TestZeroSizeNoConsumers(t *testing.T) {
	q := newBoundedMemoryQueue[string](memoryQueueSettings[string]{sizer: &requestSizer[string]{}, capacity: 0})

	err := q.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	require.ErrorIs(t, q.Offer(context.Background(), "a"), ErrQueueIsFull) // in process

	assert.NoError(t, q.Shutdown(context.Background()))
}

func consume[T any](q Queue[T], consumeFunc func(context.Context, T) error) bool {
	index, ctx, req, ok := q.Read(context.Background())
	if !ok {
		return false
	}
	consumeErr := consumeFunc(ctx, req)
	q.OnProcessingFinished(index, consumeErr)
	return true
}

type asyncConsumer struct {
	stopWG sync.WaitGroup
}

func newAsyncConsumer[T any](q Queue[T], numConsumers int, consumeFunc func(context.Context, T) error) *asyncConsumer {
	ac := &asyncConsumer{}

	ac.stopWG.Add(numConsumers)
	for i := 0; i < numConsumers; i++ {
		go func() {
			defer ac.stopWG.Done()
			for {
				index, ctx, req, ok := q.Read(context.Background())
				if !ok {
					return
				}
				consumeErr := consumeFunc(ctx, req)
				q.OnProcessingFinished(index, consumeErr)
			}
		}()
	}
	return ac
}

// Shutdown ensures that queue and all consumers are stopped.
func (qc *asyncConsumer) Shutdown(_ context.Context) error {
	qc.stopWG.Wait()
	return nil
}
