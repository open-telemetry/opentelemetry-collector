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

func TestNewUInt64Slice(t *testing.T) {
	ms := NewUInt64Slice()
	assert.Equal(t, 0, ms.Len())
	ms.FromRaw([]uint64{1, 2, 3})
	assert.Equal(t, 3, ms.Len())
	assert.Equal(t, []uint64{1, 2, 3}, ms.AsRaw())
	ms.SetAt(1, uint64(5))
	assert.Equal(t, []uint64{1, 5, 3}, ms.AsRaw())
	ms.FromRaw([]uint64{3})
	assert.Equal(t, 1, ms.Len())
	assert.Equal(t, uint64(3), ms.At(0))

	cp := NewUInt64Slice()
	ms.CopyTo(cp)
	ms.SetAt(0, uint64(2))
	assert.Equal(t, uint64(2), ms.At(0))
	assert.Equal(t, uint64(3), cp.At(0))
	ms.CopyTo(cp)
	assert.Equal(t, uint64(2), cp.At(0))

	mv := NewUInt64Slice()
	ms.MoveTo(mv)
	assert.Equal(t, 0, ms.Len())
	assert.Equal(t, 1, mv.Len())
	assert.Equal(t, uint64(2), mv.At(0))
	ms.FromRaw([]uint64{1, 2, 3})
	ms.MoveTo(mv)
	assert.Equal(t, 3, mv.Len())
	assert.Equal(t, uint64(1), mv.At(0))
}

func TestUInt64SliceReadOnly(t *testing.T) {
	raw := []uint64{1, 2, 3}
	state := internal.StateReadOnly
	ms := UInt64Slice(internal.NewUInt64Slice(&raw, &state))

	assert.Equal(t, 3, ms.Len())
	assert.Equal(t, uint64(1), ms.At(0))
	assert.Panics(t, func() { ms.Append(1) })
	assert.Panics(t, func() { ms.EnsureCapacity(2) })
	assert.Equal(t, raw, ms.AsRaw())
	assert.Panics(t, func() { ms.FromRaw(raw) })

	ms2 := NewUInt64Slice()
	ms.CopyTo(ms2)
	assert.Equal(t, ms.AsRaw(), ms2.AsRaw())
	assert.Panics(t, func() { ms2.CopyTo(ms) })

	assert.Panics(t, func() { ms.MoveTo(ms2) })
	assert.Panics(t, func() { ms2.MoveTo(ms) })
}

func TestUInt64SliceAppend(t *testing.T) {
	ms := NewUInt64Slice()
	ms.FromRaw([]uint64{1, 2, 3})
	ms.Append(5, 5)
	assert.Equal(t, 5, ms.Len())
	assert.Equal(t, uint64(5), ms.At(4))
}

func TestUInt64SliceEnsureCapacity(t *testing.T) {
	ms := NewUInt64Slice()
	ms.EnsureCapacity(4)
	assert.Equal(t, 4, cap(*ms.getOrig()))
	ms.EnsureCapacity(2)
	assert.Equal(t, 4, cap(*ms.getOrig()))
}

func TestUInt64SliceIncrementFrom(t *testing.T) {
	ms := NewUInt64Slice()
	ms.FromRaw([]uint64{10, 9})

	ms2 := NewUInt64Slice()
	ms2.FromRaw([]uint64{1, 10})

	assert.False(t, ms.IncrementFrom(ms2, 1))
	ms.EnsureCapacity(4)
	assert.True(t, ms.IncrementFrom(ms2, 1))
	assert.Equal(t, uint64(10), ms.At(0))
	assert.Equal(t, uint64(10), ms.At(1))
	assert.Equal(t, uint64(10), ms.At(2))
}
