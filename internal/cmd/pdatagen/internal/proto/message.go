// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	_ "embed"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/tmplutil"
)

var (
	//go:embed templates/message.go.tmpl
	messageTemplateBytes []byte
	messageTemplate      = tmplutil.Parse("message_internal_test.go", messageTemplateBytes)

	//go:embed templates/message_test.go.tmpl
	messageTestTemplateBytes []byte
	messageTestTemplate      = tmplutil.Parse("message_internal_test.go", messageTestTemplateBytes)
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
	return []byte(tmplutil.Execute(messageTemplate, ms.templateFields(imports, testImports)))
}

func (ms *Message) GenerateMessageTests(imports, testImports []string) []byte {
	return []byte(tmplutil.Execute(messageTestTemplate, ms.templateFields(imports, testImports)))
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
		"metadataSize":     ms.metadataSize(),
		"GenerateMetadata": ms.GenerateMetadata,
	}
}

func (ms *Message) metadataSize() int {
	if ms.metadata == nil {
		return 0
	}

	if len(ms.metadata.OptionalFields) == 0 {
		return 0
	}

	return ms.metadata.OptionalFields[len(ms.metadata.OptionalFields)-1].Value/64 + 1
}
