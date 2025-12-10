// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

func TestMemoryQueue(t *testing.T) {
	set := newSettings(request.SizerTypeItems, 7)
	q := newMemoryQueue[intRequest](set)
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, q.Offer(context.Background(), 1))
	assert.EqualValues(t, 1, q.Size())
	assert.EqualValues(t, 7, q.Capacity())

	require.NoError(t, q.Offer(context.Background(), 3))
	assert.EqualValues(t, 4, q.Size())

	// should not be able to send to the full queue
	require.ErrorIs(t, q.Offer(context.Background(), 4), ErrQueueIsFull)
	assert.EqualValues(t, 4, q.Size())

	assert.True(t, consume(q, func(_ context.Context, el intRequest) error {
		assert.EqualValues(t, 1, el)
		return nil
	}))
	assert.EqualValues(t, 3, q.Size())

	assert.True(t, consume(q, func(_ context.Context, el intRequest) error {
		assert.EqualValues(t, 3, el)
		return nil
	}))
	assert.EqualValues(t, 0, q.Size())

	require.NoError(t, q.Shutdown(context.Background()))
	assert.False(t, consume(q, func(context.Context, intRequest) error { t.FailNow(); return nil }))
	require.NoError(t, q.Shutdown(context.Background()))
}

func TestMemoryQueueBlockingCancelled(t *testing.T) {
	set := newSettings(request.SizerTypeItems, 5)
	set.BlockOnOverflow = true
	q := newMemoryQueue[intRequest](set)
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
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
	assert.True(t, consume(q, func(_ context.Context, el intRequest) error {
		assert.EqualValues(t, 3, el)
		return nil
	}))
	require.NoError(t, q.Shutdown(context.Background()))
}

func TestMemoryQueueDrainWhenShutdown(t *testing.T) {
	set := newSettings(request.SizerTypeItems, 7)
	q := newMemoryQueue[intRequest](set)
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, q.Offer(context.Background(), 1))
	require.NoError(t, q.Offer(context.Background(), 3))

	assert.True(t, consume(q, func(_ context.Context, el intRequest) error {
		assert.EqualValues(t, 1, el)
		return nil
	}))
	assert.EqualValues(t, 3, q.Size())
	require.NoError(t, q.Shutdown(context.Background()))
	assert.EqualValues(t, 3, q.Size())
	assert.True(t, consume(q, func(_ context.Context, el intRequest) error {
		assert.EqualValues(t, 3, el)
		return nil
	}))
	assert.EqualValues(t, 0, q.Size())
	assert.False(t, consume(q, func(context.Context, intRequest) error { t.FailNow(); return nil }))
	require.NoError(t, q.Shutdown(context.Background()))
}

func TestMemoryQueueOfferInvalidSize(t *testing.T) {
	set := newSettings(request.SizerTypeItems, 1)
	q := newMemoryQueue[intRequest](set)
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
	require.ErrorIs(t, q.Offer(context.Background(), -1), errInvalidSize)
	require.NoError(t, q.Shutdown(context.Background()))
}

func TestMemoryQueueRejectOverCapacityElements(t *testing.T) {
	set := newSettings(request.SizerTypeItems, 1)
	set.BlockOnOverflow = true
	q := newMemoryQueue[intRequest](set)
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
	require.ErrorIs(t, q.Offer(context.Background(), 8), errSizeTooLarge)
	require.NoError(t, q.Shutdown(context.Background()))
}

func TestMemoryQueueOfferZeroSize(t *testing.T) {
	set := newSettings(request.SizerTypeItems, 1)
	q := newMemoryQueue[intRequest](set)
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, q.Offer(context.Background(), 0))
	require.NoError(t, q.Shutdown(context.Background()))
	// Because the size 0 is ignored, nothing to drain.
	assert.False(t, consume(q, func(context.Context, intRequest) error { t.FailNow(); return nil }))
}

func TestMemoryQueueOverflow(t *testing.T) {
	set := newSettings(request.SizerTypeItems, 1)
	q := newMemoryQueue[intRequest](set)
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, q.Offer(context.Background(), 1))
	require.ErrorIs(t, q.Offer(context.Background(), 1), ErrQueueIsFull)
	require.NoError(t, q.Shutdown(context.Background()))
}

func TestMemoryQueueWaitForResultPassErrorBack(t *testing.T) {
	wg := sync.WaitGroup{}
	myErr := errors.New("test error")
	set := newSettings(request.SizerTypeItems, 100)
	set.WaitForResult = true
	q := newMemoryQueue[intRequest](set)
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, req, done, ok := q.Read(context.Background())
		assert.True(t, ok)
		assert.EqualValues(t, 1, req)
		done.OnDone(myErr)
	}()
	require.ErrorIs(t, q.Offer(context.Background(), intRequest(1)), myErr)
	require.NoError(t, q.Shutdown(context.Background()))
	wg.Wait()
}

func TestMemoryQueueWaitForResultCancelIncomingRequest(t *testing.T) {
	wg := sync.WaitGroup{}
	stop := make(chan struct{})
	set := newSettings(request.SizerTypeItems, 100)
	set.WaitForResult = true
	q := newMemoryQueue[intRequest](set)
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))

	// Consume async new data.
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _, done, ok := q.Read(context.Background())
		assert.True(t, ok)
		<-stop
		done.OnDone(nil)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-time.After(time.Second)
		cancel()
	}()
	require.ErrorIs(t, q.Offer(ctx, intRequest(1)), context.Canceled)
	close(stop)
	require.NoError(t, q.Shutdown(context.Background()))
	wg.Wait()
}

func TestMemoryQueueWaitForResultSizeAndCapacity(t *testing.T) {
	wg := sync.WaitGroup{}
	stop := make(chan struct{})
	set := newSettings(request.SizerTypeItems, 100)
	set.WaitForResult = true
	q := newMemoryQueue[intRequest](set)
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))

	// Consume async new data.
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _, done, ok := q.Read(context.Background())
		assert.True(t, ok)
		<-stop
		done.OnDone(nil)
	}()

	assert.EqualValues(t, 0, q.Size())
	assert.EqualValues(t, 100, q.Capacity())
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.NoError(t, q.Offer(context.Background(), intRequest(1)))
	}()
	assert.Eventually(t, func() bool { return q.Size() == 1 }, 1*time.Second, 10*time.Millisecond)
	assert.EqualValues(t, 100, q.Capacity())
	close(stop)
	require.NoError(t, q.Shutdown(context.Background()))
	wg.Wait()
}

func BenchmarkMemoryQueueWaitForResult(b *testing.B) {
	wg := sync.WaitGroup{}
	consumed := &atomic.Int64{}
	set := newSettings(request.SizerTypeItems, 100)
	set.WaitForResult = true
	q := newMemoryQueue[intRequest](set)
	require.NoError(b, q.Start(context.Background(), componenttest.NewNopHost()))

	// Consume async new data.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			_, req, done, ok := q.Read(context.Background())
			if !ok {
				return
			}
			consumed.Add(int64(req))
			done.OnDone(nil)
		}
	}()

	b.ReportAllocs()
	for b.Loop() {
		for range 100 {
			require.NoError(b, q.Offer(context.Background(), intRequest(1)))
		}
	}
	require.NoError(b, q.Shutdown(context.Background()))
	assert.Equal(b, int64(b.N)*100, consumed.Load())
}

func consume[T any](q readableQueue[T], consumeFunc func(context.Context, T) error) bool {
	ctx, req, done, ok := q.Read(context.Background())
	if !ok {
		return false
	}
	done.OnDone(consumeFunc(ctx, req))
	return true
}
