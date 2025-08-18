// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"fmt"
	"text/template"
)

const unmarshalProtoFloat = `{{ if .repeated -}}
	case {{ .protoFieldID }}:
		if wireType != proto.WireTypeLen {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		var length int
		length, pos, err = proto.ConsumeLen(buf, pos)
		if err != nil {
			return err
		}
		startPos := pos - length
		size := length / {{ div .bitSize 8 }}
		orig.{{ .fieldName }} = make([]{{ .goType }}, size)
		var num uint{{ .bitSize }}
		for i := 0; i < size; i++ {
			num, startPos, err = proto.ConsumeI{{ .bitSize }}(buf[:pos], startPos)
			if err != nil {
				return err
			}
			orig.{{ .fieldName }}[i] = math.Float{{ .bitSize }}frombits(num)
		}
		if startPos != pos {
			return fmt.Errorf("proto: invalid field len = %d for field {{ .fieldName }}", pos - startPos)
		}
{{- else }}
	case {{ .protoFieldID }}:
		if wireType != proto.WireTypeI{{ .bitSize }} {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		var num uint{{ .bitSize }}
		num, pos, err = proto.ConsumeI{{ .bitSize }}(buf, pos)
		if err != nil {
			return err
		}
{{ if ne .oneOfGroup "" -}}
		ofv := &{{ .oneOfMessageFullName }}{}
		ofv.{{ .fieldName }} = math.Float{{ .bitSize }}frombits(num)
		orig.{{ .oneOfGroup }} = ofv
{{- else }}
		orig.{{ .fieldName }} = math.Float{{ .bitSize }}frombits(num)
{{- end }}{{- end }}`

const unmarshalProtoFixed = `{{ if .repeated -}}
	case {{ .protoFieldID }}:
		if wireType != proto.WireTypeLen {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		var length int
		length, pos, err = proto.ConsumeLen(buf, pos)
		if err != nil {
			return err
		}
		startPos := pos - length
		size := length / {{ div .bitSize 8 }}
		orig.{{ .fieldName }} = make([]{{ .goType }}, size)
		var num uint{{ .bitSize }}
		for i := 0; i < size; i++ {
			num, startPos, err = proto.ConsumeI{{ .bitSize }}(buf[:pos], startPos)
			if err != nil {
				return err
			}
			orig.{{ .fieldName }}[i] = {{ .goType }}(num)
		}
		if startPos != pos {
			return fmt.Errorf("proto: invalid field len = %d for field {{ .fieldName }}", pos - startPos)
		}
{{- else }}
	case {{ .protoFieldID }}:
		if wireType != proto.WireTypeI{{ .bitSize }} {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		var num uint{{ .bitSize }}
		num, pos, err = proto.ConsumeI{{ .bitSize }}(buf, pos)
		if err != nil {
			return err
		}
{{ if ne .oneOfGroup "" -}}
		ofv := &{{ .oneOfMessageFullName }}{}
		ofv.{{ .fieldName }} = {{ .goType }}(num)
		orig.{{ .oneOfGroup }} = ofv
{{- else }}
		orig.{{ .fieldName }} = {{ .goType }}(num)
{{- end }}{{- end }}`

const unmarshalProtoBool = `{{ if .repeated -}}
	case {{ .protoFieldID }}:
		if wireType != proto.WireTypeLen {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		var length int
		length, pos, err = proto.ConsumeLen(buf, pos)
		if err != nil {
			return err
		}
		startPos := pos - length
		// Optimistically assume that bools are encoded as 1 byte even in variant form.
		orig.{{ .fieldName }} = make([]bool, 0, length)
		var num uint64
		for startPos < pos {
			num, startPos, err = proto.ConsumeVarint(buf[:pos], startPos)
			if err != nil {
				return err
			}
			orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, num != 0)
		}
		if startPos != pos {
			return fmt.Errorf("proto: invalid field len = %d for field {{ .fieldName }}", pos - startPos)
		}
{{- else }}
	case {{ .protoFieldID }}:
		if wireType != proto.WireTypeVarint {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		var num uint64
		num, pos, err = proto.ConsumeVarint(buf, pos)
		if err != nil {
			return err
		}
{{ if ne .oneOfGroup "" -}}
		ofv := &{{ .oneOfMessageFullName }}{}
		ofv.{{ .fieldName }} = num != 0
		orig.{{ .oneOfGroup }} = ofv
{{- else }}
		orig.{{ .fieldName }} = num != 0
{{- end }}{{- end }}`

const unmarshalProtoVarint = `{{ if .repeated -}}
	case {{ .protoFieldID }}:
		if wireType != proto.WireTypeLen {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		var length int
		length, pos, err = proto.ConsumeLen(buf, pos)
		if err != nil {
			return err
		}
		startPos := pos - length
		var num uint64
		for startPos < pos {
			num, startPos, err = proto.ConsumeVarint(buf[:pos], startPos)
			if err != nil {
				return err
			}
			orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, {{ .goType }}(num))
		}
		if startPos != pos {
			return fmt.Errorf("proto: invalid field len = %d for field {{ .fieldName }}", pos - startPos)
		}
{{- else }}
	case {{ .protoFieldID }}:
		if wireType != proto.WireTypeVarint {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		var num uint64
		num, pos, err = proto.ConsumeVarint(buf, pos)
		if err != nil {
			return err
		}
{{ if ne .oneOfGroup "" -}}
		ofv := &{{ .oneOfMessageFullName }}{}
		ofv.{{ .fieldName }} = {{ .goType }}(num)
		orig.{{ .oneOfGroup }} = ofv
{{- else }}
		orig.{{ .fieldName }} = {{ .goType }}(num)
{{- end }}{{- end }}`

