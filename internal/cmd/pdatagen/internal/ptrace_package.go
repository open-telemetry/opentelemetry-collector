// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

var ptrace = &Package{
	info: &PackageInfo{
		name: "ptrace",
		path: "ptrace",
		imports: []string{
			`"iter"`,
			`"sort"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/data"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`"go.opentelemetry.io/collector/pdata/internal/proto"`,
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
			`otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
	},
	structs: []baseStruct{
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

var resourceSpansSlice = &sliceOfPtrs{
	structName: "ResourceSpansSlice",
	element:    resourceSpans,
}

var resourceSpans = &messageStruct{
	structName:     "ResourceSpans",
	description:    "// ResourceSpans is a collection of spans from a Resource.",
	originFullName: "otlptrace.ResourceSpans",
	fields: []Field{
		resourceField,
		schemaURLField,
		&SliceField{
			fieldName:   "ScopeSpans",
			returnSlice: scopeSpansSlice,
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
		scopeField,
		schemaURLField,
		&SliceField{
			fieldName:   "Spans",
			returnSlice: spanSlice,
		},
	},
}

var spanSlice = &sliceOfPtrs{
	structName: "SpanSlice",
	element:    span,
}

var flagsField = &PrimitiveField{
	fieldName: "Flags",
	protoType: ProtoTypeFixed32,
}

var span = &messageStruct{
	structName: "Span",
	description: "// Span represents a single operation within a trace.\n" +
		"// See Span definition in OTLP: https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/trace/v1/trace.proto",
	originFullName: "otlptrace.Span",
	fields: []Field{
		traceIDField,
		spanIDField,
		&MessageField{
			fieldName:     "TraceState",
			returnMessage: traceState,
		},
		parentSpanIDField,
		nameField,
		flagsField,
		&TypedField{
			fieldName: "Kind",
			returnType: &TypedType{
				structName: "SpanKind",
				rawType:    "otlptrace.Span_SpanKind",
				isEnum:     true,
				defaultVal: "otlptrace.Span_SpanKind(0)",
				testVal:    "otlptrace.Span_SpanKind(3)",
			},
		},
		startTimeField,
		endTimeField,
		&SliceField{
			fieldName:   "Attributes",
			returnSlice: mapStruct,
		},
		&PrimitiveField{
			fieldName: "DroppedAttributesCount",
			protoType: ProtoTypeUint32,
		},
		&SliceField{
			fieldName:   "Events",
			returnSlice: spanEventSlice,
		},
		&PrimitiveField{
			fieldName: "DroppedEventsCount",
			protoType: ProtoTypeUint32,
		},
		&SliceField{
			fieldName:   "Links",
			returnSlice: spanLinkSlice,
		},
		&PrimitiveField{
			fieldName: "DroppedLinksCount",
			protoType: ProtoTypeUint32,
		},
		&MessageField{
			fieldName:     "Status",
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
		timeField,
		nameField,
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
		traceIDField,
		spanIDField,
		&MessageField{
			fieldName:     "TraceState",
			returnMessage: traceState,
		},
		flagsField,
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

var spanStatus = &messageStruct{
	structName: "Status",
	description: "// Status is an optional final status for this span. Semantically, when Status was not\n" +
		"// set, that means the span ended without errors and to assume Status.Ok (code = 0).",
	originFullName: "otlptrace.Status",
	fields: []Field{
		&TypedField{
			fieldName: "Code",
			returnType: &TypedType{
				structName: "StatusCode",
				rawType:    "otlptrace.Status_StatusCode",
				isEnum:     true,
				defaultVal: "0",
				testVal:    "1",
			},
		},
		&PrimitiveField{
			fieldName: "Message",
			protoType: ProtoTypeString,
		},
	},
}
