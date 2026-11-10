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

	// Create LRU cache with eviction callback
	cache, err := lru.NewLRU[string, *partitionBatcher](cacheSize, func(_ string, pb *partitionBatcher) {
		// Flush the partition when evicted
		mb.wp.execute(pb.shutdownInternal)
	})
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
	defer mb.lock.Unlock()

	// Fast path: partition already exists
	if pb, ok := mb.partitions.Get(key); ok {
		return pb
	}

	// Create new partition with onEmpty callback to remove from LRU after idle timeout
	newPB := newPartitionBatcher(mb.cfg, mb.sizer, mb.mergeCtx, mb.wp, mb.consumeFunc, mb.logger, func() {
		mb.lock.Lock()
		defer mb.lock.Unlock()
		mb.partitions.Remove(key)
	})
	_ = mb.partitions.Add(key, newPB)
	_ = newPB.Start(ctx, nil)
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
