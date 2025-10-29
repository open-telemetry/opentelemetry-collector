// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric_test

import (
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func ExampleNewMetrics() {
	metrics := pmetric.NewMetrics()

	resourceMetrics := metrics.ResourceMetrics().AppendEmpty()

	resourceMetrics.Resource().Attributes().PutStr("service.name", "my-service")
	resourceMetrics.Resource().Attributes().PutStr("host.name", "server-01")

	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
	scopeMetrics.Scope().SetName("my-meter")
	scopeMetrics.Scope().SetVersion("1.0.0")

	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName("requests_total")
	metric.SetDescription("Total number of requests")
	metric.SetUnit("1")

	sum := metric.SetEmptySum()
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	sum.SetIsMonotonic(true)

	dataPoint := sum.DataPoints().AppendEmpty()
	dataPoint.SetStartTimestamp(pcommon.Timestamp(1640991600000000000)) // 2022-01-01 00:00:00 UTC
	dataPoint.SetTimestamp(pcommon.Timestamp(1640995200000000000))      // 2022-01-01 01:00:00 UTC
	dataPoint.SetIntValue(1234)
	dataPoint.Attributes().PutStr("endpoint", "/api/v1/users")
	dataPoint.Attributes().PutStr("method", "GET")

	fmt.Printf("Resource metrics count: %d\n", metrics.ResourceMetrics().Len())
	fmt.Printf("Metrics count: %d\n", scopeMetrics.Metrics().Len())
	fmt.Printf("Metric name: %s\n", metric.Name())
	fmt.Printf("Data points count: %d\n", sum.DataPoints().Len())
	// Output:
	// Resource metrics count: 1
	// Metrics count: 1
	// Metric name: requests_total
	// Data points count: 1
}

func ExampleMetric_SetEmptyGauge() {
	metrics := pmetric.NewMetrics()
	resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()

	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName("cpu_usage_percent")
	metric.SetDescription("Current CPU usage percentage")
	metric.SetUnit("%")

	gauge := metric.SetEmptyGauge()

	for i := range 4 {
		dataPoint := gauge.DataPoints().AppendEmpty()
		dataPoint.SetTimestamp(pcommon.Timestamp(1640995200000000000))
		dataPoint.SetDoubleValue(45.5 + float64(i)*2.1)
		dataPoint.Attributes().PutInt("cpu", int64(i))
	}

	fmt.Printf("Metric type: %s\n", metric.Type())
	fmt.Printf("Data points count: %d\n", gauge.DataPoints().Len())
	fmt.Printf("First CPU usage: %.1f%%\n", gauge.DataPoints().At(0).DoubleValue())
	// Output:
	// Metric type: Gauge
	// Data points count: 4
	// First CPU usage: 45.5%
}

func ExampleMetric_SetEmptyHistogram() {
	metrics := pmetric.NewMetrics()
	resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()

	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName("request_duration_seconds")
	metric.SetDescription("Request duration in seconds")
	metric.SetUnit("s")

	histogram := metric.SetEmptyHistogram()
	histogram.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

	dataPoint := histogram.DataPoints().AppendEmpty()
	dataPoint.SetStartTimestamp(pcommon.Timestamp(1640995140000000000)) // 2022-01-01 00:59:00 UTC
	dataPoint.SetTimestamp(pcommon.Timestamp(1640995200000000000))      // 2022-01-01 01:00:00 UTC
	dataPoint.SetCount(100)
	dataPoint.SetSum(23.5)
	dataPoint.SetMin(0.001)
	dataPoint.SetMax(5.0)

	dataPoint.ExplicitBounds().FromRaw([]float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0})
	dataPoint.BucketCounts().FromRaw([]uint64{5, 10, 20, 25, 20, 15, 3, 2, 0, 0})

	dataPoint.Attributes().PutStr("endpoint", "/api/health")
	dataPoint.SetFlags(pmetric.DefaultDataPointFlags)

	fmt.Printf("Metric type: %s\n", metric.Type())
	fmt.Printf("Total count: %d\n", dataPoint.Count())
	fmt.Printf("Total sum: %.1f\n", dataPoint.Sum())
	fmt.Printf("Min value: %.3f\n", dataPoint.Min())
	fmt.Printf("Max value: %.1f\n", dataPoint.Max())
	fmt.Printf("Bucket count: %d\n", dataPoint.BucketCounts().Len())
	// Output:
	// Metric type: Histogram
	// Total count: 100
	// Total sum: 23.5
	// Min value: 0.001
	// Max value: 5.0
	// Bucket count: 10
}

