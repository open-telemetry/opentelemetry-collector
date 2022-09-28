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

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

var metricsFile = &File{
	Name:        "metrics",
	PackageName: "pmetric",
	imports: []string{
		`"sort"`,
		``,
		`otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"`,
	},
	testImports: []string{
		`"testing"`,
		``,
		`"github.com/stretchr/testify/assert"`,
		``,
		`"go.opentelemetry.io/collector/pdata/internal"`,
		`"go.opentelemetry.io/collector/pdata/internal/data"`,
		`otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"`,
		`"go.opentelemetry.io/collector/pdata/pcommon"`,
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

var resourceMetrics = &messageValueStruct{
	structName:     "ResourceMetrics",
	description:    "// ResourceMetrics is a collection of metrics from a Resource.",
	originFullName: "otlpmetrics.ResourceMetrics",
	fields: []baseField{
		resourceField,
		schemaURLField,
		&sliceField{
			fieldName:       "ScopeMetrics",
			originFieldName: "ScopeMetrics",
			returnSlice:     scopeMetricsSlice,
		},
	},
}

var scopeMetricsSlice = &sliceOfPtrs{
	structName: "ScopeMetricsSlice",
	element:    scopeMetrics,
}

var scopeMetrics = &messageValueStruct{
	structName:     "ScopeMetrics",
	description:    "// ScopeMetrics is a collection of metrics from a LibraryInstrumentation.",
	originFullName: "otlpmetrics.ScopeMetrics",
	fields: []baseField{
		scopeField,
		schemaURLField,
		&sliceField{
			fieldName:       "Metrics",
			originFieldName: "Metrics",
			returnSlice:     metricSlice,
		},
	},
}

var metricSlice = &sliceOfPtrs{
	structName: "MetricSlice",
	element:    metric,
}

var metric = &messageValueStruct{
	structName: "Metric",
	description: "// Metric represents one metric as a collection of datapoints.\n" +
		"// See Metric definition in OTLP: https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/metrics/v1/metrics.proto",
	originFullName: "otlpmetrics.Metric",
	fields: []baseField{
		nameField,
		&primitiveField{
			fieldName:       "Description",
			originFieldName: "Description",
			returnType:      "string",
			defaultVal:      `""`,
			testVal:         `"test_description"`,
		},
		&primitiveField{
			fieldName:       "Unit",
			originFieldName: "Unit",
			returnType:      "string",
			defaultVal:      `""`,
			testVal:         `"1"`,
		},
		&oneOfField{
			typeName:                   "MetricType",
			originFieldName:            "Data",
			originTypePrefix:           "otlpmetrics.Metric_",
			testValueIdx:               1, // Sum
			omitOriginFieldNameInNames: true,
			values: []oneOfValue{
				&oneOfMessageValue{
					fieldName:              "Gauge",
					originFieldName:        "Gauge",
					originFieldPackageName: "otlpmetrics",
					returnMessage:          gauge,
				},
				&oneOfMessageValue{
					fieldName:              "Sum",
					originFieldName:        "Sum",
					originFieldPackageName: "otlpmetrics",
					returnMessage:          sum,
				},
				&oneOfMessageValue{
					fieldName:              "Histogram",
					originFieldName:        "Histogram",
					originFieldPackageName: "otlpmetrics",
					returnMessage:          histogram,
				},
				&oneOfMessageValue{
					fieldName:              "ExponentialHistogram",
					originFieldName:        "ExponentialHistogram",
					originFieldPackageName: "otlpmetrics",
					returnMessage:          exponentialHistogram,
				},
				&oneOfMessageValue{
					fieldName:              "Summary",
					originFieldName:        "Summary",
					originFieldPackageName: "otlpmetrics",
					returnMessage:          summary,
				},
			},
		},
	},
}

var gauge = &messageValueStruct{
	structName:     "Gauge",
	description:    "// Gauge represents the type of a numeric metric that always exports the \"current value\" for every data point.",
	originFullName: "otlpmetrics.Gauge",
	fields: []baseField{
		&sliceField{
			fieldName:       "DataPoints",
			originFieldName: "DataPoints",
			returnSlice:     numberDataPointSlice,
		},
	},
}

var sum = &messageValueStruct{
	structName:     "Sum",
	description:    "// Sum represents the type of a numeric metric that is calculated as a sum of all reported measurements over a time interval.",
	originFullName: "otlpmetrics.Sum",
	fields: []baseField{
		aggregationTemporalityField,
		isMonotonicField,
		&sliceField{
			fieldName:       "DataPoints",
			originFieldName: "DataPoints",
			returnSlice:     numberDataPointSlice,
		},
	},
}

var histogram = &messageValueStruct{
	structName:     "Histogram",
	description:    "// Histogram represents the type of a metric that is calculated by aggregating as a Histogram of all reported measurements over a time interval.",
	originFullName: "otlpmetrics.Histogram",
	fields: []baseField{
		aggregationTemporalityField,
		&sliceField{
			fieldName:       "DataPoints",
			originFieldName: "DataPoints",
			returnSlice:     histogramDataPointSlice,
		},
	},
}

var exponentialHistogram = &messageValueStruct{
	structName: "ExponentialHistogram",
	description: `// ExponentialHistogram represents the type of a metric that is calculated by aggregating
	// as a ExponentialHistogram of all reported double measurements over a time interval.`,
	originFullName: "otlpmetrics.ExponentialHistogram",
	fields: []baseField{
		aggregationTemporalityField,
		&sliceField{
			fieldName:       "DataPoints",
			originFieldName: "DataPoints",
			returnSlice:     exponentialHistogramDataPointSlice,
		},
	},
}

var summary = &messageValueStruct{
	structName:     "Summary",
	description:    "// Summary represents the type of a metric that is calculated by aggregating as a Summary of all reported double measurements over a time interval.",
	originFullName: "otlpmetrics.Summary",
	fields: []baseField{
		&sliceField{
			fieldName:       "DataPoints",
			originFieldName: "DataPoints",
			returnSlice:     summaryDataPointSlice,
		},
	},
}

var numberDataPointSlice = &sliceOfPtrs{
	structName: "NumberDataPointSlice",
	element:    numberDataPoint,
}

var numberDataPoint = &messageValueStruct{
	structName:     "NumberDataPoint",
	description:    "// NumberDataPoint is a single data point in a timeseries that describes the time-varying value of a number metric.",
	originFullName: "otlpmetrics.NumberDataPoint",
	fields: []baseField{
		attributes,
		startTimeField,
		timeField,
		&oneOfField{
			typeName:         "NumberDataPointValueType",
			originFieldName:  "Value",
			originTypePrefix: "otlpmetrics.NumberDataPoint_",
			testValueIdx:     0, // Double
			values: []oneOfValue{
				&oneOfPrimitiveValue{
					fieldName:       "Double",
					originFieldName: "AsDouble",
					returnType:      "float64",
					defaultVal:      "float64(0.0)",
					testVal:         "float64(17.13)",
				},
				&oneOfPrimitiveValue{
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

var histogramDataPoint = &messageValueStruct{
	structName:     "HistogramDataPoint",
	description:    "// HistogramDataPoint is a single data point in a timeseries that describes the time-varying values of a Histogram of values.",
	originFullName: "otlpmetrics.HistogramDataPoint",
	fields: []baseField{
		attributes,
		startTimeField,
		timeField,
		countField,
		optionalDoubleSumField,
		bucketCountsField,
		explicitBoundsField,
		exemplarsField,
		dataPointFlagsField,
		&optionalPrimitiveValue{
			fieldName:        "Min",
			originFieldName:  "Min",
			originTypePrefix: "otlpmetrics.HistogramDataPoint_",
			returnType:       "float64",
			defaultVal:       "float64(0.0)",
			testVal:          "float64(9.23)",
		},
		&optionalPrimitiveValue{
			fieldName:        "Max",
			originFieldName:  "Max",
			originTypePrefix: "otlpmetrics.HistogramDataPoint_",
			returnType:       "float64",
			defaultVal:       "float64(0.0)",
			testVal:          "float64(182.55)",
		},
	},
}

var exponentialHistogramDataPointSlice = &sliceOfPtrs{
	structName: "ExponentialHistogramDataPointSlice",
	element:    exponentialHistogramDataPoint,
}

var exponentialHistogramDataPoint = &messageValueStruct{
	structName: "ExponentialHistogramDataPoint",
	description: `// ExponentialHistogramDataPoint is a single data point in a timeseries that describes the
	// time-varying values of a ExponentialHistogram of double values. A ExponentialHistogram contains
	// summary statistics for a population of values, it may optionally contain the
	// distribution of those values across a set of buckets.`,
	originFullName: "otlpmetrics.ExponentialHistogramDataPoint",
	fields: []baseField{
		attributes,
		startTimeField,
		timeField,
		countField,
		&optionalPrimitiveValue{
			fieldName:        "Sum",
			originFieldName:  "Sum",
			originTypePrefix: "otlpmetrics.ExponentialHistogramDataPoint_",
			returnType:       "float64",
			defaultVal:       "float64(0.0)",
			testVal:          "float64(17.13)",
		},
		&primitiveField{
			fieldName:       "Scale",
			originFieldName: "Scale",
			returnType:      "int32",
			defaultVal:      "int32(0)",
			testVal:         "int32(4)",
		},
		&primitiveField{
			fieldName:       "ZeroCount",
			originFieldName: "ZeroCount",
			returnType:      "uint64",
			defaultVal:      "uint64(0)",
			testVal:         "uint64(201)",
		},
		&messageValueField{
			fieldName:       "Positive",
			originFieldName: "Positive",
			returnMessage:   bucketsValues,
		},
		&messageValueField{
			fieldName:       "Negative",
			originFieldName: "Negative",
			returnMessage:   bucketsValues,
		},
		exemplarsField,
		dataPointFlagsField,
		&optionalPrimitiveValue{
			fieldName:        "Min",
			originFieldName:  "Min",
			originTypePrefix: "otlpmetrics.ExponentialHistogramDataPoint_",
			returnType:       "float64",
			defaultVal:       "float64(0.0)",
			testVal:          "float64(9.23)",
		},
		&optionalPrimitiveValue{
			fieldName:        "Max",
			originFieldName:  "Max",
			originTypePrefix: "otlpmetrics.ExponentialHistogramDataPoint_",
			returnType:       "float64",
			defaultVal:       "float64(0.0)",
			testVal:          "float64(182.55)",
		},
	},
}

var bucketsValues = &messageValueStruct{
	structName:     "Buckets",
	description:    "// Buckets are a set of bucket counts, encoded in a contiguous array of counts.",
	originFullName: "otlpmetrics.ExponentialHistogramDataPoint_Buckets",
	fields: []baseField{
		&primitiveField{
			fieldName:       "Offset",
			originFieldName: "Offset",
			returnType:      "int32",
			defaultVal:      "int32(0)",
			testVal:         "int32(909)",
		},
		bucketCountsField,
	},
}

var summaryDataPointSlice = &sliceOfPtrs{
	structName: "SummaryDataPointSlice",
	element:    summaryDataPoint,
}

var summaryDataPoint = &messageValueStruct{
	structName:     "SummaryDataPoint",
	description:    "// SummaryDataPoint is a single data point in a timeseries that describes the time-varying values of a Summary of double values.",
	originFullName: "otlpmetrics.SummaryDataPoint",
	fields: []baseField{
		attributes,
		startTimeField,
		timeField,
		countField,
		doubleSumField,
		&sliceField{
			fieldName:       "QuantileValues",
			originFieldName: "QuantileValues",
			returnSlice:     quantileValuesSlice,
		},
		dataPointFlagsField,
	},
}

var quantileValuesSlice = &sliceOfPtrs{
	structName: "ValueAtQuantileSlice",
	element:    quantileValues,
}

var quantileValues = &messageValueStruct{
	structName:     "ValueAtQuantile",
	description:    "// ValueAtQuantile is a quantile value within a Summary data point.",
	originFullName: "otlpmetrics.SummaryDataPoint_ValueAtQuantile",
	fields: []baseField{
		quantileField,
		valueFloat64Field,
	},
}

var exemplarSlice = &sliceOfValues{
	structName: "ExemplarSlice",
	element:    exemplar,
}

var exemplar = &messageValueStruct{
	structName: "Exemplar",
	description: "// Exemplar is a sample input double measurement.\n//\n" +
		"// Exemplars also hold information about the environment when the measurement was recorded,\n" +
		"// for example the span and trace ID of the active span when the exemplar was recorded.",

	originFullName: "otlpmetrics.Exemplar",
	fields: []baseField{
		timeField,
		&oneOfField{
			typeName:         "ExemplarValueType",
			originFieldName:  "Value",
			originTypePrefix: "otlpmetrics.Exemplar_",
			testValueIdx:     1, // Int
			values: []oneOfValue{
				&oneOfPrimitiveValue{
					fieldName:       "Double",
					originFieldName: "AsDouble",
					returnType:      "float64",
					defaultVal:      "float64(0.0)",
					testVal:         "float64(17.13)",
				},
				&oneOfPrimitiveValue{
					fieldName:       "Int",
					originFieldName: "AsInt",
					returnType:      "int64",
					defaultVal:      "int64(0)",
					testVal:         "int64(17)",
				},
			},
		},
		&sliceField{
			fieldName:       "FilteredAttributes",
			originFieldName: "FilteredAttributes",
			returnSlice:     mapStruct,
		},
		traceIDField,
		spanIDField,
	},
}

var dataPointFlagsField = &primitiveTypedField{
	fieldName:       "Flags",
	originFieldName: "Flags",
	returnType: &primitiveType{
		structName: "MetricDataPointFlags",
		rawType:    "uint32",
		defaultVal: "0",
		testVal:    "1",
	},
}

var exemplarsField = &sliceField{
	fieldName:       "Exemplars",
	originFieldName: "Exemplars",
	returnSlice:     exemplarSlice,
}

var countField = &primitiveField{
	fieldName:       "Count",
	originFieldName: "Count",
	returnType:      "uint64",
	defaultVal:      "uint64(0)",
	testVal:         "uint64(17)",
}

var doubleSumField = &primitiveField{
	fieldName:       "Sum",
	originFieldName: "Sum",
	returnType:      "float64",
	defaultVal:      "float64(0.0)",
	testVal:         "float64(17.13)",
}

var valueFloat64Field = &primitiveField{
	fieldName:       "Value",
	originFieldName: "Value",
	returnType:      "float64",
	defaultVal:      "float64(0.0)",
	testVal:         "float64(17.13)",
}

var bucketCountsField = &primitiveSliceField{
	fieldName:         "BucketCounts",
	originFieldName:   "BucketCounts",
	returnType:        "UInt64Slice",
	returnPackageName: "pcommon",
	defaultVal:        "[]uint64(nil)",
	rawType:           "[]uint64",
	testVal:           "[]uint64{1, 2, 3}",
}

var explicitBoundsField = &primitiveSliceField{
	fieldName:         "ExplicitBounds",
	originFieldName:   "ExplicitBounds",
	returnType:        "Float64Slice",
	returnPackageName: "pcommon",
	defaultVal:        "[]float64(nil)",
	rawType:           "[]float64",
	testVal:           "[]float64{1, 2, 3}",
}

var quantileField = &primitiveField{
	fieldName:       "Quantile",
	originFieldName: "Quantile",
	returnType:      "float64",
	defaultVal:      "float64(0.0)",
	testVal:         "float64(17.13)",
}

var isMonotonicField = &primitiveField{
	fieldName:       "IsMonotonic",
	originFieldName: "IsMonotonic",
	returnType:      "bool",
	defaultVal:      "false",
	testVal:         "true",
}

var aggregationTemporalityField = &primitiveTypedField{
	fieldName:       "AggregationTemporality",
	originFieldName: "AggregationTemporality",
	returnType: &primitiveType{
		structName: "MetricAggregationTemporality",
		rawType:    "otlpmetrics.AggregationTemporality",
		defaultVal: "otlpmetrics.AggregationTemporality(0)",
		testVal:    "otlpmetrics.AggregationTemporality(1)",
	},
}

var optionalDoubleSumField = &optionalPrimitiveValue{
	fieldName:        "Sum",
	originFieldName:  "Sum",
	originTypePrefix: "otlpmetrics.HistogramDataPoint_",
	returnType:       "float64",
	defaultVal:       "float64(0.0)",
	testVal:          "float64(17.13)",
}
