// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
)

// TestPartitionBatcherBreakingPoint finds the request count where deadlock starts occurring
func TestPartitionBatcherBreakingPoint(t *testing.T) {
	ctx := context.Background()

	exportFunc := func(ctx context.Context, req request.Request) error {
		time.Sleep(1 * time.Millisecond) // Simulate some work
		return nil
	}

	cfg := BatchConfig{
		FlushTimeout: time.Hour, // Disable timeout-based flushing
		Sizer:        request.SizerTypeItems,
		MinSize:      1, // Process requests immediately
	}

	// Test different request counts to find breaking point
	testCases := []struct {
		name        string
		numRequests int
		timeout     time.Duration
	}{
		{"20_requests", 20, 10 * time.Second},
		{"50_requests", 50, 15 * time.Second},
		{"100_requests", 100, 20 * time.Second},
		{"200_requests", 200, 30 * time.Second},
		{"500_requests", 500, 60 * time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing with %d requests", tc.numRequests)

			// Create fresh partition batcher for each test
			partitionBatcher := newPartitionBatcher(cfg, request.NewItemsSizer(), newWorkerPool(5), exportFunc, zap.NewNop())

			require.NoError(t, partitionBatcher.Start(ctx, componenttest.NewNopHost()))
			defer func() {
				require.NoError(t, partitionBatcher.Shutdown(ctx))
			}()

			var wg sync.WaitGroup
			startTime := time.Now()

			// Launch all requests concurrently
			for i := 0; i < tc.numRequests; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()

					req := &requesttest.FakeRequest{Items: 1, Bytes: 1}
					waitDone := &waitingDone{ch: make(chan error, 1)}

					partitionBatcher.Consume(ctx, req, waitDone)

					// Wait for completion
					select {
					case err := <-waitDone.ch:
						if err != nil {
							t.Errorf("Request %d failed: %v", i, err)
						}
					case <-time.After(tc.timeout):
						t.Errorf("Request %d timed out after %v", i, tc.timeout)
					}
				}(i)
			}

			// Wait for all requests to complete
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()

			select {
			case <-done:
				elapsed := time.Since(startTime)
				t.Logf("All %d requests completed successfully in %v", tc.numRequests, elapsed)
			case <-time.After(tc.timeout + 10*time.Second):
				t.Fatalf("Test with %d requests deadlocked after %v", tc.numRequests, tc.timeout+10*time.Second)
			}
		})
	}
}

// TestWorkerPoolStressWithLogging tests worker pool under stress with detailed logging
func TestWorkerPoolStressWithLogging(t *testing.T) {
	ctx := context.Background()

	exportFunc := func(ctx context.Context, req request.Request) error {
		time.Sleep(2 * time.Millisecond) // Slightly longer work to stress the pool
		return nil
	}

	cfg := BatchConfig{
		FlushTimeout: time.Hour,
		Sizer:        request.SizerTypeItems,
		MinSize:      1,
	}

	// Small worker pool to create contention
	partitionBatcher := newPartitionBatcher(cfg, request.NewItemsSizer(), newWorkerPool(3), exportFunc, zap.NewNop())

	require.NoError(t, partitionBatcher.Start(ctx, componenttest.NewNopHost()))
	defer func() {
		require.NoError(t, partitionBatcher.Shutdown(ctx))
	}()

	const numRequests = 100
	var wg sync.WaitGroup
	completed := make(chan int, numRequests)

	t.Logf("Starting %d requests with worker pool size 3", numRequests)
	startTime := time.Now()

	// Launch requests with some delay to observe worker pool behavior
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			req := &requesttest.FakeRequest{Items: 1, Bytes: 1}
			waitDone := &waitingDone{ch: make(chan error, 1)}

			requestStart := time.Now()
			partitionBatcher.Consume(ctx, req, waitDone)

			select {
			case err := <-waitDone.ch:
				requestDuration := time.Since(requestStart)
				if err != nil {
					t.Errorf("Request %d failed after %v: %v", i, requestDuration, err)
				} else {
					t.Logf("Request %d completed in %v", i, requestDuration)
					completed <- i
				}
			case <-time.After(30 * time.Second):
				t.Errorf("Request %d timed out after 30s", i)
			}
		}(i)

		// Small delay between request launches to reduce initial burst
		if i%10 == 9 {
			time.Sleep(1 * time.Millisecond)
		}
	}

	// Monitor progress
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		completedCount := 0

		for {
			select {
			case <-completed:
				completedCount++
			case <-ticker.C:
				t.Logf("Progress: %d/%d requests completed", completedCount, numRequests)
			case <-time.After(60 * time.Second):
				return
			}
		}
	}()

	// Wait for completion with extended timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		elapsed := time.Since(startTime)
		t.Logf("All %d requests completed successfully in %v", numRequests, elapsed)
	case <-time.After(60 * time.Second):
		t.Fatal("Test deadlocked after 60 seconds")
	}
}
