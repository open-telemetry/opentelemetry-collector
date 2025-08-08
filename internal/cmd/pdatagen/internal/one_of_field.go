// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"strings"
	"text/template"
)

const oneOfAccessorTemplate = `// {{ .typeFuncName }} returns the type of the {{ .lowerOriginFieldName }} for this {{ .structName }}.
// Calling this function on zero-initialized {{ .structName }} will cause a panic.
func (ms {{ .structName }}) {{ .typeFuncName }}() {{ .typeName }} {
	switch ms.{{ .origAccessor }}.{{ .originFieldName }}.(type) {
		{{- range .values }}
		{{ .GenerateType $.baseStruct $.OneOfField }}
		{{- end }}
	}
	return {{ .typeName }}Empty
}

{{ range .values }}
{{ .GenerateAccessors $.baseStruct $.OneOfField }}
{{ end }}`

const oneOfAccessorTestTemplate = `func Test{{ .structName }}_{{ .typeFuncName }}(t *testing.T) {
	tv := New{{ .structName }}()
	assert.Equal(t, {{ .typeName }}Empty, tv.{{ .typeFuncName }}())
}

{{ range .values -}}
{{ .GenerateTests $.baseStruct $.OneOfField }}
{{ end }}
`

const oneOfCopyOrigTemplate = `switch t := src.{{ .originFieldName }}.(type) {
{{- range .values }}
{{ .GenerateCopyOrig $.baseStruct $.OneOfField }}
{{- end }}
}`

const oneOfMarshalJSONTemplate = `switch ov := orig.{{ .originFieldName }}.(type) {
		{{- range .values }}
		{{ .GenerateMarshalJSON $.baseStruct $.OneOfField }}
		{{- end }}
	}`

const oneOfUnmarshalJSONTemplate = `
	{{- range .values }}
	{{ .GenerateUnmarshalJSON $.baseStruct $.OneOfField }}
	{{- end }}`

const oneOfSizeProtoTemplate = `switch orig.{{ .originFieldName }}.(type) {
	{{- range .values }}
	case *{{ $.originTypePrefix }}{{ .GetOriginFieldName }}:
		{{ .GenerateSizeProto $.baseStruct $.OneOfField }}
	{{- end }}
}`

type OneOfField struct {
	originFieldName            string
	typeName                   string
	testValueIdx               int
	values                     []oneOfValue
	omitOriginFieldNameInNames bool
}

func (of *OneOfField) GenerateAccessors(ms *messageStruct) string {
	t := template.Must(templateNew("oneOfAccessorTemplate").Parse(oneOfAccessorTemplate))
	return executeTemplate(t, of.templateFields(ms))
}

func (of *OneOfField) typeFuncName() string {
	const typeSuffix = "Type"
	if of.omitOriginFieldNameInNames {
		return typeSuffix
	}
	return of.originFieldName + typeSuffix
}

func (of *OneOfField) GenerateAccessorsTest(ms *messageStruct) string {
	t := template.Must(templateNew("oneOfAccessorTestTemplate").Parse(oneOfAccessorTestTemplate))
	return executeTemplate(t, of.templateFields(ms))
}

func (of *OneOfField) GenerateSetWithTestValue(ms *messageStruct) string {
	return of.values[of.testValueIdx].GenerateSetWithTestValue(ms, of)
}

func (of *OneOfField) GenerateCopyOrig(ms *messageStruct) string {
	t := template.Must(templateNew("oneOfCopyOrigTemplate").Parse(oneOfCopyOrigTemplate))
	return executeTemplate(t, of.templateFields(ms))
}

func (of *OneOfField) GenerateMarshalJSON(ms *messageStruct) string {
	t := template.Must(templateNew("oneOfMarshalJSONTemplate").Parse(oneOfMarshalJSONTemplate))
	return executeTemplate(t, of.templateFields(ms))
}

func (of *OneOfField) GenerateUnmarshalJSON(ms *messageStruct) string {
	t := template.Must(templateNew("oneOfUnmarshalJSONTemplate").Parse(oneOfUnmarshalJSONTemplate))
	return executeTemplate(t, of.templateFields(ms))
}

func (of *OneOfField) GenerateSizeProto(ms *messageStruct) string {
	t := template.Must(templateNew("oneOfSizeProtoTemplate").Parse(oneOfSizeProtoTemplate))
	return executeTemplate(t, of.templateFields(ms))
}

func (of *OneOfField) templateFields(ms *messageStruct) map[string]any {
	return map[string]any{
		"baseStruct":           ms,
		"OneOfField":           of,
		"packageName":          "",
		"structName":           ms.getName(),
		"typeFuncName":         of.typeFuncName(),
		"typeName":             of.typeName,
		"originFieldName":      of.originFieldName,
		"lowerOriginFieldName": strings.ToLower(of.originFieldName),
		"origAccessor":         origAccessor(ms.packageName),
		"stateAccessor":        stateAccessor(ms.packageName),
		"values":               of.values,
		"originTypePrefix":     ms.originFullName + "_",
	}
}

var _ Field = (*OneOfField)(nil)

type oneOfValue interface {
	GetOriginFieldName() string
	GenerateAccessors(ms *messageStruct, of *OneOfField) string
	GenerateTests(ms *messageStruct, of *OneOfField) string
	GenerateSetWithTestValue(ms *messageStruct, of *OneOfField) string
	GenerateCopyOrig(ms *messageStruct, of *OneOfField) string
	GenerateType(ms *messageStruct, of *OneOfField) string
	GenerateMarshalJSON(ms *messageStruct, of *OneOfField) string
	GenerateUnmarshalJSON(ms *messageStruct, of *OneOfField) string
	GenerateSizeProto(ms *messageStruct, of *OneOfField) string
}
