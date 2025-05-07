// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestSplitMetrics_noop(t *testing.T) {
	td := testdata.GenerateMetrics(20)
	splitSize := 40
	split := splitMetrics(splitSize, td)
	assert.Equal(t, td, split)

	i := 0
	td.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().RemoveIf(func(pmetric.Metric) bool {
		i++
		return i > 5
	})
	assert.Equal(t, td, split)
}

func TestSplitMetrics(t *testing.T) {
	md := testdata.GenerateMetrics(20)
	metrics := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	dataPointCount := metricDPC(metrics.At(0))
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(0, i))
		assert.Equal(t, dataPointCount, metricDPC(metrics.At(i)))
	}
	cp := pmetric.NewMetrics()
	cpMetrics := cp.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics()
	cpMetrics.EnsureCapacity(5)
	md.ResourceMetrics().At(0).ScopeMetrics().At(0).Scope().CopyTo(
		cp.ResourceMetrics().At(0).ScopeMetrics().At(0).Scope())
	md.ResourceMetrics().At(0).Resource().CopyTo(
		cp.ResourceMetrics().At(0).Resource())
	metrics.At(0).CopyTo(cpMetrics.AppendEmpty())
	metrics.At(1).CopyTo(cpMetrics.AppendEmpty())
	metrics.At(2).CopyTo(cpMetrics.AppendEmpty())
	metrics.At(3).CopyTo(cpMetrics.AppendEmpty())
	metrics.At(4).CopyTo(cpMetrics.AppendEmpty())

	splitMetricCount := 5
	splitSize := splitMetricCount * dataPointCount
	split := splitMetrics(splitSize, md)
	assert.Equal(t, splitMetricCount, split.MetricCount())
	assert.Equal(t, cp, split)
	assert.Equal(t, 15, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-0", split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-0-4", split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(4).Name())

	split = splitMetrics(splitSize, md)
	assert.Equal(t, 10, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-5", split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-0-9", split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(4).Name())

	split = splitMetrics(splitSize, md)
	assert.Equal(t, 5, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-10", split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-0-14", split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(4).Name())

	split = splitMetrics(splitSize, md)
	assert.Equal(t, 5, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-15", split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-0-19", split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(4).Name())
}

func TestSplitMetricsMultipleResourceSpans(t *testing.T) {
	md := testdata.GenerateMetrics(20)
	metrics := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	dataPointCount := metricDPC(metrics.At(0))
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(0, i))
		assert.Equal(t, dataPointCount, metricDPC(metrics.At(i)))
	}
	// add second index to resource metrics
	testdata.GenerateMetrics(20).
		ResourceMetrics().At(0).CopyTo(md.ResourceMetrics().AppendEmpty())
	metrics = md.ResourceMetrics().At(1).ScopeMetrics().At(0).Metrics()
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(1, i))
	}

	splitMetricCount := 5
	splitSize := splitMetricCount * dataPointCount
	split := splitMetrics(splitSize, md)
	assert.Equal(t, splitMetricCount, split.MetricCount())
	assert.Equal(t, 35, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-0", split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-0-4", split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(4).Name())
}

func TestSplitMetricsMultipleResourceSpans_SplitSizeGreaterThanMetricSize(t *testing.T) {
	td := testdata.GenerateMetrics(20)
	metrics := td.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	dataPointCount := metricDPC(metrics.At(0))
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(0, i))
		assert.Equal(t, dataPointCount, metricDPC(metrics.At(i)))
	}
	// add second index to resource metrics
	testdata.GenerateMetrics(20).
		ResourceMetrics().At(0).CopyTo(td.ResourceMetrics().AppendEmpty())
	metrics = td.ResourceMetrics().At(1).ScopeMetrics().At(0).Metrics()
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(1, i))
	}

	splitMetricCount := 25
	splitSize := splitMetricCount * dataPointCount
	split := splitMetrics(splitSize, td)
	assert.Equal(t, splitMetricCount, split.MetricCount())
	assert.Equal(t, 40-splitMetricCount, td.MetricCount())
	assert.Equal(t, 1, td.ResourceMetrics().Len())
	assert.Equal(t, "test-metric-int-0-0", split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-0-19", split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(19).Name())
	assert.Equal(t, "test-metric-int-1-0", split.ResourceMetrics().At(1).ScopeMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-1-4", split.ResourceMetrics().At(1).ScopeMetrics().At(0).Metrics().At(4).Name())
}

func TestSplitMetricsUneven(t *testing.T) {
	md := testdata.GenerateMetrics(10)
	metrics := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	dataPointCount := 2
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(0, i))
		assert.Equal(t, dataPointCount, metricDPC(metrics.At(i)))
	}

	splitSize := 9
	split := splitMetrics(splitSize, md)
	assert.Equal(t, 5, split.MetricCount())
	assert.Equal(t, 6, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-0", split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-0-4", split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(4).Name())

	split = splitMetrics(splitSize, md)
	assert.Equal(t, 5, split.MetricCount())
	assert.Equal(t, 1, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-4", split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-0-8", split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(4).Name())

	split = splitMetrics(splitSize, md)
	assert.Equal(t, 1, split.MetricCount())
	assert.Equal(t, "test-metric-int-0-9", split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Name())
}

