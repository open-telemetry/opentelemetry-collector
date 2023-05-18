// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/experimental/storage"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func createStorageExtension(_ string) storage.Extension {
	// After having storage moved to core, we could leverage storagetest.NewTestExtension(nil, path)
	return newMockStorageExtension()
}

func createTestClient(extension storage.Extension) storage.Client {
	client, err := extension.GetClient(context.Background(), component.KindReceiver, component.ID{}, "")
	if err != nil {
		panic(err)
	}
	return client
}

func createTestPersistentStorageWithLoggingAndCapacity(client storage.Client, logger *zap.Logger, capacity uint64) *persistentContiguousStorage {
	return newPersistentContiguousStorage(context.Background(), "foo", capacity, logger, client, newFakeTracesRequestUnmarshalerFunc())
}

func createTestPersistentStorage(client storage.Client) *persistentContiguousStorage {
	logger := zap.NewNop()
	return createTestPersistentStorageWithLoggingAndCapacity(client, logger, 1000)
}

type fakeTracesRequest struct {
	td                         ptrace.Traces
	processingFinishedCallback func()
	Request
}

func newFakeTracesRequest(td ptrace.Traces) *fakeTracesRequest {
	return &fakeTracesRequest{
		td: td,
	}
}

func (fd *fakeTracesRequest) Marshal() ([]byte, error) {
	marshaler := &ptrace.ProtoMarshaler{}
	return marshaler.MarshalTraces(fd.td)
}

func (fd *fakeTracesRequest) OnProcessingFinished() {
	if fd.processingFinishedCallback != nil {
		fd.processingFinishedCallback()
	}
}

func (fd *fakeTracesRequest) SetOnProcessingFinished(callback func()) {
	fd.processingFinishedCallback = callback
}

func newFakeTracesRequestUnmarshalerFunc() RequestUnmarshaler {
	return func(bytes []byte) (Request, error) {
		unmarshaler := ptrace.ProtoUnmarshaler{}
		traces, err := unmarshaler.UnmarshalTraces(bytes)
		if err != nil {
			return nil, err
		}
		return newFakeTracesRequest(traces), nil
	}
}

func TestPersistentStorage_CorruptedData(t *testing.T) {
	path := t.TempDir()

	traces := newTraces(5, 10)
	req := newFakeTracesRequest(traces)

	cases := []struct {
		name                               string
		corruptAllData                     bool
		corruptSomeData                    bool
		corruptCurrentlyDispatchedItemsKey bool
		corruptReadIndex                   bool
		corruptWriteIndex                  bool
		desiredQueueSize                   uint64
		desiredNumberOfDispatchedItems     int
	}{
		{
			name:                           "corrupted no items",
			corruptAllData:                 false,
			desiredQueueSize:               2,
			desiredNumberOfDispatchedItems: 1,
		},
		{
			name:                           "corrupted all items",
			corruptAllData:                 true,
			desiredQueueSize:               0,
			desiredNumberOfDispatchedItems: 0,
		},
		{
			name:                           "corrupted some items",
			corruptSomeData:                true,
			desiredQueueSize:               1,
			desiredNumberOfDispatchedItems: 1,
		},
		{
			name:                               "corrupted dispatched items key",
			corruptCurrentlyDispatchedItemsKey: true,
			desiredQueueSize:                   1,
			desiredNumberOfDispatchedItems:     1,
		},
		{
			name:                           "corrupted read index",
			corruptReadIndex:               true,
			desiredQueueSize:               0,
			desiredNumberOfDispatchedItems: 1,
		},
		{
			name:                           "corrupted write index",
			corruptWriteIndex:              true,
			desiredQueueSize:               0,
			desiredNumberOfDispatchedItems: 1,
		},
		{
			name:                               "corrupted everything",
			corruptAllData:                     true,
			corruptCurrentlyDispatchedItemsKey: true,
			corruptReadIndex:                   true,
			corruptWriteIndex:                  true,
			desiredQueueSize:                   0,
			desiredNumberOfDispatchedItems:     0,
		},
	}

	badBytes := []byte{0, 1, 2}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ext := createStorageExtension(path)
			client := createTestClient(ext)
			ps := createTestPersistentStorage(client)

			ctx := context.Background()

			// Put some items, make sure they are loaded and shutdown the storage...
			for i := 0; i < 3; i++ {
				err := ps.put(req)
				require.NoError(t, err)
			}
			require.Eventually(t, func() bool {
				return ps.size() == 2
			}, 5*time.Second, 10*time.Millisecond)
			ps.stop()

			// ... so now we can corrupt data (in several ways)
			if c.corruptAllData || c.corruptSomeData {
				_ = client.Set(ctx, "0", badBytes)
			}
			if c.corruptAllData {
				_ = client.Set(ctx, "1", badBytes)
				_ = client.Set(ctx, "2", badBytes)
			}

			if c.corruptCurrentlyDispatchedItemsKey {
				_ = client.Set(ctx, currentlyDispatchedItemsKey, badBytes)
			}

			if c.corruptReadIndex {
				_ = client.Set(ctx, readIndexKey, badBytes)
			}

			if c.corruptWriteIndex {
				_ = client.Set(ctx, writeIndexKey, badBytes)
			}

			// Reload
			newPs := createTestPersistentStorage(client)

			require.Eventually(t, func() bool {
				newPs.mu.Lock()
				defer newPs.mu.Unlock()
				return newPs.size() == c.desiredQueueSize && len(newPs.currentlyDispatchedItems) == c.desiredNumberOfDispatchedItems
			}, 5*time.Second, 10*time.Millisecond)
		})
	}
}

