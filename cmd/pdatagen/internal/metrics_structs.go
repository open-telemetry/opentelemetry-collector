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
		`otlpmetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"`,
	},
	testImports: []string{
		`"testing"`,
		``,
		`"github.com/stretchr/testify/assert"`,
		``,
		`otlpmetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"`,
	},
	structs: []baseStruct{
		resourceMetricsSlice,
		resourceMetrics,
		instrumentationLibraryMetricsSlice,
		instrumentationLibraryMetrics,
		metricSlice,
		metric,
		intGauge,
		doubleGauge,
		intSum,
		doubleSum,
		intHistogram,
		doubleHistogram,
		intDataPointSlice,
		intDataPoint,
		doubleDataPointSlice,
		doubleDataPoint,
		intHistogramDataPointSlice,
		intHistogramDataPoint,
		doubleHistogramDataPointSlice,
		doubleHistogramDataPoint,
		intExemplarSlice,
		intExemplar,
		doubleExemplarSlice,
		doubleExemplar,
	},
}

var resourceMetricsSlice = &sliceStruct{
	structName: "ResourceMetricsSlice",
	element:    resourceMetrics,
}

var resourceMetrics = &messageStruct{
	structName:     "ResourceMetrics",
	description:    "// InstrumentationLibraryMetrics is a collection of metrics from a LibraryInstrumentation.",
	originFullName: "otlpmetrics.ResourceMetrics",
	fields: []baseField{
		resourceField,
		&sliceField{
			fieldMame:       "InstrumentationLibraryMetrics",
			originFieldName: "InstrumentationLibraryMetrics",
			returnSlice:     instrumentationLibraryMetricsSlice,
		},
	},
}

var instrumentationLibraryMetricsSlice = &sliceStruct{
	structName: "InstrumentationLibraryMetricsSlice",
	element:    instrumentationLibraryMetrics,
}

var instrumentationLibraryMetrics = &messageStruct{
	structName:     "InstrumentationLibraryMetrics",
	description:    "// InstrumentationLibraryMetrics is a collection of metrics from a LibraryInstrumentation.",
	originFullName: "otlpmetrics.InstrumentationLibraryMetrics",
	fields: []baseField{
		instrumentationLibraryField,
		&sliceField{
			fieldMame:       "Metrics",
			originFieldName: "Metrics",
			returnSlice:     metricSlice,
		},
	},
}

var metricSlice = &sliceStruct{
	structName: "MetricSlice",
	element:    metric,
}

var metric = &messageStruct{
	structName: "Metric",
	description: "// Metric represents one metric as a collection of datapoints.\n" +
		"// See Metric definition in OTLP: https://github.com/open-telemetry/opentelemetry-proto/blob/master/opentelemetry/proto/metrics/v1/metrics.proto",
	originFullName: "otlpmetrics.Metric",
	fields: []baseField{
		nameField,
		&primitiveField{
			fieldMame:       "Description",
			originFieldName: "Description",
			returnType:      "string",
			defaultVal:      `""`,
			testVal:         `"test_description"`,
		},
		&primitiveField{
			fieldMame:       "Unit",
			originFieldName: "Unit",
			returnType:      "string",
			defaultVal:      `""`,
			testVal:         `"1"`,
		},
	},
}

var intGauge = &messageStruct{
	structName:     "IntGauge",
	description:    "// IntGauge represents the type of a int scalar metric that always exports the \"current value\" for every data point.",
	originFullName: "otlpmetrics.IntGauge",
	fields: []baseField{
		&sliceField{
			fieldMame:       "DataPoints",
			originFieldName: "DataPoints",
			returnSlice:     intDataPointSlice,
		},
	},
}

var doubleGauge = &messageStruct{
	structName:     "DoubleGauge",
	description:    "// DoubleGauge represents the type of a double scalar metric that always exports the \"current value\" for every data point.",
	originFullName: "otlpmetrics.DoubleGauge",
	fields: []baseField{
		&sliceField{
			fieldMame:       "DataPoints",
			originFieldName: "DataPoints",
			returnSlice:     doubleDataPointSlice,
		},
	},
}

