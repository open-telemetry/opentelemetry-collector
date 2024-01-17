// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package normal

import (
	"bytes"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
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
		resourceAttributeStrings := writeAttributes(resourceMetrics.Resource().Attributes())
		for j := 0; j < resourceMetrics.ScopeMetrics().Len(); j++ {
			scopeMetrics := resourceMetrics.ScopeMetrics().At(j)
			for k := 0; k < scopeMetrics.Metrics().Len(); k++ {
				metric := scopeMetrics.Metrics().At(k)
				var dataPointLines []string
				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					dataPointLines = writeNumberDataPoints(resourceAttributeStrings, metric, metric.Gauge().DataPoints())
				case pmetric.MetricTypeSum:
					dataPointLines = writeNumberDataPoints(resourceAttributeStrings, metric, metric.Sum().DataPoints())
				case pmetric.MetricTypeHistogram:
					dataPointLines = writeHistogramDataPoints(resourceAttributeStrings, metric)
					// TODO other cases - exponential histogram, summary
				}
				for _, line := range dataPointLines {
					buffer.WriteString(line)
				}
			}
		}
	}
	return buffer.Bytes(), nil
}

// writeAttributes returns a slice of strings in the form "attrKey=attrValue"
func writeAttributes(attributes pcommon.Map) (attributeStrings []string) {
	attributes.Range(func(k string, v pcommon.Value) bool {
		attribute := fmt.Sprintf("%s=%s", k, v.AsString())
		attributeStrings = append(attributeStrings, attribute)
		return true
	})
	return attributeStrings
}

func writeNumberDataPoints(resourceAttributeStrings []string, metric pmetric.Metric, dataPoints pmetric.NumberDataPointSlice) (lines []string) {
	for i := 0; i < dataPoints.Len(); i++ {
		dataPoint := dataPoints.At(i)
		dataPointAttributes := writeAttributes(dataPoint.Attributes())
		allAttributes := append(resourceAttributeStrings, dataPointAttributes...)

		var value string
		switch dataPoint.ValueType() {
		case pmetric.NumberDataPointValueTypeInt:
			value = fmt.Sprintf("%v", dataPoint.IntValue())
		case pmetric.NumberDataPointValueTypeDouble:
			value = fmt.Sprintf("%v", dataPoint.DoubleValue())
		}

		dataPointLine := fmt.Sprintf("%s{%s} %s\n", metric.Name(), strings.Join(allAttributes, ","), value)
		lines = append(lines, dataPointLine)
	}
	return lines
}

func writeHistogramDataPoints(resourceAttributeStrings []string, metric pmetric.Metric) (lines []string) {
	for i := 0; i < metric.Histogram().DataPoints().Len(); i++ {
		dataPoint := metric.Histogram().DataPoints().At(i)
		dataPointAttributes := writeAttributes(dataPoint.Attributes())
		allAttributes := append(resourceAttributeStrings, dataPointAttributes...)

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

		dataPointLine := fmt.Sprintf("%s{%s} %s\n", metric.Name(), strings.Join(allAttributes, ","), value)
		lines = append(lines, dataPointLine)
	}
	return lines
}
