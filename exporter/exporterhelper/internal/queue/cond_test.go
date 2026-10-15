// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/hosttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/storagetest"
)

func TestCondSignalDoesNotBlockBehindCanceledWaiters(t *testing.T) {
	t.Parallel()
	locker := newObservedLocker(2)
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

// observedLocker is a channel based lock, so goroutines blocked on it are durably
// blocked in a synctest bubble and a deadlock is reported instead of hanging.
type observedLocker struct {
	sem          chan struct{}
	observe      atomic.Bool
	lockAttempts chan struct{}
}

func newObservedLocker(attempts int) *observedLocker {
	return &observedLocker{sem: make(chan struct{}, 1), lockAttempts: make(chan struct{}, attempts)}
}

func (l *observedLocker) Lock() {
	if l.observe.Load() {
		l.lockAttempts <- struct{}{}
	}
	l.sem <- struct{}{}
}

func (l *observedLocker) Unlock() {
	<-l.sem
}

func startCondWaiter(ctx context.Context, c *cond) <-chan error {
	entered := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		c.L.Lock()
		defer c.L.Unlock()
		close(entered)
		done <- c.Wait(ctx)
	}()

	<-entered
	c.L.Lock()
	defer c.L.Unlock()
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
		cancelled   int
		live        int
		notifyFirst bool
		interleaved bool
	}{
		{name: "cancel_then_signal", cancelled: 3, live: 3},
		{name: "signal_then_cancel", cancelled: 3, live: 3, notifyFirst: true},
		{name: "mixed_cancel_then_signal", cancelled: 3, live: 3, interleaved: true},
		// No live waiter is parked on the channel, so every signal must be buffered
		// while the cancelled waiters are blocked on L.
		{name: "cancel_then_signal_without_live_waiters", cancelled: 3},
		// More signals than the channel buffer while the cancelled waiters are blocked on L.
		{name: "cancel_then_signal_overflowing_buffer", cancelled: runtime.NumCPU() + 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				locker := newObservedLocker(tt.cancelled)
				c := newCond(locker)
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				var cancelled, live []<-chan error
				for i := range tt.cancelled {
					cancelled = append(cancelled, startCondWaiter(ctx, c))
					if tt.interleaved && i < tt.live {
						live = append(live, startCondWaiter(t.Context(), c))
					}
				}
				if !tt.interleaved {
					for range tt.live {
						live = append(live, startCondWaiter(t.Context(), c))
					}
				}
				notify := func(n int) {
					for range n {
						c.Signal()
					}
				}
				c.L.Lock()
				locker.observe.Store(true)
				if tt.notifyFirst {
					notify(tt.cancelled)
				} else {
					cancel()
				}
				// With L held, these attempts prove which select branch won.
				for range tt.cancelled {
					<-locker.lockAttempts
				}
				locker.observe.Store(false)
				if tt.notifyFirst {
					cancel()
				} else {
					notify(tt.cancelled)
				}
				c.L.Unlock()
				for _, waiter := range cancelled {
					if tt.notifyFirst {
						require.NoError(t, <-waiter)
					} else {
						require.ErrorIs(t, <-waiter, context.Canceled)
					}
				}
				if tt.notifyFirst {
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
					notify(len(live))
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

// checkCondReuse verifies empty notifications leave no token for the next waiter,
// then verifies that the same condition still supports a fresh broadcast.
func checkCondReuse(t *testing.T, c *cond) {
	t.Helper()
	c.L.Lock()
	require.Zero(t, c.waiting)
	c.Signal()
	c.Signal()
	c.L.Unlock()
	waiter := startCondWaiter(t.Context(), c)
	synctest.Wait()
	select {
	case <-waiter:
		t.Fatal("waiter consumed a stale notification")
	default:
	}
	c.L.Lock()
	c.Signal()
	c.L.Unlock()
	require.NoError(t, <-waiter)
	c.L.Lock()
	require.Zero(t, c.waiting)
	c.L.Unlock()
}

func TestBlockedOfferCancellationDoesNotConsumeCapacityNotification(t *testing.T) {
	tests := []struct {
		name string
		new  func() (readableQueue[intRequest], *cond, sync.Locker, component.Host)
	}{
		{
			name: "memory",
			new: func() (readableQueue[intRequest], *cond, sync.Locker, component.Host) {
				set := newSettings(request.SizerTypeRequests, 2)
				set.BlockOnOverflow = true
				q := newMemoryQueue[intRequest](set).(*memoryQueue[intRequest])
				return q, q.hasMoreSpace, &q.mu, componenttest.NewNopHost()
			},
		},
		{
			name: "persistent",
			new: func() (readableQueue[intRequest], *cond, sync.Locker, component.Host) {
				set := newSettingsWithStorage(request.SizerTypeRequests, 2)
				set.BlockOnOverflow = true
				q := newPersistentQueue[intRequest](set).(*persistentQueue[intRequest])
				host := hosttest.NewHost(map[component.ID]component.Component{
					{}: storagetest.NewMockStorageExtension(nil),
				})
				return q, q.hasMoreSpace, &q.mu, host
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				q, hasMoreSpace, queueLock, host := tt.new()
				lockBarrier := newLockAttemptBarrier(queueLock, 3)
				hasMoreSpace.L = lockBarrier
				require.NoError(t, q.Start(context.Background(), host))

				require.NoError(t, q.Offer(context.Background(), 1))
				require.NoError(t, q.Offer(context.Background(), 2))
				_, _, firstDone, ok := q.Read(context.Background())
				require.True(t, ok)
				_, _, secondDone, ok := q.Read(context.Background())
				require.True(t, ok)

				firstCtx, cancelFirst := context.WithCancel(context.Background())
				secondCtx, cancelSecond := context.WithCancel(context.Background())

				firstResult := offerAsync(firstCtx, q, 3)
				synctest.Wait()
				secondResult := offerAsync(secondCtx, q, 4)
				synctest.Wait()
				firstLiveResult := offerAsync(context.Background(), q, 5)
				synctest.Wait()
				secondLiveResult := offerAsync(context.Background(), q, 6)
				synctest.Wait()

				lockBarrier.block.Store(true)
				cancelFirst()
				cancelSecond()
				<-lockBarrier.attempts
				<-lockBarrier.attempts

				completionReturned := make(chan struct{})
				go func() {
					firstDone.OnDone(nil)
					secondDone.OnDone(nil)
					close(completionReturned)
				}()

				synctest.Wait()
				assertClosed(t, completionReturned)
				lockBarrier.block.Store(false)
				close(lockBarrier.release)
				synctest.Wait()
				require.ErrorIs(t, <-firstResult, context.Canceled)
				require.ErrorIs(t, <-secondResult, context.Canceled)
				require.NoError(t, <-firstLiveResult)
				require.NoError(t, <-secondLiveResult)

				assert.EqualValues(t, 2, q.Size())
				consumed := make([]intRequest, 0, 2)
				for range 2 {
					_, req, done, readOK := q.Read(context.Background())
					require.True(t, readOK)
					consumed = append(consumed, req)
					done.OnDone(nil)
				}
				assert.ElementsMatch(t, []intRequest{5, 6}, consumed)
				assert.EqualValues(t, 0, q.Size())

				require.NoError(t, q.Shutdown(context.Background()))
				_, _, _, ok = q.Read(context.Background())
				assert.False(t, ok)
			})
		})
	}
}

type lockAttemptBarrier struct {
	sync.Locker
	block    atomic.Bool
	attempts chan struct{}
	release  chan struct{}
}

func newLockAttemptBarrier(locker sync.Locker, attempts int) *lockAttemptBarrier {
	return &lockAttemptBarrier{
		Locker:   locker,
		attempts: make(chan struct{}, attempts),
		release:  make(chan struct{}),
	}
}

func (b *lockAttemptBarrier) Lock() {
	if b.block.Load() {
		b.attempts <- struct{}{}
		<-b.release
	}
	b.Locker.Lock()
}

func offerAsync(ctx context.Context, q readableQueue[intRequest], req intRequest) <-chan error {
	result := make(chan error, 1)
	go func() {
		result <- q.Offer(ctx, req)
	}()
	return result
}

func assertClosed(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	default:
		t.Fatal("completion callbacks did not return")
	}
}
