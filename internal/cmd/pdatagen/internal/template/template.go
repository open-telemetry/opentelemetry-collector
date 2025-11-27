// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package template // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"

import (
	"strings"
	"text/template"

	"github.com/ettle/strcase"
)

func Parse(name string, bytes []byte) *template.Template {
	return template.Must(newTemplate(name).Parse(string(bytes)))
}

func Execute(tmpl *template.Template, data any) string {
	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		panic(err)
	}
	return sb.String()
}

func newTemplate(name string) *template.Template {
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
