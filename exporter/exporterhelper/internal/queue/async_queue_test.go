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
		})
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
		})
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
	ac := newAsyncQueue(newMemoryQueue[intRequest](set),
		4, func(_ context.Context, _ intRequest, done Done) {
			consumed.Add(1)
			done.OnDone(nil)
		})
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
	ac := newAsyncQueue(newMemoryQueue[intRequest](set),
		1, func(_ context.Context, _ intRequest, done Done) {
			<-stop
			done.OnDone(nil)
		})
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

// The queue must keep its reference until the request is reported as done, not until the ConsumeFunc
// returns. A batcher downstream keeps the Done of every request it merged until the batch is sent.
func TestAsyncMemoryQueueReleasesReferenceOnDone(t *testing.T) {
	set := newSettings(request.SizerTypeItems, 100)
	rc := set.ReferenceCounter.(*fakeReferenceCounter)

	dones := make(chan Done, 1)
	ac := newAsyncQueue(newMemoryQueue[intRequest](set),
		1, func(_ context.Context, _ intRequest, done Done) {
			dones <- done
		})
	require.NoError(t, ac.Start(context.Background(), componenttest.NewNopHost()))

	require.NoError(t, ac.Offer(context.Background(), 10))
	done := <-dones
	assert.EqualValues(t, 1, rc.count(), "reference released before the request was done")

	done.OnDone(nil)
	assert.EqualValues(t, 0, rc.count())
	require.NoError(t, ac.Shutdown(context.Background()))
}

// With WaitForResult the caller stays blocked until the element is done, so the queue takes no
// reference of its own, not even once the caller has given up and a consumer holds the element.
func TestAsyncMemoryWaitForResultQueueTakesNoReference(t *testing.T) {
	set := newSettings(request.SizerTypeItems, 100)
	set.WaitForResult = true
	rc := set.ReferenceCounter.(*fakeReferenceCounter)

	dones := make(chan Done, 1)
	ac := newAsyncQueue(newMemoryQueue[intRequest](set),
		1, func(_ context.Context, _ intRequest, done Done) {
			dones <- done
		})
	require.NoError(t, ac.Start(context.Background(), componenttest.NewNopHost()))

	ctx, cancel := context.WithCancel(context.Background())
	offerErr := make(chan error, 1)
	go func() { offerErr <- ac.Offer(ctx, 10) }()

	done := <-dones
	cancel()
	assert.EqualValues(t, 0, rc.count(), "the blocked caller already keeps the element alive")

	done.OnDone(nil)
	require.NoError(t, <-offerErr)
	assert.EqualValues(t, 0, rc.count())
	require.NoError(t, ac.Shutdown(context.Background()))
}

// With WaitForResult a canceled Offer must behave like no queue at all: the element is not exported.
func TestAsyncMemoryWaitForResultQueueCancelledNotConsumed(t *testing.T) {
	set := newSettings(request.SizerTypeItems, 1000)
	set.WaitForResult = true
	rc := set.ReferenceCounter.(*fakeReferenceCounter)

	consumed := make(chan intRequest, 10)
	release := make(chan struct{})
	// A single consumer, occupied by the first element, so the second one stays queued.
	ac := newAsyncQueue(newMemoryQueue[intRequest](set),
		1, func(_ context.Context, el intRequest, done Done) {
			consumed <- el
			<-release
			done.OnDone(nil)
		})
	require.NoError(t, ac.Start(context.Background(), componenttest.NewNopHost()))

	go func() { assert.NoError(t, ac.Offer(context.Background(), 1)) }()
	require.Equal(t, intRequest(1), <-consumed)

	ctx, cancel := context.WithCancel(context.Background())
	offerErr := make(chan error, 1)
	go func() { offerErr <- ac.Offer(ctx, 2) }()
	// Size counts the in-flight element 1 plus the queued element 2. Wait until 2 is queued, then
	// give up on it.
	assert.Eventually(t, func() bool { return ac.Size() == 3 }, time.Second, 10*time.Millisecond)
	cancel()
	require.ErrorIs(t, <-offerErr, context.Canceled)

	// The element is gone from the queue, so freeing the consumer must not export it.
	assert.EqualValues(t, 1, ac.Size())
	close(release)
	select {
	case el := <-consumed:
		t.Fatalf("element %d was exported after its Offer was canceled", el)
	case <-time.After(200 * time.Millisecond):
	}
	require.NoError(t, ac.Shutdown(context.Background()))

	// Neither the abandoned element nor the consumed one made the queue take a reference.
	assert.Zero(t, rc.count())
}

func BenchmarkAsyncMemoryQueue(b *testing.B) {
	b.Skip("Consistently fails with 'sending queue is full'")
	consumed := &atomic.Int64{}
	set := newSettings(request.SizerTypeItems, int64(10*b.N))
	ac := newAsyncQueue(newMemoryQueue[intRequest](set), 1, func(_ context.Context, _ intRequest, done Done) {
		consumed.Add(1)
		done.OnDone(nil)
	})
	require.NoError(b, ac.Start(context.Background(), componenttest.NewNopHost()))

	b.ReportAllocs()
	for b.Loop() {
		require.NoError(b, ac.Offer(context.Background(), 10))
	}
	require.NoError(b, ac.Shutdown(context.Background()))
	assert.EqualValues(b, b.N, consumed.Load())
}
