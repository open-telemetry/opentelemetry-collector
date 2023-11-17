// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/extension/experimental/storage"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var (
	marshaler   = ptrace.ProtoMarshaler{}
	unmarshaler = ptrace.ProtoUnmarshaler{}
)

func createTestClient(t testing.TB, extension storage.Extension) storage.Client {
	client, err := extension.GetClient(context.Background(), component.KindReceiver, component.ID{}, "")
	require.NoError(t, err)
	return client
}

func createTestPersistentStorageWithCapacity(client storage.Client, capacity int) *persistentQueue[ptrace.Traces] {
	pq := NewPersistentQueue[ptrace.Traces](capacity, component.DataTypeTraces, component.ID{}, marshaler.MarshalTraces,
		unmarshaler.UnmarshalTraces, exportertest.NewNopCreateSettings()).(*persistentQueue[ptrace.Traces])
	pq.initClient(context.Background(), client)
	return pq
}

func createTestPersistentStorage(client storage.Client) *persistentQueue[ptrace.Traces] {
	return createTestPersistentStorageWithCapacity(client, 1000)
}

func TestPersistentStorage_CorruptedData(t *testing.T) {
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
			client := createTestClient(t, ext)
			ps := createTestPersistentStorage(client)

			ctx := context.Background()

			// Put some items, make sure they are loaded and shutdown the storage...
			for i := 0; i < 3; i++ {
				err := ps.Offer(context.Background(), req)
				require.NoError(t, err)
			}
			assert.Equal(t, 3, ps.Size())
			_, _ = ps.Poll()
			assert.Equal(t, 2, ps.Size())
			assert.NoError(t, ps.Shutdown(context.Background()))

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
			newPs := createTestPersistentStorage(client)
			assert.Equal(t, c.desiredQueueSize, newPs.Size())
		})
	}
}

func TestPersistentStorage_CurrentlyProcessedItems(t *testing.T) {
	req := newTraces(5, 10)

	ext := NewMockStorageExtension(nil)
	client := createTestClient(t, ext)
	ps := createTestPersistentStorage(client)

	for i := 0; i < 5; i++ {
		err := ps.Offer(context.Background(), req)
		require.NoError(t, err)
	}

	requireCurrentlyDispatchedItemsEqual(t, ps, []uint64{})

	// Takes index 0 in process.
	readReq, found := ps.Poll()
	require.True(t, found)
	assert.Equal(t, req, readReq.Request)
	requireCurrentlyDispatchedItemsEqual(t, ps, []uint64{0})

	// This takes item 1 to process.
	secondReadReq, found := ps.Poll()
	require.True(t, found)
	requireCurrentlyDispatchedItemsEqual(t, ps, []uint64{0, 1})

	// Lets mark item 1 as finished, it will remove it from the currently dispatched items list.
	secondReadReq.OnProcessingFinished()
	requireCurrentlyDispatchedItemsEqual(t, ps, []uint64{0})

	// Reload the storage. Since items 0 was not finished, this should be re-enqueued at the end.
	// The queue should be essentially {3,4,0,2}.
	newPs := createTestPersistentStorage(client)
	assert.Equal(t, 4, newPs.Size())
	requireCurrentlyDispatchedItemsEqual(t, newPs, []uint64{})

	// We should be able to pull all remaining items now
	for i := 0; i < 4; i++ {
		qReq, found := newPs.Poll()
		require.True(t, found)
		qReq.OnProcessingFinished()
	}

	// The queue should be now empty
	requireCurrentlyDispatchedItemsEqual(t, newPs, []uint64{})
	assert.Equal(t, 0, newPs.Size())
	// The writeIndex should be now set accordingly
	require.EqualValues(t, 6, newPs.writeIndex)

	// There should be no items left in the storage
	for i := 0; i < int(newPs.writeIndex); i++ {
		bb, err := client.Get(context.Background(), getItemKey(uint64(i)))
		require.NoError(t, err)
		require.Nil(t, bb)
	}
}

