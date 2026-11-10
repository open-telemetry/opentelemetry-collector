// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMergeMetrics(t *testing.T) {
	mr1 := newMetricsRequest(testdata.GenerateMetrics(2))
	mr2 := newMetricsRequest(testdata.GenerateMetrics(3))
	res, err := mr1.MergeSplit(context.Background(), 0, request.SizerTypeItems, mr2)
	require.NoError(t, err)
	// Every metric has 2 data points.
	assert.Equal(t, 2*5, res[0].ItemsCount())
}

func TestMergeSplitMetrics(t *testing.T) {
	s := sizer.MetricsCountSizer{}
	tests := []struct {
		name     string
		szt      request.SizerType
		maxSize  int
		mr1      request.Request
		mr2      request.Request
		expected []request.Request
	}{
		{
			name:     "both_requests_empty",
			szt:      request.SizerTypeItems,
			maxSize:  10,
			mr1:      newMetricsRequest(pmetric.NewMetrics()),
			mr2:      newMetricsRequest(pmetric.NewMetrics()),
			expected: []request.Request{newMetricsRequest(pmetric.NewMetrics())},
		},
		{
			name:     "first_request_empty",
			szt:      request.SizerTypeItems,
			maxSize:  10,
			mr1:      newMetricsRequest(pmetric.NewMetrics()),
			mr2:      newMetricsRequest(testdata.GenerateMetrics(5)),
			expected: []request.Request{newMetricsRequest(testdata.GenerateMetrics(5))},
		},
		{
			name:     "first_empty_second_nil",
			szt:      request.SizerTypeItems,
			maxSize:  10,
			mr1:      newMetricsRequest(pmetric.NewMetrics()),
			mr2:      nil,
			expected: []request.Request{newMetricsRequest(pmetric.NewMetrics())},
		},
		{
			name:    "merge_only",
			szt:     request.SizerTypeItems,
			maxSize: 60,
			mr1:     newMetricsRequest(testdata.GenerateMetrics(10)),
			mr2:     newMetricsRequest(testdata.GenerateMetrics(14)),
			expected: []request.Request{newMetricsRequest(func() pmetric.Metrics {
				metrics := testdata.GenerateMetrics(10)
				testdata.GenerateMetrics(14).ResourceMetrics().MoveAndAppendTo(metrics.ResourceMetrics())
				return metrics
			}())},
		},
		{
			name:    "split_only",
			szt:     request.SizerTypeItems,
			maxSize: 14,
			mr1:     newMetricsRequest(pmetric.NewMetrics()),
			mr2:     newMetricsRequest(testdata.GenerateMetrics(15)), // 15 metrics, 30 data points
			expected: []request.Request{
				newMetricsRequest(testdata.GenerateMetrics(7)), // 7 metrics, 14 data points
				newMetricsRequest(testdata.GenerateMetrics(7)), // 7 metrics, 14 data points
				newMetricsRequest(testdata.GenerateMetrics(1)), // 1 metric, 2 data points
			},
		},
		{
			name:    "split_and_merge",
			szt:     request.SizerTypeItems,
			maxSize: 28,
			mr1:     newMetricsRequest(testdata.GenerateMetrics(7)),  // 7 metrics, 14 data points
			mr2:     newMetricsRequest(testdata.GenerateMetrics(25)), // 25 metrics, 50 data points
			expected: []request.Request{
				newMetricsRequest(func() pmetric.Metrics {
					metrics := testdata.GenerateMetrics(7)
					testdata.GenerateMetrics(7).ResourceMetrics().MoveAndAppendTo(metrics.ResourceMetrics())
					return metrics
				}()),
				newMetricsRequest(testdata.GenerateMetrics(14)), // 14 metrics, 28 data points
				newMetricsRequest(testdata.GenerateMetrics(4)),  // 4 metrics, 8 data points
			},
		},
		{
			name:    "scope_metrics_split",
			szt:     request.SizerTypeItems,
			maxSize: 8,
			mr1: newMetricsRequest(func() pmetric.Metrics {
				md := testdata.GenerateMetrics(4)
				extraScopeMetrics := md.ResourceMetrics().At(0).ScopeMetrics().AppendEmpty()
				testdata.GenerateMetrics(4).ResourceMetrics().At(0).ScopeMetrics().At(0).MoveTo(extraScopeMetrics)
				extraScopeMetrics.Scope().SetName("extra scope")
				return md
			}()),
			mr2: nil,
			expected: []request.Request{
				newMetricsRequest(testdata.GenerateMetrics(4)),
				newMetricsRequest(func() pmetric.Metrics {
					md := testdata.GenerateMetrics(4)
					md.ResourceMetrics().At(0).ScopeMetrics().At(0).Scope().SetName("extra scope")
					return md
				}()),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.mr1.MergeSplit(context.Background(), tt.maxSize, tt.szt, tt.mr2)
			require.NoError(t, err)
			assert.Len(t, res, len(tt.expected))
			for i := range res {
				expected := tt.expected[i].(*metricsRequest)
				actual := res[i].(*metricsRequest)
				assert.Equal(t, expected.size(&s, request.SizerTypeItems), actual.size(&s, request.SizerTypeItems))
			}
		})
	}
}

func TestSplitMetricsWithDataPointSplit(t *testing.T) {
	generateTestMetrics := func(metricType pmetric.MetricType) pmetric.Metrics {
		md := pmetric.NewMetrics()
		m := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
		m.SetName("test_metric")
		m.SetDescription("test_description")
		m.SetUnit("test_unit")
		m.Metadata().PutStr("test_metadata_key", "test_metadata_value")

		const numDataPoints = 2

		switch metricType {
		case pmetric.MetricTypeSum:
			sum := m.SetEmptySum()
			for i := range numDataPoints {
				sum.DataPoints().AppendEmpty().SetIntValue(int64(i + 1))
			}
		case pmetric.MetricTypeGauge:
			gauge := m.SetEmptyGauge()
			for i := range numDataPoints {
				gauge.DataPoints().AppendEmpty().SetIntValue(int64(i + 1))
			}
		case pmetric.MetricTypeHistogram:
			hist := m.SetEmptyHistogram()
			for i := range uint64(numDataPoints) {
				hist.DataPoints().AppendEmpty().SetCount(i + 1)
			}
		case pmetric.MetricTypeExponentialHistogram:
			expHist := m.SetEmptyExponentialHistogram()
			for i := range uint64(numDataPoints) {
				expHist.DataPoints().AppendEmpty().SetCount(i + 1)
			}
		case pmetric.MetricTypeSummary:
			summary := m.SetEmptySummary()
			for i := range uint64(numDataPoints) {
				summary.DataPoints().AppendEmpty().SetCount(i + 1)
			}
		}
		return md
	}

	tests := []struct {
		name       string
		metricType pmetric.MetricType
	}{
		{
			name:       "sum",
			metricType: pmetric.MetricTypeSum,
		},
		{
			name:       "gauge",
			metricType: pmetric.MetricTypeGauge,
		},
		{
			name:       "histogram",
			metricType: pmetric.MetricTypeHistogram,
		},
		{
			name:       "exponential_histogram",
			metricType: pmetric.MetricTypeExponentialHistogram,
		},
		{
			name:       "summary",
			metricType: pmetric.MetricTypeSummary,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate metrics with 2 data points.
			mr1 := newMetricsRequest(generateTestMetrics(tt.metricType))

			// Split by data point, so maxSize is 1.
			res, err := mr1.MergeSplit(context.Background(), 1, request.SizerTypeItems, nil)
			require.NoError(t, err)
			require.Len(t, res, 2)

			for _, req := range res {
				actualRequest := req.(*metricsRequest)
				// Each split request should contain one data point.
				assert.Equal(t, 1, actualRequest.ItemsCount())
				m := actualRequest.md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
				assert.Equal(t, "test_metric", m.Name())
				assert.Equal(t, "test_description", m.Description())
				assert.Equal(t, "test_unit", m.Unit())
				assert.Equal(t, 1, m.Metadata().Len())
				val, ok := m.Metadata().Get("test_metadata_key")
				assert.True(t, ok)
				assert.Equal(t, "test_metadata_value", val.AsString())
			}
		})
	}
}

