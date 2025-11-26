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
)

func TestAsyncMemoryQueue(t *testing.T) {
	consumed := &atomic.Int64{}

	set := newSettings(request.SizerTypeItems, 100)
	ac := newAsyncQueue(newMemoryQueue[intRequest](set),
		1, func(_ context.Context, _ intRequest, done Done) {
			consumed.Add(1)
			done.OnDone(nil)
		}, set.ReferenceCounter)
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
	ac := newAsyncQueue(newMemoryQueue[intRequest](set),
		4, func(_ context.Context, _ intRequest, done Done) {
			consumed.Add(1)
			done.OnDone(nil)
		}, set.ReferenceCounter)
	require.NoError(t, ac.Start(context.Background(), componenttest.NewNopHost()))
	wg := &sync.WaitGroup{}
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100_000 {
				assert.NoError(t, ac.Offer(context.Background(), 10))
			}
		}()
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
	ac := newAsyncQueue(newMemoryQueue[intRequest](set),
		4, func(_ context.Context, _ intRequest, done Done) {
			consumed.Add(1)
			done.OnDone(nil)
		}, set.ReferenceCounter)
	require.NoError(t, ac.Start(context.Background(), componenttest.NewNopHost()))
	wg := &sync.WaitGroup{}
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100_000 {
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
	set := newSettings(request.SizerTypeItems, 10)
	set.BlockOnOverflow = true
	ac := newAsyncQueue(newMemoryQueue[intRequest](set),
		1, func(_ context.Context, _ intRequest, done Done) {
			<-stop
			done.OnDone(nil)
		}, set.ReferenceCounter)
	require.NoError(t, ac.Start(context.Background(), componenttest.NewNopHost()))

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 10 {
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
	b.Skip("Consistently fails with 'sending queue is full'")
	consumed := &atomic.Int64{}
	set := newSettings(request.SizerTypeItems, int64(10*b.N))
	ac := newAsyncQueue(newMemoryQueue[intRequest](set), 1, func(_ context.Context, _ intRequest, done Done) {
		consumed.Add(1)
		done.OnDone(nil)
	}, set.ReferenceCounter)
	require.NoError(b, ac.Start(context.Background(), componenttest.NewNopHost()))

	b.ReportAllocs()
	for b.Loop() {
		require.NoError(b, ac.Offer(context.Background(), 10))
	}
	require.NoError(b, ac.Shutdown(context.Background()))
	assert.EqualValues(b, b.N, consumed.Load())
}
