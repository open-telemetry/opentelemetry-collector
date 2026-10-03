// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelper

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMergeProfiles(t *testing.T) {
	pr1 := newProfilesRequest(testdata.GenerateProfiles(2))
	pr2 := newProfilesRequest(testdata.GenerateProfiles(3))
	res, err := pr1.MergeSplit(context.Background(), 0, exporterhelper.RequestSizerTypeItems, pr2)
	require.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, 5, res[0].ItemsCount())
}

func TestMergeProfilesInvalidInput(t *testing.T) {
	pr2 := newProfilesRequest(testdata.GenerateProfiles(3))
	_, err := pr2.MergeSplit(context.Background(), 0, exporterhelper.RequestSizerTypeItems, &requesttest.FakeRequest{Items: 1})
	require.Error(t, err)
}

func TestMergeSplitProfiles(t *testing.T) {
	tests := []struct {
		name     string
		szt      exporterhelper.RequestSizerType
		maxSize  int
		pr1      Request
		pr2      Request
		expected []Request
	}{
		{
			name:     "both_requests_empty",
			szt:      exporterhelper.RequestSizerTypeItems,
			maxSize:  10,
			pr1:      newProfilesRequest(pprofile.NewProfiles()),
			pr2:      newProfilesRequest(pprofile.NewProfiles()),
			expected: []Request{newProfilesRequest(pprofile.NewProfiles())},
		},
		{
			name:    "first_request_empty",
			szt:     exporterhelper.RequestSizerTypeItems,
			maxSize: 10,
			pr1:     newProfilesRequest(testdata.GenerateProfiles(0)),
			pr2:     newProfilesRequest(testdata.GenerateProfiles(5)),
			expected: []Request{newProfilesRequest(func() pprofile.Profiles {
				profiles := testdata.GenerateProfiles(0)
				_ = testdata.GenerateProfiles(5).MergeTo(profiles)
				return profiles
			}())},
		},
		{
			name:     "first_empty_second_nil",
			szt:      exporterhelper.RequestSizerTypeItems,
			maxSize:  10,
			pr1:      newProfilesRequest(pprofile.NewProfiles()),
			pr2:      nil,
			expected: []Request{newProfilesRequest(pprofile.NewProfiles())},
		},
		{
			name:    "merge_only",
			szt:     exporterhelper.RequestSizerTypeItems,
			maxSize: 10,
			pr1:     newProfilesRequest(testdata.GenerateProfiles(4)),
			pr2:     newProfilesRequest(testdata.GenerateProfiles(6)),
			expected: []Request{newProfilesRequest(func() pprofile.Profiles {
				profiles := testdata.GenerateProfiles(4)
				testdata.GenerateProfiles(6).ResourceProfiles().MoveAndAppendTo(profiles.ResourceProfiles())
				return profiles
			}())},
		},
		{
			name:    "split_only",
			szt:     exporterhelper.RequestSizerTypeItems,
			maxSize: 4,
			pr1:     newProfilesRequest(testdata.GenerateProfiles(10)),
			pr2:     nil,
			expected: []Request{
				newProfilesRequest(testdata.GenerateProfiles(4)),
				newProfilesRequest(testdata.GenerateProfiles(4)),
				newProfilesRequest(testdata.GenerateProfiles(2)),
			},
		},
		{
			name:    "merge_and_split",
			szt:     exporterhelper.RequestSizerTypeItems,
			maxSize: 10,
			pr1:     newProfilesRequest(testdata.GenerateProfiles(8)),
			pr2:     newProfilesRequest(testdata.GenerateProfiles(20)),
			expected: []Request{
				newProfilesRequest(func() pprofile.Profiles {
					profiles := testdata.GenerateProfiles(8)
					testdata.GenerateProfiles(2).ResourceProfiles().MoveAndAppendTo(profiles.ResourceProfiles())
					return profiles
				}()),
				newProfilesRequest(testdata.GenerateProfiles(10)),
				newProfilesRequest(testdata.GenerateProfiles(8)),
			},
		},
		{
			name:    "scope_profiles_split",
			szt:     exporterhelper.RequestSizerTypeItems,
			maxSize: 4,
			pr1: newProfilesRequest(func() pprofile.Profiles {
				return testdata.GenerateProfiles(6)
			}()),
			pr2: nil,
			expected: []Request{
				newProfilesRequest(testdata.GenerateProfiles(4)),
				newProfilesRequest(func() pprofile.Profiles {
					return testdata.GenerateProfiles(2)
				}()),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.pr1.MergeSplit(context.Background(), tt.maxSize, tt.szt, tt.pr2)
			require.NoError(t, err)
			require.Len(t, res, len(tt.expected))
			for i, r := range res {
				assert.Equal(t, tt.expected[i].(*profilesRequest).pd, r.(*profilesRequest).pd)
			}
		})
	}
}

