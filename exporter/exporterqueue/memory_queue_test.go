// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sizerInt64 struct{}

func (s sizerInt64) Sizeof(el int64) int64 {
	return el
}

func TestMemoryQueue(t *testing.T) {
	q := newMemoryQueue[int64](memoryQueueSettings[int64]{sizer: sizerInt64{}, capacity: 7})
	require.NoError(t, q.Offer(context.Background(), 1))
	assert.EqualValues(t, 1, q.Size())
	assert.EqualValues(t, 7, q.Capacity())

	require.NoError(t, q.Offer(context.Background(), 3))
	assert.EqualValues(t, 4, q.Size())

	// should not be able to send to the full queue
	require.ErrorIs(t, q.Offer(context.Background(), 4), ErrQueueIsFull)
	assert.EqualValues(t, 4, q.Size())

	assert.True(t, consume(q, func(_ context.Context, el int64) error {
		assert.EqualValues(t, 1, el)
		return nil
	}))
	assert.EqualValues(t, 3, q.Size())

	assert.True(t, consume(q, func(_ context.Context, el int64) error {
		assert.EqualValues(t, 3, el)
		return nil
	}))
	assert.EqualValues(t, 0, q.Size())

	require.NoError(t, q.Shutdown(context.Background()))
	assert.False(t, consume(q, func(context.Context, int64) error { t.FailNow(); return nil }))
}

func TestMemoryQueueBlockingCancelled(t *testing.T) {
	q := newMemoryQueue[int64](memoryQueueSettings[int64]{sizer: sizerInt64{}, capacity: 5, blocking: true})
	require.NoError(t, q.Offer(context.Background(), 3))
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.ErrorIs(t, q.Offer(ctx, 3), context.Canceled)
	}()
	cancel()
	wg.Wait()
	assert.EqualValues(t, 3, q.Size())
	assert.True(t, consume(q, func(_ context.Context, el int64) error {
		assert.EqualValues(t, 3, el)
		return nil
	}))
	require.NoError(t, q.Shutdown(context.Background()))
}

func TestMemoryQueueDrainWhenShutdown(t *testing.T) {
	q := newMemoryQueue[int64](memoryQueueSettings[int64]{sizer: sizerInt64{}, capacity: 7})
	require.NoError(t, q.Offer(context.Background(), 1))
	require.NoError(t, q.Offer(context.Background(), 3))

	assert.True(t, consume(q, func(_ context.Context, el int64) error {
		assert.EqualValues(t, 1, el)
		return nil
	}))
	assert.EqualValues(t, 3, q.Size())
	require.NoError(t, q.Shutdown(context.Background()))
	assert.EqualValues(t, 3, q.Size())
	assert.True(t, consume(q, func(_ context.Context, el int64) error {
		assert.EqualValues(t, 3, el)
		return nil
	}))
	assert.EqualValues(t, 0, q.Size())
	assert.False(t, consume(q, func(context.Context, int64) error { t.FailNow(); return nil }))
}

func TestMemoryQueueOfferInvalidSize(t *testing.T) {
	q := newMemoryQueue[int64](memoryQueueSettings[int64]{sizer: sizerInt64{}, capacity: 1})
	require.ErrorIs(t, q.Offer(context.Background(), -1), errInvalidSize)
}

func TestMemoryQueueOfferZeroSize(t *testing.T) {
	q := newMemoryQueue[int64](memoryQueueSettings[int64]{sizer: sizerInt64{}, capacity: 1})
	require.NoError(t, q.Offer(context.Background(), 0))
	require.NoError(t, q.Shutdown(context.Background()))
	// Because the size 0 is ignored, nothing to drain.
	assert.False(t, consume(q, func(context.Context, int64) error { t.FailNow(); return nil }))
}

func TestMemoryQueueZeroCapacity(t *testing.T) {
	q := newMemoryQueue[int64](memoryQueueSettings[int64]{sizer: sizerInt64{}, capacity: 0})
	require.ErrorIs(t, q.Offer(context.Background(), 1), ErrQueueIsFull)
	require.NoError(t, q.Shutdown(context.Background()))
}

func consume[T any](q readableQueue[T], consumeFunc func(context.Context, T) error) bool {
	ctx, req, done, ok := q.Read(context.Background())
	if !ok {
		return false
	}
	done.OnDone(consumeFunc(ctx, req))
	return true
}
