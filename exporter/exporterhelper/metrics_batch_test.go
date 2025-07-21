// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMergeMetrics(t *testing.T) {
	mr1 := newMetricsRequest(testdata.GenerateMetrics(2))
	mr2 := newMetricsRequest(testdata.GenerateMetrics(3))
	res, err := mr1.MergeSplit(context.Background(), 0, RequestSizerTypeItems, mr2)
	require.NoError(t, err)
	// Every metric has 2 data points.
	assert.Equal(t, 2*5, res[0].ItemsCount())
}

func TestMergeSplitMetrics(t *testing.T) {
	s := sizer.MetricsCountSizer{}
	tests := []struct {
		name     string
		szt      RequestSizerType
		maxSize  int
		mr1      Request
		mr2      Request
		expected []Request
	}{
		{
			name:     "both_requests_empty",
			szt:      RequestSizerTypeItems,
			maxSize:  10,
			mr1:      newMetricsRequest(pmetric.NewMetrics()),
			mr2:      newMetricsRequest(pmetric.NewMetrics()),
			expected: []Request{newMetricsRequest(pmetric.NewMetrics())},
		},
		{
			name:     "first_request_empty",
			szt:      RequestSizerTypeItems,
			maxSize:  10,
			mr1:      newMetricsRequest(pmetric.NewMetrics()),
			mr2:      newMetricsRequest(testdata.GenerateMetrics(5)),
			expected: []Request{newMetricsRequest(testdata.GenerateMetrics(5))},
		},
		{
			name:     "first_empty_second_nil",
			szt:      RequestSizerTypeItems,
			maxSize:  10,
			mr1:      newMetricsRequest(pmetric.NewMetrics()),
			mr2:      nil,
			expected: []Request{newMetricsRequest(pmetric.NewMetrics())},
		},
		{
			name:    "merge_only",
			szt:     RequestSizerTypeItems,
			maxSize: 60,
			mr1:     newMetricsRequest(testdata.GenerateMetrics(10)),
			mr2:     newMetricsRequest(testdata.GenerateMetrics(14)),
			expected: []Request{newMetricsRequest(func() pmetric.Metrics {
				metrics := testdata.GenerateMetrics(10)
				testdata.GenerateMetrics(14).ResourceMetrics().MoveAndAppendTo(metrics.ResourceMetrics())
				return metrics
			}())},
		},
		{
			name:    "split_only",
			szt:     RequestSizerTypeItems,
			maxSize: 14,
			mr1:     newMetricsRequest(pmetric.NewMetrics()),
			mr2:     newMetricsRequest(testdata.GenerateMetrics(15)), // 15 metrics, 30 data points
			expected: []Request{
				newMetricsRequest(testdata.GenerateMetrics(7)), // 7 metrics, 14 data points
				newMetricsRequest(testdata.GenerateMetrics(7)), // 7 metrics, 14 data points
				newMetricsRequest(testdata.GenerateMetrics(1)), // 1 metric, 2 data points
			},
		},
		{
			name:    "split_and_merge",
			szt:     RequestSizerTypeItems,
			maxSize: 28,
			mr1:     newMetricsRequest(testdata.GenerateMetrics(7)),  // 7 metrics, 14 data points
			mr2:     newMetricsRequest(testdata.GenerateMetrics(25)), // 25 metrics, 50 data points
			expected: []Request{
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
			szt:     RequestSizerTypeItems,
			maxSize: 8,
			mr1: newMetricsRequest(func() pmetric.Metrics {
				md := testdata.GenerateMetrics(4)
				extraScopeMetrics := md.ResourceMetrics().At(0).ScopeMetrics().AppendEmpty()
				testdata.GenerateMetrics(4).ResourceMetrics().At(0).ScopeMetrics().At(0).MoveTo(extraScopeMetrics)
				extraScopeMetrics.Scope().SetName("extra scope")
				return md
			}()),
			mr2: nil,
			expected: []Request{
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
				assert.Equal(t, expected.size(&s), actual.size(&s))
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
			for i := 0; i < numDataPoints; i++ {
				sum.DataPoints().AppendEmpty().SetIntValue(int64(i + 1))
			}
		case pmetric.MetricTypeGauge:
			gauge := m.SetEmptyGauge()
			for i := 0; i < numDataPoints; i++ {
				gauge.DataPoints().AppendEmpty().SetIntValue(int64(i + 1))
			}
		case pmetric.MetricTypeHistogram:
			hist := m.SetEmptyHistogram()
			for i := uint64(0); i < uint64(numDataPoints); i++ {
				hist.DataPoints().AppendEmpty().SetCount(i + 1)
			}
		case pmetric.MetricTypeExponentialHistogram:
			expHist := m.SetEmptyExponentialHistogram()
			for i := uint64(0); i < uint64(numDataPoints); i++ {
				expHist.DataPoints().AppendEmpty().SetCount(i + 1)
			}
		case pmetric.MetricTypeSummary:
			summary := m.SetEmptySummary()
			for i := uint64(0); i < uint64(numDataPoints); i++ {
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
			res, err := mr1.MergeSplit(context.Background(), 1, RequestSizerTypeItems, nil)
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
	_, err := r1.MergeSplit(context.Background(), 10, RequestSizerTypeItems, r2)
	require.Error(t, err)
	assert.Equal(t, 36, r1.ItemsCount())
}

func TestExtractMetrics(t *testing.T) {
	for i := 0; i < 20; i++ {
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
	merged := []Request{newMetricsRequest(testdata.GenerateMetrics(1))}
	for j := 0; j < 1000; j++ {
		lr2 := newMetricsRequest(testdata.GenerateMetrics(10))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 20000, RequestSizerTypeItems, lr2)
		merged = append(merged[0:len(merged)-1], res...)
	}
	assert.Len(t, merged, 2)
}

func BenchmarkSplittingBasedOnItemCountManySmallMetrics(b *testing.B) {
	// All requests merge into a single batch.
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newMetricsRequest(testdata.GenerateMetrics(10))}
		for j := 0; j < 1000; j++ {
			lr2 := newMetricsRequest(testdata.GenerateMetrics(10))
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 20020, RequestSizerTypeItems, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 1)
	}
}

func BenchmarkSplittingBasedOnItemCountManyMetricsSlightlyAboveLimit(b *testing.B) {
	// Every incoming request results in a split.
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newMetricsRequest(testdata.GenerateMetrics(0))}
		for j := 0; j < 10; j++ {
			lr2 := newMetricsRequest(testdata.GenerateMetrics(10001))
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 20000, RequestSizerTypeItems, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 11)
	}
}

func BenchmarkSplittingBasedOnItemCountHugeMetrics(b *testing.B) {
	// One request splits into many batches.
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newMetricsRequest(testdata.GenerateMetrics(0))}
		lr2 := newMetricsRequest(testdata.GenerateMetrics(100000))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 20000, RequestSizerTypeItems, lr2)
		merged = append(merged[0:len(merged)-1], res...)
		assert.Len(b, merged, 10)
	}
}

