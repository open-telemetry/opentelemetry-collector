// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMergeMetrics(t *testing.T) {
	mr1 := newMetricsRequest(testdata.GenerateMetrics(2), nil)
	mr2 := newMetricsRequest(testdata.GenerateMetrics(3), nil)
	res, err := mr1.MergeSplit(context.Background(), exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems}, mr2)
	require.NoError(t, err)
	// Every metric has 2 data points.
	assert.Equal(t, 2*5, res[0].ItemsCount())
}

func TestMergeMetricsInvalidInput(t *testing.T) {
	mr1 := newTracesRequest(testdata.GenerateTraces(2), nil)
	mr2 := newMetricsRequest(testdata.GenerateMetrics(3), nil)
	_, err := mr1.MergeSplit(context.Background(), exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems}, mr2)
	require.Error(t, err)
}

func TestMergeSplitMetrics(t *testing.T) {
	tests := []struct {
		name     string
		cfg      exporterbatcher.SizeConfig
		mr1      Request
		mr2      Request
		expected []Request
	}{
		{
			name:     "both_requests_empty",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			mr1:      newMetricsRequest(pmetric.NewMetrics(), nil),
			mr2:      newMetricsRequest(pmetric.NewMetrics(), nil),
			expected: []Request{newMetricsRequest(pmetric.NewMetrics(), nil)},
		},
		{
			name:     "first_request_empty",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			mr1:      newMetricsRequest(pmetric.NewMetrics(), nil),
			mr2:      newMetricsRequest(testdata.GenerateMetrics(5), nil),
			expected: []Request{newMetricsRequest(testdata.GenerateMetrics(5), nil)},
		},
		{
			name:     "first_empty_second_nil",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			mr1:      newMetricsRequest(pmetric.NewMetrics(), nil),
			mr2:      nil,
			expected: []Request{newMetricsRequest(pmetric.NewMetrics(), nil)},
		},
		{
			name: "merge_only",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 60},
			mr1:  newMetricsRequest(testdata.GenerateMetrics(10), nil),
			mr2:  newMetricsRequest(testdata.GenerateMetrics(14), nil),
			expected: []Request{newMetricsRequest(func() pmetric.Metrics {
				metrics := testdata.GenerateMetrics(10)
				testdata.GenerateMetrics(14).ResourceMetrics().MoveAndAppendTo(metrics.ResourceMetrics())
				return metrics
			}(), nil)},
		},
		{
			name: "split_only",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 14},
			mr1:  newMetricsRequest(pmetric.NewMetrics(), nil),
			mr2:  newMetricsRequest(testdata.GenerateMetrics(15), nil), // 15 metrics, 30 data points
			expected: []Request{
				newMetricsRequest(testdata.GenerateMetrics(7), nil), // 7 metrics, 14 data points
				newMetricsRequest(testdata.GenerateMetrics(7), nil), // 7 metrics, 14 data points
				newMetricsRequest(testdata.GenerateMetrics(1), nil), // 1 metric, 2 data points
			},
		},
		{
			name: "split_and_merge",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 28},
			mr1:  newMetricsRequest(testdata.GenerateMetrics(7), nil),  // 7 metrics, 14 data points
			mr2:  newMetricsRequest(testdata.GenerateMetrics(25), nil), // 25 metrics, 50 data points
			expected: []Request{
				newMetricsRequest(func() pmetric.Metrics {
					metrics := testdata.GenerateMetrics(7)
					testdata.GenerateMetrics(7).ResourceMetrics().MoveAndAppendTo(metrics.ResourceMetrics())
					return metrics
				}(), nil),
				newMetricsRequest(testdata.GenerateMetrics(14), nil), // 14 metrics, 28 data points
				newMetricsRequest(testdata.GenerateMetrics(4), nil),  // 4 metrics, 8 data points
			},
		},
		{
			name: "scope_metrics_split",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 8},
			mr1: newMetricsRequest(func() pmetric.Metrics {
				md := testdata.GenerateMetrics(4)
				extraScopeMetrics := md.ResourceMetrics().At(0).ScopeMetrics().AppendEmpty()
				testdata.GenerateMetrics(4).ResourceMetrics().At(0).ScopeMetrics().At(0).MoveTo(extraScopeMetrics)
				extraScopeMetrics.Scope().SetName("extra scope")
				return md
			}(), nil),
			mr2: nil,
			expected: []Request{
				newMetricsRequest(testdata.GenerateMetrics(4), nil),
				newMetricsRequest(func() pmetric.Metrics {
					md := testdata.GenerateMetrics(4)
					md.ResourceMetrics().At(0).ScopeMetrics().At(0).Scope().SetName("extra scope")
					return md
				}(), nil),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.mr1.MergeSplit(context.Background(), tt.cfg, tt.mr2)
			require.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(res))
			for i := range res {
				assert.Equal(t, tt.expected[i], res[i])
			}
		})
	}
}

