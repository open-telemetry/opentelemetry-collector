// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue // import "go.opentelemetry.io/collector/exporter/exporterqueue"

import (
	"context"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/collector/component"
)

var chanPool = sync.Pool{
	New: func() interface{} {
		return make(chan struct{}, 1)
	},
}

func newDisabledQueue[T any](consumeFunc ConsumeFunc[T]) Queue[T] {
	return &disabledQueue[T]{
		consumeFunc: consumeFunc,
		size:        &atomic.Int64{},
	}
}

type disabledQueue[T any] struct {
	component.StartFunc
	component.ShutdownFunc
	consumeFunc ConsumeFunc[T]
	size        *atomic.Int64
}

func (d *disabledQueue[T]) Offer(ctx context.Context, req T) error {
	ch := chanPool.Get().(chan struct{})
	defer chanPool.Put(ch)
	done := func(err error) {
		ch <- struct{}{}
	}
	d.size.Add(1)
	d.consumeFunc(ctx, req, done)
	<-ch
	d.size.Add(-1)
	return nil
}

// Size returns the current number of blocked requests waiting to be processed.
func (d *disabledQueue[T]) Size() int64 {
	return d.size.Load()
}

// Capacity returns the capacity of this queue, which is 0 that means no bounds.
func (d *disabledQueue[T]) Capacity() int64 {
	return 0
}
