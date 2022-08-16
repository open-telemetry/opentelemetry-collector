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

package pmetric // import "go.opentelemetry.io/collector/pdata/pmetric"

import (
	"bytes"
	"fmt"

	"github.com/gogo/protobuf/jsonpb"
	jsoniter "github.com/json-iterator/go"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
	commonjson "go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
)

// NewJSONMarshaler returns a model.Marshaler. Marshals to OTLP json bytes.
func NewJSONMarshaler() Marshaler {
	return newJSONMarshaler()
}

type jsonMarshaler struct {
	delegate jsonpb.Marshaler
}

func newJSONMarshaler() *jsonMarshaler {
	return &jsonMarshaler{delegate: jsonpb.Marshaler{}}
}

func (e *jsonMarshaler) MarshalMetrics(md Metrics) ([]byte, error) {
	buf := bytes.Buffer{}
	err := e.delegate.Marshal(&buf, internal.MetricsToOtlp(md))
	return buf.Bytes(), err
}

type jsonUnmarshaler struct {
}

// NewJSONUnmarshaler returns a model.Unmarshaler. Unmarshals from OTLP json bytes.
func NewJSONUnmarshaler() Unmarshaler {
	return &jsonUnmarshaler{}
}

func (d *jsonUnmarshaler) UnmarshalMetrics(buf []byte) (Metrics, error) {
	iter := jsoniter.ConfigFastest.BorrowIterator(buf)
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	md := d.readMetricsData(iter)
	if iter.Error != nil {
		return Metrics{}, iter.Error
	}
	otlp.MigrateMetrics(md.ResourceMetrics)
	return internal.MetricsFromProto(md), nil
}

func (d *jsonUnmarshaler) readMetricsData(iter *jsoniter.Iterator) otlpmetrics.MetricsData {
	md := otlpmetrics.MetricsData{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "resource_metrics", "resourceMetrics":
			iter.ReadArrayCB(func(iterator *jsoniter.Iterator) bool {
				md.ResourceMetrics = append(md.ResourceMetrics, d.readResourceMetrics(iter))
				return true
			})
		default:
			iter.Skip()
		}
		return true
	})
	return md
}

func (d *jsonUnmarshaler) readResourceMetrics(iter *jsoniter.Iterator) *otlpmetrics.ResourceMetrics {
	rs := &otlpmetrics.ResourceMetrics{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "resource":
			iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
				switch f {
				case "attributes":
					iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
						rs.Resource.Attributes = append(rs.Resource.Attributes, commonjson.ReadAttribute(iter))
						return true
					})
				case "droppedAttributesCount", "dropped_attributes_count":
					rs.Resource.DroppedAttributesCount = iter.ReadUint32()
				default:
					iter.Skip()
				}
				return true
			})
		case "scopeMetrics", "scope_metrics":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				rs.ScopeMetrics = append(rs.ScopeMetrics,
					d.readScopeMetrics(iter))
				return true
			})
		case "schemaUrl", "schema_url":
			rs.SchemaUrl = iter.ReadString()
		default:
			iter.Skip()
		}
		return true
	})
	return rs
}

func (d *jsonUnmarshaler) readScopeMetrics(iter *jsoniter.Iterator) *otlpmetrics.ScopeMetrics {
	ils := &otlpmetrics.ScopeMetrics{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "scope":
			iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
				switch f {
				case "name":
					ils.Scope.Name = iter.ReadString()
				case "version":
					ils.Scope.Version = iter.ReadString()
				case "attributes":
					iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
						ils.Scope.Attributes = append(ils.Scope.Attributes,
							commonjson.ReadAttribute(iter))
						return true
					})
				case "droppedAttributesCount", "dropped_attributes_count":
					ils.Scope.DroppedAttributesCount = iter.ReadUint32()
				default:
					iter.Skip()
				}
				return true
			})
		case "metrics":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ils.Metrics = append(ils.Metrics, d.readMetric(iter))
				return true
			})
		case "schemaUrl", "schema_url":
			ils.SchemaUrl = iter.ReadString()
		default:
			iter.Skip()
		}
		return true
	})
	return ils
}

