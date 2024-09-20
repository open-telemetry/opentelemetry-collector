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
	res, err := mergeProfiles(context.Background(), pr1, pr2)
	require.NoError(t, err)
	fmt.Fprintf(os.Stdout, "%#v\n", res.(*profilesRequest).pd)
	assert.Equal(t, 5, res.(*profilesRequest).pd.SampleCount())
}

func TestMergeProfilesInvalidInput(t *testing.T) {
	pr1 := &tracesRequest{td: testdata.GenerateTraces(2)}
	pr2 := &profilesRequest{pd: testdata.GenerateProfiles(3)}
	_, err := mergeProfiles(context.Background(), pr1, pr2)
	assert.Error(t, err)
}

func TestMergeSplitProfiles(t *testing.T) {
	tests := []struct {
		name     string
		cfg      exporterbatcher.MaxSizeConfig
		pr1      exporterhelper.Request
		pr2      exporterhelper.Request
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
			name:     "both_requests_nil",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			pr1:      nil,
			pr2:      nil,
			expected: []*profilesRequest{},
		},
		{
			name:     "first_request_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			pr1:      &profilesRequest{pd: pprofile.NewProfiles()},
			pr2:      &profilesRequest{pd: testdata.GenerateProfiles(5)},
			expected: []*profilesRequest{{pd: testdata.GenerateProfiles(5)}},
		},
		{
			name:     "first_requests_nil",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			pr1:      nil,
			pr2:      &profilesRequest{pd: testdata.GenerateProfiles(5)},
			expected: []*profilesRequest{{pd: testdata.GenerateProfiles(5)}},
		},
		{
			name:     "first_nil_second_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			pr1:      nil,
			pr2:      &profilesRequest{pd: pprofile.NewProfiles()},
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
			pr1:  nil,
			pr2:  &profilesRequest{pd: testdata.GenerateProfiles(10)},
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
			res, err := mergeSplitProfiles(context.Background(), tt.cfg, tt.pr1, tt.pr2)
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
	_, err := mergeSplitProfiles(context.Background(), exporterbatcher.MaxSizeConfig{}, r1, r2)
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
