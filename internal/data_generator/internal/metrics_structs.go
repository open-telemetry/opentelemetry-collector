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
		summaryDataPointSlice,
		summaryDataPoint,
		summaryValueAtPercentileSlice,
		summaryValueAtPercentile,
	},
}

var summaryDataPointSlice = &sliceStruct{
	structName: "SummaryDataPointGeneratedSlice",
	element:    summaryDataPoint,
}

var summaryDataPoint = &messageStruct{
	structName:     "SummaryDataPoint",
	description:    "is a single data point in a timeseries that describes the time-varying values of a Summary metric",
	originFullName: "otlpmetrics.SummaryDataPoint",
	fields: []baseField{
		startTimeFiled,
		timestampFiled,
		&primitiveField{
			fieldMame:       "Count",
			originFieldName: "Count",
			returnType:      "uint64",
		},
		&primitiveField{
			fieldMame:       "Sum",
			originFieldName: "Sum",
			returnType:      "float64",
		},
		&sliceField{
			fieldMame:       "ValueAtPercentiles",
			originFieldName: "PercentileValues",
			returnSlice:     summaryValueAtPercentileSlice,
		},
		&sliceField{
			fieldMame:       "LabelsMap",
			originFieldName: "Labels",
			returnSlice:     stringMap,
		},
	},
}

var summaryValueAtPercentileSlice = &sliceStruct{
	structName: "SummaryValueAtPercentileSlice",
	element:    summaryValueAtPercentile,
}

var summaryValueAtPercentile = &messageStruct{
	structName:     "SummaryValueAtPercentile",
	description:    "represents the value at a given percentile of a distribution.",
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

var startTimeFiled = &primitiveTypedField{
	fieldMame:       "StartTime",
	originFieldName: "StartTimeUnixnano",
	returnType:      "TimestampUnixNano",
	rawType:         "uint64",
}

var timestampFiled = &primitiveTypedField{
	fieldMame:       "Timestamp",
	originFieldName: "TimestampUnixnano",
	returnType:      "TimestampUnixNano",
	rawType:         "uint64",
}
