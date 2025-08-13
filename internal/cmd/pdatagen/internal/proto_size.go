// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"fmt"
	"text/template"
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
		l = SizeProtoOrig{{ .origName }}({{ if not .nullable }}&{{ end }}orig.{{ .fieldName }}[i])
		n+= {{ .protoTagSize }} + proto.Sov(uint64(l)) + l
	}
{{- else -}}
	l = SizeProtoOrig{{ .origName }}({{ if not .nullable }}&{{ end }}orig.{{ .fieldName }})
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

func (pf *ProtoField) genSizeProto() string {
	tf := pf.sizeTemplateFields()
	switch pf.Type {
	case ProtoTypeFixed64, ProtoTypeSFixed64, ProtoTypeDouble:
		return executeTemplate(template.Must(templateNew("sizeProtoI8").Parse(sizeProtoI8)), tf)
	case ProtoTypeFixed32, ProtoTypeSFixed32, ProtoTypeFloat:
		return executeTemplate(template.Must(templateNew("sizeProtoI4").Parse(sizeProtoI4)), tf)
	case ProtoTypeInt32, ProtoTypeInt64, ProtoTypeUint32, ProtoTypeUint64, ProtoTypeEnum:
		return executeTemplate(template.Must(templateNew("sizeProtoVarint").Parse(sizeProtoVarint)), tf)
	case ProtoTypeBool:
		return executeTemplate(template.Must(templateNew("sizeProtoBool").Parse(sizeProtoBool)), tf)
	case ProtoTypeBytes, ProtoTypeString:
		return executeTemplate(template.Must(templateNew("sizeProtoBytesString").Parse(sizeProtoBytesString)), tf)
	case ProtoTypeMessage:
		return executeTemplate(template.Must(templateNew("sizeProtoMessage").Parse(sizeProtoMessage)), tf)
	case ProtoTypeSInt32, ProtoTypeSInt64:
		return executeTemplate(template.Must(templateNew("sizeProtoSignedVarint").Parse(sizeProtoSignedVarint)), tf)
	}
	panic(fmt.Sprintf("unhandled case %T", pf.Type))
}

func (pf *ProtoField) sizeTemplateFields() map[string]any {
	key := genProtoKey(pf.ID, pf.wireType())
	return map[string]any{
		"protoTagSize": len(key),
		"fieldName":    pf.Name,
		"origName":     extractNameFromFullQualified(pf.MessageFullName),
		"repeated":     pf.Repeated,
		"nullable":     pf.Nullable,
	}
}
