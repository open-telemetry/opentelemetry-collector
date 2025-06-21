// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric

import (
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func ExampleNewMetrics() {
	metrics := NewMetrics()

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
	sum.SetAggregationTemporality(AggregationTemporalityCumulative)
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
	metrics := NewMetrics()
	resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()

	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName("cpu_usage_percent")
	metric.SetDescription("Current CPU usage percentage")
	metric.SetUnit("%")

	gauge := metric.SetEmptyGauge()

	for i := 0; i < 4; i++ {
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
	metrics := NewMetrics()
	resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()

	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName("request_duration_seconds")
	metric.SetDescription("Request duration in seconds")
	metric.SetUnit("s")

	histogram := metric.SetEmptyHistogram()
	histogram.SetAggregationTemporality(AggregationTemporalityCumulative)

	dataPoint := histogram.DataPoints().AppendEmpty()
	dataPoint.SetStartTimestamp(pcommon.Timestamp(1640995140000000000)) // 2022-01-01 00:59:00 UTC
	dataPoint.SetTimestamp(pcommon.Timestamp(1640995200000000000))      // 2022-01-01 01:00:00 UTC
	dataPoint.SetCount(100)
	dataPoint.SetSum(23.5)

	dataPoint.ExplicitBounds().FromRaw([]float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0})

	dataPoint.BucketCounts().FromRaw([]uint64{5, 10, 20, 25, 20, 15, 3, 2, 0, 0})

	dataPoint.Attributes().PutStr("endpoint", "/api/health")

	fmt.Printf("Metric type: %s\n", metric.Type())
	fmt.Printf("Total count: %d\n", dataPoint.Count())
	fmt.Printf("Total sum: %.1f\n", dataPoint.Sum())
	fmt.Printf("Bucket count: %d\n", dataPoint.BucketCounts().Len())
	// Output:
	// Metric type: Histogram
	// Total count: 100
	// Total sum: 23.5
	// Bucket count: 10
}

func ExampleNumberDataPoint_Exemplars() {
	metrics := NewMetrics()
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
