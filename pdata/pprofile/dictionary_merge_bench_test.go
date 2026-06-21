// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"fmt"
	"testing"
)

// profileSizes describes the per-profile dictionary shape. Real CPU/heap
// profiles are not uniform across tables: strings, functions and locations are
// correlated and large; stacks are the most numerous (one per unique call
// path); mappings and attributes are small.
type profileSizes struct {
	strings    int
	functions  int
	mappings   int
	locations  int
	stacks     int
	attributes int
}

// realisticSizes returns the per-profile table sizes of a representative
// production service profile, scaled by factor (0.5x lean / 1x baseline / 2x
// large monolith).
func realisticSizes(scale float64) profileSizes {
	s := func(n int) int {
		v := int(float64(n) * scale)
		if v < 1 {
			v = 1
		}
		return v
	}
	return profileSizes{
		strings:    s(2500),
		functions:  s(2000),
		mappings:   s(50),
		locations:  s(2000),
		stacks:     s(4000),
		attributes: s(30),
	}
}

// fillRealisticProfile populates dict to model one profile in an exporter-queue
// batch. A configurable fraction (overlap) of every table is drawn from a
// shared, profile-independent corpus (the stdlib/runtime/common-app symbols
// present in every profile of the deployment); the rest is a profile-specific
// unique tail. Merging a sequence of these into one destination therefore
// dedups the shared core and appends only the tails — the real flush pattern,
// of which the fully-disjoint case (overlap=0) is the worst-case extreme.
//
// The shared subgraph is referentially closed (shared entries reference only
// shared entries) so that, after index remapping into the destination, shared
// entries are byte-identical across profiles and dedup. Unique entries carry
// profileID in their content so they stay distinct and append.
func fillRealisticProfile(dict ProfilesDictionary, sz profileSizes, overlap float64, profileID int) {
	const (
		linesPerLoc  = 2
		locsPerStack = 16
	)

	sharedOf := func(n int) int {
		if n <= 1 {
			return 0
		}
		return int(float64(n-1) * overlap)
	}

	strShared, fnShared := sharedOf(sz.strings), sharedOf(sz.functions)
	mapShared, locShared := sharedOf(sz.mappings), sharedOf(sz.locations)
	stkShared, attrShared := sharedOf(sz.stacks), sharedOf(sz.attributes)

	strUnique := sz.strings - 1 - strShared
	fnUnique := sz.functions - 1 - fnShared
	mapUnique := sz.mappings - 1 - mapShared
	locUnique := sz.locations - 1 - locShared
	stkUnique := sz.stacks - 1 - stkShared
	attrUnique := sz.attributes - 1 - attrShared

	// pick returns an in-bounds index into a table, preferring the requested
	// category (shared/unique) and falling back to the other when one is empty.
	pick := func(k, shared, unique int, wantShared bool) int32 {
		if wantShared {
			if shared > 0 {
				return int32(1 + k%shared)
			}
			if unique > 0 {
				return int32(1 + k%unique)
			}
			return 0
		}
		if unique > 0 {
			return int32(1 + shared + k%unique)
		}
		if shared > 0 {
			return int32(1 + k%shared)
		}
		return 0
	}
	sStr := func(k int) int32 { return pick(k, strShared, strUnique, true) }
	uStr := func(k int) int32 { return pick(k, strShared, strUnique, false) }
	sFn := func(k int) int32 { return pick(k, fnShared, fnUnique, true) }
	uFn := func(k int) int32 { return pick(k, fnShared, fnUnique, false) }
	sMap := func(k int) int32 { return pick(k, mapShared, mapUnique, true) }
	sLoc := func(k int) int32 { return pick(k, locShared, locUnique, true) }
	uLoc := func(k int) int32 { return pick(k, locShared, locUnique, false) }

	st := dict.StringTable()
	st.Append("")
	for i := 1; i <= strShared; i++ {
		st.Append(fmt.Sprintf("sstr-%d", i))
	}
	for i := 1; i <= strUnique; i++ {
		st.Append(fmt.Sprintf("p%d-ustr-%d", profileID, i))
	}

	ft := dict.FunctionTable()
	ft.AppendEmpty()
	for i := 1; i <= fnShared; i++ {
		fn := ft.AppendEmpty()
		fn.SetNameStrindex(sStr(i))
		fn.SetSystemNameStrindex(sStr(i + 1))
		fn.SetFilenameStrindex(sStr(i + 2))
		fn.SetStartLine(int64(i))
	}
	for i := 1; i <= fnUnique; i++ {
		fn := ft.AppendEmpty()
		fn.SetNameStrindex(uStr(i))
		fn.SetSystemNameStrindex(uStr(i + 1))
		fn.SetFilenameStrindex(sStr(i))
		fn.SetStartLine(int64(i))
	}

	mt := dict.MappingTable()
	mt.AppendEmpty()
	for i := 1; i <= mapShared; i++ {
		m := mt.AppendEmpty()
		m.SetFilenameStrindex(sStr(i))
		m.SetMemoryStart(uint64(i))
	}
	for i := 1; i <= mapUnique; i++ {
		m := mt.AppendEmpty()
		m.SetFilenameStrindex(uStr(i))
		m.SetMemoryStart(uint64(i))
	}

	lt := dict.LocationTable()
	lt.AppendEmpty()
	for i := 1; i <= locShared; i++ {
		loc := lt.AppendEmpty()
		loc.SetMappingIndex(sMap(i))
		loc.SetAddress(uint64(i))
		for j := 0; j < linesPerLoc; j++ {
			ln := loc.Lines().AppendEmpty()
			ln.SetFunctionIndex(sFn(i + j))
			ln.SetLine(int64(i*100 + j))
		}
	}
	for i := 1; i <= locUnique; i++ {
		loc := lt.AppendEmpty()
		loc.SetMappingIndex(sMap(i))
		loc.SetAddress(uint64(i))
		ln := loc.Lines().AppendEmpty()
		ln.SetFunctionIndex(uFn(i))
		ln.SetLine(int64(i * 100))
		if linesPerLoc > 1 {
			ln2 := loc.Lines().AppendEmpty()
			ln2.SetFunctionIndex(sFn(i))
			ln2.SetLine(int64(i*100 + 1))
		}
	}

	stkt := dict.StackTable()
	stkt.AppendEmpty()
	for i := 1; i <= stkShared; i++ {
		stk := stkt.AppendEmpty()
		for j := 0; j < locsPerStack; j++ {
			stk.LocationIndices().Append(sLoc(i + j))
		}
	}
	for i := 1; i <= stkUnique; i++ {
		stk := stkt.AppendEmpty()
		stk.LocationIndices().Append(uLoc(i))
		for j := 1; j < locsPerStack; j++ {
			stk.LocationIndices().Append(sLoc(i + j))
		}
	}

	at := dict.AttributeTable()
	at.AppendEmpty()
	for i := 1; i <= attrShared; i++ {
		a := at.AppendEmpty()
		a.SetKeyStrindex(sStr(i))
		a.Value().SetInt(int64(i))
	}
	for i := 1; i <= attrUnique; i++ {
		a := at.AppendEmpty()
		a.SetKeyStrindex(uStr(i))
		a.Value().SetInt(int64(i))
	}
}

