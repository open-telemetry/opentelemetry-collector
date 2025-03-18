// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package pcommon

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal"
)

func TestNewInt32Slice(t *testing.T) {
	ms := NewInt32Slice()
	assert.Equal(t, 0, ms.Len())
	ms.FromRaw([]int32{1, 2, 3})
	assert.Equal(t, 3, ms.Len())
	assert.Equal(t, []int32{1, 2, 3}, ms.AsRaw())
	ms.SetAt(1, int32(5))
	assert.Equal(t, []int32{1, 5, 3}, ms.AsRaw())
	ms.FromRaw([]int32{3})
	assert.Equal(t, 1, ms.Len())
	assert.Equal(t, int32(3), ms.At(0))

	cp := NewInt32Slice()
	ms.CopyTo(cp)
	ms.SetAt(0, int32(2))
	assert.Equal(t, int32(2), ms.At(0))
	assert.Equal(t, int32(3), cp.At(0))
	ms.CopyTo(cp)
	assert.Equal(t, int32(2), cp.At(0))

	mv := NewInt32Slice()
	ms.MoveTo(mv)
	assert.Equal(t, 0, ms.Len())
	assert.Equal(t, 1, mv.Len())
	assert.Equal(t, int32(2), mv.At(0))
	ms.FromRaw([]int32{1, 2, 3})
	ms.MoveTo(mv)
	assert.Equal(t, 3, mv.Len())
	assert.Equal(t, int32(1), mv.At(0))
}

func TestInt32SliceReadOnly(t *testing.T) {
	raw := []int32{1, 2, 3}
	state := internal.StateReadOnly
	ms := Int32Slice(internal.NewInt32Slice(&raw, &state))

	assert.Equal(t, 3, ms.Len())
	assert.Equal(t, int32(1), ms.At(0))
	assert.Panics(t, func() { ms.Append(1) })
	assert.Panics(t, func() { ms.EnsureCapacity(2) })
	assert.Equal(t, raw, ms.AsRaw())
	assert.Panics(t, func() { ms.FromRaw(raw) })

	ms2 := NewInt32Slice()
	ms.CopyTo(ms2)
	assert.Equal(t, ms.AsRaw(), ms2.AsRaw())
	assert.Panics(t, func() { ms2.CopyTo(ms) })

	assert.Panics(t, func() { ms.MoveTo(ms2) })
	assert.Panics(t, func() { ms2.MoveTo(ms) })
}

func TestInt32SliceAppend(t *testing.T) {
	ms := NewInt32Slice()
	ms.FromRaw([]int32{1, 2, 3})
	ms.Append(5, 5)
	assert.Equal(t, 5, ms.Len())
	assert.Equal(t, int32(5), ms.At(4))
}

func TestInt32SliceEnsureCapacity(t *testing.T) {
	ms := NewInt32Slice()
	ms.EnsureCapacity(4)
	assert.Equal(t, 4, cap(*ms.getOrig()))
	ms.EnsureCapacity(2)
	assert.Equal(t, 4, cap(*ms.getOrig()))
}

func TestInt32SliceAll(t *testing.T) {
	ms := NewInt32Slice()
	ms.FromRaw([]int32{1, 2, 3})
	assert.NotEmpty(t, ms.Len())

	var c int
	for i, v := range ms.All() {
		assert.Equal(t, ms.At(i), v, "element should match")
		c++
	}
	assert.Equal(t, ms.Len(), c, "All elements should have been visited")
}

func TestInt32SliceEqual(t *testing.T) {
	ms := NewInt32Slice()
	ms2 := NewInt32Slice()
	assert.True(t, ms.Equal(ms2))

	ms.Append(1, 2, 3)
	assert.False(t, ms.Equal(ms2))

	ms2.Append(1, 2, 3)
	assert.True(t, ms.Equal(ms2))
}

func BenchmarkInt32SliceEqual(b *testing.B) {
	ms := NewInt32Slice()
	ms.Append(1, 2, 3)
	cmp := NewInt32Slice()
	cmp.Append(1, 2, 3)

	b.ResetTimer()
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		_ = ms.Equal(cmp)
	}
}
