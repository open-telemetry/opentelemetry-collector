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

var commonFile = &File{
	Name: "common",
	imports: []string{
		`otlpcommon "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"`,
	},
	testImports: []string{
		`"testing"`,
		``,
		`"github.com/stretchr/testify/assert"`,
		``,
		`otlpcommon "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"`,
	},
	structs: []baseStruct{
		instrumentationLibrary,
		anyValueArray,
	},
}

var instrumentationLibrary = &messageValueStruct{
	structName:     "InstrumentationLibrary",
	description:    "// InstrumentationLibrary is a message representing the instrumentation library information.",
	originFullName: "otlpcommon.InstrumentationLibrary",
	fields: []baseField{
		nameField,
		&primitiveField{
			fieldName:       "Version",
			originFieldName: "Version",
			returnType:      "string",
			defaultVal:      `""`,
			testVal:         `"test_version"`,
		},
	},
}

// This will not be generated by this class.
// Defined here just to be available as returned message for the fields.
var stringMap = &slicePtrStruct{
	structName: "StringMap",
	element:    stringKeyValue,
}

var stringKeyValue = &messagePtrStruct{}

// This will not be generated by this class.
// Defined here just to be available as returned message for the fields.
var attributeMap = &slicePtrStruct{
	structName: "AttributeMap",
	element:    attributeKeyValue,
}

var attributeKeyValue = &messagePtrStruct{}

var instrumentationLibraryField = &messageValueField{
	fieldName:       "InstrumentationLibrary",
	originFieldName: "InstrumentationLibrary",
	returnMessage:   instrumentationLibrary,
}

var startTimeField = &primitiveTypedField{
	fieldName:       "StartTime",
	originFieldName: "StartTimeUnixNano",
	returnType:      "TimestampUnixNano",
	rawType:         "uint64",
	defaultVal:      "TimestampUnixNano(0)",
	testVal:         "TimestampUnixNano(1234567890)",
}

var timeField = &primitiveTypedField{
	fieldName:       "Timestamp",
	originFieldName: "TimeUnixNano",
	returnType:      "TimestampUnixNano",
	rawType:         "uint64",
	defaultVal:      "TimestampUnixNano(0)",
	testVal:         "TimestampUnixNano(1234567890)",
}

var endTimeField = &primitiveTypedField{
	fieldName:       "EndTime",
	originFieldName: "EndTimeUnixNano",
	returnType:      "TimestampUnixNano",
	rawType:         "uint64",
	defaultVal:      "TimestampUnixNano(0)",
	testVal:         "TimestampUnixNano(1234567890)",
}

var attributes = &slicePtrField{
	fieldName:       "Attributes",
	originFieldName: "Attributes",
	returnSlice:     attributeMap,
}

var nameField = &primitiveField{
	fieldName:       "Name",
	originFieldName: "Name",
	returnType:      "string",
	defaultVal:      `""`,
	testVal:         `"test_name"`,
}

var anyValue = &messageValueStruct{
	structName:     "AttributeValue",
	originFullName: "otlpcommon.AnyValue",
}

var anyValueArray = &sliceValueStruct{
	structName: "AnyValueArray",
	element:    anyValue,
}
