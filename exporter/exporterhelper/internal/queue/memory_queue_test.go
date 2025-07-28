// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"errors"
	"runtime"
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
	q := newMemoryQueue[int64](set)
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
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
	require.NoError(t, q.Shutdown(context.Background()))
}

func TestMemoryQueueBlockingCancelled(t *testing.T) {
	set := newSettings(request.SizerTypeItems, 5)
	set.BlockOnOverflow = true
	q := newMemoryQueue[int64](set)
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
	assert.True(t, consume(q, func(_ context.Context, el int64) error {
		assert.EqualValues(t, 3, el)
		return nil
	}))
	require.NoError(t, q.Shutdown(context.Background()))
}

func TestMemoryQueueDrainWhenShutdown(t *testing.T) {
	set := newSettings(request.SizerTypeItems, 7)
	q := newMemoryQueue[int64](set)
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
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
	require.NoError(t, q.Shutdown(context.Background()))
}

func TestMemoryQueueOfferInvalidSize(t *testing.T) {
	set := newSettings(request.SizerTypeItems, 1)
	q := newMemoryQueue[int64](set)
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
	require.ErrorIs(t, q.Offer(context.Background(), -1), errInvalidSize)
	require.NoError(t, q.Shutdown(context.Background()))
}

func TestMemoryQueueRejectOverCapacityElements(t *testing.T) {
	set := newSettings(request.SizerTypeItems, 1)
	set.BlockOnOverflow = true
	q := newMemoryQueue[int64](set)
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
	require.ErrorIs(t, q.Offer(context.Background(), 8), errSizeTooLarge)
	require.NoError(t, q.Shutdown(context.Background()))
}

func TestMemoryQueueOfferZeroSize(t *testing.T) {
	set := newSettings(request.SizerTypeItems, 1)
	q := newMemoryQueue[int64](set)
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, q.Offer(context.Background(), 0))
	require.NoError(t, q.Shutdown(context.Background()))
	// Because the size 0 is ignored, nothing to drain.
	assert.False(t, consume(q, func(context.Context, int64) error { t.FailNow(); return nil }))
}

func TestMemoryQueueOverflow(t *testing.T) {
	set := newSettings(request.SizerTypeItems, 1)
	q := newMemoryQueue[int64](set)
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
	q := newMemoryQueue[int64](set)
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, req, done, ok := q.Read(context.Background())
		assert.True(t, ok)
		assert.EqualValues(t, 1, req)
		done.OnDone(myErr)
	}()
	require.ErrorIs(t, q.Offer(context.Background(), int64(1)), myErr)
	require.NoError(t, q.Shutdown(context.Background()))
	wg.Wait()
}

func TestMemoryQueueWaitForResultCancelIncomingRequest(t *testing.T) {
	wg := sync.WaitGroup{}
	stop := make(chan struct{})
	set := newSettings(request.SizerTypeItems, 100)
	set.WaitForResult = true
	q := newMemoryQueue[int64](set)
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
	require.ErrorIs(t, q.Offer(ctx, int64(1)), context.Canceled)
	close(stop)
	require.NoError(t, q.Shutdown(context.Background()))
	wg.Wait()
}

func TestMemoryQueueWaitForResultSizeAndCapacity(t *testing.T) {
	wg := sync.WaitGroup{}
	stop := make(chan struct{})
	set := newSettings(request.SizerTypeItems, 100)
	set.WaitForResult = true
	q := newMemoryQueue[int64](set)
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
		assert.NoError(t, q.Offer(context.Background(), int64(1)))
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
	q := newMemoryQueue[int64](set)
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
			consumed.Add(req)
			done.OnDone(nil)
		}
	}()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		require.NoError(b, q.Offer(context.Background(), int64(1)))
	}
	require.NoError(b, q.Shutdown(context.Background()))
	assert.Equal(b, int64(b.N), consumed.Load())
}

func consume[T any](q readableQueue[T], consumeFunc func(context.Context, T) error) bool {
	ctx, req, done, ok := q.Read(context.Background())
	if !ok {
		return false
	}
	done.OnDone(consumeFunc(ctx, req))
	return true
}

func TestMemoryQueueWaitForResultDeadlock(t *testing.T) {
	// This test reproduces the deadlock where:
	// 1. Consumer calls Read() and gets stuck in mq.hasMoreElements.Wait()
	// 2. Producer calls Offer() and gets stuck waiting for done.ch
	// 3. The signal from hasMoreElements.Signal() is lost due to race condition

	// Try multiple iterations to increase chance of hitting the race
	for attempt := 0; attempt < 100; attempt++ {
		set := newSettings(request.SizerTypeItems, 100)
		set.WaitForResult = true
		q := newMemoryQueue[int64](set)
		require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))

		// Use channels to coordinate the race condition
		consumerStarted := make(chan struct{})
		consumerInRead := make(chan struct{})
		producerDone := make(chan error, 1)
		consumerDone := make(chan struct{})

		// Start consumer that will get stuck in Wait()
		go func() {
			defer close(consumerDone)
			close(consumerStarted)

			// Add some delay to increase chance of race
			time.Sleep(time.Duration(attempt) * time.Microsecond)

			close(consumerInRead)
			// This should get stuck in mq.hasMoreElements.Wait() due to lost signal
			_, req, done, ok := q.Read(context.Background())
			if !ok {
				return
			}
			assert.EqualValues(t, 1, req)
			done.OnDone(nil) // Complete successfully
		}()

		// Start producer immediately after consumer enters Read()
		go func() {
			<-consumerStarted
			<-consumerInRead

			// Small delay to let consumer potentially reach Wait()
			time.Sleep(time.Duration(attempt) * time.Microsecond)

			// This should get stuck waiting for done.ch if consumer is stuck in Wait()
			err := q.Offer(context.Background(), int64(1))
			producerDone <- err
		}()

		// Test should complete within reasonable time
		timeout := time.After(100 * time.Millisecond)

		select {
		case err := <-producerDone:
			require.NoError(t, err)
			// If producer completes, consumer should also complete
			select {
			case <-consumerDone:
				// Success - no deadlock
			case <-time.After(50 * time.Millisecond):
				t.Fatalf("Consumer stuck after producer completed on attempt %d - partial deadlock", attempt)
			}
		case <-timeout:
			t.Fatalf("Deadlock detected on attempt %d: producer and consumer both stuck", attempt)
		}

		require.NoError(t, q.Shutdown(context.Background()))
	}
}

