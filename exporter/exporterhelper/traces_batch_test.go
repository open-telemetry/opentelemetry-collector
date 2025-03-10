// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMergeTraces(t *testing.T) {
	tr1 := newTracesRequest(testdata.GenerateTraces(2), nil)
	tr2 := newTracesRequest(testdata.GenerateTraces(3), nil)
	res, err := tr1.MergeSplit(context.Background(), exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems}, tr2)
	require.NoError(t, err)
	assert.Equal(t, 5, res[0].ItemsCount())
}

func TestMergeTracesInvalidInput(t *testing.T) {
	tr1 := newLogsRequest(testdata.GenerateLogs(2), nil)
	tr2 := newTracesRequest(testdata.GenerateTraces(3), nil)
	_, err := tr1.MergeSplit(context.Background(), exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems}, tr2)
	require.Error(t, err)
}

func TestMergeSplitTraces(t *testing.T) {
	tests := []struct {
		name     string
		cfg      exporterbatcher.SizeConfig
		tr1      Request
		tr2      Request
		expected []Request
	}{
		{
			name:     "both_requests_empty",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			tr1:      newTracesRequest(ptrace.NewTraces(), nil),
			tr2:      newTracesRequest(ptrace.NewTraces(), nil),
			expected: []Request{newTracesRequest(ptrace.NewTraces(), nil)},
		},
		{
			name:     "first_request_empty",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			tr1:      newTracesRequest(ptrace.NewTraces(), nil),
			tr2:      newTracesRequest(testdata.GenerateTraces(5), nil),
			expected: []Request{newTracesRequest(testdata.GenerateTraces(5), nil)},
		},
		{
			name:     "second_request_empty",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			tr1:      newTracesRequest(testdata.GenerateTraces(5), nil),
			tr2:      newTracesRequest(ptrace.NewTraces(), nil),
			expected: []Request{newTracesRequest(testdata.GenerateTraces(5), nil)},
		},
		{
			name:     "first_empty_second_nil",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			tr1:      newTracesRequest(ptrace.NewTraces(), nil),
			tr2:      nil,
			expected: []Request{newTracesRequest(ptrace.NewTraces(), nil)},
		},
		{
			name: "merge_only",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			tr1:  newTracesRequest(testdata.GenerateTraces(5), nil),
			tr2:  newTracesRequest(testdata.GenerateTraces(5), nil),
			expected: []Request{newTracesRequest(func() ptrace.Traces {
				td := testdata.GenerateTraces(5)
				testdata.GenerateTraces(5).ResourceSpans().MoveAndAppendTo(td.ResourceSpans())
				return td
			}(), nil)},
		},
		{
			name: "split_only",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 4},
			tr1:  newTracesRequest(ptrace.NewTraces(), nil),
			tr2:  newTracesRequest(testdata.GenerateTraces(10), nil),
			expected: []Request{
				newTracesRequest(testdata.GenerateTraces(4), nil),
				newTracesRequest(testdata.GenerateTraces(4), nil),
				newTracesRequest(testdata.GenerateTraces(2), nil),
			},
		},
		{
			name: "split_and_merge",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			tr1:  newTracesRequest(testdata.GenerateTraces(4), nil),
			tr2:  newTracesRequest(testdata.GenerateTraces(20), nil),
			expected: []Request{
				newTracesRequest(func() ptrace.Traces {
					td := testdata.GenerateTraces(4)
					testdata.GenerateTraces(6).ResourceSpans().MoveAndAppendTo(td.ResourceSpans())
					return td
				}(), nil),
				newTracesRequest(testdata.GenerateTraces(10), nil),
				newTracesRequest(testdata.GenerateTraces(4), nil),
			},
		},
		{
			name: "scope_spans_split",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			tr1: newTracesRequest(func() ptrace.Traces {
				td := testdata.GenerateTraces(10)
				extraScopeTraces := testdata.GenerateTraces(5)
				extraScopeTraces.ResourceSpans().At(0).ScopeSpans().At(0).Scope().SetName("extra scope")
				extraScopeTraces.ResourceSpans().MoveAndAppendTo(td.ResourceSpans())
				return td
			}(), nil),
			tr2: nil,
			expected: []Request{
				newTracesRequest(testdata.GenerateTraces(10), nil),
				newTracesRequest(func() ptrace.Traces {
					td := testdata.GenerateTraces(5)
					td.ResourceSpans().At(0).ScopeSpans().At(0).Scope().SetName("extra scope")
					return td
				}(), nil),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.tr1.MergeSplit(context.Background(), tt.cfg, tt.tr2)
			require.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(res))
			for i := range res {
				assert.Equal(t, tt.expected[i], res[i])
			}
		})
	}
}

func TestMergeSplitTracesInputNotModifiedIfErrorReturned(t *testing.T) {
	r1 := newTracesRequest(testdata.GenerateTraces(18), nil)
	r2 := newLogsRequest(testdata.GenerateLogs(3), nil)
	_, err := r1.MergeSplit(context.Background(), exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10}, r2)
	require.Error(t, err)
	assert.Equal(t, 18, r1.ItemsCount())
}

func TestMergeSplitTracesInvalidInput(t *testing.T) {
	r1 := newTracesRequest(testdata.GenerateTraces(2), nil)
	r2 := newMetricsRequest(testdata.GenerateMetrics(3), nil)
	_, err := r1.MergeSplit(context.Background(), exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10}, r2)
	require.Error(t, err)
}

func TestExtractTraces(t *testing.T) {
	for i := 0; i < 10; i++ {
		td := testdata.GenerateTraces(10)
		extractedTraces := extractTraces(td, i)
		assert.Equal(t, i, extractedTraces.SpanCount())
		assert.Equal(t, 10-i, td.SpanCount())
	}
}

func TestMergeSplitManySmallTraces(t *testing.T) {
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10000}
	merged := []Request{newTracesRequest(testdata.GenerateTraces(1), nil)}
	for j := 0; j < 1000; j++ {
		lr2 := newTracesRequest(testdata.GenerateTraces(10), nil)
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
		merged = append(merged[0:len(merged)-1], res...)
	}
	assert.Len(t, merged, 2)
}

func BenchmarkSplittingBasedOnItemCountManySmallTraces(b *testing.B) {
	// All requests merge into a single batch.
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10010}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newTracesRequest(testdata.GenerateTraces(10), nil)}
		for j := 0; j < 1000; j++ {
			lr2 := newTracesRequest(testdata.GenerateTraces(10), nil)
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 1)
	}
}

func BenchmarkSplittingBasedOnItemCountManyTracesSlightlyAboveLimit(b *testing.B) {
	// Every incoming request results in a split.
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10000}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newTracesRequest(testdata.GenerateTraces(0), nil)}
		for j := 0; j < 10; j++ {
			lr2 := newTracesRequest(testdata.GenerateTraces(10001), nil)
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 11)
	}
}

func BenchmarkSplittingBasedOnItemCountHugeTraces(b *testing.B) {
	// One request splits into many batches.
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10000}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newTracesRequest(testdata.GenerateTraces(0), nil)}
		lr2 := newTracesRequest(testdata.GenerateTraces(100000), nil)
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
		merged = append(merged[0:len(merged)-1], res...)
		assert.Len(b, merged, 10)
	}
}
