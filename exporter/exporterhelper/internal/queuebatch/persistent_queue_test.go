// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/hosttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/storagetest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.opentelemetry.io/collector/pipeline"
)

// itemsSizer is a sizer implementation that returns the size of a queue element as the number of items it contains.
type itemsSizer struct{}

func (is *itemsSizer) Sizeof(val uint64) int64 {
	if val > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(val)
}

type uint64Encoding struct{}

func (uint64Encoding) Marshal(val uint64) ([]byte, error) {
	return binary.LittleEndian.AppendUint64([]byte{}, val), nil
}

func (uint64Encoding) Unmarshal(bytes []byte) (uint64, error) {
	if len(bytes) < 8 {
		return 0, errInvalidValue
	}
	return binary.LittleEndian.Uint64(bytes), nil
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
	if err := m.Batch(ctx, op); err != nil {
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

func (m *fakeBoundedStorageClient) Close(context.Context) error {
	return nil
}

func (m *fakeBoundedStorageClient) Batch(_ context.Context, ops ...*storage.Operation) error {
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

	m.sizeInBytes += totalAdded - totalRemoved

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

func (m *fakeBoundedStorageClient) getTotalSizeChange(ops []*storage.Operation) (totalAdded int, totalRemoved int) {
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

func (m *fakeStorageClientWithErrors) Close(context.Context) error {
	return nil
}

func (m *fakeStorageClientWithErrors) Batch(context.Context, ...*storage.Operation) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	if m.nextErrorIndex >= len(m.errors) {
		return nil
	}

	m.nextErrorIndex++
	return m.errors[m.nextErrorIndex-1]
}

func (m *fakeStorageClientWithErrors) Reset() {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.nextErrorIndex = 0
}

// createAndStartTestPersistentQueue creates and starts a fake queue with the given capacity and number of consumers.
func createAndStartTestPersistentQueue(t *testing.T, sizer request.Sizer[uint64], capacity int64, numConsumers int,
	consumeFunc func(_ context.Context, item uint64) error,
) Queue[uint64] {
	pq := newPersistentQueue[uint64](persistentQueueSettings[uint64]{
		sizer:     sizer,
		capacity:  capacity,
		signal:    pipeline.SignalTraces,
		storageID: component.ID{},
		encoding:  uint64Encoding{},
		id:        component.NewID(exportertest.NopType),
		telemetry: componenttest.NewNopTelemetrySettings(),
	})
	ac := newAsyncQueue(pq, numConsumers, func(ctx context.Context, item uint64, done Done) {
		done.OnDone(consumeFunc(ctx, item))
	})
	host := hosttest.NewHost(map[component.ID]component.Component{
		{}: storagetest.NewMockStorageExtension(nil),
	})
	require.NoError(t, ac.Start(context.Background(), host))
	t.Cleanup(func() {
		assert.NoError(t, ac.Shutdown(context.Background()))
	})
	return pq
}

func createTestPersistentQueueWithClient(client storage.Client) *persistentQueue[uint64] {
	pq := newPersistentQueue[uint64](persistentQueueSettings[uint64]{
		sizer:     request.RequestsSizer[uint64]{},
		capacity:  1000,
		signal:    pipeline.SignalTraces,
		storageID: component.ID{},
		encoding:  uint64Encoding{},
		id:        component.NewID(exportertest.NopType),
		telemetry: componenttest.NewNopTelemetrySettings(),
	}).(*persistentQueue[uint64])
	pq.initClient(context.Background(), client)
	return pq
}

func createTestPersistentQueueWithRequestsCapacity(tb testing.TB, ext storage.Extension, capacity int64) *persistentQueue[uint64] {
	return createTestPersistentQueueWithCapacityLimiter(tb, ext, request.RequestsSizer[uint64]{}, capacity)
}

func createTestPersistentQueueWithItemsCapacity(tb testing.TB, ext storage.Extension, capacity int64) *persistentQueue[uint64] {
	return createTestPersistentQueueWithCapacityLimiter(tb, ext, &itemsSizer{}, capacity)
}

func createTestPersistentQueueWithCapacityLimiter(tb testing.TB, ext storage.Extension, sizer request.Sizer[uint64],
	capacity int64,
) *persistentQueue[uint64] {
	pq := newPersistentQueue[uint64](persistentQueueSettings[uint64]{
		sizer:     sizer,
		capacity:  capacity,
		signal:    pipeline.SignalTraces,
		storageID: component.ID{},
		encoding:  uint64Encoding{},
		id:        component.NewID(exportertest.NopType),
		telemetry: componenttest.NewNopTelemetrySettings(),
	}).(*persistentQueue[uint64])
	require.NoError(tb, pq.Start(context.Background(), hosttest.NewHost(map[component.ID]component.Component{{}: ext})))
	return pq
}

func TestPersistentQueue_FullCapacity(t *testing.T) {
	tests := []struct {
		name           string
		sizer          request.Sizer[uint64]
		capacity       int64
		sizeMultiplier int64
	}{
		{
			name:           "requests_capacity",
			sizer:          request.RequestsSizer[uint64]{},
			capacity:       5,
			sizeMultiplier: 1,
		},
		{
			name:           "items_capacity",
			sizer:          &itemsSizer{},
			capacity:       55,
			sizeMultiplier: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done := make(chan struct{})
			pq := createAndStartTestPersistentQueue(t,
				tt.sizer, tt.capacity, 1,
				func(context.Context, uint64) error {
					<-done
					return nil
				})
			assert.Equal(t, int64(0), pq.Size())

			req := uint64(10)

			// First request is picked by the consumer. Wait until the consumer is blocked on done.
			require.NoError(t, pq.Offer(context.Background(), req))
			assert.Eventually(t, func() bool {
				return pq.Size() == 0
			}, 2*time.Second, 10*time.Millisecond)

			for i := 0; i < 10; i++ {
				result := pq.Offer(context.Background(), uint64(10))
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
	pq := createAndStartTestPersistentQueue(t,
		request.RequestsSizer[uint64]{}, 1001, 1,
		func(context.Context, uint64) error {
			return nil
		})
	req := uint64(10)

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
			req := uint64(10)

			consumed := &atomic.Int64{}
			pq := createAndStartTestPersistentQueue(t,
				request.RequestsSizer[uint64]{}, 1000, c.numConsumers,
				func(context.Context, uint64) error {
					consumed.Add(int64(1))
					return nil
				})

			for i := 0; i < c.numMessagesProduced; i++ {
				require.NoError(t, pq.Offer(context.Background(), req))
			}

			// Because the persistent queue is not draining after Shutdown, need to wait here for the drain.
			assert.Eventually(t, func() bool {
				return c.numMessagesProduced == int(consumed.Load())
			}, 5*time.Second, 10*time.Millisecond)
		})
	}
}

func TestPersistentBlockingQueue(t *testing.T) {
	tests := []struct {
		name  string
		sizer request.Sizer[uint64]
	}{
		{
			name:  "requests_based",
			sizer: request.RequestsSizer[uint64]{},
		},
		{
			name:  "items_based",
			sizer: &itemsSizer{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pq := newPersistentQueue[uint64](persistentQueueSettings[uint64]{
				sizer:           tt.sizer,
				capacity:        100,
				blockOnOverflow: true,
				signal:          pipeline.SignalTraces,
				storageID:       component.ID{},
				encoding:        uint64Encoding{},
				id:              component.NewID(exportertest.NopType),
				telemetry:       componenttest.NewNopTelemetrySettings(),
			})
			consumed := &atomic.Int64{}
			ac := newAsyncQueue(pq, 10, func(_ context.Context, _ uint64, done Done) {
				consumed.Add(1)
				done.OnDone(nil)
			})
			host := hosttest.NewHost(map[component.ID]component.Component{
				{}: storagetest.NewMockStorageExtension(nil),
			})
			require.NoError(t, ac.Start(context.Background(), host))

			td := uint64(10)
			wg := &sync.WaitGroup{}
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < 100_000; j++ {
						assert.NoError(t, pq.Offer(context.Background(), td))
					}
				}()
			}
			wg.Wait()
			// Because the persistent queue is not draining after Shutdown, need to wait here for the drain.
			assert.Eventually(t, func() bool {
				return int(consumed.Load()) == 1_000_000
			}, 5*time.Second, 10*time.Millisecond)
			require.NoError(t, ac.Shutdown(context.Background()))
		})
	}
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

			extensions := map[component.ID]component.Component{}
			for i := 0; i < tt.numStorages; i++ {
				extensions[component.MustNewIDWithName("file_storage", strconv.Itoa(i))] = storagetest.NewMockStorageExtension(tt.getClientError)
			}
			host := hosttest.NewHost(extensions)
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
	settings := extensiontest.NewNopSettings(factory.Type())
	extension, err := factory.Create(context.Background(), settings, extConfig)
	require.NoError(t, err)
	extensions := map[component.ID]component.Component{
		storageID: extension,
	}
	host := hosttest.NewHost(extensions)
	ownerID := component.MustNewID("foo_exporter")

	// execute
	client, err := toStorageClient(context.Background(), storageID, host, ownerID, pipeline.SignalTraces)

	// we should get an error about the extension type
	require.ErrorIs(t, err, errWrongExtensionType)
	assert.Nil(t, client)
}

func TestPersistentQueue_StopAfterBadStart(t *testing.T) {
	pq := newPersistentQueue[uint64](persistentQueueSettings[uint64]{})
	// verify that stopping a un-start/started w/error queue does not panic
	assert.NoError(t, pq.Shutdown(context.Background()))
}

func TestPersistentQueue_CorruptedData(t *testing.T) {
	req := uint64(100)
	sizer := request.RequestsSizer[uint64]{}
	encoding := uint64Encoding{}
	capacity := int64(1000)

	tests := []struct {
		name         string
		corruptFunc  func(t *testing.T, client storage.Client) // Corrupts the specific key
		keyToCorrupt string                                    // For logging/identification, can be new or old key name
		expectError  bool                                      // Whether queue initialization should ideally error or log errors and start fresh
		// We expect the queue to initialize empty or with recovered state if only one part is corrupt.
	}{
		{
			name: "corrupted new metadata key",
			corruptFunc: func(t *testing.T, client storage.Client) {
				require.NoError(t, client.Set(context.Background(), queueMetadataKey, []byte("corrupted data")))
			},
			keyToCorrupt: queueMetadataKey,
			expectError:  false, // Should log error and start as new
		},
		{
			name: "new metadata key too short",
			corruptFunc: func(t *testing.T, client storage.Client) {
				require.NoError(t, client.Set(context.Background(), queueMetadataKey, []byte{1, 2, 3}))
			},
			keyToCorrupt: queueMetadataKey,
			expectError:  false, // Should log error and start as new
		},
		{
			name: "corrupted old read index",
			corruptFunc: func(t *testing.T, client storage.Client) {
				require.NoError(t, client.Set(context.Background(), oldReadIndexKey, []byte("corrupted_ri")))
			},
			keyToCorrupt: oldReadIndexKey,
			expectError:  false, // Should log error and start as new if other old keys are missing or also corrupt
		},
		{
			name: "corrupted old write index",
			corruptFunc: func(t *testing.T, client storage.Client) {
				require.NoError(t, client.Set(context.Background(), oldWriteIndexKey, []byte("corrupted_wi")))
			},
			keyToCorrupt: oldWriteIndexKey,
			expectError:  false, // Should log error and start as new
		},
		{
			name: "corrupted old dispatched items",
			corruptFunc: func(t *testing.T, client storage.Client) {
				require.NoError(t, client.Set(context.Background(), oldCurrentlyDispatchedItemsKey, []byte("corrupted_cdi")))
			},
			keyToCorrupt: oldCurrentlyDispatchedItemsKey,
			expectError:  false, // Should log error, CDI might be empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := storagetest.NewMockStorageExtension(nil)
			client, err := mockStorage.GetClient(context.Background(), component.KindExporter, component.NewID(exportertest.NopType), pipeline.SignalTraces.String())
			require.NoError(t, err)

			// Apply corruption
			tt.corruptFunc(t, client)

			// Create and start the queue. It should handle the corruption gracefully.
			// The logger inside persistentQueue will show error messages.
			// We expect it to initialize, possibly as an empty queue.
			set := persistentQueueSettings[uint64]{
				telemetry:       componenttest.NewNopTelemetrySettings(),
				capacity:        capacity,
				sizer:           sizer,
				encoding:        encoding,
				id:              component.NewID(exportertest.NopType),
				signal:          pipeline.SignalTraces, // Example signal
				blockOnOverflow: true,                  // Default for many tests
			}
			pq := newPersistentQueue[uint64](set).(*persistentQueue[uint64])
			pq.initClient(context.Background(), client)
			require.NotNil(t, pq)

			// Offer an item to ensure the queue is operational
			err = pq.Offer(context.Background(), req)
			require.NoError(t, err)
			assert.Equal(t, sizer.Sizeof(req), pq.Size())

			// Read the item
			_, item, done, ok := pq.Read(context.Background())
			require.True(t, ok)
			assert.Equal(t, req, item)
			done.OnDone(nil)

			assert.Equal(t, int64(0), pq.Size())
			require.NoError(t, pq.Shutdown(context.Background()))
			require.NoError(t, mockStorage.Shutdown(context.Background()))
		})
	}
}

