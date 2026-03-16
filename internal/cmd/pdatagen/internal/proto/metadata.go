// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/tmplutil"
)

const metadataMessageTemplate = `
{{- range .OptionalFields }}
const fieldBlock{{ $.Name }}{{ .Name }} = uint64({{ .Value }} >> 6)
const fieldBit{{ $.Name }}{{ .Name }} = uint64(1 << {{ .Value }} & 0x3F)

func (m *{{ $.Name }}) Set{{ .Name }}(value {{ .GoType }}) {
	m.{{ .Name }} = value
	m.metadata[fieldBlock{{ $.Name }}{{ .Name }}] |= fieldBit{{ $.Name }}{{ .Name }}
}

func (m *{{ $.Name }}) Remove{{ .Name }}() {
	m.{{ .Name }} = {{ .DefaultValue }}
	m.metadata[fieldBlock{{ $.Name }}{{ .Name }}] &^= fieldBit{{ $.Name }}{{ .Name }}
}

func (m *{{ $.Name }}) Has{{ .Name }}() bool {
	return m.metadata[fieldBlock{{ $.Name }}{{ .Name }}] & fieldBit{{ $.Name }}{{ .Name }} != 0
}
{{- end }}

`

type Metadata struct {
	Name           string
	OptionalFields []*MetadataOptionalField
}

type MetadataOptionalField struct {
	*Field
	Value int
}

func newMetadata(ms *Message) *Metadata {
	meta := &Metadata{
		Name: ms.Name,
	}
	value := 0
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
			Value: value,
		})
		value++
	}
	if len(meta.OptionalFields) == 0 {
		return nil
	}
	return meta
}

func (meta *Metadata) Generate() []byte {
	return []byte(tmplutil.Execute(tmplutil.Parse("metadataMessageTemplate", []byte(metadataMessageTemplate)), meta))
}