func TestMergeSplitMetricsInputNotModifiedIfErrorReturned(t *testing.T) {
	r1 := newMetricsRequest(testdata.GenerateMetrics(18)) // 18 metrics, 36 data points
	r2 := newLogsRequest(testdata.GenerateLogs(3))
	_, err := r1.MergeSplit(context.Background(), 10, request.SizerTypeItems, r2)
	require.Error(t, err)
	assert.Equal(t, 36, r1.ItemsCount())
}

func TestExtractMetrics(t *testing.T) {
	for i := range 20 {
		md := testdata.GenerateMetrics(10)
		extractedMetrics, _ := extractMetrics(md, i, &sizer.MetricsCountSizer{})
		assert.Equal(t, i, extractedMetrics.DataPointCount())
		assert.Equal(t, 20-i, md.DataPointCount())
	}
}

func TestExtractMetricsInvalidMetric(t *testing.T) {
	md := testdata.GenerateMetricsMetricTypeInvalid()
	extractedMetrics, _ := extractMetrics(md, 10, &sizer.MetricsCountSizer{})
	assert.Equal(t, testdata.GenerateMetricsMetricTypeInvalid(), extractedMetrics)
	assert.Equal(t, 0, md.ResourceMetrics().Len())
}

func TestMergeSplitManySmallMetrics(t *testing.T) {
	// All requests merge into a single batch.
	merged := []request.Request{newMetricsRequest(testdata.GenerateMetrics(1))}
	for range 1000 {
		lr2 := newMetricsRequest(testdata.GenerateMetrics(10))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 20000, request.SizerTypeItems, lr2)
		merged = append(merged[0:len(merged)-1], res...)
	}
	assert.Len(t, merged, 2)
}

func BenchmarkSplittingBasedOnItemCountManySmallMetrics(b *testing.B) {
	testutil.SkipGCHeavyBench(b)
	// All requests merge into a single batch.
	b.ReportAllocs()
	for b.Loop() {
		merged := []request.Request{newMetricsRequest(testdata.GenerateMetrics(10))}
		for range 1000 {
			lr2 := newMetricsRequest(testdata.GenerateMetrics(10))
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 20020, request.SizerTypeItems, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 1)
	}
}

func BenchmarkSplittingBasedOnItemCountManyMetricsSlightlyAboveLimit(b *testing.B) {
	testutil.SkipGCHeavyBench(b)
	// Every incoming request results in a split.
	b.ReportAllocs()
	for b.Loop() {
		merged := []request.Request{newMetricsRequest(testdata.GenerateMetrics(0))}
		for range 10 {
			lr2 := newMetricsRequest(testdata.GenerateMetrics(10001))
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 20000, request.SizerTypeItems, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 11)
	}
}

func BenchmarkSplittingBasedOnItemCountHugeMetrics(b *testing.B) {
	testutil.SkipGCHeavyBench(b)
	// One request splits into many batches.
	b.ReportAllocs()
	for b.Loop() {
		merged := []request.Request{newMetricsRequest(testdata.GenerateMetrics(0))}
		lr2 := newMetricsRequest(testdata.GenerateMetrics(100000))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 20000, request.SizerTypeItems, lr2)
		merged = append(merged[0:len(merged)-1], res...)
		assert.Len(b, merged, 10)
	}
}

