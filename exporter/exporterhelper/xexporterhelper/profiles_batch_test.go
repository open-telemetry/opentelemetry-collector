// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelper

import (
	"context"
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
	assert.Error(t, err)
}

func TestMergeSplitProfiles(t *testing.T) {
	tests := []struct {
		name     string
		szt      exporterhelper.RequestSizerType
		maxSize  int
		pr1      exporterhelper.Request
		pr2      exporterhelper.Request
		expected []exporterhelper.Request
	}{
		{
			name:     "both_requests_empty",
			szt:      exporterhelper.RequestSizerTypeItems,
			maxSize:  10,
			pr1:      newProfilesRequest(pprofile.NewProfiles()),
			pr2:      newProfilesRequest(pprofile.NewProfiles()),
			expected: []exporterhelper.Request{newProfilesRequest(pprofile.NewProfiles())},
		},
		{
			name:     "first_request_empty",
			szt:      exporterhelper.RequestSizerTypeItems,
			maxSize:  10,
			pr1:      newProfilesRequest(pprofile.NewProfiles()),
			pr2:      newProfilesRequest(testdata.GenerateProfiles(5)),
			expected: []exporterhelper.Request{newProfilesRequest(testdata.GenerateProfiles(5))},
		},
		{
			name:     "first_empty_second_nil",
			szt:      exporterhelper.RequestSizerTypeItems,
			maxSize:  10,
			pr1:      newProfilesRequest(pprofile.NewProfiles()),
			pr2:      nil,
			expected: []exporterhelper.Request{newProfilesRequest(pprofile.NewProfiles())},
		},
		{
			name:    "merge_only",
			szt:     exporterhelper.RequestSizerTypeItems,
			maxSize: 10,
			pr1:     newProfilesRequest(testdata.GenerateProfiles(4)),
			pr2:     newProfilesRequest(testdata.GenerateProfiles(6)),
			expected: []exporterhelper.Request{newProfilesRequest(func() pprofile.Profiles {
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
			expected: []exporterhelper.Request{
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
			expected: []exporterhelper.Request{
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
			expected: []exporterhelper.Request{
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
			assert.Len(t, res, len(tt.expected))
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
		pr1      exporterhelper.Request
		pr2      exporterhelper.Request
		expected []exporterhelper.Request
	}{
		{
			name:     "both_requests_empty",
			szt:      exporterhelper.RequestSizerTypeItems,
			maxSize:  10,
			pr1:      newProfilesRequest(pprofile.NewProfiles()),
			pr2:      newProfilesRequest(pprofile.NewProfiles()),
			expected: []exporterhelper.Request{newProfilesRequest(pprofile.NewProfiles())},
		},
		{
			name:     "first_request_empty",
			szt:      exporterhelper.RequestSizerTypeItems,
			maxSize:  10,
			pr1:      newProfilesRequest(pprofile.NewProfiles()),
			pr2:      newProfilesRequest(testdata.GenerateProfiles(5)),
			expected: []exporterhelper.Request{newProfilesRequest(testdata.GenerateProfiles(5))},
		},
		{
			name:     "first_empty_second_nil",
			szt:      exporterhelper.RequestSizerTypeItems,
			maxSize:  10,
			pr1:      newProfilesRequest(pprofile.NewProfiles()),
			pr2:      nil,
			expected: []exporterhelper.Request{newProfilesRequest(pprofile.NewProfiles())},
		},
		{
			name:    "merge_only",
			szt:     exporterhelper.RequestSizerTypeItems,
			maxSize: 10,
			pr1:     newProfilesRequest(testdata.GenerateProfiles(4)),
			pr2:     newProfilesRequest(testdata.GenerateProfiles(6)),
			expected: []exporterhelper.Request{newProfilesRequest(func() pprofile.Profiles {
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
			expected: []exporterhelper.Request{
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
			expected: []exporterhelper.Request{
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
			expected: []exporterhelper.Request{
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
			expected: []exporterhelper.Request{newProfilesRequest(pprofile.NewProfiles())},
		},
		{
			name:     "first_request_empty",
			szt:      exporterhelper.RequestSizerTypeBytes,
			maxSize:  profilesMarshaler.ProfilesSize(testdata.GenerateProfiles(10)),
			pr1:      newProfilesRequest(pprofile.NewProfiles()),
			pr2:      newProfilesRequest(testdata.GenerateProfiles(5)),
			expected: []exporterhelper.Request{newProfilesRequest(testdata.GenerateProfiles(5))},
		},
		{
			name:     "first_empty_second_nil",
			szt:      exporterhelper.RequestSizerTypeBytes,
			maxSize:  profilesMarshaler.ProfilesSize(testdata.GenerateProfiles(10)),
			pr1:      newProfilesRequest(pprofile.NewProfiles()),
			pr2:      nil,
			expected: []exporterhelper.Request{newProfilesRequest(pprofile.NewProfiles())},
		},
		{
			name:    "merge_only",
			szt:     exporterhelper.RequestSizerTypeBytes,
			maxSize: profilesMarshaler.ProfilesSize(testdata.GenerateProfiles(11)),
			pr1:     newProfilesRequest(testdata.GenerateProfiles(4)),
			pr2:     newProfilesRequest(testdata.GenerateProfiles(6)),
			expected: []exporterhelper.Request{newProfilesRequest(func() pprofile.Profiles {
				profiles := testdata.GenerateProfiles(4)
				testdata.GenerateProfiles(6).ResourceProfiles().MoveAndAppendTo(profiles.ResourceProfiles())
				return profiles
			}())},
		},
		{
			name:    "split_only",
			szt:     exporterhelper.RequestSizerTypeBytes,
			maxSize: profilesMarshaler.ProfilesSize(testdata.GenerateProfiles(4)),
			pr1:     newProfilesRequest(pprofile.NewProfiles()),
			pr2:     newProfilesRequest(testdata.GenerateProfiles(10)),
			expected: []exporterhelper.Request{
				newProfilesRequest(testdata.GenerateProfiles(4)),
				newProfilesRequest(testdata.GenerateProfiles(4)),
				newProfilesRequest(testdata.GenerateProfiles(2)),
			},
		},
		{
			name:    "merge_and_split",
			szt:     exporterhelper.RequestSizerTypeBytes,
			maxSize: profilesMarshaler.ProfilesSize(testdata.GenerateProfiles(10)),
			pr1:     newProfilesRequest(testdata.GenerateProfiles(8)),
			pr2:     newProfilesRequest(testdata.GenerateProfiles(20)),
			expected: []exporterhelper.Request{
				newProfilesRequest(func() pprofile.Profiles {
					profiles := testdata.GenerateProfiles(7)
					testdata.GenerateProfiles(2).ResourceProfiles().MoveAndAppendTo(profiles.ResourceProfiles())
					return profiles
				}()),
				newProfilesRequest(testdata.GenerateProfiles(10)),
				newProfilesRequest(testdata.GenerateProfiles(9)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.pr1.MergeSplit(context.Background(), tt.maxSize, tt.szt, tt.pr2)
			require.NoError(t, err)
			assert.Len(t, res, len(tt.expected))
			for i, r := range res {
				assert.Equal(t, tt.expected[i].(*profilesRequest).pd.SampleCount(), r.(*profilesRequest).pd.SampleCount(), i)
			}
		})
	}
}

func TestExtractProfiles(t *testing.T) {
	for i := 0; i < 10; i++ {
		ld := testdata.GenerateProfiles(10)
		extractedProfiles, _ := extractProfiles(ld, i, &sizer.ProfilesCountSizer{})
		assert.Equal(t, i, extractedProfiles.SampleCount())
		assert.Equal(t, 10-i, ld.SampleCount())
	}
}

func TestMergeSplitManySmallLogs(t *testing.T) {
	// All requests merge into a single batch.
	merged := []exporterhelper.Request{newProfilesRequest(testdata.GenerateProfiles(1))}
	for j := 0; j < 1000; j++ {
		lr2 := newProfilesRequest(testdata.GenerateProfiles(10))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 10000, exporterhelper.RequestSizerTypeItems, lr2)
		merged = append(merged[0:len(merged)-1], res...)
	}
	assert.Len(t, merged, 2)
}

func BenchmarkSplittingBasedOnByteSizeManySmallProfiles(b *testing.B) {
	// All requests merge into a single batch.
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []exporterhelper.Request{newProfilesRequest(testdata.GenerateProfiles(10))}
		for j := 0; j < 1000; j++ {
			pr2 := newProfilesRequest(testdata.GenerateProfiles(10))
			res, _ := merged[len(merged)-1].MergeSplit(
				context.Background(),
				profilesMarshaler.ProfilesSize(testdata.GenerateProfiles(11000)),
				exporterhelper.RequestSizerTypeBytes,
				pr2,
			)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 1)
	}
}

func BenchmarkSplittingBasedOnByteSizeManyProfilesSlightlyAboveLimit(b *testing.B) {
	// Every incoming request results in a split.
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []exporterhelper.Request{newProfilesRequest(testdata.GenerateProfiles(0))}
		for j := 0; j < 10; j++ {
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
	for i := 0; i < b.N; i++ {
		merged := []exporterhelper.Request{newProfilesRequest(testdata.GenerateProfiles(0))}
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
