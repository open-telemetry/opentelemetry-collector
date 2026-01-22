// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"errors"
	"sync"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

// errBatcherStopped is returned when Consume is called after the batcher has been stopped.
var errBatcherStopped = errors.New("batcher is stopped")

type multiBatcher struct {
	cfg         BatchConfig
	wp          *workerPool
	sizer       request.Sizer
	partitioner Partitioner[request.Request]
	mergeCtx    func(context.Context, context.Context) context.Context
	consumeFunc sender.SendFunc[request.Request]
	shards      sync.Map
	logger      *zap.Logger

	// mu protects stopped flag and synchronizes shard creation with shutdown.
	mu      sync.Mutex
	stopped bool
}

func newMultiBatcher(
	bCfg BatchConfig,
	sizer request.Sizer,
	wp *workerPool,
	partitioner Partitioner[request.Request],
	mergeCtx func(context.Context, context.Context) context.Context,
	next sender.SendFunc[request.Request],
	logger *zap.Logger,
) *multiBatcher {
	return &multiBatcher{
		cfg:         bCfg,
		wp:          wp,
		sizer:       sizer,
		partitioner: partitioner,
		mergeCtx:    mergeCtx,
		consumeFunc: next,
		logger:      logger,
	}
}

func (mb *multiBatcher) getPartition(ctx context.Context, req request.Request) (*partitionBatcher, bool) {
	key := mb.partitioner.GetKey(ctx, req)
	s, found := mb.shards.Load(key)
	// Fast path, shard already created.
	if found {
		return s.(*partitionBatcher), true
	}

	// Slow path: need to create a new shard. Synchronize with Shutdown to prevent
	// goroutine leaks from shards created after Shutdown has started iterating.
	mb.mu.Lock()
	if mb.stopped {
		mb.mu.Unlock()
		return nil, false
	}

	// Double-check after acquiring lock in case another goroutine added the shard.
	s, found = mb.shards.Load(key)
	if found {
		mb.mu.Unlock()
		return s.(*partitionBatcher), true
	}

	newS := newPartitionBatcher(mb.cfg, mb.sizer, mb.mergeCtx, mb.wp, mb.consumeFunc, mb.logger)
	_ = newS.Start(ctx, nil)
	mb.shards.Store(key, newS)
	mb.mu.Unlock()

	return newS, true
}

func (mb *multiBatcher) Start(context.Context, component.Host) error {
	return nil
}

func (mb *multiBatcher) Consume(ctx context.Context, req request.Request, done queue.Done) {
	shard, ok := mb.getPartition(ctx, req)
	if !ok {
		// Batcher is stopped, signal done with shutdown error.
		done.OnDone(errBatcherStopped)
		return
	}
	shard.Consume(ctx, req, done)
}

func (mb *multiBatcher) Shutdown(ctx context.Context) error {
	// Set stopped flag while holding the lock to prevent new shards from being
	// created after we start iterating. This ensures all shards are properly shutdown.
	mb.mu.Lock()
	mb.stopped = true
	mb.mu.Unlock()

	var wg sync.WaitGroup
	mb.shards.Range(func(_, shard any) bool {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = shard.(*partitionBatcher).Shutdown(ctx)
		}()
		return true
	})
	wg.Wait()
	return nil
}
