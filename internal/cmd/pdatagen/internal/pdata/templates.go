// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"

import (
	_ "embed" // Blank import required for go:embed to work.

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

var (
	//go:embed templates/message.go.tmpl
	messageTemplateBytes []byte
	messageTemplate      = template.Parse("message.go", messageTemplateBytes)

	//go:embed templates/message_internal.go.tmpl
	messageInternalTemplateBytes []byte
	messageInternalTemplate      = template.Parse("message_internal.go", messageInternalTemplateBytes)

	//go:embed templates/message_test.go.tmpl
	messageTestTemplateBytes []byte
	messageTestTemplate      = template.Parse("message_test.go", messageTestTemplateBytes)

	//go:embed templates/primitive_slice.go.tmpl
	primitiveSliceTemplateBytes []byte
	primitiveSliceTemplate      = template.Parse("primitive_slice.go", primitiveSliceTemplateBytes)

	//go:embed templates/primitive_slice_internal.go.tmpl
	primitiveSliceInternalTemplateBytes []byte
	primitiveSliceInternalTemplate      = template.Parse("primitive_slice_internal.go", primitiveSliceInternalTemplateBytes)

	//go:embed templates/primitive_slice_test.go.tmpl
	primitiveSliceTestTemplateBytes []byte
	primitiveSliceTestTemplate      = template.Parse("primitive_slice_test.go", primitiveSliceTestTemplateBytes)

	//go:embed templates/slice.go.tmpl
	sliceTemplateBytes []byte
	sliceTemplate      = template.Parse("slice.go", sliceTemplateBytes)

	//go:embed templates/slice_internal.go.tmpl
	sliceInternalTemplateBytes []byte
	sliceInternalTemplate      = template.Parse("slice_internal.go", sliceInternalTemplateBytes)

	//go:embed templates/slice_test.go.tmpl
	sliceTestTemplateBytes []byte
	sliceTestTemplate      = template.Parse("slice_test.go", sliceTestTemplateBytes)
)