func TestSplitMetricsAllTypes(t *testing.T) {
	md := testdata.GenerateMetricsAllTypes()
	dataPointCount := 2
	metrics := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(0, i))
		assert.Equal(t, dataPointCount, metricDPC(metrics.At(i)))
	}

	splitSize := 2
	// Start with 7 metric types, and 2 points per-metric. Split out the first,
	// and then split by 2 for the rest so that each metric is split in half.
	// Verify that descriptors are preserved for all data types across splits.

	split := splitMetrics(1, md)
	assert.Equal(t, 1, split.MetricCount())
	assert.Equal(t, 7, md.MetricCount())
	gaugeInt := split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	assert.Equal(t, 1, gaugeInt.Gauge().DataPoints().Len())
	assert.Equal(t, "test-metric-int-0-0", gaugeInt.Name())

	split = splitMetrics(splitSize, md)
	assert.Equal(t, 2, split.MetricCount())
	assert.Equal(t, 6, md.MetricCount())
	gaugeInt = split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	gaugeDouble := split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(1)
	assert.Equal(t, 1, gaugeInt.Gauge().DataPoints().Len())
	assert.Equal(t, "test-metric-int-0-0", gaugeInt.Name())
	assert.Equal(t, 1, gaugeDouble.Gauge().DataPoints().Len())
	assert.Equal(t, "test-metric-int-0-1", gaugeDouble.Name())

	split = splitMetrics(splitSize, md)
	assert.Equal(t, 2, split.MetricCount())
	assert.Equal(t, 5, md.MetricCount())
	gaugeDouble = split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	sumInt := split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(1)
	assert.Equal(t, 1, gaugeDouble.Gauge().DataPoints().Len())
	assert.Equal(t, "test-metric-int-0-1", gaugeDouble.Name())
	assert.Equal(t, 1, sumInt.Sum().DataPoints().Len())
	assert.Equal(t, pmetric.AggregationTemporalityCumulative, sumInt.Sum().AggregationTemporality())
	assert.True(t, sumInt.Sum().IsMonotonic())
	assert.Equal(t, "test-metric-int-0-2", sumInt.Name())

	split = splitMetrics(splitSize, md)
	assert.Equal(t, 2, split.MetricCount())
	assert.Equal(t, 4, md.MetricCount())
	sumInt = split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	sumDouble := split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(1)
	assert.Equal(t, 1, sumInt.Sum().DataPoints().Len())
	assert.Equal(t, pmetric.AggregationTemporalityCumulative, sumInt.Sum().AggregationTemporality())
	assert.True(t, sumInt.Sum().IsMonotonic())
	assert.Equal(t, "test-metric-int-0-2", sumInt.Name())
	assert.Equal(t, 1, sumDouble.Sum().DataPoints().Len())
	assert.Equal(t, pmetric.AggregationTemporalityCumulative, sumDouble.Sum().AggregationTemporality())
	assert.True(t, sumDouble.Sum().IsMonotonic())
	assert.Equal(t, "test-metric-int-0-3", sumDouble.Name())

	split = splitMetrics(splitSize, md)
	assert.Equal(t, 2, split.MetricCount())
	assert.Equal(t, 3, md.MetricCount())
	sumDouble = split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	histogram := split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(1)
	assert.Equal(t, 1, sumDouble.Sum().DataPoints().Len())
	assert.Equal(t, pmetric.AggregationTemporalityCumulative, sumDouble.Sum().AggregationTemporality())
	assert.True(t, sumDouble.Sum().IsMonotonic())
	assert.Equal(t, "test-metric-int-0-3", sumDouble.Name())
	assert.Equal(t, 1, histogram.Histogram().DataPoints().Len())
	assert.Equal(t, pmetric.AggregationTemporalityCumulative, histogram.Histogram().AggregationTemporality())
	assert.Equal(t, "test-metric-int-0-4", histogram.Name())

	split = splitMetrics(splitSize, md)
	assert.Equal(t, 2, split.MetricCount())
	assert.Equal(t, 2, md.MetricCount())
	histogram = split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	exponentialHistogram := split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(1)
	assert.Equal(t, 1, histogram.Histogram().DataPoints().Len())
	assert.Equal(t, pmetric.AggregationTemporalityCumulative, histogram.Histogram().AggregationTemporality())
	assert.Equal(t, "test-metric-int-0-4", histogram.Name())
	assert.Equal(t, 1, exponentialHistogram.ExponentialHistogram().DataPoints().Len())
	assert.Equal(t, pmetric.AggregationTemporalityDelta, exponentialHistogram.ExponentialHistogram().AggregationTemporality())
	assert.Equal(t, "test-metric-int-0-5", exponentialHistogram.Name())

	split = splitMetrics(splitSize, md)
	assert.Equal(t, 2, split.MetricCount())
	assert.Equal(t, 1, md.MetricCount())
	exponentialHistogram = split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	summary := split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(1)
	assert.Equal(t, 1, exponentialHistogram.ExponentialHistogram().DataPoints().Len())
	assert.Equal(t, pmetric.AggregationTemporalityDelta, exponentialHistogram.ExponentialHistogram().AggregationTemporality())
	assert.Equal(t, "test-metric-int-0-5", exponentialHistogram.Name())
	assert.Equal(t, 1, summary.Summary().DataPoints().Len())
	assert.Equal(t, "test-metric-int-0-6", summary.Name())

	split = splitMetrics(splitSize, md)
	summary = split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	assert.Equal(t, 1, summary.Summary().DataPoints().Len())
	assert.Equal(t, "test-metric-int-0-6", summary.Name())
}

