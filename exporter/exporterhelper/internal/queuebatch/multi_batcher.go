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
	shards                   sync.Map
	logger                   *zap.Logger
	maxActivePartitionsCount int64
	activePartitionsCount    int64
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
	}
}

func (mb *multiBatcher) evictPartitionKey() {
	// TODO: find the key to be evicted.
	key := "dummy"
	// remove the key from the map.
	partition, loaded := mb.shards.LoadAndDelete(key)
	if !loaded {
		// Some other thread might have evicted the key and that will take responsibility to flush the partition.
		return
	}
	// flush partition on worker pool.
	pb := partition.(*partitionBatcher)
	mb.wp.execute(pb.flushCurrentBatchIfNecessary)
}

func (mb *multiBatcher) getPartition(ctx context.Context, req request.Request) *partitionBatcher {
	key := mb.partitioner.GetKey(ctx, req)
	s, found := mb.shards.Load(key)
	// Fast path, shard already created.
	if found {
		return s.(*partitionBatcher)
	}

	if mb.activePartitionsCount > mb.maxActivePartitionsCount {
		// we need to evict a key.
		// TODO uncomment this later.
		// mb.evictPartitionKey()
	}

	newS := newPartitionBatcher(mb.cfg, mb.sizer, mb.mergeCtx, mb.wp, mb.consumeFunc, mb.logger)
	mb.activePartitionsCount++
	_ = newS.Start(ctx, nil)
	s, loaded := mb.shards.LoadOrStore(key, newS)
	// If not loaded, there was a race condition in adding the new shard. Shutdown the newly created shard.
	if loaded {
		_ = newS.Shutdown(ctx)
		mb.activePartitionsCount--
	}
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
