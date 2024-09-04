// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMergeTraces(t *testing.T) {
	tr1 := &tracesRequest{td: testdata.GenerateTraces(2)}
	tr2 := &tracesRequest{td: testdata.GenerateTraces(3)}
	res, err := mergeTraces(context.Background(), tr1, tr2)
	assert.Nil(t, err)
	assert.Equal(t, 5, res.(*tracesRequest).td.SpanCount())
}

func TestMergeTracesInvalidInput(t *testing.T) {
	tr1 := &logsRequest{ld: testdata.GenerateLogs(2)}
	tr2 := &tracesRequest{td: testdata.GenerateTraces(3)}
	_, err := mergeTraces(context.Background(), tr1, tr2)
	assert.Error(t, err)
}

func TestMergeSplitTraces(t *testing.T) {
	tests := []struct {
		name     string
		cfg      exporterbatcher.MaxSizeConfig
		tr1      Request
		tr2      Request
		expected []*tracesRequest
	}{
		{
			name:     "both_requests_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			tr1:      &tracesRequest{td: ptrace.NewTraces()},
			tr2:      &tracesRequest{td: ptrace.NewTraces()},
			expected: []*tracesRequest{{td: ptrace.NewTraces()}},
		},
		{
			name:     "both_requests_nil",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			tr1:      nil,
			tr2:      nil,
			expected: []*tracesRequest{},
		},
		{
			name:     "first_request_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			tr1:      &tracesRequest{td: ptrace.NewTraces()},
			tr2:      &tracesRequest{td: testdata.GenerateTraces(5)},
			expected: []*tracesRequest{{td: testdata.GenerateTraces(5)}},
		},
		{
			name:     "second_request_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			tr1:      &tracesRequest{td: testdata.GenerateTraces(5)},
			tr2:      &tracesRequest{td: ptrace.NewTraces()},
			expected: []*tracesRequest{{td: testdata.GenerateTraces(5)}},
		},
		{
			name:     "first_nil_second_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			tr1:      nil,
			tr2:      &tracesRequest{td: ptrace.NewTraces()},
			expected: []*tracesRequest{{td: ptrace.NewTraces()}},
		},
		{
			name: "merge_only",
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			tr1:  &tracesRequest{td: testdata.GenerateTraces(5)},
			tr2:  &tracesRequest{td: testdata.GenerateTraces(5)},
			expected: []*tracesRequest{{td: func() ptrace.Traces {
				td := testdata.GenerateTraces(5)
				testdata.GenerateTraces(5).ResourceSpans().MoveAndAppendTo(td.ResourceSpans())
				return td
			}()}},
		},
		{
			name: "split_only",
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 4},
			tr1:  nil,
			tr2:  &tracesRequest{td: testdata.GenerateTraces(10)},
			expected: []*tracesRequest{
				{td: testdata.GenerateTraces(4)},
				{td: testdata.GenerateTraces(4)},
				{td: testdata.GenerateTraces(2)},
			},
		},
		{
			name: "split_and_merge",
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			tr1:  &tracesRequest{td: testdata.GenerateTraces(4)},
			tr2:  &tracesRequest{td: testdata.GenerateTraces(20)},
			expected: []*tracesRequest{
				{td: func() ptrace.Traces {
					td := testdata.GenerateTraces(4)
					testdata.GenerateTraces(6).ResourceSpans().MoveAndAppendTo(td.ResourceSpans())
					return td
				}()},
				{td: testdata.GenerateTraces(10)},
				{td: testdata.GenerateTraces(4)},
			},
		},
		{
			name: "scope_spans_split",
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			tr1: &tracesRequest{td: func() ptrace.Traces {
				td := testdata.GenerateTraces(10)
				extraScopeTraces := testdata.GenerateTraces(5)
				extraScopeTraces.ResourceSpans().At(0).ScopeSpans().At(0).Scope().SetName("extra scope")
				extraScopeTraces.ResourceSpans().MoveAndAppendTo(td.ResourceSpans())
				return td
			}()},
			tr2: nil,
			expected: []*tracesRequest{
				{td: testdata.GenerateTraces(10)},
				{td: func() ptrace.Traces {
					td := testdata.GenerateTraces(5)
					td.ResourceSpans().At(0).ScopeSpans().At(0).Scope().SetName("extra scope")
					return td
				}()},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := mergeSplitTraces(context.Background(), tt.cfg, tt.tr1, tt.tr2)
			assert.Nil(t, err)
			assert.Equal(t, len(tt.expected), len(res))
			for i := range res {
				assert.Equal(t, tt.expected[i], res[i].(*tracesRequest))
			}
		})
	}
}

func TestMergeSplitTracesInvalidInput(t *testing.T) {
	r1 := &tracesRequest{td: testdata.GenerateTraces(2)}
	r2 := &metricsRequest{md: testdata.GenerateMetrics(3)}
	_, err := mergeSplitTraces(context.Background(), exporterbatcher.MaxSizeConfig{MaxSizeItems: 10}, r1, r2)
	assert.Error(t, err)
}

func TestExtractTraces(t *testing.T) {
	for i := 0; i < 10; i++ {
		td := testdata.GenerateTraces(10)
		extractedTraces := extractTraces(td, i)
		assert.Equal(t, i, extractedTraces.SpanCount())
		assert.Equal(t, 10-i, td.SpanCount())
	}
}
