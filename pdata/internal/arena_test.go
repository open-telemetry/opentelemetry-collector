// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"runtime/debug"
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
	require.NotEmpty(t, st.arenas)
	a := Alloc[struct{ n int }](st)
	b := Alloc[struct{ n int }](st)
	assert.NotNil(t, a)
	assert.NotNil(t, b)
	assert.NotSame(t, a, b)

	st.DropArena()
	assert.Empty(t, st.arenas)
}

func TestDropArenaPoolsSlabs(t *testing.T) {
	prev := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), true))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), prev))
	})

	// sync.Pool never promises that Get returns what Put stored. A GC empties it, and a slab
	// parked in one P's private slot cannot be stolen by the P this goroutine later runs on.
	// Keeping the collector off rules out the first; retrying rules out the second.
	prevGC := debug.SetGCPercent(-1)
	t.Cleanup(func() { debug.SetGCPercent(prevGC) })

	type poolSlot struct{ n int }
	var pooled bool
	for range 10 {
		st := NewState()
		first := Alloc[poolSlot](st)
		first.n = 11
		require.NotNil(t, CopyString(st, "payload"))
		raw := &st.arenas[0].buf[0]
		st.RetainWire([]byte("wire"))

		st.DropArena()
		require.Empty(t, st.arenas)
		require.Nil(t, st.wire)

		next := NewState()
		require.Len(t, next.arenas, 1)
		require.Nil(t, next.wire)
		again := Alloc[poolSlot](next)
		// Only the round trips the pool actually served say anything about Release.
		if &next.arenas[0].buf[0] == raw {
			pooled = true
			assert.Same(t, first, again)
			assert.Equal(t, 0, again.n)
		}
		next.DropArena()
		if pooled {
			break
		}
	}
	assert.True(t, pooled, "DropArena should return the slab to arenaPool for the next request")
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
	require.Len(t, st.arenas, 1)
	require.Len(t, st.arenas[0].buf, chunkSize)

	var nums []int
	nums = Append(st, nums, 1)
	nums = Append(st, nums, 2)
	assert.Equal(t, []int{1, 2}, nums)
	assert.GreaterOrEqual(t, cap(nums), 2)

	// Sixty bytes left at two bytes per element predicts ten more than doubling would give,
	// so the appends that follow reuse that capacity instead of reallocating.
	var est []int
	est = AppendEstimated(st, est, 1, 60, 2)
	first := cap(est)
	assert.GreaterOrEqual(t, first, 60/(2*appendEstimateShare))
	for i := 2; i <= 10; i++ {
		est = AppendEstimated(st, est, i, 60-2*i, 2)
	}
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, est)
	assert.Equal(t, first, cap(est))

	// The estimate is capped, so one element early in a huge message cannot reserve the rest of it.
	var capped []byte
	capped = AppendEstimated(st, capped, 1, 1<<30, 1)
	assert.LessOrEqual(t, cap(capped), appendEstimateBudget+len(capped))

	// A useless estimate still grows by doubling.
	var grown []int
	for i := range 5 {
		grown = AppendEstimated(st, grown, i, 0, 0)
	}
	assert.Equal(t, []int{0, 1, 2, 3, 4}, grown)
	assert.GreaterOrEqual(t, cap(grown), 5)
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

func TestRequestTakesAnotherArenaWhenBufferIsFull(t *testing.T) {
	prev := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), true))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), prev))
	})

	st := NewState()
	_ = AllocSlice[byte](st, chunkSize, chunkSize)
	require.Len(t, st.arenas, 1)
	next := Alloc[byte](st)
	require.NotNil(t, next)
	require.Len(t, st.arenas, 2)
	assert.NotSame(t, &st.arenas[0].buf[0], &st.arenas[1].buf[0])
	st.DropArena()
}
