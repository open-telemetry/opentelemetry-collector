// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
)

// waitingDone simulates the blockingDone behavior from memory queue with WaitForResult=true
type waitingDone struct {
	ch   chan error
	done sync.Once
}

func (wd *waitingDone) OnDone(err error) {
	fmt.Printf("[DEBUG] waitingDone.OnDone called: err=%v, wd_ptr=%p\n", err, wd)
	wd.done.Do(func() {
		fmt.Printf("[DEBUG] waitingDone sending to channel: err=%v, wd_ptr=%p\n", err, wd)
		wd.ch <- err
		fmt.Printf("[DEBUG] waitingDone sent to channel: err=%v, wd_ptr=%p\n", err, wd)
	})
	fmt.Printf("[DEBUG] waitingDone.OnDone completed: err=%v, wd_ptr=%p\n", err, wd)
}

func (wd *waitingDone) Wait() error {
	return <-wd.ch
}

// TestPartitionBatcherWaitForResultDeadlock reproduces the deadlock that occurs when
// partition batcher is used with WaitForResult=true behavior under high load.
// This simulates the coordination between memory queue and partition batcher that causes the hang.
func TestPartitionBatcherWaitForResultDeadlock(t *testing.T) {
	// Stack dump goroutine for deadlock detection
	go func() {
		time.Sleep(10 * time.Second)
		stackBuf := make([]byte, 64*1024)
		stackSize := runtime.Stack(stackBuf, true)
		t.Logf("STACK DUMP after 10 seconds:\n%s", string(stackBuf[:stackSize]))
	}()

	ctx := context.Background()

	// Track exports for verification
	var exportCount int64
	exportFunc := func(ctx context.Context, req request.Request) error {
		atomic.AddInt64(&exportCount, 1)
		// Simulate some export work
		time.Sleep(1 * time.Millisecond)
		return nil
	}

	// Set up partition batcher using the proper constructor
	cfg := BatchConfig{
		FlushTimeout: time.Hour, // Disable timeout
		Sizer:        request.SizerTypeItems,
		MinSize:      100, // Moderate batch size to trigger batching
	}

	partitionBatcher := newPartitionBatcher(cfg, request.NewItemsSizer(), newWorkerPool(5), exportFunc, zap.NewNop())

	require.NoError(t, partitionBatcher.Start(ctx, componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, partitionBatcher.Shutdown(ctx))
	})

	// Simulate the pattern where many producers are waiting for results simultaneously
	const numRequests = 1000
	var succeededCount int64
	var failedCount int64

	producerWG := sync.WaitGroup{}
	for i := 0; i < numRequests; i++ {
		producerWG.Add(1)
		go func(i int) {
			defer producerWG.Done()

			req := &requesttest.FakeRequest{Items: 40, Bytes: 40}

			// Create a waiting done that simulates WaitForResult=true behavior
			waitDone := &waitingDone{ch: make(chan error, 1)}

			if i%100 == 0 {
				t.Logf("Sending request %d/%d", i+1, numRequests)
			}

			// This simulates the flow: queue.Offer() -> partition batcher -> wait for result
			// The key insight: partition batcher is called directly, but we wait for the result
			// This creates the same coordination as memoryQueue.Offer() with WaitForResult=true
			partitionBatcher.Consume(ctx, req, waitDone)

			// Wait for the result (this is what causes the blocking in WaitForResult=true)
			err := waitDone.Wait()
			if err != nil {
				atomic.AddInt64(&failedCount, 1)
				t.Logf("Request %d failed: %v", i, err)
			} else {
				atomic.AddInt64(&succeededCount, 1)
				if i%100 == 99 {
					t.Logf("Completed request %d/%d", i+1, numRequests)
				}
			}
		}(i)

		// Add slight delay to create concurrent load without overwhelming immediately
		time.Sleep(10 * time.Microsecond)
	}

	// Wait for all producers to complete
	t.Logf("Waiting for all requests to complete...")
	producerWG.Wait()

	t.Logf("All requests completed! Succeeded: %d, Failed: %d, Exports: %d",
		succeededCount, failedCount, atomic.LoadInt64(&exportCount))

	// Verify that all requests were processed successfully
	require.Equal(t, int64(numRequests), succeededCount, "All requests should succeed")
	require.Equal(t, int64(0), failedCount, "No requests should fail")
	require.True(t, atomic.LoadInt64(&exportCount) > 0, "Some exports should have occurred")
}