func TestMergeSplitMetricsInputNotModifiedIfErrorReturned(t *testing.T) {
	r1 := newMetricsRequest(testdata.GenerateMetrics(18), nil) // 18 metrics, 36 data points
	r2 := newLogsRequest(testdata.GenerateLogs(3), nil)
	_, err := r1.MergeSplit(context.Background(), exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10}, r2)
	require.Error(t, err)
	assert.Equal(t, 36, r1.ItemsCount())
}

func TestMergeSplitMetricsInvalidInput(t *testing.T) {
	r1 := newTracesRequest(testdata.GenerateTraces(2), nil)
	r2 := newMetricsRequest(testdata.GenerateMetrics(3), nil)
	_, err := r1.MergeSplit(context.Background(), exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10}, r2)
	require.Error(t, err)
}

func TestExtractMetrics(t *testing.T) {
	for i := 0; i < 20; i++ {
		md := testdata.GenerateMetrics(10)
		extractedMetrics := extractMetrics(md, i)
		assert.Equal(t, i, extractedMetrics.DataPointCount())
		assert.Equal(t, 20-i, md.DataPointCount())
	}
}

func TestExtractMetricsInvalidMetric(t *testing.T) {
	md := testdata.GenerateMetricsMetricTypeInvalid()
	extractedMetrics := extractMetrics(md, 10)
	assert.Equal(t, testdata.GenerateMetricsMetricTypeInvalid(), extractedMetrics)
	assert.Equal(t, 0, md.ResourceMetrics().Len())
}

func TestMergeSplitManySmallMetrics(t *testing.T) {
	// All requests merge into a single batch.
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 20000}
	merged := []Request{newMetricsRequest(testdata.GenerateMetrics(1), nil)}
	for j := 0; j < 1000; j++ {
		lr2 := newMetricsRequest(testdata.GenerateMetrics(10), nil)
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
		merged = append(merged[0:len(merged)-1], res...)
	}
	assert.Len(t, merged, 2)
}

func BenchmarkSplittingBasedOnItemCountManySmallMetrics(b *testing.B) {
	// All requests merge into a single batch.
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 20020}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newMetricsRequest(testdata.GenerateMetrics(10), nil)}
		for j := 0; j < 1000; j++ {
			lr2 := newMetricsRequest(testdata.GenerateMetrics(10), nil)
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 1)
	}
}

func BenchmarkSplittingBasedOnItemCountManyMetricsSlightlyAboveLimit(b *testing.B) {
	// Every incoming request results in a split.
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 20000}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newMetricsRequest(testdata.GenerateMetrics(0), nil)}
		for j := 0; j < 10; j++ {
			lr2 := newMetricsRequest(testdata.GenerateMetrics(10001), nil)
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 11)
	}
}

func BenchmarkSplittingBasedOnItemCountHugeMetrics(b *testing.B) {
	// One request splits into many batches.
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 20000}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newMetricsRequest(testdata.GenerateMetrics(0), nil)}
		lr2 := newMetricsRequest(testdata.GenerateMetrics(100000), nil)
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
		merged = append(merged[0:len(merged)-1], res...)
		assert.Len(b, merged, 10)
	}
}
