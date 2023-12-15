// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/extension/experimental/storage"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var (
	marshaler   = ptrace.ProtoMarshaler{}
	unmarshaler = ptrace.ProtoUnmarshaler{}
)

type mockHost struct {
	component.Host
	ext map[component.ID]component.Component
}

func (nh *mockHost) GetExtensions() map[component.ID]component.Component {
	return nh.ext
}

// createAndStartTestPersistentQueue creates and starts a fake queue with the given capacity and number of consumers.
func createAndStartTestPersistentQueue(t *testing.T, capacity, numConsumers int, consumeFunc func(_ context.Context, item ptrace.Traces) error) Queue[ptrace.Traces] {
	pq := NewPersistentQueue[ptrace.Traces](capacity, component.DataTypeTraces, component.ID{}, marshaler.MarshalTraces,
		unmarshaler.UnmarshalTraces, exportertest.NewNopCreateSettings())
	host := &mockHost{ext: map[component.ID]component.Component{
		{}: NewMockStorageExtension(nil),
	}}
	consumers := NewQueueConsumers(pq, numConsumers, consumeFunc)
	require.NoError(t, consumers.Start(context.Background(), host))
	t.Cleanup(func() {
		assert.NoError(t, consumers.Shutdown(context.Background()))
	})
	return pq
}

func createTestPersistentQueueWithClient(client storage.Client) *persistentQueue[ptrace.Traces] {
	pq := NewPersistentQueue[ptrace.Traces](1000, component.DataTypeTraces, component.ID{}, marshaler.MarshalTraces,
		unmarshaler.UnmarshalTraces, exportertest.NewNopCreateSettings()).(*persistentQueue[ptrace.Traces])

	pq.initClient(context.Background(), client)
	return pq
}

func createTestPersistentQueueWithCapacity(t testing.TB, ext storage.Extension, capacity int) *persistentQueue[ptrace.Traces] {
	pq := NewPersistentQueue[ptrace.Traces](capacity, component.DataTypeTraces, component.ID{}, marshaler.MarshalTraces,
		unmarshaler.UnmarshalTraces, exportertest.NewNopCreateSettings()).(*persistentQueue[ptrace.Traces])

	require.NoError(t, pq.Start(context.Background(), &mockHost{ext: map[component.ID]component.Component{{}: ext}}))
	return pq
}

func createTestPersistentQueue(t testing.TB, ext storage.Extension) *persistentQueue[ptrace.Traces] {
	return createTestPersistentQueueWithCapacity(t, ext, 1000)
}

func TestPersistentQueue_FullCapacity(t *testing.T) {
	start := make(chan struct{})
	done := make(chan struct{})
	pq := createAndStartTestPersistentQueue(t, 5, 1, func(context.Context, ptrace.Traces) error {
		<-start
		<-done
		return nil
	})
	assert.Equal(t, 0, pq.Size())

	req := newTraces(1, 10)

	// First request is picked by the consumer. Wait until the consumer is blocked on done.
	assert.NoError(t, pq.Offer(context.Background(), req))
	start <- struct{}{}
	close(start)

	for i := 0; i < 10; i++ {
		result := pq.Offer(context.Background(), newTraces(1, 10))
		if i < 5 {
			assert.NoError(t, result)
		} else {
			assert.ErrorIs(t, result, ErrQueueIsFull)
		}
	}
	assert.Equal(t, 5, pq.Size())
	close(done)
}

func TestPersistentQueue_Shutdown(t *testing.T) {
	pq := createAndStartTestPersistentQueue(t, 1001, 100, func(context.Context, ptrace.Traces) error { return nil })
	req := newTraces(1, 10)

	for i := 0; i < 1000; i++ {
		assert.NoError(t, pq.Offer(context.Background(), req))
	}

}

