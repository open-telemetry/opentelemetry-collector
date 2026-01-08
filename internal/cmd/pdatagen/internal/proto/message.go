// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	_ "embed"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

var (
	//go:embed templates/message.go.tmpl
	messageTemplateBytes []byte
	messageTemplate      = template.Parse("message_internal_test.go", messageTemplateBytes)

	//go:embed templates/message_test.go.tmpl
	messageTestTemplateBytes []byte
	messageTestTemplate      = template.Parse("message_internal_test.go", messageTestTemplateBytes)
)

type Message struct {
	Name            string
	Description     string
	OriginFullName  string
	UpstreamMessage string
	Fields          []FieldInterface
	metadata        *Metadata
}

func (ms *Message) GenerateMessage(imports, testImports []string) []byte {
	ms.metadata = newMetadata(ms)
	return []byte(template.Execute(messageTemplate, ms.templateFields(imports, testImports)))
}

func (ms *Message) GenerateMessageTests(imports, testImports []string) []byte {
	return []byte(template.Execute(messageTestTemplate, ms.templateFields(imports, testImports)))
}

func (ms *Message) GenerateMetadata() string {
	return string(ms.metadata.Generate())
}

func (ms *Message) templateFields(imports, testImports []string) map[string]any {
	return map[string]any{
		"fields":          ms.Fields,
		"messageName":     ms.Name,
		"upstreamMessage": ms.UpstreamMessage,
		"description":     ms.Description,
		"imports":         imports,
		"testImports":     testImports,
		// 0 size means no metadata is needed
		"hasMetadata":      ms.hasMetadata(),
		"GenerateMetadata": ms.GenerateMetadata,
	}
}

func (ms *Message) hasMetadata() bool {
	if ms.metadata == nil || ms.metadata.Size() == 0 {
		return false
	}

	return true
}
