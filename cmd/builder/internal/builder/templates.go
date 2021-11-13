// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
