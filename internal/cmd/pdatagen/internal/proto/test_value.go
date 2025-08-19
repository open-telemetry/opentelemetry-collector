// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const encodingTestValueScalar = `{{ if ne .oneOfGroup "" -}}
{ {{ .oneOfGroup }}: &{{ .oneOfMessageFullName }}{{ "{" }}{{ .fieldName }}: {{ .testValue }}} },
{{- else if .repeated }}
{ {{ .fieldName }}: []{{ if .nullable }}*{{ end }}{{ .goType }}{ {{ .testValue }}, {{ .testValue }}, {{ .testValue }} } },
{{- else }}
{ {{ .fieldName }}: {{ .testValue }} },
{{- end }}`

const encodingTestValueMessage = `{{ if ne .oneOfGroup "" -}}
{ {{ .oneOfGroup }}: &{{ .oneOfMessageFullName }}{{ "{" }}{{ .fieldName }}: {{ .testValue }}} },
{{- else if .repeated }}
{ {{ .fieldName }}: []{{ if .nullable }}*{{ end }}{{ .goType }}{ {{ if not .nullable }}*{{ end }}{{ .testValue }}, {{ if not .nullable }}*{{ end }}{{ .testValue }}, {{ if not .nullable }}*{{ end }}{{ .testValue }} } },
{{- else }}
{ {{ .fieldName }}: {{ if not .nullable }}*{{ end }}{{ .testValue }} },
{{- end }}`

func (pf *Field) GenTestValue() string {
	tf := pf.getTemplateFields()
	switch pf.Type {
	case TypeMessage:
		return template.Execute(template.Parse("encodingTestValueMessage", []byte(encodingTestValueMessage)), tf)
	case
		TypeDouble, TypeFloat,
		TypeFixed64, TypeSFixed64, TypeFixed32, TypeSFixed32,
		TypeInt32, TypeInt64, TypeUint32, TypeUint64,
		TypeSInt32, TypeSInt64,
		TypeBool, TypeString, TypeBytes, TypeEnum:
		return template.Execute(template.Parse("encodingTestValueScalar", []byte(encodingTestValueScalar)), tf)
	}
	return ""
}
