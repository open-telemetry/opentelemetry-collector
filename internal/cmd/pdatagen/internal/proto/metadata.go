// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"math/bits"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const metadataMessageTemplate = `
{{- range .OptionalFields }}
const fieldBit{{ $.Name }}{{ .Name }} = uint64(1 << {{ .Value }})

func (m *{{ $.Name }}) Set{{ .Name }}(value {{ .GoType }}) {
	m.{{ .Name }} = value
	m.metadata |= fieldBit{{ $.Name }}{{ .Name }}
}

func (m *{{ $.Name }}) Remove{{ .Name }}() {
	m.{{ .Name }} = {{ .DefaultValue }}
	m.metadata &^= fieldBit{{ $.Name }}{{ .Name }}
}

func (m *{{ $.Name }}) Has{{ .Name }}() bool {
	return m.metadata & fieldBit{{ $.Name }}{{ .Name }} != 0
}
{{- end }}

{{- range $key, $value := .OneOfGroups }}
type {{ $.Name }}{{ $key }}Type int32

const (
	{{ $.Name }}{{ $key }}TypeEmpty {{ $.Name }}{{ $key }}Type = iota
	{{- range $index, $field := $value.Fields }}
	{{ $.Name }}{{ $key }}Type{{ $field.Name }}
	{{- end }}
)

const (
	startBit{{ $.Name }}{{ $key }} = uint64({{ .StartBit }})
	mask{{ $.Name }}{{ $key }} = uint64({{ .Mask }})
	reversedMask{{ $.Name }}{{ $key }} = ^mask{{ $.Name }}{{ $key }}
	{{- range $index, $field := $value.Fields }}
	fieldMask{{ $.Name }}{{ $key }}{{ $field.Name }} = uint64({{ add $index 1 }} << startBit{{ $.Name }}{{ $key }})
	{{- end }}
)

func (m *{{ $.Name }}) {{ $key }}Type() {{ $.Name }}{{ $key }}Type {
	val := (m.metadata & mask{{ $.Name }}{{ $key }}) >> startBit{{ $.Name }}{{ $key }}
	return {{ $.Name }}{{ $key }}Type(val)
}

func (m *{{ $.Name }}) Reset{{ $key }}() {
	m.{{ $key }}.Reset()
	m.metadata &= reversedMask{{ $.Name }}{{ $key }}
}

{{- range $value.Fields }}
func (m *{{ $.Name }}) Set{{ .Name }}(value {{ .MemberGoType }}) {
	{{ if eq .OneOfType "Int" -}}
	m.{{ $key }}.SetInt(uint64(value))
	{{ else if eq .OneOfType "Float" -}}
	m.{{ $key }}.SetFloat(float64(value))
	{{ else if eq .OneOfType "Message" -}}
	m.{{ $key }}.SetMessage(unsafe.Pointer(value))
	{{ else if eq .OneOfType "Bytes" -}}
	m.{{ $key }}.SetBytes(&value)
	{{ else -}}
	m.{{ $key }}.Set{{ .OneOfType }}(value)
	{{ end -}}
	m.metadata &= reversedMask{{ $.Name }}{{ $key }}
	m.metadata |= fieldMask{{ $.Name }}{{ $key }}{{ .Name }}
}

func (m *{{ $.Name }}) {{ .Name }}() {{ .MemberGoType }} {
	if m.{{ $key }}Type() != {{ $.Name }}{{ $key }}Type{{ .Name }} {
		{{ if eq .OneOfType "Message" -}}
		return nil
		{{ else -}}
		return {{ .DefaultValue }}
		{{ end -}}
	}
	{{ if eq .OneOfType "Bytes" -}}
	return *m.{{ $key }}.Bytes()
	{{ else -}}
	return ({{ .MemberGoType }})(m.{{ $key }}.{{ .OneOfType }}())
	{{ end -}}
}

{{ if eq .OneOfType "Bytes" -}}
func (m *{{ $.Name }}) {{ .Name }}Ptr() *[]byte {
	if m.{{ $key }}Type() != {{ $.Name }}{{ $key }}Type{{ .Name }} {
		return {{ .DefaultValue }}
	}
	return m.{{ $key }}.Bytes()
}{{ end -}}

{{- end }}
{{- end }}

`

type Metadata struct {
	Name           string
	OptionalFields []*MetadataOptionalField
	OneOfGroups    map[string]*MetadataOneOfFields
	size           int
}

type MetadataOptionalField struct {
	*Field
	Value int
}

type MetadataOneOfFields struct {
	Fields   []*Field
	StartBit int
	Mask     uint64
}

func newMetadata(ms *Message) *Metadata {
	meta := &Metadata{
		Name:        ms.Name,
		OneOfGroups: make(map[string]*MetadataOneOfFields),
	}
	for _, fieldI := range ms.Fields {
		field, ok := fieldI.(*Field)
		if !ok {
			continue
		}
		if field.Repeated {
			continue
		}
		switch field.Type {
		case TypeDouble, TypeFloat, TypeInt32, TypeInt64, TypeUint32, TypeUint64, TypeSInt32, TypeSInt64, TypeFixed32, TypeFixed64, TypeSFixed32, TypeSFixed64, TypeBool:
			if !field.Nullable {
				continue
			}
		default:
			continue
		}
		meta.OptionalFields = append(meta.OptionalFields, &MetadataOptionalField{
			Field: field,
			Value: meta.size,
		})
		meta.size++
	}
	for _, fieldI := range ms.Fields {
		field, ok := fieldI.(*OneOfField)
		if !ok {
			continue
		}
		// Len returns the minimum number of bits required to represent n,
		// so we don't need to do +1 because we can represent the number len(field.Fields) as well.
		b := bits.Len(uint(len(field.Fields)))
		mask := uint64(0)
		for i := meta.size; i < meta.size+b; i++ {
			mask |= 1 << uint64(i)
		}
		meta.OneOfGroups[field.GroupName] = &MetadataOneOfFields{
			Fields:   field.Fields,
			StartBit: meta.size,
			Mask:     mask,
		}
		meta.size += b
	}
	if meta.size >= 64 {
		panic("metadata size is too large, implement support for array metadata")
	}
	return meta
}

func (meta *Metadata) Size() int {
	return meta.size
}

func (meta *Metadata) Generate() []byte {
	return []byte(template.Execute(template.Parse("metadataMessageTemplate", []byte(metadataMessageTemplate)), meta))
}
