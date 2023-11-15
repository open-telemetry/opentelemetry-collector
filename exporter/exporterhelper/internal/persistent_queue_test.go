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

// createTestQueue creates and starts a fake queue with the given capacity and number of consumers.
func createTestQueue(t *testing.T, capacity, numConsumers int, callback func(_ context.Context, item ptrace.Traces)) Queue[ptrace.Traces] {
	pq := NewPersistentQueue[ptrace.Traces](capacity, component.DataTypeTraces, component.ID{}, marshaler.MarshalTraces,
		unmarshaler.UnmarshalTraces, exportertest.NewNopCreateSettings())
	host := &mockHost{ext: map[component.ID]component.Component{
		{}: NewMockStorageExtension(NewMockStorageClient()),
	}}
	require.NoError(t, pq.Start(context.Background(), host))
	consumers := NewQueueConsumers(pq, numConsumers, callback)
	consumers.Start()
	t.Cleanup(func() {
		assert.NoError(t, pq.Shutdown(context.Background()))
		consumers.Shutdown()
	})
	return pq
}

// createNopPersistentQueueWithHost creates a persistent queue with a given capacity and host without consumers.
func createNopPersistentQueueWithHost(t testing.TB, capacity int, host component.Host) *persistentQueue[ptrace.Traces] {
	pq := NewPersistentQueue[ptrace.Traces](capacity, component.DataTypeTraces, component.ID{}, marshaler.MarshalTraces,
		unmarshaler.UnmarshalTraces, exportertest.NewNopCreateSettings())
	require.NoError(t, pq.Start(context.Background(), host))
	return pq.(*persistentQueue[ptrace.Traces])
}

func TestPersistentQueue_FullCapacity(t *testing.T) {
	start := make(chan struct{})
	done := make(chan struct{})
	pq := createTestQueue(t, 5, 1, func(context.Context, ptrace.Traces) {
		start <- struct{}{}
		<-done
	})
	assert.Equal(t, 0, pq.Size())

	req := newTraces(1, 10)

	// First request is picked by the consumer. Wait until the consumer is blocked on done.
	assert.NoError(t, pq.Offer(context.Background(), req))
	<-start

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

func TestPersistentQueueShutdown(t *testing.T) {
	pq := createTestQueue(t, 1001, 100, func(context.Context, ptrace.Traces) {})
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
			pq := createTestQueue(t, 1000, c.numConsumers, func(context.Context, ptrace.Traces) {
				numMessagesConsumed.Add(int32(1))
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
				var ext storage.Extension
				if tC.getClientError == nil {
					ext = NewMockStorageExtension(NewMockStorageClient())
				} else {
					ext = NewFailingStorageExtension(tC.getClientError)
				}
				extensions[component.NewIDWithName("file_storage", strconv.Itoa(i))] = ext
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
			client := NewMockStorageClient()
			ext := NewMockStorageExtension(client)
			host := &mockHost{ext: map[component.ID]component.Component{{}: ext}}
			pq := createNopPersistentQueueWithHost(t, 1000, host)

			ctx := context.Background()

			// Put some items, make sure they are loaded and shutdown the storage...
			for i := 0; i < 3; i++ {
				err := pq.Offer(context.Background(), req)
				require.NoError(t, err)
			}
			assert.Equal(t, 3, pq.Size())
			_, _ = pq.Poll()
			assert.Equal(t, 2, pq.Size())
			assert.NoError(t, pq.Shutdown(context.Background()))

			// ... so now we can corrupt data (in several ways)
			if c.corruptAllData || c.corruptSomeData {
				require.NoError(t, client.Set(ctx, "0", badBytes))
			}
			if c.corruptAllData {
				require.NoError(t, client.Set(ctx, "1", badBytes))
				require.NoError(t, client.Set(ctx, "2", badBytes))
			}

			if c.corruptCurrentlyDispatchedItemsKey {
				require.NoError(t, client.Set(ctx, currentlyDispatchedItemsKey, badBytes))
			}

			if c.corruptReadIndex {
				require.NoError(t, client.Set(ctx, readIndexKey, badBytes))
			}

			if c.corruptWriteIndex {
				require.NoError(t, client.Set(ctx, writeIndexKey, badBytes))
			}

			// Reload
			newPQ := createNopPersistentQueueWithHost(t, 1000, host)
			assert.Equal(t, c.desiredQueueSize, newPQ.Size())
		})
	}
}