func TestPersistentStorage_CurrentlyProcessedItems(t *testing.T) {
	path := t.TempDir()

	traces := newTraces(5, 10)
	req := newFakeTracesRequest(traces)

	ext := createStorageExtension(path)
	client := createTestClient(ext)
	ps := createTestPersistentStorage(client)

	for i := 0; i < 5; i++ {
		err := ps.put(req)
		require.NoError(t, err)
	}

	// Item index 0 is currently in unbuffered channel
	requireCurrentlyDispatchedItemsEqual(t, ps, []itemIndex{0})

	// Now, this will take item 0 and pull item 1 into the unbuffered channel
	readReq := <-ps.get()
	assert.Equal(t, req.td, readReq.(*fakeTracesRequest).td)
	requireCurrentlyDispatchedItemsEqual(t, ps, []itemIndex{0, 1})

	// This takes item 1 from channel and pulls another one (item 2) into the unbuffered channel
	secondReadReq := <-ps.get()
	requireCurrentlyDispatchedItemsEqual(t, ps, []itemIndex{0, 1, 2})

	// Lets mark item 1 as finished, it will remove it from the currently dispatched items list
	secondReadReq.OnProcessingFinished()
	requireCurrentlyDispatchedItemsEqual(t, ps, []itemIndex{0, 2})

	// Reload the storage. Since items 0 and 2 were not finished, those should be requeued at the end.
	// The queue should be essentially {3,4,0,2} out of which item "3" should be pulled right away into
	// the unbuffered channel. Check how many items are there, which, after the current one is fetched should go to 3.
	newPs := createTestPersistentStorage(client)
	assert.Eventually(t, func() bool {
		return newPs.size() == 3
	}, 5*time.Second, 10*time.Millisecond)

	requireCurrentlyDispatchedItemsEqual(t, newPs, []itemIndex{3})

	// We should be able to pull all remaining items now
	for i := 0; i < 4; i++ {
		req := <-newPs.get()
		req.OnProcessingFinished()
	}

	// The queue should be now empty
	requireCurrentlyDispatchedItemsEqual(t, newPs, nil)
	assert.Eventually(t, func() bool {
		return newPs.size() == 0
	}, 5*time.Second, 10*time.Millisecond)

	// The writeIndex should be now set accordingly
	require.Equal(t, 7, int(newPs.writeIndex))

	// There should be no items left in the storage
	for i := 0; i < int(newPs.writeIndex); i++ {
		bb, err := client.Get(context.Background(), newPs.itemKey(itemIndex(i)))
		require.NoError(t, err)
		require.Nil(t, bb)
	}
}

