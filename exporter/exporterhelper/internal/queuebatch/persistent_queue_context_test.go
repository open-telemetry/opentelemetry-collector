// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/hosttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/storagetest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pipeline"
)

// f is an implementation of Encoding that always fails on Marshal and Unmarshal.
type failingEncoding[T any] struct{}

// Marshal always returns an error.
func (failingEncoding[T]) Marshal(_ T) ([]byte, error) {
	return nil, errors.New("failing encoding: Marshal failed")
}

// Unmarshal always returns an error.
func (failingEncoding[T]) Unmarshal(_ []byte) (T, error) {
	var zero T
	return zero, errors.New("failing encoding: Unmarshal failed")
}

func createTestPersistentQueueWithEncoding(tb testing.TB, ext storage.Extension, capacity int64, encoding Encoding[uint64]) *persistentQueue[uint64] {
	pq := newPersistentQueue[uint64](persistentQueueSettings[uint64]{
		sizer:     request.RequestsSizer[uint64]{},
		capacity:  capacity,
		signal:    pipeline.SignalTraces,
		storageID: component.ID{},
		encoding:  encoding,
		id:        component.NewID(exportertest.NopType),
		telemetry: componenttest.NewNopTelemetrySettings(),
	}).(*persistentQueue[uint64])
	require.NoError(tb, pq.Start(context.Background(), hosttest.NewHost(map[component.ID]component.Component{{}: ext})))
	return pq
}

func TestSpanContextFromContextAndBack(t *testing.T) {
	tid := trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	sid := trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8}
	flags := trace.FlagsSampled
	state, _ := trace.ParseTraceState("foo=bar,baz=qux")
	remote := true

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: flags,
		TraceState: state,
		Remote:     remote,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	// Test spanContextFromContext
	local := localSpanContextFromTraceSpanContext(trace.SpanContextFromContext(ctx))
	assert.Equal(t, tid.String(), local.TraceID)
	assert.Equal(t, sid.String(), local.SpanID)
	assert.Equal(t, flags.String(), local.TraceFlags)
	assert.Equal(t, state.String(), local.TraceState)
	assert.Equal(t, remote, local.Remote)

	// Test contextWithLocalSpanContext
	ctx2 := contextWithLocalSpanContext(context.Background(), local)
	sc2 := trace.SpanContextFromContext(ctx2)
	assert.Equal(t, tid, sc2.TraceID())
	assert.Equal(t, sid, sc2.SpanID())
	assert.Equal(t, flags, sc2.TraceFlags())
	assert.Equal(t, state, sc2.TraceState())
	assert.Equal(t, remote, sc2.IsRemote())
}

func TestLocalSpanContextFromTraceSpanContext(t *testing.T) {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0x00},
		SpanID:     trace.SpanID{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
		TraceFlags: trace.FlagsSampled,
		TraceState: trace.TraceState{},
		Remote:     false,
	})
	local := localSpanContextFromTraceSpanContext(sc)
	assert.Equal(t, sc.TraceID().String(), local.TraceID)
	assert.Equal(t, sc.SpanID().String(), local.SpanID)
	assert.Equal(t, sc.TraceFlags().String(), local.TraceFlags)
	assert.Equal(t, sc.TraceState().String(), local.TraceState)
	assert.Equal(t, sc.IsRemote(), local.Remote)
}

func TestContextWithLocalSpanContext_InvalidHex(t *testing.T) {
	ctx := context.Background()
	bad := spanContext{
		TraceID:    "nothex",
		SpanID:     "nothex",
		TraceFlags: "nothex",
		TraceState: "invalid=state",
		Remote:     false,
	}
	ctx2 := contextWithLocalSpanContext(ctx, bad)
	// Should return the original context (no span context injected)
	sc := trace.SpanContextFromContext(ctx2)
	assert.False(t, sc.IsValid())
}

