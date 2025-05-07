// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
import (
	"context"
	"fmt"
	"sync"

	"github.com/puzpuzpuz/xsync/v3"
	"golang.org/x/sync/errgroup"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

type multiBatcher struct {
	cfg         BatchConfig
	workerPool  *chan struct{}
	sizerType   request.SizerType
	sizer       request.Sizer[request.Request]
	partitioner Partitioner[request.Request]
	consumeFunc sender.SendFunc[request.Request]

	singleShard *shardBatcher
	shards      *xsync.MapOf[string, *shardBatcher]
}

var _ Batcher[request.Request] = (*multiBatcher)(nil)

func newMultiBatcher(bCfg BatchConfig, bSet batcherSettings[request.Request]) *multiBatcher {
	var workerPool chan struct{}
	if bSet.maxWorkers != 0 {
		workerPool = make(chan struct{}, bSet.maxWorkers)
		for i := 0; i < bSet.maxWorkers; i++ {
			workerPool <- struct{}{}
		}
	}
	mb := &multiBatcher{
		cfg:         bCfg,
		workerPool:  &workerPool,
		sizerType:   bSet.sizerType,
		sizer:       bSet.sizer,
		partitioner: bSet.partitioner,
		consumeFunc: bSet.next,
	}

	if bSet.partitioner == nil {
		mb.singleShard = &shardBatcher{
			cfg:         bCfg,
			workerPool:  mb.workerPool,
			sizerType:   bSet.sizerType,
			sizer:       bSet.sizer,
			consumeFunc: bSet.next,
			stopWG:      sync.WaitGroup{},
			shutdownCh:  make(chan struct{}, 1),
		}
	} else {
		mb.shards = xsync.NewMapOf[string, *shardBatcher]()
	}
	return mb
}

func (mb *multiBatcher) getShard(ctx context.Context, req request.Request) *shardBatcher {
	if mb.singleShard != nil {
		return mb.singleShard
	}

	key := mb.partitioner.GetKey(ctx, req)
	result, _ := mb.shards.LoadOrCompute(key, func() *shardBatcher {
		s := &shardBatcher{
			cfg:         mb.cfg,
			workerPool:  mb.workerPool,
			sizerType:   mb.sizerType,
			sizer:       mb.sizer,
			consumeFunc: mb.consumeFunc,
			stopWG:      sync.WaitGroup{},
			shutdownCh:  make(chan struct{}, 1),
		}
		_ = s.Start(ctx, nil)
		return s
	})
	return result
}

func (mb *multiBatcher) Start(ctx context.Context, host component.Host) error {
	if mb.singleShard != nil {
		return mb.singleShard.Start(ctx, host)
	}
	return nil
}

func (mb *multiBatcher) Consume(ctx context.Context, req request.Request, done Done) {
	shard := mb.getShard(ctx, req)
	shard.Consume(ctx, req, done)
}

func (mb *multiBatcher) Shutdown(ctx context.Context) error {
	if mb.singleShard != nil {
		return mb.singleShard.Shutdown(ctx)
	}

	var g errgroup.Group
	mb.shards.Range(func(key string, shard *shardBatcher) bool {
		g.Go(func() error {
			if err := shard.Shutdown(ctx); err != nil {
				return fmt.Errorf("failed to shutdown partition %s: %w", key, err)
			}
			return nil
		})
		return true
	})
	return g.Wait()
}
