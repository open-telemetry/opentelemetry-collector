// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/storagetest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pipeline"
)

func TestGetAndMarshalSpanContext_FeatureGateDisabled(t *testing.T) {
	data := marshalSpanContext(context.Background())
	require.Nil(t, data)
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
	corrupted := []byte(`corrupted`)
	require.NoError(t, pq.client.Set(context.Background(), getContextKey(0), corrupted))

	// Now getNextItem should succeed for the request, but context unmarshal will fail
	_, gotReq, _, restoredCtx, err := pq.getNextItem(context.Background())
	require.NoError(t, err)
	require.Equal(t, req, gotReq)
	restoredSC := trace.SpanContextFromContext(restoredCtx)
	require.False(t, restoredSC.IsValid(), "restored context should not have a valid span context if context unmarshal fails")
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
		pq.metadata.CurrentlyDispatchedItems = []uint64{0, 1}
		// Update the storage with currently dispatched items
		require.NoError(t, pq.client.Set(context.Background(), currentlyDispatchedItemsKey, itemIndexArrayToBytes(pq.metadata.CurrentlyDispatchedItems)))
		pq.mu.Unlock()

		// Shutdown the queue
		require.NoError(t, pq.Shutdown(context.Background()))

		// Create a new queue from the same storage - this should trigger retrieveAndEnqueueNotDispatchedReqs
		newPQ := createTestPersistentQueueWithRequestsCapacity(t, ext, 1000)

		// Verify items were moved back to the queue with context preserved
		require.Equal(t, int64(2), newPQ.Size())

		// Read the first item and verify context is preserved
		restoredCtx1, readReq1, done1, found1 := newPQ.Read(context.Background())
		require.True(t, found1)
		require.Equal(t, req1, readReq1)
		require.NotNil(t, done1)

		restoredSC1 := trace.SpanContextFromContext(restoredCtx1)
		require.True(t, restoredSC1.IsValid())
		require.Equal(t, validSC.TraceID(), restoredSC1.TraceID())
		require.Equal(t, validSC.SpanID(), restoredSC1.SpanID())

		// Read the second item and verify context is preserved
		restoredCtx2, readReq2, done2, found2 := newPQ.Read(context.Background())
		require.True(t, found2)
		require.Equal(t, req2, readReq2)
		require.NotNil(t, done2)

		restoredSC2 := trace.SpanContextFromContext(restoredCtx2)
		require.True(t, restoredSC2.IsValid())
		require.Equal(t, validSC.TraceID(), restoredSC2.TraceID())
		require.Equal(t, validSC.SpanID(), restoredSC2.SpanID())
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
		pq.metadata.CurrentlyDispatchedItems = []uint64{0, 1}
		// Update the storage with currently dispatched items
		require.NoError(t, pq.client.Set(context.Background(), currentlyDispatchedItemsKey, itemIndexArrayToBytes(pq.metadata.CurrentlyDispatchedItems)))
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
		require.Equal(t, int64(2), newPQ.Size())

		// Read the first item and verify context is not preserved
		restoredCtx1, readReq1, done1, found1 := newPQ.Read(context.Background())
		require.True(t, found1)
		require.Equal(t, req1, readReq1)
		require.NotNil(t, done1)

		restoredSC1 := trace.SpanContextFromContext(restoredCtx1)
		require.False(t, restoredSC1.IsValid())

		// Read the second item and verify context is not preserved
		restoredCtx2, readReq2, done2, found2 := newPQ.Read(context.Background())
		require.True(t, found2)
		require.Equal(t, req2, readReq2)
		require.NotNil(t, done2)

		restoredSC2 := trace.SpanContextFromContext(restoredCtx2)
		require.False(t, restoredSC2.IsValid())
	})
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
	require.Equal(t, req, gotReq)
	restoredSC := trace.SpanContextFromContext(restoredCtx)
	require.True(t, restoredSC.IsValid())
	require.Equal(t, sc.TraceID(), restoredSC.TraceID())
	require.Equal(t, sc.SpanID(), restoredSC.SpanID())
	require.Equal(t, sc.TraceFlags(), restoredSC.TraceFlags())
	require.Equal(t, sc.TraceState().String(), restoredSC.TraceState().String())
	require.Equal(t, sc.IsRemote(), restoredSC.IsRemote())

	// Also test with a context with no SpanContext
	req2 := uint64(99)
	require.NoError(t, pq.Offer(context.Background(), req2))
	restoredCtx2, gotReq2, _, ok2 := pq.Read(context.Background())
	require.True(t, ok2)
	require.Equal(t, req2, gotReq2)
	restoredSC2 := trace.SpanContextFromContext(restoredCtx2)
	require.False(t, restoredSC2.IsValid())
}