func (d *jsonUnmarshaler) readMetric(iter *jsoniter.Iterator) *otlpmetrics.Metric {
	sp := &otlpmetrics.Metric{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "name":
			sp.Name = iter.ReadString()
		case "description":
			sp.Description = iter.ReadString()
		case "unit":
			sp.Unit = iter.ReadString()
		case "sum":
			sp.Data = d.readSumMetric(iter)
		case "gauge":
			sp.Data = d.readGaugeMetric(iter)
		case "histogram":
			sp.Data = d.readHistogramMetric(iter)
		case "exponential_histogram", "exponentialHistogram":
			sp.Data = d.readExponentialHistogramMetric(iter)
		case "summary":
			sp.Data = d.readSummaryMetric(iter)
		default:
			iter.Skip()
		}
		return true
	})
	return sp
}

func (d *jsonUnmarshaler) readSumMetric(iter *jsoniter.Iterator) *otlpmetrics.Metric_Sum {
	data := &otlpmetrics.Metric_Sum{
		Sum: &otlpmetrics.Sum{},
	}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "aggregation_temporality", "aggregationTemporality":
			data.Sum.AggregationTemporality = d.readAggregationTemporality(iter)
		case "is_monotonic", "isMonotonic":
			data.Sum.IsMonotonic = iter.ReadBool()
		case "data_points", "dataPoints":
			var dataPoints []*otlpmetrics.NumberDataPoint
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				dataPoints = append(dataPoints, d.readNumberDataPoint(iter))
				return true
			})
			data.Sum.DataPoints = dataPoints
		default:
			iter.Skip()
		}
		return true
	})
	return data
}

func (d *jsonUnmarshaler) readGaugeMetric(iter *jsoniter.Iterator) *otlpmetrics.Metric_Gauge {
	data := &otlpmetrics.Metric_Gauge{
		Gauge: &otlpmetrics.Gauge{},
	}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "data_points", "dataPoints":
			var dataPoints []*otlpmetrics.NumberDataPoint
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				dataPoints = append(dataPoints, d.readNumberDataPoint(iter))
				return true
			})
			data.Gauge.DataPoints = dataPoints
		default:
			iter.Skip()
		}
		return true
	})
	return data
}

func (d *jsonUnmarshaler) readHistogramMetric(iter *jsoniter.Iterator) *otlpmetrics.Metric_Histogram {
	data := &otlpmetrics.Metric_Histogram{
		Histogram: &otlpmetrics.Histogram{},
	}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "data_points", "dataPoints":
			var dataPoints []*otlpmetrics.HistogramDataPoint
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				dataPoints = append(dataPoints, d.readHistogramDataPoint(iter))
				return true
			})
			data.Histogram.DataPoints = dataPoints
		case "aggregation_temporality", "aggregationTemporality":
			data.Histogram.AggregationTemporality = d.readAggregationTemporality(iter)
		default:
			iter.Skip()
		}
		return true
	})
	return data
}

func (d *jsonUnmarshaler) readExponentialHistogramMetric(iter *jsoniter.Iterator) *otlpmetrics.Metric_ExponentialHistogram {
	data := &otlpmetrics.Metric_ExponentialHistogram{
		ExponentialHistogram: &otlpmetrics.ExponentialHistogram{},
	}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "data_points", "dataPoints":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				data.ExponentialHistogram.DataPoints = append(data.ExponentialHistogram.DataPoints,
					d.readExponentialHistogramDataPoint(iter))
				return true
			})
		case "aggregation_temporality", "aggregationTemporality":
			data.ExponentialHistogram.AggregationTemporality = d.readAggregationTemporality(iter)
		default:
			iter.Skip()
		}
		return true
	})
	return data
}

func (d *jsonUnmarshaler) readSummaryMetric(iter *jsoniter.Iterator) *otlpmetrics.Metric_Summary {
	data := &otlpmetrics.Metric_Summary{
		Summary: &otlpmetrics.Summary{},
	}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "data_points", "dataPoints":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				data.Summary.DataPoints = append(data.Summary.DataPoints,
					d.readSummaryDataPoint(iter))
				return true
			})
		default:
			iter.Skip()
		}
		return true
	})
	return data
}

