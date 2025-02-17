// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
)

func TestAsyncMemoryQueue(t *testing.T) {
	consumed := &atomic.Int64{}
	ac := newAsyncQueue(newMemoryQueue[int64](memoryQueueSettings[int64]{sizer: sizerInt64{}, capacity: 100}),
		1, func(_ context.Context, _ int64, done Done) {
			consumed.Add(1)
			done.OnDone(nil)
		})
	require.NoError(t, ac.Start(context.Background(), componenttest.NewNopHost()))
	for j := 0; j < 10; j++ {
		require.NoError(t, ac.Offer(context.Background(), 10))
	}
	require.NoError(t, ac.Shutdown(context.Background()))
	assert.EqualValues(t, 10, consumed.Load())
}

func TestAsyncMemoryQueueBlocking(t *testing.T) {
	consumed := &atomic.Int64{}
	ac := newAsyncQueue(
		newMemoryQueue[int64](memoryQueueSettings[int64]{sizer: sizerInt64{}, capacity: 100, blocking: true}),
		4, func(_ context.Context, _ int64, done Done) {
			consumed.Add(1)
			done.OnDone(nil)
		})
	require.NoError(t, ac.Start(context.Background(), componenttest.NewNopHost()))
	wg := &sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100_000; j++ {
				assert.NoError(t, ac.Offer(context.Background(), 10))
			}
		}()
	}
	wg.Wait()
	require.NoError(t, ac.Shutdown(context.Background()))
	assert.EqualValues(t, 1_000_000, consumed.Load())
}

func TestAsyncMemoryQueueBlockingCancelled(t *testing.T) {
	stop := make(chan struct{})
	ac := newAsyncQueue(
		newMemoryQueue[int64](memoryQueueSettings[int64]{sizer: sizerInt64{}, capacity: 10, blocking: true}),
		1, func(_ context.Context, _ int64, done Done) {
			<-stop
			done.OnDone(nil)
		})
	require.NoError(t, ac.Start(context.Background(), componenttest.NewNopHost()))

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 10; j++ {
			assert.NoError(t, ac.Offer(ctx, 1))
		}
		assert.ErrorIs(t, ac.Offer(ctx, 3), context.Canceled)
	}()
	// Sleep some time so that the go-routine blocks, it doesn't have to but even if it
	// does things will work.
	<-time.After(1 * time.Second)
	cancel()
	wg.Wait()

	close(stop)
	require.NoError(t, ac.Shutdown(context.Background()))
}

func BenchmarkAsyncMemoryQueue(b *testing.B) {
	q := newMemoryQueue[int64](memoryQueueSettings[int64]{sizer: sizerInt64{}, capacity: int64(10 * b.N)})
	consumed := &atomic.Int64{}
	require.NoError(b, q.Start(context.Background(), componenttest.NewNopHost()))
	ac := newAsyncQueue(q, 1, func(_ context.Context, _ int64, done Done) {
		consumed.Add(1)
		done.OnDone(nil)
	})
	require.NoError(b, ac.Start(context.Background(), componenttest.NewNopHost()))
	b.ResetTimer()
	b.ReportAllocs()
	for j := 0; j < b.N; j++ {
		require.NoError(b, q.Offer(context.Background(), 10))
	}
	require.NoError(b, ac.Shutdown(context.Background()))
	assert.EqualValues(b, b.N, consumed.Load())
}