// this test attempts to check if all the invariants are kept if the queue is recreated while
// close to full and with some items dispatched
func TestPersistentStorage_StartWithNonDispatched(t *testing.T) {
	var capacity uint64 = 5 // arbitrary small number
	path := t.TempDir()
	logger := zap.NewNop()

	traces := newTraces(5, 10)
	req := newFakeTracesRequest(traces)

	ext := createStorageExtension(path)
	client := createTestClient(ext)
	ps := createTestPersistentStorageWithLoggingAndCapacity(client, logger, capacity)

	// Put in items up to capacity
	for i := 0; i < int(capacity); i++ {
		err := ps.put(req)
		require.NoError(t, err)
	}

	// get one item out, but don't mark it as processed
	<-ps.get()
	// put one more item in
	err := ps.put(req)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return ps.size() == capacity-1
	}, 5*time.Second, 10*time.Millisecond)
	ps.stop()

	// Reload
	newPs := createTestPersistentStorageWithLoggingAndCapacity(client, logger, capacity)

	require.Eventually(t, func() bool {
		newPs.mu.Lock()
		defer newPs.mu.Unlock()
		return newPs.size() == capacity-1 && len(newPs.currentlyDispatchedItems) == 1
	}, 5*time.Second, 10*time.Millisecond)
}

func TestPersistentStorage_RepeatPutCloseReadClose(t *testing.T) {
	path := t.TempDir()

	traces := newTraces(5, 10)
	req := newFakeTracesRequest(traces)

	for i := 0; i < 10; i++ {
		ext := createStorageExtension(path)
		client := createTestClient(ext)
		ps := createTestPersistentStorage(client)
		require.Equal(t, uint64(0), ps.size())

		// Put two elements
		err := ps.put(req)
		require.NoError(t, err)
		err = ps.put(req)
		require.NoError(t, err)

		err = ext.Shutdown(context.Background())
		require.NoError(t, err)

		// TODO: when replacing mock with real storage, this could actually be uncommented
		// ext = createStorageExtension(path)
		// ps = createTestPersistentStorage(ext)

		// The first element should be already picked by loop
		require.Eventually(t, func() bool {
			return ps.size() == 1
		}, 5*time.Second, 10*time.Millisecond)

		// Lets read both of the elements we put
		readReq := <-ps.get()
		require.Equal(t, req.td, readReq.(*fakeTracesRequest).td)

		readReq = <-ps.get()
		require.Equal(t, req.td, readReq.(*fakeTracesRequest).td)
		require.Equal(t, uint64(0), ps.size())

		err = ext.Shutdown(context.Background())
		require.NoError(t, err)
	}

	// No more items
	ext := createStorageExtension(path)
	wq := createTestQueue(ext, 1000)
	require.Equal(t, 0, wq.Size())
	require.NoError(t, ext.Shutdown(context.Background()))
}

func TestPersistentStorage_EmptyRequest(t *testing.T) {
	path := t.TempDir()

	ext := createStorageExtension(path)
	client := createTestClient(ext)
	ps := createTestPersistentStorage(client)

	require.Equal(t, uint64(0), ps.size())

	err := ps.put(nil)
	require.NoError(t, err)

	require.Equal(t, uint64(0), ps.size())

	err = ext.Shutdown(context.Background())
	require.NoError(t, err)
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
			path := bb.TempDir()
			ext := createStorageExtension(path)
			client := createTestClient(ext)
			ps := createTestPersistentStorageWithLoggingAndCapacity(client, zap.NewNop(), 10000000)

			traces := newTraces(c.numTraces, c.numSpansPerTrace)
			req := newFakeTracesRequest(traces)

			bb.ResetTimer()

			for i := 0; i < bb.N; i++ {
				err := ps.put(req)
				require.NoError(bb, err)
			}

			for i := 0; i < bb.N; i++ {
				req := ps.get()
				require.NotNil(bb, req)
			}
			require.NoError(b, ext.Shutdown(context.Background()))
		})
	}
}

func TestPersistentStorage_ItemIndexMarshaling(t *testing.T) {
	cases := []struct {
		arr1 []itemIndex
		arr2 []itemIndex
	}{
		{
			arr1: []itemIndex{0, 1, 2},
			arr2: []itemIndex{0, 1, 2},
		},
		{
			arr1: []itemIndex{},
			arr2: []itemIndex{},
		},
		{
			arr1: nil,
			arr2: []itemIndex{},
		},
	}

	for _, c := range cases {
		count := 0
		if c.arr1 != nil {
			count = len(c.arr1)
		}
		t.Run(fmt.Sprintf("#elements:%d", count), func(tt *testing.T) {
			barr, err := itemIndexArrayToBytes(c.arr1)
			require.NoError(t, err)
			arr2, err := bytesToItemIndexArray(barr)
			require.NoError(t, err)
			require.Equal(t, c.arr2, arr2)
		})
	}
}

