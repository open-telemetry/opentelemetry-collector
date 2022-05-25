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
	"encoding/base64"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
	"go.opentelemetry.io/collector/pdata/internal/otlp"

	"github.com/gogo/protobuf/jsonpb"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
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
	delegate jsonpb.Unmarshaler
}

// NewJSONUnmarshaler returns a model.Unmarshaler. Unmarshals from OTLP json bytes.
func NewJSONUnmarshaler() Unmarshaler {
	return newJSONUnmarshaler()
}

func newJSONUnmarshaler() *jsonUnmarshaler {
	return &jsonUnmarshaler{delegate: jsonpb.Unmarshaler{}}
}

func (d *jsonUnmarshaler) UnmarshalMetrics(buf []byte) (Metrics, error) {
	md := otlpmetrics.MetricsData{}
	if err := d.delegate.Unmarshal(bytes.NewReader(buf), &md); err != nil {
		return Metrics{}, err
	}
	otlp.MigrateMetrics(md.ResourceMetrics)
	return internal.MetricsFromProto(md), nil
}

type jsoniterUnmarshaler struct {
}

// NewJSONiterUnmarshaler returns a model.Unmarshaler. Unmarshals from OTLP json bytes.
func NewJSONiterUnmarshaler() Unmarshaler {
	return newJSONITERUnmarshaler()
}

func newJSONITERUnmarshaler() *jsoniterUnmarshaler {
	return &jsoniterUnmarshaler{}
}

func (d *jsoniterUnmarshaler) UnmarshalMetrics(buf []byte) (Metrics, error) {
	iter := jsoniter.ConfigFastest.BorrowIterator(buf)
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	td := readMetricsData(iter)
	err := iter.Error
	return internal.MetricsFromProto(td), err
}

func readMetricsData(iter *jsoniter.Iterator) otlpmetrics.MetricsData {
	td := otlpmetrics.MetricsData{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "resource_metrics", "resourceMetrics":
			iter.ReadArrayCB(func(iterator *jsoniter.Iterator) bool {
				td.ResourceMetrics = append(td.ResourceMetrics, readResourceMetrics(iter))
				return true
			})
		}
		return true
	})
	return td
}

func readResourceMetrics(iter *jsoniter.Iterator) *otlpmetrics.ResourceMetrics {
	rs := &otlpmetrics.ResourceMetrics{}

	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "resource":
			iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
				switch f {
				case "attributes":
					iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
						rs.Resource.Attributes = append(rs.Resource.Attributes, readAttribute(iter))
						return true
					})
				case "droppedAttributesCount", "dropped_attributes_count":
					rs.Resource.DroppedAttributesCount = iter.ReadUint32()
				default:
					iter.ReportError("readResourceMetrics.resource", fmt.Sprintf("unknown field:%v", f))
				}
				return true
			})
		case "instrumentationLibraryMetrics", "instrumentation_library_metrics", "scopeMetrics", "scope_metrics":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				rs.ScopeMetrics = append(rs.ScopeMetrics,
					readInstrumentationLibraryMetrics(iter))
				return true
			})
		case "schemaUrl", "schema_url":
			rs.SchemaUrl = iter.ReadString()
		default:
			iter.ReportError("readResourceMetrics", fmt.Sprintf("unknown field:%v", f))
		}
		return true
	})
	return rs
}

func readInstrumentationLibraryMetrics(iter *jsoniter.Iterator) *otlpmetrics.ScopeMetrics {
	ils := &otlpmetrics.ScopeMetrics{}

	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "instrumentationLibrary", "instrumentation_library", "scope":
			iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
				switch f {
				case "name":
					ils.Scope.Name = iter.ReadString()
				case "version":
					ils.Scope.Version = iter.ReadString()
				default:
					iter.ReportError("readInstrumentationLibrarySpans.instrumentationLibrary", fmt.Sprintf("unknown field:%v", f))
				}
				return true
			})
		case "metrics":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ils.Metrics = append(ils.Metrics, readMetric(iter))
				return true
			})
		case "schemaUrl", "schema_url":
			ils.SchemaUrl = iter.ReadString()
		default:
			iter.ReportError("readInstrumentationLibrarySpans", fmt.Sprintf("unknown field:%v", f))
		}
		return true
	})
	return ils
}

