// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCondSignalWakesExactlyOneWaiter verifies a normal Signal wakes exactly one
// waiter, that Wait returns nil, and that it returns with the lock held.
func TestCondSignalWakesExactlyOneWaiter(t *testing.T) {
	mu := &sync.Mutex{}
	c := newCond(mu)

	const numWaiters = 5
	var woken atomic.Int64
	var wg sync.WaitGroup
	wg.Add(numWaiters)
	for i := 0; i < numWaiters; i++ {
		go func() {
			defer wg.Done()
			mu.Lock()
			err := c.Wait(context.Background())
			assert.NoError(t, err)
			woken.Add(1)
			// Wait must return with the lock held; if not, this Unlock panics.
			mu.Unlock()
		}()
	}

	// Wait for all goroutines to register as waiters.
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(c.waiters) == numWaiters
	}, time.Second, time.Millisecond)

	mu.Lock()
	c.Signal()
	mu.Unlock()

	// Exactly one waiter should wake; the rest stay blocked.
	require.Eventually(t, func() bool { return woken.Load() == 1 }, time.Second, time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	require.EqualValues(t, 1, woken.Load())

	// Wake the rest so the goroutines exit cleanly.
	mu.Lock()
	c.Broadcast()
	mu.Unlock()
	wg.Wait()
	assert.EqualValues(t, numWaiters, woken.Load())
}

// TestCondBroadcastWakesAllWaiters verifies Broadcast wakes N>1 concurrent waiters.
func TestCondBroadcastWakesAllWaiters(t *testing.T) {
	mu := &sync.Mutex{}
	c := newCond(mu)

	const numWaiters = 50
	var woken atomic.Int64
	var wg sync.WaitGroup
	wg.Add(numWaiters)
	for i := 0; i < numWaiters; i++ {
		go func() {
			defer wg.Done()
			mu.Lock()
			err := c.Wait(context.Background())
			assert.NoError(t, err)
			mu.Unlock()
			woken.Add(1)
		}()
	}

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(c.waiters) == numWaiters
	}, time.Second, time.Millisecond)

	mu.Lock()
	c.Broadcast()
	mu.Unlock()

	wg.Wait()
	assert.EqualValues(t, numWaiters, woken.Load())

	mu.Lock()
	assert.Empty(t, c.waiters)
	mu.Unlock()
}

// TestCondCancelledWaiterIsRemoved verifies a waiter whose context is cancelled
// returns ctx.Err() and is removed from the waiter set (no leak).
func TestCondCancelledWaiterIsRemoved(t *testing.T) {
	mu := &sync.Mutex{}
	c := newCond(mu)

	ctx, cancel := context.WithCancel(context.Background())

	var err error
	done := make(chan struct{})
	go func() {
		mu.Lock()
		err = c.Wait(ctx)
		mu.Unlock()
		close(done)
	}()

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(c.waiters) == 1
	}, time.Second, time.Millisecond)

	cancel()
	<-done

	assert.ErrorIs(t, err, context.Canceled)
	mu.Lock()
	assert.Empty(t, c.waiters, "cancelled waiter must be removed")
	mu.Unlock()
}

// TestCondSignalRacesCancel exercises the race where a Signal and ctx
// cancellation fire near-simultaneously for the same waiter. The Signal must
// never block under the lock, the waiter set must not leak, and the waiter must
// observe either nil (woken) or context.Canceled.
func TestCondSignalRacesCancel(t *testing.T) {
	for iter := 0; iter < 200; iter++ {
		mu := &sync.Mutex{}
		c := newCond(mu)
		ctx, cancel := context.WithCancel(context.Background())

		var err error
		done := make(chan struct{})
		go func() {
			mu.Lock()
			err = c.Wait(ctx)
			mu.Unlock()
			close(done)
		}()

		require.Eventually(t, func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(c.waiters) == 1
		}, time.Second, time.Millisecond)

		// Fire Signal and cancel concurrently to hit the race window.
		var raceWG sync.WaitGroup
		raceWG.Add(2)
		go func() {
			defer raceWG.Done()
			mu.Lock()
			c.Signal()
			mu.Unlock()
		}()
		go func() {
			defer raceWG.Done()
			cancel()
		}()
		raceWG.Wait()

		<-done
		if err != nil {
			assert.ErrorIs(t, err, context.Canceled)
		}
		mu.Lock()
		assert.Empty(t, c.waiters, "no waiter should leak after signal/cancel race")
		mu.Unlock()
	}
}

// TestCondNoDeadlockUnderLoad reproduces the original deadlock class: many
// Signal/Broadcast calls (all made while holding the lock) interleaved with
// Wait(ctx) calls that frequently cancel via context. Against the old
// buffered-channel implementation, Signal/Broadcast would block forever under
// the lock and this test would hang (caught by the deadline guard). With the
// per-waiter-channel fix it completes promptly.
func TestCondNoDeadlockUnderLoad(t *testing.T) {
	deadline := time.Now().Add(20 * time.Second)
	if d, ok := t.Deadline(); ok && d.Before(deadline) {
		deadline = d.Add(-2 * time.Second)
	}

	mu := &sync.Mutex{}
	c := newCond(mu)

	stop := make(chan struct{})
	var wg sync.WaitGroup

	// Waiters: repeatedly Wait with a short, frequently-expiring context.
	const numWaiters = 16
	for i := 0; i < numWaiters; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
				mu.Lock()
				_ = c.Wait(ctx)
				mu.Unlock()
				cancel()
			}
		}()
	}

	// Signalers: hammer Signal/Broadcast while holding the lock.
	const numSignalers = 8
	for i := 0; i < numSignalers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				mu.Lock()
				if id%2 == 0 {
					c.Signal()
				} else {
					c.Broadcast()
				}
				mu.Unlock()
			}
		}(i)
	}

	// Run for a fixed window, then signal stop. The deadline guard catches a hang.
	runFor := time.Until(deadline)
	if runFor > 3*time.Second {
		runFor = 3 * time.Second
	}
	time.Sleep(runFor)
	close(stop)

	finished := make(chan struct{})
	go func() {
		wg.Wait()
		close(finished)
	}()

	select {
	case <-finished:
		// All goroutines exited: no Signal/Broadcast/Wait deadlocked under the lock.
	case <-time.After(time.Until(deadline)):
		// A goroutine is stuck (the original-bug behavior).
		buf := make([]byte, 1<<20)
		n := runtime.Stack(buf, true)
		t.Fatalf("deadlock: goroutines did not finish before deadline\n%s", buf[:n])
	}

	mu.Lock()
	// Any remaining waiters are in-flight Waits that will drain; no shared
	// channel to desync, so this is just a sanity bound.
	assert.LessOrEqual(t, len(c.waiters), numWaiters)
	mu.Unlock()
}