// this test attempts to check if all the invariants are kept if the queue is recreated while
// close to full and with some items dispatched
func TestPersistentStorage_StartWithNonDispatched(t *testing.T) {
	req := newTraces(5, 10)

	ext := NewMockStorageExtension(nil)
	client := createTestClient(t, ext)
	ps := createTestPersistentStorageWithCapacity(client, 5)

	// Put in items up to capacity
	for i := 0; i < 5; i++ {
		err := ps.Offer(context.Background(), req)
		require.NoError(t, err)
	}

	// get one item out, but don't mark it as processed
	_, _ = ps.Poll()
	// put one more item in
	require.NoError(t, ps.Offer(context.Background(), req))

	require.Equal(t, 5, ps.Size())
	assert.NoError(t, ps.Shutdown(context.Background()))

	// Reload
	newPs := createTestPersistentStorageWithCapacity(client, 5)
	require.Equal(t, 5, newPs.Size())
}

func TestPersistentStorage_PutCloseReadClose(t *testing.T) {
	req := newTraces(5, 10)
	ext := NewMockStorageExtension(nil)
	ps := createTestPersistentStorage(createTestClient(t, ext))
	assert.Equal(t, 0, ps.Size())

	// Put two elements and close the extension
	assert.NoError(t, ps.Offer(context.Background(), req))
	assert.NoError(t, ps.Offer(context.Background(), req))
	assert.Equal(t, 2, ps.Size())
	// TODO: Remove this, after the initialization writes the readIndex.
	_, _ = ps.Poll()
	assert.NoError(t, ps.Shutdown(context.Background()))

	newPs := createTestPersistentStorage(createTestClient(t, ext))
	require.Equal(t, 2, newPs.Size())

	// Lets read both of the elements we put
	readReq, found := newPs.Poll()
	require.True(t, found)
	require.Equal(t, req, readReq.Request)

	readReq, found = newPs.Poll()
	require.True(t, found)
	require.Equal(t, req, readReq.Request)
	require.Equal(t, 0, newPs.Size())
	assert.NoError(t, newPs.Shutdown(context.Background()))
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
			client := createTestClient(b, ext)
			ps := createTestPersistentStorageWithCapacity(client, 10000000)

			req := newTraces(c.numTraces, c.numSpansPerTrace)

			bb.ResetTimer()

			for i := 0; i < bb.N; i++ {
				require.NoError(bb, ps.Offer(context.Background(), req))
			}

			for i := 0; i < bb.N; i++ {
				req, found := ps.Poll()
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

func TestPersistentStorage_ShutdownWhileConsuming(t *testing.T) {
	client := createTestClient(t, NewMockStorageExtension(nil))
	ps := createTestPersistentStorage(client)

	assert.Equal(t, 0, ps.Size())
	assert.False(t, client.(*mockStorageClient).isClosed())

	assert.NoError(t, ps.Offer(context.Background(), newTraces(5, 10)))

	req, ok := ps.Poll()
	require.True(t, ok)
	assert.False(t, client.(*mockStorageClient).isClosed())
	assert.NoError(t, ps.Shutdown(context.Background()))
	assert.False(t, client.(*mockStorageClient).isClosed())
	req.OnProcessingFinished()
	assert.True(t, client.(*mockStorageClient).isClosed())
}

func TestPersistentStorage_StorageFull(t *testing.T) {
	req := newTraces(5, 10)
	marshaled, err := marshaler.MarshalTraces(req)
	require.NoError(t, err)
	maxSizeInBytes := len(marshaled) * 5 // arbitrary small number
	freeSpaceInBytes := 1

	client := newFakeBoundedStorageClient(maxSizeInBytes)
	ps := createTestPersistentStorage(client)

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

	// Manually set the storage to only have a small amount of free space left
	newMaxSize := client.GetSizeInBytes() + freeSpaceInBytes
	client.SetMaxSizeInBytes(newMaxSize)

	// Try to put an item in, should fail
	require.Error(t, ps.Offer(context.Background(), req))

	// Take out all the items
	for i := reqCount; i > 0; i-- {
		request, found := ps.Poll()
		require.True(t, found)
		request.OnProcessingFinished()
	}

	// We should be able to put a new item in
	// However, this will fail if deleting items fails with full storage
	require.NoError(t, ps.Offer(context.Background(), req))
}

func TestPersistentStorage_ItemDispatchingFinish_ErrorHandling(t *testing.T) {
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
			ps := createTestPersistentStorage(client)
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
