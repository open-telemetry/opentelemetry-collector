// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"
	"runtime"
	"sync"
)

// cond is equivalent with sync.Cond, but context.Context aware.
// Which means Wait() will return if context is done before any signal is received.
// Also, it requires the caller to hold the c.L during all calls.
type cond struct {
	L       sync.Locker
	ch      chan struct{}
	waiting int64
}

func newCond(l sync.Locker) *cond {
	// Add NumCPU slots to reduce chance of blocking the Signal routine and avoid having to release the lock.
	return &cond{L: l, ch: make(chan struct{}, runtime.NumCPU()), waiting: 0}
}

// Signal wakes one goroutine waiting on c, if there is any.
// It requires for the caller to hold c.L during the call.
func (c *cond) Signal() {
	if c.waiting == 0 {
		return
	}
	c.waiting--
	select {
	case c.ch <- struct{}{}:
		// Happy path, no need to unlock the lock.
	default:
		// All signal slots are occupied and canceled requests are potentially blocked waiting for the lock
		// before they can free slots. Need to release the lock to allow them to free slots.
		c.L.Unlock()
		c.ch <- struct{}{}
		c.L.Lock()
	}
}

// Wait atomically unlocks c.L and suspends execution of the calling goroutine. After later resuming execution, Wait locks c.L before returning.
func (c *cond) Wait(ctx context.Context) error {
	c.waiting++
	c.L.Unlock()
	select {
	case <-ctx.Done():
		c.L.Lock()
		if c.waiting == 0 {
			// If waiting is 0, it means that there was a signal sent and nobody else waits for it.
			// Consume it, so that we don't unblock other consumer unnecessary,
			// or we don't block the producer because the channel buffer is full.
			<-c.ch
		} else {
			// Decrease the number of waiting routines.
			c.waiting--
		}
		return ctx.Err()
	case <-c.ch:
		c.L.Lock()
		return nil
	}
}