func TestMergeSplitProfilesBasedOnByteSize(t *testing.T) {
	tests := []struct {
		name     string
		szt      exporterhelper.RequestSizerType
		maxSize  int
		pr1      Request
		pr2      Request
		expected []Request
	}{
		{
			name:     "both_requests_empty",
			szt:      exporterhelper.RequestSizerTypeItems,
			maxSize:  10,
			pr1:      newProfilesRequest(pprofile.NewProfiles()),
			pr2:      newProfilesRequest(pprofile.NewProfiles()),
			expected: []Request{newProfilesRequest(pprofile.NewProfiles())},
		},
		{
			name:     "first_request_empty",
			szt:      exporterhelper.RequestSizerTypeItems,
			maxSize:  10,
			pr1:      newProfilesRequest(pprofile.NewProfiles()),
			pr2:      newProfilesRequest(testdata.GenerateProfiles(5)),
			expected: []Request{newProfilesRequest(testdata.GenerateProfiles(5))},
		},
		{
			name:     "first_empty_second_nil",
			szt:      exporterhelper.RequestSizerTypeItems,
			maxSize:  10,
			pr1:      newProfilesRequest(pprofile.NewProfiles()),
			pr2:      nil,
			expected: []Request{newProfilesRequest(pprofile.NewProfiles())},
		},
		{
			name:    "merge_only",
			szt:     exporterhelper.RequestSizerTypeItems,
			maxSize: 10,
			pr1:     newProfilesRequest(testdata.GenerateProfiles(4)),
			pr2:     newProfilesRequest(testdata.GenerateProfiles(6)),
			expected: []Request{newProfilesRequest(func() pprofile.Profiles {
				profiles := testdata.GenerateProfiles(4)
				testdata.GenerateProfiles(6).ResourceProfiles().MoveAndAppendTo(profiles.ResourceProfiles())
				return profiles
			}())},
		},
		{
			name:    "split_only",
			szt:     exporterhelper.RequestSizerTypeItems,
			maxSize: 4,
			pr1:     newProfilesRequest(testdata.GenerateProfiles(10)),
			pr2:     nil,
			expected: []Request{
				newProfilesRequest(testdata.GenerateProfiles(4)),
				newProfilesRequest(testdata.GenerateProfiles(4)),
				newProfilesRequest(testdata.GenerateProfiles(2)),
			},
		},
		{
			name:    "merge_and_split",
			szt:     exporterhelper.RequestSizerTypeItems,
			maxSize: 10,
			pr1:     newProfilesRequest(testdata.GenerateProfiles(8)),
			pr2:     newProfilesRequest(testdata.GenerateProfiles(20)),
			expected: []Request{
				newProfilesRequest(func() pprofile.Profiles {
					profiles := testdata.GenerateProfiles(8)
					testdata.GenerateProfiles(2).ResourceProfiles().MoveAndAppendTo(profiles.ResourceProfiles())
					return profiles
				}()),
				newProfilesRequest(testdata.GenerateProfiles(10)),
				newProfilesRequest(testdata.GenerateProfiles(8)),
			},
		},
		{
			name:    "scope_profiles_split",
			szt:     exporterhelper.RequestSizerTypeItems,
			maxSize: 4,
			pr1: newProfilesRequest(func() pprofile.Profiles {
				return testdata.GenerateProfiles(6)
			}()),
			pr2: nil,
			expected: []Request{
				newProfilesRequest(testdata.GenerateProfiles(4)),
				newProfilesRequest(func() pprofile.Profiles {
					return testdata.GenerateProfiles(2)
				}()),
			},
		},
		{
			name:     "both_requests_empty",
			szt:      exporterhelper.RequestSizerTypeBytes,
			maxSize:  profilesMarshaler.ProfilesSize(testdata.GenerateProfiles(10)),
			pr1:      newProfilesRequest(pprofile.NewProfiles()),
			pr2:      newProfilesRequest(pprofile.NewProfiles()),
			expected: []Request{newProfilesRequest(pprofile.NewProfiles())},
		},
		{
			name:     "first_request_empty",
			szt:      exporterhelper.RequestSizerTypeBytes,
			maxSize:  profilesMarshaler.ProfilesSize(testdata.GenerateProfiles(10)),
			pr1:      newProfilesRequest(pprofile.NewProfiles()),
			pr2:      newProfilesRequest(testdata.GenerateProfiles(5)),
			expected: []Request{newProfilesRequest(testdata.GenerateProfiles(5))},
		},
		{
			name:     "first_empty_second_nil",
			szt:      exporterhelper.RequestSizerTypeBytes,
			maxSize:  profilesMarshaler.ProfilesSize(testdata.GenerateProfiles(10)),
			pr1:      newProfilesRequest(pprofile.NewProfiles()),
			pr2:      nil,
			expected: []Request{newProfilesRequest(pprofile.NewProfiles())},
		},
		{
			name:    "merge_only",
			szt:     exporterhelper.RequestSizerTypeBytes,
			maxSize: profilesMarshaler.ProfilesSize(testdata.GenerateProfiles(13)),
			pr1:     newProfilesRequest(testdata.GenerateProfiles(4)),
			pr2:     newProfilesRequest(testdata.GenerateProfiles(6)),
			expected: []Request{newProfilesRequest(func() pprofile.Profiles {
				profiles := testdata.GenerateProfiles(4)
				testdata.GenerateProfiles(6).ResourceProfiles().MoveAndAppendTo(profiles.ResourceProfiles())
				return profiles
			}())},
		},
		{
			name:    "split_only",
			szt:     exporterhelper.RequestSizerTypeBytes,
			maxSize: profilesMarshaler.ProfilesSize(testdata.GenerateProfiles(4)),
			pr1:     newProfilesRequest(testdata.GenerateProfiles(0)),
			pr2:     newProfilesRequest(testdata.GenerateProfiles(10)),
			expected: []Request{
				newProfilesRequest(testdata.GenerateProfiles(4)),
				newProfilesRequest(testdata.GenerateProfiles(5)),
				newProfilesRequest(testdata.GenerateProfiles(1)),
			},
		},
		{
			name:    "merge_and_split",
			szt:     exporterhelper.RequestSizerTypeBytes,
			maxSize: profilesMarshaler.ProfilesSize(testdata.GenerateProfiles(10)),
			pr1:     newProfilesRequest(testdata.GenerateProfiles(8)),
			pr2:     newProfilesRequest(testdata.GenerateProfiles(20)),
			expected: []Request{
				newProfilesRequest(func() pprofile.Profiles {
					profiles := testdata.GenerateProfiles(7)
					testdata.GenerateProfiles(3).ResourceProfiles().MoveAndAppendTo(profiles.ResourceProfiles())
					return profiles
				}()),
				newProfilesRequest(testdata.GenerateProfiles(11)),
				newProfilesRequest(testdata.GenerateProfiles(7)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.pr1.MergeSplit(context.Background(), tt.maxSize, tt.szt, tt.pr2)
			require.NoError(t, err)
			require.Len(t, res, len(tt.expected))
			for i, r := range res {
				assert.Equal(t, tt.expected[i].(*profilesRequest).pd.SampleCount(), r.(*profilesRequest).pd.SampleCount(), i)
			}
		})
	}
}

func TestExtractProfiles(t *testing.T) {
	for i := range 10 {
		ld := testdata.GenerateProfiles(10)
		extractedProfiles, _ := extractProfiles(ld, i, &sizer.ProfilesSamplesCountSizer{})
		assert.Equal(t, i, extractedProfiles.SampleCount())
		assert.Equal(t, 10-i, ld.SampleCount())
	}
}

func TestMergeSplitManySmallProfiles(t *testing.T) {
	// All requests merge into a single batch.
	merged := []Request{newProfilesRequest(testdata.GenerateProfiles(1))}
	for range 1000 {
		pr2 := newProfilesRequest(testdata.GenerateProfiles(10))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 10000, exporterhelper.RequestSizerTypeItems, pr2)
		merged = append(merged[0:len(merged)-1], res...)
	}
	assert.Len(t, merged, 2)
}

