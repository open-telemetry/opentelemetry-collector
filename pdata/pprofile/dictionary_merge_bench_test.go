// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"fmt"
	"testing"
)

// fillMergeDict populates dict with n entries in each table. prefix makes the
// string content disjoint between two dictionaries so that merging one into the
// other appends (the worst case the exporter-queue batcher hits when merging
// profiles from distinct services). linesPerLoc / locsPerStack control the fan
// out that drives Location.Equal / Stack.Equal comparison cost.
func fillMergeDict(dict ProfilesDictionary, n, linesPerLoc, locsPerStack int, prefix string) {
	st := dict.StringTable()
	st.Append("")
	for i := 1; i < n; i++ {
		st.Append(fmt.Sprintf("%s-str-%d", prefix, i))
	}

	ft := dict.FunctionTable()
	ft.AppendEmpty()
	for i := 1; i < n; i++ {
		fn := ft.AppendEmpty()
		fn.SetNameStrindex(int32(i))
		fn.SetSystemNameStrindex(int32((i + 1) % n))
		fn.SetFilenameStrindex(int32((i + 2) % n))
		fn.SetStartLine(int64(i))
	}

	mt := dict.MappingTable()
	mt.AppendEmpty()
	for i := 1; i < n; i++ {
		m := mt.AppendEmpty()
		m.SetFilenameStrindex(int32(i))
		m.SetMemoryStart(uint64(i))
	}

	lt := dict.LocationTable()
	lt.AppendEmpty()
	for i := 1; i < n; i++ {
		loc := lt.AppendEmpty()
		loc.SetMappingIndex(int32(i % n))
		loc.SetAddress(uint64(i))
		for j := 0; j < linesPerLoc; j++ {
			ln := loc.Lines().AppendEmpty()
			ln.SetFunctionIndex(int32((i + j) % n))
			ln.SetLine(int64(i*100 + j))
		}
	}

	stkt := dict.StackTable()
	stkt.AppendEmpty()
	for i := 1; i < n; i++ {
		stk := stkt.AppendEmpty()
		for j := 0; j < locsPerStack; j++ {
			stk.LocationIndices().Append(int32((i + j) % n))
		}
	}
}

func BenchmarkDictionaryMergeBatch(b *testing.B) {
	const (
		linesPerLoc  = 2
		locsPerStack = 8
		batch        = 20
		perProfile   = 100
	)
	for b.Loop() {
		b.StopTimer()
		dst := NewProfiles()
		fillMergeDict(dst.Dictionary(), perProfile, linesPerLoc, locsPerStack, "dst")
		srcs := make([]Profiles, batch)
		for i := range srcs {
			s := NewProfiles()
			fillMergeDict(s.Dictionary(), perProfile, linesPerLoc, locsPerStack, fmt.Sprintf("s%d", i))
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

func BenchmarkDictionaryMerge(b *testing.B) {
	const (
		linesPerLoc  = 2
		locsPerStack = 8
	)
	for _, n := range []int{100, 250, 500, 1000} {
		b.Run(fmt.Sprintf("entries=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				src := NewProfiles()
				fillMergeDict(src.Dictionary(), n, linesPerLoc, locsPerStack, "src")
				dst := NewProfilesDictionary()
				fillMergeDict(dst, n, linesPerLoc, locsPerStack, "dst")
				b.StartTimer()

				if err := src.switchDictionary(src.Dictionary(), dst); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