func TestPersistentQueue_CurrentlyProcessedItems(t *testing.T) {
	req := uint64(50)

	ext := storagetest.NewMockStorageExtension(nil)
	ps := createTestPersistentQueueWithRequestsCapacity(t, ext, 1000)

	for i := 0; i < 5; i++ {
		require.NoError(t, ps.Offer(context.Background(), req))
	}

	requireCurrentlyDispatchedItemsEqual(t, ps, []uint64{})

	// Takes index 0 in process.
	_, readReq, _, found := ps.Read(context.Background())
	require.True(t, found)
	assert.Equal(t, req, readReq)
	requireCurrentlyDispatchedItemsEqual(t, ps, []uint64{0})

	// This takes item 1 to process.
	_, secondReadReq, secondDone, found := ps.Read(context.Background())
	require.True(t, found)
	assert.Equal(t, req, secondReadReq)
	requireCurrentlyDispatchedItemsEqual(t, ps, []uint64{0, 1})

	// Lets mark item 1 as finished, it will remove it from the currently dispatched items list.
	secondDone.OnDone(nil)
	requireCurrentlyDispatchedItemsEqual(t, ps, []uint64{0})

	// Reload the storage. Since items 0 was not finished, this should be re-enqueued at the end.
	// The queue should be essentially {3,4,0,2}.
	newPs := createTestPersistentQueueWithRequestsCapacity(t, ext, 1000)
	assert.Equal(t, int64(4), newPs.Size())
	requireCurrentlyDispatchedItemsEqual(t, newPs, []uint64{})

	// We should be able to pull all remaining items now
	for i := 0; i < 4; i++ {
		consume(newPs, func(_ context.Context, val uint64) error {
			assert.Equal(t, req, val)
			return nil
		})
	}

	// The queue should be now empty
	requireCurrentlyDispatchedItemsEqual(t, newPs, []uint64{})
	assert.Equal(t, int64(0), newPs.Size())
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
	req := uint64(50)

	ext := storagetest.NewMockStorageExtension(nil)
	ps := createTestPersistentQueueWithRequestsCapacity(t, ext, 5)

	// Put in items up to capacity
	for i := 0; i < 5; i++ {
		require.NoError(t, ps.Offer(context.Background(), req))
	}
	require.Equal(t, int64(5), ps.Size())

	require.True(t, consume(ps, func(context.Context, uint64) error {
		// Check that size is still full even when consuming the element.
		require.Equal(t, int64(5), ps.Size())
		return experr.NewShutdownErr(nil)
	}))
	require.NoError(t, ps.Shutdown(context.Background()))

	// Reload with extra capacity to make sure we re-enqueue in-progress items.
	newPs := createTestPersistentQueueWithRequestsCapacity(t, ext, 5)
	require.Equal(t, int64(5), newPs.Size())
}