func readMetric(iter *jsoniter.Iterator) *otlpmetrics.Metric {
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
			data := sp.Data.(*otlpmetrics.Metric_Sum)
			iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
				switch f {
				case "aggregation_temporality", "aggregationTemporality":
					data.Sum.AggregationTemporality = otlpmetrics.AggregationTemporality(iter.ReadInt32())
				case "is_monotonic", "isMonotonic":
					data.Sum.IsMonotonic = iter.ReadBool()
				case "data_points", "dataPoints":
					iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
						data.Sum.DataPoints = append(data.Sum.DataPoints, readNumberDataPoint(iter))
						return true
					})
				default:
					iter.ReportError("readMetric.Sum", fmt.Sprintf("unknown field:%v", f))
				}
				return true
			})
		case "gauge":
			data := sp.Data.(*otlpmetrics.Metric_Gauge)
			iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
				switch f {
				case "data_points", "dataPoints":
					iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
						data.Gauge.DataPoints = append(data.Gauge.DataPoints, readNumberDataPoint(iter))
						return true
					})
				default:
					iter.ReportError("readMetric.Gauge", fmt.Sprintf("unknown field:%v", f))
				}
				return true
			})
		case "histogram":
			data := sp.Data.(*otlpmetrics.Metric_Histogram)
			iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
				switch f {
				case "data_points", "dataPoints":
					iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
						data.Histogram.DataPoints = append(data.Histogram.DataPoints, readHistogramDataPoint(iter))
						return true
					})
				case "aggregation_temporality", "aggregationTemporality":
					data.Histogram.AggregationTemporality = otlpmetrics.AggregationTemporality(iter.ReadInt32())
				default:
					iter.ReportError("readMetric.Histogram", fmt.Sprintf("unknown field:%v", f))
				}
				return true
			})
		case "exponential_histogram", "exponentialHistogram":
			data := sp.Data.(*otlpmetrics.Metric_ExponentialHistogram)
			iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
				switch f {
				case "data_points", "dataPoints":
					iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
						data.ExponentialHistogram.DataPoints = append(data.ExponentialHistogram.DataPoints,
							readExponentialHistogramDataPoint(iter))
						return true
					})
				case "aggregation_temporality", "aggregationTemporality":
					data.ExponentialHistogram.AggregationTemporality = otlpmetrics.AggregationTemporality(iter.ReadInt32())
				default:
					iter.ReportError("readMetric.Histogram", fmt.Sprintf("unknown field:%v", f))
				}
				return true
			})
		case "summary":
			data := sp.Data.(*otlpmetrics.Metric_Summary)
			iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
				switch f {
				case "data_points", "dataPoints":
					iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
						data.Summary.DataPoints = append(data.Summary.DataPoints,
							readSummaryDataPoint(iter))
						return true
					})
				default:
					iter.ReportError("readMetric.Summary", fmt.Sprintf("unknown field:%v", f))
				}
				return true
			})

		default:
			iter.ReportError("readMetric", fmt.Sprintf("unknown field:%v", f))
		}
		return true
	})
	return sp
}

func readAttribute(iter *jsoniter.Iterator) otlpcommon.KeyValue {
	kv := otlpcommon.KeyValue{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "key":
			kv.Key = iter.ReadString()
		case "value":
			iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
				kv.Value = readAnyValue(iter, f)
				return true
			})
		default:
			iter.ReportError("readAttribute", fmt.Sprintf("unknown field:%v", f))
		}
		return true
	})
	return kv
}

