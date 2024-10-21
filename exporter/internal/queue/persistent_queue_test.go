// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/internal/experr"
	"go.opentelemetry.io/collector/extension/experimental/storage"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pipeline"
)

type itemsCounter interface {
	ItemsCount() int
}

// itemsSizer is a Sizer implementation that returns the size of a queue element as the number of items it contains.
type itemsSizer[T itemsCounter] struct{}

func (is *itemsSizer[T]) Sizeof(el T) int64 {
	return int64(el.ItemsCount())
}

type tracesRequest struct {
	traces ptrace.Traces
}

func (tr tracesRequest) ItemsCount() int {
	return tr.traces.SpanCount()
}

func marshalTracesRequest(tr tracesRequest) ([]byte, error) {
	marshaler := &ptrace.ProtoMarshaler{}
	return marshaler.MarshalTraces(tr.traces)
}

func unmarshalTracesRequest(bytes []byte) (tracesRequest, error) {
	unmarshaler := &ptrace.ProtoUnmarshaler{}
	traces, err := unmarshaler.UnmarshalTraces(bytes)
	return tracesRequest{traces: traces}, err
}

type mockHost struct {
	component.Host
	ext map[component.ID]component.Component
}

func (nh *mockHost) GetExtensions() map[component.ID]component.Component {
	return nh.ext
}

// createAndStartTestPersistentQueue creates and starts a fake queue with the given capacity and number of consumers.
func createAndStartTestPersistentQueue(t *testing.T, sizer Sizer[tracesRequest], capacity int64, numConsumers int,
	consumeFunc func(_ context.Context, item tracesRequest) error) Queue[tracesRequest] {
	pq := NewPersistentQueue[tracesRequest](PersistentQueueSettings[tracesRequest]{
		Sizer:            sizer,
		Capacity:         capacity,
		Signal:           pipeline.SignalTraces,
		StorageID:        component.ID{},
		Marshaler:        marshalTracesRequest,
		Unmarshaler:      unmarshalTracesRequest,
		ExporterSettings: exportertest.NewNopSettings(),
	})
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

func createTestPersistentQueueWithClient(client storage.Client) *persistentQueue[tracesRequest] {
	pq := NewPersistentQueue[tracesRequest](PersistentQueueSettings[tracesRequest]{
		Sizer:            &RequestSizer[tracesRequest]{},
		Capacity:         1000,
		Signal:           pipeline.SignalTraces,
		StorageID:        component.ID{},
		Marshaler:        marshalTracesRequest,
		Unmarshaler:      unmarshalTracesRequest,
		ExporterSettings: exportertest.NewNopSettings(),
	}).(*persistentQueue[tracesRequest])
	pq.initClient(context.Background(), client)
	return pq
}

func createTestPersistentQueueWithRequestsCapacity(t testing.TB, ext storage.Extension, capacity int64) *persistentQueue[tracesRequest] {
	return createTestPersistentQueueWithCapacityLimiter(t, ext, &RequestSizer[tracesRequest]{}, capacity)
}

func createTestPersistentQueueWithItemsCapacity(t testing.TB, ext storage.Extension, capacity int64) *persistentQueue[tracesRequest] {
	return createTestPersistentQueueWithCapacityLimiter(t, ext, &itemsSizer[tracesRequest]{}, capacity)
}

func createTestPersistentQueueWithCapacityLimiter(t testing.TB, ext storage.Extension, sizer Sizer[tracesRequest],
	capacity int64) *persistentQueue[tracesRequest] {
	pq := NewPersistentQueue[tracesRequest](PersistentQueueSettings[tracesRequest]{
		Sizer:            sizer,
		Capacity:         capacity,
		Signal:           pipeline.SignalTraces,
		StorageID:        component.ID{},
		Marshaler:        marshalTracesRequest,
		Unmarshaler:      unmarshalTracesRequest,
		ExporterSettings: exportertest.NewNopSettings(),
	}).(*persistentQueue[tracesRequest])
	require.NoError(t, pq.Start(context.Background(), &mockHost{ext: map[component.ID]component.Component{{}: ext}}))
	return pq
}

func TestPersistentQueue_FullCapacity(t *testing.T) {
	tests := []struct {
		name           string
		sizer          Sizer[tracesRequest]
		capacity       int64
		sizeMultiplier int
	}{
		{
			name:           "requests_capacity",
			sizer:          &RequestSizer[tracesRequest]{},
			capacity:       5,
			sizeMultiplier: 1,
		},
		{
			name:           "items_capacity",
			sizer:          &itemsSizer[tracesRequest]{},
			capacity:       55,
			sizeMultiplier: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := make(chan struct{})
			done := make(chan struct{})
			pq := createAndStartTestPersistentQueue(t, tt.sizer, tt.capacity, 1, func(context.Context, tracesRequest) error {
				<-start
				<-done
				return nil
			})
			assert.Equal(t, 0, pq.Size())

			req := newTracesRequest(1, 10)

			// First request is picked by the consumer. Wait until the consumer is blocked on done.
			require.NoError(t, pq.Offer(context.Background(), req))
			start <- struct{}{}
			close(start)

			for i := 0; i < 10; i++ {
				result := pq.Offer(context.Background(), newTracesRequest(1, 10))
				if i < 5 {
					require.NoError(t, result)
				} else {
					require.ErrorIs(t, result, ErrQueueIsFull)
				}
			}
			assert.Equal(t, 5*tt.sizeMultiplier, pq.Size())
			close(done)
		})
	}
}

