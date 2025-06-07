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

var _ Batcher[request.Request] = (*multiBatcher)(nil)

type multiBatcher struct {
	cfg         BatchConfig
	wp          *workerPool
	sizerType   request.SizerType
	sizer       request.Sizer[request.Request]
	partitioner Partitioner[request.Request]
	consumeFunc sender.SendFunc[request.Request]
	shards      sync.Map
}

func newMultiBatcher(
	bCfg BatchConfig,
	sizerType request.SizerType,
	sizer request.Sizer[request.Request],
	wp *workerPool,
	partitioner Partitioner[request.Request],
	next sender.SendFunc[request.Request],
) *multiBatcher {
	return &multiBatcher{
		cfg:         bCfg,
		wp:          wp,
		sizerType:   sizerType,
		sizer:       sizer,
		partitioner: partitioner,
		consumeFunc: next,
	}
}

func (mb *multiBatcher) getPartition(ctx context.Context, req request.Request) *partitionBatcher {
	key := mb.partitioner.GetKey(ctx, req)
	s, found := mb.shards.Load(key)
	// Fast path, shard already created.
	if found {
		return s.(*partitionBatcher)
	}
	newS := newPartitionBatcher(mb.cfg, mb.sizerType, mb.sizer, mb.wp, mb.consumeFunc)
	_ = newS.Start(ctx, nil)
	s, loaded := mb.shards.LoadOrStore(key, newS)
	// If not loaded, there was a race condition in adding the new shard. Shutdown the newly created shard.
	if loaded {
		_ = newS.Shutdown(ctx)
	}
	return s.(*partitionBatcher)
}

func (mb *multiBatcher) Start(context.Context, component.Host) error {
	return nil
}

func (mb *multiBatcher) Consume(ctx context.Context, req request.Request, done Done) {
	shard := mb.getPartition(ctx, req)
	shard.Consume(ctx, req, done)
}

func (mb *multiBatcher) Shutdown(ctx context.Context) error {
	var wg sync.WaitGroup
	mb.shards.Range(func(_ any, shard any) bool {
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
