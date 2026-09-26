// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/hosttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/storagetest"
)

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