var intSum = &messageStruct{
	structName:     "IntSum",
	description:    "// IntSum represents the type of a numeric int scalar metric that is calculated as a sum of all reported measurements over a time interval.",
	originFullName: "otlpmetrics.IntSum",
	fields: []baseField{
		aggregationTemporalityField,
		isMonotonicField,
		&sliceField{
			fieldMame:       "DataPoints",
			originFieldName: "DataPoints",
			returnSlice:     intDataPointSlice,
		},
	},
}

var doubleSum = &messageStruct{
	structName:     "DoubleSum",
	description:    "// DoubleSum represents the type of a numeric double scalar metric that is calculated as a sum of all reported measurements over a time interval.",
	originFullName: "otlpmetrics.DoubleSum",
	fields: []baseField{
		aggregationTemporalityField,
		isMonotonicField,
		&sliceField{
			fieldMame:       "DataPoints",
			originFieldName: "DataPoints",
			returnSlice:     doubleDataPointSlice,
		},
	},
}

var intHistogram = &messageStruct{
	structName:     "IntHistogram",
	description:    "// IntHistogram represents the type of a metric that is calculated by aggregating as a Histogram of all reported double measurements over a time interval.",
	originFullName: "otlpmetrics.IntHistogram",
	fields: []baseField{
		aggregationTemporalityField,
		&sliceField{
			fieldMame:       "DataPoints",
			originFieldName: "DataPoints",
			returnSlice:     intHistogramDataPointSlice,
		},
	},
}

var doubleHistogram = &messageStruct{
	structName:     "DoubleHistogram",
	description:    "// DoubleHistogram represents the type of a metric that is calculated by aggregating as a Histogram of all reported double measurements over a time interval.",
	originFullName: "otlpmetrics.DoubleHistogram",
	fields: []baseField{
		aggregationTemporalityField,
		&sliceField{
			fieldMame:       "DataPoints",
			originFieldName: "DataPoints",
			returnSlice:     doubleHistogramDataPointSlice,
		},
	},
}

var intDataPointSlice = &sliceStruct{
	structName: "IntDataPointSlice",
	element:    intDataPoint,
}

var intDataPoint = &messageStruct{
	structName:     "IntDataPoint",
	description:    "// IntDataPoint is a single data point in a timeseries that describes the time-varying values of a scalar int metric.",
	originFullName: "otlpmetrics.IntDataPoint",
	fields: []baseField{
		labelsField,
		startTimeField,
		timeField,
		valueInt64Field,
		intExemplarsField,
	},
}

var doubleDataPointSlice = &sliceStruct{
	structName: "DoubleDataPointSlice",
	element:    doubleDataPoint,
}

var doubleDataPoint = &messageStruct{
	structName:     "DoubleDataPoint",
	description:    "// DoubleDataPoint is a single data point in a timeseries that describes the time-varying value of a double metric.",
	originFullName: "otlpmetrics.DoubleDataPoint",
	fields: []baseField{
		labelsField,
		startTimeField,
		timeField,
		valueFloat64Field,
		doubleExemplarsField,
	},
}

var intHistogramDataPointSlice = &sliceStruct{
	structName: "IntHistogramDataPointSlice",
	element:    intHistogramDataPoint,
}

var intHistogramDataPoint = &messageStruct{
	structName:     "IntHistogramDataPoint",
	description:    "// IntHistogramDataPoint is a single data point in a timeseries that describes the time-varying values of a Histogram of int values.",
	originFullName: "otlpmetrics.IntHistogramDataPoint",
	fields: []baseField{
		labelsField,
		startTimeField,
		timeField,
		countField,
		intSumField,
		bucketCountsField,
		explicitBoundsField,
		intExemplarsField,
	},
}

var doubleHistogramDataPointSlice = &sliceStruct{
	structName: "DoubleHistogramDataPointSlice",
	element:    doubleHistogramDataPoint,
}

var doubleHistogramDataPoint = &messageStruct{
	structName:     "DoubleHistogramDataPoint",
	description:    "// DoubleHistogramDataPoint is a single data point in a timeseries that describes the time-varying values of a Histogram of double values.",
	originFullName: "otlpmetrics.DoubleHistogramDataPoint",
	fields: []baseField{
		labelsField,
		startTimeField,
		timeField,
		countField,
		doubleSumField,
		bucketCountsField,
		explicitBoundsField,
		doubleExemplarsField,
	},
}

