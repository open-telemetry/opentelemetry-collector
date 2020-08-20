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
		metricDescriptor,
		int64DataPointSlice,
		int64DataPoint,
		doubleDataPointSlice,
		doubleDataPoint,
		histogramDataPointSlice,
		histogramDataPoint,
		histogramBucketSlice,
		histogramBucket,
		histogramBucketExemplar,
		summaryDataPointSlice,
		summaryDataPoint,
		summaryValueAtPercentileSlice,
		summaryValueAtPercentile,
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
		"// See Metric definition in OTLP: https://github.com/open-telemetry/opentelemetry-proto/blob/master/opentelemetry/proto/metrics/v1/metrics.proto#L96",
	originFullName: "otlpmetrics.Metric",
	fields: []baseField{
		&messageField{
			fieldName:       "MetricDescriptor",
			originFieldName: "MetricDescriptor",
			returnMessage:   metricDescriptor,
		},
		&sliceField{
			fieldMame:       "Int64DataPoints",
			originFieldName: "Int64DataPoints",
			returnSlice:     int64DataPointSlice,
		},
		&sliceField{
			fieldMame:       "DoubleDataPoints",
			originFieldName: "DoubleDataPoints",
			returnSlice:     doubleDataPointSlice,
		},
		&sliceField{
			fieldMame:       "HistogramDataPoints",
			originFieldName: "HistogramDataPoints",
			returnSlice:     histogramDataPointSlice,
		},
		&sliceField{
			fieldMame:       "SummaryDataPoints",
			originFieldName: "SummaryDataPoints",
			returnSlice:     summaryDataPointSlice,
		},
	},
}

var metricDescriptor = &messageStruct{
	structName:     "MetricDescriptor",
	description:    "// MetricDescriptor is the descriptor of a metric.",
	originFullName: "otlpmetrics.MetricDescriptor",
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
		&primitiveTypedField{
			fieldMame:       "Type",
			originFieldName: "Type",
			returnType:      "MetricType",
			rawType:         "otlpmetrics.MetricDescriptor_Type",
			defaultVal:      "MetricTypeInvalid",
			testVal:         "MetricTypeInt64",
		},
	},
}

var int64DataPointSlice = &sliceStruct{
	structName: "Int64DataPointSlice",
	element:    int64DataPoint,
}

var int64DataPoint = &messageStruct{
	structName:     "Int64DataPoint",
	description:    "// Int64DataPoint is a single data point in a timeseries that describes the time-varying values of a int64 metric.",
	originFullName: "otlpmetrics.Int64DataPoint",
	fields: []baseField{
		labelsField,
		startTimeField,
		timeField,
		valueInt64Field,
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
	},
}

var histogramDataPointSlice = &sliceStruct{
	structName: "HistogramDataPointSlice",
	element:    histogramDataPoint,
}

var histogramDataPoint = &messageStruct{
	structName:     "HistogramDataPoint",
	description:    "// HistogramDataPoint is a single data point in a timeseries that describes the time-varying values of a Histogram.",
	originFullName: "otlpmetrics.HistogramDataPoint",
	fields: []baseField{
		labelsField,
		startTimeField,
		timeField,
		countField,
		sumField,
		&sliceField{
			fieldMame:       "Buckets",
			originFieldName: "Buckets",
			returnSlice:     histogramBucketSlice,
		},
		explicitBoundsField,
	},
}

var histogramBucketSlice = &sliceStruct{
	structName: "HistogramBucketSlice",
	element:    histogramBucket,
}

var histogramBucket = &messageStruct{
	structName:     "HistogramBucket",
	description:    "// HistogramBucket contains values for a histogram bucket.",
	originFullName: "otlpmetrics.HistogramDataPoint_Bucket",
	fields: []baseField{
		countField,
		&messageField{
			fieldName:       "Exemplar",
			originFieldName: "Exemplar",
			returnMessage:   histogramBucketExemplar,
		},
	},
}

var histogramBucketExemplar = &messageStruct{
	structName: "HistogramBucketExemplar",
	description: "// HistogramBucketExemplar are example points that may be used to annotate aggregated Histogram values.\n" +
		"// They are metadata that gives information about a particular value added to a Histogram bucket.",
	originFullName: "otlpmetrics.HistogramDataPoint_Bucket_Exemplar",
	fields: []baseField{
		timeField,
		valueFloat64Field,
		&sliceField{
			fieldMame:       "Attachments",
			originFieldName: "Attachments",
			returnSlice:     stringMap,
		},
	},
}

var summaryDataPointSlice = &sliceStruct{
	structName: "SummaryDataPointSlice",
	element:    summaryDataPoint,
}

var summaryDataPoint = &messageStruct{
	structName:     "SummaryDataPoint",
	description:    "// SummaryDataPoint is a single data point in a timeseries that describes the time-varying values of a Summary metric.",
	originFullName: "otlpmetrics.SummaryDataPoint",
	fields: []baseField{
		labelsField,
		startTimeField,
		timeField,
		countField,
		sumField,
		&sliceField{
			fieldMame:       "ValueAtPercentiles",
			originFieldName: "PercentileValues",
			returnSlice:     summaryValueAtPercentileSlice,
		},
	},
}

var summaryValueAtPercentileSlice = &sliceStruct{
	structName: "SummaryValueAtPercentileSlice",
	element:    summaryValueAtPercentile,
}

var summaryValueAtPercentile = &messageStruct{
	structName:     "SummaryValueAtPercentile",
	description:    "// SummaryValueAtPercentile represents the value at a given percentile of a distribution.",
	originFullName: "otlpmetrics.SummaryDataPoint_ValueAtPercentile",
	fields: []baseField{
		percentileField,
		valueFloat64Field,
	},
}

var labelsField = &sliceField{
	fieldMame:       "LabelsMap",
	originFieldName: "Labels",
	returnSlice:     stringMap,
}

var countField = &primitiveField{
	fieldMame:       "Count",
	originFieldName: "Count",
	returnType:      "uint64",
	defaultVal:      "uint64(0)",
	testVal:         "uint64(17)",
}

var sumField = &primitiveField{
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

var percentileField = &primitiveField{
	fieldMame:       "Percentile",
	originFieldName: "Percentile",
	returnType:      "float64",
	defaultVal:      "float64(0.0)",
	testVal:         "float64(0.90)",
}

var explicitBoundsField = &primitiveField{
	fieldMame:       "ExplicitBounds",
	originFieldName: "ExplicitBounds",
	returnType:      "[]float64",
	defaultVal:      "[]float64(nil)",
	testVal:         "[]float64{1, 2, 3}",
}
