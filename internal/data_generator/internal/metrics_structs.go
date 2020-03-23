// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
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
		`otlpmetrics "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"`,
	},
	structs: []baseStruct{
		int64DataPoint,
		doubleDataPoint,
		histogramDataPoint,
		histogramBucketSlice,
		histogramBucket,
		histogramBucketExemplar,
		summaryDataPoint,
		summaryValueAtPercentileSlice,
		summaryValueAtPercentile,
	},
}

var int64DataPoint = &messageStruct{
	structName:     "Int64DataPoint",
	description:    "// Int64DataPoint is a single data point in a timeseries that describes the time-varying values of a int64 metric.",
	originFullName: "otlpmetrics.Int64DataPoint",
	fields: []baseField{
		&sliceField{
			fieldMame:       "LabelsMap",
			originFieldName: "Labels",
			returnSlice:     stringMap,
		},
		startTimeField,
		timestampField,
		&primitiveField{
			fieldMame:       "Value",
			originFieldName: "Value",
			returnType:      "int64",
		},
	},
}

var doubleDataPoint = &messageStruct{
	structName:     "DoubleDataPoint",
	description:    "// DoubleDataPoint is a single data point in a timeseries that describes the time-varying value of a double metric.",
	originFullName: "otlpmetrics.DoubleDataPoint",
	fields: []baseField{
		&sliceField{
			fieldMame:       "LabelsMap",
			originFieldName: "Labels",
			returnSlice:     stringMap,
		},
		startTimeField,
		timestampField,
		&primitiveField{
			fieldMame:       "Value",
			originFieldName: "Value",
			returnType:      "float64",
		},
	},
}

var histogramDataPoint = &messageStruct{
	structName:     "HistogramDataPoint",
	description:    "// HistogramDataPoint is a single data point in a timeseries that describes the time-varying values of a Histogram.",
	originFullName: "otlpmetrics.HistogramDataPoint",
	fields: []baseField{
		&sliceField{
			fieldMame:       "LabelsMap",
			originFieldName: "Labels",
			returnSlice:     stringMap,
		},
		startTimeField,
		timestampField,
		countField,
		sumField,
		&sliceField{
			fieldMame:       "Buckets",
			originFieldName: "Buckets",
			returnSlice:     histogramBucketSlice,
		},
		&primitiveField{
			fieldMame:       "ExplicitBounds",
			originFieldName: "ExplicitBounds",
			returnType:      "[]float64",
		},
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
			fieldMame:       "Exemplar",
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
		timestampField,
		&primitiveField{
			fieldMame:       "Value",
			originFieldName: "Value",
			returnType:      "float64",
		},
		&sliceField{
			fieldMame:       "Attachments",
			originFieldName: "Attachments",
			returnSlice:     stringMap,
		},
	},
}

var summaryDataPoint = &messageStruct{
	structName:     "SummaryDataPoint",
	description:    "// SummaryDataPoint is a single data point in a timeseries that describes the time-varying values of a Summary metric.",
	originFullName: "otlpmetrics.SummaryDataPoint",
	fields: []baseField{
		&sliceField{
			fieldMame:       "LabelsMap",
			originFieldName: "Labels",
			returnSlice:     stringMap,
		},
		startTimeField,
		timestampField,
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
		&primitiveField{
			fieldMame:       "Percentile",
			originFieldName: "Percentile",
			returnType:      "float64",
		},
		&primitiveField{
			fieldMame:       "Value",
			originFieldName: "Value",
			returnType:      "float64",
		},
	},
}

var countField = &primitiveField{
	fieldMame:       "Count",
	originFieldName: "Count",
	returnType:      "uint64",
}

var sumField = &primitiveField{
	fieldMame:       "Sum",
	originFieldName: "Sum",
	returnType:      "float64",
}
