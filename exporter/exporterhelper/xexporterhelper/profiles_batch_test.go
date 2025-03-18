// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMergeProfiles(t *testing.T) {
	pr1 := newProfilesRequest(testdata.GenerateProfiles(2))
	pr2 := newProfilesRequest(testdata.GenerateProfiles(3))
	res, err := pr1.MergeSplit(context.Background(), exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems}, pr2)
	require.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, 5, res[0].ItemsCount())
}

func TestMergeProfilesInvalidInput(t *testing.T) {
	pr2 := newProfilesRequest(testdata.GenerateProfiles(3))
	_, err := pr2.MergeSplit(context.Background(), exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems}, &requesttest.FakeRequest{Items: 1})
	assert.Error(t, err)
}

func TestMergeSplitProfiles(t *testing.T) {
	tests := []struct {
		name     string
		cfg      exporterbatcher.SizeConfig
		pr1      exporterhelper.Request
		pr2      exporterhelper.Request
		expected []exporterhelper.Request
	}{
		{
			name:     "both_requests_empty",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			pr1:      newProfilesRequest(pprofile.NewProfiles()),
			pr2:      newProfilesRequest(pprofile.NewProfiles()),
			expected: []exporterhelper.Request{newProfilesRequest(pprofile.NewProfiles())},
		},
		{
			name:     "first_request_empty",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			pr1:      newProfilesRequest(pprofile.NewProfiles()),
			pr2:      newProfilesRequest(testdata.GenerateProfiles(5)),
			expected: []exporterhelper.Request{newProfilesRequest(testdata.GenerateProfiles(5))},
		},
		{
			name:     "first_empty_second_nil",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			pr1:      newProfilesRequest(pprofile.NewProfiles()),
			pr2:      nil,
			expected: []exporterhelper.Request{newProfilesRequest(pprofile.NewProfiles())},
		},
		{
			name: "merge_only",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			pr1:  newProfilesRequest(testdata.GenerateProfiles(4)),
			pr2:  newProfilesRequest(testdata.GenerateProfiles(6)),
			expected: []exporterhelper.Request{newProfilesRequest(func() pprofile.Profiles {
				profiles := testdata.GenerateProfiles(4)
				testdata.GenerateProfiles(6).ResourceProfiles().MoveAndAppendTo(profiles.ResourceProfiles())
				return profiles
			}())},
		},
		{
			name: "split_only",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 4},
			pr1:  newProfilesRequest(testdata.GenerateProfiles(10)),
			pr2:  nil,
			expected: []exporterhelper.Request{
				newProfilesRequest(testdata.GenerateProfiles(4)),
				newProfilesRequest(testdata.GenerateProfiles(4)),
				newProfilesRequest(testdata.GenerateProfiles(2)),
			},
		},
		{
			name: "merge_and_split",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			pr1:  newProfilesRequest(testdata.GenerateProfiles(8)),
			pr2:  newProfilesRequest(testdata.GenerateProfiles(20)),
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
			name: "scope_profiles_split",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 4},
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
			res, err := tt.pr1.MergeSplit(context.Background(), tt.cfg, tt.pr2)
			require.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(res))
			for i, r := range res {
				assert.Equal(t, tt.expected[i], r)
			}
		})
	}
}

func TestExtractProfiles(t *testing.T) {
	for i := 0; i < 10; i++ {
		ld := testdata.GenerateProfiles(10)
		extractedProfiles := extractProfiles(ld, i)
		assert.Equal(t, i, extractedProfiles.SampleCount())
		assert.Equal(t, 10-i, ld.SampleCount())
	}
}

func TestMergeSplitManySmallLogs(t *testing.T) {
	// All requests merge into a single batch.
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10000}
	merged := []exporterhelper.Request{newProfilesRequest(testdata.GenerateProfiles(1))}
	for j := 0; j < 1000; j++ {
		lr2 := newProfilesRequest(testdata.GenerateProfiles(10))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
		merged = append(merged[0:len(merged)-1], res...)
	}
	assert.Len(t, merged, 2)
}