func TestMergeSplitMetricsBasedOnByteSize(t *testing.T) {
	tests := []struct {
		name          string
		szt           request.SizerType
		maxSize       int
		mr1           request.Request
		mr2           request.Request
		expected      []request.Request
		expectedError string
	}{
		{
			name:     "both_requests_empty",
			szt:      request.SizerTypeBytes,
			maxSize:  metricsMarshaler.MetricsSize(testdata.GenerateMetrics(10)),
			mr1:      newMetricsRequest(pmetric.NewMetrics()),
			mr2:      newMetricsRequest(pmetric.NewMetrics()),
			expected: []request.Request{newMetricsRequest(pmetric.NewMetrics())},
		},
		{
			name:     "first_request_empty",
			szt:      request.SizerTypeBytes,
			maxSize:  metricsMarshaler.MetricsSize(testdata.GenerateMetrics(10)),
			mr1:      newMetricsRequest(pmetric.NewMetrics()),
			mr2:      newMetricsRequest(testdata.GenerateMetrics(5)),
			expected: []request.Request{newMetricsRequest(testdata.GenerateMetrics(5))},
		},
		{
			name:     "first_empty_second_nil",
			szt:      request.SizerTypeBytes,
			maxSize:  metricsMarshaler.MetricsSize(testdata.GenerateMetrics(10)),
			mr1:      newMetricsRequest(pmetric.NewMetrics()),
			mr2:      nil,
			expected: []request.Request{newMetricsRequest(pmetric.NewMetrics())},
		},
		{
			name:    "merge_only",
			szt:     request.SizerTypeBytes,
			maxSize: metricsMarshaler.MetricsSize(testdata.GenerateMetrics(15)) - 1,
			mr1:     newMetricsRequest(testdata.GenerateMetrics(7)),
			mr2:     newMetricsRequest(testdata.GenerateMetrics(7)),
			expected: []request.Request{newMetricsRequest(func() pmetric.Metrics {
				md := testdata.GenerateMetrics(7)
				testdata.GenerateMetrics(7).ResourceMetrics().MoveAndAppendTo(md.ResourceMetrics())
				return md
			}())},
		},
		{
			name:    "split_only",
			szt:     request.SizerTypeBytes,
			maxSize: metricsMarshaler.MetricsSize(testdata.GenerateMetrics(7)) + 1,
			mr1:     newMetricsRequest(pmetric.NewMetrics()),
			mr2:     newMetricsRequest(testdata.GenerateMetrics(17)),
			expected: []request.Request{
				newMetricsRequest(testdata.GenerateMetrics(7)),
				newMetricsRequest(testdata.GenerateMetrics(7)),
				newMetricsRequest(testdata.GenerateMetrics(3)),
			},
		},
		{
			name:    "merge_and_split",
			szt:     request.SizerTypeBytes,
			maxSize: metricsMarshaler.MetricsSize(testdata.GenerateMetrics(7)) + 1,
			mr1:     newMetricsRequest(testdata.GenerateMetrics(14)),
			mr2:     newMetricsRequest(testdata.GenerateMetrics(11)),
			expected: []request.Request{
				newMetricsRequest(testdata.GenerateMetrics(7)),
				newMetricsRequest(testdata.GenerateMetrics(7)),
				newMetricsRequest(testdata.GenerateMetrics(7)),
				newMetricsRequest(testdata.GenerateMetrics(4)),
			},
		},
		{
			name:    "scope_metrics_split",
			szt:     request.SizerTypeBytes,
			maxSize: metricsMarshaler.MetricsSize(testdata.GenerateMetrics(7)) + 1,
			mr1: newMetricsRequest(func() pmetric.Metrics {
				md := testdata.GenerateMetrics(7)
				extraScopeMetrics := md.ResourceMetrics().At(0).ScopeMetrics().AppendEmpty()
				testdata.GenerateMetrics(7).ResourceMetrics().At(0).ScopeMetrics().At(0).MoveTo(extraScopeMetrics)
				extraScopeMetrics.Scope().SetName("extra scope")
				return md
			}()),
			mr2: nil,
			expected: []request.Request{
				newMetricsRequest(testdata.GenerateMetrics(7)),
				newMetricsRequest(func() pmetric.Metrics {
					md := testdata.GenerateMetrics(7)
					md.ResourceMetrics().At(0).ScopeMetrics().At(0).Scope().SetName("extra scope")
					// Remove last data point.
					lastDP := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(6).Summary().DataPoints().Len()
					idx := 0
					md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(6).Summary().DataPoints().RemoveIf(func(pmetric.SummaryDataPoint) bool {
						idx++
						return idx == lastDP
					})
					return md
				}()),
				newMetricsRequest(func() pmetric.Metrics {
					md := testdata.GenerateMetrics(7)
					md.ResourceMetrics().At(0).ScopeMetrics().At(0).Scope().SetName("extra scope")
					// Remove all metrics but last one
					lastM := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().Len()
					idx := 0
					md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().RemoveIf(func(pmetric.Metric) bool {
						idx++
						return idx != lastM
					})
					// Remove all data points but last one
					lastDP := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Summary().DataPoints().Len()
					idx = 0
					md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Summary().DataPoints().RemoveIf(func(pmetric.SummaryDataPoint) bool {
						idx++
						return idx != lastDP
					})
					return md
				}()),
			},
		},
		{
			name:    "unsplittable_large_metric",
			szt:     request.SizerTypeBytes,
			maxSize: 10,
			mr1: newMetricsRequest(func() pmetric.Metrics {
				md := testdata.GenerateMetrics(1)
				md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).SetDescription(string(make([]byte, 100)))
				return md
			}()),
			mr2:           nil,
			expected:      []request.Request{},
			expectedError: "request size is greater than max size and cannot be split further, dropping items:",
		},
		{
			name:    "splittable_then_unsplittable_metric",
			szt:     request.SizerTypeBytes,
			maxSize: 1000,
			mr1: newMetricsRequest(func() pmetric.Metrics {
				md := testdata.GenerateMetrics(2)
				md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).SetDescription(string(make([]byte, 10)))
				md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(1).SetDescription(string(make([]byte, 1001)))
				return md
			}()),
			mr2: nil,
			expected: []request.Request{newMetricsRequest(func() pmetric.Metrics {
				md := testdata.GenerateMetrics(1)
				md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).SetDescription(string(make([]byte, 10)))
				return md
			}())},
			expectedError: "single data point exceeds the max size limit, dropping items:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.mr1.MergeSplit(context.Background(), tt.maxSize, tt.szt, tt.mr2)
			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
			}
			require.Len(t, res, len(tt.expected))
			for i := range res {
				assert.Equal(t, tt.expected[i].(*metricsRequest).md, res[i].(*metricsRequest).md, i)
				assert.Equal(t,
					metricsMarshaler.MetricsSize(tt.expected[i].(*metricsRequest).md),
					metricsMarshaler.MetricsSize(res[i].(*metricsRequest).md))
			}
		})
	}
}

func TestExtractGaugeDataPoints(t *testing.T) {
	tests := []struct {
		name           string
		capacity       int
		numDataPoints  int
		expectedPoints int
	}{
		{
			name:           "extract_all_points",
			capacity:       100,
			numDataPoints:  2,
			expectedPoints: 2,
		},
		{
			name:           "extract_partial_points",
			capacity:       1,
			numDataPoints:  2,
			expectedPoints: 1,
		},
		{
			name:           "no_capacity",
			capacity:       0,
			numDataPoints:  2,
			expectedPoints: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcMetric := pmetric.NewMetric()
			gauge := srcMetric.SetEmptyGauge()
			for i := 0; i < tt.numDataPoints; i++ {
				dp := gauge.DataPoints().AppendEmpty()
				dp.SetIntValue(int64(i))
			}

			sz := &mockMetricsSizer{dpSize: 1}

			destMetric := pmetric.NewMetric()
			extractGaugeDataPoints(gauge, destMetric, tt.capacity, tt.capacity, sz)

			assert.Equal(t, tt.expectedPoints, destMetric.Gauge().DataPoints().Len())
		})
	}
}

func TestExtractSumDataPoints(t *testing.T) {
	tests := []struct {
		name           string
		capacity       int
		numDataPoints  int
		expectedPoints int
	}{
		{
			name:           "extract_all_points",
			capacity:       100,
			numDataPoints:  2,
			expectedPoints: 2,
		},
		{
			name:           "extract_partial_points",
			capacity:       1,
			numDataPoints:  2,
			expectedPoints: 1,
		},
		{
			name:           "no_capacity",
			capacity:       0,
			numDataPoints:  2,
			expectedPoints: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcMetric := pmetric.NewMetric()
			sum := srcMetric.SetEmptySum()
			for i := 0; i < tt.numDataPoints; i++ {
				dp := sum.DataPoints().AppendEmpty()
				dp.SetIntValue(int64(i))
			}

			sz := &mockMetricsSizer{dpSize: 1}

			destMetric := pmetric.NewMetric()
			extractSumDataPoints(sum, destMetric, tt.capacity, tt.capacity, sz)

			assert.Equal(t, tt.expectedPoints, destMetric.Sum().DataPoints().Len())
		})
	}
}

