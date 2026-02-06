// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
import (
	"context"
	"sync"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

type multiBatcher struct {
	cfg                      BatchConfig
	wp                       *workerPool
	sizer                    request.Sizer
	partitioner              Partitioner[request.Request]
	mergeCtx                 func(context.Context, context.Context) context.Context
	consumeFunc              sender.SendFunc[request.Request]
	shards                   map[string]any
	logger                   *zap.Logger
	maxActivePartitionsCount int64
	activePartitionsCount    int64
	lruKeys                  lru
	mu                       sync.Mutex
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
		cfg:                      bCfg,
		wp:                       wp,
		sizer:                    sizer,
		partitioner:              partitioner,
		mergeCtx:                 mergeCtx,
		consumeFunc:              next,
		logger:                   logger,
		maxActivePartitionsCount: int64(10000), // TODO: fix this by reading from config.
		activePartitionsCount:    0,
		lruKeys:                  lru{km: make(map[string]*lruNode), h: nil, t: nil},
		shards:                   make(map[string]any),
	}
}

func (mb *multiBatcher) evictPartitionKey() {
	key := mb.lruKeys.keyToEvict()
	// remove the key from the map.
	partition, _ := mb.shards[key]
	mb.activePartitionsCount--
	// flush partition on worker pool.
	pb := partition.(*partitionBatcher)
	mb.wp.execute(pb.flushCurrentBatchIfNecessary)
	mb.lruKeys.evict(key)
	// confirm if flush can still go on worker pool when we delete shard.
	delete(mb.shards, key)
}

func (mb *multiBatcher) getPartition(ctx context.Context, req request.Request) *partitionBatcher {
	key := mb.partitioner.GetKey(ctx, req)
	mb.mu.Lock()
	defer mb.mu.Unlock()
	s, found := mb.shards[key]
	// Fast path, shard already created.
	if found {
		return s.(*partitionBatcher)
	}

	if mb.activePartitionsCount > mb.maxActivePartitionsCount {
		// we need to evict a key.
		mb.evictPartitionKey()
	}

	newS := newPartitionBatcher(mb.cfg, mb.sizer, mb.mergeCtx, mb.wp, mb.consumeFunc, mb.logger)
	mb.shards[key] = newS
	mb.activePartitionsCount++
	mb.lruKeys.access(key)
	return s.(*partitionBatcher)
}

func (mb *multiBatcher) Start(context.Context, component.Host) error {
	return nil
}

func (mb *multiBatcher) Consume(ctx context.Context, req request.Request, done queue.Done) {
	shard := mb.getPartition(ctx, req)
	shard.Consume(ctx, req, done)
}

func (mb *multiBatcher) Shutdown(ctx context.Context) error {
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
