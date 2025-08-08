// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"fmt"
	"text/template"
)

const marshalProtoI8 = `{{ if .repeated -}}
	l = len(orig.{{ .fieldName }})
	if l > 0 {
		for i := l - 1; i >= 0; i-- {
			pos -= 8
			encoding_binary.LittleEndian.PutUint64(buf[pos:], math.Float64bits(orig.{{ .fieldName }}[i]))
		}
		pos = proto.EncodeVarint(buf, pos, uint64(l*8))
		{{ range .protoKey -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
	}
{{- else }}
{{- if not .nullable -}}
	if orig.{{ .fieldName }} != 0 {
{{- end -}}
		pos -= 8
		encoding_binary.LittleEndian.PutUint64(buf[pos:], math.Float64bits(orig.{{ .fieldName }}))
		{{ range .protoKey -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
{{- if not .nullable -}}
	}
{{- end }}{{- end }}`

const marshalProtoI4 = `{{ if .repeated -}}
	l = len(orig.{{ .fieldName }})
	if l > 0 {
		for i := l - 1; i >= 0; i-- {
			pos -= 4
			encoding_binary.LittleEndian.PutUint32(buf[pos:], math.Float32bits(orig.{{ .fieldName }}[i]))
		}
		pos = proto.EncodeVarint(buf, pos, uint64(l*4))
		{{ range .protoKey -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
	}
{{- else }}
{{- if not .nullable -}}
	if orig.{{ .fieldName }} != 0 {
{{- end -}}
		pos -= 4
		encoding_binary.LittleEndian.PutUint32(buf[pos:], math.Float32bits(orig.{{ .fieldName }}))
		{{ range .protoKey -}}
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
		{{ range .protoKey -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
	}
{{- else }}
{{- if not .nullable -}}
	if orig.{{ .fieldName }} {
{{- end -}}
		pos--
		if orig.{{ .fieldName }} {
			buf[pos] = 1
		} else {
			buf[pos] = 0
		}
		{{ range .protoKey -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
{{- if not .nullable -}}
	}
{{- end }}{{- end }}`

const marshalProtoVarint = `{{ if .repeated -}}
	if len(orig.{{ .fieldName }}) > 0 {
		endPos := pos
		for i := l - 1; i >= 0; i-- {
			pos = proto.EncodeVarint(buf, pos, uint64(orig.{{ .fieldName }}))
		}
		pos = proto.EncodeVarint(buf, pos, uint64(endPos-pos))
		{{ range .protoKey -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
	}
{{- else }}
{{- if not .nullable -}}
	if orig.{{ .fieldName }} != 0 {
{{- end -}}
		pos = proto.EncodeVarint(buf, pos, uint64(orig.{{ .fieldName }}))
		{{ range .protoKey -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
{{- if not .nullable -}}
	}
{{- end }}{{- end }}`

const marshalProtoBytesString = `{{ if .repeated -}}
	for i := len(orig.{{ .fieldName }}) - 1; i >= 0; i-- {
		l = len(orig.{{ .fieldName }})[i]
		pos -= l
		copy(buf[pos:], orig.{{ .fieldName }}[i])
		pos = proto.EncodeVarint(buf, pos, uint64(l))
		{{ range .protoKey -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
	}
{{- else -}}
	l = len(orig.{{ .fieldName }})
{{ if not .nullable -}}
	if l > 0 {
{{- end -}}
		pos -= l
		copy(buf[pos:], orig.{{ .fieldName }})
		pos = proto.EncodeVarint(buf, pos, uint64(l))
		{{ range .protoKey -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
{{- if not .nullable -}}
	}
{{- end }}{{- end }}`

const marshalProtoMessage = `{{ if .repeated -}}
	for i := range orig.{{ .fieldName }} {
		l = MarshalProtoOrig{{ .messageName }}({{ if not .nullable }}&{{ end }}orig.{{ .fieldName }}[i], buf[:pos])
		pos -= l
		pos = proto.EncodeVarint(buf, pos, uint64(l))
		{{ range .protoKey -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
	}
{{- else }}
{{- if not .nullable -}}
	if orig.{{ .fieldName }} != nil {
{{- end -}}
		l = MarshalProtoOrig{{ .messageName }}({{ if not .nullable }}&{{ end }}orig.{{ .fieldName }}, buf[:pos])
		pos -= l
		pos = proto.EncodeVarint(buf, pos, uint64(l))
		{{ range .protoKey -}}
		pos--
		buf[pos] = {{ . }}
		{{ end -}}
{{- if not .nullable -}}
	}
{{- end }}{{- end }}`

const marshalProtoSignedVarint = `{{ if .repeated -}}
	if len(orig.{{ .fieldName }}) > 0 {
		l = 0
		for _, e := range orig.{{ .fieldName }} {
			l += proto.Soz(uint64(e))
		}
		n+= {{ .keySize }} + proto.Sov(uint64(l)) + l
	}
{{- else if not .nullable -}}
	if orig.{{ .fieldName }} != 0 {
		n+= {{ .keySize }} + proto.Soz(uint64(orig.{{ .fieldName }}))
	}
{{- else -}}
	n+= {{ .keySize }} + proto.Soz(uint64(orig.{{ .fieldName }}))
{{- end }}`

func (pf *ProtoField) genMarshalProto() string {
	tf := pf.marshalTemplateFields()
	switch pf.Type {
	case ProtoTypeFixed64, ProtoTypeSFixed64, ProtoTypeDouble:
		return executeTemplate(template.Must(templateNew("marshalProtoI8").Parse(marshalProtoI8)), tf)
	case ProtoTypeFixed32, ProtoTypeSFixed32, ProtoTypeFloat:
		return executeTemplate(template.Must(templateNew("marshalProtoI4").Parse(marshalProtoI4)), tf)
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
	return map[string]any{
		"protoKey":    genProtoKey(pf.ID, pf.wireType()),
		"fieldName":   pf.Name,
		"messageName": pf.MessageName,
		"repeated":    pf.Repeated,
		"nullable":    pf.Nullable,
	}
}
