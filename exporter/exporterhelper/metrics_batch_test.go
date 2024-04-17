// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMergeMetrics(t *testing.T) {
	mr1 := &metricsRequest{md: testdata.GenerateMetrics(2)}
	mr2 := &metricsRequest{md: testdata.GenerateMetrics(3)}
	res, err := mergeMetrics(context.Background(), mr1, mr2)
	assert.Nil(t, err)
	assert.Equal(t, 5, res.(*metricsRequest).md.MetricCount())
}

func TestMergeMetricsInvalidInput(t *testing.T) {
	mr1 := &tracesRequest{td: testdata.GenerateTraces(2)}
	mr2 := &metricsRequest{md: testdata.GenerateMetrics(3)}
	_, err := mergeMetrics(context.Background(), mr1, mr2)
	assert.Error(t, err)
}

func TestMergeSplitMetrics(t *testing.T) {
	tests := []struct {
		name     string
		cfg      exporterbatcher.MaxSizeConfig
		mr1      Request
		mr2      Request
		expected []*metricsRequest
	}{
		{
			name:     "both_requests_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			mr1:      &metricsRequest{md: pmetric.NewMetrics()},
			mr2:      &metricsRequest{md: pmetric.NewMetrics()},
			expected: []*metricsRequest{{md: pmetric.NewMetrics()}},
		},
		{
			name:     "both_requests_nil",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			mr1:      nil,
			mr2:      nil,
			expected: []*metricsRequest{},
		},
		{
			name:     "first_request_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			mr1:      &metricsRequest{md: pmetric.NewMetrics()},
			mr2:      &metricsRequest{md: testdata.GenerateMetrics(5)},
			expected: []*metricsRequest{{md: testdata.GenerateMetrics(5)}},
		},
		{
			name:     "first_requests_nil",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			mr1:      nil,
			mr2:      &metricsRequest{md: testdata.GenerateMetrics(5)},
			expected: []*metricsRequest{{md: testdata.GenerateMetrics(5)}},
		},
		{
			name:     "first_nil_second_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			mr1:      nil,
			mr2:      &metricsRequest{md: pmetric.NewMetrics()},
			expected: []*metricsRequest{{md: pmetric.NewMetrics()}},
		},
		{
			name: "merge_only",
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 60},
			mr1:  &metricsRequest{md: testdata.GenerateMetrics(10)},
			mr2:  &metricsRequest{md: testdata.GenerateMetrics(14)},
			expected: []*metricsRequest{{md: func() pmetric.Metrics {
				metrics := testdata.GenerateMetrics(10)
				testdata.GenerateMetrics(14).ResourceMetrics().MoveAndAppendTo(metrics.ResourceMetrics())
				return metrics
			}()}},
		},
		{
			name: "split_only",
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 14},
			mr1:  nil,
			mr2:  &metricsRequest{md: testdata.GenerateMetrics(15)}, // 15 metrics, 30 data points
			expected: []*metricsRequest{
				{md: testdata.GenerateMetrics(7)}, // 7 metrics, 14 data points
				{md: testdata.GenerateMetrics(7)}, // 7 metrics, 14 data points
				{md: testdata.GenerateMetrics(1)}, // 1 metric, 2 data points
			},
		},
		{
			name: "split_and_merge",
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 28},
			mr1:  &metricsRequest{md: testdata.GenerateMetrics(7)},  // 7 metrics, 14 data points
			mr2:  &metricsRequest{md: testdata.GenerateMetrics(25)}, // 25 metrics, 50 data points
			expected: []*metricsRequest{
				{md: func() pmetric.Metrics {
					metrics := testdata.GenerateMetrics(7)
					testdata.GenerateMetrics(7).ResourceMetrics().MoveAndAppendTo(metrics.ResourceMetrics())
					return metrics
				}()},
				{md: testdata.GenerateMetrics(14)}, // 14 metrics, 28 data points
				{md: testdata.GenerateMetrics(4)},  // 4 metrics, 8 data points
			},
		},
		{
			name: "scope_metrics_split",
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 8},
			mr1: &metricsRequest{md: func() pmetric.Metrics {
				md := testdata.GenerateMetrics(4)
				extraScopeMetrics := md.ResourceMetrics().At(0).ScopeMetrics().AppendEmpty()
				testdata.GenerateMetrics(4).ResourceMetrics().At(0).ScopeMetrics().At(0).MoveTo(extraScopeMetrics)
				extraScopeMetrics.Scope().SetName("extra scope")
				return md
			}()},
			mr2: nil,
			expected: []*metricsRequest{
				{md: testdata.GenerateMetrics(4)},
				{md: func() pmetric.Metrics {
					md := testdata.GenerateMetrics(4)
					md.ResourceMetrics().At(0).ScopeMetrics().At(0).Scope().SetName("extra scope")
					return md
				}()},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := mergeSplitMetrics(context.Background(), tt.cfg, tt.mr1, tt.mr2)
			assert.Nil(t, err)
			assert.Equal(t, len(tt.expected), len(res))
			for i := range res {
				assert.Equal(t, tt.expected[i], res[i].(*metricsRequest))
			}
		})
	}
}

func TestMergeSplitMetricsInvalidInput(t *testing.T) {
	r1 := &tracesRequest{td: testdata.GenerateTraces(2)}
	r2 := &metricsRequest{md: testdata.GenerateMetrics(3)}
	_, err := mergeSplitMetrics(context.Background(), exporterbatcher.MaxSizeConfig{MaxSizeItems: 10}, r1, r2)
	assert.Error(t, err)
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
