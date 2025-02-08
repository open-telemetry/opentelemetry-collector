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

func TestNewFloat64Slice(t *testing.T) {
	ms := NewFloat64Slice()
	assert.Equal(t, 0, ms.Len())
	ms.FromRaw([]float64{1, 2, 3})
	assert.Equal(t, 3, ms.Len())
	assert.Equal(t, []float64{1, 2, 3}, ms.AsRaw())
	ms.SetAt(1, float64(5))
	assert.Equal(t, []float64{1, 5, 3}, ms.AsRaw())
	ms.FromRaw([]float64{3})
	assert.Equal(t, 1, ms.Len())
	assert.InDelta(t, float64(3), ms.At(0), 0.01)

	cp := NewFloat64Slice()
	ms.CopyTo(cp)
	ms.SetAt(0, float64(2))
	assert.InDelta(t, float64(2), ms.At(0), 0.01)
	assert.InDelta(t, float64(3), cp.At(0), 0.01)
	ms.CopyTo(cp)
	assert.InDelta(t, float64(2), cp.At(0), 0.01)

	mv := NewFloat64Slice()
	ms.MoveTo(mv)
	assert.Equal(t, 0, ms.Len())
	assert.Equal(t, 1, mv.Len())
	assert.InDelta(t, float64(2), mv.At(0), 0.01)
	ms.FromRaw([]float64{1, 2, 3})
	ms.MoveTo(mv)
	assert.Equal(t, 3, mv.Len())
	assert.InDelta(t, float64(1), mv.At(0), 0.01)
}

func TestFloat64SliceReadOnly(t *testing.T) {
	raw := []float64{1, 2, 3}
	state := internal.StateReadOnly
	ms := Float64Slice(internal.NewFloat64Slice(&raw, &state))

	assert.Equal(t, 3, ms.Len())
	assert.InDelta(t, float64(1), ms.At(0), 0.01)
	assert.Panics(t, func() { ms.Append(1) })
	assert.Panics(t, func() { ms.EnsureCapacity(2) })
	assert.Equal(t, raw, ms.AsRaw())
	assert.Panics(t, func() { ms.FromRaw(raw) })

	ms2 := NewFloat64Slice()
	ms.CopyTo(ms2)
	assert.Equal(t, ms.AsRaw(), ms2.AsRaw())
	assert.Panics(t, func() { ms2.CopyTo(ms) })

	assert.Panics(t, func() { ms.MoveTo(ms2) })
	assert.Panics(t, func() { ms2.MoveTo(ms) })
}

func TestFloat64SliceAppend(t *testing.T) {
	ms := NewFloat64Slice()
	ms.FromRaw([]float64{1, 2, 3})
	ms.Append(5, 5)
	assert.Equal(t, 5, ms.Len())
	assert.InDelta(t, float64(5), ms.At(4), 0.01)
}

func TestFloat64SliceEnsureCapacity(t *testing.T) {
	ms := NewFloat64Slice()
	ms.EnsureCapacity(4)
	assert.Equal(t, 4, cap(*ms.getOrig()))
	ms.EnsureCapacity(2)
	assert.Equal(t, 4, cap(*ms.getOrig()))
}

func TestFloat64SliceIncrementFrom(t *testing.T) {
	ms := NewFloat64Slice()
	ms.FromRaw([]float64{10, 9})

	ms2 := NewFloat64Slice()
	ms2.FromRaw([]float64{1, 10})

	assert.False(t, ms.IncrementFrom(ms2, 1))
	ms.EnsureCapacity(4)
	assert.True(t, ms.IncrementFrom(ms2, 1))
	assert.InDelta(t, float64(10), ms.At(0), 0.01)
	assert.InDelta(t, float64(10), ms.At(1), 0.01)
	assert.InDelta(t, float64(10), ms.At(2), 0.01)
}

func TestFloat64SliceCollapse(t *testing.T) {
	ms := NewFloat64Slice()
	ms.FromRaw([]float64{1, 1, 1, 1, 1, 1})

	ms.Collapse(4, 0)

	assert.Equal(t, 2, ms.Len())
	assert.InDelta(t, float64(4), ms.At(0), 0.01)
	assert.InDelta(t, float64(2), ms.At(1), 0.01)
}

func TestFloat64SliceCollapseOffset(t *testing.T) {
	ms := NewFloat64Slice()
	ms.FromRaw([]float64{1, 1, 1, 1, 1, 1})

	ms.Collapse(4, 3)

	assert.Equal(t, 3, ms.Len())
	assert.InDelta(t, float64(1), ms.At(0), 0.01)
	assert.InDelta(t, float64(4), ms.At(1), 0.01)
	assert.InDelta(t, float64(1), ms.At(2), 0.01)
}
