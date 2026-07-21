// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"fmt"
	"math"
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

// TestHashValue_DistinguishesValues is the guard against hashing only the
// value's type: every distinct value below must land in its own bucket. With a
// type-only hash all four strings (and all three ints, …) would collide, and the
// bucket walk would degrade back into the linear Equal scan the index replaces.
func TestHashValue_DistinguishesValues(t *testing.T) {
	sliceOf := func(vals ...string) pcommon.Value {
		v := pcommon.NewValueSlice()
		for _, s := range vals {
			v.Slice().AppendEmpty().SetStr(s)
		}
		return v
	}
	mapOf := func(kvs ...string) pcommon.Value {
		v := pcommon.NewValueMap()
		for i := 0; i < len(kvs); i += 2 {
			v.Map().PutStr(kvs[i], kvs[i+1])
		}
		return v
	}
	bytesOf := func(b ...byte) pcommon.Value {
		v := pcommon.NewValueBytes()
		v.Bytes().Append(b...)
		return v
	}

	vals := map[string]pcommon.Value{
		"empty":       pcommon.NewValueEmpty(),
		"str-empty":   pcommon.NewValueStr(""),
		"str-a":       pcommon.NewValueStr("a"),
		"str-b":       pcommon.NewValueStr("b"),
		"str-ab":      pcommon.NewValueStr("ab"),
		"int-0":       pcommon.NewValueInt(0),
		"int-1":       pcommon.NewValueInt(1),
		"int-neg":     pcommon.NewValueInt(-1),
		"double-0.5":  pcommon.NewValueDouble(0.5),
		"double-1.5":  pcommon.NewValueDouble(1.5),
		"bool-true":   pcommon.NewValueBool(true),
		"bool-false":  pcommon.NewValueBool(false),
		"bytes-empty": bytesOf(),
		"bytes-01":    bytesOf(0, 1),
		"bytes-10":    bytesOf(1, 0),
		"slice-empty": sliceOf(),
		"slice-ab":    sliceOf("a", "b"),
		"slice-ba":    sliceOf("b", "a"),
		"map-empty":   mapOf(),
		"map-a1":      mapOf("a", "1"),
		"map-a2":      mapOf("a", "2"),
		"map-b1":      mapOf("b", "1"),
		"map-a1b2":    mapOf("a", "1", "b", "2"),
	}

	seen := make(map[uint64]string, len(vals))
	for name, v := range vals {
		h := hashValue(fnvOffset64, v)
		if prev, ok := seen[h]; ok {
			t.Errorf("hash collision between %q and %q", prev, name)
		}
		seen[h] = name
	}
}

// TestHashValue_AgreesWithEqual covers the pairs where the hash must NOT
// distinguish, because [pcommon.Value.Equal] does not: -0.0 compares equal to
// +0.0, and maps compare by key rather than by insertion order. Hashing them
// apart would append a duplicate entry where the linear scan deduped.
func TestHashValue_AgreesWithEqual(t *testing.T) {
	negZero := pcommon.NewValueDouble(math.Copysign(0, -1))
	posZero := pcommon.NewValueDouble(0)
	require.True(t, negZero.Equal(posZero), "precondition: Equal treats -0.0 as 0.0")
	assert.Equal(t, hashValue(fnvOffset64, posZero), hashValue(fnvOffset64, negZero))

	forward := pcommon.NewValueMap()
	forward.Map().PutStr("a", "1")
	forward.Map().PutInt("b", 2)
	forward.Map().PutBool("c", true)

	reverse := pcommon.NewValueMap()
	reverse.Map().PutBool("c", true)
	reverse.Map().PutInt("b", 2)
	reverse.Map().PutStr("a", "1")

	require.True(t, forward.Equal(reverse), "precondition: Map.Equal is order-independent")
	assert.Equal(t, hashValue(fnvOffset64, forward), hashValue(fnvOffset64, reverse))
}

// TestHashValue_DoesNotAllocate guards the merge path's allocation budget. The
// index is rebuilt on every MergeTo, so one allocation inside the hash is paid
// once per attribute per merge; the copying accessors (ByteSlice.AsRaw,
// Map.AsRaw) are the easy way to reintroduce that.
func TestHashValue_DoesNotAllocate(t *testing.T) {
	bytesVal := pcommon.NewValueBytes()
	bytesVal.Bytes().Append(1, 2, 3, 4)

	sliceVal := pcommon.NewValueSlice()
	sliceVal.Slice().AppendEmpty().SetStr("a")
	sliceVal.Slice().AppendEmpty().SetInt(2)

	mapVal := pcommon.NewValueMap()
	mapVal.Map().PutStr("a", "1")
	mapVal.Map().PutInt("b", 2)

	for name, v := range map[string]pcommon.Value{
		"str":    pcommon.NewValueStr("worker-1"),
		"int":    pcommon.NewValueInt(7),
		"double": pcommon.NewValueDouble(1.5),
		"bool":   pcommon.NewValueBool(true),
		"bytes":  bytesVal,
		"slice":  sliceVal,
		"map":    mapVal,
	} {
		t.Run(name, func(t *testing.T) {
			assert.Zero(t, testing.AllocsPerRun(100, func() {
				hashValue(fnvOffset64, v)
			}))
		})
	}
}

