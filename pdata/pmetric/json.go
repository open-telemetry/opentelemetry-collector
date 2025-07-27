// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric // import "go.opentelemetry.io/collector/pdata/pmetric"

import (
	"slices"

	jsoniter "github.com/json-iterator/go"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
)

var _ Marshaler = (*JSONMarshaler)(nil)

// JSONMarshaler marshals pdata.Metrics to JSON bytes using the OTLP/JSON format.
type JSONMarshaler struct{}

// MarshalMetrics to the OTLP/JSON format.
func (*JSONMarshaler) MarshalMetrics(md Metrics) ([]byte, error) {
	dest := json.BorrowStream(nil)
	defer json.ReturnStream(dest)
	md.marshalJSONStream(dest)
	return slices.Clone(dest.Buffer()), dest.Error
}

// JSONUnmarshaler unmarshals OTLP/JSON formatted-bytes to pdata.Metrics.
type JSONUnmarshaler struct{}

// UnmarshalMetrics from OTLP/JSON format into pdata.Metrics.
func (*JSONUnmarshaler) UnmarshalMetrics(buf []byte) (Metrics, error) {
	iter := jsoniter.ConfigFastest.BorrowIterator(buf)
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	md := NewMetrics()
	md.unmarshalJSONIter(iter)
	if iter.Error != nil {
		return Metrics{}, iter.Error
	}
	otlp.MigrateMetrics(md.getOrig().ResourceMetrics)
	return md, nil
}