func TestPersistentQueueStartWithNonDispatchedConcurrent(t *testing.T) {
	req := uint64(1)

	ext := storagetest.NewMockStorageExtensionWithDelay(nil, 20*time.Nanosecond)
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
				assert.True(t, consume(pq, func(context.Context, uint64) error { return nil }))
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
	assert.Zero(t, pq.Size())
}

func TestPersistentQueue_PutCloseReadClose(t *testing.T) {
	req := uint64(50)
	ext := storagetest.NewMockStorageExtension(nil)
	ps := createTestPersistentQueueWithRequestsCapacity(t, ext, 1000)
	assert.Equal(t, int64(0), ps.Size())

	// Put two elements and close the extension
	require.NoError(t, ps.Offer(context.Background(), req))
	require.NoError(t, ps.Offer(context.Background(), req))
	assert.Equal(t, int64(2), ps.Size())
	// TODO: Remove this, after the initialization writes the readIndex.
	_, _, _, _ = ps.Read(context.Background())
	require.NoError(t, ps.Shutdown(context.Background()))

	newPs := createTestPersistentQueueWithRequestsCapacity(t, ext, 1000)
	require.Equal(t, int64(2), newPs.Size())

	// Let's read both of the elements we put
	consume(newPs, func(_ context.Context, val uint64) error {
		require.Equal(t, req, val)
		return nil
	})
	assert.Equal(t, int64(1), newPs.Size())

	consume(newPs, func(_ context.Context, val uint64) error {
		require.Equal(t, req, val)
		return nil
	})
	require.Equal(t, int64(0), newPs.Size())
	require.NoError(t, newPs.Shutdown(context.Background()))
}