func (d *jsonUnmarshaler) readExemplar(iter *jsoniter.Iterator) otlpmetrics.Exemplar {
	exemplar := otlpmetrics.Exemplar{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "filtered_attributes", "filteredAttributes":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				exemplar.FilteredAttributes = append(exemplar.FilteredAttributes, commonjson.ReadAttribute(iter))
				return true
			})
		case "timeUnixNano", "time_unix_nano":
			exemplar.TimeUnixNano = uint64(commonjson.ReadInt64(iter))
		case "as_int", "asInt":
			exemplar.Value = &otlpmetrics.Exemplar_AsInt{
				AsInt: commonjson.ReadInt64(iter),
			}
		case "as_double", "asDouble":
			exemplar.Value = &otlpmetrics.Exemplar_AsDouble{
				AsDouble: iter.ReadFloat64(),
			}
		case "traceId", "trace_id":
			if err := exemplar.TraceId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("exemplar.traceId", fmt.Sprintf("parse trace_id:%v", err))
			}
		case "spanId", "span_id":
			if err := exemplar.SpanId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("exemplar.spanId", fmt.Sprintf("parse span_id:%v", err))
			}
		default:
			iter.Skip()
		}
		return true
	})
	return exemplar
}

func (d *jsonUnmarshaler) readNumberDataPoint(iter *jsoniter.Iterator) *otlpmetrics.NumberDataPoint {
	point := &otlpmetrics.NumberDataPoint{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "timeUnixNano", "time_unix_nano":
			point.TimeUnixNano = uint64(commonjson.ReadInt64(iter))
		case "start_time_unix_nano", "startTimeUnixNano":
			point.StartTimeUnixNano = uint64(commonjson.ReadInt64(iter))
		case "as_int", "asInt":
			point.Value = &otlpmetrics.NumberDataPoint_AsInt{
				AsInt: commonjson.ReadInt64(iter),
			}
		case "as_double", "asDouble":
			point.Value = &otlpmetrics.NumberDataPoint_AsDouble{
				AsDouble: iter.ReadFloat64(),
			}
		case "attributes":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.Attributes = append(point.Attributes, commonjson.ReadAttribute(iter))
				return true
			})
		case "exemplars":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.Exemplars = append(point.Exemplars, d.readExemplar(iter))
				return true
			})
		case "flags":
			point.Flags = iter.ReadUint32()
		default:
			iter.Skip()
		}
		return true
	})
	return point
}

func (d *jsonUnmarshaler) readHistogramDataPoint(iter *jsoniter.Iterator) *otlpmetrics.HistogramDataPoint {
	point := &otlpmetrics.HistogramDataPoint{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "timeUnixNano", "time_unix_nano":
			point.TimeUnixNano = uint64(commonjson.ReadInt64(iter))
		case "start_time_unix_nano", "startTimeUnixNano":
			point.StartTimeUnixNano = uint64(commonjson.ReadInt64(iter))
		case "attributes":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.Attributes = append(point.Attributes, commonjson.ReadAttribute(iter))
				return true
			})
		case "count":
			point.Count = uint64(commonjson.ReadInt64(iter))
		case "sum":
			point.Sum_ = &otlpmetrics.HistogramDataPoint_Sum{Sum: iter.ReadFloat64()}
		case "bucket_counts", "bucketCounts":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.BucketCounts = append(point.BucketCounts, uint64(commonjson.ReadInt64(iter)))
				return true
			})
		case "explicit_bounds", "explicitBounds":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.ExplicitBounds = append(point.ExplicitBounds, iter.ReadFloat64())
				return true
			})
		case "exemplars":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.Exemplars = append(point.Exemplars, d.readExemplar(iter))
				return true
			})
		case "flags":
			point.Flags = iter.ReadUint32()
		case "max":
			point.Max_ = &otlpmetrics.HistogramDataPoint_Max{
				Max: iter.ReadFloat64(),
			}
		case "min":
			point.Min_ = &otlpmetrics.HistogramDataPoint_Min{
				Min: iter.ReadFloat64(),
			}
		default:
			iter.Skip()
		}
		return true
	})
	return point
}

