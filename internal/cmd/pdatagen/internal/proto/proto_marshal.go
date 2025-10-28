// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"fmt"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
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
		l = orig.{{ .fieldName }}[i].MarshalProto(buf[:pos])
		pos -= l
		pos = proto.EncodeVarint(buf, pos, uint64(l))
		{{ range .protoTag -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
	}
{{- else if .nullable -}}
	if orig.{{ .fieldName }} != nil {
		l = orig.{{ .fieldName }}.MarshalProto(buf[:pos])
		pos -= l
		pos = proto.EncodeVarint(buf, pos, uint64(l))
		{{ range .protoTag -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
	}
{{- else -}}
	l = orig.{{ .fieldName }}.MarshalProto(buf[:pos])
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

func (pf *Field) GenMarshalProto() string {
	tf := pf.getTemplateFields()
	switch pf.Type {
	case TypeDouble, TypeFloat:
		return template.Execute(template.Parse("marshalProtoFloat", []byte(marshalProtoFloat)), tf)
	case TypeFixed64, TypeSFixed64, TypeFixed32, TypeSFixed32:
		return template.Execute(template.Parse("marshalProtoFixed", []byte(marshalProtoFixed)), tf)
	case TypeInt32, TypeInt64, TypeUint32, TypeUint64, TypeEnum:
		return template.Execute(template.Parse("marshalProtoVarint", []byte(marshalProtoVarint)), tf)
	case TypeBool:
		return template.Execute(template.Parse("marshalProtoBool", []byte(marshalProtoBool)), tf)
	case TypeBytes, TypeString:
		return template.Execute(template.Parse("marshalProtoBytesString", []byte(marshalProtoBytesString)), tf)
	case TypeMessage:
		return template.Execute(template.Parse("marshalProtoMessage", []byte(marshalProtoMessage)), tf)
	case TypeSInt32, TypeSInt64:
		return template.Execute(template.Parse("marshalProtoSignedVarint", []byte(marshalProtoSignedVarint)), tf)
	}
	panic(fmt.Sprintf("unhandled case %T", pf.Type))
}
