package proto

import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

type Message struct {
	Name          string
	Fields        []*Field
	AliasFullName string
}

func (ms *Message) GenerateProtoFile() []byte {
	return []byte(template.Execute(messageProtoTemplate, ms.templateFields()))
}

func (ms *Message) GenerateProtoTestsFile() []byte {
	return []byte(template.Execute(messageProtoTestTemplate, ms.templateFields()))
}

func (ms *Message) templateFields() map[string]any {
	return map[string]any{
		"MessageS":      ms,
		"MessageName":   ms.Name,
		"Fields":        ms.Fields,
		"AliasFullName": ms.AliasFullName,
	}
}
