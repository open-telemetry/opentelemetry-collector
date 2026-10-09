// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/internal/metadata"
)

func TestMoveAndAppendToCopiesAcrossStates(t *testing.T) {
	prev := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), true))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), prev))
	})

	src := NewTraces()
	src.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("src")
	dest := NewTraces()
	src.ResourceSpans().MoveAndAppendTo(dest.ResourceSpans())
	assert.Equal(t, 0, src.ResourceSpans().Len())
	require.Equal(t, 1, dest.ResourceSpans().Len())
	assert.Equal(t, "src", dest.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Name())

	// Dropping src must not invalidate dest after a cross-state copy.
	src.getState().DropArena()
	assert.Equal(t, "src", dest.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Name())
}

func TestMoveAndAppendToStealsWithoutArena(t *testing.T) {
	prev := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), false))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), prev))
	})

	src := NewTraces()
	src.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("src")
	moved := src.ResourceSpans().At(0).orig
	dest := NewTraces()
	src.ResourceSpans().MoveAndAppendTo(dest.ResourceSpans())
	assert.Equal(t, 0, src.ResourceSpans().Len())
	require.Equal(t, 1, dest.ResourceSpans().Len())

	// Without an arena the data is plain heap data, so dest takes it rather than copying it.
	assert.Same(t, moved, dest.ResourceSpans().At(0).orig)
}

// An arena is a []byte the garbage collector never scans for pointers, so a destination that
// owns one cannot take pointers to heap data: nothing would keep that data alive. The copy is
// required even though the source has no arena of its own to recycle.
func TestMoveAndAppendToCopiesHeapDataIntoArena(t *testing.T) {
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), false))
	})

	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), false))
	src := NewTraces()
	src.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("src")
	moved := src.ResourceSpans().At(0).orig

	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), true))
	dest := NewTraces()
	src.ResourceSpans().MoveAndAppendTo(dest.ResourceSpans())

	require.Equal(t, 1, dest.ResourceSpans().Len())
	assert.Equal(t, "src", dest.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Name())
	assert.NotSame(t, moved, dest.ResourceSpans().At(0).orig,
		"dest took a pointer to heap data its arena cannot keep alive")
}

func TestUnmarshalProtoBorrowsWireBuffer(t *testing.T) {
	prev := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), true))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), prev))
	})

	td := NewTraces()
	td.ResourceSpans().AppendEmpty().Resource().Attributes().PutStr("service.name", "svc")
	buf, err := (&ProtoMarshaler{}).MarshalTraces(td)
	require.NoError(t, err)

	got, err := (&ProtoUnmarshaler{}).UnmarshalTraces(buf)
	require.NoError(t, err)
	assert.Equal(t, "svc", got.ResourceSpans().At(0).Resource().Attributes().AsRaw()["service.name"])
}

func TestByteSliceCopyOnWriteAfterUnmarshal(t *testing.T) {
	prev := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), true))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), prev))
	})

	td := NewTraces()
	td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().TraceState().FromRaw("orig")
	buf, err := (&ProtoMarshaler{}).MarshalTraces(td)
	require.NoError(t, err)
	orig := append([]byte(nil), buf...)

	got, err := (&ProtoUnmarshaler{}).UnmarshalTraces(buf)
	require.NoError(t, err)
	sp := got.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	sp.TraceState().FromRaw("mutated")
	assert.Equal(t, orig, buf)
	assert.Equal(t, "mutated", sp.TraceState().AsRaw())
}
