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

package internal

var metricsFile = &File{
	Name: "metrics",
	imports: []string{
		`"sort"`,
		``,
		`otlpmetrics "go.opentelemetry.io/collector/model/internal/data/protogen/metrics/v1"`,
	},
	testImports: []string{
		`"testing"`,
		``,
		`"github.com/stretchr/testify/assert"`,
		``,
		`otlpmetrics "go.opentelemetry.io/collector/model/internal/data/protogen/metrics/v1"`,
	},
	structs: []baseStruct{
		resourceMetricsSlice,
		resourceMetrics,
		instrumentationLibraryMetricsSlice,
		instrumentationLibraryMetrics,
		metricSlice,
		metric,
		doubleGauge,
		doubleSum,
		histogram,
		summary,
		numberDataPointSlice,
		numberDataPoint,
		histogramDataPointSlice,
		histogramDataPoint,
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
			fieldName:       "InstrumentationLibraryMetrics",
			originFieldName: "InstrumentationLibraryMetrics",
			returnSlice:     instrumentationLibraryMetricsSlice,
		},
	},
}

var instrumentationLibraryMetricsSlice = &sliceOfPtrs{
	structName: "InstrumentationLibraryMetricsSlice",
	element:    instrumentationLibraryMetrics,
}

var instrumentationLibraryMetrics = &messageValueStruct{
	structName:     "InstrumentationLibraryMetrics",
	description:    "// InstrumentationLibraryMetrics is a collection of metrics from a LibraryInstrumentation.",
	originFullName: "otlpmetrics.InstrumentationLibraryMetrics",
	fields: []baseField{
		instrumentationLibraryField,
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
		oneofDataField,
	},
}

var doubleGauge = &messageValueStruct{
	structName:     "Gauge",
	description:    "// Gauge represents the type of a double scalar metric that always exports the \"current value\" for every data point.",
	originFullName: "otlpmetrics.Gauge",
	fields: []baseField{
		&sliceField{
			fieldName:       "DataPoints",
			originFieldName: "DataPoints",
			returnSlice:     numberDataPointSlice,
		},
	},
}

var doubleSum = &messageValueStruct{
	structName:     "Sum",
	description:    "// Sum represents the type of a numeric double scalar metric that is calculated as a sum of all reported measurements over a time interval.",
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
		&numberField{
			fields: []*oneOfPrimitiveValue{
				{
					originFullName:  "otlpmetrics.NumberDataPoint",
					name:            "DoubleVal",
					originFieldName: "Value",
					returnType:      "float64",
					defaultVal:      "float64(0.0)",
					testVal:         "float64(17.13)",
					fieldType:       "Double",
				},
				{
					originFullName:  "otlpmetrics.NumberDataPoint",
					name:            "IntVal",
					originFieldName: "Value",
					returnType:      "int64",
					defaultVal:      "int64(0)",
					testVal:         "int64(17)",
					fieldType:       "Int",
				},
			},
		},
		exemplarsField,
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
		doubleSumField,
		bucketCountsField,
		explicitBoundsField,
		exemplarsField,
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
		&numberField{
			fields: []*oneOfPrimitiveValue{
				{
					originFullName:  "otlpmetrics.Exemplar",
					name:            "DoubleVal",
					originFieldName: "Value",
					returnType:      "float64",
					defaultVal:      "float64(0.0)",
					testVal:         "float64(17.13)",
					fieldType:       "Double",
				},
				{
					originFullName:  "otlpmetrics.Exemplar",
					name:            "IntVal",
					originFieldName: "Value",
					returnType:      "int64",
					defaultVal:      "int64(0)",
					testVal:         "int64(17)",
					fieldType:       "Int",
				},
			},
		},
		&sliceField{
			fieldName:       "FilteredAttributes",
			originFieldName: "FilteredAttributes",
			returnSlice:     attributeMap,
		},
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

var bucketCountsField = &primitiveField{
	fieldName:       "BucketCounts",
	originFieldName: "BucketCounts",
	returnType:      "[]uint64",
	defaultVal:      "[]uint64(nil)",
	testVal:         "[]uint64{1, 2, 3}",
}

var explicitBoundsField = &primitiveField{
	fieldName:       "ExplicitBounds",
	originFieldName: "ExplicitBounds",
	returnType:      "[]float64",
	defaultVal:      "[]float64(nil)",
	testVal:         "[]float64{1, 2, 3}",
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
	returnType:      "AggregationTemporality",
	rawType:         "otlpmetrics.AggregationTemporality",
	defaultVal:      "AggregationTemporalityUnspecified",
	testVal:         "AggregationTemporalityCumulative",
}

var oneofDataField = &oneofField{
	copyFuncName:    "copyData",
	originFieldName: "Data",
	testVal:         "&otlpmetrics.Metric_Gauge{Gauge: &otlpmetrics.Gauge{}}",
	fillTestName:    "Gauge",
}