func BenchmarkPersistentQueue(b *testing.B) {
	ext := storagetest.NewMockStorageExtension(nil)
	ps := createTestPersistentQueueWithRequestsCapacity(b, ext, 10000000)

	req := uint64(100)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		require.NoError(b, ps.Offer(context.Background(), req))
	}

	for i := 0; i < b.N; i++ {
		require.True(b, consume(ps, func(context.Context, uint64) error { return nil }))
	}
	require.NoError(b, ext.Shutdown(context.Background()))
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
	ps := createTestPersistentQueueWithRequestsCapacity(t, storagetest.NewMockStorageExtension(nil), 1000)

	assert.Equal(t, int64(0), ps.Size())
	assert.False(t, ps.client.(*storagetest.MockStorageClient).IsClosed())

	require.NoError(t, ps.Offer(context.Background(), uint64(50)))

	_, _, done, ok := ps.Read(context.Background())
	require.True(t, ok)
	assert.False(t, ps.client.(*storagetest.MockStorageClient).IsClosed())
	require.NoError(t, ps.Shutdown(context.Background()))
	assert.False(t, ps.client.(*storagetest.MockStorageClient).IsClosed())
	done.OnDone(nil)
	assert.True(t, ps.client.(*storagetest.MockStorageClient).IsClosed())
}