func TestPersistentQueue_CurrentlyProcessedItems(t *testing.T) {
	req := newTraces(5, 10)

	client := NewMockStorageClient()
	ext := NewMockStorageExtension(client)
	host := &mockHost{ext: map[component.ID]component.Component{{}: ext}}
	pq := createNopPersistentQueueWithHost(t, 1000, host)

	for i := 0; i < 5; i++ {
		err := pq.Offer(context.Background(), req)
		require.NoError(t, err)
	}

	requireCurrentlyDispatchedItemsEqual(t, pq, []uint64{})

	// Takes index 0 in process.
	readReq, found := pq.Poll()
	require.True(t, found)
	assert.Equal(t, req, readReq.Request)
	requireCurrentlyDispatchedItemsEqual(t, pq, []uint64{0})

	// This takes item 1 to process.
	secondReadReq, found := pq.Poll()
	require.True(t, found)
	requireCurrentlyDispatchedItemsEqual(t, pq, []uint64{0, 1})

	// Lets mark item 1 as finished, it will remove it from the currently dispatched items list.
	secondReadReq.OnProcessingFinished()
	requireCurrentlyDispatchedItemsEqual(t, pq, []uint64{0})

	// Reload the storage. Since items 0 was not finished, this should be re-enqueued at the end.
	// The queue should be essentially {3,4,0,2}.
	newPQ := createNopPersistentQueueWithHost(t, 1000, host)
	assert.Equal(t, 4, newPQ.Size())
	requireCurrentlyDispatchedItemsEqual(t, newPQ, []uint64{})

	// We should be able to pull all remaining items now
	for i := 0; i < 4; i++ {
		qReq, found := newPQ.Poll()
		require.True(t, found)
		qReq.OnProcessingFinished()
	}

	// The queue should be now empty
	requireCurrentlyDispatchedItemsEqual(t, newPQ, []uint64{})
	assert.Equal(t, 0, newPQ.Size())
	// The writeIndex should be now set accordingly
	require.EqualValues(t, 6, newPQ.writeIndex)

	// There should be no items left in the storage
	for i := 0; i < int(newPQ.writeIndex); i++ {
		bb, err := client.Get(context.Background(), getItemKey(uint64(i)))
		require.NoError(t, err)
		require.Nil(t, bb)
	}
}

// this test attempts to check if all the invariants are kept if the queue is recreated while
// close to full and with some items dispatched
func TestPersistentQueue_StartWithNonDispatched(t *testing.T) {
	req := newTraces(5, 10)

	ext := NewMockStorageExtension(NewMockStorageClient())
	host := &mockHost{ext: map[component.ID]component.Component{{}: ext}}
	pq := createNopPersistentQueueWithHost(t, 5, host)

	// Put in items up to capacity
	for i := 0; i < 5; i++ {
		err := pq.Offer(context.Background(), req)
		require.NoError(t, err)
	}

	// get one item out, but don't mark it as processed
	_, _ = pq.Poll()
	// put one more item in
	require.NoError(t, pq.Offer(context.Background(), req))

	require.Equal(t, 5, pq.Size())
	assert.NoError(t, pq.Shutdown(context.Background()))

	// Reload
	newPQ := createNopPersistentQueueWithHost(t, 5, host)
	require.Equal(t, 5, newPQ.Size())
}

