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

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

import (
	"bytes"
	"strings"
	"text/template"
)

const messageValueTemplate = `{{ .description }}
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use New{{ .structName }} function to create new instances.
// Important: zero-initialized instance is not valid for use.
{{- if .isCommon }}
type {{ .structName }} internal.{{ .structName }}
{{- else }}
type {{ .structName }} struct {
	parent {{ .parent }}
	{{- if .isSliceElement }}
	idx int
	{{- end }}
}

{{ if .parentUnspecified }}
type stub{{ .structName }}Parent struct {
	orig *{{ .originName }}
}

func (vp stub{{ .structName }}Parent) EnsureMutability() {}

func (vp stub{{ .structName }}Parent) GetChildOrig() *{{ .originName }} {
	return vp.orig
}

var _ {{ .parent }} = (*stub{{ .structName }}Parent)(nil)
{{- end }}
{{- end }}

func (ms {{ .structName }}) getOrig() *{{ .originName }} {
	{{- if .isCommon }}
	return internal.{{ .structName }}(ms).GetOrig()
	{{- else if .isSliceElement }}
	return ms.parent.getChildOrig(ms.idx)
	{{- else if .parentUnspecified }}
	return ms.parent.GetChildOrig()
	{{- else }}
	return ms.parent.get{{ .structName }}Orig()
	{{- end }}
}

func (ms {{ .structName }}) ensureMutability() {
	{{- if .isCommon }}
	internal.{{ .structName }}(ms).EnsureMutability()
	{{- else if .parentUnspecified }}
	ms.parent.EnsureMutability()
	{{- else }}
	ms.parent.ensureMutability()
	{{- end }}
}

{{ if not .isCommon }}
{{- range .fields }}
{{ .GenerateParentMethods $.messageStruct }}
{{ end }}

func new{{ .structName }}FromOrig(orig *{{ .originName }}) {{ .structName }} {
	{{- if .parentUnspecified }}
	return {{ .structName }}{parent: &stub{{ .structName }}Parent{orig: orig}}
	{{- else  if .isSliceElement }}
	return {{ .structName }}{parent: new{{ .parent }}FromElementOrig(orig)}
	{{- else }}
	return {{ .structName }}{parent: new{{ .parent }}From{{ .structName }}Orig(orig)}
	{{- end }}
}

func new{{ .structName }}FromParent(parent {{ .parent }}{{- if .isSliceElement }}, idx int{{ end }}) {{ .structName }} {
	return {{ .structName }}{parent: parent{{ if .isSliceElement }}, idx: idx{{ end }}}
}
{{- end }}

// New{{ .structName }} creates a new empty {{ .structName }}.
//
// This must be used only in testing code. Users should use "AppendEmpty" when part of a Slice,
// OR directly access the member if this is embedded in another struct.
func New{{ .structName }}() {{ .structName }} {
	{{- if .isCommon }}
	return {{ .structName }}(internal.New{{ .structName }}FromOrig(&{{ .originName }}{}))
	{{- else }}
	return new{{ .structName }}FromOrig(&{{ .originName }}{})
	{{- end }}
}

// MoveTo moves all properties from the current struct overriding the destination and
// resetting the current instance to its zero value
func (ms {{ .structName }}) MoveTo(dest {{ .structName }}) {
	ms.ensureMutability()
	dest.ensureMutability()
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = {{ .originName }}{}
}

{{ range .fields -}}
{{ .GenerateAccessors $.messageStruct }}
{{ end }}

// CopyTo copies all properties from the current struct overriding the destination.
func (ms {{ .structName }}) CopyTo(dest {{ .structName }}) {
	dest.ensureMutability()
	{{- range .fields }}
	{{ .GenerateCopyToValue $.messageStruct }}
	{{- end }}
}`

const messageValueTestTemplate = `
func Test{{ .structName }}_MoveTo(t *testing.T) {
	ms := {{ .generateTestData }}
	dest := New{{ .structName }}()
	ms.MoveTo(dest)
	assert.Equal(t, New{{ .structName }}(), ms)
	assert.Equal(t, {{ .generateTestData }}, dest)
}

func Test{{ .structName }}_CopyTo(t *testing.T) {
	ms := New{{ .structName }}()
	orig := New{{ .structName }}()
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
	orig = {{ .generateTestData }}
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
}

{{ range .fields }}
{{ .GenerateAccessorsTest $.messageStruct }}
{{ end }}`

