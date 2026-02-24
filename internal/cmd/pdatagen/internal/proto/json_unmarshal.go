// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"fmt"
	"strings"

	"github.com/ettle/strcase"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/tmplutil"
)

const unmarshalJSONPrimitive = `	case {{ .allJSONTags }}:
{{ if .repeated -}}
	for iter.ReadArray() {
		orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, iter.Read{{ upperFirst .goType }}())
	}
{{ else if ne .oneOfGroup "" -}}
	{
		var ov *{{ .oneOfMessageName }}
		if !UseProtoPooling.IsEnabled() {
			ov = &{{ .oneOfMessageName }}{}
		} else {
			ov = ProtoPool{{ .oneOfMessageName }}.Get().(*{{ .oneOfMessageName }})
		}
		ov.{{ .fieldName }} = iter.Read{{ upperFirst .goType }}()
		orig.{{ .oneOfGroup }} = ov
	}
{{ else if .nullable -}}
	orig.Set{{ .fieldName }}(iter.Read{{ upperFirst .goType }}())
{{ else -}}
	orig.{{ .fieldName }} = iter.Read{{ upperFirst .goType }}()
{{- end }}`

const unmarshalJSONEnum = `	case {{ .allJSONTags }}:
{{ if .repeated -}}
	for iter.ReadArray() {
		orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, {{ .messageName }}(iter.ReadEnumValue({{ .messageName }}_value)))
	}
{{ else -}}
	orig.{{ .fieldName }} = {{ .messageName }}(iter.ReadEnumValue({{ .messageName }}_value))
{{- end }}`

const unmarshalJSONMessage = `	case {{ .allJSONTags }}:
{{ if .repeated -}}
	for iter.ReadArray() {
		orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, {{ if .nullable }}New{{ .messageName }}(){{ else }}{{ .defaultValue }}{{ end }})
		orig.{{ .fieldName }}[len(orig.{{ .fieldName }}) - 1].UnmarshalJSON(iter)
	}
{{ else if ne .oneOfGroup "" -}}
	{
		var ov *{{ .oneOfMessageName }}
		if !UseProtoPooling.IsEnabled() {
			ov = &{{ .oneOfMessageName }}{}
		} else {
			ov = ProtoPool{{ .oneOfMessageName }}.Get().(*{{ .oneOfMessageName }})
		}
		ov.{{ .fieldName }} = New{{ .messageName }}()
		ov.{{ .fieldName }}.UnmarshalJSON(iter)
		orig.{{ .oneOfGroup }} = ov
	}
{{ else -}}
	{{ if .nullable }}orig.{{ .fieldName }} = New{{ .messageName }}(){{ end }}
	orig.{{ .fieldName }}.UnmarshalJSON(iter)
{{- end }}`

const unmarshalJSONBytes = `	case {{ .allJSONTags }}:
{{ if .repeated -}}
	for iter.ReadArray() {
		orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, iter.ReadBytes())
	}
{{ else if ne .oneOfGroup "" -}}
	{
		var ov *{{ .oneOfMessageName }}
		if !UseProtoPooling.IsEnabled() {
			ov = &{{ .oneOfMessageName }}{}
		} else {
			ov = ProtoPool{{ .oneOfMessageName }}.Get().(*{{ .oneOfMessageName }})
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
		return tmplutil.Execute(tmplutil.Parse("unmarshalJSONBytes", []byte(unmarshalJSONBytes)), tf)
	case TypeMessage:
		return tmplutil.Execute(tmplutil.Parse("unmarshalJSONMessage", []byte(unmarshalJSONMessage)), tf)
	case TypeEnum:
		return tmplutil.Execute(tmplutil.Parse("unmarshalJSONEnum", []byte(unmarshalJSONEnum)), tf)
	case TypeDouble, TypeFloat,
		TypeFixed64, TypeSFixed64, TypeFixed32, TypeSFixed32,
		TypeInt32, TypeInt64, TypeUint32, TypeUint64,
		TypeSInt32, TypeSInt64,
		TypeBool, TypeString:
		return tmplutil.Execute(tmplutil.Parse("unmarshalJSONPrimitive", []byte(unmarshalJSONPrimitive)), tf)
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