func TestPersistentStorage_StopShouldCloseClient(t *testing.T) {
	path := t.TempDir()

	ext := createStorageExtension(path)
	client := createTestClient(ext)
	ps := createTestPersistentStorage(client)

	ps.stop()

	castedClient, ok := client.(*mockStorageClient)
	require.True(t, ok, "expected client to be mockStorageClient")
	require.Equal(t, uint64(1), castedClient.getCloseCount())
}

func TestPersistentStorage_StorageFull(t *testing.T) {
	var err error
	traces := newTraces(5, 10)
	req := newFakeTracesRequest(traces)
	marshaled, err := req.Marshal()
	require.NoError(t, err)
	maxSizeInBytes := len(marshaled) * 5 // arbitrary small number
	freeSpaceInBytes := 1

	client := newFakeBoundedStorageClient(maxSizeInBytes)
	ps := createTestPersistentStorage(client)

	// Put enough items in to fill the underlying storage
	reqCount := 0
	for {
		err = ps.put(req)
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
	err = ps.put(req)
	require.Error(t, err)

	// Take out all the items
	for i := reqCount; i > 0; i-- {
		request := <-ps.get()
		request.OnProcessingFinished()
	}

	// We should be able to put a new item in
	// However, this will fail if deleting items fails with full storage
	err = ps.put(req)
	require.NoError(t, err)
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
			testCase := testCase
			client := newFakeStorageClientWithErrors(testCase.storageErrors)
			ps := createTestPersistentStorage(client)
			client.Reset()

			err := ps.itemDispatchingFinish(context.Background(), 0)

			require.ErrorIs(t, err, testCase.expectedError)
		})
	}
}

func requireCurrentlyDispatchedItemsEqual(t *testing.T, pcs *persistentContiguousStorage, compare []itemIndex) {
	require.Eventually(t, func() bool {
		pcs.mu.Lock()
		defer pcs.mu.Unlock()
		return reflect.DeepEqual(pcs.currentlyDispatchedItems, compare)
	}, 5*time.Second, 10*time.Millisecond)
}

type mockStorageExtension struct {
	component.StartFunc
	component.ShutdownFunc
}

func (m mockStorageExtension) GetClient(_ context.Context, _ component.Kind, _ component.ID, _ string) (storage.Client, error) {
	return &mockStorageClient{st: map[string][]byte{}}, nil
}

func newMockStorageExtension() storage.Extension {
	return &mockStorageExtension{}
}

type mockStorageClient struct {
	st           map[string][]byte
	mux          sync.Mutex
	closeCounter uint64
}

func (m *mockStorageClient) Get(_ context.Context, s string) ([]byte, error) {
	m.mux.Lock()
	defer m.mux.Unlock()

	val, found := m.st[s]
	if !found {
		return nil, nil
	}

	return val, nil
}

func (m *mockStorageClient) Set(_ context.Context, s string, bytes []byte) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.st[s] = bytes
	return nil
}

func (m *mockStorageClient) Delete(_ context.Context, s string) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	delete(m.st, s)
	return nil
}

func (m *mockStorageClient) Close(_ context.Context) error {
	m.closeCounter++
	return nil
}

func (m *mockStorageClient) Batch(_ context.Context, ops ...storage.Operation) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	for _, op := range ops {
		switch op.Type {
		case storage.Get:
			op.Value = m.st[op.Key]
		case storage.Set:
			m.st[op.Key] = op.Value
		case storage.Delete:
			delete(m.st, op.Key)
		default:
			return errors.New("wrong operation type")
		}
	}

	return nil
}

func (m *mockStorageClient) getCloseCount() uint64 {
	return m.closeCounter
}

func newFakeBoundedStorageClient(maxSizeInBytes int) *fakeBoundedStorageClient {
	return &fakeBoundedStorageClient{
		st:             map[string][]byte{},
		MaxSizeInBytes: maxSizeInBytes,
	}
}