func TestPersistentQueue_SpanContextRoundTrip(t *testing.T) {
	require.NoError(t, featuregate.GlobalRegistry().Set("exporter.PersistRequestContext", true))
	defer func() {
		require.NoError(t, featuregate.GlobalRegistry().Set("exporter.PersistRequestContext", false))
	}()
	// Setup a minimal persistent queue using uint64Encoding and uint64
	pq := newPersistentQueue[uint64](persistentQueueSettings[uint64]{
		sizer:     request.RequestsSizer[uint64]{},
		capacity:  10,
		signal:    pipeline.SignalTraces,
		storageID: component.ID{},
		encoding:  uint64Encoding{},
		id:        component.NewID(exportertest.NopType),
		telemetry: componenttest.NewNopTelemetrySettings(),
	}).(*persistentQueue[uint64])

	ext := storagetest.NewMockStorageExtension(nil)
	client, err := ext.GetClient(context.Background(), component.KindExporter, pq.set.id, pq.set.signal.String())
	require.NoError(t, err)
	pq.initClient(context.Background(), client)

	// Create a valid SpanContext
	traceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	spanID, _ := trace.SpanIDFromHex("0102030405060708")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: 0x01,
		TraceState: trace.TraceState{},
		Remote:     true,
	})
	ctxWithSC := trace.ContextWithSpanContext(context.Background(), sc)

	// Offer a request with this context
	req := uint64(42)
	require.NoError(t, pq.Offer(ctxWithSC, req))

	// Read the request and restored context
	restoredCtx, gotReq, _, ok := pq.Read(context.Background())
	require.True(t, ok)
	assert.Equal(t, req, gotReq)
	restoredSC := trace.SpanContextFromContext(restoredCtx)
	assert.True(t, restoredSC.IsValid())
	assert.Equal(t, sc.TraceID(), restoredSC.TraceID())
	assert.Equal(t, sc.SpanID(), restoredSC.SpanID())
	assert.Equal(t, sc.TraceFlags(), restoredSC.TraceFlags())
	assert.Equal(t, sc.TraceState().String(), restoredSC.TraceState().String())
	assert.Equal(t, sc.IsRemote(), restoredSC.IsRemote())

	// Also test with a context with no SpanContext
	req2 := uint64(99)
	require.NoError(t, pq.Offer(context.Background(), req2))
	restoredCtx2, gotReq2, _, ok2 := pq.Read(context.Background())
	require.True(t, ok2)
	assert.Equal(t, req2, gotReq2)
	restoredSC2 := trace.SpanContextFromContext(restoredCtx2)
	assert.False(t, restoredSC2.IsValid())
}

func TestSpanContextUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		name          string
		expectNil     bool
		expectErr     bool
		marshaledData []byte
	}{
		{
			name:          "Invalid JSON data",
			expectNil:     true,
			expectErr:     true,
			marshaledData: []byte(`{"invalid json}`),
		},
		{
			name:          "Valid JSON but not valid spanContext",
			expectNil:     true,
			expectErr:     false,
			marshaledData: []byte(`{"foo":"bar"}`),
		},
		{
			name:          "Valid JSON with spanContext fields but invalid SC",
			expectNil:     true,
			expectErr:     false,
			marshaledData: []byte(`{"TraceID": "00000000000000000000000000000000","SpanID":"00000000000000000","TraceFlags":"0","TraceState":"0","Remote":false}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var rc requestContext
			err := json.Unmarshal(tc.marshaledData, &rc)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tc.expectNil {
				require.Empty(t, rc.SpanContext.TraceID)
				require.Empty(t, rc.SpanContext.SpanID)
				require.Empty(t, rc.SpanContext.TraceFlags)
				require.Empty(t, rc.SpanContext.TraceState)
				require.False(t, rc.SpanContext.Remote)
			}
		})
	}
}

func TestPersistentQueue_PutInternal_FailingEncoding(t *testing.T) {
	ext := storagetest.NewMockStorageExtension(nil)
	pq := createTestPersistentQueueWithItemsCapacity(t, ext, 1000)
	pq.set.encoding = failingEncoding[uint64]{}

	err := pq.Offer(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failing encoding: Marshal failed")
}

func TestPersistentQueue_GetNextItem_UnmarshalError(t *testing.T) {
	ext := storagetest.NewMockStorageExtension(nil)
	failingEncoding := &failingEncoding[uint64]{}
	ps := createTestPersistentQueueWithEncoding(t, ext, 1000, uint64Encoding{})

	// Add an item to the queue
	req := uint64(50)
	require.NoError(t, ps.Offer(context.Background(), req))

	// Attempt to read the item, expecting an Unmarshal error
	ctx := context.Background()
	// change encoding to failingEncoding so Unmarshal fails
	ps.set.encoding = failingEncoding
	_, _, _, _, err := ps.getNextItem(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failing encoding: Unmarshal failed")
}

func TestPersistentQueue_RetrieveAndEnqueueNotDispatchedReqs_ContextRestoration(t *testing.T) {
	t.Run("FeatureGateDisabled", func(t *testing.T) {
		// Disable the feature gate
		require.NoError(t, featuregate.GlobalRegistry().Set("exporter.PersistRequestContext", false))
		defer func() {
			require.NoError(t, featuregate.GlobalRegistry().Set("exporter.PersistRequestContext", true))
		}()

		client := newFakeBoundedStorageClient(1000)
		pq := createTestPersistentQueueWithClient(client)

		// Add items with context
		ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    [16]byte{1, 2, 3, 4},
			SpanID:     [8]byte{5, 6, 7, 8},
			TraceFlags: trace.FlagsSampled,
		}))
		for i := uint64(0); i < 3; i++ {
			require.NoError(t, pq.putInternal(ctx, i))
		}

		// Simulate items being dispatched
		pq.currentlyDispatchedItems = []uint64{0, 1, 2}

		// Retrieve and enqueue
		pq.retrieveAndEnqueueNotDispatchedReqs(context.Background())

		// Verify items were moved back to queue without context
		for i := uint64(0); i < 3; i++ {
			val, err := client.Get(context.Background(), getItemKey(i))
			require.NoError(t, err)
			require.NotNil(t, val)

			val, err = client.Get(context.Background(), getContextKey(i))
			require.NoError(t, err)
			require.Empty(t, val)
		}
	})

	t.Run("FeatureGateEnabled", func(t *testing.T) {
		// Enable the feature gate
		require.NoError(t, featuregate.GlobalRegistry().Set("exporter.PersistRequestContext", true))

		client := newFakeBoundedStorageClient(1000)
		pq := createTestPersistentQueueWithClient(client)

		// Add items with context
		ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    [16]byte{1, 2, 3, 4},
			SpanID:     [8]byte{5, 6, 7, 8},
			TraceFlags: trace.FlagsSampled,
		}))
		for i := uint64(0); i < 3; i++ {
			require.NoError(t, pq.putInternal(ctx, i))
		}

		// Simulate items being dispatched
		pq.currentlyDispatchedItems = []uint64{0, 1, 2}

		// Retrieve and enqueue
		pq.retrieveAndEnqueueNotDispatchedReqs(context.Background())

		// Verify items were moved back to queue with context
		for i := uint64(0); i < 3; i++ {
			val, err := client.Get(context.Background(), getItemKey(i))
			require.NoError(t, err)
			require.NotNil(t, val)

			ctxVal, err := client.Get(context.Background(), getContextKey(i))
			require.NoError(t, err)
			require.NotNil(t, ctxVal)
		}
	})
}

func TestPersistentQueue_FeatureGate_ContextPersistence(t *testing.T) {
	// Test with feature gate disabled
	t.Run("FeatureGateDisabled", func(t *testing.T) {
		// Disable the feature gate
		require.NoError(t, featuregate.GlobalRegistry().Set("exporter.PersistRequestContext", false))
		defer func() {
			require.NoError(t, featuregate.GlobalRegistry().Set("exporter.PersistRequestContext", true))
		}()

		client := newFakeBoundedStorageClient(1000)
		pq := createTestPersistentQueueWithClient(client)

		// Add an item with context
		ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    [16]byte{1, 2, 3, 4},
			SpanID:     [8]byte{5, 6, 7, 8},
			TraceFlags: trace.FlagsSampled,
		}))
		require.NoError(t, pq.putInternal(ctx, 1))

		// Verify only request key exists, no context key
		val, err := client.Get(context.Background(), getItemKey(0))
		require.NoError(t, err)
		require.NotNil(t, val)

		val, err = client.Get(context.Background(), getContextKey(0))
		require.NoError(t, err)
		require.Empty(t, val)
	})

	// Test with feature gate enabled
	t.Run("FeatureGateEnabled", func(t *testing.T) {
		// Enable the feature gate
		require.NoError(t, featuregate.GlobalRegistry().Set("exporter.PersistRequestContext", true))

		client := newFakeBoundedStorageClient(1000)
		pq := createTestPersistentQueueWithClient(client)

		// Add an item with context
		ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    [16]byte{1, 2, 3, 4},
			SpanID:     [8]byte{5, 6, 7, 8},
			TraceFlags: trace.FlagsSampled,
		}))
		require.NoError(t, pq.putInternal(ctx, 1))

		// Verify both request and context keys exist
		val, err := client.Get(context.Background(), getItemKey(0))
		require.NoError(t, err)
		require.NotNil(t, val)

		ctxVal, err := client.Get(context.Background(), getContextKey(0))
		require.NoError(t, err)
		require.NotNil(t, ctxVal)
	})
}

func TestPersistentQueue_FeatureGate_BatchOperations(t *testing.T) {
	// Test with feature gate disabled
	t.Run("FeatureGateDisabled", func(t *testing.T) {
		// Disable the feature gate
		require.NoError(t, featuregate.GlobalRegistry().Set("exporter.PersistRequestContext", false))
		defer func() {
			require.NoError(t, featuregate.GlobalRegistry().Set("exporter.PersistRequestContext", true))
		}()

		client := newFakeBoundedStorageClient(1000)
		pq := createTestPersistentQueueWithClient(client)

		// Add multiple items
		for i := uint64(0); i < 3; i++ {
			require.NoError(t, pq.putInternal(context.Background(), i))
		}

		// Verify batch operations only include request keys
		ops := make([]*storage.Operation, 0)
		for i := uint64(0); i < 3; i++ {
			ops = append(ops, storage.GetOperation(getItemKey(i)))
		}
		require.NoError(t, client.Batch(context.Background(), ops...))

		// Verify no context keys exist
		for i := uint64(0); i < 3; i++ {
			_, err := client.Get(context.Background(), getContextKey(i))
			require.NoError(t, err)
		}
	})

	// Test with feature gate enabled
	t.Run("FeatureGateEnabled", func(t *testing.T) {
		// Enable the feature gate
		require.NoError(t, featuregate.GlobalRegistry().Set("exporter.PersistRequestContext", true))

		client := newFakeBoundedStorageClient(1000)
		pq := createTestPersistentQueueWithClient(client)

		// Add multiple items with context
		ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    [16]byte{1, 2, 3, 4},
			SpanID:     [8]byte{5, 6, 7, 8},
			TraceFlags: trace.FlagsSampled,
		}))
		for i := uint64(0); i < 3; i++ {
			require.NoError(t, pq.putInternal(ctx, i))
		}

		// Verify batch operations include both request and context keys
		ops := make([]*storage.Operation, 0)
		for i := uint64(0); i < 3; i++ {
			ops = append(ops, storage.GetOperation(getItemKey(i)))
			ops = append(ops, storage.GetOperation(getContextKey(i)))
		}
		require.NoError(t, client.Batch(context.Background(), ops...))
	})
}

func TestPersistentQueue_GetNextItem_BatchError(t *testing.T) {
	// Create a storage client that will fail on Batch operations
	client := &fakeStorageClientWithErrors{
		errors: []error{errors.New("batch operation failed")},
	}
	pq := createTestPersistentQueueWithClient(client)

	req := uint64(50)
	require.NoError(t, pq.Offer(context.Background(), req))

	// Shutdown the queue after a short delay to unblock Read
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = pq.Shutdown(context.Background())
	}()

	restoredCtx, readReq, done, found := pq.Read(context.Background())

	// Should return false for found since the read failed and queue was shutdown
	assert.False(t, found)
	assert.Equal(t, uint64(0), readReq)
	assert.Nil(t, done)
	assert.Equal(t, context.Background(), restoredCtx)
}

func TestPersistentQueue_RetrieveAndEnqueueNotDispatchedReqs_SpanContextUnmarshal(t *testing.T) {
	// Enable the feature gate
	require.NoError(t, featuregate.GlobalRegistry().Set("exporter.PersistRequestContext", true))
	defer func() {
		require.NoError(t, featuregate.GlobalRegistry().Set("exporter.PersistRequestContext", false))
	}()

	t.Run("SuccessfulUnmarshal", func(t *testing.T) {
		ext := storagetest.NewMockStorageExtension(nil)
		pq := createTestPersistentQueueWithRequestsCapacity(t, ext, 1000)

		// Create a valid span context
		validSC := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    [16]byte{1, 2, 3, 4},
			SpanID:     [8]byte{5, 6, 7, 8},
			TraceFlags: trace.FlagsSampled,
		})
		ctx := trace.ContextWithSpanContext(context.Background(), validSC)

		// Add items with the valid span context
		req1 := uint64(50)
		req2 := uint64(60)
		require.NoError(t, pq.Offer(ctx, req1))
		require.NoError(t, pq.Offer(ctx, req2))

		// Manually mark items as dispatched (simulating them being read but not finished)
		pq.mu.Lock()
		pq.currentlyDispatchedItems = []uint64{0, 1}
		// Update the storage with currently dispatched items
		require.NoError(t, pq.client.Set(context.Background(), currentlyDispatchedItemsKey, itemIndexArrayToBytes(pq.currentlyDispatchedItems)))
		pq.mu.Unlock()

		// Shutdown the queue
		require.NoError(t, pq.Shutdown(context.Background()))

		// Create a new queue from the same storage - this should trigger retrieveAndEnqueueNotDispatchedReqs
		newPQ := createTestPersistentQueueWithRequestsCapacity(t, ext, 1000)

		// Verify items were moved back to the queue with context preserved
		assert.Equal(t, int64(2), newPQ.Size())

		// Read the first item and verify context is preserved
		restoredCtx1, readReq1, done1, found1 := newPQ.Read(context.Background())
		require.True(t, found1)
		assert.Equal(t, req1, readReq1)
		assert.NotNil(t, done1)

		restoredSC1 := trace.SpanContextFromContext(restoredCtx1)
		assert.True(t, restoredSC1.IsValid())
		assert.Equal(t, validSC.TraceID(), restoredSC1.TraceID())
		assert.Equal(t, validSC.SpanID(), restoredSC1.SpanID())

		// Read the second item and verify context is preserved
		restoredCtx2, readReq2, done2, found2 := newPQ.Read(context.Background())
		require.True(t, found2)
		assert.Equal(t, req2, readReq2)
		assert.NotNil(t, done2)

		restoredSC2 := trace.SpanContextFromContext(restoredCtx2)
		assert.True(t, restoredSC2.IsValid())
		assert.Equal(t, validSC.TraceID(), restoredSC2.TraceID())
		assert.Equal(t, validSC.SpanID(), restoredSC2.SpanID())
	})

	t.Run("FailedUnmarshal", func(t *testing.T) {
		ext := storagetest.NewMockStorageExtension(nil)
		pq := createTestPersistentQueueWithRequestsCapacity(t, ext, 1000)

		// Create a valid span context
		validSC := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    [16]byte{1, 2, 3, 4},
			SpanID:     [8]byte{5, 6, 7, 8},
			TraceFlags: trace.FlagsSampled,
		})
		ctx := trace.ContextWithSpanContext(context.Background(), validSC)

		// Add items with the valid span context
		req1 := uint64(50)
		req2 := uint64(60)
		require.NoError(t, pq.Offer(ctx, req1))
		require.NoError(t, pq.Offer(ctx, req2))

		// Manually mark items as dispatched (simulating them being read but not finished)
		pq.mu.Lock()
		pq.currentlyDispatchedItems = []uint64{0, 1}
		// Update the storage with currently dispatched items
		require.NoError(t, pq.client.Set(context.Background(), currentlyDispatchedItemsKey, itemIndexArrayToBytes(pq.currentlyDispatchedItems)))
		pq.mu.Unlock()

		// Corrupt the stored context data with invalid JSON
		corruptedData := []byte(`{"invalid": "json"}`)
		require.NoError(t, pq.client.Set(context.Background(), getContextKey(0), corruptedData))
		require.NoError(t, pq.client.Set(context.Background(), getContextKey(1), corruptedData))

		// Shutdown the queue
		require.NoError(t, pq.Shutdown(context.Background()))

		// Create a new queue from the same storage - this should trigger retrieveAndEnqueueNotDispatchedReqs
		newPQ := createTestPersistentQueueWithRequestsCapacity(t, ext, 1000)

		// Verify items were moved back to the queue but context is not preserved
		assert.Equal(t, int64(2), newPQ.Size())

		// Read the first item and verify context is not preserved
		restoredCtx1, readReq1, done1, found1 := newPQ.Read(context.Background())
		require.True(t, found1)
		assert.Equal(t, req1, readReq1)
		assert.NotNil(t, done1)

		restoredSC1 := trace.SpanContextFromContext(restoredCtx1)
		assert.False(t, restoredSC1.IsValid())

		// Read the second item and verify context is not preserved
		restoredCtx2, readReq2, done2, found2 := newPQ.Read(context.Background())
		require.True(t, found2)
		assert.Equal(t, req2, readReq2)
		assert.NotNil(t, done2)

		restoredSC2 := trace.SpanContextFromContext(restoredCtx2)
		assert.False(t, restoredSC2.IsValid())
	})
}

func TestPersistentQueue_PutInternal_ContextTimeout(t *testing.T) {
	// Create a blocking queue
	pq := newPersistentQueue[uint64](persistentQueueSettings[uint64]{
		sizer:           request.RequestsSizer[uint64]{},
		capacity:        2,
		blockOnOverflow: true,
		signal:          pipeline.SignalTraces,
		storageID:       component.ID{},
		encoding:        uint64Encoding{},
		id:              component.NewID(exportertest.NopType),
		telemetry:       componenttest.NewNopTelemetrySettings(),
	}).(*persistentQueue[uint64])

	// Initialize with storage
	ext := storagetest.NewMockStorageExtension(nil)
	require.NoError(t, pq.Start(context.Background(), hosttest.NewHost(map[component.ID]component.Component{{}: ext})))

	// Fill the queue to capacity
	require.NoError(t, pq.Offer(context.Background(), uint64(1)))
	require.NoError(t, pq.Offer(context.Background(), uint64(2)))

	// Try to add another item with a very short timeout
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// This should fail with context deadline exceeded
	err := pq.Offer(timeoutCtx, uint64(3))
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestPersistentQueue_GetNextItem_ContextUnmarshalFails(t *testing.T) {
	require.NoError(t, featuregate.GlobalRegistry().Set("exporter.PersistRequestContext", true))
	defer func() {
		require.NoError(t, featuregate.GlobalRegistry().Set("exporter.PersistRequestContext", false))
	}()

	ext := storagetest.NewMockStorageExtension(nil)
	pq := createTestPersistentQueueWithRequestsCapacity(t, ext, 1000)

	// Insert an item with a valid span context
	tid, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	sid, _ := trace.SpanIDFromHex("0102030405060708")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: 0x01,
		TraceState: trace.TraceState{},
		Remote:     true,
	})
	ctxWithSC := trace.ContextWithSpanContext(context.Background(), sc)
	req := uint64(42)
	require.NoError(t, pq.Offer(ctxWithSC, req))

	// Corrupt the stored context for index 0
	corrupted := []byte(`{"notjson}`)
	require.NoError(t, pq.client.Set(context.Background(), getContextKey(0), corrupted))

	// Now getNextItem should succeed for the request, but context unmarshal will fail
	_, gotReq, _, restoredCtx, err := pq.getNextItem(context.Background())
	require.Error(t, err, "unexpected end of JSON input")
	require.Equal(t, req, gotReq)
	restoredSC := trace.SpanContextFromContext(restoredCtx)
	require.False(t, restoredSC.IsValid(), "restored context should not have a valid span context if context unmarshal fails")
}

func TestPersistentQueue_RetrieveAndEnqueueNotDispatchedReqs_ContextUnmarshalFails(t *testing.T) {
	require.NoError(t, featuregate.GlobalRegistry().Set("exporter.PersistRequestContext", true))
	defer func() {
		require.NoError(t, featuregate.GlobalRegistry().Set("exporter.PersistRequestContext", false))
	}()

	ext := storagetest.NewMockStorageExtension(nil)
	pq := createTestPersistentQueueWithRequestsCapacity(t, ext, 1000)

	// Insert an item with a valid span context
	tid, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	sid, _ := trace.SpanIDFromHex("0102030405060708")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: 0x01,
		TraceState: trace.TraceState{},
		Remote:     true,
	})
	ctxWithSC := trace.ContextWithSpanContext(context.Background(), sc)
	req := uint64(42)
	require.NoError(t, pq.Offer(ctxWithSC, req))

	// Simulate the item being dispatched
	pq.mu.Lock()
	pq.currentlyDispatchedItems = []uint64{0}
	require.NoError(t, pq.client.Set(context.Background(), currentlyDispatchedItemsKey, itemIndexArrayToBytes(pq.currentlyDispatchedItems)))
	pq.mu.Unlock()

	// Corrupt the stored context for index 0
	corrupted := []byte(`{"notjson}`)
	require.NoError(t, pq.client.Set(context.Background(), getContextKey(0), corrupted))
	pq.writeIndex--
	require.NoError(t, pq.client.Set(context.Background(), writeIndexKey, itemIndexToBytes(pq.writeIndex)))
	// Call retrieveAndEnqueueNotDispatchedReqs, which should hit the unmarshal error
	pq.retrieveAndEnqueueNotDispatchedReqs(context.Background())

	// The item should be re-enqueued, but its context should not be valid
	restoredCtx, gotReq, _, ok := pq.Read(context.Background())
	require.True(t, ok)
	require.Equal(t, req, gotReq)
	restoredSC := trace.SpanContextFromContext(restoredCtx)
	require.False(t, restoredSC.IsValid(), "restored context should not have a valid span context if context unmarshal fails")
}

func TestTraceFlagsFromHex(t *testing.T) {
	// Valid 1-byte hex string
	flags, err := traceFlagsFromHex("01")
	require.NoError(t, err)
	require.NotNil(t, flags)
	require.Equal(t, trace.TraceFlags(1), *flags)

	// Invalid hex string (should error from hex.DecodeString)
	_, err = traceFlagsFromHex("zz")
	require.Error(t, err)

	// Valid hex but wrong length (should error on len(decoded) != 1)
	_, err = traceFlagsFromHex("0102")
	require.Error(t, err)
	require.Contains(t, err.Error(), errInvalidTraceFlagsLength)
}

func TestContextWithLocalSpanContext_AllBranches(t *testing.T) {
	// Valid case
	tid := "0102030405060708090a0b0c0d0e0f10"
	sid := "0102030405060708"
	flags := "01"
	state := "foo=bar"
	sc := spanContext{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: flags,
		TraceState: state,
		Remote:     true,
	}
	ctx := contextWithLocalSpanContext(context.Background(), sc)
	span := trace.SpanContextFromContext(ctx)
	require.True(t, span.IsValid())
	require.Equal(t, tid, span.TraceID().String())
	require.Equal(t, sid, span.SpanID().String())
	require.Equal(t, trace.TraceFlags(1), span.TraceFlags())
	require.Equal(t, state, span.TraceState().String())
	require.True(t, span.IsRemote())

	// Invalid TraceID
	scBadTraceID := sc
	scBadTraceID.TraceID = "nothex"
	ctx = contextWithLocalSpanContext(context.Background(), scBadTraceID)
	span = trace.SpanContextFromContext(ctx)
	require.False(t, span.IsValid())

	// Invalid SpanID
	scBadSpanID := sc
	scBadSpanID.SpanID = "nothex"
	ctx = contextWithLocalSpanContext(context.Background(), scBadSpanID)
	span = trace.SpanContextFromContext(ctx)
	require.False(t, span.IsValid())

	// Invalid TraceFlags (not hex)
	scBadFlags := sc
	scBadFlags.TraceFlags = "zz"
	ctx = contextWithLocalSpanContext(context.Background(), scBadFlags)
	span = trace.SpanContextFromContext(ctx)
	require.False(t, span.IsValid())

	// Invalid TraceFlags (wrong length)
	scBadFlagsLen := sc
	scBadFlagsLen.TraceFlags = "0102"
	ctx = contextWithLocalSpanContext(context.Background(), scBadFlagsLen)
	span = trace.SpanContextFromContext(ctx)
	require.False(t, span.IsValid())

	// Invalid TraceState
	scBadState := sc
	scBadState.TraceState = "bad=tracestate,=bad"
	ctx = contextWithLocalSpanContext(context.Background(), scBadState)
	span = trace.SpanContextFromContext(ctx)
	require.False(t, span.IsValid())
}

func TestGetAndMarshalSpanContext_FeatureGateDisabled(t *testing.T) {
	require.NoError(t, featuregate.GlobalRegistry().Set("exporter.PersistRequestContext", false))
	defer func() {
		require.NoError(t, featuregate.GlobalRegistry().Set("exporter.PersistRequestContext", true))
	}()

	data, err := getAndMarshalSpanContext(context.Background())
	require.NoError(t, err)
	require.Nil(t, data)
}

func TestPersistentQueue_Read_ItemDispatchingFinishError(t *testing.T) {
	// This error will be returned by the fake client on the second Batch call (itemDispatchingFinish)
	errItemFinish := errors.New("itemDispatchingFinish error")
	client := &fakeStorageClientWithErrors{
		errors: []error{nil, errItemFinish},
	}
	pq := createTestPersistentQueueWithClient(client)

	req := uint64(50)
	require.NoError(t, pq.Offer(context.Background(), req))

	// Shutdown the queue after a short delay to unblock Read
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = pq.Shutdown(context.Background())
	}()

	restoredCtx, readReq, done, found := pq.Read(context.Background())

	// Should return false for found since the read failed and queue was shutdown
	assert.False(t, found)
	assert.Equal(t, uint64(0), readReq)
	assert.Nil(t, done)
	assert.Equal(t, context.Background(), restoredCtx)
}

func TestPersistentQueue_Read_ItemDispatchingFinishErrorPath(t *testing.T) {
	errItemFinish := errors.New("itemDispatchingFinish error")
	client := &fakeStorageClientWithErrors{
		errors: []error{nil, nil, nil, errItemFinish, errItemFinish, errItemFinish}, // First Batch (getNextItem) succeeds, second (itemDispatchingFinish) fails
	}
	pq := createTestPersistentQueueWithClient(client)

	req := uint64(50)
	require.NoError(t, pq.Offer(context.Background(), req))

	// Shutdown the queue after a short delay to unblock Read
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = pq.Shutdown(context.Background())
	}()

	_, _, _, _ = pq.Read(context.Background()) // The error is only logged, not returned, so we can't assert on output
}