func TestExtractHistogramDataPoints(t *testing.T) {
	tests := []struct {
		name           string
		capacity       int
		numDataPoints  int
		expectedPoints int
	}{
		{
			name:           "extract_all_points",
			capacity:       100,
			numDataPoints:  2,
			expectedPoints: 2,
		},
		{
			name:           "extract_partial_points",
			capacity:       1,
			numDataPoints:  2,
			expectedPoints: 1,
		},
		{
			name:           "no_capacity",
			capacity:       0,
			numDataPoints:  2,
			expectedPoints: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcMetric := pmetric.NewMetric()
			histogram := srcMetric.SetEmptyHistogram()

			for i := 0; i < tt.numDataPoints; i++ {
				dp := histogram.DataPoints().AppendEmpty()
				dp.SetCount(uint64(i))
			}

			sz := &mockMetricsSizer{dpSize: 1}

			destMetric := pmetric.NewMetric()
			extractHistogramDataPoints(histogram, destMetric, tt.capacity, tt.capacity, sz)

			assert.Equal(t, tt.expectedPoints, destMetric.Histogram().DataPoints().Len())
		})
	}
}

func TestExtractExponentialHistogramDataPoints(t *testing.T) {
	tests := []struct {
		name           string
		capacity       int
		numDataPoints  int
		expectedPoints int
	}{
		{
			name:           "extract_all_points",
			capacity:       100,
			numDataPoints:  2,
			expectedPoints: 2,
		},
		{
			name:           "extract_partial_points",
			capacity:       1,
			numDataPoints:  2,
			expectedPoints: 1,
		},
		{
			name:           "no_capacity",
			capacity:       0,
			numDataPoints:  2,
			expectedPoints: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcMetric := pmetric.NewMetric()
			expHistogram := srcMetric.SetEmptyExponentialHistogram()
			for i := 0; i < tt.numDataPoints; i++ {
				dp := expHistogram.DataPoints().AppendEmpty()
				dp.SetCount(uint64(i))
			}

			sz := &mockMetricsSizer{dpSize: 1}

			destMetric := pmetric.NewMetric()
			extractExponentialHistogramDataPoints(expHistogram, destMetric, tt.capacity, tt.capacity, sz)

			assert.Equal(t, tt.expectedPoints, destMetric.ExponentialHistogram().DataPoints().Len())
		})
	}
}

func TestExtractSummaryDataPoints(t *testing.T) {
	tests := []struct {
		name           string
		capacity       int
		numDataPoints  int
		expectedPoints int
	}{
		{
			name:           "extract_all_points",
			capacity:       100,
			numDataPoints:  2,
			expectedPoints: 2,
		},
		{
			name:           "extract_partial_points",
			capacity:       1,
			numDataPoints:  2,
			expectedPoints: 1,
		},
		{
			name:           "no_capacity",
			capacity:       0,
			numDataPoints:  2,
			expectedPoints: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcMetric := pmetric.NewMetric()
			summary := srcMetric.SetEmptySummary()
			for i := 0; i < tt.numDataPoints; i++ {
				dp := summary.DataPoints().AppendEmpty()
				dp.SetCount(uint64(i))
			}

			sz := &mockMetricsSizer{dpSize: 1}

			destMetric := pmetric.NewMetric()
			extractSummaryDataPoints(summary, destMetric, tt.capacity, tt.capacity, sz)

			assert.Equal(t, tt.expectedPoints, destMetric.Summary().DataPoints().Len())
		})
	}
}

func TestMetricsMergeSplitUnknownSizerType(t *testing.T) {
	req := newMetricsRequest(pmetric.NewMetrics())
	// Call MergeSplit with invalid sizer
	_, err := req.MergeSplit(context.Background(), 0, request.SizerType{}, nil)
	require.EqualError(t, err, "unknown sizer type")
}

// mockMetricsSizer implements sizer.MetricsSizer interface for testing
type mockMetricsSizer struct {
	dpSize int
}

func (m *mockMetricsSizer) MetricsSize(_ pmetric.Metrics) int {
	return 0
}

func (m *mockMetricsSizer) MetricSize(_ pmetric.Metric) int {
	return 0
}

func (m *mockMetricsSizer) NumberDataPointSize(_ pmetric.NumberDataPoint) int {
	return m.dpSize
}

func (m *mockMetricsSizer) HistogramDataPointSize(_ pmetric.HistogramDataPoint) int {
	return m.dpSize
}

func (m *mockMetricsSizer) ExponentialHistogramDataPointSize(_ pmetric.ExponentialHistogramDataPoint) int {
	return m.dpSize
}

func (m *mockMetricsSizer) SummaryDataPointSize(_ pmetric.SummaryDataPoint) int {
	return m.dpSize
}

func (m *mockMetricsSizer) ResourceMetricsSize(_ pmetric.ResourceMetrics) int {
	return 0
}

func (m *mockMetricsSizer) ScopeMetricsSize(_ pmetric.ScopeMetrics) int {
	return 0
}

func (m *mockMetricsSizer) DeltaSize(size int) int {
	return size
}

// dataPointLabels returns the "name" attribute of every data point across the given requests, in order.
func dataPointLabels(reqs []request.Request) []string {
	var out []string
	label := func(attrs pcommon.Map) {
		v, _ := attrs.Get("name")
		out = append(out, v.AsString())
	}
	for _, r := range reqs {
		rms := r.(*metricsRequest).md.ResourceMetrics()
		for i := 0; i < rms.Len(); i++ {
			sms := rms.At(i).ScopeMetrics()
			for j := 0; j < sms.Len(); j++ {
				ms := sms.At(j).Metrics()
				for k := 0; k < ms.Len(); k++ {
					m := ms.At(k)
					switch m.Type() {
					case pmetric.MetricTypeGauge:
						for l := 0; l < m.Gauge().DataPoints().Len(); l++ {
							label(m.Gauge().DataPoints().At(l).Attributes())
						}
					case pmetric.MetricTypeSum:
						for l := 0; l < m.Sum().DataPoints().Len(); l++ {
							label(m.Sum().DataPoints().At(l).Attributes())
						}
					case pmetric.MetricTypeHistogram:
						for l := 0; l < m.Histogram().DataPoints().Len(); l++ {
							label(m.Histogram().DataPoints().At(l).Attributes())
						}
					case pmetric.MetricTypeExponentialHistogram:
						for l := 0; l < m.ExponentialHistogram().DataPoints().Len(); l++ {
							label(m.ExponentialHistogram().DataPoints().At(l).Attributes())
						}
					case pmetric.MetricTypeSummary:
						for l := 0; l < m.Summary().DataPoints().Len(); l++ {
							label(m.Summary().DataPoints().At(l).Attributes())
						}
					}
				}
			}
		}
	}
	return out
}

// addDataPoint appends one data point of the metric's type with a "name" attribute set to label.
func addDataPoint(m pmetric.Metric, label string) {
	var attrs pcommon.Map
	switch m.Type() {
	case pmetric.MetricTypeGauge:
		attrs = m.Gauge().DataPoints().AppendEmpty().Attributes()
	case pmetric.MetricTypeSum:
		attrs = m.Sum().DataPoints().AppendEmpty().Attributes()
	case pmetric.MetricTypeHistogram:
		attrs = m.Histogram().DataPoints().AppendEmpty().Attributes()
	case pmetric.MetricTypeExponentialHistogram:
		attrs = m.ExponentialHistogram().DataPoints().AppendEmpty().Attributes()
	case pmetric.MetricTypeSummary:
		attrs = m.Summary().DataPoints().AppendEmpty().Attributes()
	}
	attrs.PutStr("name", label)
}

