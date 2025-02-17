// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue // import "go.opentelemetry.io/collector/exporter/exporterqueue"

import (
	"context"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/collector/component"
)

var donePool = sync.Pool{
	New: func() any {
		return &blockingDone{ch: make(chan error, 1)}
	},
}

func newDisabledQueue[T any](consumeFunc ConsumeFunc[T]) Queue[T] {
	return &disabledQueue[T]{
		sizer:       &requestSizer[T]{},
		consumeFunc: consumeFunc,
		size:        &atomic.Int64{},
	}
}

type disabledQueue[T any] struct {
	component.StartFunc
	component.ShutdownFunc
	consumeFunc ConsumeFunc[T]
	sizer       sizer[T]
	size        *atomic.Int64
}

func (d *disabledQueue[T]) Offer(ctx context.Context, req T) error {
	elSize := d.sizer.Sizeof(req)
	d.size.Add(elSize)

	done := donePool.Get().(*blockingDone)
	done.queue = d
	done.elSize = elSize
	d.consumeFunc(ctx, req, done)
	// Only re-add the blockingDone instance back to the pool if successfully received the
	// message from the consumer which guarantees consumer will not use that anymore,
	// otherwise no guarantee about when the consumer will add the message to the channel so cannot reuse or close.
	select {
	case doneErr := <-done.ch:
		donePool.Put(done)
		return doneErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *disabledQueue[T]) onDone(elSize int64) {
	d.size.Add(-elSize)
}

// Size returns the current number of blocked requests waiting to be processed.
func (d *disabledQueue[T]) Size() int64 {
	return d.size.Load()
}

// Capacity returns the capacity of this queue, which is 0 that means no bounds.
func (d *disabledQueue[T]) Capacity() int64 {
	return 0
}

type blockingDone struct {
	queue interface {
		onDone(int64)
	}
	elSize int64
	ch     chan error
}

func (d *blockingDone) OnDone(err error) {
	d.queue.onDone(d.elSize)
	d.ch <- err
}
