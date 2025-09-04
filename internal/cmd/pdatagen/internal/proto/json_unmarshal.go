// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"fmt"
	"strings"

	"github.com/ettle/strcase"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const unmarshalJSONPrimitive = `	case {{ .allJSONTags }}:
{{ if .repeated -}}
	for iter.ReadArray() {
		orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, iter.Read{{ upperFirst .goType }}())
	}
{{ else if .nullable -}}
	{
		var ov *{{ .oneOfMessageFullName }}
		if !UseProtoPooling.IsEnabled() {
			ov = &{{ .oneOfMessageFullName }}{}
		} else {
			ov = ProtoPool{{ .oneOfMessageName }}.Get().(*{{ .oneOfMessageFullName }})
		}
		ov.{{ .fieldName }} = iter.Read{{ upperFirst .goType }}()
		orig.{{ .oneOfGroup }} = ov
	}
{{ else -}}
	orig.{{ .fieldName }} = iter.Read{{ upperFirst .goType }}()
{{- end }}`

const unmarshalJSONEnum = `	case {{ .allJSONTags }}:
{{ if .repeated -}}
	for iter.ReadArray() {
		orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, {{ .origFullName }}(iter.ReadEnumValue({{ .origFullName }}_value)))
	}
{{ else -}}
	orig.{{ .fieldName }} = {{ .origFullName }}(iter.ReadEnumValue({{ .origFullName }}_value))
{{- end }}`

const unmarshalJSONMessage = `	case {{ .allJSONTags }}:
{{ if .repeated -}}
	for iter.ReadArray() {
		orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, {{ if .nullable }}NewOrig{{ .origName }}(){{ else }}{{ .defaultValue }}{{ end }})
		UnmarshalJSONOrig{{ .origName }}({{ if not .nullable }}&{{ end }}orig.{{ .fieldName }}[len(orig.{{ .fieldName }}) - 1], iter)
	}
{{ else if ne .oneOfGroup "" -}}
	{
		var ov *{{ .oneOfMessageFullName }}
		if !UseProtoPooling.IsEnabled() {
			ov = &{{ .oneOfMessageFullName }}{}
		} else {
			ov = ProtoPool{{ .oneOfMessageName }}.Get().(*{{ .oneOfMessageFullName }})
		}
		ov.{{ .fieldName }} = NewOrig{{ .origName }}()
		UnmarshalJSONOrig{{ .origName }}(ov.{{ .fieldName }}, iter)
		orig.{{ .oneOfGroup }} = ov
	}
{{ else -}}
	UnmarshalJSONOrig{{ .origName }}({{ if not .nullable }}&{{ end }}orig.{{ .fieldName }}, iter)
{{- end }}`

const unmarshalJSONBytes = `	case {{ .allJSONTags }}:
{{ if .repeated -}}
	for iter.ReadArray() {
		orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, iter.ReadBytes())
	}
{{ else if ne .oneOfGroup "" -}}
	{
		var ov *{{ .oneOfMessageFullName }}
		if !UseProtoPooling.IsEnabled() {
			ov = &{{ .oneOfMessageFullName }}{}
		} else {
			ov = ProtoPool{{ .oneOfMessageName }}.Get().(*{{ .oneOfMessageFullName }})
		}
		ov.{{ .fieldName }} = iter.ReadBytes()
		orig.{{ .oneOfGroup }} = ov
	}
{{ else -}}
	orig.{{ .fieldName }} = iter.ReadBytes()
{{- end }}`

func (pf *Field) GenUnmarshalJSON() string {
	tf := pf.getTemplateFields()
	tf["allJSONTags"] = allJSONTags(pf.Name)
	switch pf.Type {
	case TypeBytes:
		return template.Execute(template.Parse("unmarshalJSONBytes", []byte(unmarshalJSONBytes)), tf)
	case TypeMessage:
		return template.Execute(template.Parse("unmarshalJSONMessage", []byte(unmarshalJSONMessage)), tf)
	case TypeEnum:
		return template.Execute(template.Parse("unmarshalJSONEnum", []byte(unmarshalJSONEnum)), tf)
	case TypeDouble, TypeFloat,
		TypeFixed64, TypeSFixed64, TypeFixed32, TypeSFixed32,
		TypeInt32, TypeInt64, TypeUint32, TypeUint64,
		TypeSInt32, TypeSInt64,
		TypeBool, TypeString:
		return template.Execute(template.Parse("unmarshalJSONPrimitive", []byte(unmarshalJSONPrimitive)), tf)
	}
	panic(fmt.Sprintf("unhandled case %T", pf.Type))
}

func allJSONTags(str string) string {
	snake := strcase.ToSnake(str)
	if !strings.EqualFold(str, snake) {
		return `"` + lowerFirst(str) + `", "` + snake + `"`
	}
	return `"` + lowerFirst(str) + `"`
}
