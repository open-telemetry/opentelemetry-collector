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
)

type multiBatcher struct {
	cfg              BatchConfig
	wp               *workerPool
	sizer            request.Sizer
	partitioner      Partitioner[request.Request]
	mergeCtx         func(context.Context, context.Context) context.Context
	consumeFunc      sender.SendFunc[request.Request]
	partitions       *lru.LRU[string, *partitionBatcher]
	logger           *zap.Logger
	cardinalityLimit int
	lock             sync.Mutex
}

const defaultCardinalityLimit = 10000

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

	if bCfg.Partition.CardinalityLimit == nil {
		mb.cardinalityLimit = defaultCardinalityLimit
	} else {
		mb.cardinalityLimit = *bCfg.Partition.CardinalityLimit
	}

	cache, err := lru.NewLRU[string, *partitionBatcher](mb.cardinalityLimit, func(_ string, pb *partitionBatcher) {
		// Flush the partition when evicted
		mb.wp.execute(pb.shutdownInternal)
	})
	if err != nil {
		return nil, err
	}

	mb.partitions = cache
	return mb, nil
}

func (mb *multiBatcher) getPartition(ctx context.Context, req request.Request) (*partitionBatcher, error) {
	key := mb.partitioner.GetKey(ctx, req)

	mb.lock.Lock()
	defer mb.lock.Unlock()

	// Fast path: partition already exists
	if pb, ok := mb.partitions.Get(key); ok {
		return pb, nil
	}

	// Atomic check: only reject if this would be a NEW partition beyond the limit
	if mb.partitions.Len() >= mb.cardinalityLimit {
		mb.logger.Error("partition cardinality limit reached, rejecting the data", zap.Int("partition_count", mb.partitions.Len()))
		return nil, errors.New("partition cardinality limit reached")
	}

	// Create new partition with onEmpty callback to remove from LRU after idle timeout
	newPB := newPartitionBatcher(mb.cfg, mb.sizer, mb.mergeCtx, mb.wp, mb.consumeFunc, mb.logger, func() {
		mb.lock.Lock()
		defer mb.lock.Unlock()
		mb.partitions.Remove(key)
	})
	_ = mb.partitions.Add(key, newPB)
	_ = newPB.Start(ctx, nil)
	return newPB, nil
}

func (mb *multiBatcher) Start(context.Context, component.Host) error {
	return nil
}

func (mb *multiBatcher) Consume(ctx context.Context, req request.Request, done queue.Done) {
	shard, err := mb.getPartition(ctx, req)
	if err != nil {
		done.OnDone(err)
		return
	}
	shard.Consume(ctx, req, done)
}

// getActivePartitionsCount is test only method
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
