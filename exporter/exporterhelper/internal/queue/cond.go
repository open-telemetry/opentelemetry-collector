// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"
	"sync"
)

// cond is equivalent with sync.Cond, but context.Context aware.
// Which means Wait() will return if context is done before any signal is received.
// Also, it requires the caller to hold the c.L during all calls.
type cond struct {
	L       sync.Locker
	waiters []chan struct{}
}

func newCond(l sync.Locker) *cond {
	return &cond{L: l}
}

// Signal wakes one goroutine waiting on c, if there is any.
// It requires for the caller to hold c.L during the call.
func (c *cond) Signal() {
	if len(c.waiters) == 0 {
		return
	}
	// Each waiter owns its own channel, so closing it wakes exactly that waiter
	// and, unlike a send on a shared buffered channel, close() can never block.
	// The previous implementation sent on a size-1 buffered channel while holding
	// c.L, which deadlocked forever once the buffer filled under load.
	w := c.waiters[0]
	c.waiters[0] = nil
	c.waiters = c.waiters[1:]
	close(w)
}

// Broadcast wakes all goroutines waiting on c.
// It requires for the caller to hold c.L during the call.
func (c *cond) Broadcast() {
	for i, w := range c.waiters {
		c.waiters[i] = nil
		close(w)
	}
	c.waiters = nil
}

// Wait atomically unlocks c.L and suspends execution of the calling goroutine. After later resuming execution, Wait locks c.L before returning.
func (c *cond) Wait(ctx context.Context) error {
	w := make(chan struct{})
	c.waiters = append(c.waiters, w)
	c.L.Unlock()
	select {
	case <-ctx.Done():
		c.L.Lock()
		c.removeWaiter(w)
		return ctx.Err()
	case <-w:
		c.L.Lock()
		return nil
	}
}

// removeWaiter removes w from the waiters slice if it is still present. A
// concurrent Signal/Broadcast may have already popped and closed it, in which
// case there is nothing to remove. It requires for the caller to hold c.L.
func (c *cond) removeWaiter(w chan struct{}) {
	for i, x := range c.waiters {
		if x == w {
			c.waiters[i] = nil
			c.waiters = append(c.waiters[:i], c.waiters[i+1:]...)
			return
		}
	}
}
