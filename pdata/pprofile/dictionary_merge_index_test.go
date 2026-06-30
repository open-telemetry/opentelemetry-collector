// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestMergeIndex_SetString_MatchesLinearScan(t *testing.T) {
	vals := []string{"", "a", "b", "a", "c", "b", "", "d", "a"}

	ref := pcommon.NewStringSlice()
	mi := newMergeIndex(NewProfilesDictionary())
	got := pcommon.NewStringSlice()

	for _, v := range vals {
		refIdx, err := SetString(ref, v)
		require.NoError(t, err)
		gotIdx, err := mi.setString(got, v)
		require.NoError(t, err)
		assert.Equal(t, refIdx, gotIdx, "index for %q", v)
	}
	assert.Equal(t, ref, got)
}

func TestMergeIndex_SetString_SeededFromExisting(t *testing.T) {
	dict := NewProfilesDictionary()
	dict.StringTable().Append("")
	dict.StringTable().Append("seed-1")
	dict.StringTable().Append("seed-2")

	mi := newMergeIndex(dict)

	idx, err := mi.setString(dict.StringTable(), "seed-2")
	require.NoError(t, err)
	assert.Equal(t, int32(2), idx, "must dedup against pre-existing entry")

	idx, err = mi.setString(dict.StringTable(), "new")
	require.NoError(t, err)
	assert.Equal(t, int32(3), idx)
	assert.Equal(t, 4, dict.StringTable().Len())
}

func TestMergeIndex_SetFunction_MatchesLinearScan(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 0))
	ref := NewFunctionSlice()
	got := NewFunctionSlice()
	mi := newMergeIndex(NewProfilesDictionary())

	for i := range 200 {
		fn := NewFunction()
		fn.SetNameStrindex(int32(r.IntN(10)))
		fn.SetSystemNameStrindex(int32(r.IntN(10)))
		fn.SetFilenameStrindex(int32(r.IntN(10)))
		fn.SetStartLine(int64(r.IntN(5)))

		refIdx, err := SetFunction(ref, fn)
		require.NoError(t, err)
		gotIdx, err := mi.setFunction(got, fn)
		require.NoError(t, err)
		assert.Equal(t, refIdx, gotIdx, "iter %d", i)
	}
	assert.Equal(t, ref, got)
}

func TestMergeIndex_SetMapping_MatchesLinearScan(t *testing.T) {
	r := rand.New(rand.NewPCG(2, 0))
	ref := NewMappingSlice()
	got := NewMappingSlice()
	mi := newMergeIndex(NewProfilesDictionary())

	for i := range 200 {
		ma := randMapping(r)

		refIdx, err := SetMapping(ref, ma)
		require.NoError(t, err)
		gotIdx, err := mi.setMapping(got, ma)
		require.NoError(t, err)
		assert.Equal(t, refIdx, gotIdx, "iter %d", i)
	}
	assert.Equal(t, ref, got)
}

func TestMergeIndex_SetLocation_MatchesLinearScan(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 0))
	ref := NewLocationSlice()
	got := NewLocationSlice()
	mi := newMergeIndex(NewProfilesDictionary())

	for i := range 200 {
		loc := randLocation(r)

		refIdx, err := SetLocation(ref, loc)
		require.NoError(t, err)
		gotIdx, err := mi.setLocation(got, loc)
		require.NoError(t, err)
		assert.Equal(t, refIdx, gotIdx, "iter %d", i)
	}
	assert.Equal(t, ref, got)
}

func TestMergeIndex_SetStack_MatchesLinearScan(t *testing.T) {
	r := rand.New(rand.NewPCG(4, 0))
	ref := NewStackSlice()
	got := NewStackSlice()
	mi := newMergeIndex(NewProfilesDictionary())

	for i := range 200 {
		st := randStack(r)

		refIdx, err := SetStack(ref, st)
		require.NoError(t, err)
		gotIdx, err := mi.setStack(got, st)
		require.NoError(t, err)
		assert.Equal(t, refIdx, gotIdx, "iter %d", i)
	}
	assert.Equal(t, ref, got)
}

func TestMergeIndex_SetAttribute_MatchesLinearScan(t *testing.T) {
	r := rand.New(rand.NewPCG(5, 0))
	ref := NewKeyValueAndUnitSlice()
	got := NewKeyValueAndUnitSlice()
	mi := newMergeIndex(NewProfilesDictionary())

	for i := range 200 {
		attr := randAttribute(r)

		refIdx, err := SetAttribute(ref, attr)
		require.NoError(t, err)
		gotIdx, err := mi.setAttribute(got, attr)
		require.NoError(t, err)
		assert.Equal(t, refIdx, gotIdx, "iter %d", i)
	}
	assert.Equal(t, ref, got)
}

