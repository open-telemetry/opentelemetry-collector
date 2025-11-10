// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"fmt"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const marshalJSONPrimitive = `{{ if .repeated -}}
	if len(orig.{{ .fieldName }}) > 0 {
		dest.WriteObjectField("{{ .jsonTag }}")
		dest.WriteArrayStart()
		dest.Write{{ upperFirst .goType }}(orig.{{ .fieldName }}[0])
		for i := 1; i < len(orig.{{ .fieldName }}); i++ {
			dest.WriteMore()
			dest.Write{{ upperFirst .goType }}(orig.{{ .fieldName }}[i])
		}
		dest.WriteArrayEnd()
	}
{{- else }}
{{- if not .nullable -}}
	if orig.{{ .fieldName }} != {{ .defaultValue }} {
{{ end -}}
		dest.WriteObjectField("{{ .jsonTag }}")
		dest.Write{{ upperFirst .goType }}(orig.{{ .fieldName }})
{{- if not .nullable -}}
	}
{{- end }}{{- end }}`

const marshalJSONEnum = `{{ if .repeated -}}
	if len(orig.{{ .fieldName }}) > 0 {
		dest.WriteObjectField("{{ .jsonTag }}")
		dest.WriteArrayStart()
		dest.WriteInt32(int32(orig.{{ .fieldName }}[0]))
		for i := 1; i < len(orig.{{ .fieldName }}); i++ {
			dest.WriteMore()
			dest.WriteInt32(int32(orig.{{ .fieldName }}[i]))
		}
		dest.WriteArrayEnd()
	}
{{- else }}
	if int32(orig.{{ .fieldName }}) != 0 {
		dest.WriteObjectField("{{ .jsonTag }}")
		dest.WriteInt32(int32(orig.{{ .fieldName }}))
	}
{{- end }}`

const marshalJSONMessage = `{{ if .repeated -}}
	if len(orig.{{ .fieldName }}) > 0 {
		dest.WriteObjectField("{{ .jsonTag }}")
		dest.WriteArrayStart()
		orig.{{ .fieldName }}[0].MarshalJSON(dest)
		for i := 1; i < len(orig.{{ .fieldName }}); i++ {
			dest.WriteMore()
			orig.{{ .fieldName }}[i].MarshalJSON(dest)
		}
		dest.WriteArrayEnd()
	}
{{- else }}
{{- if .nullable -}}
	if orig.{{ .fieldName }} != nil {
{{ end -}}
	dest.WriteObjectField("{{ .jsonTag }}")
	orig.{{ .fieldName }}.MarshalJSON(dest)
{{- if .nullable -}}
	}
{{- end }}{{- end }}`

const marshalJSONBytes = `{{ if .repeated -}}
	if len(orig.{{ .fieldName }}) > 0 {
		dest.WriteObjectField("{{ .jsonTag }}")
		dest.WriteArrayStart()
		dest.WriteBytes(orig.{{ .fieldName }}[0])
		for i := 1; i < len(orig.{{ .fieldName }}); i++ {
			dest.WriteMore()
			dest.WriteBytes(orig.{{ .fieldName }}[i])
		}
		dest.WriteArrayEnd()
	}
{{- else }}{{ if not .nullable }}
	if len(orig.{{ .fieldName }}) > 0 {
{{- end }}
		dest.WriteObjectField("{{ .jsonTag }}")
		dest.WriteBytes(orig.{{ .fieldName }})
{{- if not .nullable -}}
	}
{{- end }}{{- end }}`

func (pf *Field) GenMarshalJSON() string {
	tf := pf.getTemplateFields()
	switch pf.Type {
	case TypeBytes:
		return template.Execute(template.Parse("marshalJSONBytes", []byte(marshalJSONBytes)), tf)
	case TypeMessage:
		return template.Execute(template.Parse("marshalJSONMessage", []byte(marshalJSONMessage)), tf)
	case TypeEnum:
		return template.Execute(template.Parse("marshalJSONEnum", []byte(marshalJSONEnum)), tf)
	case TypeDouble, TypeFloat,
		TypeFixed64, TypeSFixed64, TypeFixed32, TypeSFixed32,
		TypeInt32, TypeInt64, TypeUint32, TypeUint64,
		TypeSInt32, TypeSInt64,
		TypeBool, TypeString:
		return template.Execute(template.Parse("marshalJSONPrimitive", []byte(marshalJSONPrimitive)), tf)
	}
	panic(fmt.Sprintf("unhandled case %T", pf.Type))
}