func TestPersistentQueue_ConsumersProducers(t *testing.T) {
	cases := []struct {
		numMessagesProduced int
		numConsumers        int
	}{
		{
			numMessagesProduced: 1,
			numConsumers:        1,
		},
		{
			numMessagesProduced: 100,
			numConsumers:        1,
		},
		{
			numMessagesProduced: 100,
			numConsumers:        3,
		},
		{
			numMessagesProduced: 1,
			numConsumers:        100,
		},
		{
			numMessagesProduced: 100,
			numConsumers:        100,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("#messages: %d #consumers: %d", c.numMessagesProduced, c.numConsumers), func(t *testing.T) {
			req := newTraces(1, 10)

			numMessagesConsumed := &atomic.Int32{}
			pq := createAndStartTestPersistentQueue(t, 1000, c.numConsumers, func(context.Context, ptrace.Traces) error {
				numMessagesConsumed.Add(int32(1))
				return nil
			})

			for i := 0; i < c.numMessagesProduced; i++ {
				assert.NoError(t, pq.Offer(context.Background(), req))
			}

			assert.Eventually(t, func() bool {
				return c.numMessagesProduced == int(numMessagesConsumed.Load())
			}, 5*time.Second, 10*time.Millisecond)
		})
	}
}

func newTraces(numTraces int, numSpans int) ptrace.Traces {
	traces := ptrace.NewTraces()
	batch := traces.ResourceSpans().AppendEmpty()
	batch.Resource().Attributes().PutStr("resource-attr", "some-resource")
	batch.Resource().Attributes().PutInt("num-traces", int64(numTraces))
	batch.Resource().Attributes().PutInt("num-spans", int64(numSpans))

	for i := 0; i < numTraces; i++ {
		traceID := pcommon.TraceID([16]byte{1, 2, 3, byte(i)})
		ils := batch.ScopeSpans().AppendEmpty()
		for j := 0; j < numSpans; j++ {
			span := ils.Spans().AppendEmpty()
			span.SetTraceID(traceID)
			span.SetSpanID([8]byte{1, 2, 3, byte(j)})
			span.SetName("should-not-be-changed")
			span.Attributes().PutInt("int-attribute", int64(j))
			span.Attributes().PutStr("str-attribute-1", "foobar")
			span.Attributes().PutStr("str-attribute-2", "fdslafjasdk12312312jkl")
			span.Attributes().PutStr("str-attribute-3", "AbcDefGeKKjkfdsafasdfsdasdf")
			span.Attributes().PutStr("str-attribute-4", "xxxxxx")
			span.Attributes().PutStr("str-attribute-5", "abcdef")
		}
	}

	return traces
}

