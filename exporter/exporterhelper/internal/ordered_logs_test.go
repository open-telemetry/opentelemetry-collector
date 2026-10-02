// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"encoding/binary"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"
	"go.opentelemetry.io/collector/pdata/plog"
)

type orderedDispatchResult struct {
	dispatch   OrderedLogsDispatch
	completion OrderedLogsCompletion
}

var orderedTestQueueID atomic.Uint64

type orderedTestCheckpointStore struct {
	mu       sync.Mutex
	data     map[string][]byte
	loadErr  error
	failSave int
}

func (s *orderedTestCheckpointStore) LoadCheckpoint(_ context.Context, key string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loadErr != nil {
		return nil, false, s.loadErr
	}
	value, ok := s.data[key]
	return append([]byte(nil), value...), ok, nil
}

func (s *orderedTestCheckpointStore) SaveCheckpoint(_ context.Context, key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failSave > 0 {
		s.failSave--
		return errors.New("temporary checkpoint storage failure")
	}
	if s.data == nil {
		s.data = make(map[string][]byte)
	}
	s.data[key] = append([]byte(nil), value...)
	return nil
}

func orderedTestLogs() plog.Logs {
	ld := plog.NewLogs()
	ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	return ld
}

func orderedTestGroup(key string, position OrderedPosition) *orderedLogsGroup {
	ld := orderedTestLogs()
	var queueID [16]byte
	queueID[0] = 1
	binary.BigEndian.PutUint64(queueID[8:], orderedTestQueueID.Add(1))
	return &orderedLogsGroup{
		children: []OrderedLogsDescriptor{{Request: ld, PartitionKey: key, Position: position, QueueID: queueID}},
		items:    ld.LogRecordCount(),
		bytes:    orderedLogsWire.LogsSize(ld),
	}
}

func orderedTestSettings() OrderedLogsSettings {
	return OrderedLogsSettings{
		MaxStaged: 16, MaxActivePartitions: 4, MaxReleasedRequests: 16,
		MaxReleasedBytes: 1 << 20, MaxRecoveryTailBytes: 1 << 20,
		MaxGroupRequests: 8, MaxGroupItems: 16, MaxGroupBytes: 1 << 20,
		MaxPartitionKeyBytes: 128,
	}
}

func receiveOrderedDispatch(t *testing.T, ch <-chan orderedDispatchResult) orderedDispatchResult {
	t.Helper()
	select {
	case result := <-ch:
		return result
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ordered dispatch")
		return orderedDispatchResult{}
	}
}

func TestOrderedCoordinatorTwoItemsReleaseBeforeRetirement(t *testing.T) {
	for _, consumerCount := range []int{1, 32} {
		t.Run(strconv.Itoa(consumerCount), func(t *testing.T) {
			dispatched := make(chan orderedDispatchResult, 4)
			settings := orderedTestSettings()
			settings.MaxStaged = 1
			settings.MaxGroupRequests = 1
			coordinator := newOrderedLogsCoordinator(settings, func(_ context.Context, dispatch OrderedLogsDispatch, completion OrderedLogsCompletion) error {
				dispatched <- orderedDispatchResult{dispatch: dispatch, completion: completion}
				return nil
			}, configretry.BackOffConfig{}, 0, consumerCount, nil)
			t.Cleanup(coordinator.shutdown)

			first, second := orderedTestGroup("channel-a", OrderedPositionContinue), orderedTestGroup("channel-a", OrderedPositionContinue)
			firstDone, secondDone := make(chan error, 1), make(chan error, 1)
			require.True(t, first.SetQueueCompletion(func(err error) { firstDone <- err }))
			require.True(t, second.SetQueueCompletion(func(err error) { secondDone <- err }))
			require.NoError(t, coordinator.add(context.Background(), first))
			firstDispatch := receiveOrderedDispatch(t, dispatched)
			require.Equal(t, "channel-a", firstDispatch.dispatch.PartitionKey)

			// This is the two-fragment cycle: the first fragment has no complete
			// record at the receiver, so it cannot retire until its successor is
			// written. Releasing write order must make that successor runnable.
			require.NoError(t, coordinator.add(context.Background(), second))
			firstDispatch.completion.Release()
			secondDispatch := receiveOrderedDispatch(t, dispatched)
			require.Equal(t, "channel-a", secondDispatch.dispatch.PartitionKey)

			// ACKs can arrive out of order. The later item is retained until the
			// earlier queue ordinal commits.
			secondDispatch.completion.Succeed()
			select {
			case err := <-secondDone:
				t.Fatalf("second item retired past unresolved first item: %v", err)
			case <-time.After(20 * time.Millisecond):
			}
			firstDispatch.completion.Succeed()
			require.NoError(t, <-firstDone)
			require.NoError(t, <-secondDone)
		})
	}
}

