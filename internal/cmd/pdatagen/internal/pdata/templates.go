// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"

import (
	_ "embed" // Blank import required for go:embed to work.

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/tmplutil"
)

var (
	//go:embed templates/message.go.tmpl
	messageTemplateBytes []byte
	messageTemplate      = tmplutil.Parse("message.go", messageTemplateBytes)

	//go:embed templates/message_internal.go.tmpl
	messageInternalTemplateBytes []byte
	messageInternalTemplate      = tmplutil.Parse("message_internal.go", messageInternalTemplateBytes)

	//go:embed templates/message_test.go.tmpl
	messageTestTemplateBytes []byte
	messageTestTemplate      = tmplutil.Parse("message_test.go", messageTestTemplateBytes)

	//go:embed templates/primitive_slice.go.tmpl
	primitiveSliceTemplateBytes []byte
	primitiveSliceTemplate      = tmplutil.Parse("primitive_slice.go", primitiveSliceTemplateBytes)

	//go:embed templates/primitive_slice_internal.go.tmpl
	primitiveSliceInternalTemplateBytes []byte
	primitiveSliceInternalTemplate      = tmplutil.Parse("primitive_slice_internal.go", primitiveSliceInternalTemplateBytes)

	//go:embed templates/primitive_slice_test.go.tmpl
	primitiveSliceTestTemplateBytes []byte
	primitiveSliceTestTemplate      = tmplutil.Parse("primitive_slice_test.go", primitiveSliceTestTemplateBytes)

	//go:embed templates/slice.go.tmpl
	sliceTemplateBytes []byte
	sliceTemplate      = tmplutil.Parse("slice.go", sliceTemplateBytes)

	//go:embed templates/slice_internal.go.tmpl
	sliceInternalTemplateBytes []byte
	sliceInternalTemplate      = tmplutil.Parse("slice_internal.go", sliceInternalTemplateBytes)

	//go:embed templates/slice_test.go.tmpl
	sliceTestTemplateBytes []byte
	sliceTestTemplate      = tmplutil.Parse("slice_test.go", sliceTestTemplateBytes)
)
