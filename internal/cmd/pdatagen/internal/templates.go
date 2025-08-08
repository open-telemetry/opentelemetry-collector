// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	_ "embed"
	"strings"
	"text/template"

	"github.com/ettle/strcase"
)

var (
	//go:embed templates/message.go.tmpl
	messageTemplateBytes []byte
	messageTemplate      = parseTemplate("message.go", messageTemplateBytes)

	//go:embed templates/message_internal.go.tmpl
	messageInternalTemplateBytes []byte
	messageInternalTemplate      = parseTemplate("message_internal.go", messageInternalTemplateBytes)

	//go:embed templates/message_internal_test.go.tmpl
	messageInternalTestTemplateBytes []byte
	messageInternalTestTemplate      = parseTemplate("message_internal_test.go", messageInternalTestTemplateBytes)

	//go:embed templates/message_test.go.tmpl
	messageTestTemplateBytes []byte
	messageTestTemplate      = parseTemplate("message_test.go", messageTestTemplateBytes)

	//go:embed templates/primitive_slice.go.tmpl
	primitiveSliceTemplateBytes []byte
	primitiveSliceTemplate      = parseTemplate("primitive_slice.go", primitiveSliceTemplateBytes)

	//go:embed templates/primitive_slice_internal.go.tmpl
	primitiveSliceInternalTemplateBytes []byte
	primitiveSliceInternalTemplate      = parseTemplate("primitive_slice_internal.go", primitiveSliceInternalTemplateBytes)

	//go:embed templates/primitive_slice_internal_test.go.tmpl
	primitiveSliceInternalTestTemplateBytes []byte
	primitiveSliceInternalTestTemplate      = parseTemplate("primitive_slice_internal_test.go", primitiveSliceInternalTestTemplateBytes)

	//go:embed templates/primitive_slice_test.go.tmpl
	primitiveSliceTestTemplateBytes []byte
	primitiveSliceTestTemplate      = parseTemplate("primitive_slice_test.go", primitiveSliceTestTemplateBytes)

	//go:embed templates/slice.go.tmpl
	sliceTemplateBytes []byte
	sliceTemplate      = parseTemplate("slice.go", sliceTemplateBytes)

	//go:embed templates/slice_internal.go.tmpl
	sliceInternalTemplateBytes []byte
	sliceInternalTemplate      = parseTemplate("slice_internal.go", sliceInternalTemplateBytes)

	//go:embed templates/slice_internal_test.go.tmpl
	sliceInternalTestTemplateBytes []byte
	sliceInternalTestTemplate      = parseTemplate("slice_internal_test.go", sliceInternalTestTemplateBytes)

	//go:embed templates/slice_test.go.tmpl
	sliceTestTemplateBytes []byte
	sliceTestTemplate      = parseTemplate("slice_test.go", sliceTestTemplateBytes)
)

func parseTemplate(name string, bytes []byte) *template.Template {
	return template.Must(templateNew(name).Parse(string(bytes)))
}

func executeTemplate(tmpl *template.Template, data any) string {
	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		panic(err)
	}
	return sb.String()
}

func templateNew(name string) *template.Template {
	return template.New(name).Funcs(template.FuncMap{
		"upperFirst": upperFirst,
		"lowerFirst": lowerFirst,
		"add":        add,
		"sub":        sub,
		"div":        div,
		"needSnake":  needSnake,
		"toSnake":    strcase.ToSnake,
	})
}

func upperFirst(s string) string {
	return strings.ToUpper(s[0:1]) + s[1:]
}

func lowerFirst(s string) string {
	return strings.ToLower(s[0:1]) + s[1:]
}

func add(a, b int) int {
	return a + b
}

func sub(a, b int) int {
	return a - b
}

func div(a, b int) int {
	return a / b
}

func needSnake(str string) bool {
	return strings.ToLower(str) != strcase.ToSnake(str)
}