func TestMergeIndex_SetLink_MatchesLinearScan(t *testing.T) {
	r := rand.New(rand.NewPCG(6, 0))
	ref := NewLinkSlice()
	got := NewLinkSlice()
	mi := newMergeIndex(NewProfilesDictionary())

	for i := range 200 {
		li := randLink(r)

		refIdx, err := SetLink(ref, li)
		require.NoError(t, err)
		gotIdx, err := mi.setLink(got, li)
		require.NoError(t, err)
		assert.Equal(t, refIdx, gotIdx, "iter %d", i)
	}
	assert.Equal(t, ref, got)
}

func TestMergeIndex_SetFunction_CollisionArbitratedByEqual(t *testing.T) {
	table := NewFunctionSlice()
	mi := newMergeIndex(NewProfilesDictionary())

	fnA := NewFunction()
	fnA.SetNameStrindex(1)
	fnA.SetStartLine(10)

	fnB := NewFunction()
	fnB.SetNameStrindex(2)
	fnB.SetStartLine(20)

	idxA, err := mi.setFunction(table, fnA)
	require.NoError(t, err)
	assert.Equal(t, int32(0), idxA)

	hB := hashFunction(fnB)
	mi.functions[hB] = append(mi.functions[hB], int32(0))

	idxB, err := mi.setFunction(table, fnB)
	require.NoError(t, err)
	assert.Equal(t, int32(1), idxB, "Equal must reject the colliding entry and append B")
	assert.Equal(t, 2, table.Len(), "table must have 2 entries after arbitration")
}

func TestMergeIndex_SetStack_CollisionArbitratedByEqual(t *testing.T) {
	table := NewStackSlice()
	mi := newMergeIndex(NewProfilesDictionary())

	stA := NewStack()
	stA.LocationIndices().Append(1, 2, 3)

	stB := NewStack()
	stB.LocationIndices().Append(4, 5, 6)

	idxA, err := mi.setStack(table, stA)
	require.NoError(t, err)
	assert.Equal(t, int32(0), idxA)

	hB := hashStack(stB)
	mi.stacks[hB] = append(mi.stacks[hB], int32(0))

	idxB, err := mi.setStack(table, stB)
	require.NoError(t, err)
	assert.Equal(t, int32(1), idxB, "Equal must reject the colliding entry and append B")
	assert.Equal(t, 2, table.Len(), "table must have 2 entries after arbitration")
}

func randMapping(r *rand.Rand) Mapping {
	ma := NewMapping()
	ma.SetMemoryStart(uint64(r.IntN(5)))
	ma.SetMemoryLimit(uint64(r.IntN(5)))
	ma.SetFileOffset(uint64(r.IntN(5)))
	ma.SetFilenameStrindex(int32(r.IntN(8)))
	for n := r.IntN(3); n > 0; n-- {
		ma.AttributeIndices().Append(int32(r.IntN(4)))
	}
	return ma
}

func randLocation(r *rand.Rand) Location {
	loc := NewLocation()
	loc.SetMappingIndex(int32(r.IntN(5)))
	loc.SetAddress(uint64(r.IntN(8)))
	for n := r.IntN(3); n > 0; n-- {
		loc.AttributeIndices().Append(int32(r.IntN(4)))
	}
	for n := r.IntN(3); n > 0; n-- {
		ln := loc.Lines().AppendEmpty()
		ln.SetFunctionIndex(int32(r.IntN(6)))
		ln.SetLine(int64(r.IntN(50)))
		ln.SetColumn(int64(r.IntN(10)))
	}
	return loc
}

func randStack(r *rand.Rand) Stack {
	st := NewStack()
	for n := r.IntN(6); n > 0; n-- {
		st.LocationIndices().Append(int32(r.IntN(6)))
	}
	return st
}

func randAttribute(r *rand.Rand) KeyValueAndUnit {
	a := NewKeyValueAndUnit()
	a.SetKeyStrindex(int32(r.IntN(8)))
	a.SetUnitStrindex(int32(r.IntN(4)))
	switch r.IntN(3) {
	case 0:
		a.Value().SetStr([]string{"x", "y", "z"}[r.IntN(3)])
	case 1:
		a.Value().SetInt(int64(r.IntN(4)))
	default:
		a.Value().SetBool(r.IntN(2) == 0)
	}
	return a
}

func randLink(r *rand.Rand) Link {
	li := NewLink()
	var tid [16]byte
	var sid [8]byte
	tid[0] = byte(r.IntN(3))
	sid[0] = byte(r.IntN(3))
	li.SetTraceID(pcommon.TraceID(tid))
	li.SetSpanID(pcommon.SpanID(sid))
	return li
}
