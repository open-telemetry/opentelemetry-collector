// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
import (
	"context"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2/simplelru"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

const (
	// exporterKey used to identify exporters in metrics.
	exporterKey = "exporter"
	// dataTypeKey used to identify the data type in partition cache metrics.
	dataTypeKey = "data_type"
)

type multiBatcher struct {
	cfg         BatchConfig
	wp          *workerPool
	sizer       request.Sizer
	partitioner Partitioner[request.Request]
	mergeCtx    func(context.Context, context.Context) context.Context
	consumeFunc sender.SendFunc[request.Request]
	partitions  *lru.LRU[string, *partitionBatcher]
	cacheSize   int
	tb          *metadata.TelemetryBuilder
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
		cacheSize:   bCfg.cacheSize(),
		logger:      set.logger,
	}

	// Create LRU cache with eviction callback
	cache, err := lru.NewLRU[string, *partitionBatcher](mb.cacheSize, func(_ string, pb *partitionBatcher) {
		// Flush the partition when evicted
		mb.wp.execute(pb.shutdownInternal)
	})
	if err != nil {
		return nil, err
	}

	mb.partitions = cache

	tb, err := metadata.NewTelemetryBuilder(set.telemetry)
	if err != nil {
		return nil, err
	}
	mb.tb = tb

	asyncAttr := metric.WithAttributeSet(attribute.NewSet(
		attribute.String(exporterKey, set.id.String()),
		attribute.String(dataTypeKey, set.signal.String()),
	))
	if err = tb.RegisterExporterQueueBatchPartitionCacheSizeCallback(func(_ context.Context, o metric.Int64Observer) error {
		o.Observe(mb.getActivePartitionsCount(), asyncAttr)
		return nil
	}); err != nil {
		tb.Shutdown()
		return nil, err
	}
	if err = tb.RegisterExporterQueueBatchPartitionCacheCapacityCallback(func(_ context.Context, o metric.Int64Observer) error {
		o.Observe(int64(mb.cacheSize), asyncAttr)
		return nil
	}); err != nil {
		tb.Shutdown()
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
	defer mb.tb.Shutdown()
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
