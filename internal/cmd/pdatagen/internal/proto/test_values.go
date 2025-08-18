package proto

import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const encodingTestValueNullableOneOf = `{{ if ne .oneOfGroup "" -}}
{ {{ .oneOfGroup }}: &{{ .oneOfMessageFullName }}{{ "{" }}{{ .fieldName }}: {{ .testValue }}} },
{ {{ .oneOfGroup }}: &{{ .oneOfMessageFullName }}{{ "{" }}{{ .fieldName }}: {{ .defaultValue }}} },
{{- else }}
{ {{ .fieldName }}: {{ .testValue }} },
{{- end }}`

const encodingTestValueNullableMessage = ``

func (pf *Field) GenerateTestValues(*Message) string {
	tf := pf.getTemplateFields()
	switch pf.Type {
	case TypeMessage:
		return template.Execute(template.Parse("encodingTestValueNullableMessage", []byte(encodingTestValueNullableMessage)), tf)
	case TypeDouble, TypeFloat,
		TypeFixed64, TypeSFixed64, TypeFixed32, TypeSFixed32,
		TypeInt32, TypeInt64, TypeUint32, TypeUint64,
		TypeSInt32, TypeSInt64,
		TypeBool, TypeString, TypeBytes, TypeEnum:
		return template.Execute(template.Parse("encodingTestValueNullableScalar", []byte(encodingTestValueNullableOneOf)), tf)
	}
	return ""
}
