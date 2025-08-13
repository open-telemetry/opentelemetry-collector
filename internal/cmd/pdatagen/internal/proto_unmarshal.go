// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"fmt"
	"text/template"
)

const unmarshalProtoFloat = `{{ if .repeated -}}
	case {{ .fieldID }}:
		if wireType != proto.WireTypeLen {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		prevPos := pos
		pos, err = proto.ConsumeLen(buf, pos)
		if err != nil {
			return err
		}
		size := (pos - prevPos) / {{ div .bitSize 8 }}
		orig.{{ .fieldName }} = make([]{{ .goType }}, 0, size)
		var num uint{{ .bitSize }}
		for i := 0; i < size; i++ {
			num, prevPos, err = proto.ConsumeI{{ .bitSize }}(buf[:pos], prevPos)
			if err != nil {
				return err
			}
			orig.{{ .fieldName }}[i] = math.Float{{ .bitSize }}frombits(num)
		}
		if prevPos != pos {
			return fmt.Errorf("proto: invalid field len = %d for field {{ .fieldName }}", pos - prevPos)
		}
{{- else }}
	case {{ .fieldID }}:
		if wireType != proto.WireTypeI{{ .bitSize }} {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		var num uint{{ .bitSize }}
		num, pos, err = proto.ConsumeI{{ .bitSize }}(buf, pos)
		if err != nil {
			return err
		}
		orig.{{ .fieldName }} = math.Float{{ .bitSize }}frombits(num)
{{- end }}`

const unmarshalProtoFixed = `{{ if .repeated -}}
	case {{ .fieldID }}:
		if wireType != proto.WireTypeLen {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		prevPos := pos
		pos, err = proto.ConsumeLen(buf, pos)
		if err != nil {
			return err
		}
		size := (pos - prevPos) / {{ div .bitSize 8 }}
		orig.{{ .fieldName }} = make([]{{ .goType }}, 0, size)
		var num uint{{ .bitSize }}
		for i := 0; i < size; i++ {
			num, prevPos, err = proto.ConsumeI{{ .bitSize }}(buf[:pos], prevPos)
			if err != nil {
				return err
			}
			orig.{{ .fieldName }}[i] = {{ .goType }}(num)
		}
		if prevPos != pos {
			return fmt.Errorf("proto: invalid field len = %d for field {{ .fieldName }}", pos - prevPos)
		}
{{- else }}
	case {{ .fieldID }}:
		if wireType != proto.WireTypeI{{ .bitSize }} {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		var num uint{{ .bitSize }}
		num, pos, err = proto.ConsumeI{{ .bitSize }}(buf, pos)
		if err != nil {
			return err
		}
		orig.{{ .fieldName }} = {{ .goType }}(num)
{{- end }}`

const unmarshalProtoBool = `{{ if .repeated -}}
	case {{ .fieldID }}:
		if wireType != proto.WireTypeLen {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		prevPos := pos
		pos, err = proto.ConsumeLen(buf, pos)
		if err != nil {
			return err
		}
		// Optimistically assume that bools are encoded as 1 byte even in variant form.
		orig.{{ .fieldName }} = make([]bool, 0, pos - prevPos)
		var num uint64
		for prevPos < pos {
			num, prevPos, err = proto.ConsumeVarint(buf[:pos], prevPos)
			if err != nil {
				return err
			}
			orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, num != 0)
		}
		if prevPos != pos {
			return fmt.Errorf("proto: invalid field len = %d for field {{ .fieldName }}", pos - prevPos)
		}
{{- else }}
	case {{ .fieldID }}:
		if wireType != proto.WireTypeVarint {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		var num uint64
		num, pos, err = proto.ConsumeVarint(buf, pos)
		if err != nil {
			return err
		}
		orig.{{ .fieldName }} = num != 0
{{- end }}`

const unmarshalProtoVarint = `{{ if .repeated -}}
	case {{ .fieldID }}:
		if wireType != proto.WireTypeLen {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		prevPos := pos
		pos, err = proto.ConsumeLen(buf, pos)
		if err != nil {
			return err
		}
		var num uint64
		for prevPos < pos {
			num, prevPos, err = proto.ConsumeVarint(buf[:pos], prevPos)
			if err != nil {
				return err
			}
			orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, {{ .goType }}(num))
		}
		if prevPos != pos {
			return fmt.Errorf("proto: invalid field len = %d for field {{ .fieldName }}", pos - prevPos)
		}
{{- else }}
	case {{ .fieldID }}:
		if wireType != proto.WireTypeVarint {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		var num uint64
		num, pos, err = proto.ConsumeVarint(buf, pos)
		if err != nil {
			return err
		}
		orig.{{ .fieldName }} = {{ .goType }}(num)
{{- end }}`

const unmarshalProtoString = `
	case {{ .fieldID }}:
		if wireType != proto.WireTypeLen {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		prevPos := pos
		pos, err = proto.ConsumeLen(buf, pos)
		if err != nil {
			return err
		}
{{ if .repeated -}}
		orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, string(buf[prevPos:pos]))
{{- else -}}
		orig.{{ .fieldName }} = string(buf[prevPos:pos])
{{- end }}`

const unmarshalProtoBytes = `	
	case {{ .fieldID }}:
		if wireType != proto.WireTypeLen {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		prevPos := pos
		pos, err = proto.ConsumeLen(buf, pos)
		if err != nil {
			return err
		}
{{ if .repeated -}}
		orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, make([]byte, pos - prevPos))
		copy(orig.{{ .fieldName }}[len(orig.{{ .fieldName }}) - 1], buf[prevPos:pos])
{{- else -}}
		orig.{{ .fieldName }} = make([]byte, pos - prevPos)
		copy(orig.{{ .fieldName }}, buf[prevPos:pos])
{{- end }}`

