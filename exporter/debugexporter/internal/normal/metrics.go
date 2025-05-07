// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package normal // import "go.opentelemetry.io/collector/exporter/debugexporter/internal/normal"

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"go.opentelemetry.io/collector/pdata/pmetric"
)

type normalMetricsMarshaler struct{}

// Ensure normalMetricsMarshaller implements interface pmetric.Marshaler
var _ pmetric.Marshaler = normalMetricsMarshaler{}

// NewNormalMetricsMarshaler returns a pmetric.Marshaler for normal verbosity. It writes one line of text per log record
func NewNormalMetricsMarshaler() pmetric.Marshaler {
	return normalMetricsMarshaler{}
}

func (normalMetricsMarshaler) MarshalMetrics(md pmetric.Metrics) ([]byte, error) {
	var buffer bytes.Buffer
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		resourceMetrics := md.ResourceMetrics().At(i)
		for j := 0; j < resourceMetrics.ScopeMetrics().Len(); j++ {
			scopeMetrics := resourceMetrics.ScopeMetrics().At(j)
			for k := 0; k < scopeMetrics.Metrics().Len(); k++ {
				metric := scopeMetrics.Metrics().At(k)

				var dataPointLines []string
				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					dataPointLines = writeNumberDataPoints(metric, metric.Gauge().DataPoints())
				case pmetric.MetricTypeSum:
					dataPointLines = writeNumberDataPoints(metric, metric.Sum().DataPoints())
				case pmetric.MetricTypeHistogram:
					dataPointLines = writeHistogramDataPoints(metric)
				case pmetric.MetricTypeExponentialHistogram:
					dataPointLines = writeExponentialHistogramDataPoints(metric)
				case pmetric.MetricTypeSummary:
					dataPointLines = writeSummaryDataPoints(metric)
				}
				for _, line := range dataPointLines {
					buffer.WriteString(line)
				}
			}
		}
	}
	return buffer.Bytes(), nil
}

func writeNumberDataPoints(metric pmetric.Metric, dataPoints pmetric.NumberDataPointSlice) (lines []string) {
	for i := 0; i < dataPoints.Len(); i++ {
		dataPoint := dataPoints.At(i)
		dataPointAttributes := writeAttributes(dataPoint.Attributes())

		var value string
		switch dataPoint.ValueType() {
		case pmetric.NumberDataPointValueTypeInt:
			value = strconv.FormatInt(dataPoint.IntValue(), 10)
		case pmetric.NumberDataPointValueTypeDouble:
			value = fmt.Sprintf("%v", dataPoint.DoubleValue())
		}

		dataPointLine := fmt.Sprintf("%s{%s} %s\n", metric.Name(), strings.Join(dataPointAttributes, ","), value)
		lines = append(lines, dataPointLine)
	}
	return lines
}

func writeHistogramDataPoints(metric pmetric.Metric) (lines []string) {
	for i := 0; i < metric.Histogram().DataPoints().Len(); i++ {
		dataPoint := metric.Histogram().DataPoints().At(i)
		dataPointAttributes := writeAttributes(dataPoint.Attributes())

		var value string
		value = fmt.Sprintf("count=%d", dataPoint.Count())
		if dataPoint.HasSum() {
			value += fmt.Sprintf(" sum=%v", dataPoint.Sum())
		}
		if dataPoint.HasMin() {
			value += fmt.Sprintf(" min=%v", dataPoint.Min())
		}
		if dataPoint.HasMax() {
			value += fmt.Sprintf(" max=%v", dataPoint.Max())
		}

		for bucketIndex := 0; bucketIndex < dataPoint.BucketCounts().Len(); bucketIndex++ {
			bucketBound := ""
			if bucketIndex < dataPoint.ExplicitBounds().Len() {
				bucketBound = fmt.Sprintf("le%v=", dataPoint.ExplicitBounds().At(bucketIndex))
			}
			bucketCount := dataPoint.BucketCounts().At(bucketIndex)
			value += fmt.Sprintf(" %s%d", bucketBound, bucketCount)
		}

		dataPointLine := fmt.Sprintf("%s{%s} %s\n", metric.Name(), strings.Join(dataPointAttributes, ","), value)
		lines = append(lines, dataPointLine)
	}
	return lines
}

func writeExponentialHistogramDataPoints(metric pmetric.Metric) (lines []string) {
	for i := 0; i < metric.ExponentialHistogram().DataPoints().Len(); i++ {
		dataPoint := metric.ExponentialHistogram().DataPoints().At(i)
		dataPointAttributes := writeAttributes(dataPoint.Attributes())

		var value string
		value = fmt.Sprintf("count=%d", dataPoint.Count())
		if dataPoint.HasSum() {
			value += fmt.Sprintf(" sum=%v", dataPoint.Sum())
		}
		if dataPoint.HasMin() {
			value += fmt.Sprintf(" min=%v", dataPoint.Min())
		}
		if dataPoint.HasMax() {
			value += fmt.Sprintf(" max=%v", dataPoint.Max())
		}

		// TODO display buckets

		dataPointLine := fmt.Sprintf("%s{%s} %s\n", metric.Name(), strings.Join(dataPointAttributes, ","), value)
		lines = append(lines, dataPointLine)
	}
	return lines
}

func writeSummaryDataPoints(metric pmetric.Metric) (lines []string) {
	for i := 0; i < metric.Summary().DataPoints().Len(); i++ {
		dataPoint := metric.Summary().DataPoints().At(i)
		dataPointAttributes := writeAttributes(dataPoint.Attributes())

		var value string
		value = fmt.Sprintf("count=%d", dataPoint.Count())
		value += fmt.Sprintf(" sum=%f", dataPoint.Sum())

		for quantileIndex := 0; quantileIndex < dataPoint.QuantileValues().Len(); quantileIndex++ {
			quantile := dataPoint.QuantileValues().At(quantileIndex)
			value += fmt.Sprintf(" q%v=%v", quantile.Quantile(), quantile.Value())
		}

		dataPointLine := fmt.Sprintf("%s{%s} %s\n", metric.Name(), strings.Join(dataPointAttributes, ","), value)
		lines = append(lines, dataPointLine)
	}
	return lines
}