func TestSplitMetricsBatchSizeSmallerThanDataPointCount(t *testing.T) {
	md := testdata.GenerateMetrics(2)
	metrics := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	dataPointCount := 2
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(0, i))
		assert.Equal(t, dataPointCount, metricDPC(metrics.At(i)))
	}

	splitSize := 1
	split := splitMetrics(splitSize, md)
	splitMetric := split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	assert.Equal(t, 1, split.MetricCount())
	assert.Equal(t, 2, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-0", splitMetric.Name())

	split = splitMetrics(splitSize, md)
	splitMetric = split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	assert.Equal(t, 1, split.MetricCount())
	assert.Equal(t, 1, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-0", splitMetric.Name())

	split = splitMetrics(splitSize, md)
	splitMetric = split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	assert.Equal(t, 1, split.MetricCount())
	assert.Equal(t, 1, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-1", splitMetric.Name())

	split = splitMetrics(splitSize, md)
	splitMetric = split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	assert.Equal(t, 1, split.MetricCount())
	assert.Equal(t, 1, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-1", splitMetric.Name())
}

func TestSplitMetricsMultipleILM(t *testing.T) {
	md := testdata.GenerateMetrics(20)
	metrics := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	dataPointCount := metricDPC(metrics.At(0))
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(0, i))
		assert.Equal(t, dataPointCount, metricDPC(metrics.At(i)))
	}
	// add second index to ilm
	md.ResourceMetrics().At(0).ScopeMetrics().At(0).
		CopyTo(md.ResourceMetrics().At(0).ScopeMetrics().AppendEmpty())

	// add a third index to ilm
	md.ResourceMetrics().At(0).ScopeMetrics().At(0).
		CopyTo(md.ResourceMetrics().At(0).ScopeMetrics().AppendEmpty())
	metrics = md.ResourceMetrics().At(0).ScopeMetrics().At(2).Metrics()
	for i := 0; i < metrics.Len(); i++ {
		metrics.At(i).SetName(getTestMetricName(2, i))
	}

	splitMetricCount := 40
	splitSize := splitMetricCount * dataPointCount
	split := splitMetrics(splitSize, md)
	assert.Equal(t, splitMetricCount, split.MetricCount())
	assert.Equal(t, 20, md.MetricCount())
	assert.Equal(t, "test-metric-int-0-0", split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Name())
	assert.Equal(t, "test-metric-int-0-4", split.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(4).Name())
}