func newMetricOfType(m pmetric.Metric, mt pmetric.MetricType) {
	switch mt {
	case pmetric.MetricTypeGauge:
		m.SetEmptyGauge()
	case pmetric.MetricTypeSum:
		m.SetEmptySum()
	case pmetric.MetricTypeHistogram:
		m.SetEmptyHistogram()
	case pmetric.MetricTypeExponentialHistogram:
		m.SetEmptyExponentialHistogram()
	case pmetric.MetricTypeSummary:
		m.SetEmptySummary()
	}
}

// newMetricsWithDataPoints builds one resource, scope and metric of the given type holding one
// data point per label.
func newMetricsWithDataPoints(mt pmetric.MetricType, labels ...string) pmetric.Metrics {
	md := pmetric.NewMetrics()
	m := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	newMetricOfType(m, mt)
	for _, l := range labels {
		addDataPoint(m, l)
	}
	return md
}

var allMetricTypes = []pmetric.MetricType{
	pmetric.MetricTypeGauge,
	pmetric.MetricTypeSum,
	pmetric.MetricTypeHistogram,
	pmetric.MetricTypeExponentialHistogram,
	pmetric.MetricTypeSummary,
}

func TestMergeSplitMetricsDropsOnlyOversizedDataPoint(t *testing.T) {
	oversized := strings.Repeat("x", 1000)
	// More data points than one batch holds, so the metric still has data points after
	// the pass that drops the oversized one.
	many := make([]string, 30)
	for i := range many {
		many[i] = fmt.Sprintf("dp-%02d", i)
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

	for _, mt := range allMetricTypes {
		for _, tt := range tests {
			t.Run(mt.String()+"/"+tt.name, func(t *testing.T) {
				req := newMetricsRequest(newMetricsWithDataPoints(mt, tt.labels...))
				res, err := req.MergeSplit(context.Background(), 150, request.SizerTypeBytes, nil)

				wantErr := fmt.Sprintf("single data point exceeds the max size limit, dropping items: %d", tt.wantDropped)
				require.ErrorContains(t, err, wantErr)
				assert.Equal(t, tt.wantSurvived, dataPointLabels(res),
					"data points other than the oversized ones must survive")

				for _, r := range res {
					assert.LessOrEqual(t, r.BytesSize(), 150, "no returned batch may exceed max size")
					assert.Equal(t, metricsMarshaler.MetricsSize(r.(*metricsRequest).md), r.BytesSize(),
						"the cached size must stay exact after dropping data points")
				}
			})
		}
	}
}

func TestMergeSplitMetricsAllDataPointsOversized(t *testing.T) {
	oversized := strings.Repeat("x", 1000)
	for _, mt := range allMetricTypes {
		t.Run(mt.String(), func(t *testing.T) {
			req := newMetricsRequest(newMetricsWithDataPoints(mt, oversized, oversized))

			res, err := req.MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
			require.ErrorContains(t, err, "single data point exceeds the max size limit, dropping items: 2")
			assert.Empty(t, dataPointLabels(res), "nothing can be exported when every data point is oversized")
		})
	}
}

func TestMergeSplitMetricsItemlessOversizedRequest(t *testing.T) {
	// Resource attributes alone exceed max size and there is no data point at all, so
	// nothing can be exported and nothing is lost that needs reporting.
	md := pmetric.NewMetrics()
	md.ResourceMetrics().AppendEmpty().Resource().Attributes().PutStr("big", strings.Repeat("x", 500))
	req := newMetricsRequest(md)
	require.Greater(t, req.BytesSize(), 100, "precondition: request must start oversized")

	res, err := req.MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
	require.NoError(t, err, "nothing was lost, so nothing is due to be reported")
	assert.Empty(t, res, "an oversized request holding no data points must not be returned")
}

func TestMergeSplitMetricsOversizedMetricHeaderDropsItsDataPoints(t *testing.T) {
	// The metric description fills or exceeds max size, so none of its data points can ever be
	// exported. They are dropped and reported, while the data points of the next metric survive.
	// At maxSize=100, empty Resource and Scope wrappers take 2+2 bytes plus 2 bytes per container
	// DeltaSize overhead and 2 bytes empty metric data message, so descLen=86 leaves
	// maxDataPointSize == 0 exactly.
	for _, tc := range []struct {
		name    string
		descLen int
	}{
		{name: "exact_zero_budget", descLen: 86},
		{name: "slightly_negative_budget", descLen: 90},
		{name: "exceeds_max_size", descLen: 300},
	} {
		for _, mt := range allMetricTypes {
			t.Run(tc.name+"/"+mt.String(), func(t *testing.T) {
				md := pmetric.NewMetrics()
				ms := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics()
				big := ms.AppendEmpty()
				big.SetDescription(strings.Repeat("d", tc.descLen))
				newMetricOfType(big, mt)
				addDataPoint(big, "lost_1")
				addDataPoint(big, "lost_2")
				small := ms.AppendEmpty()
				newMetricOfType(small, mt)
				addDataPoint(small, "kept")

				res, err := newMetricsRequest(md).MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
				require.ErrorContains(t, err, "single data point exceeds the max size limit, dropping items: 2")
				assert.Equal(t, []string{"kept"}, dataPointLabels(res))
				for _, r := range res {
					mr := r.(*metricsRequest)
					assert.LessOrEqual(t, metricsMarshaler.MetricsSize(mr.md), 100)
					assert.Equal(t, metricsMarshaler.MetricsSize(mr.md), mr.BytesSize())
				}
			})
		}
	}
}

func TestMergeSplitMetricsDropsOnlyOversizedAcrossResourcesScopesAndMetrics(t *testing.T) {
	// Only the oversized data point is dropped; data points in the other metric, scope and
	// resource must survive intact.
	oversized := strings.Repeat("x", 1000)
	md := pmetric.NewMetrics()
	rm1 := md.ResourceMetrics().AppendEmpty()
	sm1 := rm1.ScopeMetrics().AppendEmpty()
	m1 := sm1.Metrics().AppendEmpty()
	m1.SetEmptySum()
	addDataPoint(m1, oversized)
	m2 := sm1.Metrics().AppendEmpty()
	m2.SetEmptyGauge()
	addDataPoint(m2, "second_metric")
	m3 := rm1.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	m3.SetEmptyHistogram()
	addDataPoint(m3, "second_scope")
	m4 := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	m4.SetEmptySummary()
	addDataPoint(m4, "second_resource")
	require.Equal(t, 4, md.DataPointCount(), "precondition: four data points")

	res, err := newMetricsRequest(md).MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
	require.ErrorContains(t, err, "single data point exceeds the max size limit, dropping items: 1")
	assert.ElementsMatch(t, []string{"second_metric", "second_scope", "second_resource"}, dataPointLabels(res),
		"data points in the other metric, scope and resource must survive")
}

func TestMergeSplitMetricsEmptyOversizedResourceDoesNotStopSplitting(t *testing.T) {
	// A resource with big attributes and no data points must not stop splitting of the
	// data points behind it, and no data point may be reported as dropped.
	md := pmetric.NewMetrics()
	empty := md.ResourceMetrics().AppendEmpty()
	empty.Resource().Attributes().PutStr("big", strings.Repeat("B", 400))
	empty.ScopeMetrics().AppendEmpty()
	m := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	m.SetEmptyGauge()
	for i := range 12 {
		addDataPoint(m, fmt.Sprintf("%02d-%s", i, strings.Repeat("v", 30)))
	}
	req := newMetricsRequest(md).(*metricsRequest)
	require.Equal(t, 12, req.md.DataPointCount(), "precondition: twelve data points that each fit")
	require.Greater(t, req.BytesSize(), 100, "precondition: request starts oversized")

	res, err := req.MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
	require.NoError(t, err, "no data point is oversized, so nothing should be reported")

	survived := 0
	for _, r := range res {
		mr := r.(*metricsRequest)
		survived += mr.md.DataPointCount()
		assert.LessOrEqual(t, metricsMarshaler.MetricsSize(mr.md), 100, "no batch may exceed max size")
		assert.Equal(t, metricsMarshaler.MetricsSize(mr.md), mr.BytesSize(),
			"the cached size must stay exact after removing a data-point-less resource")
	}
	assert.Equal(t, 12, survived, "every data point must survive")
}

func TestMergeSplitMetricsStopsWhenNoProgressIsPossible(t *testing.T) {
	// The resource attributes alone fill or exceed max size, so no scope, metric or data point
	// can be added to any batch. Splitting must stop instead of looping, and must not return
	// a batch larger than max size.
	const maxSize = 300
	exactAttr := pmetric.NewMetrics()
	exactRM := exactAttr.ResourceMetrics().AppendEmpty()
	for pad := 0; ; pad++ {
		exactRM.Resource().Attributes().PutStr("pad", strings.Repeat("p", pad))
		if metricsMarshaler.MetricsSize(exactAttr) == maxSize {
			break
		}
	}
	exactPad := exactRM.Resource().Attributes().AsRaw()["pad"].(string)

	for _, tc := range []struct {
		name string
		pad  string
	}{
		{name: "exact_max_size", pad: exactPad},
		{name: "exceeds_max_size", pad: strings.Repeat("p", 400)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			md := pmetric.NewMetrics()
			rm := md.ResourceMetrics().AppendEmpty()
			rm.Resource().Attributes().PutStr("pad", tc.pad)
			m := rm.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
			m.SetEmptyGauge()
			addDataPoint(m, "first")
			addDataPoint(m, "second")

			res, err := newMetricsRequest(md).MergeSplit(context.Background(), maxSize, request.SizerTypeBytes, nil)
			require.ErrorContains(t, err, "request size is greater than max size and cannot be split further, dropping items: 2")
			assert.Empty(t, res, "data points that cannot fit any batch must not be returned")
		})
	}
}

