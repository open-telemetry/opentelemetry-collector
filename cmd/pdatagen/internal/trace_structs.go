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

var traceFile = &File{
	Name: "trace",
	imports: []string{
		`"sort"`,
		``,
		`otlptrace "go.opentelemetry.io/collector/model/internal/data/protogen/trace/v1"`,
	},
	testImports: []string{
		`"testing"`,
		``,
		`"github.com/stretchr/testify/assert"`,
		``,
		`otlptrace "go.opentelemetry.io/collector/model/internal/data/protogen/trace/v1"`,
	},
	structs: []baseStruct{
		resourceSpansSlice,
		resourceSpans,
		instrumentationLibrarySpansSlice,
		instrumentationLibrarySpans,
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
			fieldName:       "InstrumentationLibrarySpans",
			originFieldName: "InstrumentationLibrarySpans",
			returnSlice:     instrumentationLibrarySpansSlice,
		},
	},
}

var instrumentationLibrarySpansSlice = &sliceOfPtrs{
	structName: "InstrumentationLibrarySpansSlice",
	element:    instrumentationLibrarySpans,
}

var instrumentationLibrarySpans = &messageValueStruct{
	structName:     "InstrumentationLibrarySpans",
	description:    "// InstrumentationLibrarySpans is a collection of spans from a LibraryInstrumentation.",
	originFullName: "otlptrace.InstrumentationLibrarySpans",
	fields: []baseField{
		instrumentationLibraryField,
		schemaURLField,
		&sliceField{
			fieldName:       "Spans",
			originFieldName: "Spans",
			returnSlice:     spanSlice,
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
			fieldName:       "Kind",
			originFieldName: "Kind",
			returnType:      "SpanKind",
			rawType:         "otlptrace.Span_SpanKind",
			defaultVal:      "SpanKindUnspecified",
			testVal:         "SpanKindServer",
		},
		startTimeField,
		endTimeField,
		attributes,
		droppedAttributesCount,
		&sliceField{
			fieldName:       "Events",
			originFieldName: "Events",
			returnSlice:     spanEventSlice,
		},
		&primitiveField{
			fieldName:       "DroppedEventsCount",
			originFieldName: "DroppedEventsCount",
			returnType:      "uint32",
			defaultVal:      "uint32(0)",
			testVal:         "uint32(17)",
		},
		&sliceField{
			fieldName:       "Links",
			originFieldName: "Links",
			returnSlice:     spanLinkSlice,
		},
		&primitiveField{
			fieldName:       "DroppedLinksCount",
			originFieldName: "DroppedLinksCount",
			returnType:      "uint32",
			defaultVal:      "uint32(0)",
			testVal:         "uint32(17)",
		},
		&messageValueField{
			fieldName:       "Status",
			originFieldName: "Status",
			returnMessage:   spanStatus,
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
	structName: "SpanStatus",
	description: "// SpanStatus is an optional final status for this span. Semantically, when Status was not\n" +
		"// set, that means the span ended without errors and to assume Status.Ok (code = 0).",
	originFullName: "otlptrace.Status",
	fields: []baseField{
		&primitiveTypedField{
			fieldName:       "Code",
			originFieldName: "Code",
			returnType:      "StatusCode",
			rawType:         "otlptrace.Status_StatusCode",
			defaultVal:      "StatusCode(0)",
			testVal:         "StatusCode(1)",
			// Generate code without setter. Setter will be manually coded since we
			// need to also change DeprecatedCode when Code is changed according
			// to OTLP spec https://github.com/open-telemetry/opentelemetry-proto/blob/59c488bfb8fb6d0458ad6425758b70259ff4a2bd/opentelemetry/proto/trace/v1/trace.proto#L231
			manualSetter: true,
		},
		&primitiveField{
			fieldName:       "Message",
			originFieldName: "Message",
			returnType:      "string",
			defaultVal:      `""`,
			testVal:         `"cancelled"`,
		},
	},
}

var traceIDField = &primitiveStructField{
	fieldName:       "TraceID",
	originFieldName: "TraceId",
	returnType:      "TraceID",
	defaultVal:      "NewTraceID([16]byte{})",
	testVal:         "NewTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})",
}

var spanIDField = &primitiveStructField{
	fieldName:       "SpanID",
	originFieldName: "SpanId",
	returnType:      "SpanID",
	defaultVal:      "NewSpanID([8]byte{})",
	testVal:         "NewSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})",
}

var parentSpanIDField = &primitiveStructField{
	fieldName:       "ParentSpanID",
	originFieldName: "ParentSpanId",
	returnType:      "SpanID",
	defaultVal:      "NewSpanID([8]byte{})",
	testVal:         "NewSpanID([8]byte{8, 7, 6, 5, 4, 3, 2, 1})",
}

var traceStateField = &primitiveTypedField{
	fieldName:       "TraceState",
	originFieldName: "TraceState",
	returnType:      "TraceState",
	rawType:         "string",
	defaultVal:      `TraceState("")`,
	testVal:         `TraceState("congo=congos")`,
}

var droppedAttributesCount = &primitiveField{
	fieldName:       "DroppedAttributesCount",
	originFieldName: "DroppedAttributesCount",
	returnType:      "uint32",
	defaultVal:      "uint32(0)",
	testVal:         "uint32(17)",
}