func readAnyValue(iter *jsoniter.Iterator, f string) otlpcommon.AnyValue {
	switch f {
	case "stringValue", "string_value":
		return otlpcommon.AnyValue{
			Value: &otlpcommon.AnyValue_StringValue{
				StringValue: iter.ReadString(),
			},
		}
	case "boolValue", "bool_value":
		return otlpcommon.AnyValue{
			Value: &otlpcommon.AnyValue_BoolValue{
				BoolValue: iter.ReadBool(),
			},
		}
	case "intValue", "int_value":
		return otlpcommon.AnyValue{
			Value: &otlpcommon.AnyValue_IntValue{
				IntValue: readInt64(iter),
			},
		}
	case "doubleValue", "double_value":
		return otlpcommon.AnyValue{
			Value: &otlpcommon.AnyValue_DoubleValue{
				DoubleValue: iter.ReadFloat64(),
			},
		}
	case "bytesValue", "bytes_value":
		v, err := base64.StdEncoding.DecodeString(iter.ReadString())
		if err != nil {
			iter.ReportError("bytesValue", fmt.Sprintf("base64 decode:%v", err))
			return otlpcommon.AnyValue{}
		}
		return otlpcommon.AnyValue{
			Value: &otlpcommon.AnyValue_BytesValue{
				BytesValue: v,
			},
		}
	case "arrayValue", "array_value":
		return otlpcommon.AnyValue{
			Value: &otlpcommon.AnyValue_ArrayValue{
				ArrayValue: readArray(iter),
			},
		}
	case "kvlistValue", "kvlist_value":
		return otlpcommon.AnyValue{
			Value: &otlpcommon.AnyValue_KvlistValue{
				KvlistValue: readKvlistValue(iter),
			},
		}
	default:
		iter.ReportError("readAnyValue", fmt.Sprintf("unknown field:%v", f))
		return otlpcommon.AnyValue{}
	}
}

func readArray(iter *jsoniter.Iterator) *otlpcommon.ArrayValue {
	v := &otlpcommon.ArrayValue{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "values":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
					v.Values = append(v.Values, readAnyValue(iter, f))
					return true
				})
				return true
			})
		default:
			iter.ReportError("readArray", fmt.Sprintf("unknown field:%s", f))
		}
		return true
	})
	return v
}

func readKvlistValue(iter *jsoniter.Iterator) *otlpcommon.KeyValueList {
	v := &otlpcommon.KeyValueList{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "values":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				v.Values = append(v.Values, readAttribute(iter))
				return true
			})
		default:
			iter.ReportError("readKvlistValue", fmt.Sprintf("unknown field:%s", f))
		}
		return true
	})
	return v
}

func readInt64(iter *jsoniter.Iterator) int64 {
	return iter.ReadAny().ToInt64()
}

func readExemplar(iter *jsoniter.Iterator) otlpmetrics.Exemplar {
	exemplar := otlpmetrics.Exemplar{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "filtered_attributes", "filteredAttributes":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				exemplar.FilteredAttributes = append(exemplar.FilteredAttributes, readAttribute(iter))
				return true
			})
		case "timeUnixNano", "time_unix_nano":
			exemplar.TimeUnixNano = uint64(readInt64(iter))
		case "as_int", "asInt":
			exemplar.Value = &otlpmetrics.Exemplar_AsInt{
				AsInt: readInt64(iter),
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
			iter.ReportError("readAttribute", fmt.Sprintf("unknown field:%v", f))
		}
		return true
	})
	return exemplar
}

func readExponentialBuckets(iter *jsoniter.Iterator) otlpmetrics.ExponentialHistogramDataPoint_Buckets {
	buckets := otlpmetrics.ExponentialHistogramDataPoint_Buckets{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "bucket_counts", "bucketCounts":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				buckets.BucketCounts = append(buckets.BucketCounts, iter.ReadUint64())
				return true
			})
		case "offset":
			buckets.Offset = iter.ReadInt32()
		default:
			iter.ReportError("ExponentialBuckets", fmt.Sprintf("unknown field:%v", f))
		}
		return true
	})
	return buckets
}

func readNumberDataPoint(iter *jsoniter.Iterator) *otlpmetrics.NumberDataPoint {
	point := &otlpmetrics.NumberDataPoint{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "timeUnixNano", "time_unix_nano":
			point.TimeUnixNano = uint64(readInt64(iter))
		case "start_time_unix_nano", "startTimeUnixNano":
			point.StartTimeUnixNano = uint64(readInt64(iter))
		case "as_int", "asInt":
			point.Value = &otlpmetrics.NumberDataPoint_AsInt{
				AsInt: readInt64(iter),
			}
		case "as_double", "asDouble":
			point.Value = &otlpmetrics.NumberDataPoint_AsDouble{
				AsDouble: iter.ReadFloat64(),
			}
		case "attributes":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.Attributes = append(point.Attributes, readAttribute(iter))
				return true
			})
		case "exemplars":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.Exemplars = append(point.Exemplars, readExemplar(iter))
				return true
			})
		case "flags":
			point.Flags = iter.ReadUint32()
		default:
			iter.ReportError("NumberDataPoint", fmt.Sprintf("unknown field:%v", f))
		}
		return true
	})
	return point
}