const unmarshalProtoString = `
	case {{ .protoFieldID }}:
		if wireType != proto.WireTypeLen {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		var length int
		length, pos, err = proto.ConsumeLen(buf, pos)
		if err != nil {
			return err
		}
		startPos := pos - length
{{ if ne .oneOfGroup "" -}}
		ofv := &{{ .oneOfMessageFullName }}{}
		ofv.{{ .fieldName }} = string(buf[startPos:pos])
		orig.{{ .oneOfGroup }} = ofv
{{- else if .repeated -}}
		orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, string(buf[startPos:pos]))
{{- else -}}
		orig.{{ .fieldName }} = string(buf[startPos:pos])
{{- end }}`

const unmarshalProtoBytes = `	
	case {{ .protoFieldID }}:
		if wireType != proto.WireTypeLen {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		var length int
		length, pos, err = proto.ConsumeLen(buf, pos)
		if err != nil {
			return err
		}
		startPos := pos - length
{{ if ne .oneOfGroup "" -}}
		ofv := &{{ .oneOfMessageFullName }}{}
		ofv.{{ .fieldName }} = make([]byte, length)
		copy(ofv.{{ .fieldName }}, buf[startPos:pos])
		orig.{{ .oneOfGroup }} = ofv
{{- else if .repeated -}}
		orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, make([]byte, length))
		copy(orig.{{ .fieldName }}[len(orig.{{ .fieldName }}) - 1], buf[startPos:pos])
{{- else -}}
		orig.{{ .fieldName }} = make([]byte, length)
		copy(orig.{{ .fieldName }}, buf[startPos:pos])
{{- end }}`

const unmarshalProtoMessage = `
	case {{ .protoFieldID }}:
		if wireType != proto.WireTypeLen {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		var length int
		length, pos, err = proto.ConsumeLen(buf, pos)
		if err != nil {
			return err
		}
		startPos := pos - length
{{ if ne .oneOfGroup "" -}}
		ofv := &{{ .oneOfMessageFullName }}{}
		ofv.{{ .fieldName }} = NewOrigPtr{{ .origName }}()
		err = UnmarshalProtoOrig{{ .origName }}(ofv.{{ .fieldName }}, buf[startPos:pos])
		if err != nil {
			return err
		}
		orig.{{ .oneOfGroup }} = ofv
{{- else if .repeated -}}
		orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, NewOrig{{ if .nullable }}Ptr{{ end }}{{ .origName }}())
		err = UnmarshalProtoOrig{{ .origName }}({{ if not .nullable }}&{{ end }}orig.{{ .fieldName }}[len(orig.{{ .fieldName }})-1], buf[startPos:pos])
		if err != nil {
			return err
		}
{{- else }}
		{{ if .nullable }}orig.{{ .fieldName }} = NewOrigPtr{{ .origName }}(){{ end }}
		err = UnmarshalProtoOrig{{ .origName }}({{ if not .nullable }}&{{ end }}orig.{{ .fieldName }}, buf[startPos:pos]) 
		if err != nil {
			return err
		}
{{- end }}`

const unmarshalProtoSignedVarint = `{{ if .repeated -}}
	case {{ .protoFieldID }}:
		if wireType != proto.WireTypeLen {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		var length int
		length, pos, err = proto.ConsumeLen(buf, pos)
		if err != nil {
			return err
		}
		startPos := pos - length
		// Optimistically assume that bools are encoded as 1 byte even in variant form.
		orig.{{ .fieldName }} = make([]bool, 0, pos - startPos)
		var num uint64
		for startPos < pos {
			num, startPos, err = proto.ConsumeVarint(buf[:pos], startPos)
			if err != nil {
				return err
			}
			orig.{{ .fieldName }} = append(orig.{{ .fieldName }}, int{{ .bitSize }}(uint{{ .bitSize }}(num >> 1) ^ uint{{ .bitSize }}(int{{ .bitSize }}((num&1)<<{{ sub .bitSize 1 }})>>{{ sub .bitSize 1 }})))
		}
		if startPos != pos {
			return fmt.Errorf("proto: invalid field len = %d for field {{ .fieldName }}", pos - startPos)
		}
{{- else }}
	case {{ .protoFieldID }}:
		if wireType != proto.WireTypeVarint {
			return fmt.Errorf("proto: wrong wireType = %d for field {{ .fieldName }}", wireType)
		}
		var num uint64
		num, pos, err = proto.ConsumeVarint(buf, pos)
		if err != nil {
			return err
		}
{{ if ne .oneOfGroup "" -}}
		orig.{{ .oneOfGroup }} = &{{ .oneOfMessageFullName }} { {{ .fieldName }}: int{{ .bitSize }}(uint{{ .bitSize }}(num >> 1) ^ uint{{ .bitSize }}(int{{ .bitSize }}((num&1)<<{{ sub .bitSize 1 }})>>{{ sub .bitSize 1 }})) }
{{- else }}
		orig.{{ .fieldName }} = int{{ .bitSize }}(uint{{ .bitSize }}(num >> 1) ^ uint{{ .bitSize }}(int{{ .bitSize }}((num&1)<<{{ sub .bitSize 1 }})>>{{ sub .bitSize 1 }}))
{{- end }}{{- end }}`

func (pf *ProtoField) genUnmarshalProto() string {
	tf := pf.getTemplateFields()
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
