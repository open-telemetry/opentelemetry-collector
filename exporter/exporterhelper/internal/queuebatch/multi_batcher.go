// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
import (
	"context"
	"errors"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2/simplelru"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	queuebatchtelemetry "go.opentelemetry.io/collector/internal/telemetry/queuebatch"
)

type multiBatcher struct {
	cfg         BatchConfig
	wp          *workerPool
	sizer       request.Sizer
	partitioner Partitioner[request.Request]
	mergeCtx    func(context.Context, context.Context) context.Context
	consumeFunc sender.SendFunc[request.Request]
	partitions  *lru.LRU[string, *partitionBatcher]
	logger      *zap.Logger
	lock        sync.Mutex
}

func newMultiBatcher(
	bCfg BatchConfig,
	sizer request.Sizer,
	wp *workerPool,
	set batcherSettings[request.Request],
) (*multiBatcher, error) {
	mb := &multiBatcher{
		cfg:         bCfg,
		wp:          wp,
		sizer:       sizer,
		partitioner: set.partitioner,
		mergeCtx:    set.mergeCtx,
		consumeFunc: set.next,
		logger:      set.logger,
	}

	cacheSize := bCfg.Partition.CacheSize

	// Create LRU cache. Evictions are handled in getPartition so the shutdown of
	// the evicted partition can be scheduled after mb.lock is released.
	cache, err := lru.NewLRU[string, *partitionBatcher](cacheSize, nil)
	if err != nil {
		return nil, err
	}

	mb.partitions = cache

	if err := errors.Join(
		set.obsMetrics.RegisterInt(queuebatchtelemetry.MetricPartitionCacheSize, mb.getActivePartitionsCount),
		set.obsMetrics.RegisterInt(queuebatchtelemetry.MetricPartitionCacheCapacity, func() int64 { return int64(cacheSize) }),
	); err != nil {
		return nil, err
	}

	return mb, nil
}

func (mb *multiBatcher) getPartition(ctx context.Context, req request.Request) *partitionBatcher {
	key := mb.partitioner.GetKey(ctx, req)

	mb.lock.Lock()

	// Fast path: partition already exists
	if pb, ok := mb.partitions.Get(key); ok {
		mb.lock.Unlock()
		return pb
	}

	// Create the new partition. onEmpty is assigned right after construction so
	// the closure can reference the partition itself.
	newPB := newPartitionBatcher(mb.cfg, mb.sizer, mb.mergeCtx, mb.wp, mb.consumeFunc, mb.logger, nil)
	// onEmpty removes the partition from the LRU after the idle timeout. The
	// partition must then be shut down so its timer goroutine exits and a
	// pending batch is not stranded; schedule that after mb.lock is released.
	// The final flush runs on the worker executing the shutdown, so a busy
	// pool cannot wedge.
	newPB.onEmpty = func() {
		mb.lock.Lock()
		removed := mb.partitions.Remove(key)
		mb.lock.Unlock()
		if removed {
			mb.wp.execute(func() { newPB.shutdownInternal(true) })
		}
	}

	// Adding a new key to a full cache evicts the oldest partition. Remove it
	// explicitly so its shutdown can be scheduled after mb.lock is released:
	// workerPool.execute blocks until a worker is free, and waiting on that
	// while holding mb.lock would stall every other partition.
	var evicted *partitionBatcher
	if mb.partitions.Len() >= mb.cfg.Partition.CacheSize {
		_, evicted, _ = mb.partitions.RemoveOldest()
	}
	_ = mb.partitions.Add(key, newPB)
	// Start the partition before releasing the lock so its timer is initialized
	// before any other caller can pick it up from the cache.
	_ = newPB.Start(ctx, nil)
	mb.lock.Unlock()

	if evicted != nil {
		// Flush the evicted partition. The final flush runs on the worker that
		// executes the shutdown, so this never needs two workers at once.
		mb.wp.execute(func() { evicted.shutdownInternal(true) })
	}
	return newPB
}

func (mb *multiBatcher) Start(context.Context, component.Host) error {
	return nil
}

func (mb *multiBatcher) Consume(ctx context.Context, req request.Request, done queue.Done) {
	shard := mb.getPartition(ctx, req)
	shard.Consume(ctx, req, done)
}

func (mb *multiBatcher) getActivePartitionsCount() int64 {
	mb.lock.Lock()
	defer mb.lock.Unlock()
	return int64(mb.partitions.Len())
}

func (mb *multiBatcher) Shutdown(ctx context.Context) error {
	var wg sync.WaitGroup
	mb.lock.Lock()
	defer mb.lock.Unlock()
	for _, key := range mb.partitions.Keys() {
		if pb, ok := mb.partitions.Peek(key); ok {
			wg.Go(func() {
				_ = pb.Shutdown(ctx)
			})
		}
	}
	wg.Wait()
	mb.partitions.Purge()
	return nil
}
