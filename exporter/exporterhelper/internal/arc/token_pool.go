// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package arc // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/arc"

import (
	"context"
	"errors"
	"sync"
)

// TokenPool is a fair gate for request concurrency:
// - cap: maximum in-flight
// - inUse: current in-flight
// Acquire blocks while inUse >= cap. Shrinking is done by reducing cap; no
// "debt" bookkeeping or forgetting permits. This deliberately differs from a
// shrinkable semaphore that accrues forgets.

type TokenPool struct {
	mu    sync.Mutex
	cond  *sync.Cond
	cap   int
	inUse int
	dead  bool
}

func newTokenPool(initial int) *TokenPool {
	p := &TokenPool{cap: initial}
	p.cond = sync.NewCond(&p.mu)
	return p
}

func (p *TokenPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.dead = true
	p.cond.Broadcast()
}

func (p *TokenPool) Acquire(ctx context.Context) error {
	// Fast fail if already canceled.
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.dead {
		return errors.New("token pool closed")
	}

	// Wake on context cancellation; ensure goroutine exits via close(done).
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			p.cond.Broadcast()
		case <-done:
		}
	}()
	defer close(done)

	for !p.dead && p.inUse >= p.cap {
		p.cond.Wait()

		// Re-check cancellation and pool state after each wake.
		if ctx != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		if p.dead {
			return errors.New("token pool closed")
		}
	}

	if p.dead {
		return errors.New("token pool closed")
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}

	p.inUse++
	return nil
}

func (p *TokenPool) Release() {
	p.mu.Lock()
	p.inUse--
	if p.inUse < 0 {
		p.inUse = 0
	}
	p.mu.Unlock()
	p.cond.Signal()
}

func (p *TokenPool) Grow(n int) {
	if n <= 0 {
		return
	}
	p.mu.Lock()
	p.cap += n
	p.mu.Unlock()
	p.cond.Broadcast()
}

func (p *TokenPool) Shrink(n int) {
	if n <= 0 {
		return
	}
	p.mu.Lock()
	p.cap -= n
	if p.cap < 1 {
		p.cap = 1
	}
	p.mu.Unlock()
}
