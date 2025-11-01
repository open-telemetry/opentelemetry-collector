// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"
import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
)

var plog = &Package{
	info: &PackageInfo{
		name: "plog",
		path: "plog",
		imports: []string{
			`"encoding/binary"`,
			`"fmt"`,
			`"iter"`,
			`"math"`,
			`"sort"`,
			`"sync"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`"go.opentelemetry.io/collector/pdata/internal/proto"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
		testImports: []string{
			`"strconv"`,
			`"testing"`,
			`"unsafe"`,
			``,
			`"github.com/stretchr/testify/assert"`,
			`"github.com/stretchr/testify/require"`,
			`"google.golang.org/protobuf/proto"`,
			`gootlpcollectorlogs "go.opentelemetry.io/proto/slim/otlp/collector/logs/v1"`,
			`gootlplogs "go.opentelemetry.io/proto/slim/otlp/logs/v1"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
	},
	structs: []baseStruct{
		logs,
		logsData,
		resourceLogsSlice,
		resourceLogs,
		scopeLogsSlice,
		scopeLogs,
		logSlice,
		logRecord,
	},
	enums: []*proto.Enum{
		severityNumberEnum,
	},
}

var logs = &messageStruct{
	structName:    "Logs",
	description:   "// Logs is the top-level struct that is propagated through the logs pipeline.\n// Use NewLogs to create new instance, zero-initialized instance is not valid for use.",
	protoName:     "ExportLogsServiceRequest",
	upstreamProto: "gootlpcollectorlogs.ExportLogsServiceRequest",
	fields: []Field{
		&SliceField{
			fieldName:   "ResourceLogs",
			protoID:     1,
			protoType:   proto.TypeMessage,
			returnSlice: resourceLogsSlice,
		},
	},
	hasWrapper: true,
}

var logsData = &messageStruct{
	structName:    "LogsData",
	description:   "// LogsData represents the logs data that can be stored in a persistent storage,\n// OR can be embedded by other protocols that transfer OTLP logs data but do not\n// implement the OTLP protocol.",
	protoName:     "LogsData",
	upstreamProto: "gootlplogs.LogsData",
	fields: []Field{
		&SliceField{
			fieldName:   "ResourceLogs",
			protoID:     1,
			protoType:   proto.TypeMessage,
			returnSlice: resourceLogsSlice,
		},
	},
	hasOnlyInternal: true,
}

var resourceLogsSlice = &messageSlice{
	structName:      "ResourceLogsSlice",
	elementNullable: true,
	element:         resourceLogs,
}

var resourceLogs = &messageStruct{
	structName:    "ResourceLogs",
	description:   "// ResourceLogs is a collection of logs from a Resource.",
	protoName:     "ResourceLogs",
	upstreamProto: "gootlplogs.ResourceLogs",
	fields: []Field{
		&MessageField{
			fieldName:     "Resource",
			protoID:       1,
			returnMessage: resource,
		},
		&SliceField{
			fieldName:   "ScopeLogs",
			protoID:     2,
			protoType:   proto.TypeMessage,
			returnSlice: scopeLogsSlice,
		},
		&PrimitiveField{
			fieldName: "SchemaUrl",
			protoID:   3,
			protoType: proto.TypeString,
		},
		&SliceField{
			fieldName:   "DeprecatedScopeLogs",
			protoType:   proto.TypeMessage,
			protoID:     1000,
			returnSlice: scopeLogsSlice,
			// Hide accessors for this field because it is a HACK:
			// Workaround for istio 1.15 / envoy 1.23.1 mistakenly emitting deprecated field.
			hideAccessors: true,
		},
	},
}

var scopeLogsSlice = &messageSlice{
	structName:      "ScopeLogsSlice",
	elementNullable: true,
	element:         scopeLogs,
}

var scopeLogs = &messageStruct{
	structName:    "ScopeLogs",
	description:   "// ScopeLogs is a collection of logs from a LibraryInstrumentation.",
	protoName:     "ScopeLogs",
	upstreamProto: "gootlplogs.ScopeLogs",
	fields: []Field{
		&MessageField{
			fieldName:     "Scope",
			protoID:       1,
			returnMessage: scope,
		},
		&SliceField{
			fieldName:   "LogRecords",
			protoID:     2,
			protoType:   proto.TypeMessage,
			returnSlice: logSlice,
		},
		&PrimitiveField{
			fieldName: "SchemaUrl",
			protoID:   3,
			protoType: proto.TypeString,
		},
	},
}

var logSlice = &messageSlice{
	structName:      "LogRecordSlice",
	elementNullable: true,
	element:         logRecord,
}

var logRecord = &messageStruct{
	structName:    "LogRecord",
	description:   "// LogRecord are experimental implementation of OpenTelemetry Log Data Model.\n",
	protoName:     "LogRecord",
	upstreamProto: "gootlplogs.LogRecord",
	fields: []Field{
		&TypedField{
			fieldName:       "Timestamp",
			protoID:         1,
			originFieldName: "TimeUnixNano",
			returnType:      timestampType,
		},
		&TypedField{
			fieldName:       "ObservedTimestamp",
			protoID:         11,
			originFieldName: "ObservedTimeUnixNano",
			returnType:      timestampType,
		},
		&TypedField{
			fieldName: "SeverityNumber",
			protoID:   2,
			returnType: &TypedType{
				structName:  "SeverityNumber",
				protoType:   proto.TypeEnum,
				messageName: "SeverityNumber",
				defaultVal:  `SeverityNumber_SEVERITY_NUMBER_UNSPECIFIED`,
				testVal:     `SeverityNumber_SEVERITY_NUMBER_DEBUG`,
			},
		},
		&PrimitiveField{
			fieldName: "SeverityText",
			protoID:   3,
			protoType: proto.TypeString,
		},
		&MessageField{
			fieldName:     "Body",
			protoID:       5,
			returnMessage: anyValueStruct,
		},
		&SliceField{
			fieldName:   "Attributes",
			protoID:     6,
			protoType:   proto.TypeMessage,
			returnSlice: mapStruct,
		},
		&PrimitiveField{
			fieldName: "DroppedAttributesCount",
			protoID:   7,
			protoType: proto.TypeUint32,
		},
		&TypedField{
			fieldName: "Flags",
			protoID:   8,
			returnType: &TypedType{
				structName: "LogRecordFlags",
				protoType:  proto.TypeFixed32,
				defaultVal: "0",
				testVal:    "1",
			},
		},
		&TypedField{
			fieldName:       "TraceID",
			originFieldName: "TraceId",
			protoID:         9,
			returnType:      traceIDType,
		},
		&TypedField{
			fieldName:       "SpanID",
			originFieldName: "SpanId",
			protoID:         10,
			returnType:      spanIDType,
		},
		&PrimitiveField{
			fieldName: "EventName",
			protoID:   12,
			protoType: proto.TypeString,
		},
	},
}

var severityNumberEnum = &proto.Enum{
	Name:        "SeverityNumber",
	Description: "// SeverityNumber represent possible values for LogRecord.SeverityNumber",
	Fields: []*proto.EnumField{
		{Name: "SEVERITY_NUMBER_UNSPECIFIED", Value: 0},
		{Name: "SEVERITY_NUMBER_TRACE ", Value: 1},
		{Name: "SEVERITY_NUMBER_TRACE2", Value: 2},
		{Name: "SEVERITY_NUMBER_TRACE3", Value: 3},
		{Name: "SEVERITY_NUMBER_TRACE4", Value: 4},
		{Name: "SEVERITY_NUMBER_DEBUG", Value: 5},
		{Name: "SEVERITY_NUMBER_DEBUG2", Value: 6},
		{Name: "SEVERITY_NUMBER_DEBUG3", Value: 7},
		{Name: "SEVERITY_NUMBER_DEBUG4", Value: 8},
		{Name: "SEVERITY_NUMBER_INFO", Value: 9},
		{Name: "SEVERITY_NUMBER_INFO2", Value: 10},
		{Name: "SEVERITY_NUMBER_INFO3", Value: 11},
		{Name: "SEVERITY_NUMBER_INFO4", Value: 12},
		{Name: "SEVERITY_NUMBER_WARN", Value: 13},
		{Name: "SEVERITY_NUMBER_WARN2", Value: 14},
		{Name: "SEVERITY_NUMBER_WARN3", Value: 15},
		{Name: "SEVERITY_NUMBER_WARN4", Value: 16},
		{Name: "SEVERITY_NUMBER_ERROR", Value: 17},
		{Name: "SEVERITY_NUMBER_ERROR2", Value: 18},
		{Name: "SEVERITY_NUMBER_ERROR3", Value: 19},
		{Name: "SEVERITY_NUMBER_ERROR4", Value: 20},
		{Name: "SEVERITY_NUMBER_FATAL", Value: 21},
		{Name: "SEVERITY_NUMBER_FATAL2", Value: 22},
		{Name: "SEVERITY_NUMBER_FATAL3", Value: 23},
		{Name: "SEVERITY_NUMBER_FATAL4", Value: 24},
	},
}
