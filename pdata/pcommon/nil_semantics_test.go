// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pcommon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/internal/metadata"
)

// An arena is never scanned by the garbage collector, so moves and copies between objects that do
// not share one have to allocate rather than hand over pointers. Swapping an assignment for a copy
// is easy to get subtly wrong: a zero-length slice where the original produced nil reads the same
// through Len and AsRaw, but consumers compare the underlying structs and see the difference. The
// tests below pin the nil-versus-empty result of every move and copy to what the implementation
// produced before the arena, with the gate both off and on.

func forEachGateState(t *testing.T, fn func(t *testing.T)) {
	t.Helper()
	for _, tc := range []struct {
		name string
		on   bool
	}{{"gate off", false}, {"gate on", true}} {
		t.Run(tc.name, func(t *testing.T) {
			prev := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
			require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), tc.on))
			t.Cleanup(func() {
				require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), prev))
			})
			fn(t)
		})
	}
}

// The emptied* helpers return a value holding a buffer but no elements. RemoveIf reslices in
// place, so it is the one way to reach that state that every type agrees on.

func emptiedUInt64Slice(t *testing.T) UInt64Slice {
	t.Helper()
	s := NewUInt64Slice()
	s.FromRaw([]uint64{1, 2, 3})
	s.RemoveIf(func(uint64) bool { return true })
	require.NotNil(t, *s.getOrig())
	return s
}

func emptiedStringSlice(t *testing.T) StringSlice {
	t.Helper()
	s := NewStringSlice()
	s.FromRaw([]string{"a", "b"})
	s.RemoveIf(func(string) bool { return true })
	require.NotNil(t, *s.getOrig())
	return s
}

func emptiedByteSlice(t *testing.T) ByteSlice {
	t.Helper()
	s := NewByteSlice()
	s.FromRaw([]byte{1, 2, 3})
	s.RemoveIf(func(byte) bool { return true })
	require.NotNil(t, *s.getOrig())
	return s
}

func emptiedMap(t *testing.T) Map {
	t.Helper()
	m := NewMap()
	m.PutStr("k", "v")
	m.RemoveIf(func(string, Value) bool { return true })
	require.NotNil(t, *m.getOrig())
	return m
}

func emptiedSlice(t *testing.T) Slice {
	t.Helper()
	s := NewSlice()
	s.AppendEmpty()
	s.RemoveIf(func(Value) bool { return true })
	require.NotNil(t, *s.getOrig())
	return s
}

// MoveTo gives the destination the source's own value, so an empty source leaves the destination
// nil no matter what it held before.
func TestMoveToLeavesDestinationNilForEmptySource(t *testing.T) {
	forEachGateState(t, func(t *testing.T) {
		t.Run("UInt64Slice", func(t *testing.T) {
			dest := emptiedUInt64Slice(t)
			NewUInt64Slice().MoveTo(dest)
			assert.Nil(t, *dest.getOrig())
		})
		t.Run("StringSlice", func(t *testing.T) {
			dest := emptiedStringSlice(t)
			NewStringSlice().MoveTo(dest)
			assert.Nil(t, *dest.getOrig())
		})
		t.Run("ByteSlice", func(t *testing.T) {
			dest := emptiedByteSlice(t)
			NewByteSlice().MoveTo(dest)
			assert.Nil(t, *dest.getOrig())
		})
		t.Run("Map", func(t *testing.T) {
			dest := emptiedMap(t)
			NewMap().MoveTo(dest)
			assert.Nil(t, *dest.getOrig())
		})
		t.Run("Value", func(t *testing.T) {
			dest := NewValueStr("set")
			NewValueEmpty().MoveTo(dest)
			assert.Nil(t, dest.getOrig().Value)
		})
		t.Run("Resource", func(t *testing.T) {
			dest := NewResource()
			dest.Attributes().PutStr("k", "v")
			NewResource().MoveTo(dest)
			assert.Nil(t, dest.getOrig().Attributes)
		})
	})
}

// Bytes reach the arena through their own copy helper, so a value holding them gets its own
// check that moving and copying keep an empty payload nil.
func TestValueBytesKeepEmptyNil(t *testing.T) {
	forEachGateState(t, func(t *testing.T) {
		t.Run("MoveTo", func(t *testing.T) {
			dest := NewValueBytes()
			dest.Bytes().FromRaw([]byte{1, 2, 3})
			src := NewValueBytes()
			src.MoveTo(dest)
			assert.Empty(t, dest.Bytes().AsRaw())
		})
		t.Run("CopyTo", func(t *testing.T) {
			dest := NewValueBytes()
			dest.Bytes().FromRaw([]byte{1, 2, 3})
			NewValueBytes().CopyTo(dest)
			assert.Empty(t, dest.Bytes().AsRaw())
		})
	})
}

