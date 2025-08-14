// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"fmt"
	"text/template"
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
		MarshalJSONOrig{{ .origName }}({{ if not .nullable }}&{{ end }}orig.{{ .fieldName }}[0], dest)
		for i := 1; i < len(orig.{{ .fieldName }}); i++ {
			dest.WriteMore()
			MarshalJSONOrig{{ .origName }}({{ if not .nullable }}&{{ end }}orig.{{ .fieldName }}[i], dest)
		}
		dest.WriteArrayEnd()
	}
{{- else }}
{{- if .nullable -}}
	if orig.{{ .fieldName }} != nil {
{{ end -}}
	dest.WriteObjectField("{{ .jsonTag }}")
	MarshalJSONOrig{{ .origName }}({{ if not .nullable }}&{{ end }}orig.{{ .fieldName }}, dest)
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
{{- else }}
	if len(orig.{{ .fieldName }}) > 0 {
		dest.WriteObjectField("{{ .jsonTag }}")
		dest.WriteBytes(orig.{{ .fieldName }})
	}
{{- end }}`

func (pf *ProtoField) genMarshalJSON() string {
	tf := pf.marshalJSONTemplateFields()
	switch pf.Type {
	case ProtoTypeBytes:
		return executeTemplate(template.Must(templateNew("marshalJSONBytes").Parse(marshalJSONBytes)), tf)
	case ProtoTypeMessage:
		return executeTemplate(template.Must(templateNew("marshalJSONMessage").Parse(marshalJSONMessage)), tf)
	case ProtoTypeEnum:
		return executeTemplate(template.Must(templateNew("marshalJSONEnum").Parse(marshalJSONEnum)), tf)
	case ProtoTypeDouble, ProtoTypeFloat,
		ProtoTypeFixed64, ProtoTypeSFixed64, ProtoTypeFixed32, ProtoTypeSFixed32,
		ProtoTypeInt32, ProtoTypeInt64, ProtoTypeUint32, ProtoTypeUint64,
		ProtoTypeSInt32, ProtoTypeSInt64,
		ProtoTypeBool, ProtoTypeString:
		return executeTemplate(template.Must(templateNew("marshalJSONPrimitive").Parse(marshalJSONPrimitive)), tf)
	}
	panic(fmt.Sprintf("unhandled case %T", pf.Type))
}

func (pf *ProtoField) marshalJSONTemplateFields() map[string]any {
	return map[string]any{
		"goType":       pf.Type.goType(pf.MessageFullName),
		"defaultValue": pf.Type.defaultValue(pf.MessageFullName),
		"fieldName":    pf.Name,
		"origName":     extractNameFromFullQualified(pf.MessageFullName),
		"repeated":     pf.Repeated,
		"nullable":     pf.Nullable,
		"jsonTag":      pf.jsonTag(),
	}
}

func (pf *ProtoField) jsonTag() string {
	// Extract last word because for Enums we use the full name.
	return lowerFirst(extractNameFromFullQualified(pf.Name))
}
