// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/internal/metadata"
)

func TestAllocUsesArenaWhenGateEnabled(t *testing.T) {
	prev := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), true))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), prev))
	})

	st := NewState()
	require.NotNil(t, st.arena)
	a := Alloc[struct{ n int }](st)
	b := Alloc[struct{ n int }](st)
	assert.NotNil(t, a)
	assert.NotNil(t, b)
	assert.NotSame(t, a, b)

	st.DropArena()
	assert.Nil(t, st.arena)
}

func TestArenaResetReusesSlots(t *testing.T) {
	prev := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), true))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), prev))
	})

	st := NewState()
	first := Alloc[struct{ n int }](st)
	first.n = 7
	second := Alloc[struct{ n int }](st)
	require.NotSame(t, first, second)

	st.ResetArena()
	again := Alloc[struct{ n int }](st)
	assert.Same(t, first, again)
	assert.Equal(t, 0, again.n)

	slice1 := AllocSlice[int](st, 2, 8)
	slice1[0] = 9
	st.ResetArena()
	slice2 := AllocSlice[int](st, 2, 8)
	assert.Equal(t, []int{0, 0}, slice2)
	assert.Equal(t, 8, cap(slice2))
}

func TestPayloadChunksAndAppend(t *testing.T) {
	prev := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), true))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), prev))
	})

	st := NewState()
	s := CopyString(st, "hello")
	assert.Equal(t, "hello", s)
	require.NotEmpty(t, st.arena.payload)

	var nums []int
	nums = Append(st, nums, 1)
	nums = Append(st, nums, 2)
	assert.Equal(t, []int{1, 2}, nums)
	assert.GreaterOrEqual(t, cap(nums), 2)
}

func TestBorrowStringAndCopyOnWriteBytes(t *testing.T) {
	prev := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), true))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), prev))
	})

	st := NewState()
	buf := []byte("hello-world")
	st.RetainWire(buf)
	s := BorrowString(st, buf, 0, 5)
	assert.Equal(t, "hello", s)

	b := BorrowBytes(st, buf, 6, 11)
	assert.Equal(t, []byte("world"), b)
	st.CopyOnWriteBytes(&b)
	b[0] = 'W'
	assert.Equal(t, byte('w'), buf[6])
	assert.Equal(t, byte('W'), b[0])
}

func TestCopyStringAcrossArenas(t *testing.T) {
	prev := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), true))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), prev))
	})

	src := NewState()
	buf := []byte("abc")
	src.RetainWire(buf)
	s := BorrowString(src, buf, 0, 3)
	dest := NewState()
	cloned := CopyString(dest, s)
	assert.Equal(t, "abc", cloned)
	buf[0] = 'x'
	assert.Equal(t, "abc", cloned)
}
