// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const copyOther = `{{ if .repeated -}}
	dest.{{ .fieldName }} = append(dest.{{ .fieldName }}[:0], src.{{ .fieldName }}...)
{{ else if ne .oneOfGroup "" -}}
	dest.Set{{ .fieldName }}(src.{{ .fieldName }}())
{{ else if .nullable -}}
	if src.Has{{ .fieldName }}() {
		dest.Set{{ .fieldName }}(src.{{ .fieldName }})
	} else {
		dest.Remove{{ .fieldName }}()
	}
{{ else -}}
	dest.{{ .fieldName }} = src.{{ .fieldName }}
{{- end }}`

const copyMessage = `{{ if .repeated -}}
	dest.{{ .fieldName }} = Copy{{ .messageName }}{{ if .nullable }}Ptr{{ end }}Slice(dest.{{ .fieldName }}, src.{{ .fieldName }})
{{- else if ne .oneOfGroup "" -}}
	ov := New{{ .messageName }}()
	Copy{{ .messageName }}(ov, src.{{ .fieldName }}())
	dest.Set{{ .fieldName }}(ov)	
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
