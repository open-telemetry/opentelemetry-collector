// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
import (
	"context"
	"fmt"
	"sync"

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
	shardMapMu  sync.Mutex
	shards      map[string]*singleBatcher
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
	return &multiBatcher{
		cfg:         bCfg,
		workerPool:  &workerPool,
		sizerType:   bSet.sizerType,
		sizer:       bSet.sizer,
		partitioner: bSet.partitioner,
		consumeFunc: bSet.next,
		shardMapMu:  sync.Mutex{},
		shards:      make(map[string]*singleBatcher),
	}
}

func (qb *multiBatcher) getShard(ctx context.Context, req request.Request) *singleBatcher {
	key := qb.partitioner.GetKey(ctx, req)
	qb.shardMapMu.Lock()
	defer qb.shardMapMu.Unlock()

	s, ok := qb.shards[key]
	if !ok {
		s = &singleBatcher{
			cfg:         qb.cfg,
			workerPool:  qb.workerPool,
			sizerType:   qb.sizerType,
			sizer:       qb.sizer,
			consumeFunc: qb.consumeFunc,
			stopWG:      sync.WaitGroup{},
			shutdownCh:  make(chan struct{}, 1),
		}
		qb.shards[key] = s
		_ = s.Start(ctx, nil)
	}
	return s
}

func (qb *multiBatcher) Start(ctx context.Context, host component.Host) error {
	if qb.singleShard != nil {
		return qb.singleShard.Start(ctx, host)
	}
	return nil
}

func (qb *multiBatcher) Consume(ctx context.Context, req request.Request, done Done) {
	if qb.singleShard == nil {
		qb.singleShard.Consume(ctx, req, done)
		return
	}

	shard := qb.getShard(ctx, req)
	shard.Consume(ctx, req, done)
}

func (qb *multiBatcher) Shutdown(ctx context.Context) error {
	if qb.singleShard != nil {
		return qb.singleShard.Shutdown(ctx)
	}

	qb.shardMapMu.Lock()
	defer qb.shardMapMu.Unlock()

	var g errgroup.Group
	for key, shard := range qb.shards {
		g.Go(func() error {
			return fmt.Errorf("Failed to shutdown partition %s: %w", key, shard.Shutdown(ctx))
		})
	}
	return g.Wait()
}