// benchmarkFlush merges a batch of realistic profiles into one initially-empty
// destination, mirroring a single exporter-queue batch flush.
func benchmarkFlush(b *testing.B, sz profileSizes, overlap float64, batch int) {
	b.ReportAllocs()
	for b.Loop() {
		b.StopTimer()
		dst := NewProfiles()
		srcs := make([]Profiles, batch)
		for i := range srcs {
			s := NewProfiles()
			fillRealisticProfile(s.Dictionary(), sz, overlap, i)
			srcs[i] = s
		}
		b.StartTimer()

		for i := range srcs {
			if err := srcs[i].MergeTo(dst); err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkDictionaryMerge sweeps the realistic exporter-queue flush across the
// three dimensions that drive merge cost: symbol overlap between profiles,
// batch size (number of merges per flush, which drives the per-merge reseed
// term), and per-profile dictionary size.
func BenchmarkDictionaryMerge(b *testing.B) {
	base := realisticSizes(1.0)

	b.Run("overlap", func(b *testing.B) {
		for _, o := range []float64{0, 0.5, 0.9, 0.99} {
			b.Run(fmt.Sprintf("overlap=%d%%", int(o*100)), func(b *testing.B) {
				benchmarkFlush(b, base, o, 20)
			})
		}
	})

	b.Run("batch", func(b *testing.B) {
		for _, k := range []int{5, 20, 50} {
			b.Run(fmt.Sprintf("batch=%d", k), func(b *testing.B) {
				benchmarkFlush(b, base, 0.9, k)
			})
		}
	})

	b.Run("size", func(b *testing.B) {
		for _, scale := range []float64{0.5, 1, 2} {
			b.Run(fmt.Sprintf("scale=%gx", scale), func(b *testing.B) {
				benchmarkFlush(b, realisticSizes(scale), 0.9, 20)
			})
		}
	})
}
