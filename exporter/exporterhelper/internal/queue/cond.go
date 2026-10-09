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
type cond struct {
	L sync.Locker
	// Each waiter has a private buffered channel. A true notification is a Signal
	// that must be forwarded if cancellation wins; a false one is a Broadcast.
	waiters list.List // of chan bool, guarded by L
}

func newCond(l sync.Locker) *cond {
	return &cond{L: l}
}

// Signal wakes one goroutine waiting on c, if there is any.
// It requires for the caller to hold c.L during the call.
func (c *cond) Signal() {
	c.notify(true)
}

// Broadcast wakes all goroutines waiting on c.
// It requires for the caller to hold c.L during the call.
func (c *cond) Broadcast() {
	for c.waiters.Len() > 0 {
		c.notify(false)
	}
}

func (c *cond) notify(signal bool) {
	e := c.waiters.Front()
	if e == nil {
		return
	}
	c.waiters.Remove(e)
	// Retirement under L ensures this is the only send to this buffered channel,
	// so it cannot block even if the waiter has already selected cancellation.
	e.Value.(chan bool) <- signal
}

// Wait atomically unlocks c.L and suspends execution of the calling goroutine. After later resuming execution, Wait locks c.L before returning.
func (c *cond) Wait(ctx context.Context) error {
	ch := make(chan bool, 1)
	e := c.waiters.PushBack(ch)
	c.L.Unlock()
	select {
	case <-ctx.Done():
		c.L.Lock()
		select {
		case signal := <-ch:
			if signal {
				// This waiter was selected while cancellation waited for L. Pass
				// its unused signal on so another waiter can recheck its predicate.
				c.Signal()
			}
		default:
			c.waiters.Remove(e)
		}
		return ctx.Err()
	case <-ch:
		c.L.Lock()
		return nil
	}
}
