// Copyright 2020 OpenTelemetry Authors
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

package scaffold

const Gomod = `
module {{.Distribution.Module}}

go 1.15

require (
	{{- range .Extensions}}
	{{if .GoMod}}{{.GoMod}}{{end}}
	{{- end}}
	{{- range .Receivers}}
	{{if .GoMod}}{{.GoMod}}{{end}}
	{{- end}}
	{{- range .Exporters}}
	{{if .GoMod}}{{.GoMod}}{{end}}
	{{- end}}
	{{- range .Processors}}
	{{if .GoMod}}{{.GoMod}}{{end}}
	{{- end}}
	go.opentelemetry.io/collector v{{.Distribution.OtelColVersion}}
)

{{- range .Extensions}}
{{if ne .Path ""}}replace {{.GoMod}} => {{.Path}}{{end}}
{{- end}}
{{- range .Receivers}}
{{if ne .Path ""}}replace {{.GoMod}} => {{.Path}}{{end}}
{{- end}}
{{- range .Exporters}}
{{if ne .Path ""}}replace {{.GoMod}} => {{.Path}}{{end}}
{{- end}}
{{- range .Processors}}
{{if ne .Path ""}}replace {{.GoMod}} => {{.Path}}{{end}}
{{- end}}
{{- range .Replaces}}
replace {{.}}
{{- end}}
`
