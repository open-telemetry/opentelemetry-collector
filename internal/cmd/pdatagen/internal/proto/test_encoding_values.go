// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"slices"
	"strings"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const encodingTestValuesScalar = `{{ if ne .oneOfGroup "" -}}
"{{ .fieldName }}/default": { {{ .oneOfGroup }}: &{{ .oneOfMessageName }}{{ "{" }}{{ .fieldName }}: {{ .defaultValue }}} },
"{{ .fieldName }}/test": { {{ .oneOfGroup }}: &{{ .oneOfMessageName }}{{ "{" }}{{ .fieldName }}: {{ .testValue }}} },
{{- else }}
"{{ .fieldName }}/test": { {{ .fieldName }}: {{ .testValue }} },
{{- end }}`

const encodingTestValuesMessage = `{{ if ne .oneOfGroup "" -}}
"{{ .fieldName }}/default": { {{ .oneOfGroup }}: &{{ .oneOfMessageName }}{{ "{" }}{{ .fieldName }}: {{ .defaultValue }}} },
"{{ .fieldName }}/test": { {{ .oneOfGroup }}: &{{ .oneOfMessageName }}{{ "{" }}{{ .fieldName }}: {{ .testValue }}} },
{{- else }}
"{{ .fieldName }}/test": { {{ .fieldName }}: {{ .testValue }} },
{{- end }}`

func (pf *Field) GenTestEncodingValues() string {
	tf := pf.getTemplateFields()
	switch pf.Type {
	case TypeMessage:
		return template.Execute(template.Parse("encodingTestValuesMessage", []byte(encodingTestValuesMessage)), tf)
	case
		TypeDouble, TypeFloat,
		TypeFixed64, TypeSFixed64, TypeFixed32, TypeSFixed32,
		TypeInt32, TypeInt64, TypeUint32, TypeUint64,
		TypeSInt32, TypeSInt64,
		TypeBool, TypeString, TypeBytes, TypeEnum:
		return template.Execute(template.Parse("encodingTestValuesScalar", []byte(encodingTestValuesScalar)), tf)
	}
	return ""
}

const failingUnmarshalProtoValuesScalar = `
	"{{ .fieldName }}/wrong_wire_type": []byte{ {{ .wrongWireTypeArray }} },
	"{{ .fieldName }}/missing_value": []byte{ {{ .missingValueArray }} },`

func (pf *Field) GenTestFailingUnmarshalProtoValues() string {
	tf := pf.getTemplateFields()
	tf["wrongWireTypeArray"] = protoTagAsByteArray(genProtoTag(pf.ID, WireTypeEndGroup))
	tf["missingValueArray"] = protoTagAsByteArray(tf["protoTag"].([]string))
	switch pf.Type {
	case TypeMessage,
		TypeDouble, TypeFloat,
		TypeFixed64, TypeSFixed64, TypeFixed32, TypeSFixed32,
		TypeInt32, TypeInt64, TypeUint32, TypeUint64,
		TypeSInt32, TypeSInt64,
		TypeBool, TypeString, TypeBytes, TypeEnum:
		return template.Execute(template.Parse("failingUnmarshalProtoValuesScalar", []byte(failingUnmarshalProtoValuesScalar)), tf)
	}
	return ""
}

func protoTagAsByteArray(protoTag []string) string {
	slices.Reverse(protoTag)
	return strings.Join(protoTag, ", ")
}
