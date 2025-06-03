// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"encoding/json"
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
	require.Equal(t, tid.String(), local.TraceID)
	require.Equal(t, sid.String(), local.SpanID)
	require.Equal(t, flags.String(), local.TraceFlags)
	require.Equal(t, state.String(), local.TraceState)
	require.Equal(t, remote, local.Remote)

	// Test contextWithLocalSpanContext
	ctx2 := contextWithLocalSpanContext(context.Background(), local)
	sc2 := trace.SpanContextFromContext(ctx2)
	require.Equal(t, tid, sc2.TraceID())
	require.Equal(t, sid, sc2.SpanID())
	require.Equal(t, flags, sc2.TraceFlags())
	require.Equal(t, state, sc2.TraceState())
	require.Equal(t, remote, sc2.IsRemote())
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
	require.Equal(t, sc.TraceID().String(), local.TraceID)
	require.Equal(t, sc.SpanID().String(), local.SpanID)
	require.Equal(t, sc.TraceFlags().String(), local.TraceFlags)
	require.Equal(t, sc.TraceState().String(), local.TraceState)
	require.Equal(t, sc.IsRemote(), local.Remote)
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
	require.False(t, sc.IsValid())
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
