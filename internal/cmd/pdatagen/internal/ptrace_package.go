// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

var ptrace = &Package{
	info: &PackageInfo{
		name: "ptrace",
		path: "ptrace",
		imports: []string{
			`"encoding/binary"`,
			`"iter"`,
			`"math"`,
			`"sort"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/data"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`"go.opentelemetry.io/collector/pdata/internal/proto"`,
			`otlpcollectortrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/trace/v1"`,
			`otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"`,
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
			`otlpcollectortrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/trace/v1"`,
			`otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
	},
	structs: []baseStruct{
		traces,
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
}

var traces = &messageStruct{
	structName:     "Traces",
	description:    "// Traces is the top-level struct that is propagated through the traces pipeline.\n// Use NewTraces to create new instance, zero-initialized instance is not valid for use.",
	originFullName: "otlpcollectortrace.ExportTraceServiceRequest",
	fields: []Field{
		&SliceField{
			fieldName:   "ResourceSpans",
			protoID:     1,
			protoType:   ProtoTypeMessage,
			returnSlice: resourceSpansSlice,
		},
	},
	hasWrapper: true,
}

var resourceSpansSlice = &sliceOfPtrs{
	structName: "ResourceSpansSlice",
	element:    resourceSpans,
}

var resourceSpans = &messageStruct{
	structName:     "ResourceSpans",
	description:    "// ResourceSpans is a collection of spans from a Resource.",
	originFullName: "otlptrace.ResourceSpans",
	fields: []Field{
		&MessageField{
			fieldName:     "Resource",
			protoID:       1,
			returnMessage: resource,
		},
		&SliceField{
			fieldName:   "ScopeSpans",
			protoID:     2,
			protoType:   ProtoTypeMessage,
			returnSlice: scopeSpansSlice,
		},
		&PrimitiveField{
			fieldName: "SchemaUrl",
			protoID:   3,
			protoType: ProtoTypeString,
		},
	},
}

var scopeSpansSlice = &sliceOfPtrs{
	structName: "ScopeSpansSlice",
	element:    scopeSpans,
}

var scopeSpans = &messageStruct{
	structName:     "ScopeSpans",
	description:    "// ScopeSpans is a collection of spans from a LibraryInstrumentation.",
	originFullName: "otlptrace.ScopeSpans",
	fields: []Field{
		&MessageField{
			fieldName:     "Scope",
			protoID:       1,
			returnMessage: scope,
		},
		&SliceField{
			fieldName:   "Spans",
			protoID:     2,
			protoType:   ProtoTypeMessage,
			returnSlice: spanSlice,
		},
		&PrimitiveField{
			fieldName: "SchemaUrl",
			protoID:   3,
			protoType: ProtoTypeString,
		},
	},
}

var spanSlice = &sliceOfPtrs{
	structName: "SpanSlice",
	element:    span,
}

var span = &messageStruct{
	structName: "Span",
	description: "// Span represents a single operation within a trace.\n" +
		"// See Span definition in OTLP: https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/trace/v1/trace.proto",
	originFullName: "otlptrace.Span",
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
			protoType: ProtoTypeFixed32,
		},
		&PrimitiveField{
			fieldName: "Name",
			protoID:   5,
			protoType: ProtoTypeString,
		},
		&TypedField{
			fieldName: "Kind",
			protoID:   6,
			returnType: &TypedType{
				structName:  "SpanKind",
				protoType:   ProtoTypeEnum,
				messageName: "otlptrace.Span_SpanKind",
				defaultVal:  "otlptrace.Span_SpanKind(0)",
				testVal:     "otlptrace.Span_SpanKind(3)",
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
			protoType:   ProtoTypeMessage,
			returnSlice: mapStruct,
		},
		&PrimitiveField{
			fieldName: "DroppedAttributesCount",
			protoID:   10,
			protoType: ProtoTypeUint32,
		},
		&SliceField{
			fieldName:   "Events",
			protoID:     11,
			protoType:   ProtoTypeMessage,
			returnSlice: spanEventSlice,
		},
		&PrimitiveField{
			fieldName: "DroppedEventsCount",
			protoID:   12,
			protoType: ProtoTypeUint32,
		},
		&SliceField{
			fieldName:   "Links",
			protoID:     13,
			protoType:   ProtoTypeMessage,
			returnSlice: spanLinkSlice,
		},
		&PrimitiveField{
			fieldName: "DroppedLinksCount",
			protoID:   14,
			protoType: ProtoTypeUint32,
		},
		&MessageField{
			fieldName:     "Status",
			protoID:       15,
			returnMessage: spanStatus,
		},
	},
}

var spanEventSlice = &sliceOfPtrs{
	structName: "SpanEventSlice",
	element:    spanEvent,
}

var spanEvent = &messageStruct{
	structName: "SpanEvent",
	description: "// SpanEvent is a time-stamped annotation of the span, consisting of user-supplied\n" +
		"// text description and key-value pairs. See OTLP for event definition.",
	originFullName: "otlptrace.Span_Event",
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
			protoType: ProtoTypeString,
		},
		&SliceField{
			fieldName:   "Attributes",
			protoID:     3,
			protoType:   ProtoTypeMessage,
			returnSlice: mapStruct,
		},
		&PrimitiveField{
			fieldName: "DroppedAttributesCount",
			protoID:   4,
			protoType: ProtoTypeUint32,
		},
	},
}

var spanLinkSlice = &sliceOfPtrs{
	structName: "SpanLinkSlice",
	element:    spanLink,
}

var spanLink = &messageStruct{
	structName: "SpanLink",
	description: "// SpanLink is a pointer from the current span to another span in the same trace or in a\n" +
		"// different trace.\n" +
		"// See Link definition in OTLP: https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/trace/v1/trace.proto",
	originFullName: "otlptrace.Span_Link",
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
			protoType:   ProtoTypeMessage,
			returnSlice: mapStruct,
		},
		&PrimitiveField{
			fieldName: "DroppedAttributesCount",
			protoID:   5,
			protoType: ProtoTypeUint32,
		},
		&PrimitiveField{
			fieldName: "Flags",
			protoID:   6,
			protoType: ProtoTypeFixed32,
		},
	},
}

var spanStatus = &messageStruct{
	structName: "Status",
	description: "// Status is an optional final status for this span. Semantically, when Status was not\n" +
		"// set, that means the span ended without errors and to assume Status.Ok (code = 0).",
	originFullName: "otlptrace.Status",
	fields: []Field{
		&PrimitiveField{
			fieldName: "Message",
			protoID:   2,
			protoType: ProtoTypeString,
		},
		&TypedField{
			fieldName: "Code",
			protoID:   3,
			returnType: &TypedType{
				structName:  "StatusCode",
				protoType:   ProtoTypeEnum,
				messageName: "otlptrace.Status_StatusCode",
				defaultVal:  "otlptrace.Status_StatusCode(0)",
				testVal:     "otlptrace.Status_StatusCode(1)",
			},
		},
	},
}
