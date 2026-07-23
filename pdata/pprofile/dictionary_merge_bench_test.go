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
		v = max(v, 1)
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
		attrsPerLoc  = 2
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
	sAttr := func(k int) int32 { return pick(k, attrShared, attrUnique, true) }
	uAttr := func(k int) int32 { return pick(k, attrShared, attrUnique, false) }

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
		for j := range attrsPerLoc {
			loc.AttributeIndices().Append(sAttr(i + j))
		}
		for j := range linesPerLoc {
			ln := loc.Lines().AppendEmpty()
			ln.SetFunctionIndex(sFn(i + j))
			ln.SetLine(int64(i*100 + j))
		}
	}
	for i := 1; i <= locUnique; i++ {
		loc := lt.AppendEmpty()
		loc.SetMappingIndex(sMap(i))
		loc.SetAddress(uint64(i))
		loc.AttributeIndices().Append(uAttr(i))
		for j := 1; j < attrsPerLoc; j++ {
			loc.AttributeIndices().Append(sAttr(i + j))
		}
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
		for j := range locsPerStack {
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
		if i%2 == 0 {
			a.Value().SetStr(fmt.Sprintf("sattr-%d", i))
		} else {
			a.Value().SetInt(int64(i))
		}
	}
	for i := 1; i <= attrUnique; i++ {
		a := at.AppendEmpty()
		a.SetKeyStrindex(uStr(i))
		if i%2 == 0 {
			a.Value().SetStr(fmt.Sprintf("p%d-uattr-%d", profileID, i))
		} else {
			a.Value().SetInt(int64(i))
		}
	}
}

// fillHotKeyProfile models the dictionary shape a type-only value hash degrades
// on: one hot attribute key (thread.name, k8s.pod.name, http.route) carrying
// many distinct values. Every entry shares key and unit, so the value is the
// only thing that can spread them across index buckets. Each attribute gets its
// own location because only referenced attributes reach the merge path.
func fillHotKeyProfile(dict ProfilesDictionary, attrs int, overlap float64, profileID int) {
	shared := int(float64(attrs) * overlap)

	st := dict.StringTable()
	st.Append("")
	st.Append("thread.name")
	st.Append("by")

	dict.MappingTable().AppendEmpty()
	at := dict.AttributeTable()
	at.AppendEmpty()
	lt := dict.LocationTable()
	lt.AppendEmpty()

	for i := 1; i <= attrs; i++ {
		a := at.AppendEmpty()
		a.SetKeyStrindex(1)
		a.SetUnitStrindex(2)
		if i <= shared {
			a.Value().SetStr(fmt.Sprintf("shared-worker-%d", i))
		} else {
			a.Value().SetStr(fmt.Sprintf("p%d-worker-%d", profileID, i))
		}

		loc := lt.AppendEmpty()
		loc.SetAddress(uint64(i))
		loc.AttributeIndices().Append(int32(i))
	}
}

// benchmarkFlush merges a batch of profiles into one initially-empty
// destination, mirroring a single exporter-queue batch flush. fill populates the
// dictionary of the profileID'th source.
func benchmarkFlush(b *testing.B, batch int, fill func(dict ProfilesDictionary, profileID int)) {
	b.ReportAllocs()
	for b.Loop() {
		b.StopTimer()
		dst := NewProfiles()
		srcs := make([]Profiles, batch)
		for i := range srcs {
			s := NewProfiles()
			fill(s.Dictionary(), i)
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

func benchmarkRealisticFlush(b *testing.B, sz profileSizes, overlap float64, batch int) {
	benchmarkFlush(b, batch, func(dict ProfilesDictionary, profileID int) {
		fillRealisticProfile(dict, sz, overlap, profileID)
	})
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
				benchmarkRealisticFlush(b, base, o, 20)
			})
		}
	})

	b.Run("batch", func(b *testing.B) {
		for _, k := range []int{5, 20, 50} {
			b.Run(fmt.Sprintf("batch=%d", k), func(b *testing.B) {
				benchmarkRealisticFlush(b, base, 0.9, k)
			})
		}
	})

	b.Run("size", func(b *testing.B) {
		for _, scale := range []float64{0.5, 1, 2} {
			b.Run(fmt.Sprintf("scale=%gx", scale), func(b *testing.B) {
				benchmarkRealisticFlush(b, realisticSizes(scale), 0.9, 20)
			})
		}
	})
}

// BenchmarkDictionaryMergeHotKey isolates the attribute table under a single hot
// key. The realistic sweep keeps the attribute table small and its keys varied,
// which hides how well the index spreads attributes by value; this sweep does
// not, so a regression to hashing the value's type rather than its content shows
// up here as quadratic growth in the attrs dimension.
func BenchmarkDictionaryMergeHotKey(b *testing.B) {
	for _, attrs := range []int{500, 2000, 8000} {
		b.Run(fmt.Sprintf("attrs=%d", attrs), func(b *testing.B) {
			benchmarkFlush(b, 10, func(dict ProfilesDictionary, profileID int) {
				fillHotKeyProfile(dict, attrs, 0.9, profileID)
			})
		})
	}
}