const unmarshalProtoMessage = `
	case {{ .fieldID }}:
		if wireType != proto.WireTypeLen {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		prevPos := pos
		pos, err = proto.ConsumeLen(buf, pos)
		if err != nil {
			return err
		}
{{ if .repeated -}}
		orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, NewOrig{{ if .nullable }}Ptr{{ end }}{{ .origName }}())
		return UnmarshalProtoOrig{{ .origName }}({{ if not .nullable }}&{{ end }}orig.{{ .fieldName }}[len(orig.{{ .fieldName }})-1], buf[prevPos:pos])
{{- else }}
		{{ if .nullable }}orig.{{ .fieldName }} = NewOrigPtr{{ .origName }}(){{ end }}
		return UnmarshalProtoOrig{{ .origName }}({{ if not .nullable }}&{{ end }}orig.{{ .fieldName }}, buf[prevPos:pos])
{{- end }}`

const unmarshalProtoSignedVarint = `{{ if .repeated -}}
	case {{ .fieldID }}:
		if wireType != proto.WireTypeLen {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		prevPos := pos
		pos, err = proto.ConsumeLen(buf, pos)
		if err != nil {
			return err
		}
		// Optimistically assume that bools are encoded as 1 byte even in variant form.
		orig.{{ .fieldName }} = make([]bool, 0, pos - prevPos)
		var num uint64
		for prevPos < pos {
			num, prevPos, err = proto.ConsumeVarint(buf[:pos], prevPos)
			if err != nil {
				return err
			}
			orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, int{{ .bitSize }}(uint{{ .bitSize }}(num >> 1) ^ uint{{ .bitSize }}(int{{ .bitSize }}((num&1)<<{{ sub .bitSize 1 }})>>{{ sub .bitSize 1 }})))
		}
		if prevPos != pos {
			return fmt.Errorf("proto: invalid field len = %d for field {{ .fieldName }}", pos - prevPos)
		}
{{- else }}
	case {{ .fieldID }}:
		if wireType != proto.WireTypeVarint {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		var num uint64
		num, pos, err = proto.ConsumeVarint(buf, pos)
		if err != nil {
			return err
		}
		orig.{{ .fieldName }} = int{{ .bitSize }}(uint{{ .bitSize }}(num >> 1) ^ uint{{ .bitSize }}(int{{ .bitSize }}((num&1)<<{{ sub .bitSize 1 }})>>{{ sub .bitSize 1 }}))
{{- end }}`

func (pf *ProtoField) genUnmarshalProto() string {
	tf := pf.unmarshalProtoTemplateFields()
	switch pf.Type {
	case ProtoTypeDouble, ProtoTypeFloat:
		return executeTemplate(template.Must(templateNew("unmarshalProtoFloat").Parse(unmarshalProtoFloat)), tf)
	case ProtoTypeFixed64, ProtoTypeSFixed64, ProtoTypeFixed32, ProtoTypeSFixed32:
		return executeTemplate(template.Must(templateNew("unmarshalProtoFixed").Parse(unmarshalProtoFixed)), tf)
	case ProtoTypeInt32, ProtoTypeInt64, ProtoTypeUint32, ProtoTypeUint64, ProtoTypeEnum:
		return executeTemplate(template.Must(templateNew("unmarshalProtoVarint").Parse(unmarshalProtoVarint)), tf)
	case ProtoTypeBool:
		return executeTemplate(template.Must(templateNew("unmarshalProtoBool").Parse(unmarshalProtoBool)), tf)
	case ProtoTypeString:
		return executeTemplate(template.Must(templateNew("unmarshalProtoString").Parse(unmarshalProtoString)), tf)
	case ProtoTypeBytes:
		return executeTemplate(template.Must(templateNew("unmarshalProtoBytes").Parse(unmarshalProtoBytes)), tf)
	case ProtoTypeMessage:
		return executeTemplate(template.Must(templateNew("unmarshalProtoMessage").Parse(unmarshalProtoMessage)), tf)
	case ProtoTypeSInt32, ProtoTypeSInt64:
		return executeTemplate(template.Must(templateNew("unmarshalProtoSignedVarint").Parse(unmarshalProtoSignedVarint)), tf)
	}
	panic(fmt.Sprintf("unhandled case %T", pf.Type))
}

func (pf *ProtoField) unmarshalProtoTemplateFields() map[string]any {
	bitSize := 0
	switch pf.Type {
	case ProtoTypeFixed64, ProtoTypeSFixed64, ProtoTypeInt64, ProtoTypeUint64, ProtoTypeSInt64, ProtoTypeDouble:
		bitSize = 64
	case ProtoTypeFixed32, ProtoTypeSFixed32, ProtoTypeInt32, ProtoTypeUint32, ProtoTypeSInt32, ProtoTypeFloat, ProtoTypeEnum:
		bitSize = 32
	}

	return map[string]any{
		"fieldID":   pf.ID,
		"fieldName": pf.Name,
		"origName":  extractNameFromFullQualified(pf.MessageFullName),
		"repeated":  pf.Repeated,
		"nullable":  pf.Nullable,
		"bitSize":   bitSize,
		"goType":    pf.goType(),
	}
}
