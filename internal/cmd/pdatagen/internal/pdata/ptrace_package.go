// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"
import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
)

var ptrace = &Package{
	info: &PackageInfo{
		name: "ptrace",
		path: "ptrace",
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
			`gootlpcollectortrace "go.opentelemetry.io/proto/slim/otlp/collector/trace/v1"`,
			`gootlptrace "go.opentelemetry.io/proto/slim/otlp/trace/v1"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
	},
	structs: []baseStruct{
		traces,
		tracesData,
		resourceSpansSlice,
		resourceSpans,
		scopeSpansSlice,
		scopeSpans,
		spanSlice,
		span,
		spanEventSlice,
		spanEvent,
		spanLinkSlice,
		spanLink,
		spanStatus,
	},
	enums: []*proto.Enum{
		spanKindEnum,
		statusCodeEnum,
	},
}

var traces = &messageStruct{
	structName:    "Traces",
	description:   "// Traces is the top-level struct that is propagated through the traces pipeline.\n// Use NewTraces to create new instance, zero-initialized instance is not valid for use.",
	protoName:     "ExportTraceServiceRequest",
	upstreamProto: "gootlpcollectortrace.ExportTraceServiceRequest",
	fields: []Field{
		&SliceField{
			fieldName:   "ResourceSpans",
			protoID:     1,
			protoType:   proto.TypeMessage,
			returnSlice: resourceSpansSlice,
		},
	},
	hasWrapper: true,
}

var tracesData = &messageStruct{
	structName:    "TracesData",
	description:   "// TracesData represents the traces data that can be stored in a persistent storage,\n// OR can be embedded by other protocols that transfer OTLP traces data but do not\n// implement the OTLP protocol.",
	protoName:     "TracesData",
	upstreamProto: "gootlptrace.TracesData",
	fields: []Field{
		&SliceField{
			fieldName:   "ResourceSpans",
			protoID:     1,
			protoType:   proto.TypeMessage,
			returnSlice: resourceSpansSlice,
		},
	},
	hasOnlyInternal: true,
}

var resourceSpansSlice = &messageSlice{
	structName:      "ResourceSpansSlice",
	elementNullable: true,
	element:         resourceSpans,
}

var resourceSpans = &messageStruct{
	structName:    "ResourceSpans",
	description:   "// ResourceSpans is a collection of spans from a Resource.",
	protoName:     "ResourceSpans",
	upstreamProto: "gootlptrace.ResourceSpans",
	fields: []Field{
		&MessageField{
			fieldName:     "Resource",
			protoID:       1,
			returnMessage: resource,
		},
		&SliceField{
			fieldName:   "ScopeSpans",
			protoID:     2,
			protoType:   proto.TypeMessage,
			returnSlice: scopeSpansSlice,
		},
		&PrimitiveField{
			fieldName: "SchemaUrl",
			protoID:   3,
			protoType: proto.TypeString,
		},
		&SliceField{
			fieldName:   "DeprecatedScopeSpans",
			protoType:   proto.TypeMessage,
			protoID:     1000,
			returnSlice: scopeSpansSlice,
			// Hide accessors for this field because it is a HACK:
			// Workaround for istio 1.15 / envoy 1.23.1 mistakenly emitting deprecated field.
			hideAccessors: true,
		},
	},
}

var scopeSpansSlice = &messageSlice{
	structName:      "ScopeSpansSlice",
	elementNullable: true,
	element:         scopeSpans,
}

var scopeSpans = &messageStruct{
	structName:    "ScopeSpans",
	description:   "// ScopeSpans is a collection of spans from a LibraryInstrumentation.",
	protoName:     "ScopeSpans",
	upstreamProto: "gootlptrace.ScopeSpans",
	fields: []Field{
		&MessageField{
			fieldName:     "Scope",
			protoID:       1,
			returnMessage: scope,
		},
		&SliceField{
			fieldName:   "Spans",
			protoID:     2,
			protoType:   proto.TypeMessage,
			returnSlice: spanSlice,
		},
		&PrimitiveField{
			fieldName: "SchemaUrl",
			protoID:   3,
			protoType: proto.TypeString,
		},
	},
}

var spanSlice = &messageSlice{
	structName:      "SpanSlice",
	elementNullable: true,
	element:         span,
}

var span = &messageStruct{
	structName: "Span",
	description: "// Span represents a single operation within a trace.\n" +
		"// See Span definition in OTLP: https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/trace/v1/trace.proto",
	protoName:     "Span",
	upstreamProto: "gootlptrace.Span",
	fields: []Field{
		&TypedField{
			fieldName:       "TraceID",
			originFieldName: "TraceId",
			protoID:         1,
			returnType:      traceIDType,
		},
		&TypedField{
			fieldName:       "SpanID",
			originFieldName: "SpanId",
			protoID:         2,
			returnType:      spanIDType,
		},
		&MessageField{
			fieldName:     "TraceState",
			protoID:       3,
			returnMessage: traceState,
		},
		&TypedField{
			fieldName:       "ParentSpanID",
			originFieldName: "ParentSpanId",
			protoID:         4,
			returnType:      spanIDType,
		},
		&PrimitiveField{
			fieldName: "Flags",
			protoID:   16,
			protoType: proto.TypeFixed32,
		},
		&PrimitiveField{
			fieldName: "Name",
			protoID:   5,
			protoType: proto.TypeString,
		},
		&TypedField{
			fieldName: "Kind",
			protoID:   6,
			returnType: &TypedType{
				structName:  "SpanKind",
				protoType:   proto.TypeEnum,
				messageName: "SpanKind",
				defaultVal:  "SpanKind_SPAN_KIND_UNSPECIFIED",
				testVal:     "SpanKind_SPAN_KIND_CLIENT",
			},
		},
		&TypedField{
			fieldName:       "StartTimestamp",
			originFieldName: "StartTimeUnixNano",
			protoID:         7,
			returnType:      timestampType,
		},
		&TypedField{
			fieldName:       "EndTimestamp",
			originFieldName: "EndTimeUnixNano",
			protoID:         8,
			returnType:      timestampType,
		},
		&SliceField{
			fieldName:   "Attributes",
			protoID:     9,
			protoType:   proto.TypeMessage,
			returnSlice: mapStruct,
		},
		&PrimitiveField{
			fieldName: "DroppedAttributesCount",
			protoID:   10,
			protoType: proto.TypeUint32,
		},
		&SliceField{
			fieldName:   "Events",
			protoID:     11,
			protoType:   proto.TypeMessage,
			returnSlice: spanEventSlice,
		},
		&PrimitiveField{
			fieldName: "DroppedEventsCount",
			protoID:   12,
			protoType: proto.TypeUint32,
		},
		&SliceField{
			fieldName:   "Links",
			protoID:     13,
			protoType:   proto.TypeMessage,
			returnSlice: spanLinkSlice,
		},
		&PrimitiveField{
			fieldName: "DroppedLinksCount",
			protoID:   14,
			protoType: proto.TypeUint32,
		},
		&MessageField{
			fieldName:     "Status",
			protoID:       15,
			returnMessage: spanStatus,
		},
	},
}

var spanEventSlice = &messageSlice{
	structName:      "SpanEventSlice",
	elementNullable: true,
	element:         spanEvent,
}

var spanEvent = &messageStruct{
	structName: "SpanEvent",
	description: "// SpanEvent is a time-stamped annotation of the span, consisting of user-supplied\n" +
		"// text description and key-value pairs. See OTLP for event definition.",
	protoName:     "SpanEvent",
	upstreamProto: "gootlptrace.Span_Event",
	fields: []Field{
		&TypedField{
			fieldName:       "Timestamp",
			originFieldName: "TimeUnixNano",
			protoID:         1,
			returnType:      timestampType,
		},
		&PrimitiveField{
			fieldName: "Name",
			protoID:   2,
			protoType: proto.TypeString,
		},
		&SliceField{
			fieldName:   "Attributes",
			protoID:     3,
			protoType:   proto.TypeMessage,
			returnSlice: mapStruct,
		},
		&PrimitiveField{
			fieldName: "DroppedAttributesCount",
			protoID:   4,
			protoType: proto.TypeUint32,
		},
	},
}

var spanLinkSlice = &messageSlice{
	structName:      "SpanLinkSlice",
	elementNullable: true,
	element:         spanLink,
}

var spanLink = &messageStruct{
	structName: "SpanLink",
	description: "// SpanLink is a pointer from the current span to another span in the same trace or in a\n" +
		"// different trace.\n" +
		"// See Link definition in OTLP: https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/trace/v1/trace.proto",
	protoName:     "SpanLink",
	upstreamProto: "gootlptrace.Span_Link",
	fields: []Field{
		&TypedField{
			fieldName:       "TraceID",
			originFieldName: "TraceId",
			protoID:         1,
			returnType:      traceIDType,
		},
		&TypedField{
			fieldName:       "SpanID",
			originFieldName: "SpanId",
			protoID:         2,
			returnType:      spanIDType,
		},
		&MessageField{
			fieldName:     "TraceState",
			protoID:       3,
			returnMessage: traceState,
		},
		&SliceField{
			fieldName:   "Attributes",
			protoID:     4,
			protoType:   proto.TypeMessage,
			returnSlice: mapStruct,
		},
		&PrimitiveField{
			fieldName: "DroppedAttributesCount",
			protoID:   5,
			protoType: proto.TypeUint32,
		},
		&PrimitiveField{
			fieldName: "Flags",
			protoID:   6,
			protoType: proto.TypeFixed32,
		},
	},
}

var spanStatus = &messageStruct{
	structName: "Status",
	description: "// Status is an optional final status for this span. Semantically, when Status was not\n" +
		"// set, that means the span ended without errors and to assume Status.Ok (code = 0).",
	protoName:     "Status",
	upstreamProto: "gootlptrace.Status",
	fields: []Field{
		&PrimitiveField{
			fieldName: "Message",
			protoID:   2,
			protoType: proto.TypeString,
		},
		&TypedField{
			fieldName: "Code",
			protoID:   3,
			returnType: &TypedType{
				structName:  "StatusCode",
				protoType:   proto.TypeEnum,
				messageName: "StatusCode",
				defaultVal:  "StatusCode_STATUS_CODE_UNSET",
				testVal:     "StatusCode_STATUS_CODE_OK",
			},
		},
	},
}

var spanKindEnum = &proto.Enum{
	Name:        "SpanKind",
	Description: "// SpanKind is the type of span.\n// Can be used to specify additional relationships between spans in addition to a parent/child relationship.",
	Fields: []*proto.EnumField{
		{Name: "SPAN_KIND_UNSPECIFIED", Value: 0},
		{Name: "SPAN_KIND_INTERNAL", Value: 1},
		{Name: "SPAN_KIND_SERVER", Value: 2},
		{Name: "SPAN_KIND_CLIENT", Value: 3},
		{Name: "SPAN_KIND_PRODUCER", Value: 4},
		{Name: "SPAN_KIND_CONSUMER", Value: 5},
	},
}

var statusCodeEnum = &proto.Enum{
	Name:        "StatusCode",
	Description: "// StatusCode is the status of the span, for the semantics of codes see\n// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/api.md#set-status",
	Fields: []*proto.EnumField{
		{Name: "STATUS_CODE_UNSET", Value: 0},
		{Name: "STATUS_CODE_OK", Value: 1},
		{Name: "STATUS_CODE_ERROR", Value: 2},
	},
}
