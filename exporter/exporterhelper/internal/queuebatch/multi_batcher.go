// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
import (
	"context"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

type multiBatcher struct {
	cfg         BatchConfig
	wp          *workerPool
	sizer       request.Sizer
	partitioner Partitioner[request.Request]
	mergeCtx    func(context.Context, context.Context) context.Context
	consumeFunc sender.SendFunc[request.Request]
	partitions  *lru.Cache[string, *partitionBatcher]
	logger      *zap.Logger
}

func newMultiBatcher(
	bCfg BatchConfig,
	sizer request.Sizer,
	wp *workerPool,
	partitioner Partitioner[request.Request],
	mergeCtx func(context.Context, context.Context) context.Context,
	next sender.SendFunc[request.Request],
	logger *zap.Logger,
) (*multiBatcher, error) {
	mb := &multiBatcher{
		cfg:         bCfg,
		wp:          wp,
		sizer:       sizer,
		partitioner: partitioner,
		mergeCtx:    mergeCtx,
		consumeFunc: next,
		logger:      logger,
	}

	// Create LRU cache with eviction callback
	// TODO: make maxActivePartitionsCount configurable
	cache, err := lru.NewWithEvict[string, *partitionBatcher](10000, func(_ string, pb *partitionBatcher) {
		// Flush the partition when evicted
		mb.wp.execute(pb.flushCurrentBatchIfNotEmpty)
	})
	mb.partitions = cache

	if err != nil {
		return nil, err
	}

	return mb, nil
}

func (mb *multiBatcher) getPartition(ctx context.Context, req request.Request) *partitionBatcher {
	key := mb.partitioner.GetKey(ctx, req)

	// Fast path: partition already exists
	if pb, ok := mb.partitions.Get(key); ok {
		return pb
	}

	// Create new partition
	newPB := newPartitionBatcher(mb.cfg, mb.sizer, mb.mergeCtx, mb.wp, mb.consumeFunc, mb.logger)

	// Atomic add - only adds if key doesn't exist
	existed, _ := mb.partitions.ContainsOrAdd(key, newPB)
	if existed {
		// Another goroutine added it first - shut down our unused partition
		_ = newPB.Shutdown(ctx)
		// Return the one that was actually stored
		pb, _ := mb.partitions.Get(key)
		return pb
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
	return int64(mb.partitions.Len())
}

func (mb *multiBatcher) Shutdown(ctx context.Context) error {
	var wg sync.WaitGroup
	for _, key := range mb.partitions.Keys() {
		if pb, ok := mb.partitions.Peek(key); ok {
			wg.Add(1)
			go func(s *partitionBatcher) {
				defer wg.Done()
				_ = s.Shutdown(ctx)
			}(pb)
		}
	}
	wg.Wait()
	return nil
}