func TestToStorageClient(t *testing.T) {
	getStorageClientError := errors.New("unable to create storage client")
	testCases := []struct {
		desc           string
		storage        storage.Extension
		numStorages    int
		storageIndex   int
		expectedError  error
		getClientError error
	}{
		{
			desc:          "obtain storage extension by name",
			numStorages:   2,
			storageIndex:  0,
			expectedError: nil,
		},
		{
			desc:          "fail on not existing storage extension",
			numStorages:   2,
			storageIndex:  100,
			expectedError: errNoStorageClient,
		},
		{
			desc:          "invalid extension type",
			numStorages:   2,
			storageIndex:  100,
			expectedError: errNoStorageClient,
		},
		{
			desc:           "fail on error getting storage client from extension",
			numStorages:    1,
			storageIndex:   0,
			expectedError:  getStorageClientError,
			getClientError: getStorageClientError,
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			storageID := component.NewIDWithName("file_storage", strconv.Itoa(tC.storageIndex))

			var extensions = map[component.ID]component.Component{}
			for i := 0; i < tC.numStorages; i++ {
				extensions[component.NewIDWithName("file_storage", strconv.Itoa(i))] = NewMockStorageExtension(tC.getClientError)
			}
			host := &mockHost{ext: extensions}
			ownerID := component.NewID("foo_exporter")

			// execute
			client, err := toStorageClient(context.Background(), storageID, host, ownerID, component.DataTypeTraces)

			// verify
			if tC.expectedError != nil {
				assert.ErrorIs(t, err, tC.expectedError)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestInvalidStorageExtensionType(t *testing.T) {
	storageID := component.NewIDWithName("extension", "extension")

	// make a test extension
	factory := extensiontest.NewNopFactory()
	extConfig := factory.CreateDefaultConfig()
	settings := extensiontest.NewNopCreateSettings()
	extension, err := factory.CreateExtension(context.Background(), settings, extConfig)
	assert.NoError(t, err)
	var extensions = map[component.ID]component.Component{
		storageID: extension,
	}
	host := &mockHost{ext: extensions}
	ownerID := component.NewID("foo_exporter")

	// execute
	client, err := toStorageClient(context.Background(), storageID, host, ownerID, component.DataTypeTraces)

	// we should get an error about the extension type
	assert.ErrorIs(t, err, errWrongExtensionType)
	assert.Nil(t, client)
}

func TestPersistentQueue_StopAfterBadStart(t *testing.T) {
	pq := NewPersistentQueue[ptrace.Traces](1, component.DataTypeTraces, component.ID{}, marshaler.MarshalTraces, unmarshaler.UnmarshalTraces, exportertest.NewNopCreateSettings())
	// verify that stopping a un-start/started w/error queue does not panic
	assert.NoError(t, pq.Shutdown(context.Background()))
}

func TestPersistentQueue_CorruptedData(t *testing.T) {
	req := newTraces(5, 10)

	cases := []struct {
		name                               string
		corruptAllData                     bool
		corruptSomeData                    bool
		corruptCurrentlyDispatchedItemsKey bool
		corruptReadIndex                   bool
		corruptWriteIndex                  bool
		desiredQueueSize                   int
	}{
		{
			name:             "corrupted no items",
			desiredQueueSize: 3,
		},
		{
			name:             "corrupted all items",
			corruptAllData:   true,
			desiredQueueSize: 2, // - the dispatched item which was corrupted.
		},
		{
			name:             "corrupted some items",
			corruptSomeData:  true,
			desiredQueueSize: 2, // - the dispatched item which was corrupted.
		},
		{
			name:                               "corrupted dispatched items key",
			corruptCurrentlyDispatchedItemsKey: true,
			desiredQueueSize:                   2,
		},
		{
			name:             "corrupted read index",
			corruptReadIndex: true,
			desiredQueueSize: 1, // The dispatched item.
		},
		{
			name:              "corrupted write index",
			corruptWriteIndex: true,
			desiredQueueSize:  1, // The dispatched item.
		},
		{
			name:                               "corrupted everything",
			corruptAllData:                     true,
			corruptCurrentlyDispatchedItemsKey: true,
			corruptReadIndex:                   true,
			corruptWriteIndex:                  true,
			desiredQueueSize:                   0,
		},
	}

	badBytes := []byte{0, 1, 2}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ext := NewMockStorageExtension(nil)
			ps := createTestPersistentQueue(t, ext)

			// Put some items, make sure they are loaded and shutdown the storage...
			for i := 0; i < 3; i++ {
				err := ps.Offer(context.Background(), req)
				require.NoError(t, err)
			}
			assert.Equal(t, 3, ps.Size())
			require.True(t, ps.Consume(func(ctx context.Context, traces ptrace.Traces) error {
				return NewShutdownErr(nil)
			}))
			assert.Equal(t, 2, ps.Size())

			// We can corrupt data (in several ways) and not worry since we return ShutdownErr client will not be touched.
			if c.corruptAllData || c.corruptSomeData {
				require.NoError(t, ps.client.Set(context.Background(), "0", badBytes))
			}
			if c.corruptAllData {
				require.NoError(t, ps.client.Set(context.Background(), "1", badBytes))
				require.NoError(t, ps.client.Set(context.Background(), "2", badBytes))
			}

			if c.corruptCurrentlyDispatchedItemsKey {
				require.NoError(t, ps.client.Set(context.Background(), currentlyDispatchedItemsKey, badBytes))
			}

			if c.corruptReadIndex {
				require.NoError(t, ps.client.Set(context.Background(), readIndexKey, badBytes))
			}

			if c.corruptWriteIndex {
				require.NoError(t, ps.client.Set(context.Background(), writeIndexKey, badBytes))
			}

			// Cannot close until we corrupt the data because the
			assert.NoError(t, ps.Shutdown(context.Background()))

			// Reload
			newPs := createTestPersistentQueue(t, ext)
			assert.Equal(t, c.desiredQueueSize, newPs.Size())
		})
	}
}

func TestPersistentQueue_CurrentlyProcessedItems(t *testing.T) {
	req := newTraces(5, 10)

	ext := NewMockStorageExtension(nil)
	ps := createTestPersistentQueue(t, ext)

	for i := 0; i < 5; i++ {
		err := ps.Offer(context.Background(), req)
		require.NoError(t, err)
	}

	requireCurrentlyDispatchedItemsEqual(t, ps, []uint64{})

	// Takes index 0 in process.
	readReq, _, found := ps.getNextItem(context.Background())
	require.True(t, found)
	assert.Equal(t, req, readReq)
	requireCurrentlyDispatchedItemsEqual(t, ps, []uint64{0})

	// This takes item 1 to process.
	secondReadReq, onProcessingFinished, found := ps.getNextItem(context.Background())
	require.True(t, found)
	assert.Equal(t, req, secondReadReq)
	requireCurrentlyDispatchedItemsEqual(t, ps, []uint64{0, 1})

	// Lets mark item 1 as finished, it will remove it from the currently dispatched items list.
	onProcessingFinished(nil)
	requireCurrentlyDispatchedItemsEqual(t, ps, []uint64{0})

	// Reload the storage. Since items 0 was not finished, this should be re-enqueued at the end.
	// The queue should be essentially {3,4,0,2}.
	newPs := createTestPersistentQueue(t, ext)
	assert.Equal(t, 4, newPs.Size())
	requireCurrentlyDispatchedItemsEqual(t, newPs, []uint64{})

	// We should be able to pull all remaining items now
	for i := 0; i < 4; i++ {
		newPs.Consume(func(ctx context.Context, traces ptrace.Traces) error {
			assert.Equal(t, req, traces)
			return nil
		})
	}

	// The queue should be now empty
	requireCurrentlyDispatchedItemsEqual(t, newPs, []uint64{})
	assert.Equal(t, 0, newPs.Size())
	// The writeIndex should be now set accordingly
	require.EqualValues(t, 6, newPs.writeIndex)

	// There should be no items left in the storage
	for i := 0; i < int(newPs.writeIndex); i++ {
		bb, err := newPs.client.Get(context.Background(), getItemKey(uint64(i)))
		require.NoError(t, err)
		require.Nil(t, bb)
	}
}

// this test attempts to check if all the invariants are kept if the queue is recreated while
// close to full and with some items dispatched
func TestPersistentQueueStartWithNonDispatched(t *testing.T) {
	req := newTraces(5, 10)

	ext := NewMockStorageExtension(nil)
	ps := createTestPersistentQueueWithCapacity(t, ext, 5)

	// Put in items up to capacity
	for i := 0; i < 5; i++ {
		err := ps.Offer(context.Background(), req)
		require.NoError(t, err)
	}

	// get one item out, but don't mark it as processed
	<-ps.putChan
	require.True(t, ps.Consume(func(ctx context.Context, traces ptrace.Traces) error {
		// put one more item in
		require.NoError(t, ps.Offer(context.Background(), req))
		require.Equal(t, 5, ps.Size())
		return NewShutdownErr(nil)
	}))
	assert.NoError(t, ps.Shutdown(context.Background()))

	// Reload with extra capacity to make sure we re-enqueue in-progress items.
	newPs := createTestPersistentQueueWithCapacity(t, ext, 6)
	require.Equal(t, 6, newPs.Size())
}

func TestPersistentQueue_PutCloseReadClose(t *testing.T) {
	req := newTraces(5, 10)
	ext := NewMockStorageExtension(nil)
	ps := createTestPersistentQueue(t, ext)
	assert.Equal(t, 0, ps.Size())

	// Put two elements and close the extension
	assert.NoError(t, ps.Offer(context.Background(), req))
	assert.NoError(t, ps.Offer(context.Background(), req))
	assert.Equal(t, 2, ps.Size())
	// TODO: Remove this, after the initialization writes the readIndex.
	_, _, _ = ps.getNextItem(context.Background())
	assert.NoError(t, ps.Shutdown(context.Background()))

	newPs := createTestPersistentQueue(t, ext)
	require.Equal(t, 2, newPs.Size())

	// Let's read both of the elements we put
	newPs.Consume(func(ctx context.Context, traces ptrace.Traces) error {
		require.Equal(t, req, traces)
		return nil
	})
	assert.Equal(t, 1, newPs.Size())

	newPs.Consume(func(ctx context.Context, traces ptrace.Traces) error {
		require.Equal(t, req, traces)
		return nil
	})
	require.Equal(t, 0, newPs.Size())
	assert.NoError(t, newPs.Shutdown(context.Background()))
}

func BenchmarkPersistentQueue_TraceSpans(b *testing.B) {
	cases := []struct {
		numTraces        int
		numSpansPerTrace int
	}{
		{
			numTraces:        1,
			numSpansPerTrace: 1,
		},
		{
			numTraces:        1,
			numSpansPerTrace: 10,
		},
		{
			numTraces:        10,
			numSpansPerTrace: 10,
		},
	}

	for _, c := range cases {
		b.Run(fmt.Sprintf("#traces: %d #spansPerTrace: %d", c.numTraces, c.numSpansPerTrace), func(bb *testing.B) {
			ext := NewMockStorageExtension(nil)
			ps := createTestPersistentQueueWithCapacity(b, ext, 10000000)

			req := newTraces(c.numTraces, c.numSpansPerTrace)

			bb.ResetTimer()

			for i := 0; i < bb.N; i++ {
				require.NoError(bb, ps.Offer(context.Background(), req))
			}

			for i := 0; i < bb.N; i++ {
				require.True(bb, ps.Consume(func(context.Context, ptrace.Traces) error { return nil }))
			}
			require.NoError(b, ext.Shutdown(context.Background()))
		})
	}
}

func TestItemIndexMarshaling(t *testing.T) {
	cases := []struct {
		in  uint64
		out uint64
	}{
		{
			in:  0,
			out: 0,
		},
		{
			in:  1,
			out: 1,
		},
		{
			in:  0xFFFFFFFFFFFFFFFF,
			out: 0xFFFFFFFFFFFFFFFF,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("#elements:%v", c.in), func(tt *testing.T) {
			buf := itemIndexToBytes(c.in)
			out, err := bytesToItemIndex(buf)
			require.NoError(t, err)
			require.Equal(t, c.out, out)
		})
	}
}

func TestItemIndexArrayMarshaling(t *testing.T) {
	cases := []struct {
		in  []uint64
		out []uint64
	}{
		{
			in:  []uint64{0, 1, 2},
			out: []uint64{0, 1, 2},
		},
		{
			in:  []uint64{},
			out: nil,
		},
		{
			in:  nil,
			out: nil,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("#elements:%v", c.in), func(tt *testing.T) {
			buf := itemIndexArrayToBytes(c.in)
			out, err := bytesToItemIndexArray(buf)
			require.NoError(t, err)
			require.Equal(t, c.out, out)
		})
	}
}

func TestPersistentQueue_ShutdownWhileConsuming(t *testing.T) {
	ps := createTestPersistentQueue(t, NewMockStorageExtension(nil))

	assert.Equal(t, 0, ps.Size())
	assert.False(t, ps.client.(*mockStorageClient).isClosed())

	assert.NoError(t, ps.Offer(context.Background(), newTraces(5, 10)))

	_, onProcessingFinished, ok := ps.getNextItem(context.Background())
	require.True(t, ok)
	assert.False(t, ps.client.(*mockStorageClient).isClosed())
	assert.NoError(t, ps.Shutdown(context.Background()))
	assert.False(t, ps.client.(*mockStorageClient).isClosed())
	onProcessingFinished(nil)
	assert.True(t, ps.client.(*mockStorageClient).isClosed())
}

func TestPersistentQueue_StorageFull(t *testing.T) {
	req := newTraces(5, 10)
	marshaled, err := marshaler.MarshalTraces(req)
	require.NoError(t, err)
	maxSizeInBytes := len(marshaled) * 5 // arbitrary small number
	freeSpaceInBytes := 1

	client := newFakeBoundedStorageClient(maxSizeInBytes)
	ps := createTestPersistentQueueWithClient(client)

	// Put enough items in to fill the underlying storage
	reqCount := 0
	for {
		err = ps.Offer(context.Background(), req)
		if errors.Is(err, syscall.ENOSPC) {
			break
		}
		require.NoError(t, err)
		reqCount++
	}

	// Check that the size is correct
	require.Equal(t, reqCount, ps.Size(), "Size must be equal to the number of items inserted")

	// Manually set the storage to only have a small amount of free space left
	newMaxSize := client.GetSizeInBytes() + freeSpaceInBytes
	client.SetMaxSizeInBytes(newMaxSize)

	// Try to put an item in, should fail
	require.Error(t, ps.Offer(context.Background(), req))

	// Take out all the items
	// Getting the first item fails, as we can't update the state in storage, so we just delete it without returning it
	// Subsequent items succeed, as deleting the first item frees enough space for the state update
	reqCount--
	for i := reqCount; i > 0; i-- {
		require.True(t, ps.Consume(func(context.Context, ptrace.Traces) error { return nil }))
	}

	// We should be able to put a new item in
	// However, this will fail if deleting items fails with full storage
	require.NoError(t, ps.Offer(context.Background(), req))
}

func TestPersistentQueue_ItemDispatchingFinish_ErrorHandling(t *testing.T) {
	errDeletingItem := fmt.Errorf("error deleting item")
	errUpdatingDispatched := fmt.Errorf("error updating dispatched items")
	testCases := []struct {
		storageErrors []error
		expectedError error
		description   string
	}{
		{
			description:   "no errors",
			storageErrors: []error{},
			expectedError: nil,
		},
		{
			description: "error on first transaction, success afterwards",
			storageErrors: []error{
				errUpdatingDispatched,
			},
			expectedError: nil,
		},
		{
			description: "error on first and second transaction",
			storageErrors: []error{
				errUpdatingDispatched,
				errDeletingItem,
			},
			expectedError: errDeletingItem,
		},
		{
			description: "error on first and third transaction",
			storageErrors: []error{
				errUpdatingDispatched,
				nil,
				errUpdatingDispatched,
			},
			expectedError: errUpdatingDispatched,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			client := newFakeStorageClientWithErrors(testCase.storageErrors)
			ps := createTestPersistentQueueWithClient(client)
			client.Reset()

			err := ps.itemDispatchingFinish(context.Background(), 0)

			require.ErrorIs(t, err, testCase.expectedError)
		})
	}
}

func requireCurrentlyDispatchedItemsEqual(t *testing.T, pq *persistentQueue[ptrace.Traces], compare []uint64) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	assert.ElementsMatch(t, compare, pq.currentlyDispatchedItems)
}
