// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const encodingTestValuesScalar = `{{ if ne .oneOfGroup "" -}}
{ {{ .oneOfGroup }}: &{{ .oneOfMessageFullName }}{{ "{" }}{{ .fieldName }}: {{ .testValue }}} },
{ {{ .oneOfGroup }}: &{{ .oneOfMessageFullName }}{{ "{" }}{{ .fieldName }}: {{ .defaultValue }}} },
{{- else if .repeated }}
{ {{ .fieldName }}: []{{ if .nullable }}*{{ end }}{{ .goType }}{ {{ .defaultValue }}, {{ .testValue }} } },
{{- else }}
{ {{ .fieldName }}: {{ .testValue }} },
{{- end }}`

const encodingTestValuesMessage = `{{ if ne .oneOfGroup "" -}}
{ {{ .oneOfGroup }}: &{{ .oneOfMessageFullName }}{{ "{" }}{{ .fieldName }}: {{ .testValue }}} },
{ {{ .oneOfGroup }}: &{{ .oneOfMessageFullName }}{{ "{" }}{{ .fieldName }}: {{ if .nullable }}&{{ end }}{{ .defaultValue }}} },
{{- else if .repeated }}
{ {{ .fieldName }}: []{{ if .nullable }}*{{ end }}{{ .goType }}{ {{ if .nullable }}&{{ end }}{{ .defaultValue }}, {{ if not .nullable }}*{{ end }}{{ .testValue }} } },
{{- else }}
{ {{ .fieldName }}: {{ if not .nullable }}*{{ end }}{{ .testValue }} },
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