func BenchmarkSplittingBasedOnByteSizeManySmallProfiles(b *testing.B) {
	// All requests merge into a single batch.
	b.ReportAllocs()
	for b.Loop() {
		merged := []Request{newProfilesRequest(testdata.GenerateProfiles(10))}
		for range 1000 {
			pr2 := newProfilesRequest(testdata.GenerateProfiles(10))
			res, _ := merged[len(merged)-1].MergeSplit(
				context.Background(),
				profilesMarshaler.ProfilesSize(testdata.GenerateProfiles(11000)),
				exporterhelper.RequestSizerTypeBytes,
				pr2,
			)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 2)
	}
}

func BenchmarkSplittingBasedOnByteSizeManyProfilesSlightlyAboveLimit(b *testing.B) {
	// Every incoming request results in a split.
	b.ReportAllocs()
	for b.Loop() {
		merged := []Request{newProfilesRequest(testdata.GenerateProfiles(0))}
		for range 10 {
			pr2 := newProfilesRequest(testdata.GenerateProfiles(10001))
			res, _ := merged[len(merged)-1].MergeSplit(
				context.Background(),
				profilesMarshaler.ProfilesSize(testdata.GenerateProfiles(10000)),
				exporterhelper.RequestSizerTypeBytes,
				pr2,
			)
			assert.Len(b, res, 2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 11)
	}
}

func BenchmarkSplittingBasedOnByteSizeHugeProfiles(b *testing.B) {
	// One request splits into many batches.
	b.ReportAllocs()
	for b.Loop() {
		merged := []Request{newProfilesRequest(testdata.GenerateProfiles(0))}
		pr2 := newProfilesRequest(testdata.GenerateProfiles(100000))
		res, _ := merged[len(merged)-1].MergeSplit(
			context.Background(),
			profilesMarshaler.ProfilesSize(testdata.GenerateProfiles(10010)),
			exporterhelper.RequestSizerTypeBytes,
			pr2,
		)
		merged = append(merged[0:len(merged)-1], res...)
		assert.Len(b, merged, 10)
	}
}

// profileLabels returns the original payload format of every profile across the given requests, in order.
func profileLabels(reqs []Request) []string {
	var out []string
	for _, r := range reqs {
		rps := r.(*profilesRequest).pd.ResourceProfiles()
		for i := 0; i < rps.Len(); i++ {
			sps := rps.At(i).ScopeProfiles()
			for j := 0; j < sps.Len(); j++ {
				ps := sps.At(j).Profiles()
				for k := 0; k < ps.Len(); k++ {
					out = append(out, ps.At(k).OriginalPayloadFormat())
				}
			}
		}
	}
	return out
}

// addProfile appends a profile holding one sample, labeled through its original payload format.
func addProfile(ps pprofile.ProfilesSlice, label string) {
	p := ps.AppendEmpty()
	p.SetOriginalPayloadFormat(label)
	p.Samples().AppendEmpty().Values().Append(1)
}

func newProfilesWithLabels(labels ...string) pprofile.Profiles {
	pd := pprofile.NewProfiles()
	ps := pd.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles()
	for _, l := range labels {
		addProfile(ps, l)
	}
	return pd
}

func TestMergeSplitProfilesDropsOnlyOversizedProfile(t *testing.T) {
	oversized := strings.Repeat("x", 1000)
	// More profiles than one batch holds, so the scope still has profiles after the pass
	// that drops the oversized one.
	many := make([]string, 30)
	for i := range many {
		many[i] = fmt.Sprintf("profile-%02d", i)
	}

	tests := []struct {
		name         string
		labels       []string
		wantSurvived []string
		wantDropped  int
	}{
		{
			name:         "oversized_first",
			labels:       []string{oversized, "a", "b", "c"},
			wantSurvived: []string{"a", "b", "c"},
			wantDropped:  1,
		},
		{
			name:         "oversized_in_middle",
			labels:       []string{"a", "b", oversized, "c", "d"},
			wantSurvived: []string{"a", "b", "c", "d"},
			wantDropped:  1,
		},
		{
			name:         "oversized_last",
			labels:       []string{"a", "b", "c", oversized},
			wantSurvived: []string{"a", "b", "c"},
			wantDropped:  1,
		},
		{
			name:         "multiple_oversized",
			labels:       []string{"a", oversized, "b", oversized, "c"},
			wantSurvived: []string{"a", "b", "c"},
			wantDropped:  2,
		},
		{
			name:         "oversized_followed_by_several_batches",
			labels:       append([]string{oversized}, many...),
			wantSurvived: many,
			wantDropped:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newProfilesRequest(newProfilesWithLabels(tt.labels...))
			res, err := req.MergeSplit(context.Background(), 100, exporterhelper.RequestSizerTypeBytes, nil)

			wantErr := fmt.Sprintf("single profile exceeds the max size limit, dropping items: %d", tt.wantDropped)
			require.ErrorContains(t, err, wantErr)
			assert.Equal(t, tt.wantSurvived, profileLabels(res),
				"profiles other than the oversized ones must survive")

			for _, r := range res {
				assert.LessOrEqual(t, r.BytesSize(), 100, "no returned batch may exceed max size")
				assert.Equal(t, profilesMarshaler.ProfilesSize(r.(*profilesRequest).pd), r.(*profilesRequest).size(&sizer.ProfilesBytesSizer{}),
					"the cached size must stay exact after dropping profiles")
			}
		})
	}
}

