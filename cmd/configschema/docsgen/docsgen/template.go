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
	"strings"
	"text/template"
)

func tableTemplate() (*template.Template, error) {
	return template.New("table").Funcs(
		template.FuncMap{
			"join":            join,
			"cleanType":       cleanType,
			"isCompoundField": isCompoundField,
			"isDuration":      isDuration,
		},
	).Parse(tableTemplateStr)
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

func isDuration(s string) bool {
	return s == "time.Duration"
}

const tableTemplateStr = `### {{ cleanType .Type }}

| Name | Type | Default | Docs |
| ---- | ---- | ------- | ---- |
{{ range .Fields -}}
| {{ .Name }} |
{{- if .Type -}}
    {{- if isCompoundField .Kind -}}
        [{{ cleanType .Type }}](#{{ cleanType .Type }})
    {{- else -}}
		{{- if isDuration .Type -}}
			[{{ cleanType .Type }}](#{{ cleanType .Type }})
        {{- else -}}
            {{ .Type }}
        {{- end -}}
    {{- end -}}
{{- else -}}
    {{ .Kind }}
{{- end -}}
| {{ .Default }} | {{ join .Doc }} |
{{ end }}
`