// A move still has to carry the values across and clear the source.
func TestMoveToTransfersValues(t *testing.T) {
	forEachGateState(t, func(t *testing.T) {
		t.Run("UInt64Slice", func(t *testing.T) {
			dest := emptiedUInt64Slice(t)
			src := NewUInt64Slice()
			src.FromRaw([]uint64{4, 5})
			src.MoveTo(dest)
			assert.Equal(t, []uint64{4, 5}, dest.AsRaw())
			assert.Nil(t, *src.getOrig())
		})
		t.Run("Map", func(t *testing.T) {
			dest := emptiedMap(t)
			src := NewMap()
			src.PutStr("a", "b")
			src.MoveTo(dest)
			assert.Equal(t, map[string]any{"a": "b"}, dest.AsRaw())
			assert.Nil(t, *src.getOrig())
		})
	})
}

// MoveAndAppendTo hands over the source's slice whole when the destination is still nil, so an
// empty source leaves it nil rather than giving it an empty buffer.
func TestMoveAndAppendToKeepsNilDestinationNil(t *testing.T) {
	forEachGateState(t, func(t *testing.T) {
		t.Run("UInt64Slice", func(t *testing.T) {
			dest := NewUInt64Slice()
			require.Nil(t, *dest.getOrig())
			NewUInt64Slice().MoveAndAppendTo(dest)
			assert.Nil(t, *dest.getOrig())
		})
		t.Run("StringSlice", func(t *testing.T) {
			dest := NewStringSlice()
			require.Nil(t, *dest.getOrig())
			NewStringSlice().MoveAndAppendTo(dest)
			assert.Nil(t, *dest.getOrig())
		})
		t.Run("Slice", func(t *testing.T) {
			dest := NewSlice()
			require.Nil(t, *dest.getOrig())
			NewSlice().MoveAndAppendTo(dest)
			assert.Nil(t, *dest.getOrig())
		})
	})
}

// CopyTo reuses the destination's buffer instead of replacing it, so unlike a move it leaves a
// non-nil destination non-nil and only yields nil when the destination started out nil.
func TestCopyToKeepsDestinationBuffer(t *testing.T) {
	forEachGateState(t, func(t *testing.T) {
		t.Run("UInt64Slice", func(t *testing.T) {
			fresh := NewUInt64Slice()
			NewUInt64Slice().CopyTo(fresh)
			assert.Nil(t, *fresh.getOrig())

			used := emptiedUInt64Slice(t)
			NewUInt64Slice().CopyTo(used)
			assert.NotNil(t, *used.getOrig())
		})
		t.Run("Map", func(t *testing.T) {
			fresh := NewMap()
			NewMap().CopyTo(fresh)
			assert.Nil(t, *fresh.getOrig())

			used := emptiedMap(t)
			NewMap().CopyTo(used)
			assert.NotNil(t, *used.getOrig())
		})
		t.Run("Slice", func(t *testing.T) {
			fresh := NewSlice()
			NewSlice().CopyTo(fresh)
			assert.Nil(t, *fresh.getOrig())

			used := emptiedSlice(t)
			NewSlice().CopyTo(used)
			assert.NotNil(t, *used.getOrig())
		})
	})
}

// FromRaw copies into the existing buffer, so clearing a slice through it keeps the buffer.
func TestFromRawKeepsExistingBuffer(t *testing.T) {
	forEachGateState(t, func(t *testing.T) {
		t.Run("UInt64Slice", func(t *testing.T) {
			s := emptiedUInt64Slice(t)
			s.FromRaw(nil)
			assert.NotNil(t, *s.getOrig())
		})
		t.Run("StringSlice", func(t *testing.T) {
			s := emptiedStringSlice(t)
			s.FromRaw(nil)
			assert.NotNil(t, *s.getOrig())
		})
		t.Run("ByteSlice", func(t *testing.T) {
			s := emptiedByteSlice(t)
			s.FromRaw(nil)
			assert.NotNil(t, *s.getOrig())
		})
	})
}

// AsRaw copies into a fresh nil slice, so a slice holding an empty buffer reads back as nil.
func TestAsRawReturnsNilForEmptiedSlice(t *testing.T) {
	forEachGateState(t, func(t *testing.T) {
		t.Run("UInt64Slice", func(t *testing.T) {
			assert.Nil(t, emptiedUInt64Slice(t).AsRaw())
		})
		t.Run("StringSlice", func(t *testing.T) {
			assert.Nil(t, emptiedStringSlice(t).AsRaw())
		})
		t.Run("ByteSlice", func(t *testing.T) {
			assert.Nil(t, emptiedByteSlice(t).AsRaw())
		})
	})
}
