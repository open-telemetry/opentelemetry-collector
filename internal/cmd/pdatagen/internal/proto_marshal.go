// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"fmt"
	"text/template"
)

const marshalProtoFloat = `{{ if .repeated -}}
	l = len(orig.{{ .fieldName }})
	if l > 0 {
		for i := l - 1; i >= 0; i-- {
			pos -= {{ div .bitSize 8 }}
			binary.LittleEndian.PutUint{{ .bitSize }}(buf[pos:], math.Float{{ .bitSize }}bits(orig.{{ .fieldName }}[i]))
		}
		pos = proto.EncodeVarint(buf, pos, uint64(l*{{ div .bitSize 8 }}))
		{{ range .protoTag -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
	}
{{- else }}
{{- if not .nullable -}}
	if orig.{{ .fieldName }} != 0 {
{{ end -}}
		pos -= {{ div .bitSize 8 }}
		binary.LittleEndian.PutUint{{ .bitSize }}(buf[pos:], math.Float{{ .bitSize }}bits(orig.{{ .fieldName }}))
		{{ range .protoTag -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
{{- if not .nullable -}}
	}
{{- end }}{{- end }}`

const marshalProtoFixed = `{{ if .repeated -}}
	l = len(orig.{{ .fieldName }})
	if l > 0 {
		for i := l - 1; i >= 0; i-- {
			pos -= {{ div .bitSize 8 }}
			binary.LittleEndian.PutUint{{ .bitSize }}(buf[pos:], uint{{ .bitSize }}(orig.{{ .fieldName }}[i]))
		}
		pos = proto.EncodeVarint(buf, pos, uint64(l*{{ div .bitSize 8 }}))
		{{ range .protoTag -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
	}
{{- else }}
{{- if not .nullable -}}
	if orig.{{ .fieldName }} != 0 {
{{ end -}}
		pos -= {{ div .bitSize 8 }}
		binary.LittleEndian.PutUint{{ .bitSize }}(buf[pos:], uint{{ .bitSize }}(orig.{{ .fieldName }}))
		{{ range .protoTag -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
{{- if not .nullable -}}
	}
{{- end }}{{- end }}`

const marshalProtoBool = `{{ if .repeated -}}
	l = len(orig.{{ .fieldName }})
	if l > 0 {
		for i := l - 1; i >= 0; i-- {
			pos--
			if orig.{{ .fieldName }}[i] {
				buf[pos] = 1
			} else {
				buf[pos] = 0
			}
		}
		pos = proto.EncodeVarint(buf, pos, uint64(l))
		{{ range .protoTag -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
	}
{{- else }}
{{- if not .nullable -}}
	if orig.{{ .fieldName }} {
{{ end -}}
		pos--
		if orig.{{ .fieldName }} {
			buf[pos] = 1
		} else {
			buf[pos] = 0
		}
		{{ range .protoTag -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
{{- if not .nullable -}}
	}
{{- end }}{{- end }}`

const marshalProtoVarint = `{{ if .repeated -}}
	l = len(orig.{{ .fieldName }})
	if l > 0 {
		endPos := pos
		for i := l - 1; i >= 0; i-- {
			pos = proto.EncodeVarint(buf, pos, uint64(orig.{{ .fieldName }}[i]))
		}
		pos = proto.EncodeVarint(buf, pos, uint64(endPos-pos))
		{{ range .protoTag -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
	}
{{- else }}
{{- if not .nullable -}}
	if orig.{{ .fieldName }} != 0 {
{{ end -}}
		pos = proto.EncodeVarint(buf, pos, uint64(orig.{{ .fieldName }}))
		{{ range .protoTag -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
{{- if not .nullable -}}
	}
{{- end }}{{- end }}`

const marshalProtoBytesString = `{{ if .repeated -}}
	for i := len(orig.{{ .fieldName }}) - 1; i >= 0; i-- {
		l = len(orig.{{ .fieldName }}[i])
		pos -= l
		copy(buf[pos:], orig.{{ .fieldName }}[i])
		pos = proto.EncodeVarint(buf, pos, uint64(l))
		{{ range .protoTag -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
	}
{{- else -}}
	l = len(orig.{{ .fieldName }})
{{ if not .nullable -}}
	if l > 0 {
{{ end -}}
		pos -= l
		copy(buf[pos:], orig.{{ .fieldName }})
		pos = proto.EncodeVarint(buf, pos, uint64(l))
		{{ range .protoTag -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
{{- if not .nullable -}}
	}
{{- end }}{{- end }}`

