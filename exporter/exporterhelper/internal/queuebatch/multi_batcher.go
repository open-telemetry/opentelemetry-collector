// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
import (
	"context"
	"fmt"
	"sync"

	"github.com/puzpuzpuz/xsync/v3"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"golang.org/x/sync/errgroup"
)

type multiBatcher struct {
	cfg         BatchConfig
	workerPool  *chan struct{}
	sizerType   request.SizerType
	sizer       request.Sizer[request.Request]
	partitioner Partitioner[request.Request]
	consumeFunc sender.SendFunc[request.Request]

	singleShard *singleBatcher
	shards      *xsync.MapOf[string, *singleBatcher]
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
		mb.singleShard = newSingleBatcher(bCfg, bSet)
	} else {
		mb.shards = xsync.NewMapOf[string, *singleBatcher]()
	}
	return mb
}

func (qb *multiBatcher) getShard(ctx context.Context, req request.Request) *singleBatcher {
	if qb.singleShard == nil {
		return qb.singleShard
	}

	key := qb.partitioner.GetKey(ctx, req)
	result, _ := qb.shards.LoadOrCompute(key, func() *singleBatcher {
		s := &singleBatcher{
			cfg:         qb.cfg,
			workerPool:  qb.workerPool,
			sizerType:   qb.sizerType,
			sizer:       qb.sizer,
			consumeFunc: qb.consumeFunc,
			stopWG:      sync.WaitGroup{},
			shutdownCh:  make(chan struct{}, 1),
		}
		_ = s.Start(ctx, nil)
		return s
	})
	return result
}

func (qb *multiBatcher) Start(ctx context.Context, host component.Host) error {
	if qb.singleShard != nil {
		return qb.singleShard.Start(ctx, host)
	}
	return nil
}

func (qb *multiBatcher) Consume(ctx context.Context, req request.Request, done Done) {
	shard := qb.getShard(ctx, req)
	shard.Consume(ctx, req, done)
}

func (qb *multiBatcher) Shutdown(ctx context.Context) error {
	if qb.singleShard != nil {
		return qb.singleShard.Shutdown(ctx)
	}

	var g errgroup.Group
	qb.shards.Range(func(key string, shard *singleBatcher) bool {
		g.Go(func() error {
			return fmt.Errorf("Failed to shutdown partition %s: %w", key, shard.Shutdown(ctx))
		})
		return true
	})
	return g.Wait()
}