func readHistogramDataPoint(iter *jsoniter.Iterator) *otlpmetrics.HistogramDataPoint {
	point := &otlpmetrics.HistogramDataPoint{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "timeUnixNano", "time_unix_nano":
			point.TimeUnixNano = uint64(readInt64(iter))
		case "start_time_unix_nano", "startTimeUnixNano":
			point.StartTimeUnixNano = uint64(readInt64(iter))
		case "attributes":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.Attributes = append(point.Attributes, readAttribute(iter))
				return true
			})
		case "count":
			point.Count = uint64(readInt64(iter))
		case "sum":
			point.Sum_ = &otlpmetrics.HistogramDataPoint_Sum{Sum: iter.ReadFloat64()}
		case "bucket_counts", "bucketCounts":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.BucketCounts = append(point.BucketCounts, iter.ReadUint64())
				return true
			})
		case "explicit_bounds", "explicitBounds":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.ExplicitBounds = append(point.ExplicitBounds, iter.ReadFloat64())
				return true
			})
		case "exemplars":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.Exemplars = append(point.Exemplars, readExemplar(iter))
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
			iter.ReportError("HistogramDataPoint", fmt.Sprintf("unknown field:%v", f))
		}
		return true
	})
	return point
}

func readExponentialHistogramDataPoint(iter *jsoniter.Iterator) *otlpmetrics.ExponentialHistogramDataPoint {
	point := &otlpmetrics.ExponentialHistogramDataPoint{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "timeUnixNano", "time_unix_nano":
			point.TimeUnixNano = uint64(readInt64(iter))
		case "start_time_unix_nano", "startTimeUnixNano":
			point.StartTimeUnixNano = uint64(readInt64(iter))
		case "attributes":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.Attributes = append(point.Attributes, readAttribute(iter))
				return true
			})
		case "count":
			point.Count = uint64(readInt64(iter))
		case "sum":
			point.Sum_ = &otlpmetrics.ExponentialHistogramDataPoint_Sum{Sum: iter.ReadFloat64()}
		case "scale":
			point.Scale = iter.ReadInt32()
		case "zero_count", "zeroCount":
			point.ZeroCount = iter.ReadUint64()
		case "positive":
			iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
				point.Positive = readExponentialBuckets(iter)
				return true
			})
		case "negative":
			iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
				point.Negative = readExponentialBuckets(iter)
				return true
			})
		case "exemplars":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.Exemplars = append(point.Exemplars, readExemplar(iter))
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
			iter.ReportError("ExponentialHistogramDataPoint", fmt.Sprintf("unknown field:%v", f))
		}
		return true
	})
	return point
}

func readSummaryDataPoint(iter *jsoniter.Iterator) *otlpmetrics.SummaryDataPoint {
	point := &otlpmetrics.SummaryDataPoint{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "timeUnixNano", "time_unix_nano":
			point.TimeUnixNano = uint64(readInt64(iter))
		case "start_time_unix_nano", "startTimeUnixNano":
			point.StartTimeUnixNano = uint64(readInt64(iter))
		case "attributes":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.Attributes = append(point.Attributes, readAttribute(iter))
				return true
			})
		case "count":
			point.Count = uint64(readInt64(iter))
		case "sum":
			point.Sum = iter.ReadFloat64()
		case "quantile_values", "quantileValues":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				point.QuantileValues = append(point.QuantileValues, readQuantileValue(iter))
				return true
			})
		case "flags":
			point.Flags = iter.ReadUint32()
		default:
			iter.ReportError("SummaryDataPoint", fmt.Sprintf("unknown field:%v", f))
		}
		return true
	})
	return point
}

func readQuantileValue(iter *jsoniter.Iterator) *otlpmetrics.SummaryDataPoint_ValueAtQuantile {
	point := &otlpmetrics.SummaryDataPoint_ValueAtQuantile{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "quantile":
			point.Quantile = iter.ReadFloat64()
		case "value":
			point.Value = iter.ReadFloat64()
		default:
			iter.ReportError("ValueAtQuantile", fmt.Sprintf("unknown field:%v", f))
		}
		return true
	})
	return point
}
