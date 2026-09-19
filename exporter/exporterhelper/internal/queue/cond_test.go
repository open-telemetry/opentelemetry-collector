// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCondSignalDoesNotBlockBehindCanceledWaiters(t *testing.T) {
	t.Parallel()
	locker := &observedLocker{lockAttempts: make(chan struct{}, 2)}
	c := newCond(locker)

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	waiter1 := startCondWaiter(ctx1, c)
	waiter2 := startCondWaiter(ctx2, c)

	locked := make(chan struct{})
	notify := make(chan struct{})
	notified := make(chan struct{})
	go func() {
		locker.Lock()
		close(locked)
		<-notify
		c.Signal()
		c.Signal()
		locker.Unlock()
		close(notified)
	}()

	<-locked
	locker.observe.Store(true)
	cancel1()
	cancel2()
	<-locker.lockAttempts
	<-locker.lockAttempts
	locker.observe.Store(false)
	close(notify)

	select {
	case <-notified:
	case <-time.After(time.Second):
		t.Fatal("Signal blocked while holding the condition lock")
	}

	for _, waiter := range []<-chan error{waiter1, waiter2} {
		select {
		case err := <-waiter:
			require.ErrorIs(t, err, context.Canceled)
		case <-time.After(time.Second):
			t.Fatal("condition waiter did not return")
		}
	}
}

type observedLocker struct {
	mu           sync.Mutex
	observe      atomic.Bool
	lockAttempts chan struct{}
}

func (l *observedLocker) Lock() {
	if l.observe.Load() {
		l.lockAttempts <- struct{}{}
	}
	l.mu.Lock()
}

func (l *observedLocker) Unlock() {
	l.mu.Unlock()
}

func startCondWaiter(ctx context.Context, c *cond) <-chan error {
	l := c.L
	entered := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		l.Lock()
		close(entered)
		done <- c.Wait(ctx)
		l.Unlock()
	}()

	<-entered
	l.Lock()
	defer l.Unlock()
	return done
}

func TestCondSignalAccounting(t *testing.T) {
	t.Parallel()
	for _, signals := range []int{1, 3} {
		t.Run(strconv.Itoa(signals), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				c := newCond(&sync.Mutex{})
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				waiters := make([]<-chan error, 3)
				for i := range waiters {
					waiters[i] = startCondWaiter(ctx, c)
				}
				c.L.Lock()
				for range signals {
					c.Signal()
				}
				c.L.Unlock()
				synctest.Wait()
				woken := 0
				var pending []<-chan error
				for _, waiter := range waiters {
					select {
					case err := <-waiter:
						require.NoError(t, err)
						woken++
					default:
						pending = append(pending, waiter)
					}
				}
				require.Equal(t, signals, woken)
				cancel()
				for _, waiter := range pending {
					require.ErrorIs(t, <-waiter, context.Canceled)
				}
				checkCondReuse(t, c)
			})
		})
	}
}

func TestCondCancellationNotification(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name        string
		broadcast   bool
		notifyFirst bool
		interleaved bool
	}{
		{name: "cancel_then_signal"},
		{name: "signal_then_cancel", notifyFirst: true},
		{name: "mixed_cancel_then_signal", interleaved: true},
		{name: "cancel_then_broadcast", broadcast: true},
		{name: "broadcast_then_cancel", broadcast: true, notifyFirst: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				locker := &observedLocker{lockAttempts: make(chan struct{}, 6)}
				c := newCond(locker)
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				var cancelled, live []<-chan error
				for range 3 {
					cancelled = append(cancelled, startCondWaiter(ctx, c))
					if tt.interleaved {
						live = append(live, startCondWaiter(t.Context(), c))
					}
				}
				if !tt.interleaved {
					for range 3 {
						live = append(live, startCondWaiter(t.Context(), c))
					}
				}
				notify := func() {
					if tt.broadcast {
						c.Broadcast()
					} else {
						for range 3 {
							c.Signal()
						}
					}
				}
				c.L.Lock()
				locker.observe.Store(true)
				if tt.notifyFirst {
					notify()
				} else {
					cancel()
				}
				attempts := 3
				if tt.notifyFirst && tt.broadcast {
					attempts = 6
				}
				// With L held, these attempts prove which select branch won.
				for range attempts {
					<-locker.lockAttempts
				}
				locker.observe.Store(false)
				if tt.notifyFirst {
					cancel()
				} else {
					notify()
				}
				c.L.Unlock()
				for _, waiter := range cancelled {
					if tt.notifyFirst {
						require.NoError(t, <-waiter)
					} else {
						require.ErrorIs(t, <-waiter, context.Canceled)
					}
				}
				if tt.notifyFirst && !tt.broadcast {
					// These signals were consumed, so must not also wake live waiters.
					synctest.Wait()
					for _, waiter := range live {
						select {
						case <-waiter:
							t.Fatal("notification was duplicated")
						default:
						}
					}
					c.L.Lock()
					notify()
					c.L.Unlock()
				}
				for _, waiter := range live {
					require.NoError(t, <-waiter)
				}
				checkCondReuse(t, c)
			})
		})
	}
}

func TestCondBroadcastDoesNotWakeLaterWaiter(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		locker := &observedLocker{lockAttempts: make(chan struct{}, 1)}
		c := newCond(locker)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		cancelled := startCondWaiter(ctx, c)
		c.L.Lock()
		locker.observe.Store(true)
		cancel()
		<-locker.lockAttempts
		locker.observe.Store(false)
		c.Broadcast()
		// Register after Broadcast, before the cancelled waiter can retire.
		// Wait releases L and lets cancellation finish; no notification is due
		// to this waiter until the explicit Signal below.
		later := make(chan error, 1)
		go func() { later <- c.Wait(t.Context()); c.L.Unlock() }()
		require.ErrorIs(t, <-cancelled, context.Canceled)
		synctest.Wait()
		select {
		case <-later:
			t.Fatal("broadcast reached a waiter registered after the broadcast")
		default:
		}
		c.L.Lock()
		c.Signal()
		c.L.Unlock()
		require.NoError(t, <-later)
		checkCondReuse(t, c)
	})
}

// checkCondReuse verifies empty notifications leave no token for the next waiter,
// then verifies that the same condition still supports a fresh broadcast.
func checkCondReuse(t *testing.T, c *cond) {
	t.Helper()
	c.L.Lock()
	require.Zero(t, c.waiters.Len())
	c.Signal()
	c.Signal()
	c.Broadcast()
	c.L.Unlock()
	waiter := startCondWaiter(t.Context(), c)
	synctest.Wait()
	select {
	case <-waiter:
		t.Fatal("waiter consumed a stale notification")
	default:
	}
	c.L.Lock()
	c.Broadcast()
	c.L.Unlock()
	require.NoError(t, <-waiter)
	c.L.Lock()
	require.Zero(t, c.waiters.Len())
	c.L.Unlock()
}
