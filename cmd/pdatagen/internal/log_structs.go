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

var logFile = &File{
	Name: "log",
	imports: []string{
		`"sort"`,
		``,
		`otlplogs "go.opentelemetry.io/collector/model/internal/data/protogen/logs/v1"`,
	},
	testImports: []string{
		`"testing"`,
		``,
		`"github.com/stretchr/testify/assert"`,
		``,
		`otlplogs "go.opentelemetry.io/collector/model/internal/data/protogen/logs/v1"`,
	},
	structs: []baseStruct{
		resourceLogsSlice,
		resourceLogs,
		instrumentationLibraryLogsSlice,
		instrumentationLibraryLogs,
		logSlice,
		logRecord,
	},
}

var resourceLogsSlice = &sliceOfPtrs{
	structName: "ResourceLogsSlice",
	element:    resourceLogs,
}

var resourceLogs = &messageValueStruct{
	structName:     "ResourceLogs",
	description:    "// ResourceLogs is a collection of logs from a Resource.",
	originFullName: "otlplogs.ResourceLogs",
	fields: []baseField{
		resourceField,
		schemaURLField,
		&sliceField{
			fieldName:       "InstrumentationLibraryLogs",
			originFieldName: "InstrumentationLibraryLogs",
			returnSlice:     instrumentationLibraryLogsSlice,
		},
	},
}

var instrumentationLibraryLogsSlice = &sliceOfPtrs{
	structName: "InstrumentationLibraryLogsSlice",
	element:    instrumentationLibraryLogs,
}

var instrumentationLibraryLogs = &messageValueStruct{
	structName:     "InstrumentationLibraryLogs",
	description:    "// InstrumentationLibraryLogs is a collection of logs from a LibraryInstrumentation.",
	originFullName: "otlplogs.InstrumentationLibraryLogs",
	fields: []baseField{
		instrumentationLibraryField,
		schemaURLField,
		&sliceField{
			fieldName:       "Logs",
			originFieldName: "Logs",
			returnSlice:     logSlice,
		},
	},
}

var logSlice = &sliceOfPtrs{
	structName: "LogSlice",
	element:    logRecord,
}

var logRecord = &messageValueStruct{
	structName:     "LogRecord",
	description:    "// LogRecord are experimental implementation of OpenTelemetry Log Data Model.\n",
	originFullName: "otlplogs.LogRecord",
	fields: []baseField{
		&primitiveTypedField{
			fieldName:       "Timestamp",
			originFieldName: "TimeUnixNano",
			returnType:      "Timestamp",
			rawType:         "uint64",
			defaultVal:      "Timestamp(0)",
			testVal:         "Timestamp(1234567890)",
		},
		traceIDField,
		spanIDField,
		&primitiveTypedField{
			fieldName:       "Flags",
			originFieldName: "Flags",
			returnType:      "uint32",
			rawType:         "uint32",
			defaultVal:      `uint32(0)`,
			testVal:         `uint32(0x01)`,
		},
		&primitiveField{
			fieldName:       "SeverityText",
			originFieldName: "SeverityText",
			returnType:      "string",
			defaultVal:      `""`,
			testVal:         `"INFO"`,
		},
		&primitiveTypedField{
			fieldName:       "SeverityNumber",
			originFieldName: "SeverityNumber",
			returnType:      "SeverityNumber",
			rawType:         "otlplogs.SeverityNumber",
			defaultVal:      `SeverityNumberUNDEFINED`,
			testVal:         `SeverityNumberINFO`,
		},
		&primitiveField{
			fieldName:       "Name",
			originFieldName: "Name",
			returnType:      "string",
			defaultVal:      `""`,
			testVal:         `"test_name"`,
		},
		bodyField,
		attributes,
		droppedAttributesCount,
	},
}

var bodyField = &messageValueField{
	fieldName:       "Body",
	originFieldName: "Body",
	returnMessage:   anyValue,
}
