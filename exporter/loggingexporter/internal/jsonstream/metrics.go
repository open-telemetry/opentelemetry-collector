// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jsonstream

import (
	"go.opentelemetry.io/collector/model/pdata"
)

// NewJSONMetricsMarshaler returns a serializer.MetricsMarshaler to encode to OTLP JSON bytes.
func NewJSONMetricsMarshaler() pdata.MetricsMarshaler {
	return jsonMetricsMarshaler{}
}

type jsonMetricsMarshaler struct{}

// MarshalMetrics pdata.Metrics to OTLP JSON.
func (jsonMetricsMarshaler) MarshalMetrics(md pdata.Metrics) ([]byte, error) {
	buf := dataBuffer{}
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)

		attrs := rm.Resource().Attributes()

		ilms := rm.InstrumentationLibraryMetrics()
		if ilms.Len() == 0 {
			buf.object(func() {
				buf.resource("resourceMetric", attrs)
			})
			continue
		}
		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)

			lib := ilm.InstrumentationLibrary()

			metrics := ilm.Metrics()
			if metrics.Len() == 0 {
				buf.object(func() {
					buf.resource("instrumentationLibraryMetric", attrs)
					buf.instrumentationLibrary(lib)
				})
				continue
			}
			for k := 0; k < metrics.Len(); k++ {
				metric := metrics.At(k)

				buf.object(func() {
					buf.resource("metric", attrs)
					buf.instrumentationLibrary(lib)

					buf.fieldString("name", metric.Name())
					buf.fieldString("description", metric.Description())
					buf.fieldString("unit", metric.Unit())
					buf.fieldString("dataType", metric.DataType().String())
					buf.metricDataPoints(metric)
				})
			}
		}
	}

	return buf.buf.Bytes(), nil
}

func (b *dataBuffer) metricDataPoints(m pdata.Metric) {
	switch m.DataType() {
	case pdata.MetricDataTypeNone:
		return
	case pdata.MetricDataTypeGauge:
		data := m.Gauge()
		b.metricNumberDataPoints(data.DataPoints())
	case pdata.MetricDataTypeSum:
		data := m.Sum()
		b.fieldBool("isMonotonic", data.IsMonotonic())
		b.fieldString("aggregationTemporality", data.AggregationTemporality().String())
		b.metricNumberDataPoints(data.DataPoints())
	case pdata.MetricDataTypeHistogram:
		data := m.Histogram()
		b.fieldString("aggregationTemporality", data.AggregationTemporality().String())
		b.histogramDataPoints(data.DataPoints())
	case pdata.MetricDataTypeExponentialHistogram:
		data := m.ExponentialHistogram()
		b.fieldString("aggregationTemporality", data.AggregationTemporality().String())
		b.exponentialHistogramDataPoints(data.DataPoints())
	case pdata.MetricDataTypeSummary:
		data := m.Summary()
		b.summaryDataPoints(data.DataPoints())
	}
}

func (b *dataBuffer) metricNumberDataPoints(ps pdata.NumberDataPointSlice) {
	b.fieldArray("dataPoints", func() {
		for i := 0; i < ps.Len(); i++ {
			p := ps.At(i)

			b.object(func() {
				b.fieldAttrs("attributes", p.Attributes())
				b.fieldTime("startTimestamp", p.StartTimestamp())
				b.fieldTime("timestamp", p.Timestamp())

				switch p.Type() {
				case pdata.MetricValueTypeInt:
					b.fieldInt64("value", p.IntVal())
				case pdata.MetricValueTypeDouble:
					b.fieldFloat64("value", p.DoubleVal())
				}
			})
		}
	})
}

func (b *dataBuffer) histogramDataPoints(ps pdata.HistogramDataPointSlice) {
	b.fieldArray("dataPoints", func() {
		for i := 0; i < ps.Len(); i++ {
			p := ps.At(i)

			b.object(func() {
				b.fieldAttrs("attributes", p.Attributes())
				b.fieldTime("startTimestamp", p.StartTimestamp())
				b.fieldTime("timestamp", p.Timestamp())

				b.fieldUint64("count", p.Count())
				b.fieldFloat64("sum", p.Sum())

				bounds := p.ExplicitBounds()
				b.fieldArray("explicitBounds", func() {
					for _, bound := range bounds {
						b.elementFloat64(bound)
					}
				})

				b.histogramBucketCounts(p.BucketCounts())
			})
		}
	})
}

func (b *dataBuffer) exponentialHistogramDataPoints(ps pdata.ExponentialHistogramDataPointSlice) {
	b.fieldArray("dataPoints", func() {
		for i := 0; i < ps.Len(); i++ {
			p := ps.At(i)

			b.object(func() {
				b.fieldAttrs("attributes", p.Attributes())
				b.fieldTime("startTimestamp", p.StartTimestamp())
				b.fieldTime("timestamp", p.Timestamp())

				b.fieldUint64("count", p.Count())
				b.fieldFloat64("sum", p.Sum())
				b.fieldInt32("scale", p.Scale())
				b.fieldUint64("zeroCount", p.ZeroCount())

				b.exponentialHistogramBuckets("negative", p.Negative())
				b.exponentialHistogramBuckets("positive", p.Positive())
			})
		}
	})
}

func (b *dataBuffer) histogramBucketCounts(buckets []uint64) {
	b.fieldArray("bucketCounts", func() {
		for _, bucket := range buckets {
			b.elementUint64(bucket)
		}
	})
}

func (b *dataBuffer) exponentialHistogramBuckets(name string, xs pdata.Buckets) {
	b.fieldObject(name, func() {
		b.fieldInt32("offset", xs.Offset())
		b.histogramBucketCounts(xs.BucketCounts())
	})
}

func (b *dataBuffer) summaryDataPoints(ps pdata.SummaryDataPointSlice) {
	b.fieldArray("dataPoints", func() {
		for i := 0; i < ps.Len(); i++ {
			p := ps.At(i)

			b.object(func() {
				b.fieldAttrs("attributes", p.Attributes())
				b.fieldTime("startTimestamp", p.StartTimestamp())
				b.fieldTime("timestamp", p.Timestamp())

				b.fieldUint64("count", p.Count())
				b.fieldFloat64("sum", p.Sum())

				quantiles := p.QuantileValues()
				b.fieldArray("quantileValues", func() {
					for i := 0; i < quantiles.Len(); i++ {
						q := quantiles.At(i)
						b.object(func() {
							b.fieldFloat64("quantile", q.Quantile())
							b.fieldFloat64("value", q.Value())
						})
					}
				})
			})
		}
	})
}
