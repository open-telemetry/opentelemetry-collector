// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

var pmetric = &Package{
	info: &PackageInfo{
		name: "pmetric",
		path: "pmetric",
		imports: []string{
			`"encoding/binary"`,
			`"fmt"`,
			`"iter"`,
			`"math"`,
			`"sort"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/data"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`"go.opentelemetry.io/collector/pdata/internal/proto"`,
			`otlpcollectormetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/metrics/v1"`,
			`otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
		testImports: []string{
			`"testing"`,
			`"unsafe"`,
			``,
			`"github.com/stretchr/testify/assert"`,
			`"github.com/stretchr/testify/require"`,
			`"google.golang.org/protobuf/proto"`,
			`gootlpcollectormetrics "go.opentelemetry.io/proto/slim/otlp/collector/metrics/v1"`,
			`gootlpmetrics "go.opentelemetry.io/proto/slim/otlp/metrics/v1"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/data"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`otlpcollectormetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/metrics/v1"`,
			`otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
	},
	structs: []baseStruct{
		metrics,
		resourceMetricsSlice,
		resourceMetrics,
		scopeMetricsSlice,
		scopeMetrics,
		metricSlice,
		metric,
		gauge,
		sum,
		histogram,
		exponentialHistogram,
		summary,
		numberDataPointSlice,
		numberDataPoint,
		histogramDataPointSlice,
		histogramDataPoint,
		exponentialHistogramDataPointSlice,
		exponentialHistogramDataPoint,
		bucketsValues,
		summaryDataPointSlice,
		summaryDataPoint,
		quantileValuesSlice,
		quantileValues,
		exemplarSlice,
		exemplar,
	},
}

var metrics = &messageStruct{
	structName:     "Metrics",
	description:    "// Metrics is the top-level struct that is propagated through the metrics pipeline.\n// Use NewMetrics to create new instance, zero-initialized instance is not valid for use.",
	originFullName: "otlpcollectormetrics.ExportMetricsServiceRequest",
	fields: []Field{
		&SliceField{
			fieldName:   "ResourceMetrics",
			protoID:     1,
			protoType:   ProtoTypeMessage,
			returnSlice: resourceMetricsSlice,
		},
	},
	hasWrapper: true,
}

var resourceMetricsSlice = &messageSlice{
	structName:      "ResourceMetricsSlice",
	elementNullable: true,
	element:         resourceMetrics,
}

var resourceMetrics = &messageStruct{
	structName:     "ResourceMetrics",
	description:    "// ResourceMetrics is a collection of metrics from a Resource.",
	originFullName: "otlpmetrics.ResourceMetrics",
	fields: []Field{
		&MessageField{
			fieldName:     "Resource",
			protoID:       1,
			returnMessage: resource,
		},
		&SliceField{
			fieldName:   "ScopeMetrics",
			protoID:     2,
			protoType:   ProtoTypeMessage,
			returnSlice: scopeMetricsSlice,
		},
		&PrimitiveField{
			fieldName: "SchemaUrl",
			protoID:   3,
			protoType: ProtoTypeString,
		},
	},
}

var scopeMetricsSlice = &messageSlice{
	structName:      "ScopeMetricsSlice",
	elementNullable: true,
	element:         scopeMetrics,
}

var scopeMetrics = &messageStruct{
	structName:     "ScopeMetrics",
	description:    "// ScopeMetrics is a collection of metrics from a LibraryInstrumentation.",
	originFullName: "otlpmetrics.ScopeMetrics",
	fields: []Field{
		&MessageField{
			fieldName:     "Scope",
			protoID:       1,
			returnMessage: scope,
		},
		&SliceField{
			fieldName:   "Metrics",
			protoID:     2,
			protoType:   ProtoTypeMessage,
			returnSlice: metricSlice,
		},
		&PrimitiveField{
			fieldName: "SchemaUrl",
			protoID:   3,
			protoType: ProtoTypeString,
		},
	},
}

var metricSlice = &messageSlice{
	structName:      "MetricSlice",
	elementNullable: true,
	element:         metric,
}

var metric = &messageStruct{
	structName: "Metric",
	description: "// Metric represents one metric as a collection of datapoints.\n" +
		"// See Metric definition in OTLP: https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/metrics/v1/metrics.proto",
	originFullName: "otlpmetrics.Metric",
	fields: []Field{
		&PrimitiveField{
			fieldName: "Name",
			protoID:   1,
			protoType: ProtoTypeString,
		},
		&PrimitiveField{
			fieldName: "Description",
			protoID:   2,
			protoType: ProtoTypeString,
		},
		&PrimitiveField{
			fieldName: "Unit",
			protoID:   3,
			protoType: ProtoTypeString,
		},
		&OneOfField{
			typeName:                   "MetricType",
			originFieldName:            "Data",
			testValueIdx:               1, // Sum
			omitOriginFieldNameInNames: true,
			values: []oneOfValue{
				&OneOfMessageValue{
					fieldName:              "Gauge",
					protoID:                5,
					originFieldPackageName: "otlpmetrics",
					returnMessage:          gauge,
				},
				&OneOfMessageValue{
					fieldName:              "Sum",
					protoID:                7,
					originFieldPackageName: "otlpmetrics",
					returnMessage:          sum,
				},
				&OneOfMessageValue{
					fieldName:              "Histogram",
					protoID:                9,
					originFieldPackageName: "otlpmetrics",
					returnMessage:          histogram,
				},
				&OneOfMessageValue{
					fieldName:              "ExponentialHistogram",
					protoID:                10,
					originFieldPackageName: "otlpmetrics",
					returnMessage:          exponentialHistogram,
				},
				&OneOfMessageValue{
					fieldName:              "Summary",
					protoID:                11,
					originFieldPackageName: "otlpmetrics",
					returnMessage:          summary,
				},
			},
		},
		&SliceField{
			fieldName:   "Metadata",
			protoID:     12,
			protoType:   ProtoTypeMessage,
			returnSlice: mapStruct,
		},
	},
}

var gauge = &messageStruct{
	structName:     "Gauge",
	description:    "// Gauge represents the type of a numeric metric that always exports the \"current value\" for every data point.",
	originFullName: "otlpmetrics.Gauge",
	fields: []Field{
		&SliceField{
			fieldName:   "DataPoints",
			protoID:     1,
			protoType:   ProtoTypeMessage,
			returnSlice: numberDataPointSlice,
		},
	},
}

var sum = &messageStruct{
	structName:     "Sum",
	description:    "// Sum represents the type of a numeric metric that is calculated as a sum of all reported measurements over a time interval.",
	originFullName: "otlpmetrics.Sum",
	fields: []Field{
		&SliceField{
			fieldName:   "DataPoints",
			protoID:     1,
			protoType:   ProtoTypeMessage,
			returnSlice: numberDataPointSlice,
		},
		&TypedField{
			fieldName:  "AggregationTemporality",
			protoID:    2,
			returnType: aggregationTemporalityType,
		},
		&PrimitiveField{
			fieldName: "IsMonotonic",
			protoID:   3,
			protoType: ProtoTypeBool,
		},
	},
}

var histogram = &messageStruct{
	structName:     "Histogram",
	description:    "// Histogram represents the type of a metric that is calculated by aggregating as a Histogram of all reported measurements over a time interval.",
	originFullName: "otlpmetrics.Histogram",
	fields: []Field{
		&SliceField{
			fieldName:   "DataPoints",
			protoID:     1,
			protoType:   ProtoTypeMessage,
			returnSlice: histogramDataPointSlice,
		},
		&TypedField{
			fieldName:  "AggregationTemporality",
			protoID:    2,
			returnType: aggregationTemporalityType,
		},
	},
}

var exponentialHistogram = &messageStruct{
	structName: "ExponentialHistogram",
	description: `// ExponentialHistogram represents the type of a metric that is calculated by aggregating
	// as a ExponentialHistogram of all reported double measurements over a time interval.`,
	originFullName: "otlpmetrics.ExponentialHistogram",
	fields: []Field{
		&SliceField{
			fieldName:   "DataPoints",
			protoID:     1,
			protoType:   ProtoTypeMessage,
			returnSlice: exponentialHistogramDataPointSlice,
		},
		&TypedField{
			fieldName:  "AggregationTemporality",
			protoID:    2,
			returnType: aggregationTemporalityType,
		},
	},
}

var summary = &messageStruct{
	structName:     "Summary",
	description:    "// Summary represents the type of a metric that is calculated by aggregating as a Summary of all reported double measurements over a time interval.",
	originFullName: "otlpmetrics.Summary",
	fields: []Field{
		&SliceField{
			fieldName:   "DataPoints",
			protoID:     1,
			protoType:   ProtoTypeMessage,
			returnSlice: summaryDataPointSlice,
		},
	},
}

var numberDataPointSlice = &messageSlice{
	structName:      "NumberDataPointSlice",
	elementNullable: true,
	element:         numberDataPoint,
}

var numberDataPoint = &messageStruct{
	structName:     "NumberDataPoint",
	description:    "// NumberDataPoint is a single data point in a timeseries that describes the time-varying value of a number metric.",
	originFullName: "otlpmetrics.NumberDataPoint",
	fields: []Field{
		&SliceField{
			fieldName:   "Attributes",
			protoID:     7,
			protoType:   ProtoTypeMessage,
			returnSlice: mapStruct,
		},
		&TypedField{
			fieldName:       "StartTimestamp",
			originFieldName: "StartTimeUnixNano",
			protoID:         2,
			returnType:      timestampType,
		},
		&TypedField{
			fieldName:       "Timestamp",
			originFieldName: "TimeUnixNano",
			protoID:         3,
			returnType:      timestampType,
		},
		&OneOfField{
			typeName:        "NumberDataPointValueType",
			originFieldName: "Value",
			testValueIdx:    0, // Double
			values: []oneOfValue{
				&OneOfPrimitiveValue{
					fieldName:       "Double",
					protoID:         4,
					originFieldName: "AsDouble",
					protoType:       ProtoTypeDouble,
				},
				&OneOfPrimitiveValue{
					fieldName:       "Int",
					protoID:         6,
					originFieldName: "AsInt",
					protoType:       ProtoTypeSFixed64,
				},
			},
		},
		&SliceField{
			fieldName:   "Exemplars",
			protoID:     5,
			protoType:   ProtoTypeMessage,
			returnSlice: exemplarSlice,
		},
		&TypedField{
			fieldName: "Flags",
			protoID:   8,
			returnType: &TypedType{
				structName: "DataPointFlags",
				protoType:  ProtoTypeUint32,
				defaultVal: "0",
				testVal:    "1",
			},
		},
	},
}

var histogramDataPointSlice = &messageSlice{
	structName:      "HistogramDataPointSlice",
	elementNullable: true,
	element:         histogramDataPoint,
}

var histogramDataPoint = &messageStruct{
	structName:     "HistogramDataPoint",
	description:    "// HistogramDataPoint is a single data point in a timeseries that describes the time-varying values of a Histogram of values.",
	originFullName: "otlpmetrics.HistogramDataPoint",
	fields: []Field{
		&SliceField{
			fieldName:   "Attributes",
			protoID:     9,
			protoType:   ProtoTypeMessage,
			returnSlice: mapStruct,
		},
		&TypedField{
			fieldName:       "StartTimestamp",
			originFieldName: "StartTimeUnixNano",
			protoID:         2,
			returnType:      timestampType,
		},
		&TypedField{
			fieldName:       "Timestamp",
			originFieldName: "TimeUnixNano",
			protoID:         3,
			returnType:      timestampType,
		},
		&PrimitiveField{
			fieldName: "Count",
			protoID:   4,
			protoType: ProtoTypeFixed64,
		},
		&OptionalPrimitiveField{
			fieldName: "Sum",
			protoID:   5,
			protoType: ProtoTypeDouble,
		},
		&SliceField{
			fieldName:   "BucketCounts",
			protoID:     6,
			protoType:   ProtoTypeFixed64,
			returnSlice: uInt64Slice,
		},
		&SliceField{
			fieldName:   "ExplicitBounds",
			protoID:     7,
			protoType:   ProtoTypeDouble,
			returnSlice: float64Slice,
		},
		&SliceField{
			fieldName:   "Exemplars",
			protoID:     8,
			protoType:   ProtoTypeMessage,
			returnSlice: exemplarSlice,
		},
		&TypedField{
			fieldName: "Flags",
			protoID:   10,
			returnType: &TypedType{
				structName: "DataPointFlags",
				protoType:  ProtoTypeUint32,
				defaultVal: "0",
				testVal:    "1",
			},
		},
		&OptionalPrimitiveField{
			fieldName: "Min",
			protoID:   11,
			protoType: ProtoTypeDouble,
		},
		&OptionalPrimitiveField{
			fieldName: "Max",
			protoID:   12,
			protoType: ProtoTypeDouble,
		},
	},
}

var exponentialHistogramDataPointSlice = &messageSlice{
	structName:      "ExponentialHistogramDataPointSlice",
	elementNullable: true,
	element:         exponentialHistogramDataPoint,
}

var exponentialHistogramDataPoint = &messageStruct{
	structName: "ExponentialHistogramDataPoint",
	description: `// ExponentialHistogramDataPoint is a single data point in a timeseries that describes the
	// time-varying values of a ExponentialHistogram of double values. A ExponentialHistogram contains
	// summary statistics for a population of values, it may optionally contain the
	// distribution of those values across a set of buckets.`,
	originFullName: "otlpmetrics.ExponentialHistogramDataPoint",
	fields: []Field{
		&SliceField{
			fieldName:   "Attributes",
			protoID:     1,
			protoType:   ProtoTypeMessage,
			returnSlice: mapStruct,
		},
		&TypedField{
			fieldName:       "StartTimestamp",
			protoID:         2,
			originFieldName: "StartTimeUnixNano",
			returnType:      timestampType,
		},
		&TypedField{
			fieldName:       "Timestamp",
			protoID:         3,
			originFieldName: "TimeUnixNano",
			returnType:      timestampType,
		},
		&PrimitiveField{
			fieldName: "Count",
			protoID:   4,
			protoType: ProtoTypeFixed64,
		},
		&OptionalPrimitiveField{
			fieldName: "Sum",
			protoID:   5,
			protoType: ProtoTypeDouble,
		},
		&PrimitiveField{
			fieldName: "Scale",
			protoID:   6,
			protoType: ProtoTypeSInt32,
		},
		&PrimitiveField{
			fieldName: "ZeroCount",
			protoID:   7,
			protoType: ProtoTypeFixed64,
		},
		&MessageField{
			fieldName:     "Positive",
			protoID:       8,
			returnMessage: bucketsValues,
		},
		&MessageField{
			fieldName:     "Negative",
			protoID:       9,
			returnMessage: bucketsValues,
		},
		&TypedField{
			fieldName: "Flags",
			protoID:   10,
			returnType: &TypedType{
				structName: "DataPointFlags",
				protoType:  ProtoTypeUint32,
				defaultVal: "0",
				testVal:    "1",
			},
		},
		&SliceField{
			fieldName:   "Exemplars",
			protoID:     11,
			protoType:   ProtoTypeMessage,
			returnSlice: exemplarSlice,
		},
		&OptionalPrimitiveField{
			fieldName: "Min",
			protoID:   12,
			protoType: ProtoTypeDouble,
		},
		&OptionalPrimitiveField{
			fieldName: "Max",
			protoID:   13,
			protoType: ProtoTypeDouble,
		},
		&PrimitiveField{
			fieldName: "ZeroThreshold",
			protoID:   14,
			protoType: ProtoTypeDouble,
		},
	},
}

var bucketsValues = &messageStruct{
	structName:     "ExponentialHistogramDataPointBuckets",
	description:    "// ExponentialHistogramDataPointBuckets are a set of bucket counts, encoded in a contiguous array of counts.",
	originFullName: "otlpmetrics.ExponentialHistogramDataPoint_Buckets",
	fields: []Field{
		&PrimitiveField{
			fieldName: "Offset",
			protoID:   1,
			protoType: ProtoTypeSInt32,
		},
		&SliceField{
			fieldName:   "BucketCounts",
			protoID:     2,
			protoType:   ProtoTypeUint64,
			returnSlice: uInt64Slice,
		},
	},
}

var summaryDataPointSlice = &messageSlice{
	structName:      "SummaryDataPointSlice",
	elementNullable: true,
	element:         summaryDataPoint,
}

var summaryDataPoint = &messageStruct{
	structName:     "SummaryDataPoint",
	description:    "// SummaryDataPoint is a single data point in a timeseries that describes the time-varying values of a Summary of double values.",
	originFullName: "otlpmetrics.SummaryDataPoint",
	fields: []Field{
		&SliceField{
			fieldName:   "Attributes",
			protoID:     7,
			protoType:   ProtoTypeMessage,
			returnSlice: mapStruct,
		},
		&TypedField{
			fieldName:       "StartTimestamp",
			originFieldName: "StartTimeUnixNano",
			protoID:         2,
			returnType:      timestampType,
		},
		&TypedField{
			fieldName:       "Timestamp",
			originFieldName: "TimeUnixNano",
			protoID:         3,
			returnType:      timestampType,
		},
		&PrimitiveField{
			fieldName: "Count",
			protoID:   4,
			protoType: ProtoTypeFixed64,
		},
		&PrimitiveField{
			fieldName: "Sum",
			protoID:   5,
			protoType: ProtoTypeDouble,
		},
		&SliceField{
			fieldName:   "QuantileValues",
			protoID:     6,
			protoType:   ProtoTypeMessage,
			returnSlice: quantileValuesSlice,
		},
		&TypedField{
			fieldName: "Flags",
			protoID:   8,
			returnType: &TypedType{
				structName: "DataPointFlags",
				protoType:  ProtoTypeUint32,
				defaultVal: "0",
				testVal:    "1",
			},
		},
	},
}

var quantileValuesSlice = &messageSlice{
	structName:      "SummaryDataPointValueAtQuantileSlice",
	elementNullable: true,
	element:         quantileValues,
}

var quantileValues = &messageStruct{
	structName:     "SummaryDataPointValueAtQuantile",
	description:    "// SummaryDataPointValueAtQuantile is a quantile value within a Summary data point.",
	originFullName: "otlpmetrics.SummaryDataPoint_ValueAtQuantile",
	fields: []Field{
		&PrimitiveField{
			fieldName: "Quantile",
			protoID:   1,
			protoType: ProtoTypeDouble,
		},
		&PrimitiveField{
			fieldName: "Value",
			protoID:   2,
			protoType: ProtoTypeDouble,
		},
	},
}

var exemplarSlice = &messageSlice{
	structName:      "ExemplarSlice",
	elementNullable: false,
	element:         exemplar,
}

var exemplar = &messageStruct{
	structName: "Exemplar",
	description: "// Exemplar is a sample input double measurement.\n//\n" +
		"// Exemplars also hold information about the environment when the measurement was recorded,\n" +
		"// for example the span and trace ID of the active span when the exemplar was recorded.",

	originFullName: "otlpmetrics.Exemplar",
	fields: []Field{
		&SliceField{
			fieldName:   "FilteredAttributes",
			protoID:     7,
			protoType:   ProtoTypeMessage,
			returnSlice: mapStruct,
		},
		&TypedField{
			fieldName:       "Timestamp",
			originFieldName: "TimeUnixNano",
			protoID:         2,
			returnType:      timestampType,
		},
		&OneOfField{
			typeName:        "ExemplarValueType",
			originFieldName: "Value",
			testValueIdx:    1, // Int
			values: []oneOfValue{
				&OneOfPrimitiveValue{
					fieldName:       "Double",
					originFieldName: "AsDouble",
					protoID:         3,
					protoType:       ProtoTypeDouble,
				},
				&OneOfPrimitiveValue{
					fieldName:       "Int",
					originFieldName: "AsInt",
					protoID:         6,
					protoType:       ProtoTypeSFixed64,
				},
			},
		},
		&TypedField{
			fieldName:       "SpanID",
			originFieldName: "SpanId",
			protoID:         4,
			returnType:      spanIDType,
		},
		&TypedField{
			fieldName:       "TraceID",
			originFieldName: "TraceId",
			protoID:         5,
			returnType:      traceIDType,
		},
	},
}

var aggregationTemporalityType = &TypedType{
	structName:  "AggregationTemporality",
	protoType:   ProtoTypeEnum,
	messageName: "otlpmetrics.AggregationTemporality",
	defaultVal:  "otlpmetrics.AggregationTemporality(0)",
	testVal:     "otlpmetrics.AggregationTemporality(1)",
}