func TestPersistentQueue_PutCloseReadClose(t *testing.T) {
	req := newTraces(5, 10)
	ext := NewMockStorageExtension(NewMockStorageClient())
	host := &mockHost{ext: map[component.ID]component.Component{{}: ext}}
	pq := createNopPersistentQueueWithHost(t, 1000, host)
	assert.Equal(t, 0, pq.Size())

	// Put two elements and close the extension
	assert.NoError(t, pq.Offer(context.Background(), req))
	assert.NoError(t, pq.Offer(context.Background(), req))
	assert.Equal(t, 2, pq.Size())
	// TODO: Remove this, after the initialization writes the readIndex.
	_, _ = pq.Poll()
	assert.NoError(t, pq.Shutdown(context.Background()))

	newPQ := createNopPersistentQueueWithHost(t, 1000, host)
	require.Equal(t, 2, newPQ.Size())

	// Lets read both of the elements we put
	readReq, found := newPQ.Poll()
	require.True(t, found)
	require.Equal(t, req, readReq.Request)

	readReq, found = newPQ.Poll()
	require.True(t, found)
	require.Equal(t, req, readReq.Request)
	require.Equal(t, 0, newPQ.Size())
	assert.NoError(t, newPQ.Shutdown(context.Background()))
}

func BenchmarkPersistentStorage_TraceSpans(b *testing.B) {
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
			host := &mockHost{ext: map[component.ID]component.Component{{}: ext}}
			pq := createNopPersistentQueueWithHost(b, 10000000, host)

			req := newTraces(c.numTraces, c.numSpansPerTrace)

			bb.ResetTimer()

			for i := 0; i < bb.N; i++ {
				require.NoError(bb, pq.Offer(context.Background(), req))
			}

			for i := 0; i < bb.N; i++ {
				req, found := pq.Poll()
				require.True(bb, found)
				require.NotNil(bb, req)
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
			out: []uint64{},
		},
		{
			in:  nil,
			out: []uint64{},
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
	client := NewMockStorageClient().(*mockStorageClient)
	host := &mockHost{ext: map[component.ID]component.Component{{}: NewMockStorageExtension(client)}}
	pq := createNopPersistentQueueWithHost(t, 10000000, host)

	assert.Equal(t, 0, pq.Size())
	assert.False(t, client.isClosed())

	assert.NoError(t, pq.Offer(context.Background(), newTraces(5, 10)))

	req, ok := pq.Poll()
	require.True(t, ok)
	assert.False(t, client.isClosed())
	assert.NoError(t, pq.Shutdown(context.Background()))
	assert.False(t, client.isClosed())
	req.OnProcessingFinished()
	assert.True(t, client.isClosed())
}

func TestPersistentQueue_StorageFull(t *testing.T) {
	req := newTraces(5, 10)
	marshaled, err := marshaler.MarshalTraces(req)
	require.NoError(t, err)
	maxSizeInBytes := len(marshaled) * 5 // arbitrary small number
	freeSpaceInBytes := 1

	client := newFakeBoundedStorageClient(maxSizeInBytes)
	host := &mockHost{ext: map[component.ID]component.Component{{}: NewMockStorageExtension(client)}}
	pq := createNopPersistentQueueWithHost(t, 1000, host)

	// Put enough items in to fill the underlying storage
	reqCount := 0
	for {
		err = pq.Offer(context.Background(), req)
		if errors.Is(err, syscall.ENOSPC) {
			break
		}
		require.NoError(t, err)
		reqCount++
	}

	// Manually set the storage to only have a small amount of free space left
	newMaxSize := client.GetSizeInBytes() + freeSpaceInBytes
	client.SetMaxSizeInBytes(newMaxSize)

	// Try to put an item in, should fail
	require.Error(t, pq.Offer(context.Background(), req))

	// Take out all the items
	for i := reqCount; i > 0; i-- {
		request, found := pq.Poll()
		require.True(t, found)
		request.OnProcessingFinished()
	}

	// We should be able to put a new item in
	// However, this will fail if deleting items fails with full storage
	require.NoError(t, pq.Offer(context.Background(), req))
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
			host := &mockHost{ext: map[component.ID]component.Component{{}: NewMockStorageExtension(client)}}
			pq := createNopPersistentQueueWithHost(t, 1000, host)
			client.Reset()

			err := pq.itemDispatchingFinish(context.Background(), 0)

			require.ErrorIs(t, err, testCase.expectedError)
		})
	}
}

func requireCurrentlyDispatchedItemsEqual(t *testing.T, pcs *persistentQueue[ptrace.Traces], compare []uint64) {
	pcs.mu.Lock()
	defer pcs.mu.Unlock()
	assert.ElementsMatch(t, compare, pcs.currentlyDispatchedItems)
}