func (ms ResourceMetrics) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "resource":
			internal.UnmarshalJSONIterResource(internal.NewResource(&ms.orig.Resource, ms.state), iter)
		case "scopeMetrics", "scope_metrics":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.ScopeMetrics().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "schemaUrl", "schema_url":
			ms.orig.SchemaUrl = iter.ReadString()
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms ScopeMetrics) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "scope":
			internal.UnmarshalJSONIterInstrumentationScope(internal.NewInstrumentationScope(&ms.orig.Scope, ms.state), iter)
		case "metrics":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.Metrics().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "schemaUrl", "schema_url":
			ms.orig.SchemaUrl = iter.ReadString()
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms Metric) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "name":
			ms.orig.Name = iter.ReadString()
		case "description":
			ms.orig.Description = iter.ReadString()
		case "unit":
			ms.orig.Unit = iter.ReadString()
		case "metadata":
			internal.UnmarshalJSONIterMap(internal.NewMap(&ms.orig.Metadata, ms.state), iter)
		case "sum":
			ms.SetEmptySum().unmarshalJSONIter(iter)
		case "gauge":
			ms.SetEmptyGauge().unmarshalJSONIter(iter)
		case "histogram":
			ms.SetEmptyHistogram().unmarshalJSONIter(iter)
		case "exponential_histogram", "exponentialHistogram":
			ms.SetEmptyExponentialHistogram().unmarshalJSONIter(iter)
		case "summary":
			ms.SetEmptySummary().unmarshalJSONIter(iter)
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms Sum) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "aggregation_temporality", "aggregationTemporality":
			ms.orig.AggregationTemporality = readAggregationTemporality(iter)
		case "is_monotonic", "isMonotonic":
			ms.orig.IsMonotonic = iter.ReadBool()
		case "data_points", "dataPoints":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.DataPoints().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms Gauge) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "data_points", "dataPoints":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.DataPoints().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms Histogram) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "data_points", "dataPoints":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.DataPoints().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "aggregation_temporality", "aggregationTemporality":
			ms.orig.AggregationTemporality = readAggregationTemporality(iter)
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms ExponentialHistogram) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "data_points", "dataPoints":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.DataPoints().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "aggregation_temporality", "aggregationTemporality":
			ms.orig.AggregationTemporality = readAggregationTemporality(iter)
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms Summary) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "data_points", "dataPoints":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.DataPoints().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms NumberDataPoint) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "timeUnixNano", "time_unix_nano":
			ms.orig.TimeUnixNano = json.ReadUint64(iter)
		case "start_time_unix_nano", "startTimeUnixNano":
			ms.orig.StartTimeUnixNano = json.ReadUint64(iter)
		case "as_int", "asInt":
			ms.orig.Value = &otlpmetrics.NumberDataPoint_AsInt{
				AsInt: json.ReadInt64(iter),
			}
		case "as_double", "asDouble":
			ms.orig.Value = &otlpmetrics.NumberDataPoint_AsDouble{
				AsDouble: json.ReadFloat64(iter),
			}
		case "attributes":
			internal.UnmarshalJSONIterMap(internal.NewMap(&ms.orig.Attributes, ms.state), iter)
		case "exemplars":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.Exemplars().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "flags":
			ms.orig.Flags = json.ReadUint32(iter)
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms HistogramDataPoint) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "timeUnixNano", "time_unix_nano":
			ms.orig.TimeUnixNano = json.ReadUint64(iter)
		case "start_time_unix_nano", "startTimeUnixNano":
			ms.orig.StartTimeUnixNano = json.ReadUint64(iter)
		case "attributes":
			internal.UnmarshalJSONIterMap(internal.NewMap(&ms.orig.Attributes, ms.state), iter)
		case "count":
			ms.orig.Count = json.ReadUint64(iter)
		case "sum":
			ms.orig.Sum_ = &otlpmetrics.HistogramDataPoint_Sum{Sum: json.ReadFloat64(iter)}
		case "bucket_counts", "bucketCounts":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.orig.BucketCounts = append(ms.orig.BucketCounts, json.ReadUint64(iter))
				return true
			})
		case "explicit_bounds", "explicitBounds":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.orig.ExplicitBounds = append(ms.orig.ExplicitBounds, json.ReadFloat64(iter))
				return true
			})
		case "exemplars":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.Exemplars().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "flags":
			ms.orig.Flags = json.ReadUint32(iter)
		case "max":
			ms.orig.Max_ = &otlpmetrics.HistogramDataPoint_Max{
				Max: json.ReadFloat64(iter),
			}
		case "min":
			ms.orig.Min_ = &otlpmetrics.HistogramDataPoint_Min{
				Min: json.ReadFloat64(iter),
			}
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms ExponentialHistogramDataPoint) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "timeUnixNano", "time_unix_nano":
			ms.orig.TimeUnixNano = json.ReadUint64(iter)
		case "start_time_unix_nano", "startTimeUnixNano":
			ms.orig.StartTimeUnixNano = json.ReadUint64(iter)
		case "attributes":
			internal.UnmarshalJSONIterMap(internal.NewMap(&ms.orig.Attributes, ms.state), iter)
		case "count":
			ms.orig.Count = json.ReadUint64(iter)
		case "sum":
			ms.orig.Sum_ = &otlpmetrics.ExponentialHistogramDataPoint_Sum{
				Sum: json.ReadFloat64(iter),
			}
		case "scale":
			ms.orig.Scale = iter.ReadInt32()
		case "zero_count", "zeroCount":
			ms.orig.ZeroCount = json.ReadUint64(iter)
		case "positive":
			ms.Positive().unmarshalJSONIter(iter)
		case "negative":
			ms.Negative().unmarshalJSONIter(iter)
		case "exemplars":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.Exemplars().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "flags":
			ms.orig.Flags = json.ReadUint32(iter)
		case "max":
			ms.orig.Max_ = &otlpmetrics.ExponentialHistogramDataPoint_Max{
				Max: json.ReadFloat64(iter),
			}
		case "min":
			ms.orig.Min_ = &otlpmetrics.ExponentialHistogramDataPoint_Min{
				Min: json.ReadFloat64(iter),
			}
		case "zeroThreshold", "zero_threshold":
			ms.orig.ZeroThreshold = json.ReadFloat64(iter)
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms SummaryDataPoint) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "timeUnixNano", "time_unix_nano":
			ms.orig.TimeUnixNano = json.ReadUint64(iter)
		case "start_time_unix_nano", "startTimeUnixNano":
			ms.orig.StartTimeUnixNano = json.ReadUint64(iter)
		case "attributes":
			internal.UnmarshalJSONIterMap(internal.NewMap(&ms.orig.Attributes, ms.state), iter)
		case "count":
			ms.orig.Count = json.ReadUint64(iter)
		case "sum":
			ms.orig.Sum = json.ReadFloat64(iter)
		case "quantile_values", "quantileValues":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.QuantileValues().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "flags":
			ms.orig.Flags = json.ReadUint32(iter)
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms ExponentialHistogramDataPointBuckets) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "bucket_counts", "bucketCounts":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.orig.BucketCounts = append(ms.orig.BucketCounts, json.ReadUint64(iter))
				return true
			})
		case "offset":
			ms.orig.Offset = iter.ReadInt32()
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms SummaryDataPointValueAtQuantile) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "quantile":
			ms.orig.Quantile = json.ReadFloat64(iter)
		case "value":
			ms.orig.Value = json.ReadFloat64(iter)
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms Exemplar) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "filtered_attributes", "filteredAttributes":
			internal.UnmarshalJSONIterMap(internal.NewMap(&ms.orig.FilteredAttributes, ms.state), iter)
		case "timeUnixNano", "time_unix_nano":
			ms.orig.TimeUnixNano = json.ReadUint64(iter)
		case "as_int", "asInt":
			ms.orig.Value = &otlpmetrics.Exemplar_AsInt{
				AsInt: json.ReadInt64(iter),
			}
		case "as_double", "asDouble":
			ms.orig.Value = &otlpmetrics.Exemplar_AsDouble{
				AsDouble: json.ReadFloat64(iter),
			}
		case "traceId", "trace_id":
			ms.orig.TraceId.UnmarshalJSONIter(iter)
		case "spanId", "span_id":
			ms.orig.SpanId.UnmarshalJSONIter(iter)
		default:
			iter.Skip()
		}
		return true
	})
}

func readAggregationTemporality(iter *jsoniter.Iterator) otlpmetrics.AggregationTemporality {
	return otlpmetrics.AggregationTemporality(json.ReadEnumValue(iter, otlpmetrics.AggregationTemporality_value))
}
