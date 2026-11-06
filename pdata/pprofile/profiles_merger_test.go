// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMergerMatchesSequentialMergeTo merges the same batch of profiles
// sequentially through MergeTo and through a single Merger and requires both
// paths to produce byte-identical destinations.
func TestMergerMatchesSequentialMergeTo(t *testing.T) {
	const batch = 8

	sizes := realisticSizes(0.1)
	build := func() []Profiles {
		srcs := make([]Profiles, batch)
		for i := range srcs {
			srcs[i] = NewProfiles()
			fillRealisticProfile(srcs[i].Dictionary(), sizes, 0.6, i)
			rp := srcs[i].ResourceProfiles().AppendEmpty()
			sp := rp.ScopeProfiles().AppendEmpty()
			p := sp.Profiles().AppendEmpty()
			p.Samples().AppendEmpty().SetStackIndex(1)
		}
		return srcs
	}

	dstMergeTo := NewProfiles()
	for _, src := range build() {
		require.NoError(t, src.MergeTo(dstMergeTo))
	}

	dstMerger := NewProfiles()
	m := NewMerger(dstMerger)
	for _, src := range build() {
		require.NoError(t, m.Merge(src))
	}

	marshaler := &ProtoMarshaler{}
	want, err := marshaler.MarshalProfiles(dstMergeTo)
	require.NoError(t, err)
	got, err := marshaler.MarshalProfiles(dstMerger)
	require.NoError(t, err)
	assert.Equal(t, want, got, "Merger output must be identical to sequential MergeTo")
}

// TestMergerDedupsAcrossMerges verifies that dictionary entries shared by
// every merged profile end up stored exactly once in the destination.
func TestMergerDedupsAcrossMerges(t *testing.T) {
	const batch = 5

	sizes := realisticSizes(0.1)
	dst := NewProfiles()
	m := NewMerger(dst)
	for i := range batch {
		src := NewProfiles()
		fillRealisticProfile(src.Dictionary(), sizes, 0.9, i)
		rp := src.ResourceProfiles().AppendEmpty()
		sp := rp.ScopeProfiles().AppendEmpty()
		p := sp.Profiles().AppendEmpty()
		p.Samples().AppendEmpty().SetStackIndex(1)
		require.NoError(t, m.Merge(src))
	}

	assert.Equal(t, batch, dst.ResourceProfiles().Len())
	assertNoDuplicateDictEntries(t, dst.Dictionary())
}

func TestMergerSelfMerge(t *testing.T) {
	profiles := NewProfiles()
	profiles.Dictionary().StringTable().Append("", "test")
	profiles.ResourceProfiles().AppendEmpty()

	m := NewMerger(profiles)
	require.NoError(t, m.Merge(profiles))

	assert.Equal(t, 2, profiles.Dictionary().StringTable().Len())
	assert.Equal(t, 1, profiles.ResourceProfiles().Len())
}

func TestMergerSourceMarkedReadOnly(t *testing.T) {
	src := newConformantProfiles()
	dst := newConformantProfiles()

	m := NewMerger(dst)
	require.NoError(t, m.Merge(src))
	assert.True(t, src.IsReadOnly(), "source must be marked read-only after merge")
}

// TestMergerErrorPropagation requires a failed switch to surface through
// Merge and to leave the destination's resource profiles untouched.
func TestMergerErrorPropagation(t *testing.T) {
	src := newConformantProfiles()
	src.Dictionary().StackTable().AppendEmpty().LocationIndices().Append(99)
	dst := newConformantProfiles()

	m := NewMerger(dst)
	err := m.Merge(src)
	require.ErrorContains(t, err, "invalid location index")
	assert.Equal(t, 0, dst.ResourceProfiles().Len(),
		"destination must be left untouched on error")
}

// TestMergerKeepsIndexAcrossGrownDestination merges into an already-merged
// destination to prove the index tracks dictionary growth: entries appended
// by earlier merges must still dedup against later sources.
func TestMergerKeepsIndexAcrossGrownDestination(t *testing.T) {
	build := func(profileID string) Profiles {
		p := newConformantProfiles()
		p.Dictionary().StringTable().Append("shared-fn")
		p.Dictionary().StringTable().Append(profileID)
		fn := p.Dictionary().FunctionTable().AppendEmpty()
		fn.SetNameStrindex(1)
		fn.SetFilenameStrindex(2)
		loc := p.Dictionary().LocationTable().AppendEmpty()
		ln := loc.Lines().AppendEmpty()
		ln.SetFunctionIndex(1)
		stk := p.Dictionary().StackTable().AppendEmpty()
		stk.LocationIndices().Append(1)
		rp := p.ResourceProfiles().AppendEmpty()
		sp := rp.ScopeProfiles().AppendEmpty()
		prof := sp.Profiles().AppendEmpty()
		s := prof.Samples().AppendEmpty()
		s.SetStackIndex(1)
		return p
	}

	dst := newConformantProfiles()
	m := NewMerger(dst)
	require.NoError(t, m.Merge(build("first")))
	require.NoError(t, m.Merge(build("second")))

	// "shared-fn" appears in both sources and must be stored once; the two
	// profileID strings append. Empty string + shared + 2 ids = 4 strings.
	assert.Equal(t, 4, dst.Dictionary().StringTable().Len())
	// The stack and its location are identical across sources modulo the
	// filename string, so only the functions differ.
	assertNoDuplicateDictEntries(t, dst.Dictionary())
	assert.Equal(t, 2, dst.ResourceProfiles().Len())
}

// BenchmarkMergerManySmallProfiles mirrors BenchmarkDictionaryMerge but keeps
// one Merger for the whole flush, which is the access pattern the exporter
// batcher needs. Run next to BenchmarkDictionaryMergeManySmallProfiles to see
// the quadratic term disappear.
func BenchmarkMergerManySmallProfiles(b *testing.B) {
	benchmarkManySmallProfiles(b, true)
}

// BenchmarkDictionaryMergeManySmallProfiles is the MergeTo baseline for
// BenchmarkMergerManySmallProfiles: merging N small profiles one by one into
// a single destination reindexes the destination on every call.
func BenchmarkDictionaryMergeManySmallProfiles(b *testing.B) {
	benchmarkManySmallProfiles(b, false)
}

func benchmarkManySmallProfiles(b *testing.B, useMerger bool) {
	base := realisticSizes(0.06) // ~150 strings / 120 functions / 120 locations / 240 stacks
	for _, batch := range []int{64, 256, 1024} {
		b.Run(fmt.Sprintf("batch=%d", batch), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				dst := NewProfiles()
				srcs := make([]Profiles, batch)
				for i := range srcs {
					s := NewProfiles()
					fillRealisticProfile(s.Dictionary(), base, 0.9, i)
					srcs[i] = s
				}
				b.StartTimer()

				if useMerger {
					m := NewMerger(dst)
					for i := range srcs {
						if err := m.Merge(srcs[i]); err != nil {
							b.Fatal(err)
						}
					}
				} else {
					for i := range srcs {
						if err := srcs[i].MergeTo(dst); err != nil {
							b.Fatal(err)
						}
					}
				}
			}
		})
	}
}
