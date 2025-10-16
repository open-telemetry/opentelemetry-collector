// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const deleteOrigOther = `{{ if ne .oneOfGroup "" -}}
	if UseProtoPooling.IsEnabled() {
		ov.{{ .fieldName }} = {{ .defaultValue }}
		ProtoPool{{ .oneOfMessageName }}.Put(ov)
	}
{{ end }}`

const deleteOrigMessage = `{{ if .repeated -}}
	for i := range orig.{{ .fieldName }} {
	{{ if .nullable -}}
		DeleteOrig{{ .origName }}(orig.{{ .fieldName }}[i], true)
	{{- else -}}
		DeleteOrig{{ .origName }}(&orig.{{ .fieldName }}[i], false)
	{{- end }}
	}
{{- else if ne .oneOfGroup "" -}}
	DeleteOrig{{ .origName }}(ov.{{ .fieldName }}, true)
	ov.{{ .fieldName }} = nil
	ProtoPool{{ .oneOfMessageName }}.Put(ov)
{{- else if .nullable -}}
	DeleteOrig{{ .origName }}(orig.{{ .fieldName }}, true)
{{- else -}}
	DeleteOrig{{ .origName }}(&orig.{{ .fieldName }}, false)
{{- end }}
`

func (pf *Field) GenDeleteOrig() string {
	tf := pf.getTemplateFields()
	if pf.Type == TypeMessage {
		return template.Execute(template.Parse("deleteOrigMessage", []byte(deleteOrigMessage)), tf)
	}
	if pf.OneOfGroup != "" {
		return template.Execute(template.Parse("deleteOrigOther", []byte(deleteOrigOther)), tf)
	}
	return ""
}