func TestMergeSplitMetricsPartialCapacitySmallerThanNextHeader(t *testing.T) {
	// First item partially fills batch 1 leaving positive remaining capacity that is smaller
	// than the next item's header (making capacityLeft < 0 inside extraction while still
	// fitting inside maxSize). The trailing data points must survive in the next batch.
	const maxSize = 150

	t.Run("metric_header", func(t *testing.T) {
		firstOnly := pmetric.NewMetrics()
		m1 := firstOnly.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
		m1.SetEmptyGauge()
		addDataPoint(m1, strings.Repeat("a", 50))
		firstSize := metricsMarshaler.MetricsSize(firstOnly)

		secondHeaderOnly := pmetric.NewMetrics()
		m2Hdr := secondHeaderOnly.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
		m2Hdr.SetDescription(strings.Repeat("h", 80))
		m2Hdr.SetEmptyGauge()
		require.Less(t, firstSize, maxSize)
		require.Greater(t, metricsMarshaler.MetricsSize(secondHeaderOnly), maxSize-firstSize,
			"precondition: remaining capacity must be smaller than second metric header")

		md := pmetric.NewMetrics()
		sm := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
		m1.CopyTo(sm.Metrics().AppendEmpty())
		m2 := sm.Metrics().AppendEmpty()
		m2.SetDescription(strings.Repeat("h", 80))
		m2.SetEmptyGauge()
		addDataPoint(m2, "second_1")
		addDataPoint(m2, "second_2")

		res, err := newMetricsRequest(md).MergeSplit(context.Background(), maxSize, request.SizerTypeBytes, nil)
		require.NoError(t, err)
		assert.Equal(t, []string{strings.Repeat("a", 50), "second_1", "second_2"}, dataPointLabels(res))
		for _, r := range res {
			mr := r.(*metricsRequest)
			assert.LessOrEqual(t, metricsMarshaler.MetricsSize(mr.md), maxSize)
			assert.Equal(t, metricsMarshaler.MetricsSize(mr.md), mr.BytesSize())
		}
	})

	t.Run("scope_header", func(t *testing.T) {
		firstOnly := pmetric.NewMetrics()
		m1Only := firstOnly.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
		m1Only.SetEmptyGauge()
		addDataPoint(m1Only, strings.Repeat("a", 50))
		firstSize := metricsMarshaler.MetricsSize(firstOnly)

		secondHeaderOnly := pmetric.NewMetrics()
		sm2Hdr := secondHeaderOnly.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
		sm2Hdr.Scope().Attributes().PutStr("scope", strings.Repeat("s", 75))
		require.Less(t, firstSize, maxSize)
		require.Greater(t, metricsMarshaler.MetricsSize(secondHeaderOnly), maxSize-firstSize,
			"precondition: remaining capacity must be smaller than second scope header")

		md := pmetric.NewMetrics()
		rm := md.ResourceMetrics().AppendEmpty()
		m1Only.CopyTo(rm.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty())

		sm2 := rm.ScopeMetrics().AppendEmpty()
		sm2.Scope().Attributes().PutStr("scope", strings.Repeat("s", 75))
		m2 := sm2.Metrics().AppendEmpty()
		m2.SetEmptyGauge()
		addDataPoint(m2, "second_1")
		addDataPoint(m2, "second_2")

		res, err := newMetricsRequest(md).MergeSplit(context.Background(), maxSize, request.SizerTypeBytes, nil)
		require.NoError(t, err)
		assert.Equal(t, []string{strings.Repeat("a", 50), "second_1", "second_2"}, dataPointLabels(res))
		for _, r := range res {
			mr := r.(*metricsRequest)
			assert.LessOrEqual(t, metricsMarshaler.MetricsSize(mr.md), maxSize)
			assert.Equal(t, metricsMarshaler.MetricsSize(mr.md), mr.BytesSize())
		}
	})

	t.Run("resource_header", func(t *testing.T) {
		firstOnly := pmetric.NewMetrics()
		m1Only := firstOnly.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
		m1Only.SetEmptyGauge()
		addDataPoint(m1Only, strings.Repeat("a", 50))
		firstSize := metricsMarshaler.MetricsSize(firstOnly)

		secondHeaderOnly := pmetric.NewMetrics()
		rm2Hdr := secondHeaderOnly.ResourceMetrics().AppendEmpty()
		rm2Hdr.Resource().Attributes().PutStr("res", strings.Repeat("r", 75))
		require.Less(t, firstSize, maxSize)
		require.Greater(t, metricsMarshaler.MetricsSize(secondHeaderOnly), maxSize-firstSize,
			"precondition: remaining capacity must be smaller than second resource header")

		md := pmetric.NewMetrics()
		m1Only.CopyTo(md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty())

		rm2 := md.ResourceMetrics().AppendEmpty()
		rm2.Resource().Attributes().PutStr("res", strings.Repeat("r", 75))
		m2 := rm2.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
		m2.SetEmptyGauge()
		addDataPoint(m2, "second_1")
		addDataPoint(m2, "second_2")

		res, err := newMetricsRequest(md).MergeSplit(context.Background(), maxSize, request.SizerTypeBytes, nil)
		require.NoError(t, err)
		assert.Equal(t, []string{strings.Repeat("a", 50), "second_1", "second_2"}, dataPointLabels(res))
		for _, r := range res {
			mr := r.(*metricsRequest)
			assert.LessOrEqual(t, metricsMarshaler.MetricsSize(mr.md), maxSize)
			assert.Equal(t, metricsMarshaler.MetricsSize(mr.md), mr.BytesSize())
		}
	})
}