func TestPersistentQueue_StorageFull(t *testing.T) {
	marshaled, err := uint64Encoding{}.Marshal(uint64(50))
	require.NoError(t, err)
	maxSizeInBytes := len(marshaled) * 5 // arbitrary small number

	client := newFakeBoundedStorageClient(maxSizeInBytes)
	ps := createTestPersistentQueueWithClient(client)

	// Put enough items in to fill the underlying storage
	reqCount := int64(0)
	for {
		err = ps.Offer(context.Background(), uint64(50))
		if errors.Is(err, syscall.ENOSPC) {
			break
		}
		require.NoError(t, err)
		reqCount++
	}

	// Check that the size is correct
	require.Equal(t, reqCount, ps.Size(), "Size must be equal to the number of items inserted")

	// Manually set the storage to only have a small amount of free space left (needs 24).
	const minMetadataSize = 28
	newMaxSize := client.GetSizeInBytes() + 23 + minMetadataSize
	client.SetMaxSizeInBytes(newMaxSize)

	// Take out all the items
	// Getting the first item fails, as we can't update the state in storage, so we just delete it without returning it
	// Subsequent items succeed, as deleting the first item frees enough space for the state update
	reqCount--
	for i := reqCount; i > 0; i-- {
		require.True(t, consume(ps, func(context.Context, uint64) error { return nil }))
	}
	require.Equal(t, int64(0), ps.Size())

	// We should be able to put a new item in
	// However, this will fail if deleting items fails with full storage
	require.NoError(t, ps.Offer(context.Background(), uint64(50)))
}

func TestPersistentQueue_ItemDispatchingFinish_ErrorHandling(t *testing.T) {
	errDeletingItem := errors.New("error deleting item")
	errUpdatingDispatched := errors.New("error updating dispatched items")
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

			require.ErrorIs(t, ps.itemDispatchingFinish(context.Background(), 0), tt.expectedError)
		})
	}
}

func TestPersistentQueue_ItemsCapacityUsageRestoredOnShutdown(t *testing.T) {
	ext := storagetest.NewMockStorageExtension(nil)
	pq := createTestPersistentQueueWithItemsCapacity(t, ext, 100)

	assert.Equal(t, int64(0), pq.Size())

	// Fill the queue up to the capacity.
	require.NoError(t, pq.Offer(context.Background(), uint64(40)))
	require.NoError(t, pq.Offer(context.Background(), uint64(40)))
	require.NoError(t, pq.Offer(context.Background(), uint64(20)))
	assert.Equal(t, int64(100), pq.Size())

	require.ErrorIs(t, pq.Offer(context.Background(), uint64(25)), ErrQueueIsFull)
	assert.Equal(t, int64(100), pq.Size())

	assert.True(t, consume(pq, func(_ context.Context, val uint64) error {
		assert.Equal(t, uint64(40), val)
		return nil
	}))
	assert.Equal(t, int64(60), pq.Size())

	require.NoError(t, pq.Shutdown(context.Background()))

	newPQ := createTestPersistentQueueWithItemsCapacity(t, ext, 100)

	// The queue should be restored to the previous size.
	assert.Equal(t, int64(60), newPQ.Size())

	require.NoError(t, newPQ.Offer(context.Background(), uint64(10)))

	// Check the combined queue size.
	assert.Equal(t, int64(70), newPQ.Size())

	assert.True(t, consume(newPQ, func(_ context.Context, val uint64) error {
		assert.Equal(t, uint64(40), val)
		return nil
	}))
	assert.Equal(t, int64(30), newPQ.Size())

	assert.True(t, consume(newPQ, func(_ context.Context, val uint64) error {
		assert.Equal(t, uint64(20), val)
		return nil
	}))
	assert.Equal(t, int64(10), newPQ.Size())

	require.NoError(t, newPQ.Shutdown(context.Background()))
}