func ExampleMetric_SetEmptyExponentialHistogram() {
	metrics := pmetric.NewMetrics()
	resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()

	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName("response_size_bytes")
	metric.SetDescription("Response size in bytes")
	metric.SetUnit("By")

	expHist := metric.SetEmptyExponentialHistogram()
	expHist.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)

	dataPoint := expHist.DataPoints().AppendEmpty()
	dataPoint.SetStartTimestamp(pcommon.Timestamp(1640995140000000000))
	dataPoint.SetTimestamp(pcommon.Timestamp(1640995200000000000))
	dataPoint.SetCount(50)
	dataPoint.SetSum(1024.5)
	dataPoint.SetScale(4)
	dataPoint.SetZeroCount(5)
	dataPoint.SetZeroThreshold(0.001)

	// Set positive buckets
	positive := dataPoint.Positive()
	positive.SetOffset(2)
	positive.BucketCounts().FromRaw([]uint64{2, 3, 7, 8, 5})

	// Set negative buckets
	negative := dataPoint.Negative()
	negative.SetOffset(1)
	negative.BucketCounts().FromRaw([]uint64{1, 2, 3})

	dataPoint.Attributes().PutStr("protocol", "http")

	fmt.Printf("Metric type: %s\n", metric.Type())
	fmt.Printf("Scale: %d\n", dataPoint.Scale())
	fmt.Printf("Zero count: %d\n", dataPoint.ZeroCount())
	fmt.Printf("Positive buckets: %d\n", positive.BucketCounts().Len())
	fmt.Printf("Negative buckets: %d\n", negative.BucketCounts().Len())
	// Output:
	// Metric type: ExponentialHistogram
	// Scale: 4
	// Zero count: 5
	// Positive buckets: 5
	// Negative buckets: 3
}

func ExampleMetric_SetEmptySummary() {
	metrics := pmetric.NewMetrics()
	resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()

	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName("latency_seconds")
	metric.SetDescription("Request latency in seconds")
	metric.SetUnit("s")

	summary := metric.SetEmptySummary()

	dataPoint := summary.DataPoints().AppendEmpty()
	dataPoint.SetStartTimestamp(pcommon.Timestamp(1640995140000000000))
	dataPoint.SetTimestamp(pcommon.Timestamp(1640995200000000000))
	dataPoint.SetCount(1000)
	dataPoint.SetSum(125.5)

	// Add quantile values
	q50 := dataPoint.QuantileValues().AppendEmpty()
	q50.SetQuantile(0.5)
	q50.SetValue(0.1)

	q95 := dataPoint.QuantileValues().AppendEmpty()
	q95.SetQuantile(0.95)
	q95.SetValue(0.5)

	q99 := dataPoint.QuantileValues().AppendEmpty()
	q99.SetQuantile(0.99)
	q99.SetValue(1.0)

	dataPoint.Attributes().PutStr("service", "api")

	fmt.Printf("Metric type: %s\n", metric.Type())
	fmt.Printf("Count: %d\n", dataPoint.Count())
	fmt.Printf("Sum: %.1f\n", dataPoint.Sum())
	fmt.Printf("Quantiles: %d\n", dataPoint.QuantileValues().Len())
	fmt.Printf("P95: %.1f\n", q95.Value())
	// Output:
	// Metric type: Summary
	// Count: 1000
	// Sum: 125.5
	// Quantiles: 3
	// P95: 0.5
}

func ExampleNumberDataPoint_Exemplars() {
	metrics := pmetric.NewMetrics()
	resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()

	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName("http_requests_total")

	sum := metric.SetEmptySum()
	dataPoint := sum.DataPoints().AppendEmpty()
	dataPoint.SetIntValue(5000)

	exemplar := dataPoint.Exemplars().AppendEmpty()
	exemplar.SetTimestamp(pcommon.Timestamp(1640995200000000000))
	exemplar.SetIntValue(1)
	exemplar.FilteredAttributes().PutStr("trace_id", "abc123")
	exemplar.FilteredAttributes().PutStr("span_id", "def456")

	fmt.Printf("Data point value: %d\n", dataPoint.IntValue())
	fmt.Printf("Exemplars count: %d\n", dataPoint.Exemplars().Len())
	fmt.Printf("Exemplar value: %d\n", exemplar.IntValue())
	// Output:
	// Data point value: 5000
	// Exemplars count: 1
	// Exemplar value: 1
}

func ExampleDataPointFlags() {
	metrics := pmetric.NewMetrics()
	resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()

	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName("test_metric")

	gauge := metric.SetEmptyGauge()
	dataPoint := gauge.DataPoints().AppendEmpty()

	// Test default flags
	defaultFlags := pmetric.DefaultDataPointFlags
	dataPoint.SetFlags(defaultFlags)
	fmt.Printf("Default flags NoRecordedValue: %t\n", dataPoint.Flags().NoRecordedValue())

	// Test with NoRecordedValue flag
	flagsWithNoValue := defaultFlags.WithNoRecordedValue(true)
	dataPoint.SetFlags(flagsWithNoValue)
	fmt.Printf("With NoRecordedValue flag: %t\n", dataPoint.Flags().NoRecordedValue())

	// Test removing NoRecordedValue flag
	flagsWithoutNoValue := flagsWithNoValue.WithNoRecordedValue(false)
	dataPoint.SetFlags(flagsWithoutNoValue)
	fmt.Printf("Without NoRecordedValue flag: %t\n", dataPoint.Flags().NoRecordedValue())

	// Output:
	// Default flags NoRecordedValue: false
	// With NoRecordedValue flag: true
	// Without NoRecordedValue flag: false
}
