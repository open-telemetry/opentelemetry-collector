// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	_ "embed" // Blank import required for go:embed to work.

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

var (
	//go:embed templates/message_proto.go.tmpl
	messageProtoTemplateBytes []byte
	messageProtoTemplate      = template.Parse("message_proto.go", messageProtoTemplateBytes)

	//go:embed templates/message_proto_test.go.tmpl
	messageProtoTestTemplateBytes []byte
	messageProtoTestTemplate      = template.Parse("message_proto_test.go", messageProtoTestTemplateBytes)
)
