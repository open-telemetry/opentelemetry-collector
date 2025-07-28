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

// TestSimpleWorkerPoolDeadlock tests just the worker pool coordination issue
func TestSimpleWorkerPoolDeadlock(t *testing.T) {
	ctx := context.Background()

	// Simple export function that just sleeps briefly
	exportFunc := func(ctx context.Context, req request.Request) error {
		time.Sleep(1 * time.Millisecond)
		return nil
	}

	// Create partition batcher with small worker pool to stress test
	cfg := BatchConfig{
		FlushTimeout: time.Hour, // Disable timeout
		Sizer:        request.SizerTypeItems,
		MinSize:      1, // Process requests immediately
	}

	partitionBatcher := newPartitionBatcher(cfg, request.NewItemsSizer(), newWorkerPool(2), exportFunc, zap.NewNop())

	require.NoError(t, partitionBatcher.Start(ctx, componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, partitionBatcher.Shutdown(ctx))
	})

	// Send a small number of requests with waitingDone to see if any get stuck
	const numRequests = 10
	var wg sync.WaitGroup

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			req := &requesttest.FakeRequest{Items: 1, Bytes: 1}
			waitDone := &waitingDone{ch: make(chan error, 1)}

			t.Logf("Sending request %d", i)
			partitionBatcher.Consume(ctx, req, waitDone)

			// Wait for completion with timeout
			select {
			case err := <-waitDone.ch:
				if err != nil {
					t.Errorf("Request %d failed: %v", i, err)
				} else {
					t.Logf("Request %d completed successfully", i)
				}
			case <-time.After(5 * time.Second):
				t.Errorf("Request %d timed out", i)
			}
		}(i)

		// Small delay between requests
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for all requests to complete with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("All requests completed successfully")
	case <-time.After(30 * time.Second):
		t.Fatal("Test timed out - deadlock detected")
	}
}
