// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const copyOther = `{{ if .repeated -}}
	dest.{{ .fieldName }} = append(dest.{{ .fieldName }}[:0], src.{{ .fieldName }}...)
{{- else if not .nullable -}}
	dest.{{ .fieldName }} = src.{{ .fieldName }}
{{ else -}}
	var ov *{{ .oneOfMessageName }}
	if !UseProtoPooling.IsEnabled() {
		ov = &{{ .oneOfMessageName }}{}
	} else {
		ov = ProtoPool{{ .oneOfMessageName }}.Get().(*{{ .oneOfMessageName }})
	}
	ov.{{ .fieldName }} = t.{{ .fieldName }}
	dest.{{ .oneOfGroup }} = ov
{{- end }}`

const copyMessage = `{{ if .repeated -}}
	dest.{{ .fieldName }} = Copy{{ .messageName }}{{ if .nullable }}Ptr{{ end }}Slice(dest.{{ .fieldName }}, src.{{ .fieldName }})
{{- else if ne .oneOfGroup "" -}}
	var ov *{{ .oneOfMessageName }}
	if !UseProtoPooling.IsEnabled() {
		ov = &{{ .oneOfMessageName }}{}
	} else {
		ov = ProtoPool{{ .oneOfMessageName }}.Get().(*{{ .oneOfMessageName }})
	}
	ov.{{ .fieldName }} = New{{ .messageName }}()
	Copy{{ .messageName }}(ov.{{ .fieldName }}, t.{{ .fieldName }})
	dest.{{ .oneOfGroup }} = ov	
{{- else if .nullable -}}
	dest.{{ .fieldName }} = Copy{{ .messageName }}(dest.{{ .fieldName }}, src.{{ .fieldName }})
{{- else -}}
	Copy{{ .messageName }}(&dest.{{ .fieldName }}, &src.{{ .fieldName }})
{{- end }}
`

func (pf *Field) GenCopy() string {
	tf := pf.getTemplateFields()
	if pf.Type == TypeMessage {
		return template.Execute(template.Parse("copyMessage", []byte(copyMessage)), tf)
	}
	return template.Execute(template.Parse("copyOther", []byte(copyOther)), tf)
}
