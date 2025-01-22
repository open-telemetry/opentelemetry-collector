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
	assert.Equal(t, int64(0), q.Size())

	// produce two more items. The first one should be accepted, but not consumed.
	require.NoError(t, q.Offer(context.Background(), "b"))
	assert.Equal(t, int64(1), q.Size())

	// the second should be rejected since the queue is full
	require.ErrorIs(t, q.Offer(context.Background(), "c"), ErrQueueIsFull)
	assert.Equal(t, int64(1), q.Size())

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

	assert.Equal(t, int64(10), q.Size())
	numConsumed := 0
	for i := 0; i < 10; i++ {
		assert.True(t, consume(q, func(_ context.Context, item string) error {
			assert.Equal(t, strconv.FormatInt(int64(i), 10), item)
			numConsumed++
			return nil
		}))
	}
	assert.Equal(t, 10, numConsumed)
	assert.Equal(t, int64(0), q.Size())

	assert.False(t, consume(q, func(_ context.Context, item string) error {
		panic(item)
	}))
}

func TestQueueUsage(t *testing.T) {
	tests := []struct {
		name  string
		sizer sizer[uint64]
	}{
		{
			name:  "requests_based",
			sizer: &requestSizer[uint64]{},
		},
		{
			name:  "items_based",
			sizer: &itemsSizer{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := newBoundedMemoryQueue[uint64](memoryQueueSettings[uint64]{sizer: tt.sizer, capacity: int64(100)})
			consumed := &atomic.Int64{}
			require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
			ac := newAsyncConsumer(q, 1, func(context.Context, uint64) error {
				consumed.Add(1)
				return nil
			})
			for j := 0; j < 10; j++ {
				require.NoError(t, q.Offer(context.Background(), uint64(10)))
			}
			assert.NoError(t, q.Shutdown(context.Background()))
			assert.NoError(t, ac.Shutdown(context.Background()))
			assert.Equal(t, int64(10), consumed.Load())
		})
	}
}

func TestBlockingQueueUsage(t *testing.T) {
	tests := []struct {
		name  string
		sizer sizer[uint64]
	}{
		{
			name:  "requests_based",
			sizer: &requestSizer[uint64]{},
		},
		{
			name:  "items_based",
			sizer: &itemsSizer{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := newBoundedMemoryQueue[uint64](memoryQueueSettings[uint64]{sizer: tt.sizer, capacity: int64(100), blocking: true})
			consumed := &atomic.Int64{}
			require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
			ac := newAsyncConsumer(q, 10, func(context.Context, uint64) error {
				consumed.Add(1)
				return nil
			})
			wg := &sync.WaitGroup{}
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < 100_000; j++ {
						assert.NoError(t, q.Offer(context.Background(), uint64(10)))
					}
				}()
			}
			wg.Wait()
			assert.NoError(t, q.Shutdown(context.Background()))
			assert.NoError(t, ac.Shutdown(context.Background()))
			assert.Equal(t, int64(1_000_000), consumed.Load())
		})
	}
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
	q.OnProcessingFinished(index, consumeFunc(ctx, req))
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
				q.OnProcessingFinished(index, consumeFunc(ctx, req))
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

func BenchmarkOffer(b *testing.B) {
	tests := []struct {
		name  string
		sizer sizer[uint64]
	}{
		{
			name:  "requests_based",
			sizer: &requestSizer[uint64]{},
		},
		{
			name:  "items_based",
			sizer: &itemsSizer{},
		},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			q := newBoundedMemoryQueue[uint64](memoryQueueSettings[uint64]{sizer: &requestSizer[uint64]{}, capacity: int64(10 * b.N)})
			consumed := &atomic.Int64{}
			require.NoError(b, q.Start(context.Background(), componenttest.NewNopHost()))
			ac := newAsyncConsumer(q, 1, func(context.Context, uint64) error {
				consumed.Add(1)
				return nil
			})
			b.ResetTimer()
			b.ReportAllocs()
			for j := 0; j < b.N; j++ {
				require.NoError(b, q.Offer(context.Background(), uint64(10)))
			}
			assert.NoError(b, q.Shutdown(context.Background()))
			assert.NoError(b, ac.Shutdown(context.Background()))
			assert.Equal(b, int64(b.N), consumed.Load())
		})
	}
}
