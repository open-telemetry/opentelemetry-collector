// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCondSignalDoesNotBlockBehindCanceledWaiters(t *testing.T) {
	locker := &observedLocker{lockAttempts: make(chan struct{}, 2)}
	c := newCond(locker)

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	waiter1 := startCondWaiter(locker, c, ctx1)
	waiter2 := startCondWaiter(locker, c, ctx2)

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
		case <-waiter:
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

func startCondWaiter(l sync.Locker, c *cond, ctx context.Context) <-chan error {
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
	l.Unlock()
	return done
}
