// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builder // import "go.opentelemetry.io/collector/cmd/builder/internal/builder"

import (
	_ "embed"
	"text/template"
)

var (
	//go:embed templates/components.go.tmpl
	componentsBytes    []byte
	componentsTemplate = parseTemplate("components.go", componentsBytes)

	//go:embed templates/main.go.tmpl
	mainBytes    []byte
	mainTemplate = parseTemplate("main.go", mainBytes)

	//go:embed templates/main_others.go.tmpl
	mainOthersBytes    []byte
	mainOthersTemplate = parseTemplate("main_others.go", mainOthersBytes)

	//go:embed templates/main_windows.go.tmpl
	mainWindowsBytes    []byte
	mainWindowsTemplate = parseTemplate("main_windows.go", mainWindowsBytes)

	//go:embed templates/go.mod.tmpl
	goModBytes    []byte
	goModTemplate = parseTemplate("go.mod", goModBytes)
)

func parseTemplate(name string, bytes []byte) *template.Template {
	return template.Must(template.New(name).Parse(string(bytes)))
}