func TestPersistentQueue_Shutdown(t *testing.T) {
	pq := createAndStartTestPersistentQueue(t, &RequestSizer[tracesRequest]{}, 1001, 100, func(context.Context,
		tracesRequest) error {
		return nil
	})
	req := newTracesRequest(1, 10)

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
			req := newTracesRequest(1, 10)

			numMessagesConsumed := &atomic.Int32{}
			pq := createAndStartTestPersistentQueue(t, &RequestSizer[tracesRequest]{}, 1000, c.numConsumers,
				func(context.Context,
					tracesRequest) error {
					numMessagesConsumed.Add(int32(1))
					return nil
				})

			for i := 0; i < c.numMessagesProduced; i++ {
				require.NoError(t, pq.Offer(context.Background(), req))
			}

			assert.Eventually(t, func() bool {
				return c.numMessagesProduced == int(numMessagesConsumed.Load())
			}, 5*time.Second, 10*time.Millisecond)
		})
	}
}

func newTracesRequest(numTraces int, numSpans int) tracesRequest {
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

	return tracesRequest{traces: traces}
}

func TestToStorageClient(t *testing.T) {
	getStorageClientError := errors.New("unable to create storage client")
	testCases := []struct {
		name           string
		storage        storage.Extension
		numStorages    int
		storageIndex   int
		expectedError  error
		getClientError error
	}{
		{
			name:          "obtain storage extension by name",
			numStorages:   2,
			storageIndex:  0,
			expectedError: nil,
		},
		{
			name:          "fail on not existing storage extension",
			numStorages:   2,
			storageIndex:  100,
			expectedError: errNoStorageClient,
		},
		{
			name:          "invalid extension type",
			numStorages:   2,
			storageIndex:  100,
			expectedError: errNoStorageClient,
		},
		{
			name:           "fail on error getting storage client from extension",
			numStorages:    1,
			storageIndex:   0,
			expectedError:  getStorageClientError,
			getClientError: getStorageClientError,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			storageID := component.MustNewIDWithName("file_storage", strconv.Itoa(tt.storageIndex))

			var extensions = map[component.ID]component.Component{}
			for i := 0; i < tt.numStorages; i++ {
				extensions[component.MustNewIDWithName("file_storage", strconv.Itoa(i))] = NewMockStorageExtension(tt.getClientError)
			}
			host := &mockHost{ext: extensions}
			ownerID := component.MustNewID("foo_exporter")

			// execute
			client, err := toStorageClient(context.Background(), storageID, host, ownerID, pipeline.SignalTraces)

			// verify
			if tt.expectedError != nil {
				require.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestInvalidStorageExtensionType(t *testing.T) {
	storageID := component.MustNewIDWithName("extension", "extension")

	// make a test extension
	factory := extensiontest.NewNopFactory()
	extConfig := factory.CreateDefaultConfig()
	settings := extensiontest.NewNopSettings()
	extension, err := factory.Create(context.Background(), settings, extConfig)
	require.NoError(t, err)
	var extensions = map[component.ID]component.Component{
		storageID: extension,
	}
	host := &mockHost{ext: extensions}
	ownerID := component.MustNewID("foo_exporter")

	// execute
	client, err := toStorageClient(context.Background(), storageID, host, ownerID, pipeline.SignalTraces)

	// we should get an error about the extension type
	require.ErrorIs(t, err, errWrongExtensionType)
	assert.Nil(t, client)
}

func TestPersistentQueue_StopAfterBadStart(t *testing.T) {
	pq := NewPersistentQueue[tracesRequest](PersistentQueueSettings[tracesRequest]{})
	// verify that stopping a un-start/started w/error queue does not panic
	assert.NoError(t, pq.Shutdown(context.Background()))
}

func TestPersistentQueue_CorruptedData(t *testing.T) {
	req := newTracesRequest(5, 10)

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
			ps := createTestPersistentQueueWithRequestsCapacity(t, ext, 1000)

			// Put some items, make sure they are loaded and shutdown the storage...
			for i := 0; i < 3; i++ {
				err := ps.Offer(context.Background(), req)
				require.NoError(t, err)
			}
			assert.Equal(t, 3, ps.Size())
			require.True(t, consume(ps, func(context.Context, tracesRequest) error {
				return experr.NewShutdownErr(nil)
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
			require.NoError(t, ps.Shutdown(context.Background()))

			// Reload
			newPs := createTestPersistentQueueWithRequestsCapacity(t, ext, 1000)
			assert.Equal(t, c.desiredQueueSize, newPs.Size())
		})
	}
}

func TestPersistentQueue_CurrentlyProcessedItems(t *testing.T) {
	req := newTracesRequest(5, 10)

	ext := NewMockStorageExtension(nil)
	ps := createTestPersistentQueueWithRequestsCapacity(t, ext, 1000)

	for i := 0; i < 5; i++ {
		err := ps.Offer(context.Background(), req)
		require.NoError(t, err)
	}

	requireCurrentlyDispatchedItemsEqual(t, ps, []uint64{})

	// Takes index 0 in process.
	_, readReq, found := ps.getNextItem(context.Background())
	require.True(t, found)
	assert.Equal(t, req, readReq)
	requireCurrentlyDispatchedItemsEqual(t, ps, []uint64{0})

	// This takes item 1 to process.
	secondIndex, secondReadReq, found := ps.getNextItem(context.Background())
	require.True(t, found)
	assert.Equal(t, req, secondReadReq)
	requireCurrentlyDispatchedItemsEqual(t, ps, []uint64{0, 1})

	// Lets mark item 1 as finished, it will remove it from the currently dispatched items list.
	ps.OnProcessingFinished(secondIndex, nil)
	requireCurrentlyDispatchedItemsEqual(t, ps, []uint64{0})

	// Reload the storage. Since items 0 was not finished, this should be re-enqueued at the end.
	// The queue should be essentially {3,4,0,2}.
	newPs := createTestPersistentQueueWithRequestsCapacity(t, ext, 1000)
	assert.Equal(t, 4, newPs.Size())
	requireCurrentlyDispatchedItemsEqual(t, newPs, []uint64{})

	// We should be able to pull all remaining items now
	for i := 0; i < 4; i++ {
		consume(newPs, func(_ context.Context, traces tracesRequest) error {
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
	for i := uint64(0); i < newPs.writeIndex; i++ {
		bb, err := newPs.client.Get(context.Background(), getItemKey(i))
		require.NoError(t, err)
		require.Nil(t, bb)
	}
}

// this test attempts to check if all the invariants are kept if the queue is recreated while
// close to full and with some items dispatched
func TestPersistentQueueStartWithNonDispatched(t *testing.T) {
	req := newTracesRequest(5, 10)

	ext := NewMockStorageExtension(nil)
	ps := createTestPersistentQueueWithRequestsCapacity(t, ext, 5)

	// Put in items up to capacity
	for i := 0; i < 5; i++ {
		err := ps.Offer(context.Background(), req)
		require.NoError(t, err)
	}

	require.True(t, consume(ps, func(context.Context, tracesRequest) error {
		// put one more item in
		require.NoError(t, ps.Offer(context.Background(), req))
		require.Equal(t, 5, ps.Size())
		return experr.NewShutdownErr(nil)
	}))
	require.NoError(t, ps.Shutdown(context.Background()))

	// Reload with extra capacity to make sure we re-enqueue in-progress items.
	newPs := createTestPersistentQueueWithRequestsCapacity(t, ext, 6)
	require.Equal(t, 6, newPs.Size())
}

func TestPersistentQueueStartWithNonDispatchedConcurrent(t *testing.T) {
	req := newTracesRequest(1, 1)

	ext := NewMockStorageExtensionWithDelay(nil, 20*time.Nanosecond)
	pq := createTestPersistentQueueWithItemsCapacity(t, ext, 25)

	proWg := sync.WaitGroup{}
	// Sending small amount of data as windows test can't handle the test fast enough
	for j := 0; j < 5; j++ {
		proWg.Add(1)
		go func() {
			defer proWg.Done()
			// Put in items up to capacity
			for i := 0; i < 10; i++ {
				for {
					// retry infinitely so the exact amount of items are added to the queue eventually
					if err := pq.Offer(context.Background(), req); err == nil {
						break
					}
					time.Sleep(50 * time.Nanosecond)
				}
			}
		}()
	}

	conWg := sync.WaitGroup{}
	for j := 0; j < 5; j++ {
		conWg.Add(1)
		go func() {
			defer conWg.Done()
			for i := 0; i < 10; i++ {
				assert.True(t, consume(pq, func(context.Context, tracesRequest) error { return nil }))
			}
		}()
	}

	conDone := make(chan struct{})
	go func() {
		defer close(conDone)
		conWg.Wait()
	}()

	proDone := make(chan struct{})
	go func() {
		defer close(proDone)
		proWg.Wait()
	}()

	doneCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	select {
	case <-conDone:
	case <-doneCtx.Done():
		assert.Fail(t, "timed out waiting for consumers to complete")
	}

	select {
	case <-proDone:
	case <-doneCtx.Done():
		assert.Fail(t, "timed out waiting for producers to complete")
	}
	assert.Zero(t, pq.sizedChannel.Size())
}

func TestPersistentQueue_PutCloseReadClose(t *testing.T) {
	req := newTracesRequest(5, 10)
	ext := NewMockStorageExtension(nil)
	ps := createTestPersistentQueueWithRequestsCapacity(t, ext, 1000)
	assert.Equal(t, 0, ps.Size())

	// Put two elements and close the extension
	assert.NoError(t, ps.Offer(context.Background(), req))
	assert.NoError(t, ps.Offer(context.Background(), req))
	assert.Equal(t, 2, ps.Size())
	// TODO: Remove this, after the initialization writes the readIndex.
	_, _, _ = ps.getNextItem(context.Background())
	require.NoError(t, ps.Shutdown(context.Background()))

	newPs := createTestPersistentQueueWithRequestsCapacity(t, ext, 1000)
	require.Equal(t, 2, newPs.Size())

	// Let's read both of the elements we put
	consume(newPs, func(_ context.Context, traces tracesRequest) error {
		require.Equal(t, req, traces)
		return nil
	})
	assert.Equal(t, 1, newPs.Size())

	consume(newPs, func(_ context.Context, traces tracesRequest) error {
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
			ps := createTestPersistentQueueWithRequestsCapacity(b, ext, 10000000)

			req := newTracesRequest(c.numTraces, c.numSpansPerTrace)

			bb.ReportAllocs()
			bb.ResetTimer()

			for i := 0; i < bb.N; i++ {
				require.NoError(bb, ps.Offer(context.Background(), req))
			}

			for i := 0; i < bb.N; i++ {
				require.True(bb, consume(ps, func(context.Context, tracesRequest) error { return nil }))
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
		t.Run(fmt.Sprintf("#elements:%v", c.in), func(*testing.T) {
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
		t.Run(fmt.Sprintf("#elements:%v", c.in), func(_ *testing.T) {
			buf := itemIndexArrayToBytes(c.in)
			out, err := bytesToItemIndexArray(buf)
			require.NoError(t, err)
			require.Equal(t, c.out, out)
		})
	}
}

func TestPersistentQueue_ShutdownWhileConsuming(t *testing.T) {
	ps := createTestPersistentQueueWithRequestsCapacity(t, NewMockStorageExtension(nil), 1000)

	assert.Equal(t, 0, ps.Size())
	assert.False(t, ps.client.(*mockStorageClient).isClosed())

	require.NoError(t, ps.Offer(context.Background(), newTracesRequest(5, 10)))

	index, _, ok := ps.getNextItem(context.Background())
	require.True(t, ok)
	assert.False(t, ps.client.(*mockStorageClient).isClosed())
	require.NoError(t, ps.Shutdown(context.Background()))
	assert.False(t, ps.client.(*mockStorageClient).isClosed())
	ps.OnProcessingFinished(index, nil)
	assert.True(t, ps.client.(*mockStorageClient).isClosed())
}

func TestPersistentQueue_StorageFull(t *testing.T) {
	req := newTracesRequest(5, 10)
	marshaled, err := marshalTracesRequest(req)
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
		require.True(t, consume(ps, func(context.Context, tracesRequest) error { return nil }))
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
		name          string
	}{
		{
			name:          "no errors",
			storageErrors: []error{},
			expectedError: nil,
		},
		{
			name: "error on first transaction, success afterwards",
			storageErrors: []error{
				errUpdatingDispatched,
			},
			expectedError: nil,
		},
		{
			name: "error on first and second transaction",
			storageErrors: []error{
				errUpdatingDispatched,
				errDeletingItem,
			},
			expectedError: errDeletingItem,
		},
		{
			name: "error on first and third transaction",
			storageErrors: []error{
				errUpdatingDispatched,
				nil,
				errUpdatingDispatched,
			},
			expectedError: errUpdatingDispatched,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			client := newFakeStorageClientWithErrors(tt.storageErrors)
			ps := createTestPersistentQueueWithClient(client)
			client.Reset()

			err := ps.itemDispatchingFinish(context.Background(), 0)

			require.ErrorIs(t, err, tt.expectedError)
		})
	}
}

func TestPersistentQueue_ItemsCapacityUsageRestoredOnShutdown(t *testing.T) {
	ext := NewMockStorageExtension(nil)
	pq := createTestPersistentQueueWithItemsCapacity(t, ext, 100)

	assert.Equal(t, 0, pq.Size())

	// Fill the queue up to the capacity.
	assert.NoError(t, pq.Offer(context.Background(), newTracesRequest(4, 10)))
	assert.NoError(t, pq.Offer(context.Background(), newTracesRequest(4, 10)))
	assert.NoError(t, pq.Offer(context.Background(), newTracesRequest(2, 10)))
	assert.Equal(t, 100, pq.Size())

	require.ErrorIs(t, pq.Offer(context.Background(), newTracesRequest(5, 5)), ErrQueueIsFull)
	assert.Equal(t, 100, pq.Size())

	assert.True(t, consume(pq, func(_ context.Context, traces tracesRequest) error {
		assert.Equal(t, 40, traces.traces.SpanCount())
		return nil
	}))
	assert.Equal(t, 60, pq.Size())

	require.NoError(t, pq.Shutdown(context.Background()))

	newPQ := createTestPersistentQueueWithItemsCapacity(t, ext, 100)

	// The queue should be restored to the previous size.
	assert.Equal(t, 60, newPQ.Size())

	require.NoError(t, newPQ.Offer(context.Background(), newTracesRequest(2, 5)))

	// Check the combined queue size.
	assert.Equal(t, 70, newPQ.Size())

	assert.True(t, consume(newPQ, func(_ context.Context, traces tracesRequest) error {
		assert.Equal(t, 40, traces.traces.SpanCount())
		return nil
	}))
	assert.Equal(t, 30, newPQ.Size())

	assert.True(t, consume(newPQ, func(_ context.Context, traces tracesRequest) error {
		assert.Equal(t, 20, traces.traces.SpanCount())
		return nil
	}))
	assert.Equal(t, 10, newPQ.Size())

	assert.NoError(t, newPQ.Shutdown(context.Background()))
}

// This test covers the case when the items capacity queue is enabled for the first time.
func TestPersistentQueue_ItemsCapacityUsageIsNotPreserved(t *testing.T) {
	ext := NewMockStorageExtension(nil)
	pq := createTestPersistentQueueWithRequestsCapacity(t, ext, 100)

	assert.Equal(t, 0, pq.Size())

	assert.NoError(t, pq.Offer(context.Background(), newTracesRequest(4, 10)))
	assert.NoError(t, pq.Offer(context.Background(), newTracesRequest(2, 10)))
	assert.NoError(t, pq.Offer(context.Background(), newTracesRequest(5, 5)))
	assert.Equal(t, 3, pq.Size())

	assert.True(t, consume(pq, func(_ context.Context, traces tracesRequest) error {
		assert.Equal(t, 40, traces.traces.SpanCount())
		return nil
	}))
	assert.Equal(t, 2, pq.Size())

	require.NoError(t, pq.Shutdown(context.Background()))

	newPQ := createTestPersistentQueueWithItemsCapacity(t, ext, 100)

	// The queue items size cannot be restored, fall back to request-based size
	assert.Equal(t, 2, newPQ.Size())

	require.NoError(t, newPQ.Offer(context.Background(), newTracesRequest(2, 5)))

	// Only new items are correctly reflected
	assert.Equal(t, 12, newPQ.Size())

	// Consuming a restored request should reduce the restored size by 20 but it should not go to below zero
	assert.True(t, consume(newPQ, func(_ context.Context, traces tracesRequest) error {
		assert.Equal(t, 20, traces.traces.SpanCount())
		return nil
	}))
	assert.Equal(t, 0, newPQ.Size())

	// Consuming another restored request should not affect the restored size since it's already dropped to 0.
	assert.True(t, consume(newPQ, func(_ context.Context, traces tracesRequest) error {
		assert.Equal(t, 25, traces.traces.SpanCount())
		return nil
	}))
	assert.Equal(t, 0, newPQ.Size())

	// Adding another batch should update the size accordingly
	require.NoError(t, newPQ.Offer(context.Background(), newTracesRequest(5, 5)))
	assert.Equal(t, 25, newPQ.Size())

	assert.True(t, consume(newPQ, func(_ context.Context, traces tracesRequest) error {
		assert.Equal(t, 10, traces.traces.SpanCount())
		return nil
	}))
	assert.Equal(t, 15, newPQ.Size())

	assert.NoError(t, newPQ.Shutdown(context.Background()))
}

// This test covers the case when the queue is restarted with the less capacity than needed to restore the queued items.
// In that case, the queue has to be restored anyway even if it exceeds the capacity limit.
func TestPersistentQueue_RequestCapacityLessAfterRestart(t *testing.T) {
	ext := NewMockStorageExtension(nil)
	pq := createTestPersistentQueueWithRequestsCapacity(t, ext, 100)

	assert.Equal(t, 0, pq.Size())

	assert.NoError(t, pq.Offer(context.Background(), newTracesRequest(4, 10)))
	assert.NoError(t, pq.Offer(context.Background(), newTracesRequest(2, 10)))
	assert.NoError(t, pq.Offer(context.Background(), newTracesRequest(5, 5)))
	assert.NoError(t, pq.Offer(context.Background(), newTracesRequest(1, 5)))

	// Read the first request just to populate the read index in the storage.
	// Otherwise, the write index won't be restored either.
	assert.True(t, consume(pq, func(_ context.Context, traces tracesRequest) error {
		assert.Equal(t, 40, traces.traces.SpanCount())
		return nil
	}))
	assert.Equal(t, 3, pq.Size())

	require.NoError(t, pq.Shutdown(context.Background()))

	// The queue is restarted with the less capacity than needed to restore the queued items, but with the same
	// underlying storage. No need to drop requests that are over capacity since they are already in the storage.
	newPQ := createTestPersistentQueueWithRequestsCapacity(t, ext, 2)

	// The queue items size cannot be restored, fall back to request-based size
	assert.Equal(t, 3, newPQ.Size())

	// Queue is full
	require.Error(t, newPQ.Offer(context.Background(), newTracesRequest(2, 5)))

	assert.True(t, consume(newPQ, func(_ context.Context, traces tracesRequest) error {
		assert.Equal(t, 20, traces.traces.SpanCount())
		return nil
	}))
	assert.Equal(t, 2, newPQ.Size())

	// Still full
	require.Error(t, newPQ.Offer(context.Background(), newTracesRequest(2, 5)))

	assert.True(t, consume(newPQ, func(_ context.Context, traces tracesRequest) error {
		assert.Equal(t, 25, traces.traces.SpanCount())
		return nil
	}))
	assert.Equal(t, 1, newPQ.Size())

	// Now it can accept new items
	assert.NoError(t, newPQ.Offer(context.Background(), newTracesRequest(2, 5)))

	assert.NoError(t, newPQ.Shutdown(context.Background()))
}

// This test covers the case when the persistent storage is recovered from a snapshot which has
// bigger value for the used size than the size of the actual items in the storage.
func TestPersistentQueue_RestoredUsedSizeIsCorrectedOnDrain(t *testing.T) {
	ext := NewMockStorageExtension(nil)
	pq := createTestPersistentQueueWithItemsCapacity(t, ext, 1000)

	assert.Equal(t, 0, pq.Size())

	for i := 0; i < 6; i++ {
		require.NoError(t, pq.Offer(context.Background(), newTracesRequest(2, 5)))
	}
	assert.Equal(t, 60, pq.Size())

	// Consume 30 items
	for i := 0; i < 3; i++ {
		assert.True(t, consume(pq, func(context.Context, tracesRequest) error { return nil }))
	}
	// The used size is now 30, but the snapshot should have 50, because it's taken every 5 read/writes.
	assert.Equal(t, 30, pq.Size())

	// Create a new queue pointed to the same storage
	newPQ := createTestPersistentQueueWithItemsCapacity(t, ext, 1000)

	// This is an incorrect size restored from the snapshot.
	// In reality the size should be 30. Once the queue is drained, it will be updated to the correct size.
	assert.Equal(t, 50, newPQ.Size())

	assert.True(t, consume(newPQ, func(context.Context, tracesRequest) error { return nil }))
	assert.True(t, consume(newPQ, func(context.Context, tracesRequest) error { return nil }))
	assert.Equal(t, 30, newPQ.Size())

	// Now the size must be correctly reflected
	assert.True(t, consume(newPQ, func(context.Context, tracesRequest) error { return nil }))
	assert.Equal(t, 0, newPQ.Size())

	assert.NoError(t, newPQ.Shutdown(context.Background()))
	assert.NoError(t, pq.Shutdown(context.Background()))
}

func requireCurrentlyDispatchedItemsEqual(t *testing.T, pq *persistentQueue[tracesRequest], compare []uint64) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	assert.ElementsMatch(t, compare, pq.currentlyDispatchedItems)
}
