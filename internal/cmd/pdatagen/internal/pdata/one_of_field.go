// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"

import (
	"strings"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
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

const oneOfTestFailingUnmarshalProtoValuesTemplate = `
	{{- range .values }}
	{{ .GenerateTestFailingUnmarshalProtoValues $.baseStruct $.OneOfField }}
	{{- end }}`

const oneOfTestValuesTemplate = `
	{{- range .values }}
	{{ .GenerateTestEncodingValues $.baseStruct $.OneOfField }}
	{{- end }}`

const oneOfPoolOrigTemplate = `
	{{- range .values }}
		{{ .GeneratePoolOrig $.baseStruct $.OneOfField }}
	{{- end }}`

const oneOfDeleteOrigTemplate = `switch ov := orig.{{ .originFieldName }}.(type) {
	{{ range .values -}}
	case *{{ $.originTypePrefix }}{{ .GetOriginFieldName }}:
		{{ .GenerateDeleteOrig $.baseStruct $.OneOfField }}{{- end }}
	}
`

const oneOfCopyOrigTemplate = `switch t := src.{{ .originFieldName }}.(type) {
{{- range .values }}
{{ .GenerateCopyOrig $.baseStruct $.OneOfField }}
{{- end }}
}`

const oneOfMarshalJSONTemplate = `switch orig := orig.{{ .originFieldName }}.(type) {
	{{- range .values }}
	case *{{ $.originTypePrefix }}{{ .GetOriginFieldName }}:
		{{ .GenerateMarshalJSON $.baseStruct $.OneOfField }}
	{{- end }}
}`

const oneOfUnmarshalJSONTemplate = `
	{{- range .values }}
	{{ .GenerateUnmarshalJSON $.baseStruct $.OneOfField }}
	{{- end }}`

const oneOfSizeProtoTemplate = `switch orig := orig.{{ .originFieldName }}.(type) {
	case nil:
		_ = orig
		break
	{{ range .values -}}
	case *{{ $.originTypePrefix }}{{ .GetOriginFieldName }}:
		{{ .GenerateSizeProto $.baseStruct $.OneOfField }}
	{{ end -}}
}`

const oneOfMarshalProtoTemplate = `switch orig := orig.{{ .originFieldName }}.(type) {
	{{- range .values }}
	case *{{ $.originTypePrefix }}{{ .GetOriginFieldName }}:
		{{ .GenerateMarshalProto $.baseStruct $.OneOfField }}
	{{- end }}
}`

const oneOfUnmarshalProtoTemplate = `
	{{- range .values }}
		{{ .GenerateUnmarshalProto $.baseStruct $.OneOfField }}
	{{- end }}`

type OneOfField struct {
	originFieldName            string
	typeName                   string
	testValueIdx               int
	values                     []oneOfValue
	omitOriginFieldNameInNames bool
}

func (of *OneOfField) GenerateAccessors(ms *messageStruct) string {
	return template.Execute(template.Parse("oneOfAccessorTemplate", []byte(oneOfAccessorTemplate)), of.templateFields(ms))
}

func (of *OneOfField) typeFuncName() string {
	const typeSuffix = "Type"
	if of.omitOriginFieldNameInNames {
		return typeSuffix
	}
	return of.originFieldName + typeSuffix
}

func (of *OneOfField) GenerateAccessorsTest(ms *messageStruct) string {
	return template.Execute(template.Parse("oneOfAccessorTestTemplate", []byte(oneOfAccessorTestTemplate)), of.templateFields(ms))
}

func (of *OneOfField) GenerateTestValue(ms *messageStruct) string {
	return of.values[of.testValueIdx].GenerateTestValue(ms, of)
}

func (of *OneOfField) GenerateTestFailingUnmarshalProtoValues(ms *messageStruct) string {
	return template.Execute(template.Parse("oneOfTestFailingUnmarshalProtoValuesTemplate", []byte(oneOfTestFailingUnmarshalProtoValuesTemplate)), of.templateFields(ms))
}

func (of *OneOfField) GenerateTestEncodingValues(ms *messageStruct) string {
	return template.Execute(template.Parse("oneOfTestValuesTemplate", []byte(oneOfTestValuesTemplate)), of.templateFields(ms))
}

func (of *OneOfField) GeneratePoolOrig(ms *messageStruct) string {
	return template.Execute(template.Parse("oneOfPoolOrigTemplate", []byte(oneOfPoolOrigTemplate)), of.templateFields(ms))
}

func (of *OneOfField) GenerateDeleteOrig(ms *messageStruct) string {
	return template.Execute(template.Parse("oneOfDeleteOrigTemplate", []byte(oneOfDeleteOrigTemplate)), of.templateFields(ms))
}

func (of *OneOfField) GenerateCopyOrig(ms *messageStruct) string {
	return template.Execute(template.Parse("oneOfCopyOrigTemplate", []byte(oneOfCopyOrigTemplate)), of.templateFields(ms))
}

func (of *OneOfField) GenerateMarshalJSON(ms *messageStruct) string {
	return template.Execute(template.Parse("oneOfMarshalJSONTemplate", []byte(oneOfMarshalJSONTemplate)), of.templateFields(ms))
}

func (of *OneOfField) GenerateUnmarshalJSON(ms *messageStruct) string {
	return template.Execute(template.Parse("oneOfUnmarshalJSONTemplate", []byte(oneOfUnmarshalJSONTemplate)), of.templateFields(ms))
}

func (of *OneOfField) GenerateSizeProto(ms *messageStruct) string {
	t := template.Parse("oneOfSizeProtoTemplate", []byte(oneOfSizeProtoTemplate))
	return template.Execute(t, of.templateFields(ms))
}

func (of *OneOfField) GenerateMarshalProto(ms *messageStruct) string {
	t := template.Parse("oneOfMarshalProtoTemplate", []byte(oneOfMarshalProtoTemplate))
	return template.Execute(t, of.templateFields(ms))
}

func (of *OneOfField) GenerateUnmarshalProto(ms *messageStruct) string {
	t := template.Parse("oneOfUnmarshalProtoTemplate", []byte(oneOfUnmarshalProtoTemplate))
	return template.Execute(t, of.templateFields(ms))
}

func (of *OneOfField) templateFields(ms *messageStruct) map[string]any {
	return map[string]any{
		"baseStruct":           ms,
		"OneOfField":           of,
		"packageName":          "",
		"structName":           ms.getName(),
		"typeFuncName":         of.typeFuncName(),
		"typeName":             of.typeName,
		"originName":           ms.getOriginName(),
		"originFieldName":      of.originFieldName,
		"lowerOriginFieldName": strings.ToLower(of.originFieldName),
		"origAccessor":         origAccessor(ms.getHasWrapper()),
		"stateAccessor":        stateAccessor(ms.getHasWrapper()),
		"values":               of.values,
		"originTypePrefix":     ms.originFullName + "_",
	}
}

var _ Field = (*OneOfField)(nil)

type oneOfValue interface {
	GetOriginFieldName() string
	GenerateAccessors(ms *messageStruct, of *OneOfField) string
	GenerateTests(ms *messageStruct, of *OneOfField) string
	GenerateTestValue(ms *messageStruct, of *OneOfField) string
	GenerateTestFailingUnmarshalProtoValues(ms *messageStruct, of *OneOfField) string
	GenerateTestEncodingValues(ms *messageStruct, of *OneOfField) string
	GeneratePoolOrig(ms *messageStruct, of *OneOfField) string
	GenerateDeleteOrig(ms *messageStruct, of *OneOfField) string
	GenerateCopyOrig(ms *messageStruct, of *OneOfField) string
	GenerateType(ms *messageStruct, of *OneOfField) string
	GenerateMarshalJSON(ms *messageStruct, of *OneOfField) string
	GenerateUnmarshalJSON(ms *messageStruct, of *OneOfField) string
	GenerateSizeProto(ms *messageStruct, of *OneOfField) string
	GenerateMarshalProto(ms *messageStruct, of *OneOfField) string
	GenerateUnmarshalProto(ms *messageStruct, of *OneOfField) string
}
