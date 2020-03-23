// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
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
		`otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"`,
	},
	structs: []baseStruct{
		spanEvent,
		spanStatus,
	},
}

var spanEvent = &messageStruct{
	structName: "SpanEvent",
	description: "// SpanEvent is a time-stamped annotation of the span, consisting of user-supplied\n" +
		"// text description and key-value pairs. See OTLP for event definition.",
	originFullName: "otlptrace.Span_Event",
	fields: []baseField{
		timeField,
		&primitiveField{
			fieldMame:       "Name",
			originFieldName: "Name",
			returnType:      "string",
		},
		attributes,
		droppedAttributesCount,
	},
}

var spanStatus = &messageStruct{
	structName: "SpanStatus",
	description: "// SpanStatus is an optional final status for this span. Semantically when Status wasn't set\n" +
		"// it is means span ended without errors and assume Status.Ok (code = 0).",
	originFullName: "otlptrace.Status",
	fields: []baseField{
		&primitiveTypedField{
			fieldMame:       "Code",
			originFieldName: "Code",
			returnType:      "StatusCode",
			rawType:         "otlptrace.Status_StatusCode",
		},
		&primitiveField{
			fieldMame:       "Message",
			originFieldName: "Message",
			returnType:      "string",
		},
	},
}

var droppedAttributesCount = &primitiveField{
	fieldMame:       "DroppedAttributesCount",
	originFieldName: "DroppedAttributesCount",
	returnType:      "uint32",
}
