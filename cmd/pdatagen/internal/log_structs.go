// Copyright The OpenTelemetry Authors
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

var logFile = &File{
	Name: "log",
	imports: []string{
		`logsproto "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/logs/v1"`,
	},
	testImports: []string{
		`"testing"`,
		``,
		`"github.com/stretchr/testify/assert"`,
		``,
		`logsproto "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/logs/v1"`,
	},
	structs: []baseStruct{
		resourceLogsSlice,
		resourceLogs,
		logSlice,
		logRecord,
	},
}

var resourceLogsSlice = &sliceStruct{
	structName: "ResourceLogsSlice",
	element:    resourceLogs,
}

var resourceLogs = &messageStruct{
	structName:     "ResourceLogs",
	description:    "// ResourceLogs is a collection of logs from a Resource.",
	originFullName: "logsproto.ResourceLogs",
	fields: []baseField{
		resourceField,
		&sliceField{
			fieldMame:       "Logs",
			originFieldName: "Logs",
			returnSlice:     logSlice,
		},
	},
}

var logSlice = &sliceStruct{
	structName: "LogSlice",
	element:    logRecord,
}

var logRecord = &messageStruct{
	structName:     "LogRecord",
	description:    "// LogRecord are experimental implementation of OpenTelemetry Log Data Model.\n",
	originFullName: "logsproto.LogRecord",
	fields: []baseField{
		&primitiveTypedField{
			fieldMame:       "Timestamp",
			originFieldName: "TimestampUnixNano",
			returnType:      "TimestampUnixNano",
			rawType:         "uint64",
			defaultVal:      "TimestampUnixNano(0)",
			testVal:         "TimestampUnixNano(1234567890)",
		},
		traceIDField,
		spanIDField,
		&primitiveTypedField{
			fieldMame:       "Flags",
			originFieldName: "Flags",
			returnType:      "uint32",
			rawType:         "uint32",
			defaultVal:      `uint32(0)`,
			testVal:         `uint32(0x01)`,
		},
		&primitiveField{
			fieldMame:       "SeverityText",
			originFieldName: "SeverityText",
			returnType:      "string",
			defaultVal:      `""`,
			testVal:         `"INFO"`,
		},
		&primitiveTypedField{
			fieldMame:       "SeverityNumber",
			originFieldName: "SeverityNumber",
			returnType:      "logsproto.SeverityNumber",
			rawType:         "logsproto.SeverityNumber",
			defaultVal:      `logsproto.SeverityNumber_UNDEFINED_SEVERITY_NUMBER`,
			testVal:         `logsproto.SeverityNumber_INFO`,
		},
		&primitiveField{
			fieldMame:       "ShortName",
			originFieldName: "ShortName",
			returnType:      "string",
			defaultVal:      `""`,
			testVal:         `"test_name"`,
		},
		&primitiveField{
			fieldMame:       "Body",
			originFieldName: "Body",
			returnType:      "string",
			defaultVal:      `""`,
			testVal:         `"test log message"`,
		},
		attributes,
		droppedAttributesCount,
	},
}
