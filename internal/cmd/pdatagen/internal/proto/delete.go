// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const deleteOther = `{{ if ne .oneOfGroup "" -}}
	if UseProtoPooling.IsEnabled() {
		ov.{{ .fieldName }} = {{ .defaultValue }}
		ProtoPool{{ .oneOfMessageName }}.Put(ov)
	}
{{ end }}`

const deleteMessage = `{{ if .repeated -}}
	for i := range orig.{{ .fieldName }} {
	{{ if .nullable -}}
		Delete{{ .messageName }}(orig.{{ .fieldName }}[i], true)
	{{- else -}}
		Delete{{ .messageName }}(&orig.{{ .fieldName }}[i], false)
	{{- end }}
	}
{{- else if ne .oneOfGroup "" -}}
	Delete{{ .messageName }}(ov.{{ .fieldName }}, true)
	ov.{{ .fieldName }} = nil
	ProtoPool{{ .oneOfMessageName }}.Put(ov)
{{- else if .nullable -}}
	Delete{{ .messageName }}(orig.{{ .fieldName }}, true)
{{- else -}}
	Delete{{ .messageName }}(&orig.{{ .fieldName }}, false)
{{- end }}
`

func (pf *Field) GenDelete() string {
	tf := pf.getTemplateFields()
	if pf.Type == TypeMessage {
		return template.Execute(template.Parse("deleteMessage", []byte(deleteMessage)), tf)
	}
	if pf.OneOfGroup != "" {
		return template.Execute(template.Parse("deleteOther", []byte(deleteOther)), tf)
	}
	return ""
}
