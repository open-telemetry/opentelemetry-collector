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
		{{- if eq .goType "string" }}
		orig.{{ .fieldName }} = Append(st, orig.{{ .fieldName }}, CopyString(st, iter.ReadString()))
		{{- else }}
		orig.{{ .fieldName }} = Append(st, orig.{{ .fieldName }}, iter.Read{{ upperFirst .goType }}())
		{{- end }}
	}
{{ else if ne .oneOfGroup "" -}}
	{
		ov := Alloc[{{ .oneOfMessageName }}](st)
		{{- if eq .goType "string" }}
		ov.{{ .fieldName }} = CopyString(st, iter.ReadString())
		{{- else }}
		ov.{{ .fieldName }} = iter.Read{{ upperFirst .goType }}()
		{{- end }}
		orig.{{ .oneOfGroup }} = ov
	}
{{ else if .nullable -}}
	orig.Set{{ .fieldName }}(iter.Read{{ upperFirst .goType }}())
{{ else -}}
	{{- if eq .goType "string" }}
	orig.{{ .fieldName }} = CopyString(st, iter.ReadString())
	{{- else }}
	orig.{{ .fieldName }} = iter.Read{{ upperFirst .goType }}()
	{{- end }}
{{- end }}`

const unmarshalJSONEnum = `	case {{ .allJSONTags }}:
{{ if .repeated -}}
	for iter.ReadArray() {
		orig.{{ .fieldName }} = Append(st, orig.{{ .fieldName }}, {{ .messageName }}(iter.ReadEnumValue({{ .messageName }}_value)))
	}
{{ else -}}
	orig.{{ .fieldName }} = {{ .messageName }}(iter.ReadEnumValue({{ .messageName }}_value))
{{- end }}`

const unmarshalJSONMessage = `	case {{ .allJSONTags }}:
{{ if .repeated -}}
	for iter.ReadArray() {
		orig.{{ .fieldName }} = Append(st, orig.{{ .fieldName }}, {{ if .nullable }}Alloc[{{ .messageName }}](st){{ else }}{{ .defaultValue }}{{ end }})
		orig.{{ .fieldName }}[len(orig.{{ .fieldName }}) - 1].UnmarshalJSONState(iter, st)
	}
{{ else if ne .oneOfGroup "" -}}
	{
		ov := Alloc[{{ .oneOfMessageName }}](st)
		ov.{{ .fieldName }} = Alloc[{{ .messageName }}](st)
		ov.{{ .fieldName }}.UnmarshalJSONState(iter, st)
		orig.{{ .oneOfGroup }} = ov
	}
{{ else -}}
	{{ if .nullable }}orig.{{ .fieldName }} = Alloc[{{ .messageName }}](st){{ end }}
	orig.{{ .fieldName }}.UnmarshalJSONState(iter, st)
{{- end }}`

const unmarshalJSONBytes = `	case {{ .allJSONTags }}:
{{ if .repeated -}}
	for iter.ReadArray() {
		orig.{{ .fieldName }} = Append(st, orig.{{ .fieldName }}, CopyBytes(st, iter.ReadBytes()))
	}
{{ else if ne .oneOfGroup "" -}}
	{
		ov := Alloc[{{ .oneOfMessageName }}](st)
		ov.{{ .fieldName }} = CopyBytes(st, iter.ReadBytes())
		orig.{{ .oneOfGroup }} = ov
	}
{{ else -}}
	orig.{{ .fieldName }} = CopyBytes(st, iter.ReadBytes())
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