// This test covers the case when the items capacity queue is enabled for the first time.
func TestPersistentQueue_ItemsCapacityUsageIsNotPreserved(t *testing.T) {
	ext := storagetest.NewMockStorageExtension(nil)
	pq := createTestPersistentQueueWithRequestsCapacity(t, ext, 100)

	assert.Equal(t, int64(0), pq.Size())

	require.NoError(t, pq.Offer(context.Background(), uint64(40)))
	require.NoError(t, pq.Offer(context.Background(), uint64(20)))
	require.NoError(t, pq.Offer(context.Background(), uint64(25)))
	assert.Equal(t, int64(3), pq.Size())

	assert.True(t, consume(pq, func(_ context.Context, val uint64) error {
		assert.Equal(t, uint64(40), val)
		return nil
	}))
	assert.Equal(t, int64(2), pq.Size())

	require.NoError(t, pq.Shutdown(context.Background()))

	newPQ := createTestPersistentQueueWithItemsCapacity(t, ext, 100)

	// The queue items size cannot be restored, fall back to request-based size
	assert.Equal(t, int64(2), newPQ.Size())

	require.NoError(t, newPQ.Offer(context.Background(), uint64(10)))

	// Only new items are correctly reflected
	assert.Equal(t, int64(12), newPQ.Size())

	// Consuming a restored request should reduce the restored size by 20 but it should not go to below zero
	assert.True(t, consume(newPQ, func(_ context.Context, val uint64) error {
		assert.Equal(t, uint64(20), val)
		return nil
	}))
	assert.Equal(t, int64(0), newPQ.Size())

	// Consuming another restored request should not affect the restored size since it's already dropped to 0.
	assert.True(t, consume(newPQ, func(_ context.Context, val uint64) error {
		assert.Equal(t, uint64(25), val)
		return nil
	}))
	assert.Equal(t, int64(0), newPQ.Size())

	// Adding another batch should update the size accordingly
	require.NoError(t, newPQ.Offer(context.Background(), uint64(25)))
	assert.Equal(t, int64(25), newPQ.Size())

	assert.True(t, consume(newPQ, func(_ context.Context, val uint64) error {
		assert.Equal(t, uint64(10), val)
		return nil
	}))
	assert.Equal(t, int64(15), newPQ.Size())

	require.NoError(t, newPQ.Shutdown(context.Background()))
}

// This test covers the case when the queue is restarted with the less capacity than needed to restore the queued items.
// In that case, the queue has to be restored anyway even if it exceeds the capacity limit.
func TestPersistentQueue_RequestCapacityLessAfterRestart(t *testing.T) {
	ext := storagetest.NewMockStorageExtension(nil)
	pq := createTestPersistentQueueWithRequestsCapacity(t, ext, 100)

	assert.Equal(t, int64(0), pq.Size())

	require.NoError(t, pq.Offer(context.Background(), uint64(40)))
	require.NoError(t, pq.Offer(context.Background(), uint64(20)))
	require.NoError(t, pq.Offer(context.Background(), uint64(25)))
	require.NoError(t, pq.Offer(context.Background(), uint64(5)))

	// Read the first request just to populate the read index in the storage.
	// Otherwise, the write index won't be restored either.
	assert.True(t, consume(pq, func(_ context.Context, val uint64) error {
		assert.Equal(t, uint64(40), val)
		return nil
	}))
	assert.Equal(t, int64(3), pq.Size())

	require.NoError(t, pq.Shutdown(context.Background()))

	// The queue is restarted with the less capacity than needed to restore the queued items, but with the same
	// underlying storage. No need to drop requests that are over capacity since they are already in the storage.
	newPQ := createTestPersistentQueueWithRequestsCapacity(t, ext, 2)

	// The queue items size cannot be restored, fall back to request-based size
	assert.Equal(t, int64(3), newPQ.Size())

	// Queue is full
	require.Error(t, newPQ.Offer(context.Background(), uint64(10)))

	assert.True(t, consume(newPQ, func(_ context.Context, val uint64) error {
		assert.Equal(t, uint64(20), val)
		return nil
	}))
	assert.Equal(t, int64(2), newPQ.Size())

	// Still full
	require.Error(t, newPQ.Offer(context.Background(), uint64(10)))

	assert.True(t, consume(newPQ, func(_ context.Context, val uint64) error {
		assert.Equal(t, uint64(25), val)
		return nil
	}))
	assert.Equal(t, int64(1), newPQ.Size())

	// Now it can accept new items
	require.NoError(t, newPQ.Offer(context.Background(), uint64(10)))

	require.NoError(t, newPQ.Shutdown(context.Background()))
}

