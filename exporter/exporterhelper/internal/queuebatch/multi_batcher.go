// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
import (
	"context"
	"sync"

	"github.com/puzpuzpuz/xsync/v3"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

type batcherSettings[T any] struct {
	sizerType   request.SizerType
	sizer       request.Sizer[T]
	partitioner Partitioner[T]
	next        sender.SendFunc[T]
	maxWorkers  int
}

type multiBatcher struct {
	cfg         BatchConfig
	workerPool  chan struct{}
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
		workerPool:  workerPool,
		sizerType:   bSet.sizerType,
		sizer:       bSet.sizer,
		partitioner: bSet.partitioner,
		consumeFunc: bSet.next,
	}

	if bSet.partitioner == nil {
		mb.singleShard = newShard(mb.cfg, mb.sizerType, mb.sizer, mb.workerPool, mb.consumeFunc)
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
		s := newShard(mb.cfg, mb.sizerType, mb.sizer, mb.workerPool, mb.consumeFunc)
		s.start(ctx, nil)
		return s
	})
	return result
}

func (mb *multiBatcher) Start(ctx context.Context, host component.Host) error {
	if mb.singleShard != nil {
		mb.singleShard.start(ctx, host)
	}
	return nil
}

func (mb *multiBatcher) Consume(ctx context.Context, req request.Request, done Done) {
	shard := mb.getShard(ctx, req)
	shard.Consume(ctx, req, done)
}

func (mb *multiBatcher) Shutdown(ctx context.Context) error {
	if mb.singleShard != nil {
		mb.singleShard.shutdown(ctx)
		return nil
	}

	var wg sync.WaitGroup
	wg.Add(mb.shards.Size())
	mb.shards.Range(func(_ string, shard *shardBatcher) bool {
		go func() {
			shard.shutdown(ctx)
			wg.Done()
		}()
		return true
	})
	wg.Wait()
	return nil
}
