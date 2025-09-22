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
			`"go.opentelemetry.io/collector/pdata/internal/data"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`"go.opentelemetry.io/collector/pdata/internal/proto"`,
			`otlpcollectorlogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/logs/v1"`,
			`otlplogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/logs/v1"`,
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
			`"go.opentelemetry.io/collector/pdata/internal/data"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`otlpcollectorlogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/logs/v1"`,
			`otlplogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/logs/v1"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
	},
	structs: []baseStruct{
		logs,
		resourceLogsSlice,
		resourceLogs,
		scopeLogsSlice,
		scopeLogs,
		logSlice,
		logRecord,
	},
}

var logs = &messageStruct{
	structName:     "Logs",
	description:    "// Logs is the top-level struct that is propagated through the logs pipeline.\n// Use NewLogs to create new instance, zero-initialized instance is not valid for use.",
	originFullName: "otlpcollectorlogs.ExportLogsServiceRequest",
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

var resourceLogsSlice = &messageSlice{
	structName:      "ResourceLogsSlice",
	elementNullable: true,
	element:         resourceLogs,
}

var resourceLogs = &messageStruct{
	structName:     "ResourceLogs",
	description:    "// ResourceLogs is a collection of logs from a Resource.",
	originFullName: "otlplogs.ResourceLogs",
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
	},
}

var scopeLogsSlice = &messageSlice{
	structName:      "ScopeLogsSlice",
	elementNullable: true,
	element:         scopeLogs,
}

var scopeLogs = &messageStruct{
	structName:     "ScopeLogs",
	description:    "// ScopeLogs is a collection of logs from a LibraryInstrumentation.",
	originFullName: "otlplogs.ScopeLogs",
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
	structName:     "LogRecord",
	description:    "// LogRecord are experimental implementation of OpenTelemetry Log Data Model.\n",
	originFullName: "otlplogs.LogRecord",
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
				messageName: "otlplogs.SeverityNumber",
				defaultVal:  `otlplogs.SeverityNumber(0)`,
				testVal:     `otlplogs.SeverityNumber(5)`,
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
			returnMessage: anyValue,
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
