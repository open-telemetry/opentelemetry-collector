// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"fmt"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const sizeProtoI8 = `{{ if .repeated -}}
	l = len(orig.{{ .fieldName }})
	if l > 0 {
		l *= 8
		n+= {{ .protoTagSize }} + proto.Sov(uint64(l)) + l
	}
{{- else if not .nullable -}}
	if orig.{{ .fieldName }} != 0 {
		n+= {{ add .protoTagSize 8 }}
	}
{{- else -}}
	n+= {{ add .protoTagSize 8 }}
{{- end }}`

const sizeProtoI4 = `{{ if .repeated -}}
	l = len(orig.{{ .fieldName }})
	if l > 0 {
		l *= 4
		n+= + {{ .protoTagSize }} + proto.Sov(uint64(l)) + l
	}
{{- else if not .nullable -}}
	if orig.{{ .fieldName }} != 0 {
		n+= {{ add .protoTagSize 4 }}
	}
{{- else -}}
	n+= {{ add .protoTagSize 4 }}
{{- end }}`

const sizeProtoBool = `{{ if .repeated -}}
	l = len(orig.{{ .fieldName }})
	if l > 0 {
		n+= + {{ .protoTagSize }} + proto.Sov(uint64(l)) + l
	}
{{- else if not .nullable -}}
	if orig.{{ .fieldName }} {
		n+= {{ add .protoTagSize 1 }}
	}
{{- else -}}
	n+= {{ add .protoTagSize 1 }}
{{- end }}`

const sizeProtoVarint = `{{ if .repeated -}}
	if len(orig.{{ .fieldName }}) > 0 {
		l = 0
		for _, e := range orig.{{ .fieldName }} {
			l += proto.Sov(uint64(e))
		}
		n+= {{ .protoTagSize }} + proto.Sov(uint64(l)) + l
	}
{{- else if not .nullable -}}
	if orig.{{ .fieldName }} != 0 {
		n+= {{ .protoTagSize }} + proto.Sov(uint64(orig.{{ .fieldName }}))
	}
{{- else -}}
	n+= {{ .protoTagSize }} + proto.Sov(uint64(orig.{{ .fieldName }}))
{{- end }}`

const sizeProtoBytesString = `{{ if .repeated -}}
	for _, s := range orig.{{ .fieldName }} {
		l = len(s)
		n+= {{ .protoTagSize }} + proto.Sov(uint64(l)) + l
	}
{{- else if not .nullable -}}
	l = len(orig.{{ .fieldName }})
	if l > 0 {
		n+= {{ .protoTagSize }} + proto.Sov(uint64(l)) + l
	}
{{- else -}}
	l = len(orig.{{ .fieldName }})
	n+= {{ .protoTagSize }} + proto.Sov(uint64(l)) + l
{{- end }}`

const sizeProtoMessage = `{{ if .repeated -}}
	for i := range orig.{{ .fieldName }} {
		l = orig.{{ .fieldName }}[i].SizeProto()
		n+= {{ .protoTagSize }} + proto.Sov(uint64(l)) + l
	}
{{- else if .nullable -}}
	if orig.{{ .fieldName }} != nil {
		l = orig.{{ .fieldName }}.SizeProto()
		n+= {{ .protoTagSize }} + proto.Sov(uint64(l)) + l
	}
{{- else -}}
	l = orig.{{ .fieldName }}.SizeProto()
	n+= {{ .protoTagSize }} + proto.Sov(uint64(l)) + l
{{- end }}`

const sizeProtoSignedVarint = `{{ if .repeated -}}
	if len(orig.{{ .fieldName }}) > 0 {
		l = 0
		for _, e := range orig.{{ .fieldName }} {
			l += proto.Soz(uint64(e))
		}
		n+= {{ .protoTagSize }} + proto.Sov(uint64(l)) + l
	}
{{- else if not .nullable -}}
	if orig.{{ .fieldName }} != 0 {
		n+= {{ .protoTagSize }} + proto.Soz(uint64(orig.{{ .fieldName }}))
	}
{{- else -}}
	n+= {{ .protoTagSize }} + proto.Soz(uint64(orig.{{ .fieldName }}))
{{- end }}`

func (pf *Field) GenSizeProto() string {
	tf := pf.getTemplateFields()
	switch pf.Type {
	case TypeFixed64, TypeSFixed64, TypeDouble:
		return template.Execute(template.Parse("sizeProtoI8", []byte(sizeProtoI8)), tf)
	case TypeFixed32, TypeSFixed32, TypeFloat:
		return template.Execute(template.Parse("sizeProtoI4", []byte(sizeProtoI4)), tf)
	case TypeInt32, TypeInt64, TypeUint32, TypeUint64, TypeEnum:
		return template.Execute(template.Parse("sizeProtoVarint", []byte(sizeProtoVarint)), tf)
	case TypeBool:
		return template.Execute(template.Parse("sizeProtoBool", []byte(sizeProtoBool)), tf)
	case TypeBytes, TypeString:
		return template.Execute(template.Parse("sizeProtoBytesString", []byte(sizeProtoBytesString)), tf)
	case TypeMessage:
		return template.Execute(template.Parse("sizeProtoMessage", []byte(sizeProtoMessage)), tf)
	case TypeSInt32, TypeSInt64:
		return template.Execute(template.Parse("sizeProtoSignedVarint", []byte(sizeProtoSignedVarint)), tf)
	}
	panic(fmt.Sprintf("unhandled case %T", pf.Type))
}