func TestMergeSplitProfilesAllProfilesOversized(t *testing.T) {
	oversized := strings.Repeat("x", 1000)
	req := newProfilesRequest(newProfilesWithLabels(oversized, oversized))

	res, err := req.MergeSplit(context.Background(), 100, exporterhelper.RequestSizerTypeBytes, nil)
	require.ErrorContains(t, err, "single profile exceeds the max size limit, dropping items: 2")
	assert.Empty(t, profileLabels(res), "nothing can be exported when every profile is oversized")
}

func TestMergeSplitProfilesItemlessOversizedRequest(t *testing.T) {
	// Resource attributes alone exceed max size and there is no profile at all, so
	// nothing can be exported and nothing is lost that needs reporting.
	pd := pprofile.NewProfiles()
	pd.ResourceProfiles().AppendEmpty().Resource().Attributes().PutStr("big", strings.Repeat("x", 500))
	req := newProfilesRequest(pd)
	require.Greater(t, req.BytesSize(), 100, "precondition: request must start oversized")

	res, err := req.MergeSplit(context.Background(), 100, exporterhelper.RequestSizerTypeBytes, nil)
	require.NoError(t, err, "nothing was lost, so nothing is due to be reported")
	assert.Empty(t, res, "an oversized request holding no profiles must not be returned")
}

