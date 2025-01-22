// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

var pcommon = &Package{
	info: &PackageInfo{
		name: "pcommon",
		path: "pcommon",
		imports: []string{
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"`,
			`otlpresource "go.opentelemetry.io/collector/pdata/internal/data/protogen/resource/v1"`,
		},
		testImports: []string{
			`"testing"`,
			``,
			`"github.com/stretchr/testify/assert"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
		},
	},
	structs: []baseStruct{
		scope,
		resource,
		byteSlice,
		float64Slice,
		uInt64Slice,
		int64Slice,
		int32Slice,
		stringSlice,
	},
}

var scope = &messageValueStruct{
	structName:     "InstrumentationScope",
	packageName:    "pcommon",
	description:    "// InstrumentationScope is a message representing the instrumentation scope information.",
	originFullName: "otlpcommon.InstrumentationScope",
	fields: []baseField{
		nameField,
		&primitiveField{
			fieldName:  "Version",
			returnType: "string",
			defaultVal: `""`,
			testVal:    `"test_version"`,
		},
		attributes,
		droppedAttributesCount,
	},
}

// This will not be generated by this class.
// Defined here just to be available as returned message for the fields.
var mapStruct = &sliceOfPtrs{
	structName:  "Map",
	packageName: "pcommon",
}

var scopeField = &messageValueField{
	fieldName:     "Scope",
	returnMessage: scope,
}

var traceState = &messageValueStruct{
	structName:  "TraceState",
	packageName: "pcommon",
}

var timestampType = &primitiveType{
	structName:  "Timestamp",
	packageName: "pcommon",
	rawType:     "uint64",
	defaultVal:  "0",
	testVal:     "1234567890",
}

var startTimeField = &primitiveTypedField{
	fieldName:       "StartTimestamp",
	originFieldName: "StartTimeUnixNano",
	returnType:      timestampType,
}

var timeField = &primitiveTypedField{
	fieldName:       "Timestamp",
	originFieldName: "TimeUnixNano",
	returnType:      timestampType,
}

var endTimeField = &primitiveTypedField{
	fieldName:       "EndTimestamp",
	originFieldName: "EndTimeUnixNano",
	returnType:      timestampType,
}

var attributes = &sliceField{
	fieldName:   "Attributes",
	returnSlice: mapStruct,
}

var nameField = &primitiveField{
	fieldName:  "Name",
	returnType: "string",
	defaultVal: `""`,
	testVal:    `"test_name"`,
}

var anyValue = &messageValueStruct{
	structName:     "Value",
	packageName:    "pcommon",
	originFullName: "otlpcommon.AnyValue",
}

var traceIDField = &primitiveTypedField{
	fieldName:       "TraceID",
	originFieldName: "TraceId",
	returnType:      traceIDType,
}

var traceIDType = &primitiveType{
	structName:  "TraceID",
	packageName: "pcommon",
	rawType:     "data.TraceID",
	defaultVal:  "data.TraceID([16]byte{})",
	testVal:     "data.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})",
}

var spanIDField = &primitiveTypedField{
	fieldName:       "SpanID",
	originFieldName: "SpanId",
	returnType:      spanIDType,
}

var parentSpanIDField = &primitiveTypedField{
	fieldName:       "ParentSpanID",
	originFieldName: "ParentSpanId",
	returnType:      spanIDType,
}

var spanIDType = &primitiveType{
	structName:  "SpanID",
	packageName: "pcommon",
	rawType:     "data.SpanID",
	defaultVal:  "data.SpanID([8]byte{})",
	testVal:     "data.SpanID([8]byte{8, 7, 6, 5, 4, 3, 2, 1})",
}

var schemaURLField = &primitiveField{
	fieldName:  "SchemaUrl",
	returnType: "string",
	defaultVal: `""`,
	testVal:    `"https://opentelemetry.io/schemas/1.5.0"`,
}

var resource = &messageValueStruct{
	structName:     "Resource",
	packageName:    "pcommon",
	description:    "// Resource is a message representing the resource information.",
	originFullName: "otlpresource.Resource",
	fields: []baseField{
		attributes,
		droppedAttributesCount,
	},
}

var resourceField = &messageValueField{
	fieldName:     "Resource",
	returnMessage: resource,
}

var byteSlice = &primitiveSliceStruct{
	structName:           "ByteSlice",
	packageName:          "pcommon",
	itemType:             "byte",
	testOrigVal:          "1, 2, 3",
	testInterfaceOrigVal: []any{1, 2, 3},
	testSetVal:           "5",
	testNewVal:           "1, 5, 3",
}

var float64Slice = &primitiveSliceStruct{
	structName:           "Float64Slice",
	packageName:          "pcommon",
	itemType:             "float64",
	testOrigVal:          "1, 2, 3",
	testInterfaceOrigVal: []any{1, 2, 3},
	testSetVal:           "5",
	testNewVal:           "1, 5, 3",
}

var uInt64Slice = &primitiveSliceStruct{
	structName:           "UInt64Slice",
	packageName:          "pcommon",
	itemType:             "uint64",
	testOrigVal:          "1, 2, 3",
	testInterfaceOrigVal: []any{1, 2, 3},
	testSetVal:           "5",
	testNewVal:           "1, 5, 3",
}

var int64Slice = &primitiveSliceStruct{
	structName:           "Int64Slice",
	packageName:          "pcommon",
	itemType:             "int64",
	testOrigVal:          "1, 2, 3",
	testInterfaceOrigVal: []any{1, 2, 3},
	testSetVal:           "5",
	testNewVal:           "1, 5, 3",
}

var int32Slice = &primitiveSliceStruct{
	structName:           "Int32Slice",
	packageName:          "pcommon",
	itemType:             "int32",
	testOrigVal:          "1, 2, 3",
	testInterfaceOrigVal: []any{1, 2, 3},
	testSetVal:           "5",
	testNewVal:           "1, 5, 3",
}

var stringSlice = &primitiveSliceStruct{
	structName:           "StringSlice",
	packageName:          "pcommon",
	itemType:             "string",
	testOrigVal:          `"a", "b", "c"`,
	testInterfaceOrigVal: []any{`"a"`, `"b"`, `"c"`},
	testSetVal:           `"d"`,
	testNewVal:           `"a", "d", "c"`,
}
