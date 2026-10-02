// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

func regressionCoordinator(t *testing.T, workers int) (*orderedLogsCoordinator, chan orderedDispatchResult) {
	t.Helper()
	dispatched := make(chan orderedDispatchResult, 16)
	c := newOrderedLogsCoordinator(orderedTestSettings(), func(_ context.Context, d OrderedLogsDispatch, done OrderedLogsCompletion) error {
		dispatched <- orderedDispatchResult{d, done}
		return nil
	}, configretry.BackOffConfig{}, 0, workers, nil)
	t.Cleanup(c.shutdown)
	return c, dispatched
}

func TestOrderedTerminalFailureSchedulesOtherPartitionsAndFencesCallbacks(t *testing.T) {
	c, dispatched := regressionCoordinator(t, 1)
	a, successor, other := orderedTestGroup("p", OrderedPositionContinue), orderedTestGroup("p", OrderedPositionContinue), orderedTestGroup("q", OrderedPositionEnd)
	for _, g := range []*orderedLogsGroup{a, successor, other} {
		g.SetQueueCompletion(func(error) {})
	}
	require.NoError(t, c.add(context.Background(), a))
	first := receiveOrderedDispatch(t, dispatched)
	first.completion.Release()
	require.NoError(t, c.add(context.Background(), successor))
	late := receiveOrderedDispatch(t, dispatched)
	require.NoError(t, c.add(context.Background(), other))
	first.completion.Fail(errors.New("writer failed"))
	ready := receiveOrderedDispatch(t, dispatched)
	require.Equal(t, "q", ready.dispatch.PartitionKey)
	require.Error(t, first.dispatch.StreamContext.Err())
	late.completion.Release()
	late.completion.Succeed()
	c.mu.Lock()
	require.Zero(t, c.releasedRequests)
	require.Zero(t, c.releasedBytes)
	require.Equal(t, 1, c.activeWrites)
	c.mu.Unlock()
	ready.completion.Succeed()
	again := orderedTestGroup("p", OrderedPositionEnd)
	again.SetQueueCompletion(func(error) {})
	require.NoError(t, c.add(context.Background(), again))
	fresh := receiveOrderedDispatch(t, dispatched)
	require.Greater(t, fresh.dispatch.StreamAttempt, first.dispatch.StreamAttempt)
	fresh.completion.Succeed()
}

