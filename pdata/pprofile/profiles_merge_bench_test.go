// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"strconv"
	"testing"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// These benchmarks quantify the two ways of establishing the index 0
// sentinel discussed in
// https://github.com/open-telemetry/opentelemetry-collector/issues/15897:
//
//   - approach 1: the interning helpers reserve index 0 of an empty table
//     (BenchmarkSetStringReserveEmptyBulk),
//   - approach 2: constructors are born with the sentinels
//     (BenchmarkMergeToPreSeededDest, BenchmarkSeedSentinels).
//
// BenchmarkMergeTo covers the approach implemented in this PR, seeding the
// destination tables inside MergeTo itself. The delta between BenchmarkMergeTo
// and BenchmarkMergeToPreSeededDest is the per-merge cost of the seeding,
// since the pre-seeded variant leaves ensureDictionarySentinels with nothing
// to do.

// benchSink keeps the interning results alive so the compiler cannot
// eliminate them.
var benchSink int32

func benchMergeSource(nStrings, nFunctions, nAttributes, nLocations, nStacks, nProfiles int) Profiles {
	ps := NewProfiles()
	d := ps.Dictionary()

	// Sentinel index 0 for every table, as the spec requires.
	d.StringTable().Append("")
	d.AttributeTable().AppendEmpty()
	d.FunctionTable().AppendEmpty()
	d.LinkTable().AppendEmpty()
	d.LocationTable().AppendEmpty()
	d.MappingTable().AppendEmpty()
	d.StackTable().AppendEmpty()

	for i := 0; i < nStrings; i++ {
		d.StringTable().Append("string-" + strconv.Itoa(i))
	}

	for i := 0; i < nFunctions; i++ {
		fn := d.FunctionTable().AppendEmpty()
		fn.SetNameStrindex(int32(1 + (i*3)%nStrings))
		fn.SetSystemNameStrindex(int32(1 + (i*3+1)%nStrings))
		fn.SetFilenameStrindex(int32(1 + (i*3+2)%nStrings))
		fn.SetStartLine(int64(i))
	}

	for i := 0; i < nAttributes; i++ {
		attr := d.AttributeTable().AppendEmpty()
		attr.SetKeyStrindex(int32(1 + i%nStrings))
		attr.SetUnitStrindex(int32(1 + (i*7)%nStrings))
		attr.Value().SetStr("attr-value-" + strconv.Itoa(i))
	}

	for i := 0; i < nLocations; i++ {
		loc := d.LocationTable().AppendEmpty()
		loc.SetAddress(0x400000 + uint64(i))
		line := loc.Lines().AppendEmpty()
		line.SetFunctionIndex(int32(1 + i%nFunctions))
		line.SetLine(int64(i))
		line2 := loc.Lines().AppendEmpty()
		line2.SetFunctionIndex(int32(1 + (i+1)%nFunctions))
		line2.SetLine(int64(i + 1))
	}

	for i := 0; i < nStacks; i++ {
		st := d.StackTable().AppendEmpty()
		st.LocationIndices().Append(int32(1+i%nLocations), int32(1+(i+1)%nLocations))
	}

	rp := ps.ResourceProfiles().AppendEmpty()
	sp := rp.ScopeProfiles().AppendEmpty()
	for i := 0; i < nProfiles; i++ {
		p := sp.Profiles().AppendEmpty()
		s := p.Samples().AppendEmpty()
		s.SetStackIndex(int32(1 + i%nStacks))
		s.AttributeIndices().Append(int32(1 + i%nAttributes))
		s.Values().Append(1, 2, 3)
	}

	return ps
}

// BenchmarkMergeTo measures the full merge with the per-merge sentinel
// seeding this PR adds, into a fresh destination dictionary.
func BenchmarkMergeTo(b *testing.B) {
	tmpl := benchMergeSource(2000, 500, 300, 500, 300, 100)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// MergeTo consumes the source and moves the resource profiles into
		// the destination, so both are rebuilt every iteration. Rebuilding is
		// excluded from the measurement.
		b.StopTimer()
		src := NewProfiles()
		tmpl.CopyTo(src)
		dst := NewProfiles()
		b.StartTimer()

		if err := src.MergeTo(dst); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMergeToPreSeededDest measures the same merge into a destination
// whose sentinels were seeded once, which is the merge-path cost of
// constructors born with the sentinels (approach 2).
func BenchmarkMergeToPreSeededDest(b *testing.B) {
	tmpl := benchMergeSource(2000, 500, 300, 500, 300, 100)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		src := NewProfiles()
		tmpl.CopyTo(src)
		dst := NewProfiles()
		// Approach 2 pays the seeding in the constructor, so it is done
		// outside the timed section. The merge itself only interns.
		ensureDictionarySentinels(dst.Dictionary())
		b.StartTimer()

		if err := src.MergeTo(dst); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSeedSentinels measures the one-time cost of establishing the
// sentinels on an empty dictionary, which approach 2 pays per construction
// and this PR pays per merge.
func BenchmarkSeedSentinels(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ensureDictionarySentinels(NewProfilesDictionary())
	}
}

func benchStringValues(n int) []string {
	values := make([]string, n)
	for i := 0; i < n; i++ {
		values[i] = "interned-string-" + strconv.Itoa(i)
	}
	return values
}

// setStringReserveEmpty is SetString with the approach 1 guard: an empty
// table gets its index 0 reserved for the zero value before the value is
// interned, so the first real entry lands at index 1.
func setStringReserveEmpty(table pcommon.StringSlice, val string) (int32, error) {
	if table.Len() == 0 {
		table.Append("")
	}
	return SetString(table, val)
}

// BenchmarkSetStringBulk interns a workload of unique strings into an empty
// table with the stock helper, which leaves index 0 free to be claimed by
// real data.
func BenchmarkSetStringBulk(b *testing.B) {
	values := benchStringValues(2000)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		table := pcommon.NewStringSlice()
		for _, v := range values {
			idx, err := SetString(table, v)
			if err != nil {
				b.Fatal(err)
			}
			benchSink += idx
		}
	}
}

// BenchmarkSetStringReserveEmptyBulk runs the same workload through the
// approach 1 variant, which reserves index 0 when the table is empty.
func BenchmarkSetStringReserveEmptyBulk(b *testing.B) {
	values := benchStringValues(2000)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		table := pcommon.NewStringSlice()
		for _, v := range values {
			idx, err := setStringReserveEmpty(table, v)
			if err != nil {
				b.Fatal(err)
			}
			benchSink += idx
		}
	}
}