const messageValueGenerateTestTemplate = `func {{ if .isCommon }}G{{ else }}g{{ end }}enerateTest{{ .structName }}() {{ .structName }} {
	{{- if .isCommon }}
	tv := New{{ .structName }}FromOrig(&{{ .originName }}{}) 
	FillTest{{ .structName }}(tv)
	{{- else }}
	tv := New{{ .structName }}()
	fillTest{{ .structName }}(tv)
	{{- end }}
	return tv
}

func {{ if .isCommon }}F{{ else }}f{{ end }}illTest{{ .structName }}(tv {{ .structName }}) {
	{{- range .fields }}
	{{ .GenerateSetWithTestValue $.messageStruct }}
	{{- end }}
}`

const messageValueAliasTemplate = `
type {{ .structName }} struct {
	parent {{ .parent }}
	{{- if .isSliceElement }}
	idx int
	{{- end }}
}

type stub{{ .structName }}Parent struct {
	orig *{{ .originName }}
}

func (vp stub{{ .structName }}Parent) EnsureMutability() {}

func (vp stub{{ .structName }}Parent) GetChildOrig() *{{ .originName }} {
	return vp.orig
}

var _ {{ .parent }} = (*stub{{ .structName }}Parent)(nil)

func (ms {{ .structName }}) GetOrig() *{{ .originName }} {
	return ms.parent.GetChildOrig({{ if .isSliceElement }}ms.idx{{ end }})
}

func (ms {{ .structName }}) EnsureMutability() {
	ms.parent.EnsureMutability()
}

func New{{ .structName }}FromOrig(orig *{{ .originName }}) {{ .structName }} {
	return {{ .structName }}{parent: &stub{{ .structName }}Parent{orig: orig}}
}

func New{{ .structName }}FromParent(parent {{ .parent }}) {{ .structName }} {
	return {{ .structName }}{parent: parent}
}

{{- range .fields }}
{{ .GenerateParentMethods $.messageStruct }}
{{ end }}`

type baseStruct interface {
	getName() string
	getPackageName() string
	getParent() string
	getOrigType() string
	generateStruct(sb *bytes.Buffer)
	generateTests(sb *bytes.Buffer)
	generateTestValueHelpers(sb *bytes.Buffer)
	generateInternal(sb *bytes.Buffer)
}

// messageValueStruct generates a struct for a proto message. The struct can be generated both as a common struct
// that can be used as a field in struct from other packages and as an isolated struct with depending on a package name.
type messageValueStruct struct {
	structName     string
	packageName    string
	description    string
	originFullName string
	parent         string
	fields         []baseField
}

func (ms *messageValueStruct) getName() string {
	return ms.structName
}

func (ms *messageValueStruct) getPackageName() string {
	return ms.packageName
}

func (ms *messageValueStruct) getParent() string {
	return ms.parent
}

func (ms *messageValueStruct) getOrigType() string {
	return ms.originFullName
}

func (ms *messageValueStruct) generateStruct(sb *bytes.Buffer) {
	t := template.Must(template.New("messageValueTemplate").Parse(messageValueTemplate))
	if err := t.Execute(sb, ms.templateFields(false)); err != nil {
		panic(err)
	}
}

func (ms *messageValueStruct) generateTests(sb *bytes.Buffer) {
	t := template.Must(template.New("messageValueTestTemplate").Parse(messageValueTestTemplate))
	if err := t.Execute(sb, ms.templateFields(false)); err != nil {
		panic(err)
	}
}

func (ms *messageValueStruct) generateTestValueHelpers(sb *bytes.Buffer) {
	t := template.Must(template.New("messageValueGenerateTestTemplate").Parse(messageValueGenerateTestTemplate))
	if err := t.Execute(sb, ms.templateFields(false)); err != nil {
		panic(err)
	}
}

func (ms *messageValueStruct) generateInternal(sb *bytes.Buffer) {
	t := template.Must(template.New("messageValueAliasTemplate").Parse(messageValueAliasTemplate))
	if err := t.Execute(sb, ms.templateFields(true)); err != nil {
		panic(err)
	}
}

func (ms *messageValueStruct) templateFields(forInternal bool) map[string]any {
	return map[string]any{
		"messageStruct": ms,
		"fields":        ms.fields,
		"structName":    ms.structName,
		"originName":    ms.originFullName,
		"parent": func() string {
			if ms.parent == "" {
				if forInternal {
					return "Parent[*" + ms.originFullName + "]"
				}
				return "internal.Parent[*" + ms.originFullName + "]"
			}
			return ms.parent
		}(),
		"isSliceElement":    strings.HasSuffix(ms.parent, "Slice"),
		"parentUnspecified": ms.parent == "",
		"generateTestData": func() string {
			if usedByOtherDataTypes(ms.packageName) {
				return ms.structName + "(internal.GenerateTest" + ms.structName + "())"
			}
			return "generateTest" + ms.structName + "()"
		}(),
		"description": ms.description,
		"isCommon":    usedByOtherDataTypes(ms.packageName),
	}
}

var _ baseStruct = (*messageValueStruct)(nil)