func TestMergeSplitMetricsBasedOnByteSize(t *testing.T) {
	tests := []struct {
		name             string
		szt              RequestSizerType
		maxSize          int
		mr1              Request
		mr2              Request
		expected         []Request
		expectSplitError bool
	}{
		{
			name:     "both_requests_empty",
			szt:      RequestSizerTypeBytes,
			maxSize:  metricsMarshaler.MetricsSize(testdata.GenerateMetrics(10)),
			mr1:      newMetricsRequest(pmetric.NewMetrics()),
			mr2:      newMetricsRequest(pmetric.NewMetrics()),
			expected: []Request{newMetricsRequest(pmetric.NewMetrics())},
		},
		{
			name:     "first_request_empty",
			szt:      RequestSizerTypeBytes,
			maxSize:  metricsMarshaler.MetricsSize(testdata.GenerateMetrics(10)),
			mr1:      newMetricsRequest(pmetric.NewMetrics()),
			mr2:      newMetricsRequest(testdata.GenerateMetrics(5)),
			expected: []Request{newMetricsRequest(testdata.GenerateMetrics(5))},
		},
		{
			name:     "first_empty_second_nil",
			szt:      RequestSizerTypeBytes,
			maxSize:  metricsMarshaler.MetricsSize(testdata.GenerateMetrics(10)),
			mr1:      newMetricsRequest(pmetric.NewMetrics()),
			mr2:      nil,
			expected: []Request{newMetricsRequest(pmetric.NewMetrics())},
		},
		{
			name:    "merge_only",
			szt:     RequestSizerTypeBytes,
			maxSize: metricsMarshaler.MetricsSize(testdata.GenerateMetrics(15)) - 1,
			mr1:     newMetricsRequest(testdata.GenerateMetrics(7)),
			mr2:     newMetricsRequest(testdata.GenerateMetrics(7)),
			expected: []Request{newMetricsRequest(func() pmetric.Metrics {
				md := testdata.GenerateMetrics(7)
				testdata.GenerateMetrics(7).ResourceMetrics().MoveAndAppendTo(md.ResourceMetrics())
				return md
			}())},
		},
		{
			name:    "split_only",
			szt:     RequestSizerTypeBytes,
			maxSize: metricsMarshaler.MetricsSize(testdata.GenerateMetrics(7)) + 1,
			mr1:     newMetricsRequest(pmetric.NewMetrics()),
			mr2:     newMetricsRequest(testdata.GenerateMetrics(17)),
			expected: []Request{
				newMetricsRequest(testdata.GenerateMetrics(7)),
				newMetricsRequest(testdata.GenerateMetrics(7)),
				newMetricsRequest(testdata.GenerateMetrics(3)),
			},
		},
		{
			name:    "merge_and_split",
			szt:     RequestSizerTypeBytes,
			maxSize: metricsMarshaler.MetricsSize(testdata.GenerateMetrics(7)) + 1,
			mr1:     newMetricsRequest(testdata.GenerateMetrics(14)),
			mr2:     newMetricsRequest(testdata.GenerateMetrics(11)),
			expected: []Request{
				newMetricsRequest(testdata.GenerateMetrics(7)),
				newMetricsRequest(testdata.GenerateMetrics(7)),
				newMetricsRequest(testdata.GenerateMetrics(7)),
				newMetricsRequest(testdata.GenerateMetrics(4)),
			},
		},
		{
			name:    "scope_metrics_split",
			szt:     RequestSizerTypeBytes,
			maxSize: metricsMarshaler.MetricsSize(testdata.GenerateMetrics(7)) + 1,
			mr1: newMetricsRequest(func() pmetric.Metrics {
				md := testdata.GenerateMetrics(7)
				extraScopeMetrics := md.ResourceMetrics().At(0).ScopeMetrics().AppendEmpty()
				testdata.GenerateMetrics(7).ResourceMetrics().At(0).ScopeMetrics().At(0).MoveTo(extraScopeMetrics)
				extraScopeMetrics.Scope().SetName("extra scope")
				return md
			}()),
			mr2: nil,
			expected: []Request{
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
			szt:     RequestSizerTypeBytes,
			maxSize: 10,
			mr1: newMetricsRequest(func() pmetric.Metrics {
				md := testdata.GenerateMetrics(1)
				md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).SetDescription(string(make([]byte, 100)))
				return md
			}()),
			mr2:              nil,
			expected:         []Request{},
			expectSplitError: true,
		},
		{
			name:    "splittable_then_unsplittable_metric",
			szt:     RequestSizerTypeBytes,
			maxSize: 1000,
			mr1: newMetricsRequest(func() pmetric.Metrics {
				md := testdata.GenerateMetrics(2)
				md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).SetDescription(string(make([]byte, 10)))
				md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(1).SetDescription(string(make([]byte, 1001)))
				return md
			}()),
			mr2: nil,
			expected: []Request{newMetricsRequest(func() pmetric.Metrics {
				md := testdata.GenerateMetrics(1)
				md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).SetDescription(string(make([]byte, 10)))
				return md
			}())},
			expectSplitError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.mr1.MergeSplit(context.Background(), tt.maxSize, tt.szt, tt.mr2)
			if tt.expectSplitError {
				require.ErrorContains(t, err, "one datapoint size is greater than max size, dropping items:")
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
			removedSize := extractGaugeDataPoints(gauge, destMetric, tt.capacity, sz)

			assert.Equal(t, tt.expectedPoints, destMetric.Gauge().DataPoints().Len())
			if tt.expectedPoints > 0 {
				assert.Equal(t, tt.expectedPoints, removedSize)
			}
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
			removedSize := extractSumDataPoints(sum, destMetric, tt.capacity, sz)

			assert.Equal(t, tt.expectedPoints, destMetric.Sum().DataPoints().Len())
			if tt.expectedPoints > 0 {
				assert.Equal(t, tt.expectedPoints, removedSize)
			}
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
				dp.SetCount(uint64(i)) //nolint:gosec // disable G115
			}

			sz := &mockMetricsSizer{dpSize: 1}

			destMetric := pmetric.NewMetric()
			removedSize := extractHistogramDataPoints(histogram, destMetric, tt.capacity, sz)

			assert.Equal(t, tt.expectedPoints, destMetric.Histogram().DataPoints().Len())
			if tt.expectedPoints > 0 {
				assert.Equal(t, tt.expectedPoints, removedSize)
			}
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
				dp.SetCount(uint64(i)) //nolint:gosec // disable G115
			}

			sz := &mockMetricsSizer{dpSize: 1}

			destMetric := pmetric.NewMetric()
			removedSize := extractExponentialHistogramDataPoints(expHistogram, destMetric, tt.capacity, sz)

			assert.Equal(t, tt.expectedPoints, destMetric.ExponentialHistogram().DataPoints().Len())
			if tt.expectedPoints > 0 {
				assert.Equal(t, tt.expectedPoints, removedSize)
			}
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
				dp.SetCount(uint64(i)) //nolint:gosec // disable G115
			}

			sz := &mockMetricsSizer{dpSize: 1}

			destMetric := pmetric.NewMetric()
			removedSize := extractSummaryDataPoints(summary, destMetric, tt.capacity, sz)

			assert.Equal(t, tt.expectedPoints, destMetric.Summary().DataPoints().Len())
			if tt.expectedPoints > 0 {
				assert.Equal(t, tt.expectedPoints, removedSize)
			}
		})
	}
}

func TestMetricsMergeSplitUnknownSizerType(t *testing.T) {
	req := newMetricsRequest(pmetric.NewMetrics())
	// Call MergeSplit with invalid sizer
	_, err := req.MergeSplit(context.Background(), 0, RequestSizerType{}, nil)
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