func TestOrderedRetryFencesStaleCompletionOnReusedTask(t *testing.T) {
	callbacks := []struct {
		name string
		call func(OrderedLogsCompletion)
	}{
		{name: "release", call: func(done OrderedLogsCompletion) { done.Release() }},
		{name: "succeed", call: func(done OrderedLogsCompletion) { done.Succeed() }},
		{name: "fail", call: func(done OrderedLogsCompletion) { done.Fail(errors.New("late failure")) }},
	}
	for _, workers := range []int{1, 32, 500} {
		for _, callback := range callbacks {
			t.Run(strconv.Itoa(workers)+"/"+callback.name, func(t *testing.T) {
				dispatched := make(chan orderedDispatchResult, 16)
				retry := configretry.BackOffConfig{Enabled: true, InitialInterval: time.Millisecond, Multiplier: 1, MaxInterval: time.Millisecond}
				c := newOrderedLogsCoordinator(orderedTestSettings(), func(_ context.Context, d OrderedLogsDispatch, done OrderedLogsCompletion) error {
					dispatched <- orderedDispatchResult{d, done}
					return nil
				}, retry, 0, workers, nil)
				t.Cleanup(c.shutdown)

				groups := []*orderedLogsGroup{
					orderedTestGroup("p", OrderedPositionContinue),
					orderedTestGroup("p", OrderedPositionContinue),
					orderedTestGroup("p", OrderedPositionEnd),
				}
				completed := make([]chan error, len(groups))
				for i, group := range groups {
					done := make(chan error, 2)
					completed[i] = done
					require.True(t, group.SetQueueCompletion(func(err error) { done <- err }))
					require.NoError(t, c.add(context.Background(), group))
				}
				assertPending := func(done <-chan error) {
					t.Helper()
					select {
					case err := <-done:
						t.Fatalf("queue ownership retired prematurely: %v", err)
					default:
					}
				}
				assertCompleted := func(done <-chan error) {
					t.Helper()
					select {
					case err := <-done:
						require.NoError(t, err)
					case <-time.After(time.Second):
						t.Fatal("timed out waiting for queue retirement")
					}
				}

				first := receiveOrderedDispatch(t, dispatched)
				first.completion.Release()
				stale := receiveOrderedDispatch(t, dispatched)
				// Fail a different request so the stale handle's outcomeOnce and
				// releaseOnce remain unused. Retry resets these same task pointers.
				first.completion.Fail(errors.New("retry the unretired prefix"))
				replayedFirst := receiveOrderedDispatch(t, dispatched)
				require.Equal(t, groups[0].children[0].QueueID, replayedFirst.dispatch.QueueID)
				require.Greater(t, replayedFirst.dispatch.StreamAttempt, first.dispatch.StreamAttempt)
				require.Error(t, stale.dispatch.StreamContext.Err())
				replayedFirst.completion.Release()
				replayedSecond := receiveOrderedDispatch(t, dispatched)
				require.Equal(t, groups[1].children[0].QueueID, replayedSecond.dispatch.QueueID)
				oldHandle := stale.completion.(*orderedLogsCompletion)
				newHandle := replayedSecond.completion.(*orderedLogsCompletion)
				require.Same(t, oldHandle.partition, newHandle.partition)
				require.Same(t, oldHandle.task, newHandle.task)
				require.Greater(t, newHandle.epoch, oldHandle.epoch)

				callback.call(stale.completion)
				func() {
					c.mu.Lock()
					defer c.mu.Unlock()
					p := newHandle.partition
					require.Equal(t, newHandle.epoch, p.epoch, "stale failure must not start another retry")
					require.False(t, p.retrying)
					require.True(t, p.active)
					require.Same(t, newHandle.task, p.activeTask)
					require.True(t, newHandle.task.dispatched)
					require.False(t, newHandle.task.released, "stale release must not release the new write")
					require.False(t, newHandle.task.committed, "stale success must not commit the new request")
					require.Len(t, p.tasks, 3)
					require.False(t, p.tasks[2].dispatched)
					require.Equal(t, 3, c.staged)
					require.Equal(t, 1, c.activeWrites)
					require.Equal(t, 1, c.releasedRequests)
					require.Equal(t, replayedFirst.completion.(*orderedLogsCompletion).task.bytes, c.releasedBytes)
				}()
				require.NoError(t, replayedSecond.dispatch.StreamContext.Err())
				for _, done := range completed {
					assertPending(done)
				}

				// Committing the first replay must still leave the second owned by
				// the queue; only the new handle may release and retire it.
				replayedFirst.completion.Succeed()
				assertCompleted(completed[0])
				assertPending(completed[1])
				assertPending(completed[2])
				replayedSecond.completion.Release()
				last := receiveOrderedDispatch(t, dispatched)
				require.Equal(t, groups[2].children[0].QueueID, last.dispatch.QueueID)
				require.Equal(t, replayedSecond.dispatch.StreamAttempt, last.dispatch.StreamAttempt)
				replayedSecond.completion.Succeed()
				assertCompleted(completed[1])
				last.completion.Succeed()
				assertCompleted(completed[2])
				for _, done := range completed {
					assertPending(done) // Queue completion must fire exactly once.
				}
				select {
				case extra := <-dispatched:
					t.Fatalf("unexpected dispatch after stale callback: %+v", extra.dispatch)
				default:
				}
			})
		}
	}
}

func TestOrderedAtomicGroupRequestSizerChargesChildren(t *testing.T) {
	a, b := orderedTestGroup("p", OrderedPositionContinue), orderedTestGroup("q", OrderedPositionEnd)
	g := &orderedLogsGroup{children: append(a.children, b.children...)}
	require.Equal(t, int64(2), (request.RequestsSizer{}).Sizeof(g))
}
