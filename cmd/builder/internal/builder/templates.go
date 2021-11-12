package builder

import (
	_ "embed"
	"text/template"
)

var (
	//go:embed templates/components.go.tmpl
	componentsBytes    []byte
	componentsTemplate = parseTemplate("component_header.go.tmpl", componentsBytes)

	//go:embed templates/main.go.tmpl
	mainBytes    []byte
	mainTemplate = parseTemplate("main.go.tmpl", mainBytes)

	//go:embed templates/main_others.go.tmpl
	mainOthersBytes    []byte
	mainOthersTemplate = parseTemplate("main_others.go.tmpl", mainOthersBytes)

	//go:embed templates/main_windows.go.tmpl
	mainWindowsBytes    []byte
	mainWindowsTemplate = parseTemplate("main_windows.go.tmpl", mainWindowsBytes)

	//go:embed templates/go.mod.tmpl
	goModBytes    []byte
	goModTemplate = parseTemplate("go.mod.tmpl", goModBytes)
)

func parseTemplate(name string, bytes []byte) *template.Template {
	return template.Must(template.New(name).Parse(string(bytes)))
}
