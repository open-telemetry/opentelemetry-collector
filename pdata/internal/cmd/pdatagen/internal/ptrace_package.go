// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

var ptrace = &Package{
	info: &PackageInfo{
		name: "ptrace",
		path: "ptrace",
		imports: []string{
			`"sort"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/data"`,
			`otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
		testImports: []string{
			`"testing"`,
			`"unsafe"`,
			``,
			`"github.com/stretchr/testify/assert"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/data"`,
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

var resourceSpans = &messageValueStruct{
	structName:     "ResourceSpans",
	description:    "// ResourceSpans is a collection of spans from a Resource.",
	originFullName: "otlptrace.ResourceSpans",
	fields: []baseField{
		resourceField,
		schemaURLField,
		&sliceField{
			fieldName:   "ScopeSpans",
			returnSlice: scopeSpansSlice,
		},
	},
}

var scopeSpansSlice = &sliceOfPtrs{
	structName: "ScopeSpansSlice",
	element:    scopeSpans,
}

var scopeSpans = &messageValueStruct{
	structName:     "ScopeSpans",
	description:    "// ScopeSpans is a collection of spans from a LibraryInstrumentation.",
	originFullName: "otlptrace.ScopeSpans",
	fields: []baseField{
		scopeField,
		schemaURLField,
		&sliceField{
			fieldName:   "Spans",
			returnSlice: spanSlice,
		},
	},
}

var spanSlice = &sliceOfPtrs{
	structName: "SpanSlice",
	element:    span,
}

var flagsField = &primitiveField{
	fieldName:  "Flags",
	returnType: "uint32",
	defaultVal: `uint32(0)`,
	testVal:    `uint32(0xf)`,
}

var span = &messageValueStruct{
	structName: "Span",
	description: "// Span represents a single operation within a trace.\n" +
		"// See Span definition in OTLP: https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/trace/v1/trace.proto",
	originFullName: "otlptrace.Span",
	fields: []baseField{
		traceIDField,
		spanIDField,
		traceStateField,
		parentSpanIDField,
		nameField,
		flagsField,
		&primitiveTypedField{
			fieldName: "Kind",
			returnType: &primitiveType{
				structName: "SpanKind",
				rawType:    "otlptrace.Span_SpanKind",
				defaultVal: "otlptrace.Span_SpanKind(0)",
				testVal:    "otlptrace.Span_SpanKind(3)",
			},
		},
		startTimeField,
		endTimeField,
		attributes,
		droppedAttributesCount,
		&sliceField{
			fieldName:   "Events",
			returnSlice: spanEventSlice,
		},
		&primitiveField{
			fieldName:  "DroppedEventsCount",
			returnType: "uint32",
			defaultVal: "uint32(0)",
			testVal:    "uint32(17)",
		},
		&sliceField{
			fieldName:   "Links",
			returnSlice: spanLinkSlice,
		},
		&primitiveField{
			fieldName:  "DroppedLinksCount",
			returnType: "uint32",
			defaultVal: "uint32(0)",
			testVal:    "uint32(17)",
		},
		&messageValueField{
			fieldName:     "Status",
			returnMessage: spanStatus,
		},
	},
}

var spanEventSlice = &sliceOfPtrs{
	structName: "SpanEventSlice",
	element:    spanEvent,
}

var spanEvent = &messageValueStruct{
	structName: "SpanEvent",
	description: "// SpanEvent is a time-stamped annotation of the span, consisting of user-supplied\n" +
		"// text description and key-value pairs. See OTLP for event definition.",
	originFullName: "otlptrace.Span_Event",
	fields: []baseField{
		timeField,
		nameField,
		attributes,
		droppedAttributesCount,
	},
}

var spanLinkSlice = &sliceOfPtrs{
	structName: "SpanLinkSlice",
	element:    spanLink,
}

var spanLink = &messageValueStruct{
	structName: "SpanLink",
	description: "// SpanLink is a pointer from the current span to another span in the same trace or in a\n" +
		"// different trace.\n" +
		"// See Link definition in OTLP: https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/trace/v1/trace.proto",
	originFullName: "otlptrace.Span_Link",
	fields: []baseField{
		traceIDField,
		spanIDField,
		traceStateField,
		flagsField,
		attributes,
		droppedAttributesCount,
	},
}

var spanStatus = &messageValueStruct{
	structName: "Status",
	description: "// Status is an optional final status for this span. Semantically, when Status was not\n" +
		"// set, that means the span ended without errors and to assume Status.Ok (code = 0).",
	originFullName: "otlptrace.Status",
	fields: []baseField{
		&primitiveTypedField{
			fieldName: "Code",
			returnType: &primitiveType{
				structName: "StatusCode",
				rawType:    "otlptrace.Status_StatusCode",
				defaultVal: "0",
				testVal:    "1",
			},
		},
		&primitiveField{
			fieldName:  "Message",
			returnType: "string",
			defaultVal: `""`,
			testVal:    `"cancelled"`,
		},
	},
}

var traceStateField = &messageValueField{
	fieldName:     "TraceState",
	returnMessage: traceState,
}

var droppedAttributesCount = &primitiveField{
	fieldName:  "DroppedAttributesCount",
	returnType: "uint32",
	defaultVal: "uint32(0)",
	testVal:    "uint32(17)",
}
