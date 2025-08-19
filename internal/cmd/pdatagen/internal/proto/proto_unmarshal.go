// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"fmt"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
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

func (pf *Field) GenUnmarshalProto() string {
	tf := pf.getTemplateFields()
	switch pf.Type {
	case TypeDouble, TypeFloat:
		return template.Execute(template.Parse("unmarshalProtoFloat", []byte(unmarshalProtoFloat)), tf)
	case TypeFixed64, TypeSFixed64, TypeFixed32, TypeSFixed32:
		return template.Execute(template.Parse("unmarshalProtoFixed", []byte(unmarshalProtoFixed)), tf)
	case TypeInt32, TypeInt64, TypeUint32, TypeUint64, TypeEnum:
		return template.Execute(template.Parse("unmarshalProtoVarint", []byte(unmarshalProtoVarint)), tf)
	case TypeBool:
		return template.Execute(template.Parse("unmarshalProtoBool", []byte(unmarshalProtoBool)), tf)
	case TypeString:
		return template.Execute(template.Parse("unmarshalProtoString", []byte(unmarshalProtoString)), tf)
	case TypeBytes:
		return template.Execute(template.Parse("unmarshalProtoBytes", []byte(unmarshalProtoBytes)), tf)
	case TypeMessage:
		return template.Execute(template.Parse("unmarshalProtoMessage", []byte(unmarshalProtoMessage)), tf)
	case TypeSInt32, TypeSInt64:
		return template.Execute(template.Parse("unmarshalProtoSignedVarint", []byte(unmarshalProtoSignedVarint)), tf)
	}
	panic(fmt.Sprintf("unhandled case %T", pf.Type))
}