func TestMergeSplitMetricsNeverExceedsMaxSizeWhenDataPrefixGrows(t *testing.T) {
	// The data points sit inside a nested data message whose length prefix widens once its
	// content passes a varint width boundary. Every batch must stay within max size and keep
	// an exact cached size across those boundaries, for every metric type.
	for _, mt := range allMetricTypes {
		t.Run(mt.String(), func(t *testing.T) {
			for maxSize := 120; maxSize < 420; maxSize += 7 {
				md := pmetric.NewMetrics()
				m := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
				m.SetName("m")
				newMetricOfType(m, mt)
				switch mt {
				case pmetric.MetricTypeSum:
					m.Sum().SetIsMonotonic(true)
					m.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
				case pmetric.MetricTypeHistogram:
					m.Histogram().SetAggregationTemporality(pmetric.AggregationTemporalityDelta)
				case pmetric.MetricTypeExponentialHistogram:
					m.ExponentialHistogram().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
				}
				for i := range 120 {
					addDataPoint(m, strconv.Itoa(i))
				}

				res, err := newMetricsRequest(md).MergeSplit(context.Background(), maxSize, request.SizerTypeBytes, nil)
				require.NoError(t, err)
				survived := 0
				for _, r := range res {
					mr := r.(*metricsRequest)
					survived += mr.md.DataPointCount()
					assert.LessOrEqualf(t, metricsMarshaler.MetricsSize(mr.md), maxSize, "maxSize=%d: batch exceeds max size", maxSize)
					assert.Equalf(t, metricsMarshaler.MetricsSize(mr.md), mr.BytesSize(), "maxSize=%d: cached size drifted", maxSize)
				}
				assert.Equalf(t, 120, survived, "maxSize=%d: every data point must survive", maxSize)
			}
		})
	}
}

func TestMergeSplitMetricsEmptyMetricWrapperUnderOversizedResourceDoesNotStopSplitting(t *testing.T) {
	t.Run("oversized_resource_with_empty_metrics_before_valid_resource", func(t *testing.T) {
		md := pmetric.NewMetrics()
		emptyRM := md.ResourceMetrics().AppendEmpty()
		emptyRM.Resource().Attributes().PutStr("big", strings.Repeat("B", 400))
		emptySM := emptyRM.ScopeMetrics().AppendEmpty()
		for _, mt := range allMetricTypes {
			m := emptySM.Metrics().AppendEmpty()
			m.SetName("empty-" + mt.String())
			newMetricOfType(m, mt)
		}

		validM := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
		validM.SetEmptyGauge()
		for i := range 12 {
			addDataPoint(validM, fmt.Sprintf("%02d-%s", i, strings.Repeat("v", 30)))
		}
		req := newMetricsRequest(md).(*metricsRequest)
		require.Equal(t, 12, req.md.DataPointCount())
		require.Greater(t, req.BytesSize(), 100)

		res, err := req.MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
		require.NoError(t, err, "empty metric wrappers hold no data points, so nothing was lost")

		survived := 0
		for _, r := range res {
			mr := r.(*metricsRequest)
			survived += mr.md.DataPointCount()
			assert.LessOrEqual(t, metricsMarshaler.MetricsSize(mr.md), 100)
			assert.Equal(t, metricsMarshaler.MetricsSize(mr.md), mr.BytesSize(),
				"cached size must stay exact after pruning empty metric wrappers")
		}
		assert.Equal(t, 12, survived, "every trailing data point must survive")
	})

	t.Run("itemless_oversized_resource_with_empty_metric_wrapper", func(t *testing.T) {
		md := pmetric.NewMetrics()
		rm := md.ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().PutStr("big", strings.Repeat("B", 400))
		rm.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptyGauge()
		req := newMetricsRequest(md).(*metricsRequest)
		require.Greater(t, req.BytesSize(), 100)

		res, err := req.MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
		require.NoError(t, err, "no data point was dropped")
		assert.Empty(t, res)
	})

	t.Run("oversized_resource_with_mixed_empty_and_non_empty_metrics_stops_cleanly", func(t *testing.T) {
		md := pmetric.NewMetrics()
		rm := md.ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().PutStr("big", strings.Repeat("B", 400))
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Metrics().AppendEmpty().SetEmptyGauge()
		validM := sm.Metrics().AppendEmpty()
		validM.SetEmptyGauge()
		addDataPoint(validM, "first")
		addDataPoint(validM, "second")

		res, err := newMetricsRequest(md).MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
		require.ErrorContains(t, err, "request size is greater than max size and cannot be split further, dropping items: 2")
		assert.Empty(t, res)
	})

	t.Run("empty_metric_pruned_before_partial_scope_keeps_exact_size", func(t *testing.T) {
		md := pmetric.NewMetrics()
		rm := md.ResourceMetrics().AppendEmpty()

		sm0 := rm.ScopeMetrics().AppendEmpty()
		m0 := sm0.Metrics().AppendEmpty()
		m0.SetName(strings.Repeat("a", 50))
		m0.SetEmptyGauge()
		addDataPoint(m0, "s0")

		sm1 := rm.ScopeMetrics().AppendEmpty()
		mEmpty := sm1.Metrics().AppendEmpty()
		mEmpty.SetName(strings.Repeat("e", 120))
		mEmpty.SetEmptyGauge()
		mValid := sm1.Metrics().AppendEmpty()
		mValid.SetName(strings.Repeat("v", 30))
		mValid.SetEmptyGauge()
		addDataPoint(mValid, "s1")

		const maxSize = 100
		res, err := newMetricsRequest(md).MergeSplit(context.Background(), maxSize, request.SizerTypeBytes, nil)
		require.NoError(t, err)
		require.Len(t, res, 2)
		assert.Equal(t, []string{"s0", "s1"}, dataPointLabels(res))
		for _, r := range res {
			mr := r.(*metricsRequest)
			assert.LessOrEqual(t, mr.BytesSize(), maxSize)
			assert.Equal(t, metricsMarshaler.MetricsSize(mr.md), mr.BytesSize())
		}
	})

	t.Run("empty_metric_pruned_before_scope_allows_same_batch_pack", func(t *testing.T) {
		md := pmetric.NewMetrics()
		rm := md.ResourceMetrics().AppendEmpty()

		sm0 := rm.ScopeMetrics().AppendEmpty()
		m0 := sm0.Metrics().AppendEmpty()
		m0.SetEmptyGauge()
		addDataPoint(m0, "s0")

		sm1 := rm.ScopeMetrics().AppendEmpty()
		mEmpty := sm1.Metrics().AppendEmpty()
		mEmpty.SetName(strings.Repeat("e", 80))
		mEmpty.SetEmptyGauge()
		mValid := sm1.Metrics().AppendEmpty()
		mValid.SetEmptyGauge()
		addDataPoint(mValid, "s1")

		sm2 := rm.ScopeMetrics().AppendEmpty()
		m2 := sm2.Metrics().AppendEmpty()
		m2.SetEmptyGauge()
		addDataPoint(m2, "s2")

		const maxSize = 100
		res, err := newMetricsRequest(md).MergeSplit(context.Background(), maxSize, request.SizerTypeBytes, nil)
		require.NoError(t, err)
		require.Len(t, res, 1, "pruning empty metric before fitting sm1 lets sm0+sm1+sm2 stay in one batch")
		assert.Equal(t, []string{"s0", "s1", "s2"}, dataPointLabels(res))
		mr := res[0].(*metricsRequest)
		assert.LessOrEqual(t, mr.BytesSize(), maxSize)
		assert.Equal(t, metricsMarshaler.MetricsSize(mr.md), mr.BytesSize())
	})
}

