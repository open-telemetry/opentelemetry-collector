// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/internal/queue"

import (
	"context"

	"go.opentelemetry.io/collector/component"
)

// DisabledBatcher is a special-case of Batcher that has no size limit for sending. Any items read from the queue will
// be sent out (asynchronously) immediately regardless of the size.
type DisabledBatcher struct {
	BaseBatcher
}

// Start starts the goroutine that reads from the queue and flushes asynchronously.
func (qb *DisabledBatcher) Start(_ context.Context, _ component.Host) error {
	// maxWorker being -1 means batcher is disabled. This is for testing queue sender metrics.
	if qb.maxWorkers == -1 {
		return nil
	}

	qb.startWorkerPool()

	// This goroutine reads and then flushes.
	// 1. Reading from the queue is blocked until the queue is non-empty or until the queue is stopped.
	// 2. flushAsync() blocks until there are idle workers in the worker pool.
	qb.stopWG.Add(1)
	go func() {
		defer qb.stopWG.Done()
		for {
			idx, _, req, ok := qb.queue.Read(context.Background())
			if !ok {
				return
			}
			qb.flushAsync(batch{
				req:     req,
				ctx:     context.Background(),
				idxList: []uint64{idx}})
		}
	}()
	return nil
}

// Shutdown ensures that queue and all Batcher are stopped.
func (qb *DisabledBatcher) Shutdown(_ context.Context) error {
	qb.stopWG.Wait()
	return nil
}
