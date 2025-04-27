// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

type defaultBatcher struct {
	cfg         BatchConfig
	workerPool  *chan struct{}
	sizerType   request.SizerType
	sizer       request.Sizer[request.Request]
	partitioner Partitioner[request.Request]
	consumeFunc sender.SendFunc[request.Request]

	shardMapMu sync.Mutex
	shards     map[string]*shard
}

func newDefaultBatcher(bCfg BatchConfig, bSet batcherSettings[request.Request]) *defaultBatcher {
	// TODO: Determine what is the right behavior for this in combination with async queue.
	var workerPool chan struct{}
	if bSet.maxWorkers != 0 {
		workerPool = make(chan struct{}, bSet.maxWorkers)
		for i := 0; i < bSet.maxWorkers; i++ {
			workerPool <- struct{}{}
		}
	}
	return &defaultBatcher{
		cfg:         bCfg,
		workerPool:  &workerPool,
		sizerType:   bSet.sizerType,
		sizer:       bSet.sizer,
		partitioner: bSet.partitioner,
		consumeFunc: bSet.next,
		shardMapMu:  sync.Mutex{},
		shards:      make(map[string]*shard),
	}
}

func (qb *defaultBatcher) getShard(ctx context.Context, req request.Request) *shard {
	key := ""
	if qb.partitioner != nil {
		key = qb.partitioner.GetKey(ctx, req)
	}
	qb.shardMapMu.Lock()
	defer qb.shardMapMu.Unlock()

	s, ok := qb.shards[key]
	if !ok {
		s = &shard{
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

func (qb *defaultBatcher) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (qb *defaultBatcher) Consume(ctx context.Context, req request.Request, done Done) {
	shard := qb.getShard(ctx, req)
	shard.Consume(ctx, req, done)
}

func (qb *defaultBatcher) Shutdown(ctx context.Context) error {
	qb.shardMapMu.Lock()
	defer qb.shardMapMu.Unlock()
	stopWG := sync.WaitGroup{}
	for _, shard := range qb.shards {
		stopWG.Add(1)
		go func() {
			_ = shard.Shutdown(ctx)
			stopWG.Done()
		}()
	}
	stopWG.Wait()
	return nil
}
