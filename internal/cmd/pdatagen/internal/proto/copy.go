// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/tmplutil"
)

const copyOther = `{{ if .repeated -}}
	{{ if eq .goType "string" -}}
	dest.{{ .fieldName }} = dest.{{ .fieldName }}[:0]
	for _, v := range src.{{ .fieldName }} {
		dest.{{ .fieldName }} = Append(st, dest.{{ .fieldName }}, CopyString(st, v))
	}
	{{ else if .isBytes -}}
	dest.{{ .fieldName }} = CopyBytes(st, src.{{ .fieldName }})
	{{ else -}}
	dest.{{ .fieldName }} = CopySlice(st, dest.{{ .fieldName }}, src.{{ .fieldName }})
	{{ end }}
{{ else if ne .oneOfGroup "" -}}
	ov := Alloc[{{ .oneOfMessageName }}](st)
	{{ if .isBytes -}}
	ov.{{ .fieldName }} = CopyBytes(st, t.{{ .fieldName }})
	{{ else if eq .goType "string" -}}
	ov.{{ .fieldName }} = CopyString(st, t.{{ .fieldName }})
	{{ else -}}
	ov.{{ .fieldName }} = t.{{ .fieldName }}
	{{ end -}}
	dest.{{ .oneOfGroup }} = ov
{{ else if .nullable -}}
	if src.Has{{ .fieldName }}() {
		dest.Set{{ .fieldName }}(src.{{ .fieldName }})
	} else {
		dest.Remove{{ .fieldName }}()
	}
{{ else if eq .goType "string" -}}
	dest.{{ .fieldName }} = CopyString(st, src.{{ .fieldName }})
{{ else if .isBytes -}}
	dest.{{ .fieldName }} = CopyBytes(st, src.{{ .fieldName }})
{{ else -}}
	dest.{{ .fieldName }} = src.{{ .fieldName }}
{{- end }}`

const copyMessage = `{{ if .repeated -}}
	dest.{{ .fieldName }} = Copy{{ .messageName }}{{ if .nullable }}Ptr{{ end }}Slice(dest.{{ .fieldName }}, src.{{ .fieldName }}, st)
{{- else if ne .oneOfGroup "" -}}
	ov := Alloc[{{ .oneOfMessageName }}](st)
	ov.{{ .fieldName }} = Alloc[{{ .messageName }}](st)
	Copy{{ .messageName }}(ov.{{ .fieldName }}, t.{{ .fieldName }}, st)
	dest.{{ .oneOfGroup }} = ov	
{{- else if .nullable -}}
	dest.{{ .fieldName }} = Copy{{ .messageName }}(dest.{{ .fieldName }}, src.{{ .fieldName }}, st)
{{- else -}}
	Copy{{ .messageName }}(&dest.{{ .fieldName }}, &src.{{ .fieldName }}, st)
{{- end }}
`

func (pf *Field) GenCopy() string {
	tf := pf.getTemplateFields()
	if pf.Type == TypeMessage {
		return tmplutil.Execute(tmplutil.Parse("copyMessage", []byte(copyMessage)), tf)
	}
	return tmplutil.Execute(tmplutil.Parse("copyOther", []byte(copyOther)), tf)
}