// this storage client mimics the behavior of actual storage engines with limited storage space available
// in general, real storage engines often have a per-write-transaction storage overhead, needing to keep
// both the old and the new value stored until the transaction is committed
// this is useful for testing the persistent queue queue behavior with a full disk
type fakeBoundedStorageClient struct {
	MaxSizeInBytes int
	st             map[string][]byte
	sizeInBytes    int
	mux            sync.Mutex
}

func (m *fakeBoundedStorageClient) Get(ctx context.Context, key string) ([]byte, error) {
	op := storage.GetOperation(key)
	err := m.Batch(ctx, op)
	if err != nil {
		return nil, err
	}

	return op.Value, nil
}

func (m *fakeBoundedStorageClient) Set(ctx context.Context, key string, value []byte) error {
	return m.Batch(ctx, storage.SetOperation(key, value))
}

func (m *fakeBoundedStorageClient) Delete(ctx context.Context, key string) error {
	return m.Batch(ctx, storage.DeleteOperation(key))
}

func (m *fakeBoundedStorageClient) Close(_ context.Context) error {
	return nil
}

func (m *fakeBoundedStorageClient) Batch(_ context.Context, ops ...storage.Operation) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	totalAdded, totalRemoved := m.getTotalSizeChange(ops)

	// the assumption here is that the new data needs to coexist with the old data on disk
	// for the transaction to succeed
	// this seems to be true for the file storage extension at least
	if m.sizeInBytes+totalAdded > m.MaxSizeInBytes {
		return fmt.Errorf("insufficient space available: %w", syscall.ENOSPC)
	}

	for _, op := range ops {
		switch op.Type {
		case storage.Get:
			op.Value = m.st[op.Key]
		case storage.Set:
			m.st[op.Key] = op.Value
		case storage.Delete:
			delete(m.st, op.Key)
		default:
			return errors.New("wrong operation type")
		}
	}

	m.sizeInBytes += (totalAdded - totalRemoved)

	return nil
}

func (m *fakeBoundedStorageClient) SetMaxSizeInBytes(newMaxSize int) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.MaxSizeInBytes = newMaxSize
}

func (m *fakeBoundedStorageClient) GetSizeInBytes() int {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.sizeInBytes
}

func (m *fakeBoundedStorageClient) getTotalSizeChange(ops []storage.Operation) (totalAdded int, totalRemoved int) {
	totalAdded, totalRemoved = 0, 0
	for _, op := range ops {
		switch op.Type {
		case storage.Set:
			if oldValue, ok := m.st[op.Key]; ok {
				totalRemoved += len(oldValue)
			} else {
				totalAdded += len(op.Key)
			}
			totalAdded += len(op.Value)
		case storage.Delete:
			if value, ok := m.st[op.Key]; ok {
				totalRemoved += len(op.Key)
				totalRemoved += len(value)
			}
		default:
		}
	}
	return totalAdded, totalRemoved
}

func newFakeStorageClientWithErrors(errors []error) *fakeStorageClientWithErrors {
	return &fakeStorageClientWithErrors{
		errors: errors,
	}
}

// this storage client just returns errors from a list in order
// used for testing error handling
type fakeStorageClientWithErrors struct {
	errors         []error
	nextErrorIndex int
	mux            sync.Mutex
}

func (m *fakeStorageClientWithErrors) Get(ctx context.Context, key string) ([]byte, error) {
	op := storage.GetOperation(key)
	err := m.Batch(ctx, op)
	if err != nil {
		return nil, err
	}

	return op.Value, nil
}

func (m *fakeStorageClientWithErrors) Set(ctx context.Context, key string, value []byte) error {
	return m.Batch(ctx, storage.SetOperation(key, value))
}

func (m *fakeStorageClientWithErrors) Delete(ctx context.Context, key string) error {
	return m.Batch(ctx, storage.DeleteOperation(key))
}

func (m *fakeStorageClientWithErrors) Close(_ context.Context) error {
	return nil
}

func (m *fakeStorageClientWithErrors) Batch(_ context.Context, _ ...storage.Operation) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	if m.nextErrorIndex >= len(m.errors) {
		return nil
	}

	err := m.errors[m.nextErrorIndex]
	m.nextErrorIndex++
	return err
}

func (m *fakeStorageClientWithErrors) Reset() {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.nextErrorIndex = 0
}