func (d *jsonUnmarshaler) readExponentialHistogramDataPoint(iter *jsoniter.Iterator) *otlpmetrics.ExponentialHistogramDataPoint {
	point := &otlpmetrics.ExponentialHistogramDataPoint{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "timeUnixNano", "time_unix_nano":
			point.TimeUnixNano = uint64(commonjson.ReadInt64(iter))
		case "start_time_unix_nano", "startTimeUnixNano":
			point.StartTimeUnixNano = uint64(commonjson.ReadInt64(iter))
		case "attributes":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.Attributes = append(point.Attributes, commonjson.ReadAttribute(iter))
				return true
			})
		case "count":
			point.Count = uint64(commonjson.ReadInt64(iter))
		case "sum":
			point.Sum_ = &otlpmetrics.ExponentialHistogramDataPoint_Sum{
				Sum: iter.ReadFloat64(),
			}
		case "scale":
			point.Scale = iter.ReadInt32()
		case "zero_count", "zeroCount":
			point.ZeroCount = uint64(commonjson.ReadInt64(iter))
		case "positive":
			positive := otlpmetrics.ExponentialHistogramDataPoint_Buckets{}
			iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
				switch f {
				case "bucket_counts", "bucketCounts":
					iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
						positive.BucketCounts = append(positive.BucketCounts, uint64(commonjson.ReadInt64(iter)))
						return true
					})
				case "offset":
					positive.Offset = iter.ReadInt32()
				default:
					iter.Skip()
				}
				point.Positive = positive
				return true
			})
		case "negative":
			negative := otlpmetrics.ExponentialHistogramDataPoint_Buckets{}
			iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
				switch f {
				case "bucket_counts", "bucketCounts":
					iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
						negative.BucketCounts = append(negative.BucketCounts, uint64(commonjson.ReadInt64(iter)))
						return true
					})
				case "offset":
					negative.Offset = iter.ReadInt32()
				default:
					iter.Skip()
				}
				point.Negative = negative
				return true
			})
		case "exemplars":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.Exemplars = append(point.Exemplars, d.readExemplar(iter))
				return true
			})
		case "flags":
			point.Flags = iter.ReadUint32()
		case "max":
			point.Max_ = &otlpmetrics.ExponentialHistogramDataPoint_Max{
				Max: iter.ReadFloat64(),
			}
		case "min":
			point.Min_ = &otlpmetrics.ExponentialHistogramDataPoint_Min{
				Min: iter.ReadFloat64(),
			}
		default:
			iter.Skip()
		}
		return true
	})
	return point
}

func (d *jsonUnmarshaler) readSummaryDataPoint(iter *jsoniter.Iterator) *otlpmetrics.SummaryDataPoint {
	point := &otlpmetrics.SummaryDataPoint{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "timeUnixNano", "time_unix_nano":
			point.TimeUnixNano = uint64(commonjson.ReadInt64(iter))
		case "start_time_unix_nano", "startTimeUnixNano":
			point.StartTimeUnixNano = uint64(commonjson.ReadInt64(iter))
		case "attributes":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.Attributes = append(point.Attributes, commonjson.ReadAttribute(iter))
				return true
			})
		case "count":
			point.Count = uint64(commonjson.ReadInt64(iter))
		case "sum":
			point.Sum = iter.ReadFloat64()
		case "quantile_values", "quantileValues":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.QuantileValues = append(point.QuantileValues, d.readQuantileValue(iter))
				return true
			})
		case "flags":
			point.Flags = iter.ReadUint32()
		default:
			iter.Skip()
		}
		return true
	})
	return point
}

func (d *jsonUnmarshaler) readQuantileValue(iter *jsoniter.Iterator) *otlpmetrics.SummaryDataPoint_ValueAtQuantile {
	point := &otlpmetrics.SummaryDataPoint_ValueAtQuantile{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "quantile":
			point.Quantile = iter.ReadFloat64()
		case "value":
			point.Value = iter.ReadFloat64()
		default:
			iter.Skip()
		}
		return true
	})
	return point
}

func (d *jsonUnmarshaler) readAggregationTemporality(iter *jsoniter.Iterator) otlpmetrics.AggregationTemporality {
	value := iter.ReadAny()
	if v := value.ToInt(); v > 0 {
		return otlpmetrics.AggregationTemporality(v)
	}
	v := value.ToString()
	return otlpmetrics.AggregationTemporality(otlpmetrics.AggregationTemporality_value[v])
}