func TestMergeSplitMetricsOversizedContainerHeaderDropsOnlyUnexportableContainer(t *testing.T) {
	sz := &sizer.MetricsBytesSizer{}

	t.Run("oversized_resource_header_followed_by_valid_resource", func(t *testing.T) {
		const maxSize = 110
		exactRM := pmetric.NewResourceMetrics()
		for pad := 0; ; pad++ {
			exactRM.Resource().Attributes().PutStr("pad", strings.Repeat("r", pad))
			if sz.ResourceMetricsSize(exactRM) == maxSize-(sz.DeltaSize(maxSize)-maxSize) {
				break
			}
		}
		exactPad := exactRM.Resource().Attributes().AsRaw()["pad"].(string)

		for _, tc := range []struct {
			name string
			pad  string
		}{
			{name: "exact_zero_budget", pad: exactPad},
			{name: "exceeds_max_size", pad: strings.Repeat("r", 300)},
		} {
			t.Run(tc.name, func(t *testing.T) {
				md := pmetric.NewMetrics()
				bigRM := md.ResourceMetrics().AppendEmpty()
				bigRM.Resource().Attributes().PutStr("pad", tc.pad)
				bigM := bigRM.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
				bigM.SetEmptyGauge()
				addDataPoint(bigM, "lost_1")
				addDataPoint(bigM, "lost_2")

				validRM := md.ResourceMetrics().AppendEmpty()
				validM := validRM.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
				validM.SetEmptyGauge()
				addDataPoint(validM, "kept_1")
				addDataPoint(validM, "kept_2")

				res, err := newMetricsRequest(md).MergeSplit(context.Background(), maxSize, request.SizerTypeBytes, nil)
				require.ErrorContains(t, err, "request size is greater than max size and cannot be split further, dropping items: 2")
				assert.Equal(t, []string{"kept_1", "kept_2"}, dataPointLabels(res))
				for _, r := range res {
					mr := r.(*metricsRequest)
					assert.LessOrEqual(t, metricsMarshaler.MetricsSize(mr.md), maxSize)
					assert.Equal(t, metricsMarshaler.MetricsSize(mr.md), mr.BytesSize())
				}
			})
		}
	})

	t.Run("oversized_scope_header_followed_by_valid_scope_and_resource", func(t *testing.T) {
		const maxSize = 110
		maxScopeSize := maxSize - (sz.DeltaSize(maxSize) - maxSize) - sz.ResourceMetricsSize(pmetric.NewResourceMetrics())
		exactSM := pmetric.NewScopeMetrics()
		for pad := 0; ; pad++ {
			exactSM.Scope().Attributes().PutStr("pad", strings.Repeat("s", pad))
			if sz.ScopeMetricsSize(exactSM) == maxScopeSize-(sz.DeltaSize(maxScopeSize)-maxScopeSize) {
				break
			}
		}
		exactPad := exactSM.Scope().Attributes().AsRaw()["pad"].(string)

		for _, tc := range []struct {
			name string
			pad  string
		}{
			{name: "exact_zero_budget", pad: exactPad},
			{name: "exceeds_max_size", pad: strings.Repeat("s", 300)},
		} {
			t.Run(tc.name, func(t *testing.T) {
				md := pmetric.NewMetrics()
				rm0 := md.ResourceMetrics().AppendEmpty()
				bigSM := rm0.ScopeMetrics().AppendEmpty()
				bigSM.Scope().Attributes().PutStr("pad", tc.pad)
				bigM := bigSM.Metrics().AppendEmpty()
				bigM.SetEmptyGauge()
				addDataPoint(bigM, "lost_1")
				addDataPoint(bigM, "lost_2")

				validSM := rm0.ScopeMetrics().AppendEmpty()
				validScopeM := validSM.Metrics().AppendEmpty()
				validScopeM.SetEmptyGauge()
				addDataPoint(validScopeM, "kept_scope")

				validRM := md.ResourceMetrics().AppendEmpty()
				validResM := validRM.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
				validResM.SetEmptyGauge()
				addDataPoint(validResM, "kept_resource")

				res, err := newMetricsRequest(md).MergeSplit(context.Background(), maxSize, request.SizerTypeBytes, nil)
				require.ErrorContains(t, err, "request size is greater than max size and cannot be split further, dropping items: 2")
				assert.Equal(t, []string{"kept_scope", "kept_resource"}, dataPointLabels(res))
				for _, r := range res {
					mr := r.(*metricsRequest)
					assert.LessOrEqual(t, metricsMarshaler.MetricsSize(mr.md), maxSize)
					assert.Equal(t, metricsMarshaler.MetricsSize(mr.md), mr.BytesSize())
				}
			})
		}
	})
}