// TestMergeIndex_SetAttribute_SameKeyDistinctValues is the shape the type-only
// hash handled worst and the existing differential test missed: one hot key
// (think thread.name or process.pid) carrying many distinct values. All entries
// share key and unit, so only the value can spread them across buckets.
func TestMergeIndex_SetAttribute_SameKeyDistinctValues(t *testing.T) {
	const distinct = 200

	ref := NewKeyValueAndUnitSlice()
	got := NewKeyValueAndUnitSlice()
	mi := newMergeIndex(NewProfilesDictionary())

	// Two passes: the second must dedup every entry against the first.
	for pass := range 2 {
		for i := range distinct {
			attr := NewKeyValueAndUnit()
			attr.SetKeyStrindex(1)
			attr.SetUnitStrindex(2)
			attr.Value().SetStr(fmt.Sprintf("worker-%d", i))

			refIdx, err := SetAttribute(ref, attr)
			require.NoError(t, err)
			gotIdx, err := mi.setAttribute(got, attr)
			require.NoError(t, err)
			assert.Equal(t, refIdx, gotIdx, "pass %d, value %d", pass, i)
		}
	}

	assert.Equal(t, ref, got)
	assert.Equal(t, distinct, got.Len(), "second pass must dedup entirely")

	longest := 0
	for _, bucket := range mi.attributes {
		longest = max(longest, len(bucket))
	}
	assert.Less(t, longest, distinct/10,
		"values must spread across buckets; a type-only hash would put all %d in one", distinct)
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
	switch r.IntN(7) {
	case 0:
		a.Value().SetStr([]string{"x", "y", "z"}[r.IntN(3)])
	case 1:
		a.Value().SetInt(int64(r.IntN(4)))
	case 2:
		a.Value().SetBool(r.IntN(2) == 0)
	case 3:
		a.Value().SetDouble(float64(r.IntN(4)) / 2)
	case 4:
		a.Value().SetEmptyBytes().Append(byte(r.IntN(3)), byte(r.IntN(3)))
	case 5:
		s := a.Value().SetEmptySlice()
		for n := r.IntN(3); n > 0; n-- {
			s.AppendEmpty().SetInt(int64(r.IntN(3)))
		}
	default:
		m := a.Value().SetEmptyMap()
		for _, k := range []string{"k0", "k1", "k2"}[:r.IntN(4)] {
			m.PutInt(k, int64(r.IntN(3)))
		}
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

// TestMergeIndex_TableOverflowGuards drives the int32-overflow guard in every
// set* method. The threshold is temporarily lowered so the guard fires on the
// first novel entry instead of requiring billions of rows, and each method must
// return its own table-specific sentinel.
func TestMergeIndex_TableOverflowGuards(t *testing.T) {
	// Mutates the package-global maxDictTableLen, so this test must never call
	// t.Parallel() or run concurrently with another test that performs a merge.
	// t.Cleanup restores the default even if an assertion fails.
	orig := maxDictTableLen
	maxDictTableLen = 1
	t.Cleanup(func() { maxDictTableLen = orig })

	dic := newConformantProfiles().Dictionary()
	mi := newMergeIndex(dic)

	fn := NewFunction()
	fn.SetNameStrindex(1)
	ma := NewMapping()
	ma.SetFileOffset(1)
	loc := NewLocation()
	loc.SetAddress(1)
	st := NewStack()
	st.LocationIndices().Append(1)
	attr := NewKeyValueAndUnit()
	attr.SetKeyStrindex(1)
	li := NewLink()
	li.SetTraceID(pcommon.TraceID([16]byte{1}))

	_, err := mi.setString(dic.StringTable(), "overflow")
	require.ErrorIs(t, err, errTooManyStringTableEntries)
	_, err = mi.setFunction(dic.FunctionTable(), fn)
	require.ErrorIs(t, err, errTooManyFunctionTableEntries)
	_, err = mi.setMapping(dic.MappingTable(), ma)
	require.ErrorIs(t, err, errTooManyMappingTableEntries)
	_, err = mi.setLocation(dic.LocationTable(), loc)
	require.ErrorIs(t, err, errTooManyLocationTableEntries)
	_, err = mi.setStack(dic.StackTable(), st)
	require.ErrorIs(t, err, errTooManyStackTableEntries)
	_, err = mi.setAttribute(dic.AttributeTable(), attr)
	require.ErrorIs(t, err, errTooManyAttributeTableEntries)
	_, err = mi.setLink(dic.LinkTable(), li)
	require.ErrorIs(t, err, errTooManyLinkTableEntries)
}
