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

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

var ptrace = &Package{
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
