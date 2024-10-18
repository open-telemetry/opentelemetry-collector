// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelperprofiles

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMergeProfiles(t *testing.T) {
	pr1 := &profilesRequest{pd: testdata.GenerateProfiles(2)}
	pr2 := &profilesRequest{pd: testdata.GenerateProfiles(3)}
	res, err := pr1.Merge(context.Background(), pr2)
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%#v\n", res.(*profilesRequest).pd)
	assert.Equal(t, 5, res.(*profilesRequest).pd.SampleCount())
}

func TestMergeProfilesInvalidInput(t *testing.T) {
	pr1 := &tracesRequest{td: testdata.GenerateTraces(2)}
	pr2 := &profilesRequest{pd: testdata.GenerateProfiles(3)}
	_, err := pr2.Merge(context.Background(), pr1)
	assert.Error(t, err)
}

func TestMergeSplitProfiles(t *testing.T) {
	tests := []struct {
		name     string
		cfg      exporterbatcher.MaxSizeConfig
		pr1      exporterhelper.BatchRequest
		pr2      exporterhelper.BatchRequest
		expected []*profilesRequest
	}{
		{
			name:     "both_requests_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			pr1:      &profilesRequest{pd: pprofile.NewProfiles()},
			pr2:      &profilesRequest{pd: pprofile.NewProfiles()},
			expected: []*profilesRequest{{pd: pprofile.NewProfiles()}},
		},
		{
			name:     "first_request_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			pr1:      &profilesRequest{pd: pprofile.NewProfiles()},
			pr2:      &profilesRequest{pd: testdata.GenerateProfiles(5)},
			expected: []*profilesRequest{{pd: testdata.GenerateProfiles(5)}},
		},
		{
			name:     "first_empty_second_nil",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			pr1:      &profilesRequest{pd: pprofile.NewProfiles()},
			pr2:      nil,
			expected: []*profilesRequest{{pd: pprofile.NewProfiles()}},
		},
		{
			name: "merge_only",
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			pr1:  &profilesRequest{pd: testdata.GenerateProfiles(4)},
			pr2:  &profilesRequest{pd: testdata.GenerateProfiles(6)},
			expected: []*profilesRequest{{pd: func() pprofile.Profiles {
				profiles := testdata.GenerateProfiles(4)
				testdata.GenerateProfiles(6).ResourceProfiles().MoveAndAppendTo(profiles.ResourceProfiles())
				return profiles
			}()}},
		},
		{
			name: "split_only",
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 4},
			pr1:  &profilesRequest{pd: testdata.GenerateProfiles(10)},
			pr2:  nil,
			expected: []*profilesRequest{
				{pd: testdata.GenerateProfiles(4)},
				{pd: testdata.GenerateProfiles(4)},
				{pd: testdata.GenerateProfiles(2)},
			},
		},
		{
			name: "merge_and_split",
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			pr1:  &profilesRequest{pd: testdata.GenerateProfiles(8)},
			pr2:  &profilesRequest{pd: testdata.GenerateProfiles(20)},
			expected: []*profilesRequest{
				{pd: func() pprofile.Profiles {
					profiles := testdata.GenerateProfiles(8)
					testdata.GenerateProfiles(2).ResourceProfiles().MoveAndAppendTo(profiles.ResourceProfiles())
					return profiles
				}()},
				{pd: testdata.GenerateProfiles(10)},
				{pd: testdata.GenerateProfiles(8)},
			},
		},
		{
			name: "scope_profiles_split",
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 4},
			pr1: &profilesRequest{pd: func() pprofile.Profiles {
				return testdata.GenerateProfiles(6)
			}()},
			pr2: nil,
			expected: []*profilesRequest{
				{pd: testdata.GenerateProfiles(4)},
				{pd: func() pprofile.Profiles {
					return testdata.GenerateProfiles(2)
				}()},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.pr1.MergeSplit(context.Background(), tt.cfg, tt.pr2)
			require.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(res))
			for i, r := range res {
				assert.Equal(t, tt.expected[i], r.(*profilesRequest))
			}
		})

	}
}

func TestMergeSplitProfilesInvalidInput(t *testing.T) {
	r1 := &tracesRequest{td: testdata.GenerateTraces(2)}
	r2 := &profilesRequest{pd: testdata.GenerateProfiles(3)}
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

type tracesRequest struct {
	td     ptrace.Traces
	pusher consumer.ConsumeTracesFunc
}

func (req *tracesRequest) Export(ctx context.Context) error {
	return req.pusher(ctx, req.td)
}

func (req *tracesRequest) ItemsCount() int {
	return req.td.SpanCount()
}

func (req *tracesRequest) Merge(_ context.Context, _ exporterhelper.BatchRequest) (exporterhelper.BatchRequest, error) {
	return nil, nil
}

// MergeSplit splits and/or merges the profiles into multiple requests based on the MaxSizeConfig.
func (req *tracesRequest) MergeSplit(_ context.Context, _ exporterbatcher.MaxSizeConfig, _ exporterhelper.BatchRequest) (
	[]exporterhelper.BatchRequest, error) {
	return nil, nil
}
