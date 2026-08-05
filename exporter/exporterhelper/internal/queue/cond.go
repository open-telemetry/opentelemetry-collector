// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"container/list"
	"context"
	"sync"
)

// cond is equivalent with sync.Cond, but context.Context aware.
// Which means Wait() will return if context is done before any signal is received.
// Also, it requires the caller to hold the c.L during all calls.
//
// Every waiter owns a buffered channel of size one and is woken through that
// channel alone, so Signal and Broadcast never block. This matters because they
// are called while c.L is held: a Signal that blocked would hold the lock and
// stall every other user of the queue.
type cond struct {
	L       sync.Locker
	waiters list.List // of chan struct{}
}

func newCond(l sync.Locker) *cond {
	return &cond{L: l}
}

// Signal wakes one goroutine waiting on c, if there is any.
// It requires for the caller to hold c.L during the call.
func (c *cond) Signal() {
	e := c.waiters.Front()
	if e == nil {
		return
	}
	c.waiters.Remove(e)
	// The waiter's channel is buffered and receives at most this one signal,
	// since it is no longer in the list, so this cannot block.
	e.Value.(chan struct{}) <- struct{}{}
}

// Broadcast wakes all goroutines waiting on c.
// It requires for the caller to hold c.L during the call.
func (c *cond) Broadcast() {
	for e := c.waiters.Front(); e != nil; {
		next := e.Next()
		c.waiters.Remove(e)
		e.Value.(chan struct{}) <- struct{}{}
		e = next
	}
}

// Wait atomically unlocks c.L and suspends execution of the calling goroutine. After later resuming execution, Wait locks c.L before returning.
func (c *cond) Wait(ctx context.Context) error {
	ch := make(chan struct{}, 1)
	e := c.waiters.PushBack(ch)
	c.L.Unlock()
	select {
	case <-ctx.Done():
		c.L.Lock()
		select {
		case <-ch:
			// A signal was delivered to us between the context being done and
			// re-acquiring the lock. We are not going to use it, so pass it on
			// rather than dropping a wakeup that a queued item depends on.
			c.Signal()
		default:
			// Not signaled, so we are still queued and have to remove ourselves.
			c.waiters.Remove(e)
		}
		return ctx.Err()
	case <-ch:
		c.L.Lock()
		return nil
	}
}
