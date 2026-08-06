// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"fmt"
	"sync/atomic"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

var _ Batcher[request.Request] = (*shardedBatcher)(nil)

// shardedBatcher spreads unpartitioned requests across independent partitionBatcher instances.
// All shards share one worker pool, so worker concurrency remains bounded by the queue setting.
type shardedBatcher struct {
	shards []*partitionBatcher
	next   atomic.Uint64
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
		shards: make([]*partitionBatcher, 0, shardCount),
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
	return err
}

func (sb *shardedBatcher) Consume(ctx context.Context, req request.Request, done queue.Done) {
	idx := sb.next.Add(1) - 1
	sb.shards[idx%uint64(len(sb.shards))].Consume(ctx, req, done)
}

func (sb *shardedBatcher) Shutdown(ctx context.Context) error {
	var err error
	for _, shard := range sb.shards {
		err = multierr.Append(err, shard.Shutdown(ctx))
	}
	return err
}
