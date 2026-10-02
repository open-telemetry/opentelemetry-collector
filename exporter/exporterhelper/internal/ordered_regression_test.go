// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

func (s *orderedTestCheckpointStore) SaveCheckpointAndItems(_ context.Context, key string, value []byte, updates []request.QueueItemUpdate) error {
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
	for _, update := range updates {
		s.data[fmt.Sprintf("item-%d", update.Token)] = append([]byte(nil), update.Value...)
	}
	return nil
}

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

func checkpointTail(t *testing.T, store *orderedTestCheckpointStore, g *orderedLogsGroup) {
	t.Helper()
	encoded, err := (orderedLogsEncoding{}).Marshal(context.Background(), g)
	require.NoError(t, err)
	store.data[orderedLogsCheckpointKey] = encoded
}

func TestOrderedRecoveryOverlapAllowsSuccessorBeforeTailACK(t *testing.T) {
	for _, workers := range []int{1, 32} {
		t.Run(strconv.Itoa(workers), func(t *testing.T) {
			store := &orderedTestCheckpointStore{data: make(map[string][]byte)}
			a := orderedTestGroup("p", OrderedPositionContinue)
			checkpointTail(t, store, a)
			c, dispatched := regressionCoordinator(t, workers)
			a.SetQueueCheckpointStore(store)
			done := make(chan error, 1)
			a.SetQueueCompletion(func(err error) { done <- err })
			require.NoError(t, c.add(context.Background(), a))
			recovery := receiveOrderedDispatch(t, dispatched)
			b := orderedTestGroup("p", OrderedPositionEnd)
			b.SetQueueCompletion(func(error) {})
			require.NoError(t, c.add(context.Background(), b))
			recovery.completion.Release()
			successor := receiveOrderedDispatch(t, dispatched)
			require.Equal(t, b.children[0].QueueID, successor.dispatch.QueueID)
			successor.completion.Succeed()
			recovery.completion.Succeed()
			require.NoError(t, <-done)
			select {
			case d := <-dispatched:
				t.Fatalf("unexpected duplicate: %+v", d.dispatch)
			default:
			}
		})
	}
}

func TestOrderedRecoveryACKDoesNotReleaseSuccessorWrite(t *testing.T) {
	store := &orderedTestCheckpointStore{data: make(map[string][]byte)}
	checkpointTail(t, store, orderedTestGroup("p", OrderedPositionContinue))
	c, dispatched := regressionCoordinator(t, 1)
	b := orderedTestGroup("p", OrderedPositionContinue)
	b.SetQueueCheckpointStore(store)
	b.SetQueueCompletion(func(error) {})
	require.NoError(t, c.add(context.Background(), b))
	recovery := receiveOrderedDispatch(t, dispatched)
	recovery.completion.Release()
	active := receiveOrderedDispatch(t, dispatched)
	next := orderedTestGroup("p", OrderedPositionEnd)
	next.SetQueueCompletion(func(error) {})
	require.NoError(t, c.add(context.Background(), next))
	recovery.completion.Succeed()
	select {
	case d := <-dispatched:
		t.Fatalf("old ACK released successor: %+v", d.dispatch)
	case <-time.After(20 * time.Millisecond):
	}
	c.mu.Lock()
	require.Equal(t, 1, c.activeWrites)
	c.mu.Unlock()
	active.completion.Release()
	last := receiveOrderedDispatch(t, dispatched)
	require.Equal(t, next.children[0].QueueID, last.dispatch.QueueID)
	active.completion.Succeed()
	last.completion.Succeed()
}