const marshalProtoMessage = `{{ if .repeated -}}
	for i := len(orig.{{ .fieldName }}) - 1; i >= 0; i-- {
		l = MarshalProtoOrig{{ .origName }}({{ if not .nullable }}&{{ end }}orig.{{ .fieldName }}[i], buf[:pos])
		pos -= l
		pos = proto.EncodeVarint(buf, pos, uint64(l))
		{{ range .protoTag -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
	}
{{- else }}
	l = MarshalProtoOrig{{ .origName }}({{ if not .nullable }}&{{ end }}orig.{{ .fieldName }}, buf[:pos])
	pos -= l
	pos = proto.EncodeVarint(buf, pos, uint64(l))
	{{ range .protoTag -}}
	pos--
	buf[pos] = {{ . }}
	{{ end -}}
{{- end }}`

const marshalProtoSignedVarint = `{{ if .repeated -}}
	l = len(orig.{{ .fieldName }})
	if l > 0 {
		endPos := pos
		for i := l - 1; i >= 0; i-- {
			pos = proto.EncodeVarint(buf, pos, uint64((uint{{ .bitSize }}(orig.{{ .fieldName }}[i])<<1)^uint{{ .bitSize }}(orig.{{ .fieldName }}[i]>>{{ sub .bitSize 1}})))
		}
		pos = proto.EncodeVarint(buf, pos, uint64(endPos-pos))
		{{ range .protoTag -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
	}
{{- else }}
{{- if not .nullable -}}
	if orig.{{ .fieldName }} != 0 {
{{ end -}}
		pos = proto.EncodeVarint(buf, pos, uint64((uint{{ .bitSize }}(orig.{{ .fieldName }})<<1)^uint{{ .bitSize }}(orig.{{ .fieldName }}>>{{ sub .bitSize 1}})))
		{{ range .protoTag -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
{{- if not .nullable -}}
	}
{{- end }}{{- end }}`

func (pf *ProtoField) genMarshalProto() string {
	tf := pf.marshalTemplateFields()
	switch pf.Type {
	case ProtoTypeDouble, ProtoTypeFloat:
		return executeTemplate(template.Must(templateNew("marshalProtoFloat").Parse(marshalProtoFloat)), tf)
	case ProtoTypeFixed64, ProtoTypeSFixed64, ProtoTypeFixed32, ProtoTypeSFixed32:
		return executeTemplate(template.Must(templateNew("marshalProtoFixed").Parse(marshalProtoFixed)), tf)
	case ProtoTypeInt32, ProtoTypeInt64, ProtoTypeUint32, ProtoTypeUint64, ProtoTypeEnum:
		return executeTemplate(template.Must(templateNew("marshalProtoVarint").Parse(marshalProtoVarint)), tf)
	case ProtoTypeBool:
		return executeTemplate(template.Must(templateNew("marshalProtoBool").Parse(marshalProtoBool)), tf)
	case ProtoTypeBytes, ProtoTypeString:
		return executeTemplate(template.Must(templateNew("marshalProtoBytesString").Parse(marshalProtoBytesString)), tf)
	case ProtoTypeMessage:
		return executeTemplate(template.Must(templateNew("marshalProtoMessage").Parse(marshalProtoMessage)), tf)
	case ProtoTypeSInt32, ProtoTypeSInt64:
		return executeTemplate(template.Must(templateNew("marshalProtoSignedVarint").Parse(marshalProtoSignedVarint)), tf)
	}
	panic(fmt.Sprintf("unhandled case %T", pf.Type))
}

func (pf *ProtoField) marshalTemplateFields() map[string]any {
	bitSize := 0
	switch pf.Type {
	case ProtoTypeFixed64, ProtoTypeSFixed64, ProtoTypeInt64, ProtoTypeUint64, ProtoTypeSInt64, ProtoTypeDouble:
		bitSize = 64
	case ProtoTypeFixed32, ProtoTypeSFixed32, ProtoTypeInt32, ProtoTypeUint32, ProtoTypeSInt32, ProtoTypeFloat, ProtoTypeEnum:
		bitSize = 32
	}

	return map[string]any{
		"protoTag":  genProtoKey(pf.ID, pf.wireType()),
		"fieldName": pf.Name,
		"origName":  extractNameFromFullQualified(pf.MessageFullName),
		"repeated":  pf.Repeated,
		"nullable":  pf.Nullable,
		"bitSize":   bitSize,
	}
}
