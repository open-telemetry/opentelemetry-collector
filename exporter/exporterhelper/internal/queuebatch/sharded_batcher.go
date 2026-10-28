// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

var (
	_ Batcher[request.Request] = (*shardedBatcher)(nil)
	_ fullQueueFlusher         = (*shardedBatcher)(nil)
)

// fullQueueFlusher is a Batcher that can flush its partial batches when the queue runs out of
// capacity, so producers do not wait for min_size or flush_timeout to free space.
type fullQueueFlusher interface {
	// onQueueFull returns the hook the queue calls when it has no free capacity, or nil when the
	// batcher does not need to be notified. The hook must not block.
	onQueueFull() func()
}

// shardedBatcher spreads unpartitioned requests across independent partitionBatcher instances.
// All shards share one worker pool, so worker concurrency remains bounded by the queue setting.
type shardedBatcher struct {
	shards []*partitionBatcher
	next   atomic.Uint64

	// flushCh has a buffer of one so repeated flush requests collapse into one.
	flushCh      chan struct{}
	shutdownCh   chan struct{}
	shutdownOnce sync.Once
	stopWG       sync.WaitGroup
}

func newShardedBatcher(
	cfg BatchConfig,
	sizer request.Sizer,
	mergeCtx func(context.Context, context.Context) context.Context,
	wp *workerPool,
	next sender.SendFunc[request.Request],
	logger *zap.Logger,
	shardCount int,
) (*shardedBatcher, error) {
	if shardCount <= 0 {
		return nil, fmt.Errorf("queue_batch: shard count must be positive, found %d", shardCount)
	}

	sb := &shardedBatcher{
		shards:     make([]*partitionBatcher, 0, shardCount),
		flushCh:    make(chan struct{}, 1),
		shutdownCh: make(chan struct{}),
	}
	for range shardCount {
		sb.shards = append(sb.shards, newPartitionBatcher(cfg, sizer, mergeCtx, wp, next, logger, nil))
	}
	return sb, nil
}

func (sb *shardedBatcher) Start(ctx context.Context, host component.Host) error {
	var err error
	for _, shard := range sb.shards {
		err = multierr.Append(err, shard.Start(ctx, host))
	}
	if sb.flushesOnFull() {
		sb.stopWG.Go(func() {
			for {
				select {
				case <-sb.shutdownCh:
					return
				case <-sb.flushCh:
					sb.flushPartialBatches()
				}
			}
		})
	}
	return err
}

// flushesOnFull reports whether a full queue must flush the partial batches. With a single shard
// the batcher behaves like the previous unsharded one, so neither the hook nor the flush
// goroutine is wired.
func (sb *shardedBatcher) flushesOnFull() bool {
	return len(sb.shards) > 1
}

func (sb *shardedBatcher) onQueueFull() func() {
	if !sb.flushesOnFull() {
		return nil
	}
	return sb.requestFlush
}

func (sb *shardedBatcher) requestFlush() {
	select {
	case sb.flushCh <- struct{}{}:
	default:
	}
}

func (sb *shardedBatcher) flushPartialBatches() {
	for _, shard := range sb.shards {
		shard.flushCurrentBatchOrRemovePartition()
	}
}

func (sb *shardedBatcher) Consume(ctx context.Context, req request.Request, done queue.Done) {
	idx := sb.next.Add(1) - 1
	sb.shards[idx%uint64(len(sb.shards))].Consume(ctx, req, done)
}

func (sb *shardedBatcher) Shutdown(ctx context.Context) error {
	sb.shutdownOnce.Do(func() { close(sb.shutdownCh) })
	sb.stopWG.Wait()
	var err error
	for _, shard := range sb.shards {
		err = multierr.Append(err, shard.Shutdown(ctx))
	}
	return err
}