func TestOrderedCoordinatorRejectsInvalidGroupWithoutAdmittingPrefix(t *testing.T) {
	dispatched := make(chan orderedDispatchResult, 2)
	coordinator := newOrderedLogsCoordinator(orderedTestSettings(), func(_ context.Context, dispatch OrderedLogsDispatch, completion OrderedLogsCompletion) error {
		dispatched <- orderedDispatchResult{dispatch: dispatch, completion: completion}
		return nil
	}, configretry.BackOffConfig{}, 0, 2, nil)
	t.Cleanup(coordinator.shutdown)
	validGroup := orderedTestGroup("valid-channel", OrderedPositionContinue)
	invalidGroup := orderedTestGroup("", OrderedPositionContinue)
	valid := validGroup.children[0].Request
	invalid := invalidGroup.children[0].Request
	group := &orderedLogsGroup{
		children: []OrderedLogsDescriptor{
			validGroup.children[0],
			invalidGroup.children[0],
		},
		items: 2,
		bytes: orderedLogsWire.LogsSize(valid) + orderedLogsWire.LogsSize(invalid),
	}
	require.True(t, group.SetQueueCompletion(func(error) {}))
	require.Error(t, coordinator.add(context.Background(), group))
	coordinator.mu.Lock()
	require.Zero(t, coordinator.staged)
	require.Empty(t, coordinator.parts)
	coordinator.mu.Unlock()
	select {
	case result := <-dispatched:
		t.Fatalf("part of an invalid request group was dispatched: %+v", result.dispatch)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestOrderedGroupRetainsEnvelopeUntilEveryPartitionChildCompletes(t *testing.T) {
	dispatched := make(chan orderedDispatchResult, 2)
	coordinator := newOrderedLogsCoordinator(orderedTestSettings(), func(_ context.Context, dispatch OrderedLogsDispatch, completion OrderedLogsCompletion) error {
		dispatched <- orderedDispatchResult{dispatch: dispatch, completion: completion}
		return nil
	}, configretry.BackOffConfig{}, 0, 2, nil)
	t.Cleanup(coordinator.shutdown)
	first := orderedTestGroup("group-channel-one", OrderedPositionContinue)
	second := orderedTestGroup("group-channel-two", OrderedPositionContinue)
	group := &orderedLogsGroup{
		children: append(append([]OrderedLogsDescriptor(nil), first.children...), second.children...),
		items:    first.items + second.items,
		bytes:    first.bytes + second.bytes,
	}
	done := make(chan error, 1)
	require.True(t, group.SetQueueCompletion(func(err error) { done <- err }))
	require.NoError(t, coordinator.add(context.Background(), group))
	firstDispatch := receiveOrderedDispatch(t, dispatched)
	secondDispatch := receiveOrderedDispatch(t, dispatched)
	firstDispatch.completion.Release()
	secondDispatch.completion.Release()
	firstDispatch.completion.Fail(consumererror.NewPermanent(errors.New("permanent child failure")))
	select {
	case err := <-done:
		t.Fatalf("shared queue envelope retired while a sibling child was still active: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	secondDispatch.completion.Succeed()
	require.Error(t, <-done)
}

func TestOrderedCoordinatorFailureReplaysUnretiredPrefix(t *testing.T) {
	dispatched := make(chan orderedDispatchResult, 4)
	retry := configretry.BackOffConfig{Enabled: true, InitialInterval: time.Millisecond, Multiplier: 1, MaxInterval: time.Millisecond}
	coordinator := newOrderedLogsCoordinator(orderedTestSettings(), func(_ context.Context, dispatch OrderedLogsDispatch, completion OrderedLogsCompletion) error {
		dispatched <- orderedDispatchResult{dispatch: dispatch, completion: completion}
		return nil
	}, retry, 0, 1, nil)
	t.Cleanup(coordinator.shutdown)

	group := orderedTestGroup("channel-retry", OrderedPositionContinue)
	done := make(chan error, 1)
	require.True(t, group.SetQueueCompletion(func(err error) { done <- err }))
	require.NoError(t, coordinator.add(context.Background(), group))
	first := receiveOrderedDispatch(t, dispatched)
	first.completion.Release()
	first.completion.Fail(errors.New("ambiguous write"))

	replayed := receiveOrderedDispatch(t, dispatched)
	require.False(t, replayed.dispatch.Recovery, "normal queue items are replayed without the RECOVER_EVENT flag")
	require.Greater(t, replayed.dispatch.StreamAttempt, first.dispatch.StreamAttempt)
	replayed.completion.Release()
	replayed.completion.Succeed()
	require.NoError(t, <-done)
}

func TestOrderedCoordinatorRetriesWithRecoveryTailPrelude(t *testing.T) {
	dispatched := make(chan orderedDispatchResult, 8)
	retry := configretry.BackOffConfig{Enabled: true, InitialInterval: time.Millisecond, Multiplier: 1, MaxInterval: time.Millisecond}
	coordinator := newOrderedLogsCoordinator(orderedTestSettings(), func(_ context.Context, dispatch OrderedLogsDispatch, completion OrderedLogsCompletion) error {
		dispatched <- orderedDispatchResult{dispatch: dispatch, completion: completion}
		return nil
	}, retry, 0, 1, nil)
	t.Cleanup(coordinator.shutdown)

	first := orderedTestGroup("channel-tail", OrderedPositionContinue)
	firstDone := make(chan error, 1)
	require.True(t, first.SetQueueCompletion(func(err error) { firstDone <- err }))
	require.NoError(t, coordinator.add(context.Background(), first))
	firstDispatch := receiveOrderedDispatch(t, dispatched)
	firstDispatch.completion.Release()
	firstDispatch.completion.Succeed()
	require.NoError(t, <-firstDone)

	second := orderedTestGroup("channel-tail", OrderedPositionContinue)
	secondDone := make(chan error, 1)
	require.True(t, second.SetQueueCompletion(func(err error) { secondDone <- err }))
	require.NoError(t, coordinator.add(context.Background(), second))
	secondDispatch := receiveOrderedDispatch(t, dispatched)
	secondDispatch.completion.Release()
	secondDispatch.completion.Fail(errors.New("ambiguous write"))

	tailReplay := receiveOrderedDispatch(t, dispatched)
	require.True(t, tailReplay.dispatch.Recovery, "the last acknowledged continuation is the RECOVER_EVENT prelude")
	require.Equal(t, first.children[0].Request, tailReplay.dispatch.Request)
	tailReplay.completion.Release()
	tailReplay.completion.Succeed()

	queueReplay := receiveOrderedDispatch(t, dispatched)
	require.False(t, queueReplay.dispatch.Recovery, "unretired queue entries replay as ordinary fragments")
	require.Equal(t, second.children[0].Request, queueReplay.dispatch.Request)
	queueReplay.completion.Release()
	queueReplay.completion.Succeed()
	require.NoError(t, <-secondDone)
}

func TestOrderedCoordinatorBoundsAggregateRecoveryTails(t *testing.T) {
	dispatched := make(chan orderedDispatchResult, 4)
	settings := orderedTestSettings()
	settings.MaxRecoveryTailBytes = orderedLogsTailSize(orderedTestGroup("channel-tail-one", OrderedPositionContinue).children[0])
	coordinator := newOrderedLogsCoordinator(settings, func(_ context.Context, dispatch OrderedLogsDispatch, completion OrderedLogsCompletion) error {
		dispatched <- orderedDispatchResult{dispatch: dispatch, completion: completion}
		return nil
	}, configretry.BackOffConfig{}, 0, 2, nil)
	t.Cleanup(coordinator.shutdown)

	first := orderedTestGroup("channel-tail-one", OrderedPositionContinue)
	second := orderedTestGroup("channel-tail-two", OrderedPositionContinue)
	require.True(t, first.SetQueueCompletion(func(error) {}))
	secondDone := make(chan error, 1)
	require.True(t, second.SetQueueCompletion(func(err error) { secondDone <- err }))
	require.NoError(t, coordinator.add(context.Background(), first))
	firstDispatch := receiveOrderedDispatch(t, dispatched)
	firstDispatch.completion.Release()
	firstDispatch.completion.Succeed()

	require.NoError(t, coordinator.add(context.Background(), second))
	secondDispatch := receiveOrderedDispatch(t, dispatched)
	secondDispatch.completion.Release()
	secondDispatch.completion.Succeed()
	select {
	case err := <-secondDone:
		t.Fatalf("second tail exceeded the aggregate recovery-tail byte limit: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	// Closing the first stream frees its tail budget and lets the second success
	// establish its own recovery tail.
	end := orderedTestGroup("channel-tail-one", OrderedPositionEnd)
	require.True(t, end.SetQueueCompletion(func(error) {}))
	require.NoError(t, coordinator.add(context.Background(), end))
	endDispatch := receiveOrderedDispatch(t, dispatched)
	endDispatch.completion.Release()
	endDispatch.completion.Succeed()
	select {
	case err := <-secondDone:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("second tail was not admitted after the first stream ended")
	}
}

func TestOrderedCoordinatorKeepsQueueItemUntilTailCheckpointIsDurable(t *testing.T) {
	dispatched := make(chan orderedDispatchResult, 1)
	coordinator := newOrderedLogsCoordinator(orderedTestSettings(), func(_ context.Context, dispatch OrderedLogsDispatch, completion OrderedLogsCompletion) error {
		dispatched <- orderedDispatchResult{dispatch: dispatch, completion: completion}
		return nil
	}, configretry.BackOffConfig{}, 0, 1, nil)
	t.Cleanup(coordinator.shutdown)
	store := &orderedTestCheckpointStore{failSave: 1}
	group := orderedTestGroup("channel-durable-tail", OrderedPositionContinue)
	group.SetQueueCheckpointStore(store)
	done := make(chan error, 1)
	require.True(t, group.SetQueueCompletion(func(err error) { done <- err }))
	require.NoError(t, coordinator.add(context.Background(), group))
	dispatch := receiveOrderedDispatch(t, dispatched)
	dispatch.completion.Release()
	dispatch.completion.Succeed()
	select {
	case err := <-done:
		t.Fatalf("queue item retired before its recovery-tail checkpoint was durable: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("queue item did not retire after checkpoint storage recovered")
	}
	store.mu.Lock()
	checkpoint := store.data[orderedLogsCheckpointKey]
	store.mu.Unlock()
	require.NotEmpty(t, checkpoint)
}

func TestOrderedCoordinatorDoesNotReplayCheckpointedQueueItemTwice(t *testing.T) {
	dispatched := make(chan orderedDispatchResult, 2)
	store := &orderedTestCheckpointStore{data: make(map[string][]byte)}
	tail := orderedTestGroup("channel-duplicate-tail", OrderedPositionContinue)
	encoded, err := (orderedLogsEncoding{}).Marshal(context.Background(), tail)
	require.NoError(t, err)
	store.data[orderedLogsCheckpointKey] = encoded

	coordinator := newOrderedLogsCoordinator(orderedTestSettings(), func(_ context.Context, dispatch OrderedLogsDispatch, completion OrderedLogsCompletion) error {
		dispatched <- orderedDispatchResult{dispatch: dispatch, completion: completion}
		return nil
	}, configretry.BackOffConfig{}, 0, 1, nil)
	t.Cleanup(coordinator.shutdown)
	queuedDuplicate := orderedTestGroup("channel-duplicate-tail", OrderedPositionContinue)
	queuedDuplicate.children[0].QueueID = tail.children[0].QueueID
	queuedDuplicateDone := make(chan error, 1)
	queuedDuplicate.SetQueueCheckpointStore(store)
	require.True(t, queuedDuplicate.SetQueueCompletion(func(err error) { queuedDuplicateDone <- err }))
	require.NoError(t, coordinator.add(context.Background(), queuedDuplicate))

	recovery := receiveOrderedDispatch(t, dispatched)
	require.True(t, recovery.dispatch.Recovery)
	recovery.completion.Release()
	recovery.completion.Succeed()
	require.NoError(t, <-queuedDuplicateDone)
	select {
	case duplicate := <-dispatched:
		t.Fatalf("checkpointed tail was also dispatched as ordinary queue work: %+v", duplicate.dispatch)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestOrderedCoordinatorClearsPersistentTailAfterEnd(t *testing.T) {
	dispatched := make(chan orderedDispatchResult, 2)
	store := &orderedTestCheckpointStore{data: make(map[string][]byte)}
	coordinator := newOrderedLogsCoordinator(orderedTestSettings(), func(_ context.Context, dispatch OrderedLogsDispatch, completion OrderedLogsCompletion) error {
		dispatched <- orderedDispatchResult{dispatch: dispatch, completion: completion}
		return nil
	}, configretry.BackOffConfig{}, 0, 1, nil)
	t.Cleanup(coordinator.shutdown)

	continuation := orderedTestGroup("channel-end", OrderedPositionContinue)
	continuation.SetQueueCheckpointStore(store)
	continuationDone := make(chan error, 1)
	require.True(t, continuation.SetQueueCompletion(func(err error) { continuationDone <- err }))
	require.NoError(t, coordinator.add(context.Background(), continuation))
	continued := receiveOrderedDispatch(t, dispatched)
	continued.completion.Release()
	continued.completion.Succeed()
	require.NoError(t, <-continuationDone)
	store.mu.Lock()
	checkpointWithTail := append([]byte(nil), store.data[orderedLogsCheckpointKey]...)
	store.mu.Unlock()
	_, decoded, err := (orderedLogsEncoding{}).Unmarshal(checkpointWithTail)
	require.NoError(t, err)
	require.Len(t, decoded.(*orderedLogsGroup).children, 1)

	end := orderedTestGroup("channel-end", OrderedPositionEnd)
	end.SetQueueCheckpointStore(store)
	endDone := make(chan error, 1)
	require.True(t, end.SetQueueCompletion(func(err error) { endDone <- err }))
	require.NoError(t, coordinator.add(context.Background(), end))
	ended := receiveOrderedDispatch(t, dispatched)
	ended.completion.Release()
	ended.completion.Succeed()
	require.NoError(t, <-endDone)
	store.mu.Lock()
	checkpointWithoutTail := append([]byte(nil), store.data[orderedLogsCheckpointKey]...)
	store.mu.Unlock()
	_, decoded, err = (orderedLogsEncoding{}).Unmarshal(checkpointWithoutTail)
	require.NoError(t, err)
	require.Empty(t, decoded.(*orderedLogsGroup).children)
}

func TestOrderedCoordinatorShutdownCancelsRestoredRecoveryPrelude(t *testing.T) {
	store := &orderedTestCheckpointStore{data: make(map[string][]byte)}
	tail := orderedTestGroup("channel-shutdown-prelude", OrderedPositionContinue)
	encoded, err := (orderedLogsEncoding{}).Marshal(context.Background(), tail)
	require.NoError(t, err)
	store.data[orderedLogsCheckpointKey] = encoded
	started := make(chan struct{})
	canceled := make(chan struct{})
	coordinator := newOrderedLogsCoordinator(orderedTestSettings(), func(ctx context.Context, dispatch OrderedLogsDispatch, _ OrderedLogsCompletion) error {
		if dispatch.Recovery {
			close(started)
			<-ctx.Done()
			close(canceled)
		}
		return ctx.Err()
	}, configretry.BackOffConfig{}, 0, 1, nil)
	queued := orderedTestGroup("channel-shutdown-prelude", OrderedPositionContinue)
	queued.SetQueueCheckpointStore(store)
	done := make(chan error, 1)
	require.True(t, queued.SetQueueCompletion(func(err error) { done <- err }))
	require.NoError(t, coordinator.add(context.Background(), queued))
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("restored recovery prelude was not dispatched")
	}
	coordinator.shutdown()
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not cancel the recovery prelude context")
	}
	require.True(t, experr.IsShutdownErr(<-done))
}

func TestOrderedCoordinatorShutdownUsesPersistentQueueRecoveryError(t *testing.T) {
	dispatched := make(chan orderedDispatchResult, 1)
	coordinator := newOrderedLogsCoordinator(orderedTestSettings(), func(_ context.Context, dispatch OrderedLogsDispatch, completion OrderedLogsCompletion) error {
		dispatched <- orderedDispatchResult{dispatch: dispatch, completion: completion}
		return nil
	}, configretry.BackOffConfig{}, 0, 1, nil)
	group := orderedTestGroup("channel-shutdown", OrderedPositionContinue)
	done := make(chan error, 1)
	require.True(t, group.SetQueueCompletion(func(err error) { done <- err }))
	require.NoError(t, coordinator.add(context.Background(), group))
	_ = receiveOrderedDispatch(t, dispatched)
	coordinator.shutdown()
	shutdownErr := <-done
	require.Error(t, shutdownErr)
	require.True(t, experr.IsShutdownErr(shutdownErr))
}

func TestOrderedCoordinatorPreservesQueueItemWhenCheckpointLoadFails(t *testing.T) {
	coordinator := newOrderedLogsCoordinator(orderedTestSettings(), func(context.Context, OrderedLogsDispatch, OrderedLogsCompletion) error {
		t.Fatal("request must not dispatch without restored checkpoint state")
		return nil
	}, configretry.BackOffConfig{}, 0, 1, nil)
	t.Cleanup(coordinator.shutdown)
	group := orderedTestGroup("channel-checkpoint-read", OrderedPositionContinue)
	group.SetQueueCheckpointStore(&orderedTestCheckpointStore{loadErr: errors.New("checkpoint storage unavailable")})
	done := make(chan error, 1)
	require.True(t, group.SetQueueCompletion(func(err error) { done <- err }))
	err := coordinator.add(context.Background(), group)
	require.Error(t, err)
	require.True(t, experr.IsShutdownErr(err), "persistent queue must retain the request for a later recovery attempt")
	require.Zero(t, coordinator.staged)
}

func TestOrderedCoordinatorHonorsConfiguredWriteConcurrency(t *testing.T) {
	dispatched := make(chan orderedDispatchResult, 2)
	coordinator := newOrderedLogsCoordinator(orderedTestSettings(), func(_ context.Context, dispatch OrderedLogsDispatch, completion OrderedLogsCompletion) error {
		dispatched <- orderedDispatchResult{dispatch: dispatch, completion: completion}
		return nil
	}, configretry.BackOffConfig{}, 0, 1, nil)
	t.Cleanup(coordinator.shutdown)

	first := orderedTestGroup("channel-one", OrderedPositionContinue)
	second := orderedTestGroup("channel-two", OrderedPositionContinue)
	require.True(t, first.SetQueueCompletion(func(error) {}))
	require.True(t, second.SetQueueCompletion(func(error) {}))
	require.NoError(t, coordinator.add(context.Background(), first))
	require.NoError(t, coordinator.add(context.Background(), second))
	firstDispatch := receiveOrderedDispatch(t, dispatched)
	select {
	case other := <-dispatched:
		t.Fatalf("second partition exceeded one write permit: %+v", other.dispatch)
	case <-time.After(20 * time.Millisecond):
	}
	firstDispatch.completion.Release()
	secondDispatch := receiveOrderedDispatch(t, dispatched)
	firstDispatch.completion.Succeed()
	secondDispatch.completion.Release()
	secondDispatch.completion.Succeed()
}

func TestOrderedLogsGroupEncodingRoundTrip(t *testing.T) {
	group := orderedTestGroup("channel-with-binary-\x00-key", OrderedPositionEnd)
	encoded, err := (orderedLogsEncoding{}).Marshal(context.Background(), group)
	require.NoError(t, err)
	_, decodedRequest, err := (orderedLogsEncoding{}).Unmarshal(encoded)
	require.NoError(t, err)
	decoded := decodedRequest.(*orderedLogsGroup)
	require.Len(t, decoded.children, 1)
	require.Equal(t, group.children[0].PartitionKey, decoded.children[0].PartitionKey)
	require.Equal(t, OrderedPositionEnd, decoded.children[0].Position)
	require.Equal(t, group.children[0].QueueID, decoded.children[0].QueueID)
	require.Equal(t, 1, decoded.ItemsCount())
}