// This test covers the case when the persistent storage is recovered from a snapshot.
func TestPersistentQueue_RestoredUsedSizeIsCorrectedOnDrain(t *testing.T) {
	ext := storagetest.NewMockStorageExtension(nil)
	pq := createTestPersistentQueueWithItemsCapacity(t, ext, 1000)

	assert.Equal(t, int64(0), pq.Size())

	for i := 0; i < 6; i++ {
		require.NoError(t, pq.Offer(context.Background(), uint64(10)))
	}
	assert.Equal(t, int64(60), pq.Size())

	// Consume 30 items
	for i := 0; i < 3; i++ {
		assert.True(t, consume(pq, func(context.Context, uint64) error { return nil }))
	}
	assert.Equal(t, int64(30), pq.Size())

	// Create a new queue pointed to the same storage
	newPQ := createTestPersistentQueueWithItemsCapacity(t, ext, 1000)

	assert.Equal(t, int64(30), newPQ.Size())

	assert.True(t, consume(newPQ, func(context.Context, uint64) error { return nil }))
	assert.True(t, consume(newPQ, func(context.Context, uint64) error { return nil }))
	assert.Equal(t, int64(10), newPQ.Size())

	// Now the size must be correctly reflected
	assert.True(t, consume(newPQ, func(context.Context, uint64) error { return nil }))
	assert.Equal(t, int64(0), newPQ.Size())

	require.NoError(t, newPQ.Shutdown(context.Background()))
	require.NoError(t, pq.Shutdown(context.Background()))
}

func requireCurrentlyDispatchedItemsEqual(t *testing.T, pq *persistentQueue[uint64], compare []uint64) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	assert.ElementsMatch(t, compare, pq.currentlyDispatchedItems)
}

func TestQueueMetadataMarshaling(t *testing.T) {
	cases := []struct {
		name string
		in   *queueMetadata
		out  *queueMetadata
		err  bool
	}{
		{
			name: "valid full metadata",
			in: &queueMetadata{
				ReadIndex:                10,
				WriteIndex:               20,
				QueueSize:                100,
				CurrentlyDispatchedItems: []uint64{1, 2, 3},
			},
			out: &queueMetadata{
				ReadIndex:                10,
				WriteIndex:               20,
				QueueSize:                100,
				CurrentlyDispatchedItems: []uint64{1, 2, 3},
			},
		},
		{
			name: "valid empty dispatched items",
			in: &queueMetadata{
				ReadIndex:                5,
				WriteIndex:               5,
				QueueSize:                0,
				CurrentlyDispatchedItems: []uint64{},
			},
			out: &queueMetadata{
				ReadIndex:                5,
				WriteIndex:               5,
				QueueSize:                0,
				CurrentlyDispatchedItems: []uint64{},
			},
		},
		{
			name: "valid nil dispatched items", // Should be treated as empty
			in: &queueMetadata{
				ReadIndex:                7,
				WriteIndex:               8,
				QueueSize:                10,
				CurrentlyDispatchedItems: nil,
			},
			out: &queueMetadata{
				ReadIndex:                7,
				WriteIndex:               8,
				QueueSize:                10,
				CurrentlyDispatchedItems: []uint64{}, // Unmarshal should set this to empty
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf, err := marshalQueueMetadata(tc.in, nil)
			require.NoError(t, err)
			require.NotNil(t, buf)

			outMeta, err := unmarshalQueueMetadata(buf)
			if tc.err {
				require.Error(t, err)
				require.Nil(t, buf)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.out, outMeta)
			}
		})
	}

	t.Run("unmarshal nil data", func(t *testing.T) {
		_, err := unmarshalQueueMetadata(nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, errValueNotSet)
	})

	t.Run("unmarshal short data", func(t *testing.T) {
		shortData := make([]byte, 27) // Less than min 28
		_, err := unmarshalQueueMetadata(shortData)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "queue metadata too short")
	})

	t.Run("unmarshall short data for cdi", func(t *testing.T) {
		meta := &queueMetadata{
			ReadIndex:                1,
			WriteIndex:               2,
			QueueSize:                1,
			CurrentlyDispatchedItems: []uint64{10, 20},
		}
		buf, err := marshalQueueMetadata(meta, nil)
		require.NoError(t, err)
		// Truncate the buffer to make CDI part too short
		require.NoError(t, err)
		truncatedBuf := buf[:len(buf)-1]
		_, err = unmarshalQueueMetadata(truncatedBuf)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "queue metadata too short")
	})
}