func TestMergeSplitProfilesDropsOnlyOversizedAcrossResourcesAndScopes(t *testing.T) {
	// Only the oversized profile is dropped; profiles in the other scope and resource
	// must survive intact.
	oversized := strings.Repeat("x", 1000)
	pd := pprofile.NewProfiles()
	rp1 := pd.ResourceProfiles().AppendEmpty()
	addProfile(rp1.ScopeProfiles().AppendEmpty().Profiles(), oversized)
	addProfile(rp1.ScopeProfiles().AppendEmpty().Profiles(), "second_scope")
	addProfile(pd.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles(), "second_resource")
	require.Equal(t, 3, pd.SampleCount(), "precondition: three samples")

	res, err := newProfilesRequest(pd).MergeSplit(context.Background(), 100, exporterhelper.RequestSizerTypeBytes, nil)
	require.ErrorContains(t, err, "single profile exceeds the max size limit, dropping items: 1")
	assert.ElementsMatch(t, []string{"second_scope", "second_resource"}, profileLabels(res),
		"profiles in the other scope and resource must survive")
}

func TestMergeSplitProfilesEmptyOversizedResourceDoesNotStopSplitting(t *testing.T) {
	// A resource with big attributes and no profiles must not stop splitting of the
	// profiles behind it, and no sample may be reported as dropped.
	pd := pprofile.NewProfiles()
	empty := pd.ResourceProfiles().AppendEmpty()
	empty.Resource().Attributes().PutStr("big", strings.Repeat("B", 400))
	empty.ScopeProfiles().AppendEmpty()
	ps := pd.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles()
	for range 12 {
		addProfile(ps, strings.Repeat("v", 40))
	}
	req := newProfilesRequest(pd).(*profilesRequest)
	require.Equal(t, 12, req.pd.SampleCount(), "precondition: twelve profiles that each fit")
	require.Greater(t, req.BytesSize(), 100, "precondition: request starts oversized")

	res, err := req.MergeSplit(context.Background(), 100, exporterhelper.RequestSizerTypeBytes, nil)
	require.NoError(t, err, "no profile is oversized, so nothing should be reported")

	survived := 0
	for _, r := range res {
		pr := r.(*profilesRequest)
		survived += pr.pd.SampleCount()
		assert.LessOrEqual(t, profilesMarshaler.ProfilesSize(pr.pd), 100, "no batch may exceed max size")
		assert.Equal(t, profilesMarshaler.ProfilesSize(pr.pd), pr.size(&sizer.ProfilesBytesSizer{}),
			"the cached size must stay exact after removing a profile-less resource")
	}
	assert.Equal(t, 12, survived, "every sample must survive")
}

func TestMergeSplitProfilesStopsWhenNoProgressIsPossible(t *testing.T) {
	// The resource attributes alone fill max size exactly, so no scope or profile can be
	// added to any batch. Splitting must stop instead of looping, and must not return a
	// batch larger than max size.
	const maxSize = 300
	pd := pprofile.NewProfiles()
	rp := pd.ResourceProfiles().AppendEmpty()
	for pad := 0; ; pad++ {
		rp.Resource().Attributes().PutStr("pad", strings.Repeat("p", pad))
		if profilesMarshaler.ProfilesSize(pd) == maxSize {
			break
		}
	}
	ps := rp.ScopeProfiles().AppendEmpty().Profiles()
	addProfile(ps, "first")
	addProfile(ps, "second")

	res, err := newProfilesRequest(pd).MergeSplit(context.Background(), maxSize, exporterhelper.RequestSizerTypeBytes, nil)
	require.ErrorContains(t, err, "request size is greater than max size and cannot be split further, dropping items: 2")
	assert.Empty(t, res, "samples that cannot fit any batch must not be returned")
}
