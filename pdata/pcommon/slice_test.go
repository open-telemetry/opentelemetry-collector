// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pcommon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testutil"
)

func TestSlice_AsFromRaw(t *testing.T) {
	es := NewSlice()
	assert.Equal(t, 0, es.Len())

	raw := []any{int64(1), float64(2.3), true, "test", []any{"other"}, map[string]any{"key": "value", "int": int64(2)}}
	require.NoError(t, es.FromRaw(raw))
	assert.Equal(t, 6, es.Len())
	assert.Equal(t, raw, es.AsRaw())
}

func TestInvalidSlice(t *testing.T) {
	es := Slice{}

	assert.Panics(t, func() { es.Len() })
	assert.Panics(t, func() { es.At(0) })
	assert.Panics(t, func() { es.CopyTo(Slice{}) })
	assert.Panics(t, func() { es.EnsureCapacity(1) })
	assert.Panics(t, func() { es.AppendEmpty() })
	assert.Panics(t, func() { es.MoveAndAppendTo(Slice{}) })
	assert.Panics(t, func() { es.RemoveIf(func(Value) bool { return false }) })
	assert.Panics(t, func() { es.AsRaw() })
	assert.Panics(t, func() { _ = es.FromRaw([]any{3}) })
}

func TestSliceEqual(t *testing.T) {
	es := NewSlice()
	es2 := NewSlice()
	assert.True(t, es.Equal(es2))

	v := es.AppendEmpty()
	v.SetStr("test")
	assert.False(t, es.Equal(es2))

	v = es2.AppendEmpty()
	v.SetStr("test")
	assert.True(t, es.Equal(es2))
}

func BenchmarkSliceEqual(b *testing.B) {
	testutil.SkipMemoryBench(b)

	es := NewSlice()
	v := es.AppendEmpty()
	v.SetStr("test")
	cmp := NewSlice()
	v = cmp.AppendEmpty()
	v.SetStr("test")

	b.ReportAllocs()

	for b.Loop() {
		_ = es.Equal(cmp)
	}
}