func TestPersistentQueue_MigrationFromOldFormat(t *testing.T) {
	req := uint64(100)
	sizer := request.RequestsSizer[uint64]{}
	encoding := uint64Encoding{}
	var capacity int64 = 1000

	mockStorage := storagetest.NewMockStorageExtension(nil)
	client, err := mockStorage.GetClient(context.Background(), component.KindExporter, component.NewID(exportertest.NopType), pipeline.SignalTraces.String())
	require.NoError(t, err)

	// 1. Populate storage with old format data
	oldRI := uint64(5)
	oldWI := uint64(10)
	oldCDI := []uint64{6, 7}
	oldQS := uint64(50)

	retrievedWI := oldWI + uint64(len(oldCDI)) // Retrieve currentlyDispatchedItems after initPersistentContiguousQueue

	require.NoError(t, client.Set(context.Background(), oldReadIndexKey, itemIndexToBytes(oldRI)))
	require.NoError(t, client.Set(context.Background(), oldWriteIndexKey, itemIndexToBytes(oldWI)))
	require.NoError(t, client.Set(context.Background(), oldCurrentlyDispatchedItemsKey, itemIndexArrayToBytes(oldCDI)))
	require.NoError(t, client.Set(context.Background(), oldQueueSizeKey, itemIndexToBytes(oldQS)))

	// Add some items to storage that correspond to the indices
	for i := uint64(0); i < oldWI; i++ {
		itemData, _ := encoding.Marshal(req + i) // Store unique items
		require.NoError(t, client.Set(context.Background(), getItemKey(i), itemData))
	}

	// 2. Create and start the queue
	set := persistentQueueSettings[uint64]{
		telemetry:       componenttest.NewNopTelemetrySettings(),
		capacity:        capacity,
		sizer:           sizer,
		encoding:        encoding,
		id:              component.NewID(exportertest.NopType),
		signal:          pipeline.SignalTraces,
		blockOnOverflow: true,
	}

	pq := newPersistentQueue[uint64](set).(*persistentQueue[uint64])
	pq.initClient(context.Background(), client) // This will trigger initPersistentContiguousQueue

	// 3. Verify in-memory state matches old format data
	pq.mu.Lock()
	assert.Equal(t, oldRI, pq.readIndex, "Read index should be migrated")
	assert.Equal(t, retrievedWI, pq.writeIndex, "Write index should be migrated")

	if pq.isRequestSized {
		queueSize := retrievedWI - oldRI
		//nolint:gosec
		assert.Equal(t, int64(queueSize), pq.queueSize, "Queue size should be calculated for requestSized")
	} else {
		assert.Equal(t, int64(oldQS), pq.queueSize, " Queue size should be migrated for non-requestSized")
	}
	pq.mu.Unlock()

	// 4. Verify new metadata key exists and contains the migrated data
	newMetaBytes, err := client.Get(context.Background(), queueMetadataKey)
	require.NoError(t, err, "New metadata key should exist after migration")
	require.NotNil(t, newMetaBytes)

	migratedMeta, err := unmarshalQueueMetadata(newMetaBytes)
	require.NoError(t, err)
	assert.Equal(t, oldRI, migratedMeta.ReadIndex)
	assert.Equal(t, retrievedWI, migratedMeta.WriteIndex)
	if pq.isRequestSized {
		//nolint:gosec
		assert.Equal(t, int64(retrievedWI-oldRI), migratedMeta.QueueSize)
	} else {
		assert.Equal(t, int64(oldQS), migratedMeta.QueueSize)
	}

	// 5. Verify old format keys are gone or ignored.
	// The current implementation doesn't delete old keys automatically.
	// So we just verify operations update the new key

	newItem := uint64(200)
	err = pq.Offer(context.Background(), newItem)
	require.NoError(t, err)

	pq.mu.Lock()
	exceptedWriteIndexAfterPut := retrievedWI + 1
	exceptedQueueSizeAfterPut := pq.queueSize // This was updated in Offer/putInternal
	pq.mu.Unlock()

	newMetaBytesAfterPut, err := client.Get(context.Background(), queueMetadataKey)
	require.NoError(t, err)
	metaAfterPut, err := unmarshalQueueMetadata(newMetaBytesAfterPut)
	require.NoError(t, err)
	assert.Equal(t, exceptedWriteIndexAfterPut, metaAfterPut.WriteIndex)
	assert.Equal(t, exceptedQueueSizeAfterPut, metaAfterPut.QueueSize) // queueSize in metadata reflects the sizer

	// Shutdown
	require.NoError(t, pq.Shutdown(context.Background()))
	require.NoError(t, mockStorage.Shutdown(context.Background()))
}

// Helper functions for old format data, similar to what was removed from main code
func itemIndexArrayToBytes(arr []uint64) []byte {
	size := len(arr)
	buf := make([]byte, 0, 4+size*8)
	//nolint:gosec
	buf = binary.LittleEndian.AppendUint32(buf, uint32(size))
	for _, item := range arr {
		buf = binary.LittleEndian.AppendUint64(buf, item)
	}
	return buf
}

func itemIndexToBytes(value uint64) []byte {
	return binary.LittleEndian.AppendUint64([]byte{}, value)
}
