// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

var pmetric = &Package{
	info: &PackageInfo{
		name: "pmetric",
		path: "pmetric",
		imports: []string{
			`"iter"`,
			`"sort"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/data"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
		testImports: []string{
			`"testing"`,
			`"unsafe"`,
			``,
			`"github.com/stretchr/testify/assert"`,
			`"github.com/stretchr/testify/require"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/data"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
	},
	structs: []baseStruct{
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

var resourceMetricsSlice = &sliceOfPtrs{
	structName: "ResourceMetricsSlice",
	element:    resourceMetrics,
}

var resourceMetrics = &messageStruct{
	structName:     "ResourceMetrics",
	description:    "// ResourceMetrics is a collection of metrics from a Resource.",
	originFullName: "otlpmetrics.ResourceMetrics",
	fields: []Field{
		resourceField,
		schemaURLField,
		&SliceField{
			fieldName:   "ScopeMetrics",
			returnSlice: scopeMetricsSlice,
		},
	},
}

var scopeMetricsSlice = &sliceOfPtrs{
	structName: "ScopeMetricsSlice",
	element:    scopeMetrics,
}

var scopeMetrics = &messageStruct{
	structName:     "ScopeMetrics",
	description:    "// ScopeMetrics is a collection of metrics from a LibraryInstrumentation.",
	originFullName: "otlpmetrics.ScopeMetrics",
	fields: []Field{
		scopeField,
		schemaURLField,
		&SliceField{
			fieldName:   "Metrics",
			returnSlice: metricSlice,
		},
	},
}

var metricSlice = &sliceOfPtrs{
	structName: "MetricSlice",
	element:    metric,
}

var metric = &messageStruct{
	structName: "Metric",
	description: "// Metric represents one metric as a collection of datapoints.\n" +
		"// See Metric definition in OTLP: https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/metrics/v1/metrics.proto",
	originFullName: "otlpmetrics.Metric",
	fields: []Field{
		nameField,
		&PrimitiveField{
			fieldName:  "Description",
			returnType: "string",
			defaultVal: `""`,
			testVal:    `"test_description"`,
		},
		&PrimitiveField{
			fieldName:  "Unit",
			returnType: "string",
			defaultVal: `""`,
			testVal:    `"1"`,
		},
		&SliceField{
			fieldName:   "Metadata",
			returnSlice: mapStruct,
		},
		&OneOfField{
			typeName:                   "MetricType",
			originFieldName:            "Data",
			testValueIdx:               1, // Sum
			omitOriginFieldNameInNames: true,
			values: []oneOfValue{
				&OneOfMessageValue{
					fieldName:              "Gauge",
					originFieldPackageName: "otlpmetrics",
					returnMessage:          gauge,
				},
				&OneOfMessageValue{
					fieldName:              "Sum",
					originFieldPackageName: "otlpmetrics",
					returnMessage:          sum,
				},
				&OneOfMessageValue{
					fieldName:              "Histogram",
					originFieldPackageName: "otlpmetrics",
					returnMessage:          histogram,
				},
				&OneOfMessageValue{
					fieldName:              "ExponentialHistogram",
					originFieldPackageName: "otlpmetrics",
					returnMessage:          exponentialHistogram,
				},
				&OneOfMessageValue{
					fieldName:              "Summary",
					originFieldPackageName: "otlpmetrics",
					returnMessage:          summary,
				},
			},
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
			returnSlice: numberDataPointSlice,
		},
	},
}

var sum = &messageStruct{
	structName:     "Sum",
	description:    "// Sum represents the type of a numeric metric that is calculated as a sum of all reported measurements over a time interval.",
	originFullName: "otlpmetrics.Sum",
	fields: []Field{
		aggregationTemporalityField,
		isMonotonicField,
		&SliceField{
			fieldName:   "DataPoints",
			returnSlice: numberDataPointSlice,
		},
	},
}

var histogram = &messageStruct{
	structName:     "Histogram",
	description:    "// Histogram represents the type of a metric that is calculated by aggregating as a Histogram of all reported measurements over a time interval.",
	originFullName: "otlpmetrics.Histogram",
	fields: []Field{
		aggregationTemporalityField,
		&SliceField{
			fieldName:   "DataPoints",
			returnSlice: histogramDataPointSlice,
		},
	},
}

var exponentialHistogram = &messageStruct{
	structName: "ExponentialHistogram",
	description: `// ExponentialHistogram represents the type of a metric that is calculated by aggregating
	// as a ExponentialHistogram of all reported double measurements over a time interval.`,
	originFullName: "otlpmetrics.ExponentialHistogram",
	fields: []Field{
		aggregationTemporalityField,
		&SliceField{
			fieldName:   "DataPoints",
			returnSlice: exponentialHistogramDataPointSlice,
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
			returnSlice: summaryDataPointSlice,
		},
	},
}

var numberDataPointSlice = &sliceOfPtrs{
	structName: "NumberDataPointSlice",
	element:    numberDataPoint,
}

var numberDataPoint = &messageStruct{
	structName:     "NumberDataPoint",
	description:    "// NumberDataPoint is a single data point in a timeseries that describes the time-varying value of a number metric.",
	originFullName: "otlpmetrics.NumberDataPoint",
	fields: []Field{
		attributes,
		startTimeField,
		timeField,
		&OneOfField{
			typeName:        "NumberDataPointValueType",
			originFieldName: "Value",
			testValueIdx:    0, // Double
			values: []oneOfValue{
				&OneOfPrimitiveValue{
					fieldName:       "Double",
					originFieldName: "AsDouble",
					returnType:      "float64",
					defaultVal:      "float64(0.0)",
					testVal:         "float64(17.13)",
				},
				&OneOfPrimitiveValue{
					fieldName:       "Int",
					originFieldName: "AsInt",
					returnType:      "int64",
					defaultVal:      "int64(0)",
					testVal:         "int64(17)",
				},
			},
		},
		exemplarsField,
		dataPointFlagsField,
	},
}

var histogramDataPointSlice = &sliceOfPtrs{
	structName: "HistogramDataPointSlice",
	element:    histogramDataPoint,
}

var histogramDataPoint = &messageStruct{
	structName:     "HistogramDataPoint",
	description:    "// HistogramDataPoint is a single data point in a timeseries that describes the time-varying values of a Histogram of values.",
	originFullName: "otlpmetrics.HistogramDataPoint",
	fields: []Field{
		attributes,
		startTimeField,
		timeField,
		countField,
		&SliceField{
			fieldName:   "BucketCounts",
			returnSlice: uInt64Slice,
		},
		&SliceField{
			fieldName:   "ExplicitBounds",
			returnSlice: float64Slice,
		},
		exemplarsField,
		dataPointFlagsField,
		sumField,
		minField,
		maxField,
	},
}

var exponentialHistogramDataPointSlice = &sliceOfPtrs{
	structName: "ExponentialHistogramDataPointSlice",
	element:    exponentialHistogramDataPoint,
}

var exponentialHistogramDataPoint = &messageStruct{
	structName: "ExponentialHistogramDataPoint",
	description: `// ExponentialHistogramDataPoint is a single data point in a timeseries that describes the
	// time-varying values of a ExponentialHistogram of double values. A ExponentialHistogram contains
	// summary statistics for a population of values, it may optionally contain the
	// distribution of those values across a set of buckets.`,
	originFullName: "otlpmetrics.ExponentialHistogramDataPoint",
	fields: []Field{
		attributes,
		startTimeField,
		timeField,
		countField,
		&PrimitiveField{
			fieldName:  "Scale",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(4)",
		},
		&PrimitiveField{
			fieldName:  "ZeroCount",
			returnType: "uint64",
			defaultVal: "uint64(0)",
			testVal:    "uint64(201)",
		},
		&MessageField{
			fieldName:     "Positive",
			returnMessage: bucketsValues,
		},
		&MessageField{
			fieldName:     "Negative",
			returnMessage: bucketsValues,
		},
		exemplarsField,
		dataPointFlagsField,
		sumField,
		minField,
		maxField,
		&PrimitiveField{
			fieldName:  "ZeroThreshold",
			returnType: "float64",
			defaultVal: "float64(0.0)",
			testVal:    "float64(0.5)",
		},
	},
}

var bucketsValues = &messageStruct{
	structName:     "ExponentialHistogramDataPointBuckets",
	description:    "// ExponentialHistogramDataPointBuckets are a set of bucket counts, encoded in a contiguous array of counts.",
	originFullName: "otlpmetrics.ExponentialHistogramDataPoint_Buckets",
	fields: []Field{
		&PrimitiveField{
			fieldName:  "Offset",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(909)",
		},
		&SliceField{
			fieldName:   "BucketCounts",
			returnSlice: uInt64Slice,
		},
	},
}

var summaryDataPointSlice = &sliceOfPtrs{
	structName: "SummaryDataPointSlice",
	element:    summaryDataPoint,
}

var summaryDataPoint = &messageStruct{
	structName:     "SummaryDataPoint",
	description:    "// SummaryDataPoint is a single data point in a timeseries that describes the time-varying values of a Summary of double values.",
	originFullName: "otlpmetrics.SummaryDataPoint",
	fields: []Field{
		attributes,
		startTimeField,
		timeField,
		countField,
		doubleSumField,
		&SliceField{
			fieldName:   "QuantileValues",
			returnSlice: quantileValuesSlice,
		},
		dataPointFlagsField,
	},
}

var quantileValuesSlice = &sliceOfPtrs{
	structName: "SummaryDataPointValueAtQuantileSlice",
	element:    quantileValues,
}

var quantileValues = &messageStruct{
	structName:     "SummaryDataPointValueAtQuantile",
	description:    "// SummaryDataPointValueAtQuantile is a quantile value within a Summary data point.",
	originFullName: "otlpmetrics.SummaryDataPoint_ValueAtQuantile",
	fields: []Field{
		quantileField,
		valueFloat64Field,
	},
}

var exemplarSlice = &sliceOfValues{
	structName: "ExemplarSlice",
	element:    exemplar,
}

var exemplar = &messageStruct{
	structName: "Exemplar",
	description: "// Exemplar is a sample input double measurement.\n//\n" +
		"// Exemplars also hold information about the environment when the measurement was recorded,\n" +
		"// for example the span and trace ID of the active span when the exemplar was recorded.",

	originFullName: "otlpmetrics.Exemplar",
	fields: []Field{
		timeField,
		&OneOfField{
			typeName:        "ExemplarValueType",
			originFieldName: "Value",
			testValueIdx:    1, // Int
			values: []oneOfValue{
				&OneOfPrimitiveValue{
					fieldName:       "Double",
					originFieldName: "AsDouble",
					returnType:      "float64",
					defaultVal:      "float64(0.0)",
					testVal:         "float64(17.13)",
				},
				&OneOfPrimitiveValue{
					fieldName:       "Int",
					originFieldName: "AsInt",
					returnType:      "int64",
					defaultVal:      "int64(0)",
					testVal:         "int64(17)",
				},
			},
		},
		&SliceField{
			fieldName:   "FilteredAttributes",
			returnSlice: mapStruct,
		},
		traceIDField,
		spanIDField,
	},
}

var dataPointFlagsField = &TypedField{
	fieldName: "Flags",
	returnType: &TypedType{
		structName: "DataPointFlags",
		rawType:    "uint32",
		defaultVal: "0",
		testVal:    "1",
	},
}

var exemplarsField = &SliceField{
	fieldName:   "Exemplars",
	returnSlice: exemplarSlice,
}

var countField = &PrimitiveField{
	fieldName:  "Count",
	returnType: "uint64",
	defaultVal: "uint64(0)",
	testVal:    "uint64(17)",
}

var doubleSumField = &PrimitiveField{
	fieldName:  "Sum",
	returnType: "float64",
	defaultVal: "float64(0.0)",
	testVal:    "float64(17.13)",
}

var valueFloat64Field = &PrimitiveField{
	fieldName:  "Value",
	returnType: "float64",
	defaultVal: "float64(0.0)",
	testVal:    "float64(17.13)",
}

var quantileField = &PrimitiveField{
	fieldName:  "Quantile",
	returnType: "float64",
	defaultVal: "float64(0.0)",
	testVal:    "float64(17.13)",
}

var isMonotonicField = &PrimitiveField{
	fieldName:  "IsMonotonic",
	returnType: "bool",
	defaultVal: "false",
	testVal:    "true",
}

var aggregationTemporalityField = &TypedField{
	fieldName: "AggregationTemporality",
	returnType: &TypedType{
		structName: "AggregationTemporality",
		rawType:    "otlpmetrics.AggregationTemporality",
		isEnum:     true,
		defaultVal: "otlpmetrics.AggregationTemporality(0)",
		testVal:    "otlpmetrics.AggregationTemporality(1)",
	},
}

var sumField = &OptionalPrimitiveField{
	fieldName:  "Sum",
	returnType: "float64",
	defaultVal: "float64(0.0)",
	testVal:    "float64(17.13)",
}

var minField = &OptionalPrimitiveField{
	fieldName:  "Min",
	returnType: "float64",
	defaultVal: "float64(0.0)",
	testVal:    "float64(9.23)",
}

var maxField = &OptionalPrimitiveField{
	fieldName:  "Max",
	returnType: "float64",
	defaultVal: "float64(0.0)",
	testVal:    "float64(182.55)",
}
