// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package docsgen

import (
	"path"
	"strings"
	"text/template"
)

func headerTemplate(templateDir string) (*template.Template, error) {
	const fname = "header.tmpl"
	tmpl := template.New(fname)
	tmpl = tmpl.Funcs(template.FuncMap{
		"packageName": packageName,
	})
	return tmpl.ParseFiles(path.Join(templateDir, fname))
}

func tableTemplate(templateDir string) (*template.Template, error) {
	const fname = "table.tmpl"
	tmpl := template.New(fname)
	tmpl = tmpl.Funcs(
		template.FuncMap{
			"join":            join,
			"cleanType":       cleanType,
			"isCompoundField": isCompoundField,
		},
	)
	return tmpl.ParseFiles(path.Join(templateDir, fname))
}

func isCompoundField(kind string) bool {
	return kind == "struct" || kind == "ptr"
}

func join(s string) string {
	return strings.ReplaceAll(s, "\n", " ")
}

func cleanType(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "*", ""), ".", "-")
}

// packageName takes an input, e.g. "*otlpreceiver.Config" and return "otlpreceiver"
func packageName(s string) string {
	dotIdx := strings.Index(s, ".")
	replaced := strings.ReplaceAll(s[:dotIdx], "*", "")
	return strings.Title(replaced)
}
