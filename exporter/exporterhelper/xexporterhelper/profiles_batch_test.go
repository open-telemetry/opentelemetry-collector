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
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMergeProfiles(t *testing.T) {
	pr1 := newProfilesRequest(testdata.GenerateProfiles(2), nil)
	pr2 := newProfilesRequest(testdata.GenerateProfiles(3), nil)
	res, err := pr1.MergeSplit(context.Background(), exporterbatcher.MaxSizeConfig{}, pr2)
	require.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, 5, res[0].ItemsCount())
}

func TestMergeProfilesInvalidInput(t *testing.T) {
	pr1 := &dummyRequest{}
	pr2 := newProfilesRequest(testdata.GenerateProfiles(3), nil)
	_, err := pr2.MergeSplit(context.Background(), exporterbatcher.MaxSizeConfig{}, pr1)
	assert.Error(t, err)
}

func TestMergeSplitProfiles(t *testing.T) {
	tests := []struct {
		name     string
		cfg      exporterbatcher.MaxSizeConfig
		pr1      exporterhelper.Request
		pr2      exporterhelper.Request
		expected []exporterhelper.Request
	}{
		{
			name:     "both_requests_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			pr1:      newProfilesRequest(pprofile.NewProfiles(), nil),
			pr2:      newProfilesRequest(pprofile.NewProfiles(), nil),
			expected: []exporterhelper.Request{newProfilesRequest(pprofile.NewProfiles(), nil)},
		},
		{
			name:     "first_request_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			pr1:      newProfilesRequest(pprofile.NewProfiles(), nil),
			pr2:      newProfilesRequest(testdata.GenerateProfiles(5), nil),
			expected: []exporterhelper.Request{newProfilesRequest(testdata.GenerateProfiles(5), nil)},
		},
		{
			name:     "first_empty_second_nil",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			pr1:      newProfilesRequest(pprofile.NewProfiles(), nil),
			pr2:      nil,
			expected: []exporterhelper.Request{newProfilesRequest(pprofile.NewProfiles(), nil)},
		},
		{
			name: "merge_only",
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			pr1:  newProfilesRequest(testdata.GenerateProfiles(4), nil),
			pr2:  newProfilesRequest(testdata.GenerateProfiles(6), nil),
			expected: []exporterhelper.Request{newProfilesRequest(func() pprofile.Profiles {
				profiles := testdata.GenerateProfiles(4)
				testdata.GenerateProfiles(6).ResourceProfiles().MoveAndAppendTo(profiles.ResourceProfiles())
				return profiles
			}(), nil)},
		},
		{
			name: "split_only",
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 4},
			pr1:  newProfilesRequest(testdata.GenerateProfiles(10), nil),
			pr2:  nil,
			expected: []exporterhelper.Request{
				newProfilesRequest(testdata.GenerateProfiles(4), nil),
				newProfilesRequest(testdata.GenerateProfiles(4), nil),
				newProfilesRequest(testdata.GenerateProfiles(2), nil),
			},
		},
		{
			name: "merge_and_split",
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			pr1:  newProfilesRequest(testdata.GenerateProfiles(8), nil),
			pr2:  newProfilesRequest(testdata.GenerateProfiles(20), nil),
			expected: []exporterhelper.Request{
				newProfilesRequest(func() pprofile.Profiles {
					profiles := testdata.GenerateProfiles(8)
					testdata.GenerateProfiles(2).ResourceProfiles().MoveAndAppendTo(profiles.ResourceProfiles())
					return profiles
				}(), nil),
				newProfilesRequest(testdata.GenerateProfiles(10), nil),
				newProfilesRequest(testdata.GenerateProfiles(8), nil),
			},
		},
		{
			name: "scope_profiles_split",
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 4},
			pr1: newProfilesRequest(func() pprofile.Profiles {
				return testdata.GenerateProfiles(6)
			}(), nil),
			pr2: nil,
			expected: []exporterhelper.Request{
				newProfilesRequest(testdata.GenerateProfiles(4), nil),
				newProfilesRequest(func() pprofile.Profiles {
					return testdata.GenerateProfiles(2)
				}(), nil),
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

func TestMergeSplitProfilesInvalidInput(t *testing.T) {
	r1 := &dummyRequest{}
	r2 := newProfilesRequest(testdata.GenerateProfiles(3), nil)
	_, err := r2.MergeSplit(context.Background(), exporterbatcher.MaxSizeConfig{}, r1)
	assert.Error(t, err)
}

func TestExtractProfiles(t *testing.T) {
	for i := 0; i < 10; i++ {
		ld := testdata.GenerateProfiles(10)
		extractedProfiles := extractProfiles(ld, i)
		assert.Equal(t, i, extractedProfiles.SampleCount())
		assert.Equal(t, 10-i, ld.SampleCount())
	}
}

// dummyRequest implements Request. It is for checking that merging two request types would fail
type dummyRequest struct{}

func (req *dummyRequest) Export(_ context.Context) error {
	return nil
}

func (req *dummyRequest) ItemsCount() int {
	return 1
}

func (req *dummyRequest) MergeSplit(_ context.Context, _ exporterbatcher.MaxSizeConfig, _ exporterhelper.Request) (
	[]exporterhelper.Request, error,
) {
	return nil, nil
}