var intExemplarSlice = &sliceStruct{
	structName: "IntExemplarSlice",
	element:    intExemplar,
}

var intExemplar = &messageStruct{
	structName: "IntExemplar",
	description: "// IntExemplar is a sample input int measurement.\n//\n" +
		"// Exemplars also hold information about the environment when the measurement was recorded,\n" +
		"// for example the span and trace ID of the active span when the exemplar was recorded.",

	originFullName: "otlpmetrics.IntExemplar",
	fields: []baseField{
		timeField,
		valueInt64Field,
		&sliceField{
			fieldMame:       "FilteredLabels",
			originFieldName: "FilteredLabels",
			returnSlice:     stringMap,
		},
	},
}

var doubleExemplarSlice = &sliceStruct{
	structName: "DoubleExemplarSlice",
	element:    doubleExemplar,
}

var doubleExemplar = &messageStruct{
	structName: "DoubleExemplar",
	description: "// DoubleExemplar is a sample input double measurement.\n//\n" +
		"// Exemplars also hold information about the environment when the measurement was recorded,\n" +
		"// for example the span and trace ID of the active span when the exemplar was recorded.",

	originFullName: "otlpmetrics.DoubleExemplar",
	fields: []baseField{
		timeField,
		valueFloat64Field,
		&sliceField{
			fieldMame:       "FilteredLabels",
			originFieldName: "FilteredLabels",
			returnSlice:     stringMap,
		},
	},
}

var labelsField = &sliceField{
	fieldMame:       "LabelsMap",
	originFieldName: "Labels",
	returnSlice:     stringMap,
}

var intExemplarsField = &sliceField{
	fieldMame:       "Exemplars",
	originFieldName: "Exemplars",
	returnSlice:     intExemplarSlice,
}

var doubleExemplarsField = &sliceField{
	fieldMame:       "Exemplars",
	originFieldName: "Exemplars",
	returnSlice:     doubleExemplarSlice,
}

var countField = &primitiveField{
	fieldMame:       "Count",
	originFieldName: "Count",
	returnType:      "uint64",
	defaultVal:      "uint64(0)",
	testVal:         "uint64(17)",
}

var intSumField = &primitiveField{
	fieldMame:       "Sum",
	originFieldName: "Sum",
	returnType:      "int64",
	defaultVal:      "int64(0.0)",
	testVal:         "int64(1713)",
}

var doubleSumField = &primitiveField{
	fieldMame:       "Sum",
	originFieldName: "Sum",
	returnType:      "float64",
	defaultVal:      "float64(0.0)",
	testVal:         "float64(17.13)",
}

var valueInt64Field = &primitiveField{
	fieldMame:       "Value",
	originFieldName: "Value",
	returnType:      "int64",
	defaultVal:      "int64(0)",
	testVal:         "int64(-17)",
}

var valueFloat64Field = &primitiveField{
	fieldMame:       "Value",
	originFieldName: "Value",
	returnType:      "float64",
	defaultVal:      "float64(0.0)",
	testVal:         "float64(17.13)",
}

var bucketCountsField = &primitiveField{
	fieldMame:       "BucketCounts",
	originFieldName: "BucketCounts",
	returnType:      "[]uint64",
	defaultVal:      "[]uint64(nil)",
	testVal:         "[]uint64{1, 2, 3}",
}

var explicitBoundsField = &primitiveField{
	fieldMame:       "ExplicitBounds",
	originFieldName: "ExplicitBounds",
	returnType:      "[]float64",
	defaultVal:      "[]float64(nil)",
	testVal:         "[]float64{1, 2, 3}",
}

var isMonotonicField = &primitiveField{
	fieldMame:       "IsMonotonic",
	originFieldName: "IsMonotonic",
	returnType:      "bool",
	defaultVal:      "false",
	testVal:         "true",
}

var aggregationTemporalityField = &primitiveTypedField{
	fieldMame:       "AggregationTemporality",
	originFieldName: "AggregationTemporality",
	returnType:      "AggregationTemporality",
	rawType:         "otlpmetrics.AggregationTemporality",
	defaultVal:      "AggregationTemporalityUnspecified",
	testVal:         "AggregationTemporalityCumulative",
}