func TestOrderedPersistentGroupRetiresChildrenAtomicallyWithTail(t *testing.T) {
	store := &orderedTestCheckpointStore{data: make(map[string][]byte)}
	c, dispatched := regressionCoordinator(t, 2)
	a, b, q := orderedTestGroup("p", OrderedPositionContinue), orderedTestGroup("p", OrderedPositionContinue), orderedTestGroup("q", OrderedPositionContinue)
	group := &orderedLogsGroup{children: []OrderedLogsDescriptor{a.children[0], b.children[0], q.children[0]}, items: 3, bytes: a.bytes + b.bytes + q.bytes}
	group.SetQueueItemToken(7)
	group.SetQueueCheckpointStore(store)
	group.SetQueueCompletion(func(error) {})
	require.NoError(t, c.add(context.Background(), group))
	first, other := receiveOrderedDispatch(t, dispatched), receiveOrderedDispatch(t, dispatched)
	if first.dispatch.PartitionKey == "q" {
		first, other = other, first
	}
	first.completion.Release()
	second := receiveOrderedDispatch(t, dispatched)
	first.completion.Succeed()
	second.completion.Succeed()
	store.mu.Lock()
	body := append([]byte(nil), store.data["item-7"]...)
	store.mu.Unlock()
	_, decoded, err := (orderedLogsEncoding{}).Unmarshal(body)
	require.NoError(t, err)
	restored := decoded.(*orderedLogsGroup)
	require.True(t, restored.retired[a.children[0].QueueID])
	require.True(t, restored.retired[b.children[0].QueueID])
	require.False(t, restored.retired[q.children[0].QueueID])
	c.shutdown()
	restarted, replay := regressionCoordinator(t, 2)
	restored.SetQueueItemToken(7)
	restored.SetQueueCheckpointStore(store)
	restored.SetQueueCompletion(func(error) {})
	require.NoError(t, restarted.add(context.Background(), restored))
	prelude, unresolved := receiveOrderedDispatch(t, replay), receiveOrderedDispatch(t, replay)
	if prelude.dispatch.PartitionKey == "q" {
		prelude, unresolved = unresolved, prelude
	}
	require.True(t, prelude.dispatch.Recovery)
	require.Equal(t, b.children[0].QueueID, prelude.dispatch.QueueID)
	prelude.completion.Release()
	successor := orderedTestGroup("p", OrderedPositionEnd)
	successor.SetQueueCompletion(func(error) {})
	require.NoError(t, restarted.add(context.Background(), successor))
	next := receiveOrderedDispatch(t, replay)
	require.Equal(t, successor.children[0].QueueID, next.dispatch.QueueID, "retired A must never follow recovery B")
	prelude.completion.Succeed()
	unresolved.completion.Succeed()
	next.completion.Succeed()
	_ = other
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

func TestOrderedAtomicGroupRequestSizerChargesChildren(t *testing.T) {
	a, b := orderedTestGroup("p", OrderedPositionContinue), orderedTestGroup("q", OrderedPositionEnd)
	g := &orderedLogsGroup{children: append(a.children, b.children...)}
	require.Equal(t, int64(2), (request.RequestsSizer{}).Sizeof(g))
}

func TestOrderedRetiredEnvelopeDoesNotConsumePartitionCapacity(t *testing.T) {
	store := &orderedTestCheckpointStore{data: make(map[string][]byte)}
	checkpointTail(t, store, orderedTestGroup("q", OrderedPositionContinue))
	settings := orderedTestSettings()
	settings.MaxActivePartitions = 1
	c := newOrderedLogsCoordinator(settings, func(context.Context, OrderedLogsDispatch, OrderedLogsCompletion) error { return nil }, configretry.BackOffConfig{}, 0, 1, nil)
	t.Cleanup(c.shutdown)
	group := orderedTestGroup("p", OrderedPositionEnd)
	group.retired = map[[16]byte]bool{group.children[0].QueueID: true}
	group.SetQueueCheckpointStore(store)
	done := make(chan error, 1)
	group.SetQueueCompletion(func(err error) { done <- err })
	added := make(chan error, 1)
	go func() { added <- c.add(context.Background(), group) }()
	select {
	case err := <-added:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("retired envelope waited for a partition slot")
	}
	require.NoError(t, <-done)
}

func TestOrderedRecoveryErrorFencesLaterQueueItems(t *testing.T) {
	store := &orderedTestCheckpointStore{loadErr: errors.New("disk I/O failure")}
	c, dispatched := regressionCoordinator(t, 1)
	a := orderedTestGroup("p", OrderedPositionContinue)
	a.SetQueueCheckpointStore(store)
	require.Error(t, c.add(context.Background(), a))
	store.loadErr = nil
	b := orderedTestGroup("p", OrderedPositionContinue)
	b.SetQueueCheckpointStore(store)
	require.Error(t, c.add(context.Background(), b))
	select {
	case <-dispatched:
		t.Fatal("later item overtook a recovery failure")
	default:
	}
	c.mu.Lock()
	require.Empty(t, c.parts)
	c.mu.Unlock()
}
