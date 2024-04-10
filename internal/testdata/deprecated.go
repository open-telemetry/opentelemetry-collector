// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testdata

import (
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	ptestdata "go.opentelemetry.io/collector/pdata/testdata"
)

const (
	TestGaugeDoubleMetricName          = ptestdata.TestGaugeDoubleMetricName
	TestGaugeIntMetricName             = ptestdata.TestGaugeIntMetricName
	TestSumDoubleMetricName            = ptestdata.TestSumDoubleMetricName
	TestSumIntMetricName               = ptestdata.TestSumIntMetricName
	TestHistogramMetricName            = ptestdata.TestHistogramMetricName
	TestExponentialHistogramMetricName = ptestdata.TestExponentialHistogramMetricName
	TestSummaryMetricName              = ptestdata.TestSummaryMetricName
)

// Deprecated: [v0.98.0] use pdata/testdata.GenerateMetricsAllTypesEmpty instead.
func GenerateMetricsAllTypesEmpty() pmetric.Metrics {
	return ptestdata.GenerateMetricsAllTypesEmpty()
}

// Deprecated: [v0.98.0] use pdata/testdata.GenerateMetricsMetricTypeInvalid instead.
func GenerateMetricsMetricTypeInvalid() pmetric.Metrics {
	return ptestdata.GenerateMetricsMetricTypeInvalid()
}

// Deprecated: [v0.98.0] use pdata/testdata.GenerateMetricsAllTypes instead.
func GenerateMetricsAllTypes() pmetric.Metrics {
	return ptestdata.GenerateMetricsAllTypes()
}

// Deprecated: [v0.98.0] use pdata/testdata.GenerateMetrics instead.
func GenerateMetrics(count int) pmetric.Metrics {
	return ptestdata.GenerateMetrics(count)
}

// Deprecated: [v0.98.0] use pdata/testdata.GenerateLogs instead.
func GenerateLogs(count int) plog.Logs {
	return ptestdata.GenerateLogs(count)
}

// Deprecated: [v0.98.0] use pdata/testdata.GenerateTraces instead.
func GenerateTraces(spanCount int) ptrace.Traces {
	return ptestdata.GenerateTraces(count)
}
