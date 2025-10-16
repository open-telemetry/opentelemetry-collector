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
			`"go.opentelemetry.io/collector/pdata/internal/data"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`"go.opentelemetry.io/collector/pdata/internal/proto"`,
			`otlpcollectortrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/trace/v1"`,
			`otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"`,
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
			protoType:   proto.TypeMessage,
			returnSlice: resourceSpansSlice,
		},
	},
	hasWrapper: true,
}

var resourceSpansSlice = &messageSlice{
	structName:      "ResourceSpansSlice",
	elementNullable: true,
	element:         resourceSpans,
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
			protoType:   proto.TypeMessage,
			returnSlice: scopeSpansSlice,
		},
		&PrimitiveField{
			fieldName: "SchemaUrl",
			protoID:   3,
			protoType: proto.TypeString,
		},
	},
}

var scopeSpansSlice = &messageSlice{
	structName:      "ScopeSpansSlice",
	elementNullable: true,
	element:         scopeSpans,
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
	originFullName: "otlptrace.Status",
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
				messageName: "otlptrace.Status_StatusCode",
				defaultVal:  "otlptrace.Status_StatusCode(0)",
				testVal:     "otlptrace.Status_StatusCode(1)",
			},
		},
	},
}
