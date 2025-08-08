// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

var plog = &Package{
	info: &PackageInfo{
		name: "plog",
		path: "plog",
		imports: []string{
			`"iter"`,
			`"sort"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/data"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`"go.opentelemetry.io/collector/pdata/internal/proto"`,
			`otlplogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/logs/v1"`,
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
			`otlplogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/logs/v1"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
	},
	structs: []baseStruct{
		resourceLogsSlice,
		resourceLogs,
		scopeLogsSlice,
		scopeLogs,
		logSlice,
		logRecord,
	},
}

var resourceLogsSlice = &sliceOfPtrs{
	structName: "ResourceLogsSlice",
	element:    resourceLogs,
}

var resourceLogs = &messageStruct{
	structName:     "ResourceLogs",
	description:    "// ResourceLogs is a collection of logs from a Resource.",
	originFullName: "otlplogs.ResourceLogs",
	fields: []Field{
		resourceField,
		schemaURLField,
		&SliceField{
			fieldName:   "ScopeLogs",
			returnSlice: scopeLogsSlice,
		},
	},
}

var scopeLogsSlice = &sliceOfPtrs{
	structName: "ScopeLogsSlice",
	element:    scopeLogs,
}

var scopeLogs = &messageStruct{
	structName:     "ScopeLogs",
	description:    "// ScopeLogs is a collection of logs from a LibraryInstrumentation.",
	originFullName: "otlplogs.ScopeLogs",
	fields: []Field{
		scopeField,
		schemaURLField,
		&SliceField{
			fieldName:   "LogRecords",
			returnSlice: logSlice,
		},
	},
}

var logSlice = &sliceOfPtrs{
	structName: "LogRecordSlice",
	element:    logRecord,
}

var logRecord = &messageStruct{
	structName:     "LogRecord",
	description:    "// LogRecord are experimental implementation of OpenTelemetry Log Data Model.\n",
	originFullName: "otlplogs.LogRecord",
	fields: []Field{
		&TypedField{
			fieldName:       "ObservedTimestamp",
			originFieldName: "ObservedTimeUnixNano",
			returnType:      timestampType,
		},
		&TypedField{
			fieldName:       "Timestamp",
			originFieldName: "TimeUnixNano",
			returnType:      timestampType,
		},
		traceIDField,
		spanIDField,
		&TypedField{
			fieldName: "Flags",
			returnType: &TypedType{
				structName: "LogRecordFlags",
				rawType:    "uint32",
				defaultVal: "0",
				testVal:    "1",
			},
		},
		&PrimitiveField{
			fieldName: "EventName",
			protoType: ProtoTypeString,
		},
		&PrimitiveField{
			fieldName: "SeverityText",
			protoType: ProtoTypeString,
		},
		&TypedField{
			fieldName: "SeverityNumber",
			returnType: &TypedType{
				structName: "SeverityNumber",
				rawType:    "otlplogs.SeverityNumber",
				isEnum:     true,
				defaultVal: `otlplogs.SeverityNumber(0)`,
				testVal:    `otlplogs.SeverityNumber(5)`,
			},
		},
		bodyField,
		&SliceField{
			fieldName:   "Attributes",
			returnSlice: mapStruct,
		},
		&PrimitiveField{
			fieldName: "DroppedAttributesCount",
			protoType: ProtoTypeUint32,
		},
	},
}

var bodyField = &MessageField{
	fieldName:     "Body",
	returnMessage: anyValue,
}