func TestMemoryQueueAsyncDeadlock(t *testing.T) {
	// Test the deadlock in async queue setup (simulating batch processor)
	set := newSettings(request.SizerTypeItems, 1000) // Larger capacity
	set.WaitForResult = true

	memQueue := newMemoryQueue[int64](set)
	require.NoError(t, memQueue.Start(context.Background(), componenttest.NewNopHost()))

	// Create async queue with 1 consumer (like batch processor)
	consumeFunc := func(ctx context.Context, req int64, done Done) {
		// Simulate some processing time to create backpressure
		time.Sleep(10 * time.Millisecond)
		done.OnDone(nil)
	}

	asyncQueue := newAsyncQueue(memQueue, 1, consumeFunc)
	require.NoError(t, asyncQueue.Start(context.Background(), componenttest.NewNopHost()))

	// Send multiple concurrent requests (like the batch processor test)
	numRequests := 50 // Reduce to avoid filling queue
	errChan := make(chan error, numRequests)

	// Send all requests concurrently to create contention
	for i := 0; i < numRequests; i++ {
		go func(val int64) {
			err := asyncQueue.Offer(context.Background(), val)
			errChan <- err
		}(int64(i))
	}

	// Wait for all requests to complete with timeout
	timeout := time.After(10 * time.Second)
	completed := 0

	for completed < numRequests {
		select {
		case err := <-errChan:
			require.NoError(t, err)
			completed++
		case <-timeout:
			t.Fatalf("Deadlock detected: only %d of %d requests completed", completed, numRequests)
		}
	}

	require.NoError(t, asyncQueue.Shutdown(context.Background()))
}

func TestMemoryQueueBlockOnOverflowDeadlock(t *testing.T) {
	// Stress test to reproduce condition variable race deadlock
	// Mimics exact batch processor configuration: 1000 queue size, single consumer, items-based sizer
	set := newSettings(request.SizerTypeItems, 1000) // Match batch processor default
	set.WaitForResult = true
	set.BlockOnOverflow = true // This is KEY - producers will block when queue is full
	set.NumConsumers = 1

	memQueue := newMemoryQueue[int64](set)

	// Create async queue with consumer (like queuebatch does)
	asyncQueue := newAsyncQueue(memQueue, set.NumConsumers, func(ctx context.Context, req int64, done Done) {
		// SLOW processing to let queue fill up and trigger BlockOnOverflow
		time.Sleep(50 * time.Millisecond) // Slower than production rate to create backpressure
		done.OnDone(nil)
	})

	require.NoError(t, asyncQueue.Start(context.Background(), componenttest.NewNopHost()))
	defer func() { require.NoError(t, asyncQueue.Shutdown(context.Background())) }()

	// Rapid fire producers to fill queue and trigger BlockOnOverflow blocking
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	producerCount := 10     // More concurrent producers to increase race likelihood
	itemsPerProducer := 200 // Each item size=1, so 10*200=2000 items total vs queue capacity 1000

	// Launch producers that will block on BlockOnOverflow
	for producerID := 0; producerID < producerCount; producerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < itemsPerProducer; i++ {
				select {
				case <-ctx.Done():
					return // Timeout - likely deadlock
				default:
					// This Offer() call should BLOCK when queue fills up (due to BlockOnOverflow=true)
					// and should wait for done.ch (due to WaitForResult=true)
					// If condition variable race occurs, this will deadlock
					// Use small values since itemsSizer returns the value itself as size
					err := asyncQueue.Offer(ctx, int64(1)) // Size 1 - all items same size to maximize count
					if err != nil && err != context.DeadlineExceeded {
						t.Logf("Producer %d item %d unexpected error: %v", id, i, err)
					}
				}
			}
		}(producerID)
	}

	// Monitor for deadlock
	done := make(chan bool)
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("Stress test completed successfully - no deadlock detected")
	case <-time.After(20 * time.Second):
		// Capture stack trace for analysis
		stackBuf := make([]byte, 64*1024)
		stackSize := runtime.Stack(stackBuf, true)
		t.Logf("DEADLOCK DETECTED - Stack trace:\n%s", string(stackBuf[:stackSize]))
		t.Fatal("Deadlock detected in BlockOnOverflow + WaitForResult stress test")
	}
}
